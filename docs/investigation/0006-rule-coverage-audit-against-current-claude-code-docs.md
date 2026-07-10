---
id: INV-0006
title: "Rule coverage audit against current Claude Code docs"
status: Concluded
author: Donald Gifford
created: 2026-07-09
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0006: Rule coverage audit against current Claude Code docs

**Status:** Concluded
**Author:** Donald Gifford
**Date:** 2026-07-09

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Environment](#environment)
- [Findings](#findings)
  - [Observation 1 — divergences that produce wrong results today](#observation-1--divergences-that-produce-wrong-results-today)
  - [Observation 2 — rules stricter than the current spec](#observation-2--rules-stricter-than-the-current-spec)
  - [Observation 3 — agents are a coverage hole](#observation-3--agents-are-a-coverage-hole)
  - [Observation 4 — the agent model question](#observation-4--the-agent-model-question)
  - [Observation 5 — new surface with no coverage yet](#observation-5--new-surface-with-no-coverage-yet)
- [Conclusion](#conclusion)
- [Recommendation](#recommendation)
- [References](#references)
<!--toc:end-->

## Question

Is the shipping ruleset (v1.2.0, fingerprint `e7f26796`, 30 rules) still
aligned with the current Anthropic Claude Code documentation for agents,
skills, commands, plugins, marketplaces, hooks, MCP servers, and CLAUDE.md —
and specifically, should claudelint validate the agent `model` frontmatter
field and/or offer an enforcement rule for pinning a model vs. requiring
`inherit`?

## Hypothesis

The ruleset lags the docs. Claude Code has shipped continuously since the
Phase 2 rules were written (docs carry version markers through v2.1.205);
expected gaps: new hook events, new frontmatter fields, and zero agent-kind
rules. Suspected but unconfirmed going in: the `.mcp.json` top-level key
divergence flagged in DESIGN-0002 §2.2 ("if Claude Code standardizes on
`mcpServers`, revisit").

## Context

Triggered by the post-v0.2.0 question "are we missing anything from the most
current Anthropic documentation?", plus a concrete feature question about
enforcing agent `model` frontmatter. The last systematic doc-vs-ruleset pass
was INV-0005 (the `claude-skills` dogfood) during Phase 2.

**Triggered by:** DESIGN-0002 §2.2 open watch-item; user request 2026-07-09

## Approach

1. Dump the shipping catalog: `claudelint rules --json` (30 rules,
   v1.2.0/`e7f26796`).
2. Inventory what the parsers and rules actually check, with file:line refs —
   hardcoded tool list, hook-event list, required-frontmatter keys, MCP/plugin/
   marketplace field extraction, thresholds.
3. Fetch the current official docs for each artifact kind (sub-agents, skills,
   slash-commands, plugins-reference, plugin-marketplaces, hooks, hooks-guide,
   mcp, memory) and extract the full field/constraint tables.
4. Diff 2 against 3; classify each gap as (a) wrong-today, (b) stricter than
   spec, or (c) new surface with no coverage.

## Environment

| Component | Version / Value |
|-----------|----------------|
| claudelint ruleset | v1.2.0, fingerprint `e7f26796`, 30 rules |
| claudelint release | v0.2.3 (latest tag) |
| Docs source | `code.claude.com/docs/en/*`, fetched 2026-07-09 |
| Claude Code doc version markers | through v2.1.205 |

## Findings

### Observation 1 — divergences that produce wrong results today

These are cases where linting a config that is **valid per current docs**
produces a false positive, or where clearly invalid input passes silently.

**1a. Hook event list is 9 of ~29 events — false errors on valid configs.**
`KnownHookEvents` (`internal/artifact/knowndata.go:42-52`) has `PreToolUse`,
`PostToolUse`, `UserPromptSubmit`, `Notification`, `Stop`, `SubagentStop`,
`PreCompact`, `SessionStart`, `SessionEnd`. The docs now also define `Setup`,
`SessionEnd` reasons aside, `UserPromptExpansion`, `StopFailure`,
`PostToolUseFailure`, `PostToolBatch`, `PermissionRequest`,
`PermissionDenied`, `MessageDisplay`, `ConfigChange`, `CwdChanged`,
`FileChanged`, `WorktreeCreate`, `WorktreeRemove`, `PostCompact`,
`SubagentStart`, `TeammateIdle`, `TaskCreated`, `TaskCompleted`,
`InstructionsLoaded`, `Elicitation`, `ElicitationResult`. Because
`hooks/event-name-known` is a default-**error** rule, a settings file using
`SubagentStart` (documented) fails the lint today.

**1b. Hook entries now have five types; we only understand `command`.**
Valid `type` values are `command`, `http`, `mcp_tool`, `prompt`, `agent`,
each with different required fields (`command` / `url` / `server`+`tool` /
`prompt`). The parser (`internal/artifact/parse_json.go`) ignores `type`
entirely and only extracts `command` + `timeout`. Consequences: an `http`
hook is parsed as a command-less entry (no rule catches a missing `url`); a
typo'd `type` passes silently; `hooks/no-unsafe-shell` never inspects
`prompt`/`agent` hooks. Also: command hooks have a **default 600 s timeout**
per the docs, so `hooks/timeout-present`'s premise ("a runaway hook can hang
the session") is now a style nudge, not a correctness issue — its message
and doc entry should say so.

**1c. `.mcp.json` top-level key: docs say `mcpServers`, we parse `servers`.**
The watch-item in DESIGN-0002 §2.2 has resolved against us: every current
doc example uses `mcpServers{}` at the root of project `.mcp.json`
(`code.claude.com/docs/en/mcp`). `ParseMCPFile`
(`internal/artifact/parse_mcp.go:37`) reads `servers` — a doc-conforming
`.mcp.json` lints as "no servers" (all MCP rules silently skip). This is the
highest-value single fix in the audit.

**1d. MCP transports: we only model stdio.**
Server entries support `type` (`stdio` default, `http`, `sse` — deprecated,
`ws`), `url`, `headers`, `headersHelper`, `oauth{}`, `timeout` (ms, min
1000), `alwaysLoad`. We parse none of these. `mcp/command-required` would
false-positive on a valid `http` server (which has `url`, no `command`) the
moment the 1c fix lands, so 1c and 1d must ship together:
`command-required` applies to stdio only; a matching `url`-required check
covers the remote transports.

**1e. Marketplace plugin sources can be objects; we only parse strings.**
Docs define object sources — `{"source": "github", "repo": ...}`,
`{"source": "url", ...}`, `{"source": "git-subdir", ...}`,
`{"source": "npm", ...}` — alongside relative-path strings. The parser
(`internal/artifact/parse_marketplace.go`) reads `source` as a string, so an
object source parses as empty and `marketplace/plugin-source-valid` (error)
fires on a doc-valid marketplace. Note this is the same class of bug as the
nested-`metadata.version` false positive found in INV-0005 — the docs have
since **confirmed** `metadata.version`/`owner{}` as documented
backward-compat shapes, so our existing fallback parsing is correct and can
be annotated as spec-compliant rather than defensive.

**1f. `allowed-tools` string forms and permission-rule syntax.**
Docs (skills + slash-commands): `allowed-tools` accepts a YAML list **or a
space/comma-separated string**, and entries may be permission rules like
`Bash(git add:*)`, `mcp__github`, or `Agent(worker)`. Our `asStringList`
(`internal/artifact/parse_markdown.go:248`) never splits strings, so
`allowed-tools: Read, Grep` is one unknown tool named `"Read, Grep"` →
`commands/allowed-tools-known` (error) false-positives. The rule also flags
any parenthesized or `mcp__*` entry. Separately, `KnownTools`
(`internal/artifact/knowndata.go:12-29`, 16 names) predates the `Task` →
`Agent` rename (v2.1.63; both valid) and lacks `Agent`, `Skill`, and the
MCP wildcard forms.

**1g. Commands and skills have merged.**
`.claude/commands/*.md` still works but is documented as a legacy alias for
skills; both share one frontmatter model (17 fields — `when_to_use`,
`arguments`, `argument-hint`, `disable-model-invocation`, `user-invocable`,
`allowed-tools`, `disallowed-tools`, `model`, `effort`, `context`, `agent`,
`hooks`, `paths`, `shell`, ...). Our `KindCommand`/`KindSkill` split remains
useful for discovery, but rules that only run on one kind (e.g.
`commands/allowed-tools-known` not running on skills) now leave documented
fields unlinted on the other kind.

### Observation 2 — rules stricter than the current spec

Not bugs — places where claudelint demands more than Claude Code requires.
Each needs a deliberate keep/relax decision, and the rules doc should state
"stricter than spec, by design" for the keepers.

| Rule | We require | Docs say |
|---|---|---|
| `plugin/manifest-fields` | `name` + `version` | only `name` required; missing `version` falls back to git commit SHA |
| `marketplace/version-semver` | `version` present + semver | root `version` is optional |
| `schema/frontmatter-required` (skill) | `name` + `description` | `name` optional (defaults to dir name); `description` recommended (falls back to first body paragraph) |
| `schema/frontmatter-required` (agent) | `name` + `description` | both genuinely required — matches spec |

Recommendation per row is in the Recommendation section. The skill-name
check has a subtlety worth keeping: `name` only controls invocation for
plugin-root SKILL.md, so requiring it is cheap hygiene, but the diagnostic
message should stop implying Claude Code requires it.

### Observation 3 — agents are a coverage hole

`KindAgent` exists, but the parser extracts only `name`, `description`,
`tools` (`internal/artifact/parse_md_kinds.go:56-70`), and **zero
agent-specific rules are registered** — agents get `schema/frontmatter-required`
plus the every-kind rules and nothing else.

The current sub-agents doc defines 16 frontmatter fields: `name`,
`description`, `model`, `tools`, `disallowedTools`, `permissionMode`,
`maxTurns`, `skills`, `mcpServers`, `hooks`, `memory`, `background`,
`effort`, `isolation`, `color`, `initialPrompt`. Three documented behaviors
make agents unusually lint-worthy:

1. **Unknown tool names in `tools` are silently ignored** by Claude Code —
   a typo'd tool never errors at runtime; the agent just quietly lacks it.
   This is exactly the failure mode a linter exists for.
2. **`name` has a real constraint** (lowercase letters and hyphens only)
   that Claude Code does not loudly enforce; duplicates resolve by
   undocumented filesystem order.
3. **Plugin subagents silently ignore `mcpServers`, `hooks`, and
   `permissionMode`** — declaring them in a plugin's `agents/*.md` does
   nothing. A rule can catch dead config that authors believe is active.

### Observation 4 — the agent model question

What the docs actually specify (`code.claude.com/docs/en/sub-agents.md`):

- Valid `model` values: aliases (`sonnet`, `opus`, `haiku`, `fable`), full
  model IDs (e.g. `claude-opus-4-8`), or `inherit`.
- **Omitted means `inherit`** — the default changed to inherit-from-parent;
  an explicit `inherit` and an absent key are equivalent (since v2.1.196
  even via the `CLAUDE_CODE_SUBAGENT_MODEL` env override path).
- Values excluded by an org's `availableModels` allowlist are skipped at
  resolution time and fall back to the inherited model — misconfiguration
  degrades silently rather than failing.

That maps onto two rules with different jobs:

**`agents/model-valid` (validation — default-enabled, warning).** If `model`
is present, check it is an alias, `inherit`, or matches a full-ID shape
(`^claude-[a-z0-9-]+$`). Catches `model: sonet` typos, which today silently
resolve to the inherited model. No options needed. Also applies verbatim to
the skill/command `model` field (same value set per the skills doc), so the
check should live in shared validation with per-kind registration.

**`agents/model-policy` (enforcement — opt-in).** Governance rule for repos
and marketplaces that want a stance, with an option taking one of two
shapes:

    rule "agents/model-policy" {
      options = { require = "inherit" }   # every agent must inherit
    }

    rule "agents/model-policy" {
      options = { allowlist = ["inherit", "haiku", "sonnet"] }
    }

`require = "inherit"` treats an absent key as compliant (absence ==
inherit per the docs) and flags any pinned model — the right default for
cost/consistency governance and for marketplace review. `allowlist` permits
deliberate pinning (e.g. cheap models for utility agents) while blocking
everything else.

**Pattern tension worth deciding explicitly:** the house pattern
(`mcp/server-allowlist`, per CLAUDE.md) is "default-enabled, nil option →
loud config error". That is right for a security rule; for a governance
rule it would make every un-configured repo yell about every agent.
CLAUDE.md already anticipated this: "Adding `Rule.Enabled()` for one rule
is over-engineering; **revisit only if multiple rules need the pattern**."
`agents/model-policy` is that second rule. The Phase 4 design should either
add an opt-in mechanism (config-driven enable, or `Rule.Enabled()`) or
accept a documented silent no-op when the option is unset — silent no-op is
acceptable here precisely because absence-of-policy is a valid state,
unlike absence-of-allowlist on a security rule.

### Observation 5 — new surface with no coverage yet

Lower urgency; candidates for the backlog rather than the next release.

- **Agents (beyond Observation 3/4):** `agents/name-format` (lowercase +
  hyphens), `agents/tools-known` (the silent-ignore catch),
  `agents/plugin-ignored-fields` (plugin agents declaring
  `mcpServers`/`hooks`/`permissionMode`), enum checks for
  `permissionMode`/`effort`/`color`/`isolation`/`memory`.
- **Skills:** `skills/description-length` — `description` + `when_to_use`
  are truncated at 1,536 combined chars, so overlong descriptions lose
  their trigger phrases silently; body-size guidance is now "under 500
  lines" (our `max_words = 1000` is compatible; consider a lines variant).
  `context: fork` / `agent` pairing check (an `agent` field without
  `context: fork` does nothing).
- **Marketplaces:** reserved-name check (16 reserved/impersonation-blocked
  marketplace names documented), `renames{}` cycle validation, source path
  traversal (`..` rejected), relative sources must start with `./`,
  kebab-case name format (claude.ai sync rejects violations).
- **Hooks:** per-type required-field validation once 1b lands;
  `disableAllHooks` awareness; `timeout` unit sanity (seconds, not ms).
- **MCP:** `sse` transport deprecation warning; secrets scanning in
  `headers` (today only `env` is scanned); `timeout` minimum (1000 ms).
- **CLAUDE.md:** `@path` import resolution (imports exist on disk; ≤ 4-hop
  depth documented); the docs' size guidance is "under 200 lines" vs our
  default `max_lines = 500` — defensible, but worth a comment in the rule.
- **Not worth chasing yet:** path-scoped rule files (`.claude/rules/*.md`
  with `paths:` frontmatter) as a ninth artifact kind, LSP servers, output
  styles, monitors (`experimental.*`), `userConfig` schema validation.
  Revisit when they stabilize (monitors/themes are explicitly marked
  experimental).

## Conclusion

**Answer: Yes — the ruleset has drifted, in both directions.**

Seven findings produce wrong results today (Observation 1), the worst being
the `.mcp.json` `servers` → `mcpServers` divergence (valid files lint as
empty) and the 9-of-29 hook-event list (valid configs fail with errors).
Four rules are deliberately-or-accidentally stricter than the documented
spec (Observation 2). Agents — the artifact kind Donald asked about — have
the largest gap: a 16-field documented spec, three silent-failure behaviors
worth linting, and zero agent rules shipped (Observation 3).

On the model question (Observation 4): yes, and it should be two rules —
always-on **validation** of the value set (`sonnet`/`opus`/`haiku`/`fable`/
full-ID/`inherit`), and **opt-in enforcement** (`require = "inherit"` or an
allowlist) following — and finally forcing the anticipated revisit of — the
opt-in-rule pattern.

## Recommendation

Phase the work as three PRs, in this order:

1. **Correctness fixes (patch/minor, no new rules):** expand
   `KnownHookEvents` to the full documented set; accept `mcpServers` as the
   primary `.mcp.json` key (keep `servers` with a deprecation info
   diagnostic for one release); parse marketplace object sources so
   `plugin-source-valid` stops false-positive-ing (and start validating the
   documented per-type required fields); split string-form `allowed-tools`
   on commas/whitespace and accept permission-rule + `mcp__*` syntax;
   refresh `KnownTools` (add `Agent`, `Skill`; keep `Task` as alias).
   Update Observation 2 rows: keep `plugin/manifest-fields` and skill
   frontmatter checks as documented stricter-than-spec defaults, but split
   `marketplace/version-semver` (missing → info, non-semver → error).
2. **Agent rule package (minor):** `agents/model-valid`,
   `agents/name-format`, `agents/tools-known`,
   `agents/plugin-ignored-fields` — requires extending the agent parser to
   the documented field set (at minimum `model`, `disallowedTools`,
   `permissionMode`, `mcpServers`, `hooks`, plus ranges). This bumps the
   ruleset fingerprint and needs the coverage-gate check per package.
3. **Policy + design pass:** `agents/model-policy` plus the opt-in
   mechanism decision (second consumer of the pattern → design the
   general solution), hook per-type validation, MCP transport rules.
   Items in Observation 5 feed the backlog.

Write a DESIGN doc (Phase 4 / DESIGN-0005) covering 2–3 before
implementing; item 1 is executable directly against DESIGN-0001/0002 with
doc updates to the affected sections (DESIGN-0002 §2.2 in particular).

## References

- Rule catalog: `claudelint rules --json` @ v1.2.0 / `e7f26796`
- Source inventory: `internal/artifact/knowndata.go`,
  `internal/artifact/parse_json.go`, `internal/artifact/parse_mcp.go`,
  `internal/artifact/parse_marketplace.go`,
  `internal/artifact/parse_md_kinds.go`, `internal/rules/**`
- [Sub-agents](https://code.claude.com/docs/en/sub-agents) — frontmatter
  spec incl. `model` resolution and `inherit` default
- [Skills](https://code.claude.com/docs/en/skills) and
  [Slash commands](https://code.claude.com/docs/en/slash-commands) —
  merged frontmatter model, `allowed-tools` forms, 1,536-char cap
- [Hooks reference](https://code.claude.com/docs/en/hooks) and
  [Hooks guide](https://code.claude.com/docs/en/hooks-guide) — event list,
  five hook types, timeout defaults
- [MCP](https://code.claude.com/docs/en/mcp) — `mcpServers` key,
  transports, oauth/headers fields
- [Plugins reference](https://code.claude.com/docs/en/plugins-reference)
  and [Plugin marketplaces](https://code.claude.com/docs/en/plugin-marketplaces)
  — manifest schemas, source types, reserved names, `renames`
- [Memory](https://code.claude.com/docs/en/memory) — CLAUDE.md locations,
  `@path` imports, size guidance
- [DESIGN-0002](../design/0002-phase-2-marketplaces-mcp-rules-and-github-action.md)
  §2.2 — the `servers` vs `mcpServers` watch-item this INV resolves
- [INV-0005](./0005-phase-2-dogfood-findings-marketplaces-mcp-and-spec-divergence.md)
  — prior dogfood pass; nested `metadata.version` finding now confirmed
  as documented backward-compat
