#!/usr/bin/bash

set -x

# Add starship to fish shell.
mkdir -p ~/.config/fish
echo "starship init fish | source" >>~/.config/fish/config.fish

# Add starship to bash shell.
echo 'eval "$(starship init bash)"' >>~/.bashrc

cd /workspace

# Update JS packages.
npm update || exit -1
echo 'set --export PATH "/workspace/node_modules/.bin" $PATH' >> ~/.config/fish/config.fish

# Install Go packages.
echo 'set --export PATH "$HOME/go/bin" /go/bin /usr/local/go/bin $PATH' >> ~/.config/fish/config.fish
export PATH="$HOME/go/bin:/go/bin:/usr/local/go/bin:$PATH" && \
    go mod tidy && \
    go install golang.org/x/tools/gopls@latest && \
    go install github.com/air-verse/air@latest && \
    go install github.com/a-h/templ/cmd/templ@latest && \
    curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.8.0 && \
    golangci-lint custom && \
    mv /tmp/golangci-lint-v2 $(go env GOPATH)/bin/

# Install gcloud cli
cd $HOME && \
    curl -O https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-linux-x86_64.tar.gz && \
    tar -xf google-cloud-cli-linux-x86_64.tar.gz && \
    rm google-cloud-cli-linux-x86_64.tar.gz && \
    sudo apk add python3 && \
    google-cloud-sdk/install.sh --usage-reporting false --quiet --additional-components app-engine-go && \
    echo 'source /home/vscode/google-cloud-sdk/path.fish.inc' >> ~/.config/fish/config.fish

# Install pulumictl
RUN curl -fsSL https://get.pulumi.com | sh
echo 'set --export PULUMI_INSTALL "$HOME/.pulumi"' >> ~/.config/fish/config.fish
echo 'set --export PATH $PULUMI_INSTALL/bin $PATH' >> ~/.config/fish/config.fish

exit 0
