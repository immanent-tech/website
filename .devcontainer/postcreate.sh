#!/usr/bin/bash

set -x

cd /workspace

# Oh My Posh.
source base/.devcontainer/postcreate.scripts.d/oh-my-posh.sh

# Frontend.
source base/.devcontainer/postcreate.scripts.d/frontend.sh

# Go.
source base/.devcontainer/postcreate.scripts.d/go.sh

# Pulumi.
source base/.devcontainer/postcreate.scripts.d/pulumi.sh

# Google Cloud.
source base/.devcontainer/postcreate.scripts.d/gcloud.sh

