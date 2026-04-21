# Releasing claudelint

This is the checklist for cutting a tagged release. Most of the work
happens in CI via goreleaser; the human steps are review, merge, and
tagging.

## Prerequisites

- `gh` CLI authenticated as a maintainer of the repo.
- `goreleaser` installed locally (`mise install`).
- A GPG key matching `$GPG_FINGERPRINT` (see `.goreleaser.yml`).
- `CODECOV_TOKEN` available to CI (already configured).

## Cutting `v0.1.0`

1. Make sure `main` is green: lint, test, build, license-check, and
   coverage-gate all pass on the latest `main` commit.
2. Update `CHANGELOG.md` — move items under `## [Unreleased]` to a
   new `## [0.1.0] - <date>` section and bump the compare link.
3. Dry-run the release locally:

   ```bash
   make release-local
   ```

   Inspect the generated artifacts under `dist/` for each target
   platform (linux/darwin/windows × amd64/arm64 where applicable).

4. Tag and push:

   ```bash
   make release TAG=v0.1.0
   ```

   This runs `git tag -a v0.1.0 -m "Release v0.1.0"` and
   `git push origin v0.1.0`. The push triggers
   `.github/workflows/release.yml`, which invokes goreleaser to build,
   sign, and publish archives to GitHub Releases.

5. Verify the release landed:

   ```bash
   gh release view v0.1.0
   go install github.com/donaldgifford/claudelint/cmd/claudelint@v0.1.0
   claudelint version
   ```

   The `version` subcommand should report `v0.1.0` with a non-dev
   commit hash and the ruleset fingerprint from
   `internal/rules/expected_fingerprint.txt`.

6. Announce the release (optional). Copy the CHANGELOG entry into the
   release notes on GitHub if goreleaser's auto-generated notes need a
   lead paragraph.

## Hotfix releases

For patch-level fixes on an already-released tag:

1. Branch from the tag (`git checkout -b hotfix/v0.1.1 v0.1.0`).
2. Land the fix via PR to a release branch, then cherry-pick to
   `main` once merged.
3. Run `make release TAG=v0.1.1` from the release branch.

## Ruleset fingerprint drift

A rule addition, removal, or metadata change bumps the fingerprint.
CI fails on drift. Resolution:

1. `make build && ./build/bin/claudelint version` — note the new
   fingerprint line.
2. Update `internal/rules/expected_fingerprint.txt` with the new value.
3. Bump `rules.RulesetVersion` per semver (add → minor bump; remove or
   rename → major bump; metadata-only change → patch bump).
4. Commit both changes together with a `feat(rules):` or
   `fix(rules):` prefix so release notes pick it up.
