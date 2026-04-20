---
id: DESIGN-0001
title: "Claudelint linter architecture and rule engine"
status: Draft
author: Donald Gifford
created: 2026-04-18
---
<!-- markdownlint-disable-file MD025 MD041 -->

# DESIGN 0001: Claudelint linter architecture and rule engine

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-04-18

<!--toc:start-->
- [Overview](#overview)
- [Goals and Non-Goals](#goals-and-non-goals)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Background](#background)
- [Detailed Design](#detailed-design)
  - [Package layout](#package-layout)
  - [Core interfaces](#core-interfaces)
  - [Execution flow](#execution-flow)
  - [Parsers](#parsers)
  - [Built-in rules (MVP shortlist)](#built-in-rules-mvp-shortlist)
  - [Ruleset versioning](#ruleset-versioning)
  - [Config schema v1](#config-schema-v1)
  - [Suppression model](#suppression-model)
- [API / Interface Changes](#api--interface-changes)
  - [CLI](#cli)
  - [Library](#library)
- [Data Model](#data-model)
- [Testing Strategy](#testing-strategy)
- [Migration / Rollout Plan](#migration--rollout-plan)
- [Open Questions](#open-questions)
- [References](#references)
<!--toc:end-->

## Overview

Detailed design for the claudelint core. The architecture is three
layers with a single direction of dependency:

```
     Parsers  →  Engine  →  Rules
   (bytes →    (orchestrates,  (tiny, pure
    typed       schedules,      Check funcs
    artifact)   reports)        behind one
                                interface)
```

- **Parsers** turn bytes on disk into typed `Artifact` values.
- **The engine** is where the complexity lives: config loading, artifact
  discovery, rule scheduling, concurrency, diagnostic collection,
  suppression handling, and reporting.
- **Rules** are small. Each rule is a focused function behind a common
  `Rule` interface — roughly "given this typed artifact, return any
  diagnostics." Rules do no I/O, hold no state, and know nothing about
  how the engine dispatches them.

This is the same shape as `golangci-lint` / `staticcheck` /
`go/analysis`: the runner is a substantial piece of code; an individual
linter is often a few dozen lines. Adding a rule means implementing one
interface and registering it — no engine changes required.

Rules are **built-in to the binary and versioned with it**. v1 does
not support third-party plugin rules; that is an explicit non-goal
(see Open Questions for how we'd reconsider later).

This document covers the MVP scope described in RFC-0001 and IMPL-0001.
The format-conversion subcommand is described separately and gated on
INV-0001.

## Goals and Non-Goals

### Goals

- Single static Go binary runnable from CLI, pre-commit, and CI.
- Deterministic, line-accurate diagnostics for every rule.
- Rule authoring is simple: implement one interface, register in an
  `init()`. No engine changes required to add a rule.
- Rules ship built-in to the binary. A claudelint release pins exactly
  the ruleset it shipped with.
- Config and rules are versioned. Breaking changes require a new schema
  version, not silent behavior shifts.
- Fast enough on large repos that developers leave it enabled (target:
  < 200ms for a typical plugin repo; < 2s for a monorepo with 10k files).

### Non-Goals

- General-purpose Markdown linting (delegate to `markdownlint`).
- Runtime behavioral analysis of plugins (we are static-only).
- Third-party / plugin rules in v1. All rules live in-tree and are
  released with the binary.
- Lossless cross-ecosystem conversion. See INV-0001 — conversion is
  best-effort with explicit diagnostics for dropped fields.
- Auto-fix in v1. Diagnostics may include fix hints but `claudelint fix`
  is deferred to a later milestone.

## Background

Claude Code artifacts live in predictable places:

- `CLAUDE.md` at any depth.
- `.claude/settings.json` and `.claude/settings.local.json`.
- `.claude/commands/*.md` (slash commands).
- `.claude/agents/*.md` (subagent definitions).
- `.claude/skills/<name>/SKILL.md` (+ `references/`, `scripts/`, etc.).
- `.claude/hooks/*.json` or hooks declared inline in `settings.json`.
- Plugin manifests (`plugin.json` / `plugin.yaml`) at plugin roots.

Each artifact is hand-authored Markdown with YAML frontmatter, JSON, or
a combination. Existing generic linters (`markdownlint`, `yamllint`,
`jsonlint`) cover generic syntax; none understand Claude-specific
semantics (required frontmatter fields, hook event names, tool
allowlists, skill trigger rules).

## Detailed Design

### Package layout

```
cmd/claudelint/          # CLI entrypoint (cobra)
internal/config/         # HCL loader + schema v1
internal/discovery/      # filesystem walker + artifact classification
internal/artifact/       # typed artifact structs + parsers
internal/rules/          # Rule + Context interfaces, registry
                         # (Register/All/Get), RulesetVersion +
                         # RulesetFingerprint
internal/rules/<kind>/   # built-in rule implementations
                         # (one rule per file; e.g.
                         # internal/rules/skills/bodysize.go)
internal/rules/all/      # blank-imports every rule subpackage so
                         # each rule's init() runs
internal/engine/         # runner, worker pool, aggregation
internal/diag/           # Diagnostic, Severity, Range, Position, Fix
internal/reporter/       # text, json, github (sarif in phase 2)
internal/convert/        # (phase 3) format conversion
```

Dependency direction is strictly one-way: rule packages import
`internal/rules` for the `Rule` interface and `Register`, and
`internal/engine` imports `internal/rules` to enumerate and dispatch.
Rule packages never import the engine.

### Core interfaces

The contract between the engine and rules is small on purpose. A rule
implementor only has to understand `Artifact`, `Rule`, and
`Diagnostic`. The engine owns everything else.

```go
// ArtifactKind identifies what we're linting.
type ArtifactKind string

const (
    KindClaudeMD   ArtifactKind = "claude_md"
    KindSkill      ArtifactKind = "skill"
    KindCommand    ArtifactKind = "command"
    KindAgent      ArtifactKind = "agent"
    KindHook       ArtifactKind = "hook"
    KindPlugin     ArtifactKind = "plugin"
)

// Artifact is the parsed, typed view of a file on disk.
// Produced by a Parser, consumed by Rules. Rules see only Artifact,
// never raw bytes or paths except through this interface.
type Artifact interface {
    Kind() ArtifactKind
    Path() string
    Source() []byte     // raw bytes, for position reporting
}

// Rule is the unit of analysis. Rules are:
//   - pure: no I/O, no global state, no cross-file awareness (v1)
//   - small: typical rule ≤ 50 LOC plus a table-driven test
//   - focused: one rule checks one property
// Inspired by go/analysis.Analyzer and golangci-lint's linter contract.
type Rule interface {
    ID() string              // e.g., "skills/require-name"
    Category() string        // schema | content | security | style
    DefaultSeverity() Severity
    // DefaultOptions returns the rule's option keys and default values.
    // The engine fills in unspecified options from this map and uses
    // the Go types of the default values to validate user-supplied
    // options before Check is called. `claudelint rules <id>` prints
    // this map so users can see available knobs. Return nil if the
    // rule takes no options.
    DefaultOptions() map[string]any
    AppliesTo() []ArtifactKind
    Check(ctx Context, a Artifact) []Diagnostic
}

// Context is everything a rule is allowed to see beyond the artifact:
// resolved options for this rule, the rule's own ID, and a logger.
// Kept deliberately narrow so rules stay testable.
type Context interface {
    Option(key string) any    // rule-specific options from HCL config
    Logger() Logger
}

// Diagnostic is what the reporter prints.
type Diagnostic struct {
    RuleID   string
    Severity Severity
    Path     string
    Range    Range       // line/col start+end; zero value means file-level
    Message  string      // short, imperative
    Detail   string      // long-form; may include fix hint
    Fix      *Fix        // v1 placeholder, always nil. JSON: omitempty.
                         // Shape will be validated when `claudelint fix`
                         // ships; kept now for forward JSON compat.
}
```

A minimal rule, end to end:

```go
// internal/rules/skills/requirename.go
package skills

import "claudelint/internal/rules"

func init() { rules.Register(&requireName{}) }

type requireName struct{}

func (requireName) ID() string              { return "skills/require-name" }
func (requireName) Category() string        { return "schema" }
func (requireName) DefaultSeverity() Severity { return SeverityError }
func (requireName) AppliesTo() []ArtifactKind { return []ArtifactKind{KindSkill} }

func (requireName) Check(ctx Context, a Artifact) []Diagnostic {
    s := a.(*artifact.Skill)
    if s.Frontmatter.Name != "" {
        return nil
    }
    return []Diagnostic{{
        RuleID:   "skills/require-name",
        Severity: SeverityError,
        Path:     s.Path(),
        Range:    s.FrontmatterRange,
        Message:  `skill frontmatter is missing "name"`,
    }}
}
```

That's the whole surface area for a new rule: one file, one `init()`
registration, a table-driven test next to it. The engine picks it up
automatically.

### Execution flow

1. **Bootstrap.** Parse CLI flags (cobra). Locate config by walking up
   from cwd until `.claudelint.hcl` is found, or accept `--config=PATH`.
2. **Load config.** Parse HCL into an internal `Config` struct.
   Validate schema version. Apply rule overrides.
3. **Discover.** Walk the filesystem honoring full `git status`-style
   ignore semantics — root `.gitignore`, nested `.gitignore` files,
   `.git/info/exclude`, and the global `~/.config/git/ignore` — plus
   config `ignore.paths`. Classify each file into an `ArtifactKind`
   using the path/filename patterns in the Background section.
   Unrecognized files are skipped silently.
4. **Parse.** Each artifact type has a parser that returns a typed
   struct. Parse errors become `error`-severity diagnostics under the
   `schema/parse` rule ID and cause the artifact to be skipped for
   further rules. `schema/parse` is a *pseudo-rule*: registered in the
   registry so it appears in `claudelint rules` and can be targeted by
   suppressions, but its `Check` method is never invoked — the engine
   synthesizes the diagnostic directly from the `ParseError`.
5. **Run rules.** The engine dispatches one goroutine per artifact
   (bounded pool sized to `GOMAXPROCS`); within each goroutine, the
   applicable rules for that artifact's `Kind` run serially. Each rule
   sees one artifact at a time (no cross-file state in v1). Coarse
   granularity keeps scheduling overhead low on repos with many small
   artifacts; profile before changing.
6. **Collect diagnostics.** Aggregate, sort by path/line, dedupe
   identical diagnostics.
7. **Apply suppressions.** Drop diagnostics matched by
   `// claudelint:ignore=<rule-id>` comments on the same line or the
   preceding line, or by config-level `disable`.
8. **Report.** Hand off to the selected reporter.
9. **Exit.** Non-zero if any `error`-severity diagnostic remains;
   `--max-warnings=N` optionally promotes warning counts to failure.

### Parsers

| Kind | Parser |
|------|--------|
| `claude_md` | Markdown parser with frontmatter extraction. YAML frontmatter uses `github.com/goccy/go-yaml` for precise line/column positions on every key. Body split into directive blocks by heading. |
| `skill` | Frontmatter (`name`, `description`, optional `allowed-tools`, `model`) + Markdown body. Companion files indexed. |
| `command` | Frontmatter (`description`, `argument-hint`, `allowed-tools`) + body. |
| `agent` | Frontmatter (`name`, `description`, `tools`) + system prompt body. |
| `hook` | JSON — either dedicated file or the `hooks` stanza inside `settings.json`. |
| `plugin` | JSON or YAML manifest; fields per the Claude plugin spec (`name`, `version`, `commands`, `skills`, `agents`, …). |

All parsers preserve byte offsets so diagnostics can report exact
line/column ranges, including inside Markdown bodies.

### Built-in rules (MVP shortlist)

| Rule ID | Kind | Severity | What it checks |
|---------|------|----------|----------------|
| `schema/parse` | * | error | File parses at all |
| `schema/frontmatter-required` | skill, command, agent | error | `name` and `description` present |
| `skills/body-size` | skill | warning | `SKILL.md` body ≤ configurable word count |
| `skills/trigger-clarity` | skill | warning | `description` contains an imperative trigger phrase |
| `commands/allowed-tools-known` | command | error | Every tool in `allowed-tools` is a real tool name |
| `hooks/event-name-known` | hook | error | `event` is a valid hook event |
| `hooks/no-unsafe-shell` | hook | warning | Hook command does not pipe `curl ... \| sh` or similar |
| `hooks/timeout-present` | hook | warning | Long-running hooks declare a `timeout` |
| `claude_md/size` | claude_md | warning | File ≤ configurable line count |
| `claude_md/duplicate-directives` | claude_md | warning | No two rules contradict |
| `plugin/manifest-fields` | plugin | error | Required manifest fields present and well-typed |
| `plugin/semver` | plugin | warning | `version` is valid semver |
| `style/no-emoji` | * | info | (off by default) no emoji in output-influencing text |
| `security/secrets` | * | error | No high-entropy strings resembling API keys |

Each rule lives in `internal/rules/<kind>/<id>.go` with a table-driven
test in the same package. The rule registry is built via `init()`.

### Ruleset versioning

The ruleset is versioned independently of the binary, similar to how
Go module pseudo-versions pair semver with a content hash:

- `RulesetVersion` — a hand-bumped semver constant in
  `internal/rules` (e.g. `"v1.2.0"`). Human-readable; goes in release
  notes when rules are added, removed, or their defaults change.
- `RulesetFingerprint` — a hash computed at `init()` time from the
  sorted list of registered rule IDs plus each rule's default
  severity and default options. Auto-updates whenever the ruleset
  changes, even accidentally.

A CI guardrail test recomputes the fingerprint and compares it
against a checked-in `expected_fingerprint.txt`. Any drift fails CI
with `"ruleset changed; bump RulesetVersion and update
expected_fingerprint.txt"`. This catches the case where rules are
added without bumping semver, or semver is bumped without a real
change.

`claudelint version` prints both:

```
claudelint v0.3.1
ruleset    v1.2.0 (a1b2c3d4)
```

### Config schema v1

```hcl
claudelint {
  version = "1"
}

# Optional: change severity or disable rules wholesale.
rule "skills/body-size" {
  severity = "error"
  options  = { max_words = 750 }
}

rule "style/no-emoji" {
  enabled = false
}

# Per-artifact defaults.
rules "skills" {
  forbid_emojis       = true
  require_frontmatter = ["name", "description"]
}

ignore {
  paths = ["vendor/**", "testdata/**"]
}

output {
  format = "text"   # text | json | sarif | github
}
```

### Suppression model

- **In-source (Markdown artifacts only).** Recognized inside HTML
  comments in Markdown-backed artifacts (`CLAUDE.md`, skills,
  commands, agents):
  `<!-- claudelint:ignore=<id>[,<id>...] -->` applies to the same line
  or the next non-blank line; `<!-- claudelint:ignore-file=<id> -->`
  applies to the whole file. JSON-backed artifacts (hooks, plugin
  manifests) do **not** support in-source suppressions — JSON has no
  standard comment syntax, and config-level path globs are strictly
  more expressive for those files.
- **Config (any artifact).** `rule "<id>" { enabled = false }` disables
  globally; `{ severity = "..." }` overrides severity;
  `{ paths = ["glob", ...] }` disables for a path glob.

Suppression IDs must exist; unknown IDs are themselves a warning
(`meta/unknown-rule`).

## API / Interface Changes

### CLI

```
claudelint                       # run against the cwd
claudelint run ./path            # run against a subtree
claudelint run --format=sarif    # machine output
claudelint rules                 # list built-in rules with defaults
claudelint rules <id>            # show rule details and rationale
claudelint init                  # scaffold .claudelint.hcl
claudelint convert ...           # (phase 3)
claudelint version               # binary + ruleset version
```

`run` is deliberately chosen over `lint` — the tool will eventually do
more than lint (e.g. `convert`), and `run` matches the mental model of
"run the engine against these files." Bare `claudelint` is shorthand
for `claudelint run`.

`claudelint version` prints both the binary version and the ruleset
version baked into it, so CI can pin against either.

### Library

The `internal/` layout is internal by design for v1. We may promote
`engine`, `artifact`, and `diag` to a public `pkg/` once the API is
stable (post-1.0).

## Data Model

No persistent storage. All state is in-memory per run.

## Testing Strategy

- **Unit tests per rule.** Table-driven, fixture-based. Each rule has
  `testdata/ok/` and `testdata/bad/` inputs with golden diagnostic JSON.
- **Parser tests.** Round-trip byte offsets so diagnostics carry
  correct positions.
- **Config tests.** Every error path in the HCL loader has a test with a
  line/column assertion.
- **End-to-end.** `go test ./cmd/claudelint` runs the binary against a
  fixture repo and asserts against captured JSON output (SARIF once
  Phase 2 lands).
- **Benchmarks.** A synthetic 10k-file repo checked in under
  `testdata/bench/`; performance regressions fail CI.
- **Dogfooding.** Run claudelint against its own repo in CI.

## Migration / Rollout Plan

- v0.1: discovery + `CLAUDE.md`, skill, and command rules.
- v0.2: agents, hooks, plugin manifests.
- v0.3: SARIF + suppressions + pre-commit hook.
- v0.x → v1.0: stabilize config schema; freeze rule IDs.
- v1.x: convert subcommand (gated on INV-0001).

Breaking changes to the HCL schema bump the top-level `version`; the
loader refuses unknown versions with a clear upgrade message.

## Open Questions

- **Cross-artifact rules (Phase 2).** v1 rules are strictly
  per-artifact. A concrete case for cross-artifact analysis —
  `hooks/duplicate-declaration` when the same event is declared in
  both `settings.json` and `settings.local.json` — is deferred to
  Phase 2. Likely shape when we add it: a sibling `CorpusRule`
  interface with `Check(ctx, []Artifact) []Diagnostic`, registered the
  same way, run serially after per-artifact rules. We want code
  experience with the per-artifact engine before committing to the
  shape.
- **Third-party rules.** Deliberately out of scope for v1 (rules are
  built-in and versioned with the binary). If demand appears, options
  are RPC/subprocess rules or WASM modules — Go's `plugin.Open` is too
  fragile for a distributed CLI. A plugin API would require
  stabilizing the `Rule` / `Artifact` / `Diagnostic` surface first.
- **`.gitignore` library choice.** Full `git status` semantics are
  required, but the specific library is an implementation detail to
  decide in Phase 1.1. Candidates:
  `github.com/sabhiram/go-gitignore` (widely used, small) or
  `github.com/go-git/go-git/v5`'s matcher (heavier, more correct).
  Pick during implementation based on correctness tests against
  `git status` output on fixture repos.

## References

- RFC-0001 — Claudelint
- ADR-0001 — Use HCL as config format
- IMPL-0001 — Phase 1 plan
- INV-0001 — Format conversion investigation
- SARIF 2.1.0: https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html
