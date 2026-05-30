// docker-bake.hcl — multi-arch build pipeline for claudelint.
//
// Targets:
//   - default: local single-arch build (used by `docker buildx bake`)
//   - ci:      linux/amd64 build + push of `:dev-ci` for PR validation
//   - release: multi-arch build + push to GHCR (CI only, gated on tag)
//
// CI workflow consumes this via docker/bake-action@v6 with the `targets`
// input. The release workflow merges in tag-derived image refs from
// docker/metadata-action's bake-file outputs.

variable "REGISTRY" {
  default = "ghcr.io/donaldgifford/claudelint"
}

variable "TAG" {
  default = "dev"
}

variable "VERSION" {
  default = "0.0.0-dev"
}

group "default" {
  targets = ["claudelint"]
}

group "ci" {
  targets = ["claudelint-ci"]
}

group "release" {
  targets = ["claudelint-release"]
}

target "_common" {
  context    = "."
  dockerfile = "Dockerfile"
  args = {
    VERSION = "${VERSION}"
  }
  labels = {
    "org.opencontainers.image.source"      = "https://github.com/donaldgifford/claudelint"
    "org.opencontainers.image.licenses"    = "Apache-2.0"
    "org.opencontainers.image.description" = "claudelint"
    "org.opencontainers.image.title"       = "claudelint"
  }
  // Annotations land on the OCI manifest (and the index, for multi-arch
  // builds). GHCR reads these — not labels — to populate the version
  // page's source-repo link, description, and license badge.
  annotations = [
    "index,manifest:org.opencontainers.image.source=https://github.com/donaldgifford/claudelint",
    "index,manifest:org.opencontainers.image.licenses=Apache-2.0",
    "index,manifest:org.opencontainers.image.description=claudelint",
    "index,manifest:org.opencontainers.image.title=claudelint",
  ]
}

// Stub providing default `tags` for local `docker buildx bake`. CI
// runs override this target via docker/metadata-action's
// bake-file-tags output so the bake pushes the same semver-derived
// image refs the metadata-action emits — which is what cosign then
// signs in the next step. The release target inherits from this and
// does NOT declare tags itself, so the override actually takes
// effect (with HCL inheritance, a child's tags list replaces the
// parent's, not extends it).
target "docker-metadata-action" {
  tags = [
    "${REGISTRY}:${TAG}",
    "${REGISTRY}:latest",
  ]
}

target "claudelint" {
  inherits = ["_common"]
  tags     = ["${REGISTRY}:${TAG}"]
  platforms = [
    "linux/amd64",
  ]
}

// CI builds are linux/amd64 only — emulated arm64 builds via QEMU on
// GitHub's ubuntu-latest runners take ~25 min and dominate PR feedback
// time. Multi-arch coverage is restored in the release target, which
// runs only on tag pushes.
target "claudelint-ci" {
  inherits  = ["_common"]
  tags      = ["${REGISTRY}:${TAG}-ci"]
  platforms = ["linux/amd64"]
  attest = [
    "type=sbom",
    "type=provenance,mode=min",
  ]
}

target "claudelint-release" {
  inherits = ["_common", "docker-metadata-action"]
  // tags intentionally omitted — they come from docker-metadata-action
  // (defaults for local bake; CI overrides via metadata-action).
  platforms = [
    "linux/amd64",
    "linux/arm64",
  ]
  output = ["type=registry"]
  attest = [
    "type=sbom",
    "type=provenance,mode=max",
  ]
}
