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

# Install Go packages.
go mod tidy
go install github.com/air-verse/air@latest
go install github.com/a-h/templ/cmd/templ@latest
go install golang.org/x/tools/gopls@latest


exit 0
