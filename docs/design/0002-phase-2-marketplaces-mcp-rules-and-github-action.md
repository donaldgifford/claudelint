---
id: DESIGN-0002
title: "Phase 2 — marketplaces, MCP rules, and GitHub Action"
status: Draft
author: Donald Gifford
created: 2026-04-23
---
<!-- markdownlint-disable-file MD025 MD041 -->

# DESIGN 0002: Phase 2 — marketplaces, MCP rules, and GitHub Action

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-04-23

<!--toc:start-->
- [Overview](#overview)
- [Goals and Non-Goals](#goals-and-non-goals)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Background](#background)
- [Detailed Design](#detailed-design)
  - [1. Marketplace.json handling](#1-marketplacejson-handling)
    - [New artifact kind](#new-artifact-kind)
    - [Discovery + classification](#discovery--classification)
    - [Rule package rules/marketplace/](#rule-package-rulesmarketplace)
  - [2. MCP rule package](#2-mcp-rule-package)
    - [What is the artifact?](#what-is-the-artifact)
    - [New artifact kind](#new-artifact-kind-1)
    - [Rule package rules/mcp/](#rule-package-rulesmcp)
  - [3. GitHub Action](#3-github-action)
    - [Distribution shape](#distribution-shape)
    - [Inputs / outputs](#inputs--outputs)
    - [Integration with GitHub code scanning](#integration-with-github-code-scanning)
- [API / Interface Changes](#api--interface-changes)
  - [CLI](#cli)
  - [Artifact kinds](#artifact-kinds)
  - [Config schema](#config-schema)
- [Data Model](#data-model)
- [Testing Strategy](#testing-strategy)
- [Migration / Rollout Plan](#migration--rollout-plan)
- [Related / Deferred from RFC-0001 Phase 2](#related--deferred-from-rfc-0001-phase-2)
- [Open Questions](#open-questions)
- [References](#references)
<!--toc:end-->

## Overview

Phase 1 shipped a working linter for single-plugin and `.claude/`-rooted
layouts. Phase 2 closes three gaps that surfaced when comparing against
`stbenjam/claudelint`: no awareness of **plugin marketplaces**, no rules
for **MCP servers**, and no turnkey **CI integration**. This doc covers
all three under one design because they share a common ingestion and
reporting story (new artifact kinds + SARIF-ready output → GitHub
Actions).

## Goals and Non-Goals

### Goals

- Recognize and lint `.claude-plugin/marketplace.json` manifests, and
  correctly classify the three marketplace repo shapes documented in the
  Claude Code plugin-marketplaces spec: **single plugin**, **traditional
  marketplace** (`plugins/<name>/` subdirectories), and **flat
  marketplace** (`source: "./"` at the marketplace root).
- Add a `rules/mcp/` rule package that lints `.mcp.json` server
  manifests and plugin-embedded MCP server definitions.
- Ship a **reusable GitHub Action** (`donaldgifford/claudelint-action@v1`)
  so consumers drop a 5-line workflow step instead of scripting
  `curl | sh` themselves.
- Emit **SARIF 2.1.0** output so the Action can upload results to the
  GitHub code-scanning tab.
- Preserve the Phase 1 architecture: no engine changes, no rule-package
  dependencies on the engine, built-in rules only.

### Non-Goals

- **Third-party / plugin-loaded rules.** Still intentionally out of
  scope (see DESIGN-0001 "Non-Goals"). If we need extensibility, revisit
  in Phase 3+.
- **Format conversion (`claudelint convert`).** Phase 3, gated on
  INV-0001.
- **Non-GitHub CI integrations.** GitLab/Bitbucket Actions-equivalents
  can follow once the pattern is proven; `github` reporter already works
  from any CI.
- **Publishing to GitHub Marketplace.** We'll ship the Action as a
  public repo with tags; Marketplace listing is a follow-up polish item.

## Background

Three drivers:

1. **Comparison with `stbenjam/claudelint`** (see 2026-04-23 session
   notes): that project's headline differentiators are marketplace
   detection and MCP rules. Neither is hard to add given our
   architecture, and both cover real-world artifacts our Phase 1 silently
   ignores.
2. **RFC-0001 Phase 2 leftovers.** SARIF output + GitHub code-scanning
   integration were in the original Phase 2 list but pre-dated the
   decision to ship a first-class Action. Folding them together is
   cheaper than shipping them separately.
3. **Dogfooding findings from INV-0003.** The Phase 1.8 dogfood run
   already revealed that our discovery assumed a `.claude/` parent;
   marketplace repos break that assumption further, and we want to fix
   both gaps before encouraging broader adoption.

## Detailed Design

### 1. Marketplace.json handling

#### New artifact kind

Add `KindMarketplace` to `internal/artifact/types.go`. The manifest
lives at `.claude-plugin/marketplace.json` and carries a `plugins[]`
array — each entry has a `source` field that drives layout:

- `source: "./"` — the marketplace *is* the plugin (flat layout); all
  plugin components sit at the repo root alongside `.claude-plugin/`.
- `source: "./plugins/foo"` — traditional layout; plugin lives in a
  subdirectory.
- `source: "<git-url>"` — external; out of scope for linting (we only
  lint local files).

A new artifact type `Marketplace` carries:

```go
type Marketplace struct {
    base
    Raw     []byte            // full file bytes for byte-offset ranges
    Plugins []MarketplacePlugin
}

type MarketplacePlugin struct {
    Name        string
    Source      string       // verbatim source field
    SourceRange diag.Range   // for pointing diagnostics at the plugins[].source string
    Resolved    string       // repo-relative path, "" if external/unresolvable
}
```

#### Discovery + classification

Extend `internal/discovery/classify.go` with a **pre-pass** that runs
*before* the per-file classifier:

1. If `<root>/.claude-plugin/marketplace.json` exists, parse it and
   build a map `resolved-path -> MarketplacePlugin`.
2. For each `plugins[].source` that resolves to a local path, record
   it as a **plugin root** override so the existing classifier treats
   `<root>/<source>` as a plugin-distribution layout (reusing
   `classifyPluginLayout` from Phase 1).
3. Emit the marketplace manifest itself as a `KindMarketplace` artifact.

This means the classifier never needs to "guess" marketplace shapes —
the manifest is the source of truth, and we just drive discovery from
it. Repos without a `marketplace.json` continue to flow through the
Phase 1 path unchanged.

#### Rule package `rules/marketplace/`

Initial rule set (each ~50 LOC, one rule per file, per IMPL-0001
conventions):

| Rule ID | Severity | Catches |
|---|---|---|
| `marketplace/name` | error | `name` missing or empty |
| `marketplace/version-semver` | error | `version` missing or not a valid semver |
| `marketplace/plugins-nonempty` | warn | `plugins: []` (nothing to distribute) |
| `marketplace/plugin-source-valid` | error | `plugins[].source` missing, empty, or references a path that does not exist in the repo |
| `marketplace/plugin-name-unique` | error | duplicate `plugins[].name` values |
| `marketplace/plugin-name-matches-dir` | warn | `plugins[].name` does not match the directory basename of `source` (UX guardrail) |
| `marketplace/author-required` | info | `author` missing (not fatal — many public marketplaces omit it) |

### 2. MCP rule package

#### What is the artifact?

MCP servers are declared two ways in the Claude Code ecosystem:

1. **Project-scoped `.mcp.json`** at the repo root — a `servers{}` map
   keyed by server name, each with `command`, `args`, `env`, and
   optional `disabled`.
2. **Plugin-embedded MCP servers** — a plugin's `plugin.json` may carry
   an `mcp.servers{}` field with the same shape.

Both are JSON, both use the same schema; the only difference is the
parent document. We handle both with one parser and one rule set.

#### New artifact kind

Add `KindMCPServer` to `internal/artifact/types.go`. The artifact
represents a single *server entry* (not the containing file), so we
can attach diagnostics to individual servers with precise byte ranges
— the same approach we use for Hook entries today.

```go
type MCPServer struct {
    base
    Name     string
    Command  string
    Args     []string
    Env      map[string]string
    Disabled bool
    // Embedded indicates whether this came from a plugin's plugin.json
    // (true) or a standalone .mcp.json (false). Some rules only apply
    // to one context.
    Embedded bool
}
```

Parsing reuses `buger/jsonparser` (same as hooks/plugin manifests in
Phase 1) to preserve byte offsets for `Range`.

#### Rule package `rules/mcp/`

Initial rule set:

| Rule ID | Severity | Catches |
|---|---|---|
| `mcp/command-required` | error | `command` missing or empty |
| `mcp/command-exists-on-path` | warn | `command` is a bare name (not absolute, no `/`) AND not a common shell builtin — catches typos like `"uvv"` instead of `"uvx"` |
| `mcp/no-secrets-in-env` | error | `env{}` value matches the existing `security/secrets` regexes (API keys, tokens). Reuses `rules/security/secrets.go` matcher |
| `mcp/no-unsafe-shell` | error | `command: "bash"` or `"sh"` with `-c` in args — same class of bug as `hooks/nounsafeshell` |
| `mcp/disabled-commented` | info | `disabled: true` without an adjacent comment (JSON has no comments, so check for a `description` field or a `// ` in `name`) — purely advisory |
| `mcp/server-name-required` | error | map key is empty string |

Share the secrets matcher with `rules/security/` rather than duplicating
it — the rule file just imports the shared regex bundle.

### 3. GitHub Action

#### Distribution shape

Separate public repo: `donaldgifford/claudelint-action`. Composite
action (not Docker) so startup is fast and we avoid the pull penalty on
every job. Structure:

```
claudelint-action/
├── action.yml
├── README.md
├── LICENSE
└── .github/workflows/test.yml
```

`action.yml` pulls the claudelint binary from the corresponding
`claudelint` release using the `version` input (defaults to `latest`),
runs it, and optionally uploads SARIF.

Versioning:

- Tag `v1`, `v1.0.0`, `v1.0`, `v1` (floating major) per GitHub Actions
  convention.
- Pin to claudelint versions via input `version: v0.2.0` in the consumer
  workflow; the Action itself does not track claudelint's semver
  one-to-one.

#### Inputs / outputs

```yaml
inputs:
  version:
    description: 'claudelint version to install (e.g. v0.2.0, or "latest")'
    default: 'latest'
  path:
    description: 'Path(s) to lint, space-separated'
    default: '.'
  format:
    description: 'Output format: text | json | github | sarif'
    default: 'github'
  config:
    description: 'Path to .claudelint.hcl (defaults to discovery)'
    default: ''
  max-warnings:
    description: 'Fail if warnings exceed this count (-1 = disabled)'
    default: '-1'
  upload-sarif:
    description: 'If true, upload SARIF to GitHub code-scanning'
    default: 'false'
  sarif-category:
    description: 'Category name for the uploaded SARIF'
    default: 'claudelint'

outputs:
  diagnostics-count:
    description: 'Total diagnostic count (errors + warnings + info)'
  error-count:
    description: 'Error-severity diagnostic count'
  sarif-path:
    description: 'Path to the generated SARIF file (if format=sarif)'
```

A two-job example consumer workflow lives in the action's README:

```yaml
- uses: donaldgifford/claudelint-action@v1
  with:
    format: sarif
    upload-sarif: true
```

#### Integration with GitHub code scanning

`format: sarif` emits SARIF 2.1.0 written to
`${{ runner.temp }}/claudelint.sarif`. When `upload-sarif: true`, the
action invokes `github/codeql-action/upload-sarif@v4` internally —
the consumer does not need to add a second step. Permissions required
by the consumer workflow: `security-events: write`.

SARIF rule metadata is emitted from the same `internal/rules/version.go`
source of truth that produces `rules list --json` today, so the SARIF
"rules" table stays in lockstep with the CLI.

## API / Interface Changes

### CLI

New `--format=sarif` value on `claudelint run`. No other CLI surface
changes. `rules list --json` output is unchanged but extended with a
`help_uri` field (used by SARIF to link rule IDs to docs).

### Artifact kinds

- `KindMarketplace` (new)
- `KindMCPServer` (new)

### Config schema

One new block, optional:

```hcl
marketplace {
  # Override auto-detection; rarely needed.
  manifest = ".claude-plugin/marketplace.json"

  # If set, restrict linting to these plugin names (skip others).
  only = ["foo", "bar"]
}
```

Existing `ignore.paths` globbing continues to work for carving out
specific plugins inside a marketplace.

## Data Model

Two new artifact kinds, both emitted by extended parsers. The `Result`
type in `internal/engine/runner.go` gains no new fields — SARIF is a
reporter-level concern, not an engine-level one.

A new `internal/reporter/sarif.go` produces SARIF 2.1.0 documents.
Structure mirrors `internal/reporter/json.go`; reuses the same
`help_uri` and rule metadata sources.

## Testing Strategy

- **Unit tests per rule** (`rules/marketplace/*_test.go`,
  `rules/mcp/*_test.go`) following the Phase 1 table-driven pattern.
- **Parser fixtures** under `internal/artifact/testdata/{ok,bad}/` for
  the three marketplace shapes and both MCP shapes.
- **SARIF reporter test**: golden-file comparison against a known
  diagnostic set; validate with the upstream SARIF JSON Schema
  (`https://json.schemastore.org/sarif-2.1.0.json`).
- **Action E2E**: the `claudelint-action` repo ships its own workflow
  (`test.yml`) that runs the Action against a fixture directory
  checked into the repo and asserts on the job summary / outputs.
- **Fingerprint guardrail** from Phase 1 will flag the new rules on
  first build — expected; we bump `rules.Version` to `0.2.0` and
  regenerate the checked-in fingerprint.

## Migration / Rollout Plan

1. Land `rules/marketplace/` + `KindMarketplace` discovery on `main`
   behind no feature flag. Existing repos without `marketplace.json`
   are unaffected.
2. Land `rules/mcp/` + `KindMCPServer`. Add `.mcp.json` to the default
   discovery globs.
3. Add `internal/reporter/sarif.go` and the `--format=sarif` CLI flag.
4. Stand up `donaldgifford/claudelint-action` (separate repo) pointing
   at claudelint `v0.2.0`. Tag `v1.0.0` + move `v1` after the Action's
   own E2E passes against the claudelint fixture repo.
5. Dogfood on the Anthropic marketplace + two of Donald's own plugin
   repos before announcing.
6. Cut `claudelint v0.2.0` (minor bump) — new rules are additive;
   existing configs continue to work; users opt in by setting the
   default in their workflow or config.

## Related / Deferred from RFC-0001 Phase 2

RFC-0001 Phase 2 originally listed four items. Tracking their
disposition:

- **SARIF output + GitHub code-scanning integration** → **in scope**
  here (prerequisite for the Action).
- **In-source `//claudelint:ignore` suppressions** → **already shipped
  in Phase 1** (Markdown HTML-comment flavor).
- **`claudelint init` scaffolder** → **already shipped in Phase 1**.
- **Published ruleset versioning + deprecation policy** → **partially
  shipped in Phase 1** (semver + fingerprint). Deprecation policy
  (how we sunset a rule across minor/major releases) is **still
  deferred**; will be tackled in a small follow-up ADR, not in this
  design.
- **Pre-commit hook** → **deferred**. Low effort but adds a maintenance
  surface; wait for a real user ask before shipping.

## Open Questions

1. **Marketplace external sources** — do we lint `plugins[].source`
   entries that point at external git URLs? Current plan: no, but emit
   an `info` diagnostic that says "external source skipped" so users
   know we didn't silently ignore it. Worth validating with real
   marketplaces.
2. **MCP `command` resolution on different OSes** — `mcp/command-exists-on-path`
   is intentionally a *warning*, not an error, because CI runners
   may not have the same tools installed as user machines. Is that
   the right default?
3. **Action distribution — composite vs Docker** — composite is
   proposed for startup speed, but Docker gives hermetic builds. If
   the binary-download step turns out to be flaky (rate limits, GPG
   verification failures), we may need to fall back to Docker. Test
   during rollout step 4.
4. **SARIF `partialFingerprints`** — should we emit stable hashes per
   diagnostic so GitHub deduplicates findings across runs? Yes in
   principle, but computing a hash that is stable across trivial
   whitespace changes is non-trivial. Propose: ship without
   fingerprints in v0.2.0, add in v0.3.0 once we see real noise.

## References

- [RFC-0001](../rfc/0001-claudelint-go-based-linter-with-hcl-config-for-claude-plugins.md)
  — proposal (Phase 2 scope).
- [DESIGN-0001](0001-claudelint-linter-architecture-and-rule-engine.md)
  — Phase 1 architecture (parsers / engine / rules layering that this
  doc extends).
- [IMPL-0001](../impl/0001-phase-1-core-linter-for-claudemd-skills-plugins-and-hooks.md)
  — Phase 1 task breakdown (reference for ruleset-versioning + rule
  file conventions).
- [INV-0003](../investigation/0003-phase-18-dogfood-findings-on-external-claude-plugins.md)
  — motivates broader layout coverage.
- Claude Code plugin-marketplaces spec:
  `https://docs.claude.com/en/docs/claude-code/plugin-marketplaces`.
- MCP spec: `https://modelcontextprotocol.io/specification`.
- SARIF 2.1.0: `https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html`.
- `github/codeql-action/upload-sarif`:
  `https://github.com/github/codeql-action/tree/main/upload-sarif`.
- Prior art: `stbenjam/claudelint` (Python) — marketplace detection +
  MCP rules.
