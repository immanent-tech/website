# Copyright 2025 Joshua Rich <joshua.rich@gmail.com>.
# SPDX-License-Identifier: 	AGPL-3.0-or-later

# Alpine base.
# https://hub.docker.com/_/alpine/
FROM --platform=$BUILDPLATFORM docker.io/alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS builder

ARG TARGETOS
ARG TARGETARCH
ARG APPVERSION

WORKDIR /build

# Copy go from official image.
# https://hub.docker.com/_/golang
COPY --from=docker.io/golang:1.26.3-alpine-3.23@sha256:91eda9776261207ea25fd06b5b7fed8d397dd2c0a283e77f2ab6e91bfa71079d /usr/local/go/ /usr/local/go/
# Update $PATH.
ENV PATH="/root/go/bin:/usr/local/go/bin:/usr/local/bin:${PATH}"

# Install tools.
RUN apk add libstdc++ upx npm

# Copy and download dependency using go mod.
COPY go.mod go.sum ./
RUN go mod download

# Copy source.
COPY . .

# install and build/bundle frontend assets
RUN <<EOF
npm clean-install && \
    npm run build:prod && \
    npm version patch
EOF

# Set necessary environment variables and build your project.
ENV CGO_ENABLED=0
RUN go build -ldflags="-s -w -X github.com/immanent-tech/www-immanent-tech/config.Version=$APPVERSION" -o webserver

# compress binary with upx
RUN upx --best --lzma webserver

FROM docker.io/alpine:3.23.4@sha256:5b10f432ef3da1b8d4c7eb6c487f2f5a8f096bc91145e68878dd4a5019afde11 AS server

ENV IMMANENT_TECH_WEB_CONTAINER=1

# Add labels.
LABEL org.opencontainers.image.source=https://github.com/immanent-tech/website
LABEL org.opencontainers.image.url=https://immanent.tech
LABEL org.opencontainers.image.title="Immanent Tech Website"
LABEL org.opencontainers.image.description="The Immanent Tech Website."
LABEL org.opencontainers.image.licenses=AGPL-3.0-or-later

# Install supporting packages required for certain functionality.
RUN apk add ca-certificates tzdata

# Copy project's binary and templates from /build to the scratch container.
COPY --from=builder /build/webserver /

# Allow custom uid and gid
ARG UID=1000
ARG GID=1000

# Add user
RUN addgroup --gid "${GID}" imtech && \
    adduser --disabled-password --gecos "" --ingroup imtech \
    --uid "${UID}" imtech
USER imtech

# Set entry point.
ENTRYPOINT ["/webserver"]
CMD ["serve", "--no-log-file"]
