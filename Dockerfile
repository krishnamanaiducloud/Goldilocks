############## 1. Builder Stage (Compiles the Go binary) ##############
ARG GO_IMAGE=cgr.dev/chainguard/go:latest-dev@sha256:21b175db480c45f8edd504d1a9bf80cc15f6026b2f427b0c03822c00a9dbb95b
ARG APK_REPOSITORY=https://packages.wolfi.dev/os
ARG WOLFI_REPO_DIGEST=f0031424cf46f7db780ce63a45f0fd6aa6f85f601e6bb3b7a91fe3d4d5b7d2cc

FROM ${GO_IMAGE} AS builder
ARG APK_REPOSITORY
ARG WOLFI_REPO_DIGEST

USER root

WORKDIR /app

# Install required tools (git for go mod or version injection)
COPY wolfi-signing.rsa.pub /tmp/wolfi-signing.rsa.pub
RUN echo "${WOLFI_REPO_DIGEST}  /tmp/wolfi-signing.rsa.pub" | sha256sum -c - \
    && mv /tmp/wolfi-signing.rsa.pub /etc/apk/keys/wolfi-signing.rsa.pub \
    && printf '%s\n' "${APK_REPOSITORY}" > /etc/apk/repositories \
    && apk upgrade --no-cache \
    && apk add --no-cache git

# Copy dependency files first (better caching)
COPY go.mod go.sum ./
RUN --mount=type=cache,id=goldilocks-go-mod,target=/go/pkg/mod,sharing=locked \
    go mod download

# Copy the rest of the source code
COPY main.go ./
COPY cmd ./cmd
COPY pkg ./pkg

RUN --mount=type=cache,id=goldilocks-go-mod,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,id=goldilocks-go-build,target=/root/.cache/go-build,sharing=locked \
    go test -mod=readonly ./...

# Build the Goldilocks binary with optimizations (static, embed friendly)
ARG VERSION=dev
ARG COMMIT=none
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG TARGETVARIANT
RUN --mount=type=cache,id=goldilocks-go-mod,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,id=goldilocks-go-build,target=/root/.cache/go-build,sharing=locked \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} \
    go build -mod=readonly -trimpath -o goldilocks \
    -ldflags="-X main.version=${VERSION} -X main.commit=${COMMIT} -s -w" \
    main.go

RUN chmod 0555 /app/goldilocks

############## 2. Minimal non-root runtime image ##############
FROM cgr.dev/chainguard/static:latest@sha256:c5fed92d0fda728930795cafb61192d8abc9e2f606b22996b212e0169f1d42e6

LABEL org.opencontainers.image.authors="FairwindsOps, Inc." \
      org.opencontainers.image.vendor="FairwindsOps, Inc." \
      org.opencontainers.image.title="Embark-Goldilocks" \
      org.opencontainers.image.description="Goldilocks is a utility that can help you identify a starting point for resource requests                                                                                  and limits. This image is build for the Embark Project" \
      org.opencontainers.image.documentation="https://goldilocks.docs.fairwinds.com/" \
      org.opencontainers.image.source="https://github.com/FairwindsOps/goldilocks" \
      org.opencontainers.image.url="https://github.com/FairwindsOps/goldilocks" \
      org.opencontainers.image.licenses="Apache License 2.0"

WORKDIR /

# Copy only the compiled binary (templates/assets are embedded in binary)
COPY --from=builder /app/goldilocks /goldilocks

USER 65532:65532

# Default entrypoint (run program)
ENTRYPOINT ["/goldilocks"]
