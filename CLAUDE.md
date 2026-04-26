# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project status

Phase 1 MVP shipped as v0.0.1. Phase 2 shipped as **v0.1.0** (PR [#9](https://github.com/donaldgifford/claudelint/pull/9) merged 2026-04-25 → release published the same day). Patch **v0.1.1** shipped via PR [#18](https://github.com/donaldgifford/claudelint/pull/18) — fixed the `hooks/timeout-present` false-positive on plugin `hooks/hooks.json` (issue #14). PR [#19](https://github.com/donaldgifford/claudelint/pull/19) is queued as **v0.2.0** with three more `claude-skills` migration items (issues #15/#16/#17).

After PR #19 merges, ruleset will be `v1.2.0`, fingerprint `e7f26796`, with the catalog covering: 8 marketplace rules, 7 MCP rules (added `mcp/server-allowlist`), 4 hook rules, 3 skill rules (added `skills/no-version-field`), plus the original schema/security/style/claude_md/commands/plugin rules. (IMPL-0002 calls Phase 2 "v0.2.0" throughout — that was the planning name; semver math from `v0.0.1 + minor` actually produces `v0.1.0`. The `v0.2.0` queued via PR #19 is a coincidence with that name.)

Phase 2 delivered: two new artifact kinds (`KindMarketplace`, `KindMCPServer`), the marketplace + MCP rule packages, `Rule.HelpURI()`, `claudelint rules --json`, `--format=sarif` with vendored SARIF 2.1.0 schema validation, a multi-arch `ghcr.io/donaldgifford/claudelint` image via goreleaser (tags: `0.1.0`, `v0`, `v0.1`, `latest`), and companion-action scaffolding at `companion/claudelint-action/` ready to push to `donaldgifford/claudelint-action`. INV-0005 captures the `donaldgifford/claude-skills` dogfood pass; two false positives (nested marketplace shape, missing `AskUserQuestion`) were fixed in-flight.

`run` supports `--format=text|json|github|sarif`, `--sarif-file=<path>`, `--quiet`, `--verbose`, `--max-warnings=N`, `--no-color`, `--profile=<dir>` (pprof), and exit codes (0/1/2). `make self-check`, `make coverage-gate`, `make bench`, and `make profile` are all wired. Phase 1 dogfooding captured in INV-0003.

The architecture and phased rollout are specified in `docs/` — **read the docs before writing code**:

- `docs/rfc/0001-*.md` — the proposal (why claudelint exists, scope, phases)
- `docs/adr/0001-*.md` — HCL chosen as the config format
- `docs/design/0001-*.md` — Phase 1 architecture, interfaces, package layout, built-in rules, ruleset versioning
- `docs/design/0002-*.md` — Phase 2 architecture: marketplaces, MCP rules, GitHub Action, SARIF
- `docs/impl/0001-*.md` — Phase 1 task breakdown
- `docs/impl/0002-*.md` — Phase 2 task breakdown with success criteria per sub-phase

Phase 2 is shipped. The next outstanding work is bootstrapping the `donaldgifford/claudelint-action` repo from `companion/claudelint-action/` (instructions in `companion/README.md`) and tagging it `v1.0.0` once its own test workflow is green. After that, Phase 3 (`convert` subcommand, gated on INV-0001) is the next planned phase. When extending an existing phase or planning a new one, follow the corresponding IMPL doc in order; do not improvise architecture that contradicts the matching DESIGN doc without updating it first.

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
- **Hook parser accepts one canonical nested shape only** — see DESIGN-0001 §Hook shape. Settings files, plugin `hooks/hooks.json`, and `.claude/hooks/*.json` all use `{"hooks": {"<EventName>": [{"matcher", "hooks": [...]}]}}`. A dedicated hook file missing the `hooks` key fails parsing loudly. The pre-#14 flat `{event, matcher, command, timeout}` shape was a parser-author assumption and is no longer accepted.
- **Range emission helpers for new rules:** for rules that walk `Source()` bytes (regex matches, etc.), use `artifact.ResolveOffsetRange(src, start, end)` to convert byte offsets to a `diag.Range`. For rules that target a frontmatter key, use `s.Frontmatter.KeyRange("<key>")`. Pre-parsed fields already carry their own ranges (e.g. `MCPServer.NameRange`, `Skill.Body`). File-level `(0,0)` ranges break per-line suppression markers — never emit them for content rules.
- **Opt-in rules are implemented inside the rule, not the engine** — there is no `Rule.Enabled() bool` hook today. Pattern (see `mcp/server-allowlist`): default-enabled rule whose `DefaultOptions` declares the trigger key with `nil`; if the user hasn't supplied it, emit a loud config-error diagnostic per artifact rather than silently no-op'ing. Adding `Rule.Enabled()` for one rule is over-engineering; revisit only if multiple rules need the pattern.

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
- **Squash-merge subject must keep the `(#N)` suffix.** `pr-semver-bump` looks at the merge-commit message for the PR number; if you override `--subject` with `gh pr merge` and drop the suffix, the SHA-based fallback often fails (search-API indexing lag) and the workflow errors out. Either accept GitHub's default subject or include `(#N)` manually.
- Release assets are defined in `.goreleaser.yml`; `.codecov.yml` gates coverage reporting.
- **Docker images** ship from the same goreleaser run as the binaries (Phase 2 onward). Image is `ghcr.io/donaldgifford/claudelint`; tag layout per release is `:<version>` **without leading `v`** (goreleaser strips it on the bare-version tag), plus `:v<major>`, `:v<major>.<minor>`, and `:latest`. Multi-arch manifest covers `linux/amd64` + `linux/arm64`.
- **GPG release signing:** the `GPG_PRIVATE_KEY` repo secret must be the output of `gpg --armor --export-secret-keys <keyid>` (begins with `-----BEGIN PGP PRIVATE KEY BLOCK-----`). The public-export variant imports cleanly but fails at sign-time with `gpg: skipped "***": No secret key`. `GPG_FINGERPRINT` is also required.
