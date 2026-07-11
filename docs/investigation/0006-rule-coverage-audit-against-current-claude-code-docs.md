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

> **Implemented by
> [IMPL-0004](../impl/0004-phase-4-ruleset-alignment-and-agent-rules.md)**
> (ruleset v1.5.0, 53 rules): every divergence fix and proposed rule in
> this audit landed across its five phases, with two engine-driven
> severity splits (version-semver/version-missing,
> transport-known/transport-deprecated) and the opt-in mechanism
> specified in
> [DESIGN-0005](../design/0005-agent-rules-and-opt-in-rule-mechanism.md).

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Environment](#environment)
- [Findings](#findings)
  - [Existing rules — audit by kind](#existing-rules--audit-by-kind)
    - [Cross-cutting rules](#cross-cutting-rules)
    - [CLAUDE.md rules](#claudemd-rules)
    - [Skill rules](#skill-rules)
    - [Command rules](#command-rules)
    - [Hook rules](#hook-rules)
    - [Plugin rules](#plugin-rules)
    - [Marketplace rules](#marketplace-rules)
    - [MCP rules](#mcp-rules)
  - [Parser and shared-helper gaps](#parser-and-shared-helper-gaps)
  - [Proposed new rules by kind](#proposed-new-rules-by-kind)
    - [Agents (no rules exist today)](#agents-no-rules-exist-today)
    - [Skills and commands](#skills-and-commands)
    - [Hooks](#hooks)
    - [Marketplaces](#marketplaces)
    - [MCP servers](#mcp-servers)
    - [CLAUDE.md](#claudemd)
    - [Deliberately not chasing yet](#deliberately-not-chasing-yet)
  - [Detail — divergences that produce wrong results today](#detail--divergences-that-produce-wrong-results-today)
  - [Detail — rules stricter than the current spec](#detail--rules-stricter-than-the-current-spec)
  - [Detail — the agent model question](#detail--the-agent-model-question)
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
4. Diff 2 against 3; classify each existing rule and enumerate missing rules,
   per artifact kind.

## Environment

| Component | Version / Value |
|-----------|----------------|
| claudelint ruleset | v1.2.0, fingerprint `e7f26796`, 30 rules |
| claudelint release | v0.2.3 (latest tag) |
| Docs source | `code.claude.com/docs/en/*`, fetched 2026-07-09 |
| Claude Code doc version markers | through v2.1.205 |

## Findings

### Existing rules — audit by kind

Verdict legend:

- **Fix** — produces wrong results against doc-valid input today
- **Update** — behavior/premise/message needs expansion or adjustment
- **Keep** — aligned with current docs, no change
- **Keep (stricter)** — deliberately stricter than spec; document the stance
- **Decide** — stricter than spec in a way that needs a keep/relax call

#### Cross-cutting rules

| Rule | Verdict | Action needed |
|---|---|---|
| `schema/parse` | Keep | None. |
| `schema/frontmatter-required` | Decide | Skill `name`/`description` are optional/recommended per docs (name defaults to dir name; description falls back to first body paragraph). Agent `name`+`description` genuinely required — matches spec. Keep skill checks as best-practice but reword messages to not claim Claude Code requires them. |
| `security/secrets` | Keep | None. Raw-source scan already covers new file surfaces. |
| `style/no-emoji` | Keep | None. |

#### CLAUDE.md rules

| Rule | Verdict | Action needed |
|---|---|---|
| `claude_md/size` | Keep | Docs now advise "under 200 lines"; our default `max_lines = 500` is more lenient. Add a doc note; no behavior change. |
| `claude_md/duplicate-directives` | Keep | None. |

#### Skill rules

| Rule | Verdict | Action needed |
|---|---|---|
| `skills/trigger-clarity` | Keep | Still valid — `description` drives auto-invocation. New `when_to_use` field carries trigger phrases too; rule should check the concatenation once the parser reads it. |
| `skills/body-size` | Keep | Docs guidance is now "under 500 lines" (we count words, default 1000). Compatible; consider an optional `max_lines` variant. |
| `skills/no-version-field` | Keep | Docs now explicitly confirm `version` is accepted-but-ignored. Rule validated; cite the doc in its help text. |

#### Command rules

| Rule | Verdict | Action needed |
|---|---|---|
| `commands/allowed-tools-known` | Fix | (a) String form never splits on commas/whitespace — `allowed-tools: Read, Grep` is one unknown tool. (b) Permission-rule syntax (`Bash(git add:*)`), `mcp__*` patterns, and `Agent(...)` forms all flagged as unknown. (c) `KnownTools` (16 names) predates the `Task` → `Agent` rename and lacks `Agent`/`Skill`. (d) Commands and skills now share one frontmatter model — rule should also run on skill `allowed-tools`. |

#### Hook rules

| Rule | Verdict | Action needed |
|---|---|---|
| `hooks/event-name-known` | Fix | Knows 9 of ~29 documented events; default-error rule fails valid configs (`SubagentStart`, `Setup`, `PermissionRequest`, ...). Expand list. |
| `hooks/timeout-present` | Update | Command hooks have a documented 600 s default timeout — premise ("runaway hook can hang the session") is stale. Keep as style nudge; fix message + docs; account for per-type defaults (prompt 30 s, agent 60 s). |
| `hooks/no-unsafe-shell` | Update | Only meaningful for `type: command` (and only shell-form: `args` present means exec-form, no shell). Needs `type` parsed to scope correctly and to skip exec-form false positives. |

#### Plugin rules

| Rule | Verdict | Action needed |
|---|---|---|
| `plugin/manifest-fields` | Keep (stricter) | Docs require only `name`; `version` optional (falls back to git SHA). Keep requiring `version` as an opinionated default — pinned versions are what marketplaces should ship — but document the stance and consider downgrading the version half to warning. |
| `plugin/semver` | Keep | None. |

#### Marketplace rules

| Rule | Verdict | Action needed |
|---|---|---|
| `marketplace/name` | Keep | Presence check matches spec (kebab-case format is a new-rule candidate below). |
| `marketplace/version-semver` | Update | Root `version` is optional per docs. Split: missing → info, present-but-not-semver → error. |
| `marketplace/author-required` | Update | Docs make root `owner{name}` **required** (we treat author/owner as info-level nicety). Align: missing owner → warning or error; keep parsing both `author` and `owner` shapes (both documented). |
| `marketplace/plugins-nonempty` | Keep | `plugins[]` required per docs; warning severity is a reasonable floor. |
| `marketplace/plugin-source-valid` | Fix | Object sources (`github`/`url`/`git-subdir`/`npm`) parse as empty string → false positive on doc-valid marketplaces. After parser fix, validate documented per-type required fields (`repo`, `url`, `path`, `package`). |
| `marketplace/plugin-name-unique` | Keep | Matches `claude plugin validate` behavior. |
| `marketplace/plugin-name-matches-dir` | Keep | Style rule; scope to local-path sources only (object sources have no local dir). |
| `marketplace/external-source-skipped` | Update | Once object sources are parsed they are structured data, not "skipped" — rework or retire this info rule. |

#### MCP rules

| Rule | Verdict | Action needed |
|---|---|---|
| `mcp/command-required` | Fix | `command` is required **only for stdio** transport. Doc-valid `http`/`ws` servers (url, no command) would false-positive once the `mcpServers` key fix lands. Scope to stdio; pair with new url-required rule. |
| `mcp/server-name-required` | Keep | None. |
| `mcp/command-exists-on-path` | Update | Scope to stdio transport. |
| `mcp/no-unsafe-shell` | Keep | Still correct for stdio `command`. |
| `mcp/no-secrets-in-env` | Update | Docs add `headers` (and `headersHelper`) — secrets in headers are at least as likely as in env. Extend or add sibling rule. |
| `mcp/disabled-commented` | Keep | None. |
| `mcp/server-allowlist` | Keep | None. (Opt-in pattern discussion below applies to the *next* opt-in rule, not this one.) |

### Parser and shared-helper gaps

Several "rule bugs" above are actually parser bugs — the rule layer never
sees the data. These block the fixes and the new rules:

| Parser / helper | Gap | Effect | Blocks |
|---|---|---|---|
| `ParseMCPFile` (`parse_mcp.go:37`) | Reads top-level `servers`; docs standardized on `mcpServers` | Doc-valid `.mcp.json` lints as zero servers — every MCP rule silently skips | All MCP rules |
| `ParseMCPFile` server entries | Only `command`/`args`/`env`/`disabled`; no `type`, `url`, `headers`, `headersHelper`, `oauth`, `timeout`, `alwaysLoad` | Remote transports invisible; transport-aware scoping impossible | `mcp/command-required` fix, url-required, header secrets |
| `ParseMarketplace` (`parse_marketplace.go`) | `source` read as string only | Object sources parse empty → `plugin-source-valid` false positive | Source-shape validation, reserved names |
| `ParseHook` (`parse_json.go`) | `type` ignored; only `command`+`timeout` extracted | `http`/`mcp_tool`/`prompt`/`agent` hooks invisible; typo'd `type` passes | Hook type rules, `no-unsafe-shell` scoping |
| `ParseAgent` (`parse_md_kinds.go:56`) | Only `name`/`description`/`tools` | No agent rules can see `model`, `disallowedTools`, `permissionMode`, `mcpServers`, `hooks`, etc. | Entire agent rule package |
| `asStringList` (`parse_markdown.go:248`) | Never splits comma/space-separated strings | `allowed-tools`/`tools` string forms become single bogus entries | `commands/allowed-tools-known` fix, `agents/tools-known` |
| `ParseSkill`/`ParseCommand` | Missing new fields (`when_to_use`, `model` on command, `context`, `agent`, `disable-model-invocation`, ...) | Limits skill/command rule expansion | `skills/description-length`, fork pairing, model-valid on commands |

### Proposed new rules by kind

Phase column maps to the [Recommendation](#recommendation): **PR 1** =
correctness fixes, **PR 2** = agent package, **PR 3** = policy + design
pass. Every proposed rule lands in one of the three PRs — nothing is
deferred to a later backlog.

#### Agents (no rules exist today)

| Proposed rule | Checks | Severity | Phase |
|---|---|---|---|
| `agents/model-valid` | `model` is an alias (`sonnet`/`opus`/`haiku`/`fable`), `inherit`, or full-ID shape (`^claude-[a-z0-9-]+$`). Catches typos that silently fall back to the inherited model. Same value set applies to skill/command `model` — share the validator. | warning | PR 2 |
| `agents/name-format` | `name` is lowercase letters + hyphens only (documented constraint; duplicates resolve by undocumented filesystem order). | warning | PR 2 |
| `agents/tools-known` | Entries in `tools`/`disallowedTools` are known tools, `mcp__*` patterns, or `Agent(...)` forms — Claude Code **silently ignores** unknown names. | warning | PR 2 |
| `agents/plugin-ignored-fields` | Plugin-distributed agents declaring `mcpServers`, `hooks`, or `permissionMode` — documented as ignored for plugin subagents (dead config). | warning | PR 2 |
| `agents/model-policy` | Opt-in enforcement: `require = "inherit"` or `allowlist = [...]`. | error (opt-in) | PR 3 |
| `agents/field-enums` | `permissionMode` (7 values), `effort` (5), `color` (8), `isolation` (`worktree`), `memory` (`user`/`project`/`local`) are valid enum members. | warning | PR 2 |

#### Skills and commands

| Proposed rule | Checks | Severity | Phase |
|---|---|---|---|
| `skills/description-length` | `description` + `when_to_use` combined ≤ 1,536 chars (documented truncation silently drops trigger phrases). | warning | PR 3 |
| `skills/fork-agent-pairing` | `agent:` set without `context: fork` does nothing. | warning | PR 3 |
| (extend) `commands/allowed-tools-known` → skills | Same rule runs on skill `allowed-tools`/`disallowed-tools` after the merged-model parser update. | error | PR 1 |

#### Hooks

| Proposed rule | Checks | Severity | Phase |
|---|---|---|---|
| `hooks/type-known` | `type` ∈ `command`/`http`/`mcp_tool`/`prompt`/`agent` (absent defaults to `command`). | error | PR 3 |
| `hooks/type-fields` | Per-type required fields present: `command` → `command`; `http` → `url`; `mcp_tool` → `server`+`tool`; `prompt`/`agent` → `prompt`. | error | PR 3 |

#### Marketplaces

| Proposed rule | Checks | Severity | Phase |
|---|---|---|---|
| `marketplace/reserved-name` | Name not in the 16 documented reserved names (and obvious impersonations); claude.ai sync blocks these. | error | PR 3 |
| `marketplace/name-format` | Marketplace + plugin-entry names kebab-case (claude.ai sync rejects violations). | warning | PR 3 |
| `marketplace/source-path-safety` | Relative sources start with `./`; no `..` traversal (validator-rejected). | error | PR 3 |
| `marketplace/renames-valid` | `renames{}` chains terminate (at `null` or a listed plugin) and contain no cycles. | error | PR 3 |

#### MCP servers

| Proposed rule | Checks | Severity | Phase |
|---|---|---|---|
| `mcp/url-required` | `http`/`sse`/`ws` transports declare `url` (counterpart to stdio-scoped `command-required`). | error | PR 1 |
| `mcp/transport-known` | `type` ∈ `stdio`/`http`/`sse`/`ws`; flag `sse` as documented-deprecated (info). | warning | PR 3 |
| `mcp/no-secrets-in-headers` | Secrets scan over `headers` values (or fold into `no-secrets-in-env`). | error | PR 3 |
| `mcp/timeout-minimum` | `timeout` is in **milliseconds** with documented minimum 1000 — catches seconds-vs-ms confusion. | warning | PR 3 |

#### CLAUDE.md

| Proposed rule | Checks | Severity | Phase |
|---|---|---|---|
| `claude_md/import-exists` | `@path` imports resolve on disk; flag chains beyond the documented 4-hop depth. | warning | PR 3 |

#### Deliberately not chasing yet

Path-scoped rule files (`.claude/rules/*.md` with `paths:` frontmatter) as a
ninth artifact kind, LSP servers, output styles, monitors
(`experimental.*`), and `userConfig` schema validation — monitors/themes are
explicitly marked experimental in the docs; revisit when they stabilize.

### Detail — divergences that produce wrong results today

Supporting detail for the **Fix** verdicts above.

**`.mcp.json` top-level key.** The watch-item in DESIGN-0002 §2.2 has
resolved against us: every current doc example uses `mcpServers{}` at the
root of project `.mcp.json`. `ParseMCPFile` reads `servers` — a
doc-conforming `.mcp.json` lints as "no servers" and every MCP rule silently
skips. Highest-value single fix in the audit. Migration: accept `mcpServers`
as primary, keep `servers` for one release with a deprecation info
diagnostic (mirrors how the hook flat-shape removal was handled in #14/#18).

**Hook events and types.** The event list grew from our 9 to ~29, and hook
entries gained four non-command types with distinct required fields and
timeout defaults (command/http/mcp_tool 600 s; prompt 30 s; agent 60 s;
`UserPromptSubmit` caps at 30 s). Command hooks also gained `args`
(exec-form — no shell, so shell-smell heuristics don't apply), `async`,
`asyncRewake`, `shell`, `if`, and `statusMessage`. `disableAllHooks` is a
new settings-level key.

**Marketplace sources.** Docs define object sources — `github{repo,ref,sha}`,
`url{url,ref,sha}`, `git-subdir{url,path,ref,sha}`, `npm{package,version,registry}` —
alongside `./`-prefixed relative strings. Note: INV-0005's nested
`metadata.version` false positive is now **documented** backward-compat
(`metadata.version`, `metadata.description`, `owner{name,email}`), so our
existing fallback parsing is spec-compliant, not defensive.

**`allowed-tools` forms.** Docs (skills + slash-commands, now one merged
model): YAML list **or** space/comma-separated string; entries may be
permission rules (`Bash(git add:*)`), MCP patterns (`mcp__github`,
`mcp__server__*`), or `Agent(...)` forms. `Task` was renamed `Agent` in
v2.1.63 (both valid).

### Detail — rules stricter than the current spec

Supporting detail for the **Decide** / **Keep (stricter)** verdicts: being
stricter than the runtime is legitimate for a linter — Claude Code's
runtime posture is "silently tolerate", ours is "loudly flag" — but each
divergence should be a documented stance, not an accident. The rules doc
should carry a "stricter than spec, by design" note on: skill
`name`/`description` requiredness, plugin `version` requiredness. The one
genuine correction: `marketplace/version-semver` errors on a *missing*
root version that the docs call optional — that half should soften to info.

### Detail — the agent model question

What the docs specify:

- Valid `model` values: aliases (`sonnet`, `opus`, `haiku`, `fable`), full
  model IDs (e.g. `claude-opus-4-8`), or `inherit`.
- **Omitted means `inherit`** — an explicit `inherit` and an absent key are
  equivalent (since v2.1.196 even via the `CLAUDE_CODE_SUBAGENT_MODEL`
  override path).
- Values excluded by an org's `availableModels` allowlist are skipped at
  resolution and silently fall back to the inherited model.

Hence the two-rule split in the tables above: `agents/model-valid` is
always-on typo protection (a bad value never errors at runtime — it
silently resolves to the parent model); `agents/model-policy` is governance
for repos/marketplaces that want a stance, where `require = "inherit"`
treats an absent key as compliant and `allowlist` permits deliberate
pinning (e.g. `haiku` for cheap utility agents).

**Pattern tension to resolve in PR 3's design:** the house opt-in pattern
(`mcp/server-allowlist`) is "default-enabled, nil option → loud config
error" — right for security, wrong for governance (every un-configured
repo would yell about every agent). CLAUDE.md anticipated this: "revisit
only if multiple rules need the pattern." `agents/model-policy` is that
second rule. Options: add an opt-in mechanism (config-driven enable or
`Rule.Enabled()`), or accept a documented silent no-op when unset — silent
no-op is defensible here because absence-of-policy is a valid state,
unlike absence-of-allowlist on a security rule.

## Conclusion

**Answer: Yes — the ruleset has drifted, in both directions.**

Of 30 shipping rules: **4 Fix** (wrong against doc-valid input),
**7 Update**, **17 Keep**, **2 Decide/Keep-stricter** — see the per-kind
tables. The worst two fixes are the `.mcp.json` `servers` → `mcpServers`
divergence (valid files lint as empty) and the 9-of-29 hook-event list
(valid configs fail with errors). Seven parser gaps block most of the
fixes and all of the agent work. Agents have the largest greenfield gap: a
16-field documented spec, three silent-failure behaviors worth linting,
and zero agent rules shipped.

On the model question: yes, and it should be two rules — always-on
**validation** of the value set (`sonnet`/`opus`/`haiku`/`fable`/full-ID/
`inherit`), and **opt-in enforcement** (`require = "inherit"` or an
allowlist), which forces the anticipated opt-in-pattern design decision.

## Recommendation

Phase the work as three PRs, in this order:

1. **PR 1 — correctness fixes (no new rules except `mcp/url-required`):**
   everything marked Fix + the parser gaps that back them — `mcpServers`
   key (deprecation path for `servers`), full hook-event list, marketplace
   object sources, `allowed-tools` string splitting + permission-rule
   syntax, `KnownTools` refresh. Split `marketplace/version-semver`
   severities. Update DESIGN-0002 §2.2 and the affected rules docs.
2. **PR 2 — agent rule package:** extend `ParseAgent` to the documented
   field set, then `agents/model-valid`, `agents/name-format`,
   `agents/tools-known`, `agents/plugin-ignored-fields`,
   `agents/field-enums`. Fingerprint bump; per-package coverage gate
   applies.
3. **PR 3 — policy + design pass:** `agents/model-policy` plus the opt-in
   mechanism decision, hook type rules (`type-known`, `type-fields`), MCP
   transport rules (`transport-known`, `no-secrets-in-headers`,
   `timeout-minimum`), marketplace name/safety rules (`reserved-name`,
   `name-format`, `source-path-safety`, `renames-valid`),
   `skills/description-length`, `skills/fork-agent-pairing`, and
   `claude_md/import-exists`.

Write DESIGN-0005 covering PRs 2–3 before implementing; PR 1 is executable
directly against DESIGN-0001/0002 with doc updates to the affected sections.

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
