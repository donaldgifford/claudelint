# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

Phase 1 MVP shipped (v0.0.1). Phase 2 in progress on branch `docs/impl-0002-phase-2`. Phase 2.1-2.6 complete: two new artifact kinds (`KindMarketplace`, `KindMCPServer`), eight marketplace rules, six MCP rules, `Rule.HelpURI()`, `claudelint rules --json`, and `--format=sarif` with vendored SARIF 2.1.0 schema validation. Phase 2.7 (Docker), 2.8 (release+dogfood), 2.9 (companion Action repo) remain. Ruleset is now at `v1.1.0`.

`run` supports `--format=text|json|github|sarif`, `--sarif-file=<path>`, `--quiet`, `--verbose`, `--max-warnings=N`, `--no-color`, `--profile=<dir>` (pprof), and exit codes (0/1/2). `make self-check`, `make coverage-gate`, `make bench`, and `make profile` are all wired. Phase 1 dogfooding captured in INV-0003.

The architecture and phased rollout are specified in `docs/` — **read the docs before writing code**:

- `docs/rfc/0001-*.md` — the proposal (why claudelint exists, scope, phases)
- `docs/adr/0001-*.md` — HCL chosen as the config format
- `docs/design/0001-*.md` — Phase 1 architecture, interfaces, package layout, built-in rules, ruleset versioning
- `docs/design/0002-*.md` — Phase 2 architecture: marketplaces, MCP rules, GitHub Action, SARIF
- `docs/impl/0001-*.md` — Phase 1 task breakdown
- `docs/impl/0002-*.md` — Phase 2 task breakdown with success criteria per sub-phase

When continuing Phase 2, follow IMPL-0002 in order. Do not improvise architecture that contradicts DESIGN-0002 without updating the doc first.

## Architecture (target)

Three layers, one-way dependency, golangci-lint / go/analysis shape:

```
Parsers → Engine → Rules
```

- **Parsers** (`internal/artifact/`) turn bytes into typed `Artifact` values.
- **Engine** (`internal/engine/`) owns discovery, config, scheduling, concurrency, suppression, and reporting — this is where the complexity lives.
- **Rules** (`internal/rules/<kind>/`) are small (~50 LOC), pure, focused, and implement one `Rule` interface from `internal/rules`. Each rule file runs `Register()` in `init()`. `internal/rules/all/` blank-imports every subpackage so registration happens.

Rule packages must not import the engine. Rules are **built-in to the binary and versioned with it** — no third-party plugin rules in v1. The linted artifact kinds are `KindClaudeMD`, `KindSkill`, `KindCommand`, `KindAgent`, `KindHook`, `KindPlugin`, `KindMarketplace`, `KindMCPServer`. Every `Rule` implements `HelpURI() string`; `rules.DefaultHelpURI(id)` gives a free README anchor.

Cobra subcommands live in `internal/cli/` (one file per command). `cmd/claudelint/main.go` is deliberately thin: it only translates `-ldflags` version/commit into a `cli.BuildInfo` and calls `cli.Execute`. This keeps the CLI testable without spawning a process.

Key decisions already locked in (see IMPL-0001 "Resolved Decisions"):

- HCL v2 (`hashicorp/hcl/v2`) for config
- `github.com/goccy/go-yaml` for YAML (precise line/column)
- `github.com/sabhiram/go-gitignore` for `.gitignore` matching; discovery layers root + nested + global + `.git/info/exclude` on top
- Cobra for the CLI; subcommands are `run`, `rules`, `init`, `version`, `convert` (convert is Phase 3, gated on INV-0001)
- Concurrent runner, worker pool sized to `GOMAXPROCS`
- Ruleset versioning: semver constant + auto-computed fingerprint hash, with a CI guardrail test
- Suppressions: Markdown-only in-source (`<!-- claudelint:ignore=<id> -->`); config-level for JSON
- `schema/parse` is registered as a pseudo-rule but synthesized by the engine from `ParseError`
- pprof profiling is a Phase 1 requirement, not a nice-to-have
- SARIF 2.1.0 output is the CI-facing format; schema is vendored under `internal/reporter/testdata/` so `make ci` stays network-free. Validator is `github.com/santhosh-tekuri/jsonschema/v5` (test-only dep).
- Project-scoped `.mcp.json` files use the top-level key `servers{}` per DESIGN-0002 (not `mcpServers{}`; see §2.2 for rationale — if Claude Code standardizes on `mcpServers`, revisit both the parser and the design doc).

## Common commands

Everything funnels through `make`. The CLI is invoked as `claudelint run` (not `lint`).

- `make build` — build `build/bin/claudelint` with version/commit ldflags
- `make test` — `go test -v -race ./...`
- `make test-pkg PKG=./internal/rules/skills` — test a single package
- `make test-coverage` — race + coverage profile to `coverage.out`
- `make test-report` — coverage + open HTML report
- `make lint` / `make lint-fix` — `golangci-lint run` (config in `.golangci.yml`, Uber Go Style Guide)
- `make fmt` — `gofmt -s -w` + `goimports -local github.com/donaldgifford`
- `make check` — lint + test (pre-commit gate)
- `make ci` — lint + test + build + license-check (matches CI)
- `make release-local` — dry-run goreleaser with `--snapshot --skip=publish --skip=sign`

Running a single Go test: `go test -run TestFoo ./internal/rules/skills` (or use `make test-pkg`).

## Tooling

- `mise.toml` pins Go (currently `1.26.1`), linters, formatters, and dev tools. Run `mise install` to bootstrap. When adding a tool to the project, update `mise.toml` rather than installing it globally.
- `.golangci.yml` is derived from Uber's Go Style Guide — do not relax rules casually; prefer fixing the code.
- Commit-time hooks are not configured; CI runs on push/PR to `main` via `.github/workflows/ci.yml` (labeler + lint + test).

## Documentation workflow (docz)

Docs are managed by the `docz` CLI (config in `.docz.yaml`). Six doc types: `rfc`, `adr`, `design`, `impl`, `plan`, `investigation` — each has its own `id_prefix`, status set, and `README.md` index table.

- `docz create <type>` to add a new doc with correct frontmatter and numbering
- `docz update` to regenerate every `README.md` index table **and** in-file ToCs. Run this after editing any doc — stale indexes fail PR review.
- docz's ToC generator is brittle around unbalanced triple-backticks at line start. If a ToC truncates mid-document, replace fenced code blocks inside that section with 4-space-indented code blocks.
- `wiki.auto_update: true` means `docz update` also refreshes `mkdocs.yml` nav.

Use `Skill` with the `docz:*` skills for doc lifecycle work rather than reinventing the frontmatter by hand.

## Git / PR conventions

- Branch prefixes drive auto-labeling (`.github/labeler.yml`): `feat/`, `fix/`, `chore/`, `docs/`, `bug/`. Use the `git-workflow:branch` skill.
- **Releases are label-driven, not manual.** `.github/workflows/release.yml` runs `jefflinse/pr-semver-bump` on every push to `main`; the merged PR's label (`major` / `minor` / `patch` / `dont-release`) determines the version bump and tag. Do **not** run `git tag` or `make release TAG=...` by hand — it would race the workflow. (`RELEASE.md` predates this and is stale for Phase 1; keep that in mind when reading it.)
- Release assets are defined in `.goreleaser.yml`; `.codecov.yml` gates coverage reporting.
- **No Docker distribution in Phase 1.** The docker-build CI job and the `docker:` release job were removed (they referenced a nonexistent `docker-bake.hcl`). If Docker comes back in a later phase, a real `docker-bake.hcl` + `Dockerfile` need to land first.
- **GPG release signing:** the `GPG_PRIVATE_KEY` repo secret must be the output of `gpg --armor --export-secret-keys <keyid>` (begins with `-----BEGIN PGP PRIVATE KEY BLOCK-----`). The public-export variant imports cleanly but fails at sign-time with `gpg: skipped "***": No secret key`. `GPG_FINGERPRINT` is also required.
