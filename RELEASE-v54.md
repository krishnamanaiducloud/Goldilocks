# v54 dependency and container refresh

This release retains the existing statically linked Go binary and non-root runtime architecture. The production image has no glibc package; adding one would increase attack surface without benefiting a `CGO_ENABLED=0` binary.

## Dependency decisions

- Go 1.27.1 is the latest stable toolchain reported by the official Go download API at validation time.
- Direct application modules already match the latest stable releases reported by the Go module proxy.
- Compatible updates: `fxamacker/cbor/v2` 2.9.6, `go-openapi/jsonpointer` 1.0.2, `go-openapi/jsonreference` 1.0.3 and `prometheus/common` 0.72.0.
- Keep dependency-selected Kubernetes pseudo-versions rather than pulling untagged development snapshots solely because their timestamps are newer.
- Documentation retains stable VitePress 1.6.4 and updates its compatible Vite 6 override to 6.4.4. The documentation toolchain is not copied into either runtime image.
- Refresh digest-pinned Chainguard Go, static and Wolfi bases. Debug still reuses the tested production binary and fails its build if the installed Wolfi glibc package is older than 2.44-r8.
- Debug tooling is downloaded in a build-only GNU-wget stage and installed offline against the signed APK index. The final image contains no download-stage Python/tools and does not blindly upgrade all base packages.
- Remove organization-specific branding from image labels and the embedded dashboard; no private configuration is introduced.

## Validation and publication

Official npm bulk advisory checks returned no advisory packages for the documentation lockfile. OSV batch checks of the 64 modules listed in go.mod and all 85 downloaded module versions recorded in go.sum returned no vulnerable modules; these are dependency checks, not a claim that every possible application issue or image vulnerability has been eliminated.

Both images must pass their build and non-root startup checks before publication. The Dockerfile runs `CGO_ENABLED=0 go test -trimpath -p=2 -mod=readonly ./...` in the builder before producing the release binary, using the same static/trimpath compilation mode as the final application. Build parallelism is bounded to avoid exhausting memory on shared builders. Cluster end-to-end tests need an external Kubernetes cluster and are not implied by an image build.

Release validation completed: all eight Go packages passed their unit tests; production and debug both started under arbitrary UID 1000820000 with a read-only root filesystem. Debug reported glibc 2.44-r8. Fresh-database Trivy scans of both published image digests returned zero OS/library findings at scan time; this is not a guarantee against future advisories.

Only versioned tags `mohankrishna999/goldilocks:v54` and `mohankrishna999/goldilocks:v54-debug` are in scope. Do not publish `latest` or date-based production tags.
