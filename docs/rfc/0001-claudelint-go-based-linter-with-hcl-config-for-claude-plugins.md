---
id: RFC-0001
title: "Claudelint: a Go-based linter with HCL config for Claude plugins, skills, hooks, and CLAUDE.md"
status: Draft
author: Donald Gifford
created: 2026-04-18
---
<!-- markdownlint-disable-file MD025 MD041 -->

# RFC 0001: Claudelint — a Go-based linter with HCL config for Claude plugins, skills, hooks, and CLAUDE.md

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-04-18

<!--toc:start-->
- [Summary](#summary)
- [Problem Statement](#problem-statement)
- [Proposed Solution](#proposed-solution)
- [Design](#design)
- [Alternatives Considered](#alternatives-considered)
- [Implementation Phases](#implementation-phases)
  - [Phase 1 — Core linter (MVP)](#phase-1--core-linter-mvp)
  - [Phase 2 — CI & ecosystem integrations](#phase-2--ci--ecosystem-integrations)
  - [Phase 3 — Stretch: format conversion](#phase-3--stretch-format-conversion)
- [Risks and Mitigations](#risks-and-mitigations)
- [Success Criteria](#success-criteria)
- [References](#references)
<!--toc:end-->

## Summary

Claudelint is a Go-based command-line linter that statically analyzes Claude
Code artifacts — plugins, skills, hooks, slash commands, subagents, and
`CLAUDE.md` memory files — against a configurable rule set expressed in HCL.
It ships as a single static binary with machine-readable output suitable for
CI, pre-commit, and editor integrations. A stretch goal is bidirectional
format conversion between Claude and adjacent ecosystems (OpenAI Apps /
Custom GPTs, OpenCode skills and plugins) where schemas overlap.

## Problem Statement

The Claude Code ecosystem is growing quickly — plugins, skills, hooks,
subagents, and `CLAUDE.md` files are now the primary way teams codify agent
behavior. There is no shared tooling to validate these artifacts:

- **No schema validation.** Skill frontmatter, plugin manifests, and hook
  definitions are hand-authored Markdown or JSON with no machine check.
  Typos in trigger fields, missing `name`/`description`, or malformed
  `allowed-tools` are discovered at runtime — or not at all.
- **No behavioral lint.** Common authoring mistakes (oversized `SKILL.md`,
  missing trigger guidance, leaking secrets into hooks, unsafe `Bash`
  allowlists, ambiguous `CLAUDE.md` directives) have no automated check.
- **No portability story.** Teams that maintain skill/plugin libraries
  across Claude, OpenAI, and OpenCode must re-author each artifact by hand.
- **No idiomatic CI story.** Without a linter, `CLAUDE.md` files and skill
  directories drift over time and lose the properties that made them useful.

Impact: slower authoring, silent bugs in production agent harnesses, and
duplicated effort across ecosystems.

## Proposed Solution

Build `claudelint`, a single-binary Go CLI with three responsibilities:

1. **Discover** Claude artifacts in a repository — `CLAUDE.md` files,
   `.claude/` directories, `skills/*/SKILL.md`, `commands/*.md`,
   `hooks/*.json`, `plugins/*/plugin.json`, agent frontmatter.
2. **Lint** them against a configurable rule set. Rules check schema
   conformance, content structure, security posture, and style. Rules are
   versioned and can be enabled, disabled, or tuned per-repo.
3. **Convert** (stretch) between formats — Claude ↔ OpenAI Apps / Custom
   GPTs, Claude ↔ OpenCode — where the underlying concepts map.

Configuration lives in `.claudelint.hcl` at the repo root. HCL (HashiCorp
Configuration Language) is chosen for its ergonomics around typed blocks,
comments, references, and partial overrides — see ADR-0001.

Example:

```hcl
claudelint {
  version = "1"
}

rules "skills" {
  max_body_words     = 500
  require_frontmatter = ["name", "description"]
  forbid_emojis      = true
}

rules "hooks" {
  forbid_network_in = ["PreToolUse", "PostToolUse"]
  require_timeout   = true
}

rules "claude_md" {
  max_lines = 200
  forbid_duplicate_rules = true
}

ignore {
  paths = ["vendor/**", "testdata/**"]
}
```

The tool prints human-readable diagnostics by default and supports
`--format=json|sarif|github` for CI integration.

## Design

High-level architecture (details in DESIGN-0001):

```
+-------------------+    +------------------+    +-------------------+
|   Discovery       | -> |  Parsers         | -> |   Rule Engine     |
|   (fs walker,     |    |  (md+frontmatter,|    |   (built-in and   |
|    gitignore)     |    |   json, hcl)     |    |    pluggable)     |
+-------------------+    +------------------+    +-------------------+
                                                           |
                                                           v
                                                 +-------------------+
                                                 |   Reporter        |
                                                 |   (text/json/     |
                                                 |    sarif/github)  |
                                                 +-------------------+
```

Key design points:

- **Config:** `.claudelint.hcl` parsed with `hashicorp/hcl/v2`. Schema is
  versioned (`claudelint.version`). Invalid config is a fatal error.
- **Artifact model:** each discovered artifact becomes a typed `Artifact`
  value (`SkillArtifact`, `HookArtifact`, `ClaudeMDArtifact`, …) with
  source location. Rules consume artifacts, not raw bytes.
- **Rule engine:** built-in rules implement a `Rule` interface
  (`ID()`, `Category()`, `Check(ctx, artifact) []Diagnostic`). Rules are
  registered in a registry keyed by artifact kind.
- **Diagnostics:** every diagnostic carries rule ID, severity, file,
  line/column where possible, short message, and long-form rationale with
  a fix hint. Severity (`error`, `warning`, `info`) is rule-default but
  overridable in config.
- **Output:** SARIF for code scanning platforms, GitHub Actions annotations,
  JSON for tooling, and plain text for humans.
- **Converter (stretch):** a separate `claudelint convert` subcommand with
  a lossy/best-effort translator between Claude and OpenAI/OpenCode
  artifact schemas (see INV-0001).

## Alternatives Considered

- **YAML/TOML for config.** YAML is ubiquitous but weakly typed, whitespace
  sensitive, and poor at expressing conditional blocks; TOML is fine for
  flat config but awkward for nested rule blocks. HCL gives us typed
  blocks, labels, and comments with good Go library support. See ADR-0001.
- **Reuse `golangci-lint`'s plugin model.** Its Go-AST focus does not
  match our artifact types (Markdown, JSON, HCL). We borrow the UX (rule
  IDs, severity, `//nolint`-style inline suppressions) but not the engine.
- **Node/TypeScript implementation.** The Claude ecosystem already has
  Node tooling; however, a Go single-binary is easier to distribute via
  `go install`, Homebrew, and GitHub releases, and avoids requiring Node
  in CI for teams that don't otherwise need it.
- **No linter, publish only a schema.** JSON Schema alone cannot express
  the behavioral rules (oversized skills, secret leakage in hooks,
  duplicated CLAUDE.md directives) that deliver most of the value.

## Implementation Phases

### Phase 1 — Core linter (MVP)

- Discovery + parsers for `CLAUDE.md`, skills (`SKILL.md` + frontmatter),
  hooks (`.claude/settings*.json`), slash commands (`commands/*.md`),
  subagents (`agents/*.md`).
- HCL config loader with schema v1.
- ~15 built-in rules covering schema conformance, basic content/style, and
  obvious security issues.
- Text, JSON, and GitHub Actions output.

### Phase 2 — CI & ecosystem integrations

- SARIF output + GitHub code scanning integration.
- Pre-commit hook and `golangci-lint`-style `//claudelint:ignore`
  suppressions.
- `claudelint init` scaffolder for new repos.
- Published ruleset versioning + deprecation policy.

### Phase 3 — Stretch: format conversion

- `claudelint convert` subcommand.
- Claude ↔ OpenAI (Apps/Custom GPTs) skill/plugin translation, best-effort.
- Claude ↔ OpenCode skill/plugin translation.
- Round-trip validation and diagnostics for unsupported features.

## Risks and Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Claude artifact formats evolve and break the linter | High | High | Version the ruleset (`claudelint.version`); treat unknown fields as warnings, not errors; fast release cadence pinned to Claude changelog. |
| HCL unfamiliar to users who expect YAML | Medium | Medium | Ship a `claudelint init` scaffolder with a well-commented default config; provide a YAML → HCL translator. |
| Format conversion produces lossy output users mistake for lossless | High | Medium | `convert` always emits a diagnostics report of dropped/approximated fields; non-zero exit when loss exceeds a configurable threshold. |
| Rule bikeshedding slows adoption | Medium | Medium | Ship with conservative defaults; every rule off-by-default unless it catches a known incident pattern. |
| Overlap/confusion with existing Markdown/JSON linters | Low | Medium | Scope is strictly Claude-artifact semantics; delegate generic Markdown lint to `markdownlint`. |

## Success Criteria

- A contributor to this repo can run `claudelint` locally in under 2s on a
  typical plugin repo and see actionable diagnostics.
- CI on at least three real plugin repositories surfaces at least one
  previously unknown issue.
- Config is stable: no breaking schema change in `v1.x`.
- Stretch: `claudelint convert` can round-trip a non-trivial OpenCode
  skill to Claude and back with a documented loss report.

## References

- ADR-0001 — Use HCL as the linter configuration format
- DESIGN-0001 — Claudelint linter architecture and rule engine
- IMPL-0001 — Phase 1: core linter for CLAUDE.md, skills, plugins, hooks
- INV-0001 — OpenAI and OpenCode skill/plugin format compatibility with Claude
- HashiCorp HCL: https://github.com/hashicorp/hcl
- SARIF 2.1.0 spec: https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html
