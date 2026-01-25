// Copyright 2025 Joshua Rich <joshua.rich@gmail.com>.
// SPDX-License-Identifier: 	AGPL-3.0-or-later

package main

import (
	"fmt"
	"os"
	"strconv"

	"github.com/pulumi/pulumi-cloudflare/sdk/v6/go/cloudflare"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/artifactregistry"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudrun"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/cloudrunv2"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/organizations"
	"github.com/pulumi/pulumi-gcp/sdk/v9/go/gcp/secretmanager"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

const (
	defaultCloudProvider = "gcp"
	serverProvider       = "immanent_tech"
	envPrefix            = "IMMANENT_TECH_WEB_"
)

type Config struct {
	Server   *config.Config
	Provider *config.Config
}

func (c *Config) GetGCPRegion() string {
	return c.Provider.Require("region")
}

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		var err error
		cfg := &Config{
			Server:   config.New(ctx, serverProvider),
			Provider: config.New(ctx, defaultCloudProvider),
		}

		project, err := organizations.LookupProject(ctx, &organizations.LookupProjectArgs{}, nil)
		if err != nil {
			return err
		}

		port, err := strconv.Atoi(os.Getenv(envPrefix + "PORT"))
		if err != nil {
			return err
		}

		// Create ghcr artifact repository.
		github_auth_secret, err := secretmanager.NewSecret(ctx, "github_auth_secret", &secretmanager.SecretArgs{
			SecretId: pulumi.String("github_auth"),
			Replication: &secretmanager.SecretReplicationArgs{
				Auto: &secretmanager.SecretReplicationAutoArgs{},
			},
		})
		if err != nil {
			return err
		}
		github_auth_secret_version, err := secretmanager.NewSecretVersion(
			ctx,
			"github_auth_secret_version",
			&secretmanager.SecretVersionArgs{
				Secret:     github_auth_secret.ID(),
				SecretData: cfg.Server.GetSecret("github_pat"),
			},
		)
		if err != nil {
			return err
		}
		_, err = secretmanager.NewSecretIamMember(ctx, "secret-access", &secretmanager.SecretIamMemberArgs{
			SecretId: github_auth_secret.ID(),
			Role:     pulumi.String("roles/secretmanager.secretAccessor"),
			Member: pulumi.Sprintf(
				"serviceAccount:service-%v@gcp-sa-artifactregistry.iam.gserviceaccount.com",
				project.Number,
			),
		})
		if err != nil {
			return err
		}

		// Useful reference: https://alphasec.io/how-to-deploy-a-github-container-image-to-google-cloud-run/
		repoResource, err := artifactregistry.NewRepository(ctx, "repo-ghcr-remote", &artifactregistry.RepositoryArgs{
			Location:     pulumi.String(cfg.GetGCPRegion()),
			RepositoryId: pulumi.String("ghcr-remote"),
			Description:  pulumi.String("Remote repository configuration for ghcr.io"),
			Format:       pulumi.String("DOCKER"),
			Mode:         pulumi.String("REMOTE_REPOSITORY"),
			RemoteRepositoryConfig: &artifactregistry.RepositoryRemoteRepositoryConfigArgs{
				DisableUpstreamValidation: pulumi.Bool(true),
				DockerRepository: &artifactregistry.RepositoryRemoteRepositoryConfigDockerRepositoryArgs{
					CustomRepository: &artifactregistry.RepositoryRemoteRepositoryConfigDockerRepositoryCustomRepositoryArgs{
						Uri: pulumi.String("https://ghcr.io"),
					},
				},
				UpstreamCredentials: &artifactregistry.RepositoryRemoteRepositoryConfigUpstreamCredentialsArgs{
					UsernamePasswordCredentials: &artifactregistry.RepositoryRemoteRepositoryConfigUpstreamCredentialsUsernamePasswordCredentialsArgs{
						Username:              cfg.Server.GetSecret("github_username"),
						PasswordSecretVersion: github_auth_secret_version.Name,
					},
				},
			},
		})
		if err != nil {
			return err
		}

		// Create server instance.
		//
		serverImage := pulumi.Sprintf(
			"%s-docker.pkg.dev/%s/ghcr-remote/%s:%s",
			cfg.GetGCPRegion(),
			*project.ProjectId,
			cfg.Server.Get("server_image"),
			cfg.Server.Get("version"),
		)
		serverResource, err := cloudrunv2.NewService(ctx, "immanent-tech-web-server", &cloudrunv2.ServiceArgs{
			Location:   pulumi.String(cfg.GetGCPRegion()),
			Ingress:    pulumi.String("INGRESS_TRAFFIC_ALL"),
			IapEnabled: pulumi.Bool(false),
			Template: cloudrunv2.ServiceTemplateArgs{
				MaxInstanceRequestConcurrency: pulumi.Int(cfg.Server.GetInt("server_concurrency")),
				Containers: cloudrunv2.ServiceTemplateContainerArray{
					&cloudrunv2.ServiceTemplateContainerArgs{
						Image: serverImage,
						Resources: cloudrunv2.ServiceTemplateContainerResourcesArgs{
							Limits: pulumi.ToStringMap(map[string]string{
								"memory": cfg.Server.Get("server_memory"),
								"cpu":    cfg.Server.Get("server_cpu"),
							}),
						},
						Ports: &cloudrunv2.ServiceTemplateContainerPortsArgs{
							ContainerPort: pulumi.Int(port),
							Name:          pulumi.String("h2c"),
						},
						LivenessProbe: &cloudrunv2.ServiceTemplateContainerLivenessProbeArgs{
							HttpGet: cloudrunv2.ServiceTemplateContainerLivenessProbeHttpGetArgs{
								Path: pulumi.String("/health-check"),
							},
						},
						Envs: cloudrunv2.ServiceTemplateContainerEnvArray{
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "BASEURL"),
								Value: pulumi.String(os.Getenv(envPrefix + "BASEURL")),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "ENVIRONMENT"),
								Value: pulumi.String(os.Getenv(envPrefix + "ENVIRONMENT")),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "LOGLEVEL"),
								Value: pulumi.String(os.Getenv(envPrefix + "LOGLEVEL")),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "HOST"),
								Value: pulumi.String(os.Getenv(envPrefix + "HOST")),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "PORT"),
								Value: pulumi.String(os.Getenv(envPrefix + "PORT")),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "READTIMEOUT"),
								Value: pulumi.String(os.Getenv(envPrefix + "READTIMEOUT")),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "WRITETIMEOUT"),
								Value: pulumi.String(os.Getenv(envPrefix + "WRITETIMEOUT")),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "IDLETIMEOUT"),
								Value: pulumi.String(os.Getenv(envPrefix + "IDLETIMEOUT")),
							},
							// CSP.
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "CSP_BASEURI"),
								Value: pulumi.String(os.Getenv(envPrefix + "CSP_BASEURI")),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "CSP_CONNECTSRC"),
								Value: pulumi.String(os.Getenv(envPrefix + "CSP_CONNECTSRC")),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "CSP_DEFAULTSRC"),
								Value: pulumi.String(os.Getenv(envPrefix + "CSP_DEFAULTSRC")),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "CSP_FONTSRC"),
								Value: pulumi.String(os.Getenv(envPrefix + "CSP_FONTSRC")),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "CSP_IMGSRC"),
								Value: pulumi.String(os.Getenv(envPrefix + "CSP_IMGSRC")),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "CSP_MEDIASRC"),
								Value: pulumi.String(os.Getenv(envPrefix + "CSP_MEDIASRC")),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "CSP_SCRIPTSRC"),
								Value: pulumi.String(os.Getenv(envPrefix + "CSP_SCRIPTSRC")),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "CSP_STYLESRC"),
								Value: pulumi.String(os.Getenv(envPrefix + "CSP_STYLESRC")),
							},
							// CORS.
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "CORS_ALLOWEDORIGINS"),
								Value: pulumi.String(os.Getenv(envPrefix + "CORS_ALLOWEDORIGINS")),
							},
							&cloudrunv2.ServiceTemplateContainerEnvArgs{
								Name:  pulumi.String(envPrefix + "CORS_MAXAGE"),
								Value: pulumi.String(os.Getenv(envPrefix + "CORS_MAXAGE")),
							},
						},
					},
				},
			},
		},
			pulumi.Protect(false),
			pulumi.DependsOn([]pulumi.Resource{repoResource}),
		)
		if err != nil {
			return err
		}
		// Create an IAM member to make the service publicly accessible.
		_, err = cloudrun.NewIamMember(ctx, "invoker", &cloudrun.IamMemberArgs{
			Service:  serverResource.Name,
			Location: pulumi.String(cfg.GetGCPRegion()),
			Role:     pulumi.String("roles/run.invoker"),
			Member:   pulumi.String("allUsers"),
		})
		if err != nil {
			return err
		}

		setState("server_urls", serverResource.Urls)

		mappingResource, err := cloudrun.NewDomainMapping(
			ctx,
			"immanent-tech-website-domain-mapping",
			&cloudrun.DomainMappingArgs{
				Location: pulumi.String(cfg.GetGCPRegion()),
				Metadata: &cloudrun.DomainMappingMetadataArgs{
					Namespace: pulumi.String(project.Name),
				},
				Name:    pulumi.String(cfg.Server.Get("domain_name")),
				Project: pulumi.String(project.Name),
				Spec: &cloudrun.DomainMappingSpecArgs{
					RouteName: serverResource.Name,
				},
			},
			pulumi.Protect(false),
			pulumi.DependsOn([]pulumi.Resource{serverResource}),
		)
		if err != nil {
			return err
		}

		setState("server_dns_records", mappingResource.Statuses.Index(pulumi.Int(0)).ResourceRecords())

		// Update DNS.
		server_dns := getState[cloudrun.DomainMappingStatusResourceRecordArrayOutput]("server_dns_records")
		server_dns.ToDomainMappingStatusResourceRecordArrayOutput().
			ApplyT(func(array []cloudrun.DomainMappingStatusResourceRecord) ([]cloudrun.DomainMappingStatusResourceRecord, error) {
				for _, record := range array {
					name := *record.Name
					value := *record.Rrdata
					recordType := *record.Type
					ctx.Log.Info(fmt.Sprintf("Server DNS record: %s %s %s", name, value, recordType), nil)
					_, err := cloudflare.NewDnsRecord(
						ctx,
						"server_dns_"+recordType+"_"+value,
						&cloudflare.DnsRecordArgs{
							Name:    pulumi.String(cfg.Server.Get("domain_name")),
							Type:    pulumi.String(recordType),
							Ttl:     pulumi.Float64(1),
							ZoneId:  cfg.Server.GetSecret("cloudflare_zoneid"),
							Content: pulumi.String(value),
							Proxied: pulumi.Bool(true),
						},
					)
					if err != nil {
						return nil, err
					}
				}
				return array, nil
			})

		return nil
	})
}
