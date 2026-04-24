# syntax=docker/dockerfile:1.6
#
# Runtime image for claudelint. Goreleaser provides a pre-built binary
# at build context root (one per target architecture via --build-arg);
# this file only packages it into a distroless base so the image is
# minimal (<10 MB) and has no shell.
#
# Local builds: prefer `make docker-local`, which runs goreleaser in
# snapshot mode and feeds the binary into this file.
FROM gcr.io/distroless/static-debian12:nonroot

ARG TARGETARCH=amd64

# Goreleaser places the binary next to this Dockerfile during `dockers`
# processing. Keep the filename unqualified so amd64 and arm64 builds
# share this stage.
COPY claudelint /usr/local/bin/claudelint

# OCI image metadata. Version/revision are filled in by goreleaser.
LABEL org.opencontainers.image.title="claudelint" \
      org.opencontainers.image.description="Linter for Claude Code artifacts" \
      org.opencontainers.image.url="https://github.com/donaldgifford/claudelint" \
      org.opencontainers.image.source="https://github.com/donaldgifford/claudelint" \
      org.opencontainers.image.licenses="MIT"

ENTRYPOINT ["/usr/local/bin/claudelint"]
CMD ["run", "."]
