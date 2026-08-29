############## 1. Builder Stage (Compiles the Go binary) ##############
ARG GO_IMAGE=cgr.dev/chainguard/go:latest-dev@sha256:3e9fd37454302e8a618bba86f59ce4c4bd8ae108e2bcf0bbb188a4252903b51e

FROM ${GO_IMAGE} AS builder

USER root

WORKDIR /app

# Install required tools (git for go mod or version injection)
RUN apk upgrade --no-cache \
    && apk add --no-cache git

# Copy dependency files first (better caching)
COPY go.mod go.sum ./
RUN --mount=type=cache,id=goldilocks-go-mod,target=/go/pkg/mod,sharing=locked \
    go mod download

# Copy the rest of the source code
COPY main.go ./
COPY cmd ./cmd
COPY pkg ./pkg

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
FROM cgr.dev/chainguard/static:latest@sha256:96d02f455d5a73b817c0602910748609cf8471b1cc9522f78c75cedb1f67d072

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
