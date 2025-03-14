# Builder Stage (Compiles the Go binary)
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install required tools
RUN apk add --no-cache git

# Copy dependency files first (better caching)
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the Goldilocks binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o goldilocks -ldflags "-s -w"

# Ensure the binary is executable
RUN chmod +x /app/goldilocks

# Final Minimal Image (Production-Ready)
FROM gcr.io/distroless/static:nonroot

LABEL org.opencontainers.image.authors="FairwindsOps, Inc." \
      org.opencontainers.image.vendor="FairwindsOps, Inc." \
      org.opencontainers.image.title="Embark-Goldilocks" \
      org.opencontainers.image.description="Goldilocks is a utility that can help you identify a starting point for resource requests and limits. This image is build for the Embark Project" \
      org.opencontainers.image.documentation="https://goldilocks.docs.fairwinds.com/" \
      org.opencontainers.image.source="https://github.com/FairwindsOps/goldilocks" \
      org.opencontainers.image.url="https://github.com/FairwindsOps/goldilocks" \
      org.opencontainers.image.licenses="Apache License 2.0"

WORKDIR /

# Copy only the compiled binary from the builder stage
COPY --from=builder /app/goldilocks /goldilocks
COPY --from=builder /app/pkg/dashboard/templates /templates
COPY --from=builder /app/pkg/dashboard/assets /assets

# Use non-root user (security best practice)
USER nonroot

# Set correct entrypoint
ENTRYPOINT ["/goldilocks"]
