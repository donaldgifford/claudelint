---
id: DESIGN-0005
title: "Agent rules and opt-in rule mechanism"
status: Approved
author: Donald Gifford
created: 2026-07-10
---
<!-- markdownlint-disable-file MD025 MD041 -->

# DESIGN 0005: Agent rules and opt-in rule mechanism

**Status:** Approved
**Author:** Donald Gifford
**Date:** 2026-07-10

<!--toc:start-->
- [Overview](#overview)
- [Goals and Non-Goals](#goals-and-non-goals)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Background](#background)
- [Detailed Design](#detailed-design)
  - [1. Extended Agent artifact model](#1-extended-agent-artifact-model)
  - [2. Plugin-distributed agent detection](#2-plugin-distributed-agent-detection)
  - [3. Shared model-value validator](#3-shared-model-value-validator)
  - [4. Phase 4 rules](#4-phase-4-rules)
  - [5. Engine-level opt-in mechanism](#5-engine-level-opt-in-mechanism)
  - [6. agents/model-policy (Phase 5)](#6-agentsmodel-policy-phase-5)
- [API / Interface Changes](#api--interface-changes)
- [Data Model](#data-model)
- [Testing Strategy](#testing-strategy)
- [Migration / Rollout Plan](#migration--rollout-plan)
- [Open Questions](#open-questions)
- [References](#references)
<!--toc:end-->

## Overview

Agents are the last artifact kind with zero rules: the docs define a
16-field frontmatter spec with at least three silent-failure behaviors
(typo'd `model` silently inherits, unknown `tools` entries silently
ignored, plugin agents silently dropping `mcpServers`/`hooks`/
`permissionMode`). This design specifies the extended `Agent` artifact
model, the five always-on agent rules (IMPL-0004 Phase 4), the
engine-level opt-in mechanism resolved as IMPL-0004 OQ4 option (a), and
the opt-in `agents/model-policy` rule (Phase 5).

Approval provenance: OQ4 was resolved by Donald on 2026-07-10 as option
(a) — engine-level, config-driven enable — in
[IMPL-0004 Resolved Decisions](../impl/0004-phase-4-ruleset-alignment-and-agent-rules.md);
this document details that mechanism.

## Goals and Non-Goals

### Goals

- Parse the documented subagent frontmatter fields needed by Phases 4–5.
- Specify five always-on agent rules precisely enough to implement
  without further design decisions.
- Define one house convention for opt-in rules, enforced by the engine,
  and migrate `mcp/server-allowlist` onto it.
- Share the model-value validator across agent, skill, and command
  `model` fields.

### Non-Goals

- Validating `mcpServers` / `hooks` *contents* inside agent frontmatter
  (presence is enough for Phase 4; deep validation can reuse the MCP and
  hook rule packages in a later phase).
- Cross-file analysis (e.g. does `skills:` reference an existing skill,
  duplicate agent names across scopes). Rules stay pure over one
  artifact.
- `initialPrompt` content linting.
- A general plugin system for third-party rules — opt-in rules are still
  built-in (DESIGN-0001 non-goal unchanged).

## Background

INV-0006 audited the ruleset against the current Claude Code docs and
found the agent gap: a 16-field documented spec (subagents reference,
fetched 2026-07-10) against a parser that reads three fields. The
proposed rules and their phases come from INV-0006's "Agents" table; the
open mechanism question (how does a rule ship disabled-by-default?) was
IMPL-0004 OQ4.

Prior art in this repo: `mcp/server-allowlist` implemented opt-in
*inside* the rule — default-enabled, `DefaultOptions` declaring a nil
`allowlist`, loud config-error diagnostic when unconfigured. CLAUDE.md
explicitly said adding an engine hook for one rule would be
over-engineering and to revisit when a second rule needed the pattern.
`agents/model-policy` is that second rule.

## Detailed Design

### 1. Extended Agent artifact model

The subagents reference documents 16 frontmatter fields. `ParseAgent`
extends to the following. Every key's range is available via
`Frontmatter.KeyRange(key)` — the frontmatter parser already records
byte-accurate ranges for all keys, so no new range plumbing is needed;
rules anchor diagnostics per the range-emission conventions.

| Field | Parsed as | Notes |
|---|---|---|
| `name` | `Name string` | already parsed |
| `description` | `Description string` | already parsed |
| `tools` | `Tools []string` | already parsed; via shared tool-list splitter |
| `disallowedTools` | `DisallowedTools []string` | via shared tool-list splitter |
| `model` | `Model string` | raw declared value; absent = inherit |
| `permissionMode` | `PermissionMode string` | ignored for plugin agents (rule fodder) |
| `maxTurns` | `MaxTurns int64` | 0 = absent or non-numeric; presence via KeyRange |
| `skills` | `Skills []string` | verbatim string list (skill names, not tool syntax) |
| `mcpServers` | `HasMCPServers bool` | presence only; value shape (name refs or inline configs) not modeled in Phase 4 |
| `hooks` | `HasHooks bool` | presence only |
| `memory` | `Memory string` | enum `user`/`project`/`local` |
| `background` | `Background bool` | absent = false ("Claude chooses"); declared-vs-absent via KeyRange if ever needed |
| `effort` | `Effort string` | enum `low`/`medium`/`high`/`xhigh`/`max` |
| `isolation` | `Isolation string` | enum: `worktree` only |
| `color` | `Color string` | enum of 8 values |
| `initialPrompt` | `InitialPrompt string` | parsed for completeness; no Phase 4/5 rule consumes it |

### 2. Plugin-distributed agent detection

`agents/plugin-ignored-fields` needs to know whether an agent file is
plugin-distributed. Detection is a **path heuristic applied at
discovery time**, following the `IndexSkillCompanions` precedent (the
parser stays pure over bytes; the discovery/engine wiring enriches the
artifact with filesystem context):

- After a successful `ParseAgent`, discovery walks up from the agent
  file's directory; if any ancestor (up to the scan root) contains
  `.claude-plugin/plugin.json`, it sets `Agent.PluginDistributed = true`.
- `.claude/agents/**` under a plain project root does not match; a
  plugin repo layout (`<root>/.claude-plugin/plugin.json` +
  `<root>/agents/*.md`) and marketplace checkouts
  (`plugins/<name>/.claude-plugin/plugin.json`) do.

The flag is data on the artifact, so the rule stays pure and unit tests
can set it directly.

### 3. Shared model-value validator

One validator in `internal/artifact` (alongside `KnownTools`), consumed
by `agents/model-valid` for agent, skill, and command `model` fields
and by `agents/model-policy` option validation:

- `KnownModelAliases` = `sonnet`, `opus`, `haiku`, `fable`.
- `IsValidModelRef(v string) bool` — true when `v` is an alias,
  `inherit`, or matches the full-ID shape `^claude-[a-z0-9-]+$`.
- Empty string is the caller's concern (absent key = inherit; rules
  skip it).

### 4. Phase 4 rules

New package `internal/rules/agents/`, blank-imported by
`internal/rules/all/`. All five are always-on. Diagnostics never use
file-level `(0,0)` ranges.

| Rule | Category / severity | AppliesTo | Fires when | Range | Message sketch |
|---|---|---|---|---|---|
| `agents/model-valid` | schema / warning | agent, skill, command | `model` declared, non-empty, and `!IsValidModelRef` | `KeyRange("model")` | `model "sonet" is not a known value; want sonnet, opus, haiku, fable, inherit, or a full model ID (claude-...)` |
| `agents/name-format` | schema / warning | agent | `name` non-empty and not `^[a-z]+(-[a-z]+)*$` | `KeyRange("name")` | `agent name "My_Agent" should be lowercase letters and hyphens` (empties are `schema/frontmatter-required`'s job) |
| `agents/tools-known` | schema / warning | agent | entry in `tools`/`disallowedTools` fails `IsKnownTool` and `IsToolPattern` | `KeyRange` of the offending key | `unknown tool "Wrte" in tools — Claude Code silently ignores unknown names` |
| `agents/plugin-ignored-fields` | content / warning | agent | `PluginDistributed` and any of `permissionMode` / `mcpServers` / `hooks` declared | `KeyRange` of each offending key (one diagnostic per key) | `"permissionMode" is ignored for plugin-distributed subagents — dead config` |
| `agents/field-enums` | schema / warning | agent | declared enum field outside its documented set | `KeyRange` of the offending key | `permissionMode "yolo" is not a documented value; want default, acceptEdits, auto, dontAsk, bypassPermissions, plan, or manual` |

Enum sets for `agents/field-enums` (from the subagents reference,
2026-07-10):

- `permissionMode`: `default`, `acceptEdits`, `auto`, `dontAsk`,
  `bypassPermissions`, `plan`, `manual` (documented alias for
  `default`, v2.1.200+).
- `effort`: `low`, `medium`, `high`, `xhigh`, `max`.
- `color`: `red`, `blue`, `green`, `yellow`, `purple`, `orange`,
  `pink`, `cyan`.
- `isolation`: `worktree`.
- `memory`: `user`, `project`, `local`.

No options on any Phase 4 rule. `agents/model-valid` runs on skills and
commands too (same pattern as `commands/allowed-tools-known` running on
skills — the ID keeps its home package's prefix; rules.md notes the
extra kinds).

### 5. Engine-level opt-in mechanism

Per OQ4 option (a). An opt-in rule is **fully skipped** unless the
user's `.claudelint.hcl` carries a `rule "<id>"` block for it.

Mechanics:

- New optional interface in `internal/rules`:

  ```go
  // OptIn marks a rule as disabled unless the user's config
  // explicitly carries a rule block for it. The engine checks for
  // this interface when filtering active rules.
  type OptIn interface {
      OptIn() bool
  }
  ```

- Engine rule filtering (`Runner.activeRules`): when a rule implements
  `OptIn` and `OptIn()` is true, the rule runs only if the config has a
  `rule "<id>"` block (any block — even empty). `enabled = false`
  inside the block still disables it. No block → skipped entirely, no
  diagnostics of any kind.
- Config exposes `HasRuleBlock(id string) bool` to support the check.
- `claudelint rules` marks these entries `(opt-in)`; `rules --json`
  gains an additive `"opt_in": true` field
  (`docs/rules-json-schema.md` updated accordingly).
- The fingerprint buffer gains `optin=<bool>` per rule so flipping a
  rule's opt-in status is visible drift.
- **`mcp/server-allowlist` migrates**: it implements `OptIn`. With no
  rule block it is skipped (previously: loud config-error per
  artifact). With a rule block but no `allowlist` option, the loud
  config-error diagnostic is preserved — an explicit enable without an
  allowlist is still a misconfiguration, not a silent no-op.
- CLAUDE.md's "opt-in rules are implemented inside the rule" note is
  superseded by this convention (updated in the same commit).

### 6. agents/model-policy (Phase 5)

Governance rule, `error` severity, opt-in via §5.

Options (exactly one required):

- `require = "inherit"` — every agent must inherit: compliant when
  `model` is absent or `inherit`; anything else fires.
- `allowlist = ["opus", "claude-sonnet-5", "inherit"]` — declared
  `model` must be in the list; an absent `model` key is evaluated as
  `inherit` (the documented default).

Config-error diagnostics (loud, per artifact, mirroring
server-allowlist): both options set, neither set, `require` set to
anything but `"inherit"`, or an allowlist entry failing
`IsValidModelRef`. Applies to agents only in Phase 5; extending to
skill/command `model` can ride a later minor once policy demand exists.

## API / Interface Changes

- `internal/artifact`: `Agent` struct gains 12 fields (§1, Data Model
  below); `KnownModelAliases`, `IsValidModelRef`; agent enum sets
  exported for rule reuse (`AgentPermissionModes`, `AgentEffortLevels`,
  `AgentColors`, `AgentMemoryScopes`).
- `internal/rules`: new `OptIn` interface; fingerprint buffer gains
  `optin=` component (one-time fingerprint change, acked in Phase 4).
- `internal/engine`: opt-in filtering in `activeRules`;
  `Config.HasRuleBlock`.
- `internal/cli`: `rules` command renders `(opt-in)`; `rules --json`
  adds `opt_in`.
- No CLI flag changes; no config schema changes beyond the existing
  `rule` block semantics.

## Data Model

```go
type Agent struct {
    Base
    Frontmatter Frontmatter
    Body        diag.Range

    Name            string
    Description     string
    Tools           []string
    DisallowedTools []string
    Model           string
    PermissionMode  string
    MaxTurns        int64
    Skills          []string
    HasMCPServers   bool
    HasHooks        bool
    Memory          string
    Background      bool
    Effort          string
    Isolation       string
    Color           string
    InitialPrompt   string

    // PluginDistributed is set by discovery (not the parser) when the
    // agent file lives under a root containing .claude-plugin/plugin.json.
    PluginDistributed bool
}
```

## Testing Strategy

- Parser: table-driven tests asserting every new field's value;
  KeyRange presence for each declared key; absent-key zero values.
- Rules: valid + invalid fixture pairs per rule; a plugin-rooted agent
  fixture (under `testdata/.../.claude-plugin/`) and an identical
  project-level agent proving `plugin-ignored-fields` scoping.
- Opt-in engine behavior: engine-level tests — opt-in rule with no
  block (skipped), empty block (runs), block with `enabled = false`
  (skipped), server-allowlist block without allowlist (config error
  preserved).
- Fingerprint: one deliberate ack when `optin=` joins the buffer.
- Coverage: `internal/rules/agents` must clear the 55% floor.

## Migration / Rollout Plan

1. **Phase 4 PR (minor release):** parser extension + discovery flag +
   five always-on rules + the `OptIn` interface/engine support +
   `mcp/server-allowlist` migration. Release notes call out the
   server-allowlist behavior change: repos that never configured it
   stop seeing per-server config errors (they opted into nothing);
   repos with a block keep exact current behavior.
2. **Phase 5 PR (minor release):** `agents/model-policy` on the now-
   proven mechanism, plus the remaining INV-0006 validation rules.

No data migration; `.claudelint.hcl` files need no edits.

## Open Questions

None. OQ4 (the only contested mechanic) was resolved option (a) by
Donald on 2026-07-10; this document is its write-up.

## References

- [INV-0006](../investigation/0006-rule-coverage-audit-against-current-claude-code-docs.md)
  — audit findings, proposed rule tables
- [IMPL-0004](../impl/0004-phase-4-ruleset-alignment-and-agent-rules.md)
  — phase breakdown, Resolved Decisions (OQ4)
- [DESIGN-0001](0001-claudelint-linter-architecture-and-rule-engine.md)
  — rule interface, range-emission conventions
- [Sub-agents reference](https://code.claude.com/docs/en/sub-agents) —
  16-field frontmatter spec (fetched 2026-07-10)
