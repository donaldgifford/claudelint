---
id: IMPL-0004
title: "Phase 4 - Ruleset alignment and agent rules"
status: Draft
author: Donald Gifford
created: 2026-07-09
---
<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL 0004: Phase 4 - Ruleset alignment and agent rules

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-07-09

<!--toc:start-->
- [Objective](#objective)
- [Scope](#scope)
  - [In Scope](#in-scope)
  - [Out of Scope](#out-of-scope)
- [Implementation Phases](#implementation-phases)
  - [Phase 1 — Parser foundations](#phase-1--parser-foundations)
    - [Tasks](#tasks)
    - [Success Criteria](#success-criteria)
  - [Phase 2 — Correctness rule fixes](#phase-2--correctness-rule-fixes)
    - [Tasks](#tasks-1)
    - [Success Criteria](#success-criteria-1)
  - [Phase 3 — DESIGN-0005 for the agent package and opt-in mechanism](#phase-3--design-0005-for-the-agent-package-and-opt-in-mechanism)
    - [Tasks](#tasks-2)
    - [Success Criteria](#success-criteria-2)
  - [Phase 4 — Agent rule package](#phase-4--agent-rule-package)
    - [Tasks](#tasks-3)
    - [Success Criteria](#success-criteria-3)
  - [Phase 5 — Policy and remaining validation rules](#phase-5--policy-and-remaining-validation-rules)
    - [Tasks](#tasks-4)
    - [Success Criteria](#success-criteria-4)
- [File Changes](#file-changes)
- [Testing Plan](#testing-plan)
- [Dependencies](#dependencies)
- [Resolved Decisions](#resolved-decisions)
- [References](#references)
<!--toc:end-->

## Objective

Close the drift found in
[INV-0006](../investigation/0006-rule-coverage-audit-against-current-claude-code-docs.md):
fix the four rules that produce wrong results against doc-valid input, update
the seven that have stale premises, extend the parsers that block those fixes,
and ship the agent rule package (including `model` validation and the opt-in
`model-policy` enforcement rule). At completion the ruleset validates every
artifact kind against the Claude Code docs as of 2026-07-09 (version markers
through v2.1.205).

**Implements:** INV-0006 Recommendation (PR 1 → Phases 1–2, PR 2 → Phases 3–4,
PR 3 → Phase 5).

## Scope

### In Scope

- Parser changes: `.mcp.json` `mcpServers` key + transport fields,
  marketplace object sources, hook `type` + per-type fields, comma/space
  splitting for tool lists, agent frontmatter field extension.
- `KnownTools` and `KnownHookEvents` refresh.
- Fixes/updates to the 11 existing rules marked Fix/Update in INV-0006.
- New rules: `mcp/url-required` plus the agent package
  (`agents/model-valid`, `agents/name-format`, `agents/tools-known`,
  `agents/plugin-ignored-fields`, `agents/field-enums`) and the Phase 5
  batch (`agents/model-policy`, `hooks/type-known`, `hooks/type-fields`,
  `mcp/transport-known`, `mcp/no-secrets-in-headers`, `mcp/timeout-minimum`,
  `marketplace/reserved-name`, `marketplace/name-format`,
  `marketplace/source-path-safety`, `marketplace/renames-valid`,
  `skills/description-length`, `skills/fork-agent-pairing`,
  `claude_md/import-exists`).
- DESIGN-0005 covering the agent package and the opt-in rule mechanism.
- Ruleset version + fingerprint bumps per phase; `docs/rules/rules.md` and
  README anchors updated alongside each rule change.
- One dogfood pass against `donaldgifford/claude-skills` per shipped PR.

### Out of Scope

- Experimental surfaces: monitors, themes, LSP servers, `userConfig` schema
  validation (docs mark these unstable; INV-0006 "deliberately not chasing
  yet").
- Path-scoped rule files (`.claude/rules/*.md`) as a ninth artifact kind.
- `convert` subcommand (original Phase 3 plan, still gated on INV-0001).
- Any engine-level scheduling/discovery changes beyond what the opt-in
  mechanism decision (OQ4) requires.

---

## Implementation Phases

Phases 1+2 ship together as one PR/release; Phase 4 and Phase 5 are one
PR/release each. Phase 3 is a docs-only PR. Every code phase follows the
house rule-change checklist: rule file + table-driven tests + `Register()` +
`rules.DefaultHelpURI` anchor in README + `docs/rules/rules.md` entry +
fingerprint guardrail acknowledgment + coverage-gate floor (55% per
`internal/` package).

---

### Phase 1 — Parser foundations

Extend the artifact layer so the rule layer can see what the docs define.
No rule behavior changes in this phase beyond what parsing unlocks; all
new fields carry ranges per the range-emission conventions (pre-parsed
field ranges preferred).

#### Tasks

- [x] `ParseMCPFile`: accept `mcpServers` as the primary top-level key;
      keep `servers` accepted but tag the artifact so a deprecation
      diagnostic can fire (per OQ1 decision). _(`MCPServer.LegacyServersKey`
      set for servers{}-keyed files; mcpServers wins when both present)_
- [x] `MCPServer` type: parse `type` (default `stdio`), `url`, `headers`
      (string map), `headersHelper`, `timeout` (number, ms), `alwaysLoad`;
      add ranges for `type` and `url`. `oauth{}` parsed as present/absent
      only (no field validation this phase). _(`Transport` holds the raw
      declared value; `EffectiveTransport()` applies the stdio default so
      rules can distinguish declared-vs-defaulted)_
- [x] `ParseMarketplace`: parse `source` as string **or** object; model the
      four documented object shapes (`github{repo,ref,sha}`,
      `url{url,ref,sha}`, `git-subdir{url,path,ref,sha}`,
      `npm{package,version,registry}`) into a typed `MarketplaceSource`
      with a `Kind` discriminator; keep `Resolved` semantics for local
      string paths. _(`MarketplacePlugin.SourceInfo`; string forms classify
      as `local`/`external-string`, unknown object discriminators as
      `invalid`; `SourceRange` covers the object span for object forms)_
- [x] `ParseMarketplace`: parse root `owner{name,email}` as a distinct
      field (today folded into `Author`); parse `renames{}` map (for
      Phase 5). _(`OwnerName`/`OwnerEmail`/`OwnerRange` + `Renames`
      map — JSON null targets parse as "" meaning "removed"; legacy
      merged `Author` view preserved for existing rules)_
- [x] `ParseHook`: parse `type` (default `command` when absent), `url`,
      `server`, `tool`, `prompt`, `args` (presence = exec-form), `async`,
      and `shell` per hook entry; keep `command`/`timeout`/`matcher`
      extraction as-is. _(`HookEntry.Type` holds the raw declared value;
      `EffectiveType()` applies the command default; `ExecForm` flags
      args[] presence so no-unsafe-shell can skip direct-spawn entries)_
- [x] Tool-list splitting: new shared helper that accepts YAML list, or
      comma/whitespace-separated string, and returns entries — used by
      command/skill `allowed-tools`, `disallowed-tools`, and agent
      `tools`/`disallowedTools`. Entries preserve permission-rule syntax
      (`Bash(git add:*)`) and `mcp__*` patterns as single tokens.
      _(`SplitToolList` splits on commas/whitespace only outside
      parentheses; YAML list elements pass through verbatim. Skill and
      command `allowed-tools` + agent `tools` now use it; the
      `disallowed-tools`/`disallowedTools` call sites land with the
      new-field parsing task below)_
- [x] `KnownTools`: add `Agent`, `Skill`; keep `Task` (documented alias);
      add a helper that classifies `mcp__<server>`, `mcp__<server>__*`,
      `Agent(...)`, and `Tool(args)` permission-rule forms as
      structurally-valid rather than unknown. _(`IsToolPattern` — accepts
      MCP patterns and permission-rule forms whose base is a known tool
      or MCP pattern with a non-empty specifier; bare names stay
      `IsKnownTool`'s job)_
- [x] `KnownHookEvents`: expand to the full documented event set
      (~29 events; exact list + exact casing from the hooks reference,
      recorded in a table in the rules doc). _(30 events as of 2026-07 —
      the reference grew by one since the INV-0006 audit; lifecycle-grouped
      table added to `docs/rules/rules.md` under `hooks/event-name-known`,
      count pinned by test)_
- [x] `ParseSkill`/`ParseCommand`: parse `when_to_use`, `model` (command),
      `context`, `agent`, `disable-model-invocation`, `user-invocable`,
      `disallowed-tools` — fields needed by Phases 2 and 5.
      _(Skill and Command now share the merged frontmatter model;
      `UserInvocable` is a `*bool` so rules can tell declared-false from
      absent-defaults-true; `disallowed-tools` goes through the shared
      splitter)_
- [x] Fixture sweep: add doc-valid testdata files exercising every new
      shape (an `mcpServers` file with `http` + `stdio` servers, a
      marketplace with all four object source types, a hooks file using
      new events and all five hook types, skills/commands with
      string-form `allowed-tools`). _(mcp/marketplace fixtures landed
      with their parser tasks; `all_types.json` gained `Setup` +
      `PermissionRequest` groups; new `ok/skills/deployer.md` and
      `ok/commands/commit.md` cover string-form tool lists + merged
      model fields, wired into `TestFixturesOK`)_

#### Success Criteria

- All new fields parse with correct values and ranges in unit tests.
- The doc-valid fixtures parse with **zero** `schema/parse` errors.
- Existing fixtures still parse identically (no behavior change for old
  shapes except the `servers`-key tagging).
- `just check` green; coverage floor holds for `internal/artifact`.

---

### Phase 2 — Correctness rule fixes

Fix the four wrong-today rules, adjust the Update-verdict rules, and add
`mcp/url-required`. Ships with Phase 1 as one PR (ruleset minor bump,
fingerprint change).

#### Tasks

- [x] `hooks/event-name-known`: validate against the expanded event list;
      diagnostic suggests the nearest known event on likely typos
      (case-insensitive match). _(expanded list picked up automatically
      via `IsKnownHookEvent`; new `artifact.SuggestHookEvent` powers the
      did-you-mean message; rules.md notes the suggestion behavior)_
- [x] `commands/allowed-tools-known`: consume the shared splitter; accept
      permission-rule and `mcp__*` forms; extend `AppliesTo` to run on
      skill `allowed-tools`/`disallowed-tools` too (per OQ7 decision).
      _(splitter consumed at parse time; rule accepts `IsToolPattern`
      forms and checks both tool-list keys on command + skill. The
      `AppliesTo` change flipped the fingerprint, so the ruleset minor
      bump landed here: `v1.3.0`, fingerprint tracked in
      `expected_fingerprint.txt` per subsequent drift)_
- [x] `mcp/command-required`: fire only when `type` is `stdio` (or absent).
      _(gates on `EffectiveTransport() == "stdio"`; message now names the
      transport requirement)_
- [x] New `mcp/url-required` (schema/error): `http`/`sse`/`ws` transports
      must declare a non-empty `url`. _(range prefers `TransportRange`;
      unknown transports deliberately out of scope pending
      `mcp/transport-known` in Phase 5; rules.md + README rows added,
      shared prose section with `mcp/command-required`)_
- [x] `mcp/command-exists-on-path`: skip non-stdio transports.
      _(no fingerprint impact — behavior-only change inside Check)_
- [x] `marketplace/plugin-source-valid`: object sources validate per-type
      required fields (`repo` / `url` / `url`+`path` / `package`); string
      sources keep the non-empty check; `sha` when present must be a
      40-char hex string. _(one diagnostic per missing requirement,
      anchored to `SourceRange`; required-fields table added to
      rules.md)_
- [x] `marketplace/external-source-skipped`: rework — object sources are
      now structured; the info diagnostic applies only to sources whose
      content genuinely can't be checked locally. _(fires on the four
      remote object kinds + remote string shorthands with a
      kind-aware locator in the message; absent/invalid sources left to
      plugin-source-valid to avoid double-reporting)_
- [x] `marketplace/version-semver`: split — missing root `version` → info;
      present-but-not-semver → error. _(implemented as two rules — the
      engine assigns one severity per rule (`finalizeDiagnostic`
      overrides per-diagnostic values), so the info half is the new
      `marketplace/version-missing` (style/info) and `version-semver`
      keeps error for declared-but-malformed versions)_
- [x] `marketplace/author-required`: align with documented required
      `owner{name}` (per OQ5 decision). _(became two rules for the same
      one-severity-per-rule reason as the version split:
      `marketplace/owner-required` (schema/warning, satisfied by owner
      or legacy author) + `marketplace/author-legacy` (style/info
      rename hint when only the legacy string is present))_
- [x] Deprecation diagnostic for the legacy `servers` key in `.mcp.json`
      (synthesized like `schema/parse` or a dedicated rule — pick during
      build) (per OQ1 decision). _(dedicated rule
      `mcp/legacy-servers-key` (schema/info) — simpler than engine
      synthesis since `LegacyServersKey` is already parsed; identical
      per-server diagnostics collapse to one per file via the engine's
      exact-duplicate dedupe)_
- [x] `hooks/timeout-present`: reword message + rules.md entry around the
      documented 600 s default (severity per OQ8 decision). _(stays
      warning; message cites the per-type documented default — 600 s
      command/http/mcp_tool, 30 s prompt, 60 s agent — and frames the
      nudge as fail-faster-in-CI, not hang prevention)_
- [x] `hooks/no-unsafe-shell`: skip exec-form entries (`args` present) and
      non-`command` hook types. _(gates on `EffectiveType() == "command"
      && !ExecForm`; rules.md entry corrected — it described eval/unquoted-
      var smells the rule never actually checked)_
- [x] `schema/frontmatter-required`: reword skill diagnostics to
      best-practice phrasing without changing behavior (per OQ2 decision).
      _(skill messages name the runtime fallback — directory-name for
      `name`, invocation-matching for `description`; command/agent keep
      the plain form; stricter-than-spec blockquote added to rules.md)_
- [x] `skills/no-version-field` + `plugin/manifest-fields`: help-text
      updates citing the current docs (stricter-than-spec note for plugin
      `version`, per OQ3 decision). _(no-version-field cites the
      accepted-but-ignored doc confirmation; manifest-fields' version
      message names the git-SHA fallback; rules.md gains the
      stricter-by-design blockquote and drops its wrong claim that
      `description` was checked)_
- [x] Ruleset version bump + fingerprint ack; `docs/rules/rules.md` +
      README rule anchors updated for every touched rule. _(v1.3.0 landed
      with the first fingerprint-flipping change; expected_fingerprint.txt
      re-acked per drift. Catalog is 34 rules; verified every registered
      ID appears in both the rules.md and README tables; README prose
      sweep mirrored all reworded entries and both headers now say v1.3)_
- [x] Update DESIGN-0002 §2.2 (`servers` vs `mcpServers`) to record the
      resolution; cross-link INV-0006. _(resolution blockquote in the
      MCP artifact section, linking INV-0006 and noting the v1.3.0
      transport-field additions; CLAUDE.md's locked-decision bullet
      updated from "servers{} — revisit if docs standardize" to the
      adopted mcpServers{} + legacy-tag state)_
- [x] Dogfood: run against a `donaldgifford/claude-skills` checkout and
      the claudelint repo itself (`just self-check`); triage every
      new/changed diagnostic. _(both clean on 2026-07-10: self-check
      0 diagnostics / 3 files; claude-skills@b2d72f1 0 diagnostics /
      149 files — nothing to triage. `claudelint version` reports
      ruleset v1.3.0 (6900b22e), catalog 34 rules)_

#### Success Criteria

- The Phase 1 doc-valid fixtures lint with zero false positives
  (previously: `mcpServers` files invisible, object sources erroring,
  comma-form `allowed-tools` erroring, new hook events erroring).
- Regression fixtures prove old invalid inputs still flag (unknown tool,
  unknown event, missing stdio command, empty source).
- `claudelint rules --json` reflects the new catalog; fingerprint
  guardrail test updated deliberately.
- Dogfood passes clean or with only agreed-expected diagnostics.
- `just ci` green; release ships as a minor via the label-driven flow.

---

### Phase 3 — DESIGN-0005 for the agent package and opt-in mechanism

Docs-only gate before the agent code. Small by design — INV-0006 already
contains the research; the DESIGN settles the contested mechanics.

#### Tasks

- [x] `docz create design` → DESIGN-0005 "Agent rules and opt-in rule
      mechanism".
- [x] Specify the extended `Agent` artifact model (which of the 16
      documented fields are parsed, their types/ranges; which are noted
      but unparsed). _(all 16 modeled: 12 new typed fields,
      `mcpServers`/`hooks` as presence bools, ranges via the existing
      `Frontmatter.KeyRange`; field/enum set re-verified against the
      sub-agents reference 2026-07-10)_
- [x] Specify each Phase 4 rule: id, category, severity, options,
      diagnostics, range targets, and the shared model-value validator
      reused by skill/command `model` (values: `sonnet`/`opus`/`haiku`/
      `fable`/`inherit`/full-ID shape). _(DESIGN-0005 §4 table + §3
      `IsValidModelRef`)_
- [x] Resolve the opt-in mechanism (OQ4) and write the chosen pattern up
      as the house convention, superseding the CLAUDE.md
      `mcp/server-allowlist` note. _(DESIGN-0005 §5 — `rules.OptIn`
      interface + `HasRuleBlock` gate; CLAUDE.md bullet replaced)_
- [x] Define how plugin-distributed agents are detected (path heuristic:
      agent file under a root containing `.claude-plugin/plugin.json`)
      for `agents/plugin-ignored-fields`. _(DESIGN-0005 §2 — discovery
      sets `Agent.PluginDistributed`, `IndexSkillCompanions` precedent)_
- [x] `docz update`; PR with `dont-release` label. _(indexes + mkdocs nav
      regenerated. No separate PR: per the work-loop instruction all
      IMPL-0004 phases proceed sequentially on
      `feat/impl-0004-ruleset-alignment`, so DESIGN-0005 rides this
      branch's PR instead of a standalone dont-release one)_

#### Success Criteria

- DESIGN-0005 status Approved (Donald sign-off on the OQ4 resolution in
  particular).
- CLAUDE.md opt-in-rules note updated to point at the new convention.

---

### Phase 4 — Agent rule package

First agent-specific rules. One PR, minor release, fingerprint bump.

#### Tasks

- [x] Extend `ParseAgent` per DESIGN-0005: `model`, `disallowedTools`,
      `permissionMode`, `maxTurns`, `skills`, `mcpServers` (presence),
      `hooks` (presence), `memory`, `background`, `effort`, `isolation`,
      `color` — with key ranges. _(plus `initialPrompt` for the full
      16-field set; new `has`/`asInt64` markdownDoc helpers; ranges via
      the existing `Frontmatter.KeyRange`; `PluginDistributed` field
      added for discovery to set)_
- [x] New package `internal/rules/agents/` with blank-import registration
      in `internal/rules/all/`.
- [x] `agents/model-valid` (schema/warning): shared validator; also
      registered for skill + command `model` fields.
      _(`artifact.IsValidModelRef` + `KnownModelAliases`; agent enum
      sets added alongside for field-enums; message names the full
      valid-value list per the Phase 4 success criterion)_
- [x] `agents/name-format` (schema/warning): lowercase letters and hyphens
      only, per the documented constraint. _(`^[a-z]+(-[a-z]+)*$` — also
      rejects leading/trailing/double hyphens; empty names left to
      `schema/frontmatter-required`)_
- [x] `agents/tools-known` (schema/warning): shared splitter + classifier
      over `tools` and `disallowedTools`; diagnostic explains Claude Code
      silently ignores unknown names. _(splitting happens at parse time;
      the rule reuses `IsKnownTool` + `IsToolPattern` like
      `commands/allowed-tools-known`)_
- [x] `agents/plugin-ignored-fields` (content/warning): plugin-rooted
      agents declaring `mcpServers`/`hooks`/`permissionMode` (documented
      as ignored for plugin subagents). _(detection via
      `artifact.MarkPluginDistributed` — bounded ancestor walk for
      `.claude-plugin/plugin.json`, called from the CLI's parse wiring
      with the scan root so the walk can't escape the repo; one
      diagnostic per declared key at its `KeyRange`)_
- [x] `agents/field-enums` (schema/warning): `permissionMode`, `effort`,
      `color`, `isolation`, `memory` enum membership. _(uses the
      `Agent*` enum sets from knowndata.go; isolation is a one-entry
      set (`worktree`); messages hardcode the documented value order
      since map iteration is unstable)_
- [x] Opt-in mechanism (DESIGN-0005 §5, scheduled here by its rollout
      plan): `rules.OptIn` interface + `rules.IsOptIn`; engine skips
      opt-in rules without a `rule "<id>"` block
      (`Config.HasRuleBlock`); fingerprint gains `optin=` component;
      `claudelint rules` shows `(opt-in)` and `rules --json` gains
      `opt_in`; `mcp/server-allowlist` migrated (no block → skipped;
      block without `allowlist` → loud config error preserved);
      rules-json-schema.md + rule prose updated.
- [ ] Fixtures: valid agent exercising all documented fields; invalid
      variants per rule; a plugin-rooted agent fixture.
- [ ] Ruleset version bump + fingerprint ack; rules.md + README anchors;
      `claudelint rules <id>` detail entries.
- [ ] Coverage: new `internal/rules/agents` package must clear the 55%
      floor (plan tests before code).
- [ ] Dogfood pass (claude-skills ships plugin agents — prime corpus).

#### Success Criteria

- All five rules fire on their invalid fixtures with correct ranges and
  pass their valid fixtures.
- A typo'd `model: sonet` produces a diagnostic naming the valid values.
- `agents/plugin-ignored-fields` stays silent on project-level
  (`.claude/agents/`) definitions declaring the same fields.
- Dogfood run on claude-skills triaged; no untriaged false positives.
- `just ci` green; minor release ships.

---

### Phase 5 — Policy and remaining validation rules

The governance rule plus the remaining validation batch from INV-0006's
proposed tables. One PR, minor release, fingerprint bump.

#### Tasks

- [ ] `agents/model-policy` (error, opt-in per the OQ4/DESIGN-0005
      mechanism): `require = "inherit"` (absent key compliant) or
      `allowlist = [...]` option shapes.
- [ ] `hooks/type-known` (schema/error): `type` in the five documented
      values; absent = `command`.
- [ ] `hooks/type-fields` (schema/error): per-type required fields
      (`command` needs `command`; `http` needs `url`; `mcp_tool` needs
      `server` + `tool`; `prompt`/`agent` need `prompt`).
- [ ] `mcp/transport-known` (schema/warning): `type` in
      `stdio`/`http`/`sse`/`ws`; `sse` additionally flagged as
      documented-deprecated (info).
- [ ] `mcp/no-secrets-in-headers` (security/error): reuse the
      `no-secrets-in-env` detector over `headers` values (or fold into
      that rule — pick during build, note in rules.md either way).
- [ ] `mcp/timeout-minimum` (schema/warning): `timeout` below 1000 flagged
      with a seconds-vs-milliseconds hint.
- [ ] `marketplace/reserved-name` (schema/error): the 16 documented
      reserved names, exact match; impersonation heuristics deliberately
      NOT attempted (enforced server-side by claude.ai).
- [ ] `marketplace/name-format` (style/warning): kebab-case for
      marketplace name + plugin entry names.
- [ ] `marketplace/source-path-safety` (security/error): relative sources
      start with `./`; no `..` segments.
- [ ] `marketplace/renames-valid` (schema/error): chains terminate at
      `null` or a listed plugin; no cycles.
- [ ] `skills/description-length` (content/warning): `description` +
      `when_to_use` combined at most 1,536 chars.
- [ ] `skills/fork-agent-pairing` (schema/warning): `agent:` without
      `context: fork`.
- [ ] `claude_md/import-exists` (content/warning): `@path` imports resolve
      relative to the file (respecting `~`); flag chains beyond 4 hops;
      skip code spans/fences per documented parser behavior.
- [ ] Fixtures for every rule (valid + invalid pairs).
- [ ] Ruleset version bump + fingerprint ack; rules.md + README anchors.
- [ ] Dogfood pass; add "implemented by IMPL-0004" note to INV-0006 and
      flip this doc to Completed.

#### Success Criteria

- Every new rule has valid/invalid fixture coverage and correct ranges
  (no file-level `(0,0)` ranges — per-line suppression markers must
  work).
- `agents/model-policy` unset behaves exactly per the DESIGN-0005
  mechanism (no noise on unconfigured repos; enforcement when
  configured).
- Full catalog: `claudelint rules --json` lists the expanded ruleset
  (~49 rules) and `docs/rules-json-schema.md` still validates.
- Dogfood clean; `just ci` green; minor release ships.
- IMPL-0004 → Completed; INV-0006 cross-referenced as implemented.

---

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/artifact/parse_mcp.go` | Modify | `mcpServers` key, transport fields, deprecation tagging |
| `internal/artifact/parse_marketplace.go` | Modify | Object sources, `owner{}`, `renames{}` |
| `internal/artifact/parse_json.go` | Modify | Hook `type` + per-type fields, exec-form detection |
| `internal/artifact/parse_md_kinds.go` | Modify | Agent field extension; skill/command new fields |
| `internal/artifact/parse_markdown.go` | Modify | Shared comma/space tool-list splitter |
| `internal/artifact/knowndata.go` | Modify | `KnownTools` refresh, `KnownHookEvents` expansion, model-value + tool-pattern classifiers |
| `internal/artifact/types.go` | Modify | New fields + ranges on `MCPServer`, `MarketplacePlugin`, `HookEntry`, `Agent`, `Skill`, `Command` |
| `internal/rules/agents/` | Create | Six-rule agent package (Phases 4–5) |
| `internal/rules/{hooks,mcp,marketplace,skills,claudemd,commands}/` | Modify/Create | Rule fixes + new rules per phase tables |
| `internal/rules/all/all.go` | Modify | Register the `agents` package |
| `internal/rules/` version/fingerprint source | Modify | Ruleset version + fingerprint per code phase |
| `docs/rules/rules.md`, `README.md` | Modify | Catalog + anchors per phase |
| `docs/design/0005-*.md` | Create | Phase 3 output |
| `docs/design/0002-*.md` | Modify | §2.2 resolution note |
| `testdata/**` | Create | Doc-valid + invalid fixtures per phase |

## Testing Plan

- **Unit (per rule/parser)**: table-driven tests; every new parser field
  asserted with value + range; every rule asserted on valid + invalid
  fixtures.
- **Fixture integration**: the doc-valid corpus from Phase 1 runs through
  `claudelint run` with zero unexpected diagnostics; retained as
  regression fixtures for later phases.
- **Fingerprint guardrail**: deliberate acknowledgment in each code
  phase; CI fails on unacknowledged drift.
- **Coverage gate**: `just coverage-gate` — 55% floor per `internal/`
  package, including the new `internal/rules/agents`.
- **Dogfood**: `just self-check` plus a `donaldgifford/claude-skills`
  checkout after Phases 2, 4, and 5 (`cd` into the fixture repo — config
  discovery walks up from CWD).
- **Docs**: `just lint-md` + `just docs-check` for rules.md/README edits.

## Dependencies

- INV-0006 (merged) — findings and rule tables this implements.
- DESIGN-0005 (Phase 3 output) — gates Phases 4–5.
- DESIGN-0001/0002 — existing architecture constraints (rule shape,
  range-emission conventions, rules never import the engine).
- Current docs snapshot: `code.claude.com/docs/en/*` fetched 2026-07-09;
  re-verify the event list and enum values at Phase 1 build time in case
  the docs have moved again.

## Resolved Decisions

All Open Questions resolved 2026-07-10 — every one on option (a), the
recommendation. Task references like "(per OQ1 decision)" point here.

- **OQ1 — `.mcp.json` `servers` key**: accept both keys; `mcpServers`
  wins when both present; info diagnostic on `servers` ("deprecated key;
  rename to mcpServers"); drop `servers` support at the next major
  ruleset rev.
- **OQ2 — skill `name`/`description` requiredness**: keep both required
  at current severity; reword diagnostics to best-practice phrasing
  ("skills should declare X — Claude Code falls back to Y"); document
  the stricter-than-spec stance in rules.md.
- **OQ3 — plugin `version` requirement**: keep requiring `version` at
  error; document as stricter-by-design (pinned versions make update
  semantics explicit for marketplace review).
- **OQ4 — opt-in mechanism**: engine-level config-driven enable. A rule
  may declare itself opt-in; the engine runs it only when
  `.claudelint.hcl` has a `rule` block for it; no block → fully skipped
  and shown as "opt-in, disabled" in `claudelint rules`.
  `mcp/server-allowlist` migrates to the same mechanism (its
  loud-error-when-unconfigured behavior preserved for explicit enables
  without an allowlist). Details specified in DESIGN-0005 (Phase 3);
  CLAUDE.md convention note updated there.
- **OQ5 — marketplace `owner` vs `author`**: repurpose
  `marketplace/author-required` into `marketplace/owner-required`
  checking root `owner.name` at warning severity (rule-ID rename rides
  the fingerprint bump); legacy `author` satisfies the check with an
  info-level "rename to owner" hint.
- **OQ6 — release cadence**: each code PR ships as its own minor release
  via the label-driven flow; ruleset version bumps minor each time.
- **OQ7 — `allowed-tools` rule placement**: extend
  `commands/allowed-tools-known` `AppliesTo` to `command, skill`; keep
  the existing rule ID; rules.md notes it covers both kinds.
- **OQ8 — `hooks/timeout-present`**: keep at warning with a rewritten
  message ("no explicit timeout; Claude Code defaults to 600 s —
  declare one to fail faster in CI").

## References

- [INV-0006](../investigation/0006-rule-coverage-audit-against-current-claude-code-docs.md)
  — audit tables this implements, including per-rule verdicts and doc
  URLs
- [DESIGN-0001](../design/0001-claudelint-linter-architecture-and-rule-engine.md)
  — rule interface, range conventions, registration
- [DESIGN-0002](../design/0002-phase-2-marketplaces-mcp-rules-and-github-action.md)
  — §2.2 `.mcp.json` key decision being revised
- DESIGN-0005 — Phase 3 output (agent package + opt-in mechanism)
- [INV-0005](../investigation/0005-phase-2-dogfood-findings-marketplaces-mcp-and-spec-divergence.md)
  — dogfood-pass precedent
- Claude Code docs: sub-agents, skills, slash-commands, hooks,
  hooks-guide, mcp, plugins-reference, plugin-marketplaces, memory
  (`code.claude.com/docs/en/*`, fetched 2026-07-09)
