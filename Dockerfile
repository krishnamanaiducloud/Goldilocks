############## 1. Builder Stage (Compiles the Go binary) ##############
ARG GO_IMAGE=cgr.dev/chainguard/go:latest-dev@sha256:1e5b0977fb7db0d314b2b91faad65b35e2b137242f2d9f34ff5eb4e30a47772f
ARG APK_REPOSITORY=https://apk.cgr.dev/chainguard
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
    && apk --timeout 60 upgrade --no-cache \
    && apk --timeout 60 add --no-cache git

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
    CGO_ENABLED=0 go test -trimpath -p=2 -mod=readonly ./...

# Build the Goldilocks binary with optimizations (static, embed friendly)
ARG VERSION=v54
ARG COMMIT=none
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG TARGETVARIANT
RUN --mount=type=cache,id=goldilocks-go-mod,target=/go/pkg/mod,sharing=locked \
    --mount=type=cache,id=goldilocks-go-build,target=/root/.cache/go-build,sharing=locked \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} \
    go build -p=2 -mod=readonly -trimpath -o goldilocks \
    -ldflags="-X main.version=${VERSION} -X main.commit=${COMMIT} -s -w" \
    main.go

RUN chmod 0555 /app/goldilocks

############## 2. Minimal non-root runtime image ##############
FROM cgr.dev/chainguard/static:latest@sha256:fe55470f22d3259488d9d3739168d8f04da67755f0b69382bc26eda4a7d3d327

LABEL org.opencontainers.image.authors="FairwindsOps, Inc." \
      org.opencontainers.image.vendor="FairwindsOps, Inc." \
      org.opencontainers.image.title="Goldilocks" \
      org.opencontainers.image.description="Goldilocks helps identify a starting point for Kubernetes resource requests and limits." \
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
