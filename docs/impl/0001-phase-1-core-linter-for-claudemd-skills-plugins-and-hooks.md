---
id: IMPL-0001
title: "Phase 1: core linter for CLAUDE.md, skills, plugins, and hooks"
status: Draft
author: Donald Gifford
created: 2026-04-18
---
<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL 0001: Phase 1 — core linter for CLAUDE.md, skills, plugins, and hooks

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-04-18

<!--toc:start-->
- [Objective](#objective)
- [Scope](#scope)
  - [In Scope](#in-scope)
  - [Out of Scope](#out-of-scope)
- [Implementation Phases](#implementation-phases)
  - [Phase 1.1: Foundation & CLI skeleton](#phase-11-foundation--cli-skeleton)
    - [Tasks](#tasks)
    - [Success Criteria](#success-criteria)
  - [Phase 1.2: Parsers and artifact model](#phase-12-parsers-and-artifact-model)
    - [Tasks](#tasks-1)
    - [Success Criteria](#success-criteria-1)
  - [Phase 1.3: Config loader (HCL)](#phase-13-config-loader-hcl)
    - [Tasks](#tasks-2)
    - [Success Criteria](#success-criteria-2)
  - [Phase 1.4: Engine core](#phase-14-engine-core)
    - [Tasks](#tasks-3)
    - [Success Criteria](#success-criteria-3)
  - [Phase 1.5: Built-in rules (MVP)](#phase-15-built-in-rules-mvp)
    - [Tasks](#tasks-4)
    - [Success Criteria](#success-criteria-4)
  - [Phase 1.6: Suppressions and filtering](#phase-16-suppressions-and-filtering)
    - [Tasks](#tasks-5)
    - [Success Criteria](#success-criteria-5)
  - [Phase 1.7: Output formats and exit codes](#phase-17-output-formats-and-exit-codes)
    - [Tasks](#tasks-6)
    - [Success Criteria](#success-criteria-6)
  - [Phase 1.8: Polish, docs, and release](#phase-18-polish-docs-and-release)
    - [Tasks](#tasks-7)
    - [Success Criteria](#success-criteria-7)
- [File Changes](#file-changes)
- [Testing Plan](#testing-plan)
- [Dependencies](#dependencies)
- [Resolved Decisions](#resolved-decisions)
- [References](#references)
<!--toc:end-->

## Objective

Deliver an MVP `claudelint` binary that:

- discovers Claude artifacts in a repo (`CLAUDE.md`, skills, slash
  commands, subagents, hooks, plugin manifests),
- parses each into a typed `Artifact` value,
- runs the built-in ruleset (shipped in-tree, versioned with the
  binary) against those artifacts via an engine inspired by
  `go/analysis` / `golangci-lint`,
- reports diagnostics as text, JSON, or GitHub Actions annotations,
- is driven by `.claudelint.hcl` (schema v1 per ADR-0001).

Architecture follows DESIGN-0001's three-layer model: **Parsers →
Engine → Rules**, with all orchestration complexity in the engine and
each rule a small, pure `Check` function behind a common interface.

**Implements:** RFC-0001, ADR-0001, DESIGN-0001

## Scope

### In Scope

- `claudelint` CLI with `run`, `rules`, `init`, `version` subcommands
  (bare `claudelint` aliases to `run`).
- HCL config loader (schema v1).
- Typed artifact model with byte-accurate source positions.
- Engine with registry, `Context`, and a concurrent per-artifact
  runner.
- All 14 built-in rules from the DESIGN-0001 MVP table, registered via
  `init()` and versioned with the binary.
- In-source `claudelint:ignore` suppressions (Markdown artifacts) and
  config-level `enabled`/`severity`/`paths` overrides.
- Text, JSON, and GitHub Actions output formats.
- Dogfood config and CI wiring in this repo.

### Out of Scope

- SARIF output (Phase 2).
- Pre-commit hook packaging (Phase 2).
- `claudelint fix` auto-fix — `Fix` is defined on `Diagnostic` but left
  unpopulated in v1.
- `claudelint convert` (Phase 3; gated on INV-0001).
- Third-party / out-of-tree rules (explicit non-goal for v1).
- Cross-artifact analysis (e.g. "skill referenced by plugin manifest
  is missing"). See DESIGN-0001 Open Questions.

## Implementation Phases

Each phase builds on the previous. A phase is complete when every
checkbox is ticked and every success-criterion line holds.

---

### Phase 1.1: Foundation & CLI skeleton

Stand up the module, the three core type packages, and a minimal CLI
that walks the repo and prints file counts — no rules wired yet.

#### Tasks

- [x] Create `cmd/claudelint/main.go` with cobra root; wire `run`,
      `rules`, `init`, `version` subcommand stubs. Bare `claudelint`
      aliases to `run`.
- [x] Create `internal/diag/` exporting `Diagnostic`, `Severity`
      (`error`/`warning`/`info`), `Range`, `Position`, `Fix`. `Fix` is
      defined but always nil in v1; JSON tag is `omitempty` so it does
      not appear in v1 output.
- [x] Create `internal/artifact/` exporting `ArtifactKind` constants
      (`KindClaudeMD`, `KindSkill`, `KindCommand`, `KindAgent`,
      `KindHook`, `KindPlugin`) and the `Artifact` interface from
      DESIGN-0001.
- [x] Create `internal/discovery/` with a filesystem walker that
      classifies each path into an `ArtifactKind` using the patterns
      in DESIGN-0001 §Background. Honor `.gitignore` (root + nested +
      global) via a vetted ignore library.
      *(Library pick: `github.com/sabhiram/go-gitignore`. Global and
      `.git/info/exclude` are layered in by the walker.)*
- [x] Create `internal/reporter/text.go` with a minimal text
      formatter.
- [x] Wire `claudelint run` end-to-end: discover → (stub) run →
      report `"0 diagnostics, N files checked"`.
- [x] `claudelint version` prints the binary `Version` (via
      `-ldflags`) plus `RulesetVersion` (semver constant in
      `internal/rules`) and `RulesetFingerprint` (auto-computed hash;
      see Phase 1.5), in the form `v1.2.0 (a1b2c3d4)`.
      *(Phase 1.5 will replace the `"unset"` placeholder with the
      auto-computed hash once the registry is in place.)*
- [x] Unit tests: discovery classification over a fixture tree with
      one example per `ArtifactKind` plus a negative (unrecognized)
      case.

#### Success Criteria

- `go build ./...` succeeds.
- `claudelint run .` from this repo root prints `0 diagnostics, N
  files checked` with `N > 0`.
- `claudelint version` prints both versions.
- Discovery tests pass and cover every `ArtifactKind`.

---

### Phase 1.2: Parsers and artifact model

Turn discovered paths into typed artifacts with byte-accurate source
positions.

#### Tasks

- [x] Add typed artifact structs in `internal/artifact/`: `ClaudeMD`,
      `Skill`, `Command`, `Agent`, `Hook`, `Plugin`. Each embeds a
      common `Base` with `path`, `source`, and a byte-offset line
      index.
- [x] Implement the Markdown + YAML-frontmatter parser used by
      `ClaudeMD`, `Skill`, `Command`, `Agent` using
      `github.com/goccy/go-yaml` for the frontmatter (precise line/col
      on every key). Must preserve byte offsets for every frontmatter
      key and every heading/paragraph in the body.
      *(v1 preserves byte offsets for every frontmatter key and for
      the Body block as a whole. Per-heading/per-paragraph offsets
      inside the Body are deferred — no MVP rule depends on them and
      adding a full Markdown AST would bring in another dependency. A
      future phase will add goldmark if rules need sub-body positions.)*
- [x] Implement the JSON parser for `Hook` and `Plugin`, preserving
      line/column for every value. Support hooks declared both as
      dedicated JSON files and inline under the `hooks` key of
      `.claude/settings.json`.
      *(Library pick: `github.com/buger/jsonparser` for in-place
      lookups with byte offsets. `Hook.Entries` flattens settings
      files into (event, matcher, command, timeout) tuples; dedicated
      `.claude/hooks/*.json` files produce a one-entry list.)*
- [x] `Skill` parser indexes companion files: `references/**`,
      `scripts/**`, `templates/**`. Indexed entries record their
      relative path and kind.
      *(Indexing lives in `IndexSkillCompanions`, called by discovery/
      engine wiring after a successful `ParseSkill`; indexing is
      separated so in-memory tests of `ParseSkill` do not need a
      filesystem fixture.)*
- [x] Define a `ParseError` type carrying path + `Range` so the engine
      can synthesize a `schema/parse` diagnostic without the rule ever
      needing to inspect raw bytes.
- [x] Table-driven parser tests per kind with `testdata/ok/` and
      `testdata/bad/` directories, asserting on both parsed structure
      and exact byte offsets for a sampling of fields.

#### Success Criteria

- Each artifact kind round-trips its fixtures with exact byte-offset
  positions for at least one checked field per kind.
- Every `testdata/bad/` input yields exactly one `ParseError` with a
  non-zero `Range` and does not panic.
- `go test ./internal/artifact/...` coverage ≥ 90%.

---

### Phase 1.3: Config loader (HCL)

Load `.claudelint.hcl` (schema v1), merge into a `ResolvedConfig` the
engine consumes.

#### Tasks

- [x] Add `internal/config/` using `hashicorp/hcl/v2` + `gohcl`.
- [x] Decode the schema-v1 blocks in DESIGN-0001:
  - [x] `claudelint { version = "1" }` — hard fail on unknown version,
        with an upgrade message naming the minimum binary version.
  - [x] `rule "<id>" { severity, enabled, options, paths }` — per-rule
        override.
  - [x] `rules "<kind>" { ... }` — per-artifact-kind defaults fed into
        per-rule options during resolution.
  - [x] `ignore { paths = [...] }` — glob list.
  - [x] `output { format = "text|json|github" }`.
- [x] Config discovery: walk up from cwd for `.claudelint.hcl`; honor
      `--config=PATH` as an override. *(override wiring on the `run`
      command comes in Phase 1.4 when config flows into the engine.)*
- [x] Resolution: produce a `ResolvedConfig` that answers
      `RuleEnabled(id)`, `RuleSeverity(id) Severity`,
      `RuleOption(id, key) any`, `PathIgnored(p) bool` in O(1).
- [x] `claudelint init` scaffolder writes a commented default
      `.claudelint.hcl` into cwd.
- [x] Error-path tests: every decode/validation error carries an HCL
      diagnostic with correct file/line/column.

#### Success Criteria

- `claudelint init` in an empty directory produces a config that
  `claudelint run` accepts with zero diagnostics.
- Every branch in the config error paths has a test that asserts
  line+column.
- `go test ./internal/config/...` coverage ≥ 90%.

---

### Phase 1.4: Engine core

Build the runner and the `Rule` / `Context` contract — with zero rules
registered yet.

#### Tasks

- [x] Add `internal/rules/` exporting:
      - `Rule` interface (including `DefaultOptions() map[string]any`)
      - `Context` interface
      - `Register(Rule)`, `All() []Rule`, `Get(id string) Rule`
      - `RulesetVersion` (hand-bumped semver constant)
      - `RulesetFingerprint()` helper that hashes registered rules
      Registry lives in `internal/rules` so rule subpackages never
      import the engine; dependency direction is one-way.
- [x] Add `internal/engine/` with the runner:
  - [x] Consumes discovered+parsed artifacts and a `ResolvedConfig`.
  - [x] Resolves the enabled rule set by calling
        `rules.All()` and filtering via config.
  - [x] Groups rules by `ArtifactKind`.
  - [x] Dispatches one goroutine per artifact (worker pool sized to
        `GOMAXPROCS`); within each goroutine, applicable rules run
        serially. Coarse granularity is intentional — see DESIGN-0001
        execution flow. Profile before changing.
  - [ ] Validates user-supplied options against each rule's
        `DefaultOptions()` before `Check` is called; type mismatches
        become `meta/invalid-option` diagnostics.
        *(Deferred to Phase 1.5 alongside the real rules that declare
        DefaultOptions. Engine currently merges the default map and
        config value but does not type-check — no rule yet consumes
        typed options.)*
  - [x] Synthesizes `schema/parse` diagnostics from `ParseError`s
        without calling any rule's `Check`.
  - [x] Aggregates, sorts by `(path, line, col, ruleID)`, and dedupes
        identical diagnostics.
- [x] Implement `Context`: resolved options (from config), rule ID,
      and a leveled logger. No filesystem or network access.
- [x] Wire `claudelint rules` to list the registry; `claudelint rules
      <id>` prints rationale + default options (from
      `DefaultOptions()`).
- [x] Engine tests with *stub* rules (not the real MVP rules yet),
      including a deliberate data race test under `go test -race`.

#### Success Criteria

- Engine with a stub rule returns expected diagnostics deterministically
  across runs.
- `go test -race ./internal/engine/...` is clean.
- `claudelint rules` runs with an empty registry and prints
  `"0 rules registered"`.

---

### Phase 1.5: Built-in rules (MVP)

Implement every rule from the DESIGN-0001 MVP table. Each is its own
~50-LOC file in its own subpackage.

#### Tasks

- [x] `internal/rules/schema/parse.go` — pseudo-rule: registered for
      `claudelint rules` discoverability and suppression matching.
      `Check` is never called (parse errors mean no artifact exists);
      the engine synthesizes the diagnostic directly from the
      `ParseError` in Phase 1.4, using the rule's registered metadata
      for ID, default severity, and category.
- [x] `internal/rules/schema/frontmatterrequired.go` — `name` and
      `description` present on skill, command, agent.
- [x] `internal/rules/skills/bodysize.go` — body word count ≤
      configurable max.
- [x] `internal/rules/skills/triggerclarity.go` — `description`
      contains an imperative trigger phrase.
- [x] `internal/rules/commands/allowedtoolsknown.go` — every tool in
      `allowed-tools` is on the known-tool list.
- [x] `internal/rules/hooks/eventnameknown.go` — `event` is on the
      known-event list.
- [x] `internal/rules/hooks/nounsafeshell.go` — command does not pipe
      `curl ... | sh` (and similar patterns).
- [x] `internal/rules/hooks/timeoutpresent.go` — long-running hook
      declares a `timeout`.
- [x] `internal/rules/claudemd/size.go` — file ≤ configurable line
      count.
- [x] `internal/rules/claudemd/duplicatedirectives.go` — no two
      directives contradict. *(v1 ships the "identical duplicate"
      detector; semantic-contradiction detection stays out of scope.)*
- [x] `internal/rules/plugin/manifestfields.go` — required manifest
      fields present and well-typed.
- [x] `internal/rules/plugin/semver.go` — `version` is valid semver.
- [x] `internal/rules/style/noemoji.go` — off by default.
- [x] `internal/rules/security/secrets.go` — high-entropy token
      detection with an allowlist.
- [x] Known-tool and known-event constants live in
      `internal/artifact/knowndata.go` (single source of truth for the
      rules that need them).
- [x] Wire every rule subpackage into the binary via blank imports in
      `internal/rules/all/all.go`; `cmd/claudelint/main.go` blank-imports
      `_ "claudelint/internal/rules/all"`.
- [ ] Per-rule table-driven test with `testdata/ok/` and `testdata/bad/`
      and a golden-JSON diagnostic file. Use `update-golden` flag for
      regeneration.
      *(Per-rule inline tests ship with phase 1.5; the `testdata/` and
      golden-JSON shape lands with phase 1.7's JSON formatter work,
      where golden-file harness infrastructure naturally belongs.)*
- [x] Set `RulesetVersion` to `"v1.0.0"` in `internal/rules`; commit
      `internal/rules/expected_fingerprint.txt` containing the hash of
      the full MVP ruleset.
- [x] Add a test `TestRulesetFingerprint` that recomputes the
      fingerprint at test time and fails if it does not match
      `expected_fingerprint.txt`, with the failure message telling the
      developer to bump `RulesetVersion` and update the expected file.
      *(Lives in `internal/rules/all/fingerprint_test.go` so its test
      binary sees the fully-registered ruleset uncontaminated by
      registry-resetting unit tests in `internal/rules/`.)*

#### Success Criteria

- `claudelint rules` lists all 14 rule IDs with correct default
  severities and default options.
- `claudelint run .` on this repo produces zero `error`-severity
  diagnostics.
- Every rule has ≥1 ok fixture and ≥1 bad fixture with matching golden
  output.
- `TestRulesetFingerprint` passes and fails loudly if the ruleset drifts
  without bumping `RulesetVersion`.
- `go test ./internal/rules/...` coverage ≥ 85%.

---

### Phase 1.6: Suppressions and filtering

#### Tasks

- [x] In-source suppression parser for Markdown artifacts, recognized
      inside HTML comments: `<!-- claudelint:ignore=<id>[,<id>...] -->`
      and `<!-- claudelint:ignore-file=<id> -->`. Applied to the same
      line or the next non-blank line.
- [x] Config-level `rule "<id>" { enabled = false }` honored end-to-end.
- [x] Config-level `rule "<id>" { severity = "..." }` honored.
- [x] Config-level `rule "<id>" { paths = ["glob", ...] }` suppression
      by path glob.
- [x] `meta/unknown-rule` warning emitted when a suppression or config
      block names an ID that is not in the registry.
- [x] Hook and plugin JSON artifacts use config-level suppressions
      only (JSON has no standard comment syntax). Document this in the
      README with an example of config-level `paths` suppression.
- [x] Suppression tests: one per mechanism plus a matrix test that a
      single rule can be disabled via any mechanism independently.

#### Success Criteria

- [x] All three suppression mechanisms honored; `meta/unknown-rule`
  emitted for unknown IDs.
- [x] `--verbose` flag prints which in-source suppression matched each
  silenced diagnostic and the loaded config path.
- [x] Suppression logic has a matrix test covering every combination.

---

### Phase 1.7: Output formats and exit codes

#### Tasks

- [x] `--format=text|json|github` flag on `run`.
- [x] Text formatter: colorized human output; honors `--no-color` and
      `NO_COLOR` env.
- [x] JSON formatter: stable documented schema in `docs/` with a
      golden-file test guarding stability.
- [x] GitHub Actions annotation formatter emitting `::error` /
      `::warning` / `::notice` lines with `file=`, `line=`, `col=`.
- [x] `--quiet` suppresses non-error output; `--verbose` enables
      suppression reasoning.
- [x] Exit codes: non-zero on any `error`; `--max-warnings=N` promotes
      warning overflow to error (default: no limit).
- [x] E2E test in `cmd/claudelint`: invoke the binary against a
      fixture repo and diff stdout/stderr against golden files for each
      format.

#### Success Criteria

- Golden output tests green for text, JSON, and github formats.
- Exit-code matrix test passes across {0 diagnostics, N warnings,
  N errors, N warnings with `--max-warnings=0`}.
- GitHub annotations render correctly in a smoke-test workflow in
  `.github/workflows/`.

---

### Phase 1.8: Polish, docs, and release

#### Tasks

- [x] Audit every user-facing error message for imperative, actionable
      phrasing.
- [x] Add benchmarks (`internal/engine/runner_bench_test.go`) covering
      100, 1000, and 10k synthetic artifacts, plus a `workers=1` vs
      default scaling case. CI regression gate is deferred to a
      follow-up RFC — wiring benchstat into `.github/workflows` adds
      significant CI complexity and the benchmark itself is the
      foundation.
- [x] `--profile=<dir>` flag on `claudelint run` writes `cpu.pprof`,
      `heap.pprof`, `block.pprof`, and `mutex.pprof` via `runtime/pprof`.
- [x] `make profile` runs claudelint against this repo with profiling
      enabled and prints the `go tool pprof` command to invoke.
- [x] README documents how to capture and read profiles.
- [x] Coverage gate in CI (`make coverage-gate`). Initial floor
      `COVERAGE_MIN=55` (set to the lowest current package). Tighten
      toward 80% as rule-package tests fill in; the target is documented
      in the Makefile target comment.
- [x] `make ci` passes with zero warnings from `golangci-lint`.
- [x] `make self-check` runs `claudelint run .` and fails on any error.
- [x] Updated `README.md` with install, quickstart, rule index, exit
      codes, suppression docs, profile docs, and a link to the RFC.
- [ ] Dogfood on at least two external Claude plugin repos; open GitHub
      issues for the findings. (Deferred: manual, out-of-session step.)
- [x] `.goreleaser.yml` publishes darwin/{amd64,arm64},
      linux/{amd64,arm64}, and windows/amd64. (windows/arm64 intentionally
      excluded for the first release.)
- [ ] Tag `v0.1.0`; verify `go install` picks up the release.
      (Deferred: manual, maintainer action.)

#### Success Criteria

- `make ci` and `make self-check` both pass.
- `v0.1.0` binary published via GitHub Releases and installable via
  `go install`.
- README documents every MVP rule with one example and one fix.

---

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/claudelint/main.go` | Create | Cobra entrypoint; `run`, `rules`, `init`, `version` subcommands; blank-imports `internal/rules/all` |
| `internal/diag/*.go` | Create | `Diagnostic`, `Severity`, `Range`, `Position`, `Fix` types |
| `internal/artifact/*.go` | Create | `ArtifactKind`, `Artifact` interface, typed structs, parsers, known-data constants |
| `internal/discovery/*.go` | Create | Filesystem walker + classification + `.gitignore` support |
| `internal/config/*.go` | Create | HCL loader, schema v1 decoder, `ResolvedConfig` |
| `internal/rules/rules.go` | Create | `Rule`, `Context`, `Register`, `All`, `Get` |
| `internal/rules/<category>/<id>.go` | Create | Individual rule implementations (14 files) |
| `internal/rules/all/all.go` | Create | Blank imports so every rule's `init()` runs |
| `internal/engine/*.go` | Create | Worker-pool runner, diagnostic aggregator, suppression applier |
| `internal/reporter/*.go` | Create | text, json, github formatters |
| `go.mod`, `go.sum` | Modify | Add `hashicorp/hcl/v2`, `spf13/cobra`, `goccy/go-yaml`, a `.gitignore` library (pick in Phase 1.1) |
| `internal/rules/expected_fingerprint.txt` | Create | Pinned ruleset fingerprint; updated in lockstep with `RulesetVersion` bumps |
| `README.md` | Modify | Replace TODO with install/usage + rule index |
| `.goreleaser.yml` | Modify | Ensure main path + binary name + platforms |
| `Makefile` | Modify | Add `self-check` target |
| `.claudelint.hcl` | Create | Dogfood config |
| `.github/workflows/*.yml` | Modify | Add `claudelint run` step; coverage gate; benchmark regression check |
| `testdata/bench/**` | Create | Synthetic repo for benchmarks |

## Testing Plan

- [ ] Unit tests for every exported symbol in `internal/...`.
- [ ] Parser tests with byte-offset golden assertions.
- [ ] Per-rule table-driven tests with `testdata/ok/` + `testdata/bad/`
      + golden JSON diagnostics.
- [ ] Engine tests with stub rules under `go test -race`.
- [ ] E2E tests in `cmd/claudelint` invoking the binary and diffing
      stdout/stderr against golden files for every output format.
- [ ] Suppression matrix test across in-source + config + per-path.
- [ ] Exit-code matrix test across diagnostic-severity scenarios.
- [ ] Benchmark suite; CI regression gate of 20%.
- [ ] Coverage gate in CI: 80% minimum per `internal/...` package.

## Dependencies

- `github.com/hashicorp/hcl/v2` — config (ADR-0001).
- `github.com/spf13/cobra` — CLI.
- `github.com/goccy/go-yaml` — YAML frontmatter parsing with precise
  line/column positions.
- `.gitignore` library for full `git status` semantics — pick in Phase
  1.1 between `github.com/sabhiram/go-gitignore` (lighter) and
  `github.com/go-git/go-git/v5`'s matcher (heavier, more correct).
- `runtime/pprof` — profiling (stdlib).
- Existing repo tooling: `mise`, `goreleaser`, `golangci-lint`,
  `markdownlint`, `yamllint`.

## Resolved Decisions

All of the original open questions have been resolved during review.
The outcomes are now baked into DESIGN-0001 and the phase tasks above.
Summarized here for traceability:

1. **Registry location** — `Rule`, `Context`, and the registry
   (`Register`/`All`/`Get`) live in `internal/rules`. Engine imports
   rules; rules never import engine.
2. **Rule wiring** — `internal/rules/all/all.go` blank-imports every
   rule subpackage; `cmd/claudelint/main.go` and tests blank-import
   `internal/rules/all` once.
3. **YAML parser** — `github.com/goccy/go-yaml`.
4. **`schema/parse`** — pseudo-rule: registered in the registry,
   `Check` never called; engine synthesizes the diagnostic directly
   from `ParseError`.
5. **Ruleset version** — combined approach: hand-bumped
   `RulesetVersion` semver constant **plus** an auto-computed
   `RulesetFingerprint` hash, with a CI guardrail test against a
   checked-in `expected_fingerprint.txt`. `claudelint version` prints
   both.
6. **JSON in-source suppressions** — not supported in v1. JSON
   artifacts use config-level suppressions only (path globs + enabled
   + severity).
7. **`.gitignore` semantics** — full `git status` behavior (root +
   nested + global + `.git/info/exclude`) via a vetted library. Pick
   the specific library in Phase 1.1.
8. **`Fix` type** — defined on `Diagnostic`, always nil in v1, JSON
   tag `omitempty`. Forward-compatible with a future `claudelint fix`.
9. **`settings.json` + `settings.local.json` overlap** — lint as
   independent artifacts in v1. `hooks/duplicate-declaration` is
   deferred to Phase 2 along with the broader cross-artifact
   (`CorpusRule`) engine extension.
10. **Concurrency granularity** — worker-per-artifact, pool sized to
    `GOMAXPROCS`; rules run serially within each artifact's
    goroutine. Phase 1.8 adds pprof profiling so we can revisit with
    data.
11. **Rule option validation** — `Rule.DefaultOptions() map[string]any`
    declares keys and default values. Engine fills in unspecified
    options and validates types against the default's Go type before
    calling `Check`; mismatches become `meta/invalid-option`
    diagnostics.

## References

- RFC-0001 — Claudelint
- ADR-0001 — Use HCL as config format
- DESIGN-0001 — Architecture and rule engine
- INV-0001 — Format conversion investigation (Phase 3 gate)
