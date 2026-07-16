############## 1. Builder Stage (Compiles the Go binary) ##############
ARG GO_IMAGE=golang:1.26.5-alpine3.24@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2

FROM ${GO_IMAGE} AS builder

WORKDIR /app

# Install required tools (git for go mod or version injection)
RUN apk upgrade --no-cache \
    && apk add --no-cache git

# Copy dependency files first (better caching)
COPY go.mod go.sum ./
RUN go mod download

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
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} GOARM=${TARGETVARIANT#v} \
    go build -o goldilocks \
    -ldflags="-X main.version=${VERSION} -X main.commit=${COMMIT} -s -w" \
    main.go

RUN chmod g+rwX /app/goldilocks

############## 2. Final Image WITH SHELL (distroless:debug) ##############
FROM gcr.io/distroless/static-debian13:nonroot@sha256:f7f8f729987ad0fdf6b05eeeae94b26e6a0f613bdf46feea7fc40f7bd72953e6

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

# Default entrypoint (run program)
ENTRYPOINT ["/goldilocks"]
