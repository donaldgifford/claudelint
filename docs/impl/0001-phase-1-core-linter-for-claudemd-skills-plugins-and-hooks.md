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
  - [Phase 1.1: Foundation](#phase-11-foundation)
    - [Tasks](#tasks)
    - [Success Criteria](#success-criteria)
  - [Phase 1.2: Parsers and artifact model](#phase-12-parsers-and-artifact-model)
    - [Tasks](#tasks-1)
    - [Success Criteria](#success-criteria-1)
  - [Phase 1.3: Config loader (HCL)](#phase-13-config-loader-hcl)
    - [Tasks](#tasks-2)
    - [Success Criteria](#success-criteria-2)
  - [Phase 1.4: Rule engine and built-in rules](#phase-14-rule-engine-and-built-in-rules)
    - [Tasks](#tasks-3)
    - [Success Criteria](#success-criteria-3)
  - [Phase 1.5: Output formats and exit codes](#phase-15-output-formats-and-exit-codes)
    - [Tasks](#tasks-4)
    - [Success Criteria](#success-criteria-4)
  - [Phase 1.6: Polish, docs, and release prep](#phase-16-polish-docs-and-release-prep)
    - [Tasks](#tasks-5)
    - [Success Criteria](#success-criteria-5)
- [File Changes](#file-changes)
- [Testing Plan](#testing-plan)
- [Dependencies](#dependencies)
- [References](#references)
<!--toc:end-->

## Objective

Deliver an MVP `claudelint` binary that discovers Claude artifacts in a
repo, parses them into typed structures, runs a built-in rule set loaded
from `.claudelint.hcl`, and prints human-readable diagnostics.

**Implements:** RFC-0001, DESIGN-0001, ADR-0001

## Scope

### In Scope

- `claudelint` CLI with `lint`, `rules`, `init`, `version` subcommands.
- HCL config loader (schema v1) per ADR-0001.
- Discovery of `CLAUDE.md`, skills, slash commands, subagents, hooks
  (both dedicated JSON and inline in `settings.json`), plugin manifests.
- ~15 built-in rules (see DESIGN-0001 table).
- Text, JSON, and GitHub Actions output formats.
- In-source `// claudelint:ignore` suppressions and config-level
  disables/severity overrides.
- `make ci` wiring and dogfooding this repo.

### Out of Scope

- SARIF output (Phase 2).
- Pre-commit hook (Phase 2).
- `claudelint fix` auto-fix (later).
- `claudelint convert` format conversion (Phase 3; gated on INV-0001).
- Third-party rule plugins.

## Implementation Phases

Each phase builds on the previous one. A phase is complete when all its
tasks are checked off and its success criteria are met.

---

### Phase 1.1: Foundation

Establish the skeleton: CLI, logging, module structure, and a no-op
`lint` command that walks the repo and prints discovered file counts.

#### Tasks

- [ ] Add `cmd/claudelint/main.go` with cobra root + subcommands.
- [ ] Add `internal/discovery/` with a walker that honors `.gitignore`
      and classifies files into `ArtifactKind`.
- [ ] Add `internal/diag/` with `Diagnostic`, `Severity`, `Range`.
- [ ] Add `internal/reporter/text.go` producing human-readable output.
- [ ] Wire `claudelint lint` to discover + report `(0 diagnostics, N files)`.
- [ ] Add `claudelint version` printing build info.
- [ ] Write unit tests for discovery classification.

#### Success Criteria

- `go build ./...` succeeds.
- `claudelint lint .` prints discovered file counts on this repo.
- Discovery tests cover every `ArtifactKind` with fixture inputs.

---

### Phase 1.2: Parsers and artifact model

Turn discovered paths into typed artifacts.

#### Tasks

- [ ] Add `internal/artifact/` with interfaces and structs per
      DESIGN-0001.
- [ ] Markdown + YAML frontmatter parser preserving byte offsets.
- [ ] JSON parser for hooks and plugin manifests with position info.
- [ ] Skill parser that indexes companion files (`references/`,
      `scripts/`).
- [ ] `schema/parse` rule that reports parse failures.
- [ ] Table-driven parser tests with `testdata/ok` and `testdata/bad`.

#### Success Criteria

- Every artifact kind round-trips fixture inputs with accurate
  line/column positions.
- Malformed inputs produce a single `schema/parse` diagnostic and do
  not crash the linter.

---

### Phase 1.3: Config loader (HCL)

#### Tasks

- [ ] Add `internal/config/` using `hashicorp/hcl/v2`.
- [ ] Implement schema v1 decoding: `claudelint`, `rule`, `rules`,
      `ignore`, `output` blocks.
- [ ] Version check: reject unknown top-level `version` values with a
      clear upgrade message.
- [ ] `claudelint init` scaffolder writing a commented default config.
- [ ] Locate config by walking up from cwd; support `--config=PATH`.
- [ ] Tests for every error path with line/column assertions.

#### Success Criteria

- Invalid config files produce HCL-style diagnostics pointing at the
  exact token.
- `claudelint init` in an empty directory produces a config that
  `claudelint lint` accepts without diagnostics.

---

### Phase 1.4: Rule engine and built-in rules

#### Tasks

- [ ] Add `internal/engine/` with rule registry and concurrent runner.
- [ ] Implement the MVP rule list from DESIGN-0001.
- [ ] Each rule lives in `internal/rules/<kind>/<id>.go` with a
      table-driven test.
- [ ] In-source `claudelint:ignore` suppression parser.
- [ ] Config-level disables, severity overrides, and per-path globs.
- [ ] `claudelint rules` prints the registry; `claudelint rules <id>`
      prints rationale and default options.

#### Success Criteria

- Running `claudelint lint .` on this repo produces zero errors.
- Each rule has at least one `ok` and one `bad` fixture test.
- Suppressions are honored and unknown IDs warn via `meta/unknown-rule`.

---

### Phase 1.5: Output formats and exit codes

#### Tasks

- [ ] `--format=text|json|github` (SARIF deferred to Phase 2).
- [ ] Exit non-zero on any `error`; `--max-warnings=N` flag.
- [ ] `--quiet`, `--no-color`, `NO_COLOR` env honored.

#### Success Criteria

- JSON output is stable and documented; a golden-file test guards
  against accidental changes.
- GitHub Actions annotations render correctly in a smoke-test workflow.

---

### Phase 1.6: Polish, docs, and release prep

#### Tasks

- [ ] Audit error messages for consistency (imperative, actionable).
- [ ] Ensure `make ci` passes with zero warnings.
- [ ] Test coverage ≥ 80% in `internal/...`.
- [ ] Benchmark against `testdata/bench/` synthetic repo; fail CI on
      regression > 20%.
- [ ] Update `README.md` with install, quickstart, and rule index.
- [ ] Dogfood on at least two external plugin repos; file issues for
      findings.
- [ ] Cut a `v0.1.0` release via `goreleaser`.

#### Success Criteria

- `make ci` passes.
- `claudelint` ships as a single static binary via GitHub Releases and
  `go install`.
- README documents every MVP rule with one example and one fix.

---

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `cmd/claudelint/main.go` | Create | Cobra entrypoint + subcommand wiring |
| `internal/config/*.go` | Create | HCL loader, schema v1 |
| `internal/discovery/*.go` | Create | File walker + artifact classification |
| `internal/artifact/*.go` | Create | Typed artifact model + parsers |
| `internal/engine/*.go` | Create | Rule registry + runner |
| `internal/rules/**/*.go` | Create | Built-in rules |
| `internal/diag/*.go` | Create | Diagnostic and severity types |
| `internal/reporter/*.go` | Create | text/json/github formatters |
| `go.mod`, `go.sum` | Modify | Add `hashicorp/hcl/v2`, `spf13/cobra`, `goccy/go-yaml` (or `yaml.v3`) |
| `README.md` | Modify | Replace TODO with install/usage |
| `.goreleaser.yml` | Modify | Confirm main path and binary name |
| `Makefile` | Modify | Add `lint-self` target that runs claudelint on this repo |
| `.claudelint.hcl` | Create | Dogfood config |

## Testing Plan

- [ ] Unit tests for every exported function in `internal/...`.
- [ ] Parser tests with golden byte-offset assertions.
- [ ] Rule tests with `testdata/ok/` and `testdata/bad/` per rule.
- [ ] Integration tests in `cmd/claudelint` that invoke the binary
      against fixture repos and diff stdout/stderr.
- [ ] Benchmarks in `testdata/bench/` with `go test -bench=.`.
- [ ] Coverage gate in CI: fail if `internal/...` drops below 80%.

## Dependencies

- `github.com/hashicorp/hcl/v2`
- `github.com/spf13/cobra`
- `gopkg.in/yaml.v3` (or `github.com/goccy/go-yaml` for better error
  positions)
- Existing repo tooling: `mise`, `goreleaser`, `golangci-lint`,
  `markdownlint`, `yamllint`.

## References

- RFC-0001 — Claudelint
- ADR-0001 — Use HCL as config format
- DESIGN-0001 — Architecture and rule engine
- INV-0001 — Format conversion investigation (Phase 3 gate)
