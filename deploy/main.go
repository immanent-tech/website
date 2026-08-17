// Copyright 2026 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package main

import (
	"fmt"
	"os"

	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudrun"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudrunv2"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/iam"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/projects"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/serviceaccount"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {

		cfg := config.New(ctx, "")
		projectID := cfg.Require("projectId")
		region := cfg.Require("region")
		repoOwner := cfg.Require("repoOwner")
		repoName := cfg.Require("repoName")

		// 1. Artifact Registry Docker repo.
		// Pulumi is idempotent — no skip-if-exists logic needed.
		if _, err := artifactregistry.NewRepository(ctx, "artifact-registry-repo", &artifactregistry.RepositoryArgs{
			RepositoryId: pulumi.String("www-immanent-tech"),
			Format:       pulumi.String("DOCKER"),
			Location:     pulumi.String(region),
			Project:      pulumi.String(projectID),
		}); err != nil {
			return fmt.Errorf("create artifact registry repo: %w", err)
		}

		// 2. Service account for GitHub Actions to impersonate.
		ghSA, err := serviceaccount.NewAccount(ctx, "gh-actions-deployer", &serviceaccount.AccountArgs{
			AccountId:   pulumi.String("gh-actions-deployer"),
			DisplayName: pulumi.String("GitHub Actions Deployer"),
			Project:     pulumi.String(projectID),
		})
		if err != nil {
			return fmt.Errorf("create actions deployer service account: %w", err)
		}

		// 3. Grant the service account permission to push images.
		if _, err = projects.NewIAMMember(ctx, "gh-actions-ar-writer", &projects.IAMMemberArgs{
			Project: pulumi.String(projectID),
			Role:    pulumi.String("roles/artifactregistry.writer"),
			Member:  pulumi.Sprintf("serviceAccount:%s", ghSA.Email),
		}); err != nil {
			return fmt.Errorf("create actions repository writer: %w", err)
		}

		// 4. Workload Identity Pool.
		pool, err := iam.NewWorkloadIdentityPool(ctx, "github-pool", &iam.WorkloadIdentityPoolArgs{
			WorkloadIdentityPoolId: pulumi.String("github-pool"),
			DisplayName:            pulumi.String("GitHub Actions Pool"),
			Project:                pulumi.String(projectID),
		})
		if err != nil {
			return fmt.Errorf("create workload identity pool: %w", err)
		}

		// 5. OIDC provider trusting GitHub Actions tokens.
		if _, err = iam.NewWorkloadIdentityPoolProvider(ctx, "github-provider", &iam.WorkloadIdentityPoolProviderArgs{
			WorkloadIdentityPoolId:         pool.WorkloadIdentityPoolId,
			WorkloadIdentityPoolProviderId: pulumi.String("github-provider"),
			DisplayName:                    pulumi.String("GitHub Provider"),
			Project:                        pulumi.String(projectID),
			AttributeMapping: pulumi.StringMap{
				"google.subject":       pulumi.String("assertion.sub"),
				"attribute.repository": pulumi.String("assertion.repository"),
			},
			AttributeCondition: pulumi.String(
				fmt.Sprintf("assertion.repository=='%s/%s'", repoOwner, repoName),
			),
			Oidc: &iam.WorkloadIdentityPoolProviderOidcArgs{
				IssuerUri: pulumi.String("https://token.actions.githubusercontent.com"),
			},
		}); err != nil {
			return fmt.Errorf("create workload identity pool provider: %w", err)
		}

		// 6. Allow only this repo to impersonate the service account.
		// pool.Name resolves to the full resource path:
		//   projects/{project_number}/locations/global/workloadIdentityPools/github-pool
		if _, err = serviceaccount.NewIAMMember(ctx, "gh-actions-wif-binding", &serviceaccount.IAMMemberArgs{
			ServiceAccountId: ghSA.Name,
			Role:             pulumi.String("roles/iam.workloadIdentityUser"),
			Member: pulumi.Sprintf(
				"principalSet://iam.googleapis.com/%s/attribute.repository/%s/%s",
				pool.Name,
				repoOwner,
				repoName,
			),
		}); err != nil {
			return fmt.Errorf("create actions service account binding: %w", err)
		}

		// Create the service account for the server.
		serverSA, err := serviceaccount.NewAccount(ctx, "website-sa", &serviceaccount.AccountArgs{
			AccountId:   pulumi.String("website-sa"),
			DisplayName: pulumi.String("Website Service Account"),
			Project:     pulumi.String(projectID),
		})
		if err != nil {
			return fmt.Errorf("create server service account: %w", err)
		}
		serverImage := pulumi.Sprintf(
			"%s-docker.pkg.dev/%s/%s/%s:%s",
			region,
			projectID,
			repoOwner,
			repoName,
			config.Require(ctx, "version"),
		)

		serverResource, err := cloudrunv2.NewService(
			ctx,
			"website-server",
			&cloudrunv2.ServiceArgs{
				Description:        pulumi.String("Immanent Tech Website"),
				DeletionProtection: pulumi.Bool(false),
				LaunchStage:        pulumi.String("GA"),
				Location:           pulumi.String(region),
				Ingress:            pulumi.String("INGRESS_TRAFFIC_ALL"),
				InvokerIamDisabled: pulumi.Bool(true),
				Template: cloudrunv2.ServiceTemplateArgs{
					ServiceAccount: serverSA.Email,
					Scaling: &cloudrunv2.ServiceTemplateScalingArgs{
						MaxInstanceCount: pulumi.Int(config.RequireInt(ctx, "server_max_instances")),
						MinInstanceCount: pulumi.Int(0),
					},
					MaxInstanceRequestConcurrency: pulumi.Int(
						config.RequireInt(ctx, "server_concurrency"),
					),
					Containers: cloudrunv2.ServiceTemplateContainerArray{
						&cloudrunv2.ServiceTemplateContainerArgs{
							Name:  pulumi.String("foragd-server"),
							Image: serverImage,
							Resources: cloudrunv2.ServiceTemplateContainerResourcesArgs{
								Limits: pulumi.ToStringMap(map[string]string{
									"memory": config.Require(ctx, "server_memory"),
									"cpu":    config.Require(ctx, "server_cpu"),
								}),
								CpuIdle:         pulumi.Bool(true),
								StartupCpuBoost: pulumi.Bool(false),
							},
							Ports: &cloudrunv2.ServiceTemplateContainerPortsArgs{
								ContainerPort: pulumi.Int(config.RequireInt(ctx, "server_port")),
								Name:          pulumi.String("h2c"),
							},
							LivenessProbe: &cloudrunv2.ServiceTemplateContainerLivenessProbeArgs{
								HttpGet: cloudrunv2.ServiceTemplateContainerLivenessProbeHttpGetArgs{
									Path: pulumi.String("/health-check"),
								},
							},
							Envs: cloudrunv2.ServiceTemplateContainerEnvArray{
								// Server config.
								cloudrunServiceEnv("APP_VERSION", nil),
								cloudrunServiceEnv("APP_BASEURL", nil),
								cloudrunServiceEnv("APP_NAME", nil),
								cloudrunServiceEnv("APP_ID", nil),
								cloudrunServiceEnv("APP_ENVIRONMENT", nil),
								cloudrunServiceEnv("LOG_LEVEL", nil),
								// CSP.
								cloudrunServiceEnv("CSP_CONNECTSRC", nil),
								cloudrunServiceEnv("CSP_IMGSRC", nil),
								cloudrunServiceEnv("CSP_SCRIPTSRC", nil),
								cloudrunServiceEnv("CSP_FRAMESRC", nil),
								// CORS.
								cloudrunServiceEnv("CORS_ALLOWEDORIGINS", nil),
								cloudrunServiceEnv("CORS_MAXAGE", nil),
								// Cloudflare
								cloudrunServiceEnv("CLOUDFLARE_TURNSTILE_KEY", nil),
								// Umami
								cloudrunServiceEnv("UMAMI_ID", nil),
							},
						},
					},
				},
			},
			pulumi.Protect(false),
		)
		if err != nil {
			return fmt.Errorf("create server resource: %w", err)
		}

		// Create an IAM member to make the service publicly accessible.
		_, err = cloudrunv2.NewServiceIamMember(ctx, "website-public-access", &cloudrunv2.ServiceIamMemberArgs{
			Name:     serverResource.Name,
			Location: pulumi.String(region),
			Project:  pulumi.String(projectID),
			Role:     pulumi.String("roles/run.invoker"),
			Member:   pulumi.String("allUsers"),
		})
		if err != nil {
			return fmt.Errorf("create server public access: %w", err)
		}

		// Create domain mappings.
		mapping, err := cloudrun.NewDomainMapping(ctx, "website-domain-mapping", &cloudrun.DomainMappingArgs{
			Location: pulumi.String(region),
			Name:     pulumi.String(config.Require(ctx, "domain")),
			Metadata: &cloudrun.DomainMappingMetadataArgs{
				Namespace: pulumi.String(projectID),
			},
			Spec: &cloudrun.DomainMappingSpecArgs{
				RouteName: serverResource.Name,
			},
		})
		if err != nil {
			return fmt.Errorf("create domain mapping: %w", err)
		}

		// Export the domain mappings so you can see what to add at your registrar after deploy.
		dnsRecords := mapping.Statuses.ApplyT(
			func(statuses []cloudrun.DomainMappingStatus) any {
				if len(statuses) == 0 || len(statuses[0].ResourceRecords) == 0 {
					return nil
				}
				return statuses[0].ResourceRecords
			},
		)
		ctx.Export("dnsRecords", dnsRecords)

		return nil
	})
}

func cloudrunServiceEnv(name string, value any) *cloudrunv2.ServiceTemplateContainerEnvArgs {
	switch value {
	case nil:
		return &cloudrunv2.ServiceTemplateContainerEnvArgs{
			Name:  pulumi.String(name),
			Value: pulumi.String(os.Getenv(name)),
		}
	default:
		switch valueType := value.(type) {
		case string:
			return &cloudrunv2.ServiceTemplateContainerEnvArgs{
				Name:  pulumi.String(name),
				Value: pulumi.String(valueType),
			}
		case pulumi.StringOutput:
			return &cloudrunv2.ServiceTemplateContainerEnvArgs{
				Name:  pulumi.String(name),
				Value: valueType,
			}
		default:
			return nil
		}
	}
}
