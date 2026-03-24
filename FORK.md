# Goldilocks Fork Documentation

This repository is a fork/clone of the original [Goldilocks project by FairwindsOps](https://github.com/FairwindsOps/goldilocks).

## Purpose of This Fork

This fork was created for the **Embark Project** to provide customized Kubernetes resource recommendation capabilities with enhanced deployment options and container image variants.

## Differences from Upstream

### 1. Container Image Strategy

This fork provides multiple Dockerfile variants to support different deployment scenarios:

- **`Dockerfile`** - Production multi-stage build using distroless base
- **`Dockerfile-debug`** - Debug variant with shell access
- **`Dockerfile-openshift`** - OpenShift-compatible variant
- **`Dockerfile-openshift-debug`** - OpenShift debug variant with shell
- **`Dockerfile-nw`** - Network-specific variant
- **`Dockerfile-nw1`** - Alternative network variant
- **`Dockerfile-v1`** - Legacy version 1 build
- **`Dockerfile-v2`** - Legacy version 2 build
- **`Dockerfile-10-20-2025`** - Dated snapshot build

**Rationale**: Different deployment environments (OpenShift, standard Kubernetes, debugging scenarios) require different base images and configurations.

### 2. Build Approach

**Original**: Uses GoReleaser with pre-built binaries copied into minimal Alpine images.

**This Fork**: Uses multi-stage Docker builds that compile from source, providing:
- Self-contained builds without external dependencies
- Flexibility to modify build flags per environment
- Consistent build environment across all variants

### 3. Base Image Selection

**Original**: `alpine:3.23` (simple, lightweight)

**This Fork**: 
- Builder stage: `golang:1.24.6-alpine3.22`
- Runtime stage: `gcr.io/distroless/static:nonroot` (even more minimal and secure)

**Rationale**: Distroless images have smaller attack surface and no shell by default, improving security posture.

### 4. Code Improvements

#### Better Error Output
```go
// Original uses println()
println("message")

// This fork uses os.Stderr.WriteString()
os.Stderr.WriteString("message")
```

**Rationale**: `os.Stderr.WriteString` is more appropriate for non-output messages and follows Go best practices.

### 5. Container Labels

Enhanced OCI labels for the Embark Project:
```dockerfile
org.opencontainers.image.title="Embark-Goldilocks"
org.opencontainers.image.description="...This image is build for the Embark Project"
```

## Maintenance Strategy

### Syncing with Upstream

This fork periodically syncs with the upstream Goldilocks repository to incorporate:
- Security patches
- Bug fixes
- New features
- Dependency updates


### Dependency Management

Dependencies are kept in sync with upstream to ensure:
- Security vulnerability patches
- Kubernetes API compatibility
- Go version alignment

## Building This Fork

### Standard Build
```bash
docker build -t goldilocks:latest -f Dockerfile .
```

### Debug Build
```bash
docker build -t goldilocks:debug -f Dockerfile-debug .
```

### OpenShift Build
```bash
docker build -t goldilocks:openshift -f Dockerfile-openshift .
```

### From Source
```bash
make build
```

## Deployment Differences

### Standard Kubernetes
Use the standard `Dockerfile` for production deployments.

### OpenShift
Use `Dockerfile-openshift` or `Dockerfile-openshift-debug` which are compatible with OpenShift's security context constraints.

### Debugging
Use `Dockerfile-debug` or `Dockerfile-openshift-debug` when you need shell access for troubleshooting.

## Contributing

### To This Fork
For Embark-specific changes, follow the standard pull request process for this repository.

### To Upstream
If you've made improvements that would benefit the broader community, consider contributing them back to the [original Goldilocks project](https://github.com/FairwindsOps/goldilocks).

## License

This fork maintains the same Apache License 2.0 as the original project. See [LICENSE](LICENSE) file for details.

## Upstream Resources

- **Original Repository**: https://github.com/FairwindsOps/goldilocks
- **Documentation**: https://goldilocks.docs.fairwinds.com/
- **Community**: [Fairwinds Slack](https://join.slack.com/t/fairwindscommunity/shared_invite/zt-2na8gtwb4-DGQ4qgmQbczQyB2NlFlYQQ)

## Support

For issues specific to this fork, please open an issue in this repository.

For general Goldilocks questions, refer to the [upstream documentation](https://goldilocks.docs.fairwinds.com/) or [community resources](https://www.fairwinds.com/open-source-software).

---

