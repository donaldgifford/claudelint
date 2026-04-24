---
id: IMPL-0002
title: "Phase 2 — marketplaces, MCP rules, SARIF, and GitHub Action"
status: Draft
author: Donald Gifford
created: 2026-04-23
---
<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL 0002: Phase 2 — marketplaces, MCP rules, SARIF, and GitHub Action

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-04-23

<!--toc:start-->
- [Objective](#objective)
- [Scope](#scope)
  - [In Scope](#in-scope)
  - [Out of Scope](#out-of-scope)
- [Implementation Phases](#implementation-phases)
  - [Phase 2.1 — Marketplace parser and artifact kind](#phase-21--marketplace-parser-and-artifact-kind)
    - [Tasks](#tasks)
    - [Success Criteria](#success-criteria)
  - [Phase 2.2 — Marketplace discovery and rules/marketplace/](#phase-22--marketplace-discovery-and-rulesmarketplace)
    - [Tasks](#tasks-1)
    - [Success Criteria](#success-criteria-1)
  - [Phase 2.3 — MCP parser and artifact kind](#phase-23--mcp-parser-and-artifact-kind)
    - [Tasks](#tasks-2)
    - [Success Criteria](#success-criteria-2)
  - [Phase 2.4 — rules/mcp/ package](#phase-24--rulesmcp-package)
    - [Tasks](#tasks-3)
    - [Success Criteria](#success-criteria-3)
  - [Phase 2.5 — Rule metadata: help_uri + rules --json](#phase-25--rule-metadata-helpuri--rules---json)
    - [Tasks](#tasks-4)
    - [Success Criteria](#success-criteria-4)
  - [Phase 2.6 — SARIF reporter and --format=sarif](#phase-26--sarif-reporter-and---formatsarif)
    - [Tasks](#tasks-5)
    - [Success Criteria](#success-criteria-5)
  - [Phase 2.7 — Docker image via goreleaser](#phase-27--docker-image-via-goreleaser)
    - [Tasks](#tasks-6)
    - [Success Criteria](#success-criteria-6)
  - [Phase 2.8 — Release v0.2.0 and dogfood](#phase-28--release-v020-and-dogfood)
    - [Tasks](#tasks-7)
    - [Success Criteria](#success-criteria-7)
  - [Phase 2.9 — claudelint-action companion repo](#phase-29--claudelint-action-companion-repo)
    - [Tasks](#tasks-8)
    - [Success Criteria](#success-criteria-8)
- [File Changes](#file-changes)
- [Testing Plan](#testing-plan)
- [Dependencies](#dependencies)
- [Resolved Decisions](#resolved-decisions)
- [References](#references)
<!--toc:end-->

## Objective

Implement Phase 2 of claudelint: **plugin marketplace awareness**,
**MCP server linting**, **SARIF output**, a supplementary **Docker
distribution**, and a companion **GitHub Action** repo. Ship the
result as `claudelint v0.2.0` and `donaldgifford/claudelint-action@v1`.

**Implements:** [DESIGN-0002](../design/0002-phase-2-marketplaces-mcp-rules-and-github-action.md)
— refer there for architecture, rule tables, and Resolved Decisions
(external sources, `command-exists-on-path` severity, Action
distribution shape, SARIF fingerprints).

## Scope

### In Scope

- Two new artifact kinds: `KindMarketplace`, `KindMCPServer`.
- Two new rule packages: `rules/marketplace/` (8 rules),
  `rules/mcp/` (6 rules).
- `internal/reporter/sarif.go` — SARIF 2.1.0 output.
- `--format=sarif` on `claudelint run`.
- `claudelint rules --json` — rule metadata listing (new).
- `Rule.HelpURI()` method + populated per rule.
- Docker image built by goreleaser, published to
  `ghcr.io/donaldgifford/claudelint`.
- `claudelint v0.2.0` release (minor bump).
- `donaldgifford/claudelint-action` companion repo — composite action,
  pinned to the v0.2.0 binary.

### Out of Scope

- Third-party/plugin-loaded rules (DESIGN-0001 decision, still deferred).
- `claudelint convert` (Phase 3, gated on INV-0001).
- Non-GitHub CI integrations beyond the Docker container (GitLab/Jenkins
  recipes live in README, not as first-class features).
- Pre-commit hook (RFC-0001 Phase 2 item, deferred).
- Ruleset deprecation policy (follow-up ADR).
- SARIF `partialFingerprints` (deferred to v0.3.0 per DESIGN-0002 Q4).
- Publishing to GitHub Marketplace listing (follow-up).

---

## Implementation Phases

Phases are sequential — each builds on earlier work. A phase is
complete when every task is checked off and every success criterion is
met. Commit after each numbered phase using conventional-commit
messages, same cadence as IMPL-0001.

---

### Phase 2.1 — Marketplace parser and artifact kind

Add the `Marketplace` artifact type and its JSON parser. No discovery
or rule changes yet — this phase only extends the artifact layer.

#### Tasks

- [x] Add `KindMarketplace` to the `ArtifactKind` enum in
  `internal/artifact/artifact.go`.
- [x] Add `Marketplace` and `MarketplacePlugin` structs to
  `internal/artifact/types.go` per DESIGN-0002 §1.
- [x] Implement `ParseMarketplace(path string, src []byte) (Marketplace, error)`
  in a new `internal/artifact/parse_marketplace.go`, mirroring
  `ParsePlugin` in `parse_json.go`. Use `buger/jsonparser` to capture
  byte offsets for `plugins[].source`.
- [x] Resolve each `plugins[].source` relative to the repo root; set
  `Resolved = ""` for external (git URL) sources.
- [x] Add frontmatter-free fixtures under
  `internal/artifact/testdata/ok/marketplaces/` (flat, traditional,
  mixed-external layouts) and `testdata/bad/` (malformed JSON,
  missing `plugins`, non-string `source`).
- [x] Add `internal/artifact/parse_marketplace_test.go` — table-driven
  tests covering each fixture and byte-offset assertions.

#### Success Criteria

- `go test ./internal/artifact/...` passes.
- Parsing the three fixture shapes yields correct `Resolved` paths.
- `SourceRange` points at the right byte span in the JSON for a known
  fixture (verified by slicing `Raw`).
- `make lint` passes.

---

### Phase 2.2 — Marketplace discovery and rules/marketplace/

Wire the marketplace manifest into discovery so plugins declared
inside it are walked, and ship the eight rules in the rule table.

#### Tasks

- [x] Add a marketplace **pre-pass** helper to
  `internal/discovery/marketplace.go`. `LoadMarketplaceHints(absRoot)`
  parses `<root>/.claude-plugin/marketplace.json` and returns the
  declared local plugin roots. The walker itself already emits the
  `KindMarketplace` candidate via `Classify()`; the hint is structural
  plumbing for future engine-level on-disk validation.
- [x] ~~Extend `Classify()` to accept explicit plugin roots.~~
  *Dropped.* `classifyPluginLayout()` already handles the traditional
  (`./plugins/foo`), flat (`./`), and versioned plugin-distribution
  shapes. Re-confirmed via smoke test: a marketplace fixture with a
  nested `plugins/donald-loop/.claude/skills/loop/SKILL.md` discovers
  correctly with the existing classifier. The "plugin-root hint"
  parameter would be a dead one.
- [x] Add the optional `marketplace {}` config block to
  `internal/config/schema.go` (manifest override + `only` list).
- [x] Create `internal/rules/marketplace/` with one file per rule
  (~50 LOC each), each registering in `init()`:
  - [x] `marketplace/name` (error) — `name.go`
  - [x] `marketplace/version-semver` (error) — `versionsemver.go`
  - [x] `marketplace/plugins-nonempty` (warn) — `pluginsnonempty.go`
  - [x] `marketplace/plugin-source-valid` (error) —
    `pluginsourcevalid.go` *(missing/empty source; on-disk existence
    check deferred — rules stay pure over the artifact)*
  - [x] `marketplace/plugin-name-unique` (error) —
    `pluginnameunique.go`
  - [x] `marketplace/plugin-name-matches-dir` (warn) —
    `pluginnamematchesdir.go`
  - [x] `marketplace/author-required` (info) — `authorrequired.go`
  - [x] `marketplace/external-source-skipped` (info) —
    `externalsourceskipped.go`
- [x] Add blank import of `rules/marketplace` to
  `internal/rules/all/all.go`.
- [x] Add `internal/rules/marketplace/marketplace_test.go` — table-
  driven tests per rule (ok + bad cases).
- [x] Update the expected ruleset fingerprint in
  `internal/rules/expected_fingerprint.txt` (Phase 1 guardrail will
  flag the new rules; regenerate and commit the new hash). *(initial
  bump to `39d3d488` applied when `KindMarketplace` widened
  `security/secrets`'s `AppliesTo`; will re-regen as more rules
  land.)*
- [x] Bump `RulesetVersion` in `internal/rules/version.go` from
  `v1.0.0` to `v1.1.0` (minor — additive rules).

#### Success Criteria

- `claudelint run` against a fixture marketplace discovers both the
  manifest and every local plugin's artifacts.
- All eight marketplace rules report correct diagnostics on their
  dedicated bad fixtures and stay silent on their good fixtures.
- Ruleset fingerprint test passes after the update.
- `make self-check` passes on this repo (no new diagnostics on
  claudelint itself).

---

### Phase 2.3 — MCP parser and artifact kind

Add `KindMCPServer` and a shared parser that extracts server entries
from both standalone `.mcp.json` and plugin-embedded `mcp.servers{}`.

#### Tasks

- [x] Add `KindMCPServer` to the `ArtifactKind` enum.
- [x] Add `MCPServer` struct to `internal/artifact/types.go` per
  DESIGN-0002 §2 (includes `Embedded bool`).
- [x] Implement `ParseMCPFile(path string, src []byte) ([]*MCPServer, *ParseError)`
  and `ParseMCPEmbedded(pluginPath string, src []byte) ([]*MCPServer, error)`
  in `internal/artifact/parse_mcp.go`. Both delegate to a shared
  `collectServers()` so rule-relevant fields stay consistent.
- [x] Wire `parseOne` so a plugin manifest with `mcp.servers{}` emits
  each embedded server as an independent `KindMCPServer` candidate
  alongside the `Plugin` artifact. `parseOne` now returns a slice so
  one file can fan out cleanly; single-artifact kinds return a
  one-element slice via `wrapOne`.
- [x] Map `.mcp.json` (at any depth) to `KindMCPServer` in
  `Classify()`. Discovery picks it up through the existing walk.
- [x] Fixtures: `testdata/ok/mcp/standalone.json`,
  `testdata/ok/mcp/embedded_in_plugin.json`, plus bad-case
  (`mcp_missing_command.json`, `mcp_nonobject_servers.json`).
- [x] Parser tests in `internal/artifact/parse_mcp_test.go`.

#### Success Criteria

- `go test ./internal/artifact/...` passes.
- Both standalone and embedded paths produce identical
  `MCPServer` structs for an identical server spec.
- `Embedded` is `false` for standalone and `true` for embedded.
- Byte offsets for `command` and each `env[key]` are accurate.

---

### Phase 2.4 — rules/mcp/ package

Ship the six MCP rules. Share the secrets matcher with
`rules/security/` rather than duplicating.

#### Tasks

- [x] Expose a narrow `security.MatchesSecret([]byte) bool` from the
  existing `internal/rules/security/` package so
  `rules/mcp/nosecretsinenv.go` can call it without the regex tables
  leaking out of the `security` package.
- [x] Create `internal/rules/mcp/` with:
  - [x] `mcp/command-required` (error) — `commandrequired.go`
  - [x] `mcp/command-exists-on-path` (warn) — `commandexistsonpath.go`
    *(typo-catcher against a curated runner allowlist; path-qualified
    commands and absolute paths are skipped)*
  - [x] `mcp/no-secrets-in-env` (error) — `nosecretsinenv.go`
  - [x] `mcp/no-unsafe-shell` (error) — `nounsafeshell.go` (mirror
    `hooks/nounsafeshell.go`)
  - [x] `mcp/disabled-commented` (info) — `disabledcommented.go`
  - [x] `mcp/server-name-required` (error) — `servernamerequired.go`
- [x] Blank-import `rules/mcp` from `internal/rules/all/all.go`.
- [x] `internal/rules/mcp/mcp_test.go` — table-driven per rule.
- [x] Update `expected_fingerprint.txt` (now `4cee5ee7`).

#### Success Criteria

- All six rules fire correctly on their dedicated bad fixtures.
- `mcp/no-secrets-in-env` catches the same patterns
  `security/secrets` catches (shared matcher, verified by one shared
  test fixture).
- `mcp/command-exists-on-path` does **not** fire when `command` is an
  absolute path or contains a `/`.
- `make lint` passes.

---

### Phase 2.5 — Rule metadata: help_uri + rules --json

SARIF needs stable per-rule metadata. Add a `HelpURI()` method and a
`claudelint rules --json` listing so the reporter and external tools
read from one source of truth.

#### Tasks

- [x] Extend the `Rule` interface in `internal/rules/rules.go` with
  `HelpURI() string`. Document the convention: URL in
  `README.md` for Phase 2; a dedicated rules docs site later.
- [x] Provide a default via a small embeddable helper (e.g.
  `rules.DefaultHelpURI(id)`) that returns
  `"https://github.com/donaldgifford/claudelint/blob/main/README.md#rule-<id>"`
  so rule authors can just return the default unless they override.
- [x] Touch every existing rule to return a URI (default is fine for
  Phase 1 rules; new Phase 2 rules use the same default).
- [x] Add `--json` flag to `claudelint rules` in
  `internal/cli/rules.go`. Output schema:

  ```json
  {
    "schema_version": "1",
    "ruleset_version": "...",
    "fingerprint": "...",
    "rules": [
      {
        "id": "...",
        "category": "...",
        "default_severity": "...",
        "applies_to": ["skill", "plugin"],
        "help_uri": "...",
        "default_options": {}
      }
    ]
  }
  ```

- [x] Document the new schema in `docs/rules-json-schema.md` (analogous
  to Phase 1's `docs/json-output-schema.md`).
- [x] `internal/cli/rules_test.go` — assert both text and JSON output
  shapes against a fixed subset of rules.

#### Success Criteria

- `claudelint rules --json | jq .rules[0].help_uri` yields a non-empty
  URL for every rule.
- `HelpURI()` has a unit test asserting the default helper resolves to
  a well-formed URL.
- Fingerprint test still passes (the interface change alone does not
  affect the fingerprint because the hash is over ID/Category/etc.,
  not method surface — confirm with `RulesetFingerprint()`).

---

### Phase 2.6 — SARIF reporter and --format=sarif

Produce SARIF 2.1.0 output and wire it into the CLI.

#### Tasks

- [x] Add `internal/reporter/sarif.go` exporting
  `SARIF(w io.Writer, s Summary, opts SARIFOptions) error` (signature
  extended with SARIFOptions so the CLI can thread its BuildInfo to
  `runs[0].tool.driver.version`). Structure mirrors `JSON()`.
  Top-level document includes:
  - `$schema`: `https://json.schemastore.org/sarif-2.1.0.json`
  - `version`: `2.1.0`
  - `runs[0].tool.driver`: `name=claudelint`, `version=<app>`,
    `informationUri`, `rules[]` populated from
    `rulesreg.All()` with `id`, `name`, `shortDescription`,
    `helpUri`, `defaultConfiguration.level` (from severity).
  - `runs[0].results[]`: one entry per diagnostic, with
    `ruleId`, `level`, `message.text`, `locations[0].physicalLocation`
    (artifactLocation.uri + region.startLine/startColumn/endLine/endColumn).
- [x] Severity mapping: `Error → error`, `Warning → warning`,
  `Info → note`.
- [x] Add `formatSARIF` to the format enum in `internal/cli/run.go`;
  update `validateFormat()` and the switch.
- [x] Accept an optional `SARIF_PATH` via `--sarif-file=<path>` so the
  Action can control where the file lands; default is stdout (parity
  with other formats).
- [x] Vendor the SARIF 2.1.0 JSON Schema under
  `internal/reporter/testdata/sarif-2.1.0.json` (keeps `make ci`
  network-free).
- [x] Add `internal/reporter/sarif_test.go` — schema-validation test
  (the vendored schema is loaded and every SARIF doc the reporter
  emits is validated against it). Skipped a byte-for-byte golden file
  in favor of shape assertions — the doc carries a live rules catalog
  so every new rule would re-golden the fixture.

#### Success Criteria

- `claudelint run --format=sarif .` produces valid SARIF on a known
  fixture.
- Output passes schema validation against SARIF 2.1.0.
- Every rule referenced by a `result.ruleId` is present in
  `runs[0].tool.driver.rules[]`.
- Golden file matches byte-for-byte after formatting.

---

### Phase 2.7 — Docker image via goreleaser

Publish `ghcr.io/donaldgifford/claudelint:<tag>` as a supplementary
distribution. Keep the composite Action's binary-download path
independent of this image.

#### Tasks

- [x] Create a repo-root `Dockerfile`. Single-stage on top of
  `gcr.io/distroless/static-debian12:nonroot`, copies the
  goreleaser-built binary to `/usr/local/bin/claudelint`,
  `ENTRYPOINT ["/usr/local/bin/claudelint"]`, `CMD ["run", "."]`.
- [x] Add a `dockers:` stanza to `.goreleaser.yml`:
  - Builds `linux/amd64` and `linux/arm64` images via two `dockers`
    entries with `use: buildx` and per-platform
    `--platform` flags. (Stayed on classic `dockers` rather than
    `dockers_v2`; v2 is still stabilizing. Revisit in a later phase.)
  - Tags: `ghcr.io/donaldgifford/claudelint:{{ .Version }}`,
    `:v{{ .Major }}`, `:v{{ .Major }}.{{ .Minor }}`, `:latest`.
  - OCI labels for `org.opencontainers.image.title`, `.description`,
    `.url`, `.source`, `.licenses` (static in the Dockerfile) and
    `.version`, `.revision`, `.created` (goreleaser-injected).
- [x] Add a `docker_manifests:` stanza to create a multi-arch manifest.
- [x] Add `docker login ghcr.io` + QEMU + buildx setup to
  `.github/workflows/release.yml` before `goreleaser release --clean`
  (using `${{ secrets.GITHUB_TOKEN }}`).
- [x] Add a `make docker-local` target that runs
  `goreleaser release --snapshot --clean --skip=publish,sign
  --skip=validate` and prints the snapshot-tag smoke command. (Final
  run-check blocked by Docker daemon availability in this env; config
  passes `goreleaser check`.)
- [x] Update `README.md` with a "Running in CI" section covering
  docker invocation for non-GitHub runners (GitLab CI, Jenkins,
  generic shell) and the github/codeql-action SARIF-upload pattern.

#### Success Criteria

- `make docker-local` builds both architectures and the image runs
  `claudelint version` successfully.
- `docker inspect` shows the expected OCI labels.
- `README.md` contains copy-pasteable Docker recipes.

---

### Phase 2.8 — Release v0.2.0 and dogfood

Cut the release and validate against real marketplaces before
announcing.

#### Tasks

- [x] Update `CHANGELOG.md` with a v0.2.0 section describing the new
  artifact kinds, rule packages, SARIF output, Docker image, and
  ruleset version bump.
- [ ] Merge the final Phase 2 PR to `main` with the `minor` release
  label so `jefflinse/pr-semver-bump` produces `v0.2.0`. (Deferred
  until Phase 2.9 lands; the release cuts once on the merge to `main`.)
- [ ] Verify the release workflow produces:
  - binaries for darwin/linux/windows × amd64/arm64 (plus windows
    amd64 only);
  - signed checksums;
  - a working `ghcr.io/donaldgifford/claudelint:v0.2.0` image;
  - the image tagged as `:latest` and `:v0`.

  (Deferred — validated at release time, after Phase 2.9.)
- [x] Dogfood `claudelint run` against `donaldgifford/claude-skills`
  (the donald-loop / docz / go-development marketplace) as the
  primary Phase 2 dogfood target.
- [x] Record dogfood findings in
  `docs/investigation/0005-phase-2-dogfood-findings-marketplaces-mcp-and-spec-divergence.md`.
  Primary finding: marketplace manifests in the wild nest `version` and
  `owner` instead of top-level `version`/`author`, so two rules produce
  false positives on conforming real marketplaces. Tracked as follow-up
  for v0.3.0 rather than blocking the ship.

#### Success Criteria

- `claudelint v0.2.0` is tagged on `main` via the release workflow,
  with all expected assets attached.
- `docker run --rm ghcr.io/donaldgifford/claudelint:v0.2.0 version`
  prints `v0.2.0`.
- INV-0005 is written and closed, or the findings are captured as
  follow-up issues for v0.3.0.
- No new false positives surface on the three dogfood targets that
  would block real-world adoption.

---

### Phase 2.9 — claudelint-action companion repo

Stand up `donaldgifford/claudelint-action` as a separate public repo.
Minimal scope here: bootstrap + wire up to v0.2.0. A separate IMPL
doc in *that* repo tracks its own lifecycle.

#### Tasks

- [ ] Create `donaldgifford/claudelint-action` on GitHub (public,
  Apache-2.0, README stub).
- [ ] Scaffold `action.yml` per DESIGN-0002 §3.2 (inputs, outputs).
- [ ] Composite-action steps:
  - [ ] Resolve `version` input (map `latest` to the GitHub API
    "latest release" tag).
  - [ ] Download the matching binary for `runner.os` /
    `runner.arch`.
  - [ ] Verify the checksum against the release's `checksums.txt`.
  - [ ] Invoke `claudelint run` with the supplied `path`, `format`,
    `config`, `max-warnings`.
  - [ ] If `format=sarif` and `upload-sarif=true`, call
    `github/codeql-action/upload-sarif@v4`.
- [ ] `.github/workflows/test.yml` in the action repo: checks out a
  fixture directory (embedded under `fixtures/`) and runs the Action
  against it; asserts on expected outputs (`diagnostics-count`,
  `error-count`).
- [ ] Tag `v1.0.0` + move floating `v1` tag after the test workflow
  passes on `main`.
- [ ] Add a "Quickstart" section to the claudelint README that
  points at the Action.

#### Success Criteria

- Action's own CI runs green on Linux and macOS runners (Windows is
  best-effort; flag as follow-up if it's painful).
- A consumer workflow with `uses: donaldgifford/claudelint-action@v1`
  produces the same diagnostics as running the binary locally.
- SARIF uploads appear under the consuming repo's **Code scanning**
  tab.

---

## File Changes

Rough inventory. Not exhaustive — intent is to make review coverage
obvious. New files are marked Create; edits are Modify.

| File | Action | Description |
|---|---|---|
| `internal/artifact/artifact.go` | Modify | Add `KindMarketplace`, `KindMCPServer` |
| `internal/artifact/types.go` | Modify | Add `Marketplace`, `MarketplacePlugin`, `MCPServer` |
| `internal/artifact/parse_marketplace.go` | Create | Marketplace JSON parser |
| `internal/artifact/parse_mcp.go` | Create | MCP JSON parser (shared by standalone + embedded) |
| `internal/artifact/parse_json.go` | Modify | Extend `ParsePlugin` to emit embedded MCP servers |
| `internal/artifact/testdata/ok/marketplaces/**` | Create | Fixtures |
| `internal/artifact/testdata/ok/mcp/**` | Create | Fixtures |
| `internal/discovery/walk.go` | Modify | Add marketplace pre-pass; include `.mcp.json` in default globs |
| `internal/discovery/classify.go` | Modify | Accept plugin-root hints; classify `.mcp.json` |
| `internal/discovery/marketplace.go` | Create | Marketplace pre-pass helper (if extracted) |
| `internal/config/schema.go` | Modify | Add optional `marketplace {}` block |
| `internal/rules/rules.go` | Modify | Add `HelpURI() string` to `Rule` interface |
| `internal/rules/version.go` | Modify | Bump `RulesetVersion` |
| `internal/rules/help.go` | Create | `DefaultHelpURI(id)` helper |
| `internal/rules/expected_fingerprint.txt` | Modify | Regenerated hash |
| `internal/rules/all/all.go` | Modify | Blank-import new rule packages |
| `internal/rules/marketplace/**` | Create | 8 rule files + test |
| `internal/rules/mcp/**` | Create | 6 rule files + test |
| `internal/rules/security/secrets.go` | Modify | Factor matcher for reuse |
| `internal/reporter/sarif.go` | Create | SARIF 2.1.0 emitter |
| `internal/reporter/sarif_test.go` | Create | Golden + schema test |
| `internal/cli/run.go` | Modify | Add `formatSARIF`, `--sarif-file` flag |
| `internal/cli/rules.go` | Modify | Add `--json` flag |
| `docs/rules-json-schema.md` | Create | Schema doc for `rules --json` |
| `docs/investigation/0005-*.md` | Create | Dogfood investigation |
| `build/docker/Dockerfile` | Create | Distroless/alpine image |
| `.goreleaser.yml` | Modify | `dockers:` + `docker_manifests:` |
| `.github/workflows/release.yml` | Modify | GHCR login before goreleaser |
| `Makefile` | Modify | `make docker-local` |
| `README.md` | Modify | "Running in CI" + Quickstart (Action) |
| `CHANGELOG.md` | Modify | v0.2.0 entry |

Out-of-repo (`donaldgifford/claudelint-action`):

| File | Action | Description |
|---|---|---|
| `action.yml` | Create | Composite action definition |
| `README.md` | Create | Usage + inputs/outputs |
| `.github/workflows/test.yml` | Create | Action E2E |
| `fixtures/**` | Create | Sample dirty project |

## Testing Plan

- **Unit tests per rule** — table-driven, bad + good fixtures, Range
  assertions (IMPL-0001 pattern).
- **Parser tests** — byte-offset assertions on `Raw`.
- **Reporter tests** — JSON and SARIF use golden files under
  `testdata/` for stability; regen via `-update` flag convention.
- **Schema validation** — SARIF output validated against the 2.1.0
  JSON Schema. Vendor the schema or fetch once at test startup per
  Open Question 4.
- **Fingerprint guardrail** — existing test fails on new rules until
  `expected_fingerprint.txt` is updated. This is by design; each
  phase that changes the rule set must regen the hash.
- **Integration (`cmd/claudelint/e2e_test.go`)** — extend to cover
  `--format=sarif`, a marketplace fixture, and an MCP fixture.
- **Action E2E** — lives in the `claudelint-action` repo; asserts
  diagnostics-count/error-count outputs and (optionally) that
  SARIF uploads succeed on a PR-scoped run.
- **Docker smoke** — `make docker-local` runs `claudelint version`
  inside the image and greps for `v0.2.0` (or the snapshot version).

## Dependencies

- **DESIGN-0002** — this doc is the implementation plan for that design.
- **Phase 1 code** — everything here builds on the parser/engine/rule
  architecture shipped in IMPL-0001.
- **External:**
  - `github.com/buger/jsonparser` (already present; used by new parsers).
  - SARIF 2.1.0 JSON Schema (vendored or fetched in tests).
  - GoReleaser `dockers:` and `docker_manifests:` features.
  - `github/codeql-action/upload-sarif@v4` (used by the Action repo).
- **Release prerequisites** (blocking Phase 2.8):
  - GPG signing working end-to-end (see CLAUDE.md GPG gotcha — must be
    re-verified with a fresh `v0.1.0` or `v0.2.0` dry run before cut).
  - `GITHUB_TOKEN` has `write:packages` scope for GHCR push (default
    for `${{ secrets.GITHUB_TOKEN }}` on modern runners, but confirm).

## Resolved Decisions

The six original Open Questions were resolved during implementation
review (2026-04-24):

1. **Ruleset version bump semantics →** bump `RulesetVersion` from
   `v1.0.0` to `v1.1.0` (minor — additive rules), independent of
   claudelint's app version `v0.2.0`. DESIGN-0002 §Testing Strategy
   conflated the two; the ruleset has its own semver trajectory.
2. **Embedded MCP servers →** emit each server as an independent
   `KindMCPServer` candidate from the walker (option b). `rules/mcp/`
   rules only ever see `KindMCPServer` artifacts. Matches how hook
   entries are handled today; no cross-kind rule logic needed. The
   walker gains a small embedding-aware code path in `parse_json.go`.
3. **Secrets matcher sharing →** expose a narrow
   `security.MatchesSecret([]byte) bool` from the existing
   `internal/rules/security/` package (option a). Keeps the regex
   tables owned by one package.
4. **SARIF schema validation →** vendor the SARIF 2.1.0 JSON Schema
   under `internal/reporter/testdata/sarif-2.1.0.json`. Tests stay
   hermetic; `make ci` remains network-free.
5. **Dockerfile location and base image →** repo-root `Dockerfile`
   (standard for `docker build .`, simpler goreleaser config) on
   `gcr.io/distroless/static-debian12`. Users who want a shell can
   run the binary on their own base.
6. **Dogfood target (Phase 2.8) →** `donaldgifford/claude-skills` is
   the primary Phase 2 dogfood target. Additional marketplaces can be
   added opportunistically but are not required to complete the
   phase.

## References

- [DESIGN-0002](../design/0002-phase-2-marketplaces-mcp-rules-and-github-action.md)
  — architecture, rule tables, Resolved Decisions.
- [DESIGN-0001](../design/0001-claudelint-linter-architecture-and-rule-engine.md)
  — Phase 1 architecture (parsers / engine / rules).
- [IMPL-0001](0001-phase-1-core-linter-for-claudemd-skills-plugins-and-hooks.md)
  — Phase 1 task breakdown; template for this doc's shape.
- [INV-0003](../investigation/0003-phase-18-dogfood-findings-on-external-claude-plugins.md)
  — prior dogfood findings; motivates broader layout coverage in
  Phase 2.2.
- [RFC-0001](../rfc/0001-claudelint-go-based-linter-with-hcl-config-for-claude-plugins.md)
  — original Phase 2 scope (SARIF, Actions integration).
- Claude Code plugin-marketplaces spec:
  `https://docs.claude.com/en/docs/claude-code/plugin-marketplaces`.
- MCP specification:
  `https://modelcontextprotocol.io/specification`.
- SARIF 2.1.0:
  `https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html`.
- GoReleaser `dockers` docs:
  `https://goreleaser.com/customization/docker/`.
- GitHub Actions composite actions:
  `https://docs.github.com/en/actions/creating-actions/creating-a-composite-action`.
- Prior art: `stbenjam/claudelint` — marketplace + MCP linting in
  Python.
