# Copyright 2025 Joshua Rich <joshua.rich@gmail.com>.
# SPDX-License-Identifier: 	AGPL-3.0-or-later

ARG ALPINE_VERSION=3.24.1@sha256:79ff19e9084a00eece421b2523fb93e22d730e2c0e525905de047e848e56d95f
ARG GOLANG_VERSION=1.26.5-alpine3.24@sha256:111d79159b2326f7e80c4a4706e1ba166acb0e2611df853955f3621828cd49e8

# Copy go from official image.
# https://hub.docker.com/_/golang
FROM docker.io/golang:${GOLANG_VERSION} AS golang
# Alpine base.
# https://hub.docker.com/_/alpine/
FROM --platform=$BUILDPLATFORM docker.io/alpine:${ALPINE_VERSION} AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /build

COPY --from=golang /usr/local/go/ /usr/local/go/
# Update $PATH.
ENV PATH="/root/go/bin:/usr/local/go/bin:/usr/local/bin:${PATH}"

# Install tools.
RUN apk add libstdc++ upx npm

# Copy and download dependency using go mod.
COPY go.mod go.sum ./
COPY base/go.mod base/go.sum ./base/
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
RUN go build -ldflags="-s -w" -o webserver

# compress binary with upx
RUN upx --best --lzma webserver

FROM --platform=$BUILDPLATFORM docker.io/alpine:${ALPINE_VERSION} AS server

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
CMD ["serve"]
