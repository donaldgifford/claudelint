---
id: IMPL-0005
title: "Phase 5 — Upstream spec drift detection"
status: In Progress
author: Donald Gifford
created: 2026-09-13
---

<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL-0005: Phase 5 — Upstream spec drift detection

**Status:** In Progress
**Author:** Donald Gifford
**Date:** 2026-09-13

<!--toc:start-->
- [Objective](#objective)
- [Scope](#scope)
  - [In Scope](#in-scope)
  - [Out of Scope](#out-of-scope)
- [Implementation Phases](#implementation-phases)
  - [Phase 1 — Tool, digest, and workflow](#phase-1--tool-digest-and-workflow)
    - [Tasks](#tasks)
    - [Success Criteria](#success-criteria)
  - [Phase 2 — Guardrail and catch-up](#phase-2--guardrail-and-catch-up)
    - [Tasks](#tasks-1)
    - [Success Criteria](#success-criteria-1)
    - [Dogfood results (2026-09-14)](#dogfood-results-2026-09-14)
  - [Phase 3 — Runtime validator and rendered page](#phase-3--runtime-validator-and-rendered-page)
    - [Tasks](#tasks-2)
    - [Success Criteria](#success-criteria-2)
- [File Changes](#file-changes)
- [Testing Plan](#testing-plan)
- [Dependencies](#dependencies)
- [Resolved Decisions](#resolved-decisions)
  - [Amendments made while implementing](#amendments-made-while-implementing)
- [Open Questions](#open-questions)
- [References](#references)
<!--toc:end-->

## Objective

Build the `specdrift` tool, its weekly workflow, and the guardrail test
specified in
[DESIGN-0006](../design/0006-upstream-spec-drift-detection.md), so that
changes to the Claude Code artifact specs (skill, agent, and command
frontmatter; plugin and marketplace manifests; hooks; MCP; the built-in
tool list) surface within a week as a tracking issue and, once the code
is tied to the digest, as a failing test that names the Go identifier to
update. Along the way, close the drift that has accumulated since
INV-0006: three missing hook events, a 45-versus-18 tool list, two
undocumented marketplace source kinds, and one missing reserved name.

**Implements:** DESIGN-0006 (all twelve design questions resolved
2026-09-12, every one on option (a); this doc's task references such as
"per DESIGN OQ3" point at that document's Resolved Decisions).

## Scope

### In Scope

- `internal/upstream` library: source table, fetcher, GFM table scanner,
  one extractor per digest section, deterministic digest and lock
  emitters, structural diff with text / JSON / Markdown renderers.
- `cmd/specdrift` thin `main` with `pull`, `digest`, `diff`, `check`,
  `render`, and `validate-fixtures` subcommands.
- The committed `internal/upstream/digest.json`, `sources.lock.json`,
  `acknowledged.json`, and `runtime_fixtures.json`.
- `.github/workflows/spec-drift.yml`, `scripts/spec-drift-issue.sh`, the
  `spec-drift` label, and the `spec-check` / `spec-sync` /
  `spec-validate-fixtures` recipes.
- The guardrail test and the exported parser key lists it needs.
- Catch-up fixes the guardrail demands on day one: `KnownHookEvents`,
  `KnownTools`, marketplace `archive` and `command` source kinds, the
  `claude-tag-plugins` reserved name, a deprecated-and-removed tools
  table backed by `artifact.DeprecatedTools`, and acknowledgements for
  every documented field no rule consumes.
- The rendered `docs/rules/upstream-spec.md` page, the `claudelint
  version` spec line, and the `rules --json` `upstream_version` field.
- Ruleset version bump, rules-doc and README updates, one dogfood pass
  per code PR, CLAUDE.md updates.

### Out of Scope

- New lint rules for newly documented fields (for example a
  `skills/portable-fields` rule over the six-field allowlist). The digest
  makes them cheap to write later; they get their own IMPL.
- Parsing documented fields that no rule consumes (see OQ4). Presence in
  the digest plus an acknowledgement is the deliverable.
- Generating `knowndata.go` from the digest (DESIGN OQ7 chose guard-only).
- Deep extraction for experimental surfaces (monitors, themes, channels,
  LSP): field names only.
- Committing raw upstream pages in any form (DESIGN OQ6).
- Changes to `just ci`, `just test`, or `ci.yml` beyond the one
  `render --check` step in Phase 3. Everything that touches the network
  lives in `spec-drift.yml` and the `spec-*` recipes.

---

## Implementation Phases

Three PRs, one per phase, in the order DESIGN OQ10 fixed. Phase 1 changes
no shipped binary; Phases 2 and 3 do and ship as minor releases through
the label-driven flow. Every code phase follows the house checklist:
table-driven tests, `just check` green, 55% coverage floor per
`internal/` package, `just lint-md` and `just docs-check` for any docs
touched, fingerprint guardrail acknowledged if it moves, and no bot
commits from the workflow.

---

### Phase 1 — Tool, digest, and workflow

Land the library, the CLI, the first committed digest, and the weekly
workflow that files the tracking issue. After this phase upstream changes
are visible within a week; nothing in the linter changes yet.

Branch `chore/spec-drift-tool`; label per OQ1.

#### Tasks

- [x] `internal/upstream/source.go`: the source table from DESIGN §1 as a
      sorted slice of `Source{ID, Tier, URL, Ext}` (fifteen entries,
      Tiers A–D — this doc previously said sixteen; the DESIGN §1 table
      is the source of truth and lists fifteen), plus the two
      `code.claude.com/schemas/*.json` probes, which are fetched on every
      run but never fail one. They are recorded in the work-directory
      manifest with their HTTP status and kept out of the committed lock
      (see Amendments).
- [x] `internal/upstream/fetch.go`: `Fetch(ctx, sources, workDir)` using
      `http.NewRequestWithContext` (the `noctx` and `bodyclose` linters
      are on), three retries with backoff on 5xx and transport errors,
      a 30-second per-source timeout, a `User-Agent` naming the repo,
      redirects followed. Writes `<id>.<ext>` and `manifest.json`
      (`url`, `status`, `sha256`, `etag`, `last_modified`, `bytes`) into
      the work directory. Any Tier A–D source failing after retries is a
      hard error (exit 2).
- [x] `internal/upstream/table.go`: line-oriented GFM table scanner
      (header row, delimiter row, body rows split on unescaped `|`,
      backticks stripped from the first cell) and section scoping: from
      an anchor heading to the next heading of equal or higher level. No
      Markdown dependency added.
- [x] `internal/upstream/digest.go`: the `Digest`, `Field`, and section
      types from DESIGN "Data Model"; a `Marshal` that emits sorted keys,
      sorted arrays, two-space indent, LF, trailing newline;
      `digest_version` pinned at 1.
- [x] Extractors, one file per source, each declaring source id, anchor
      regex, columns, sanity floor (per OQ8), and implementing the
      `Extractor` interface with hard failure on a missing anchor,
      unexpected headers, or an under-floor row count (DESIGN OQ11):
  - [x] `extract_skills.go` — `skills.frontmatter` (`### Frontmatter
        reference`), `skills.portable_fields` (`#### Using skill
        frontmatter outside Claude Code`), `skills.substitutions`
        (`#### Available string substitutions`).
  - [x] `extract_agents.go` — `agents.frontmatter` (first table under
        `### Write subagent files` whose first row is `name`) and
        `agents.enums` from the backticked tokens in the `model`,
        `permissionMode`, `effort`, `memory`, `isolation`, and `color`
        description cells.
  - [x] `extract_plugins.go` — `plugins.manifest_fields` (the three
        tables under `## Plugin manifest schema`), `plugins.locations`
        (`### File locations reference`), `plugins.substitution_fields`
        (`### Environment variables`).
  - [x] `extract_marketplaces.go` — `marketplace.fields`,
        `marketplace.owner_fields`, `marketplace.plugin_entry_fields`,
        `marketplace.sources` (one table per `###` under `## Plugin
        sources`; `archive` and `command` included), and
        `marketplace.reserved_names` from the `**Reserved names**`
        paragraph.
  - [x] `extract_hooks.go` — `hooks.events` (every `###` under `## Hook
        events`), `hooks.types`, `hooks.handler_fields` (per-type tables
        under `### Hook handler fields`), `hooks.timeout_defaults`
        (numbers parsed from the `timeout` row: by type and by event
        override).
  - [x] `extract_mcp.go` — `mcp.transports` from the `mcp.md` `type`
        enumeration; `mcp.server_fields` from the SchemaStore
        plugin-manifest `mcpServers` definition (DESIGN OQ5),
        cross-checked against the `plugins-reference.md` MCP section.
  - [x] `extract_tools.go` — `tools.builtin` from the first table after
        the `tools-reference.md` H1; each row also carries
        `deprecated: true` when its description cell begins with
        "Deprecated" (the docs' own signal, consumed by the Phase 2
        `DeprecatedTools` guardrail row).
  - [x] `extract_schemastore.go` — `properties` keys and the hook-event,
        hook-type, MCP-type, source-kind, and `defaultMode` enums from
        the three schemas via a small JSON-pointer walk.
  - [x] `extract_portable.go` — `portable.*` from the agentskills spec
        table, the `MAX_*` constants and `ALLOWED_FIELDS` in
        `validator.py`, and the allowlist, name regex, and description
        constraints in `quick_validate.py`.
  - [x] `extract_changelog.go` — `meta.claude_code_latest` from the first
        `## X.Y.Z` heading; `meta.docs_max_marker` as the highest
        `v2.N.N` marker across the docs pages (also written per page
        into the lock as `version_marker`).
- [x] `disagreements.go`: compute docs-versus-SchemaStore set differences
      for hook events, hook types, MCP transports, marketplace source
      kinds, and plugin manifest properties; write them under
      `digest.disagreements` (informational per DESIGN OQ9).
- [x] `lock.go`: `sources.lock.json` with `url`, `sha256`,
      `last_modified`, `version_marker` per source. No timestamps.
- [x] `diff.go`: structural diff per DESIGN §5 (`Change{Path, Kind,
      Item, Old, New}`), with `meta.*`, `schemastore.*`, and
      `disagreements` partitioned into an informational block that never
      affects the exit code; renderers for `text`, `json`, and
      `markdown` (Markdown groups by section, one line per change, and
      appends "sources changed without affecting the digest" from the
      lock diff).
- [x] `internal/upstream/command.go`: `NewRootCommand()` (cobra, already
      a dependency) wiring `pull`, `digest`, `diff`, and `check` with the
      flags from DESIGN §2; `check --work DIR` keeps the fetched pages for
      artifact upload; `check --update` rewrites the committed digest and
      lock; exit codes 0 / 1 / 2. `cmd/specdrift/main.go` only calls it,
      mirroring `cmd/claudelint`.
- [x] Golden snippet fixtures under `internal/upstream/testdata/snippets/`
      produced per OQ6, one per extractor, plus one SchemaStore excerpt
      per schema and the three Tier C files.
- [x] Tests: table scanner (escaped pipes, alignment rows, trailing
      whitespace, nested backticks, table ending at a heading versus a
      blank line); every extractor against its snippet with the exact
      expected set; negatives for missing anchor, wrong headers, and
      under-floor count; digest determinism (twice → identical bytes);
      diff goldens for add / remove / change / no-op / informational-only;
      fetch against `httptest.Server` covering retry, timeout, redirect,
      and manifest fields; `check` end-to-end against an `httptest`-served
      fixture set asserting each exit code. No test touches the network.
- [x] Run `go run ./cmd/specdrift check --update` on the branch and commit
      the first `digest.json` and `sources.lock.json`; run it twice and
      assert a clean `git diff`.
- [x] `justfile`: `spec-check` and `spec-sync` under `[group('spec')]`,
      documented as network recipes; neither joins `ci` or `check`.
- [x] `.github/workflows/spec-drift.yml` per DESIGN §6: weekly Monday
      06:00 UTC cron, `workflow_dispatch`, `pull_request` on
      `internal/upstream/**`, `cmd/specdrift/**`, and the workflow file;
      `contents: read` at the top, `issues: write` on the job; steps for
      check, artifact upload (90 days), step summary, PR warning, issue
      script, and fail-on-exit-2.
- [x] `scripts/spec-drift-issue.sh`: ensure the `spec-drift` label,
      locate the single open issue, read the `<!-- specdrift:diff-sha256
      -->` marker, and create / comment / close per DESIGN OQ3; a
      `--dry-run` flag prints the `gh` commands instead of running them.
      Add `spec-drift` to `LABEL_COLORS` and `LABEL_DESCRIPTIONS` in
      `scripts/labels.sh` and create it with that script.
- [x] `just lint-actions` and `just lint-config` clean on the new
      workflow.
- [ ] **deferred - human required** — Validate on the branch before
      merge: a `workflow_dispatch` run against the committed digest
      reports no drift and creates no issue; a scratch commit that
      removes `PreModelSwitch` from the committed digest makes the next
      dispatch open an issue whose body lists
      `hooks.events added: PreModelSwitch`; reverting the scratch commit
      makes the following run close it. Delete the scratch issue
      afterwards.

      GitHub only offers `workflow_dispatch` for workflows that exist on
      the default branch, so this sequence cannot run until the PR
      merges. What can be checked before merge has been: the
      pull-request trigger exercises the check step against live pages
      on this PR; `scripts/spec-drift-issue.sh --dry-run` was run
      against a real clean report (no-op) and a real drift report
      (creates the issue, body carries the marker and lists
      `hooks.events added: PreModelSwitch`); `actionlint` and
      `shellcheck` are clean; and the `spec-drift` label was created
      with `scripts/labels.sh`. Run the three dispatches immediately
      after merge.
- [x] CLAUDE.md: add `just spec-check` / `just spec-sync` to Common
      commands and a "Upstream spec drift" bullet under Git / PR
      conventions describing the workflow, the label, and the rule that
      the digest is regenerated with `just spec-sync` in the same PR as
      any code that catches up.
- [x] Coverage: `internal/upstream` clears the 55% floor
      (`just coverage-gate`).
- [x] PR opened with the label from OQ1; `just ci` green.
      ([#59](https://github.com/donaldgifford/claudelint/pull/59))

#### Success Criteria

- `go run ./cmd/specdrift check` from a clean checkout on merge day exits
  0 and prints "no drift"; running `digest` twice yields identical bytes.
- Removing one hook event from the committed digest makes `check` exit 1
  and the Markdown report name `hooks.events` and the event.
- A fixture with its anchor heading removed makes `digest` exit 2, name
  the source and anchor, and write no digest.
- The three-run dispatch sequence (clean → staled → clean) produces
  exactly one issue, one comment, and one close.
- `just test` passes offline; `just ci` is unchanged and green;
  `just lint-actions` is clean.
- `internal/upstream` coverage is at or above 55%.

---

### Phase 2 — Guardrail and catch-up

Tie the digest to the code. Export the parser key lists, add the
guardrail test and the acknowledgement file, then make every fix the
guardrail demands on day one. This is the phase that converts the weekly
issue into a failing test.

Branch `fix/upstream-drift-2026-09`; label `minor` (the ruleset changes;
version per OQ3).

**Shipped differently.** Phases 1 and 2 landed on one branch,
`feat/impl-0005-spec-drift-tool`, under PR #59, because Phase 1 was
still open when Phase 2 started. The label moved from `dont-release` to
`minor` when the ruleset bump landed, which is the part that matters —
the branch name is cosmetic, the label drives the release.

#### Tasks

- [x] `internal/artifact`: export `SkillFrontmatterKeys`,
      `CommandFrontmatterKeys`, `AgentFrontmatterKeys`,
      `PluginManifestKeys`, and `HookEntryKeys` as sorted string slices;
      `ParseSkill`, `ParseCommand`, `ParseAgent`, `ParsePlugin`, and
      `ParseHook` read keys through them instead of string literals so
      the lists cannot drift from the parsers. No behaviour change;
      existing parser tests stay green. (`HookEntryKeys` is one row
      beyond the DESIGN §7 table; note it there.)
- [x] `internal/upstream/acknowledged.json` and a loader that rejects
      unknown digest paths and empty reasons; `LoadEmbedded()` for the
      digest and the acknowledgements via `go:embed`.
- [x] `internal/upstream/guard_test.go`: one subtest per DESIGN §7 row —
      hook events, hook types, tools, model aliases, the four agent enum
      sets, the five key lists against their digest sections, marketplace
      source kinds against `MarketplaceSourceKind`, reserved names
      against `reservedMarketplaceNames` (exported for the test as
      `ReservedMarketplaceNames`), MCP transports against the set
      `mcp/transport-known` uses, and timeout defaults against the values
      `hooks/timeout-present` cites. Failure messages follow the
      fingerprint test: identifier, digest path, delta, and the two ways
      to resolve. A stale acknowledgement (item no longer upstream) fails.
- [x] `KnownHookEvents`: add `DirectoryAdded`, `PreModelSwitch`,
      `PostModelSwitch`; update the count in the `knowndata.go` comment,
      the lifecycle table under `hooks/event-name-known` in
      `docs/rules/rules.md`, and the "30 events" sentence in `README.md`.
- [x] `KnownTools`: add the 31 documented tools missing today; per OQ2
      keep `Task` with an acknowledgement citing the v2.1.63 rename and
      drop `BashOutput`, `KillShell`, and `MultiEdit`; update the
      `agents/tools-known` and `commands/allowed-tools-known` prose in
      `rules.md` and README where they enumerate tools.
- [x] `artifact.DeprecatedTools` (OQ2 addendum): a map of tool name to
      `{Status, Since, ReplacedBy, Source}` with status `renamed`,
      `removed`, or `deprecated`. Seed it from the Claude Code changelog
      where an entry exists and from the first docs marker at which the
      tool left the table where none does: `Task` renamed to `Agent`
      (v2.1.63, INV-0006); `BashOutput` removed, use `TaskOutput`
      (v2.0.64, changelog "Unshipped ... BashOutputTool"); `KillShell`
      removed, use `TaskStop`, and `MultiEdit` removed, use `Edit` (no
      changelog entry; record the docs marker and say so); `TaskOutput`
      deprecated, use `Read` (v2.1.83, changelog). The two tools-known
      rules consult it before `KnownTools`, so a removed or renamed tool
      gets "removed in v2.0.64; use TaskOutput" instead of "unknown
      tool"; deprecated-but-documented tools stay known and produce no
      diagnostic. `docs/rules/rules.md` gains a "Deprecated and removed
      tools" table under `agents/tools-known` (tool, status, version,
      replacement, source), linked from `commands/allowed-tools-known`.
      Guardrail row: every `removed` or `renamed` entry must be absent
      from `tools.builtin` and every `deprecated` entry present with the
      `deprecated` flag; a mismatch means the table is stale.
- [x] Marketplace sources (drift found while writing this doc): add
      `SourceArchive` (`{"source": "archive", "url", "sha256"}`) and
      `SourceCommand` (`{"source": "command", "command", "timeout",
      "mode"}`) to `MarketplaceSourceKind` and the parser;
      `marketplace/plugin-source-valid` validates `url` for archive
      (HTTPS only, `sha256` as 64 hex when present) and `command` for
      command sources; `marketplace/external-source-skipped` gains
      kind-aware wording for both; rules.md per-type table extended.
      Fixture `ok/marketplaces/archive_command/`.
- [x] `reservedMarketplaceNames`: add `claude-tag-plugins` (17 documented
      today); update the count in the comment and in rules.md.
- [x] Acknowledgements for everything documented that no rule consumes
      (per OQ4): the ten unparsed skill fields, the nineteen unparsed
      plugin manifest fields, agent `experimental`, the hook handler
      fields outside `HookEntryKeys` (`if`, `statusMessage`, `once`,
      `asyncRewake`, `headers`, `allowedEnvVars`, `input`, `model`), the
      MCP `oauth.*` subfields, marketplace `allowCrossMarketplaceDependenciesOn`,
      `renames` is parsed already, plugin-entry `headers` /
      `headersHelper` / `relevance`, and the per-event timeout overrides
      in `hooks.timeout_defaults.by_event`. Each entry carries a reason
      that names the rule or phase that would consume it.
- [x] Verify the remaining rows come out equal with no acknowledgement:
      model aliases, the four agent enum sets, hook types, MCP
      transports, timeout defaults by type.
- [x] Fixtures: a hooks file using the three new events; an agent and a
      skill declaring newly documented tools (`EnterPlanMode`,
      `SendMessage`, `TaskCreate`) in `tools` / `allowed-tools`; the
      archive-and-command marketplace; a marketplace named
      `claude-tag-plugins` in the reserved-name rule test.
- [x] Ruleset version bump per OQ3; amend the `RulesetVersion` doc
      comment in `internal/rules/version.go` so known-data changes are
      an explicit bump trigger; `docs/rules/rules.md` header and README
      rows updated. The fingerprint should not move (no rule ids,
      severities, options, or `AppliesTo` change); if it does, ack it
      deliberately.
- [x] `just spec-sync` on the branch so the digest and lock reflect
      upstream on the day the fixes land; guardrail green with every
      acknowledgement reasoned.
- [x] Dogfood: `just self-check` and a `donaldgifford/claude-skills`
      checkout (`cd` into it first — config discovery walks up from
      CWD); triage every new or removed diagnostic. Expect the
      `agents/tools-known` and `commands/allowed-tools-known` warning
      count to drop.
- [x] DESIGN-0006 §7 table updated for `HookEntryKeys` and
      `DeprecatedTools`; PR labelled `minor`; `just ci` green.

#### Success Criteria

- `go test ./internal/upstream/...` passes with zero unacknowledged
  items; every acknowledgement has a reason naming a rule or phase.
- Deleting `PreModelSwitch` from `KnownHookEvents` fails the guardrail
  with a message naming `artifact.KnownHookEvents`, `hooks.events`, and
  the missing event; adding a bogus acknowledgement for an item that is
  not upstream also fails.
- Doc-valid fixtures using the 33 events, the 45 tools, archive and
  command sources, and the 17 reserved names lint with zero false
  positives; the existing regression fixtures still flag.
- An agent declaring `tools: BashOutput` gets a warning naming v2.0.64
  and `TaskOutput`; one declaring `TaskOutput` gets none; the rules.md
  deprecated-tools table matches `DeprecatedTools` row for row.
- `claudelint version` shows the bumped ruleset; `just ci` green; minor
  release ships via the label flow.
- Dogfood clean or fully triaged.

#### Dogfood results (2026-09-14)

`just self-check` on this repo: 0 diagnostics over 3 files, unchanged.

`donaldgifford/claude-skills` at `32d17b4`, 158 files, before and after
this branch:

| Rule | Severity | Before | After |
| --- | --- | --- | --- |
| `agents/name-format` | warning | 7 | 7 |
| `commands/allowed-tools-known` | error | 0 | 4 |

The expected drop in tools-known findings did not happen because there
was nothing to drop: claude-skills declared none of the 31
newly-documented tools, so the catch-up removed no false positives
there. It added four true positives instead — four
`infrastructure-as-code` commands declare `Task`, which the tools
reference dropped when it was renamed to `Agent` in v2.1.63.

Worth knowing before this ships: the runtime still accepts `Task`, so
those four commands work today, and `commands/allowed-tools-known` is an
error rule. A repository that is clean on ruleset v1.5.0 and declares
`Task` will fail on v1.6.0. That is the intended reading of OQ2 — the
rename is real and silent — but it is a breaking change for a name that
still functions, and it is the reason this phase ships as a minor bump
with the deprecated-tools table in `rules.md` rather than as a patch.
Downstream fix is a one-word rename; `severity = "warning"` on the rule
is the escape hatch for anyone not ready.

Rules cannot lower the severity of an individual diagnostic: the engine
assigns severity per rule (`runner.go`), so "renamed tool" cannot be a
warning inside an error rule without an engine change. Not attempted
here — it would contradict DESIGN-0001 without an amendment.

---

### Phase 3 — Runtime validator and rendered page

Put the Claude Code runtime's own validator in the loop, give humans a
page to link to, and expose the digest version from the binary.

Branch `chore/spec-drift-runtime`; label `minor` (the `version` and
`rules --json` output change).

#### Tasks

- [x] `internal/upstream/runtime_fixtures.json` with entries
      `{path, kind, expect}` per OQ5, covering at least: the two existing
      `.claude-plugin` fixtures as `pass`, one plugin manifest missing
      `name` as `fail`, one marketplace with duplicate plugin names as
      `fail`, one `agents/` directory as `pass`, and one `skills/`
      directory as the probe that asserts `contents` is still empty.
- [x] `specdrift validate-fixtures --claude <bin>`: runs
      `claude plugin validate --strict --json <path>` per entry with an
      empty `CLAUDE_CONFIG_DIR`, parses the JSON, compares `success` to
      `expect`, records `claude --version`, and reports disagreements
      with the runtime's `errors` and `warnings` verbatim (Markdown and
      JSON). Exit 0 / 1 / 2. Tested with a fake `claude` script on
      `PATH` returning canned JSON.
- [ ] Workflow wiring per OQ10: Node from `mise.toml` via
      `jdx/mise-action`, `npm install -g @anthropic-ai/claude-code@latest`
      (DESIGN OQ8), run `validate-fixtures`, append its report to the
      drift report before the issue script runs. A runtime-step failure
      must not suppress the drift report.
- [x] `just spec-validate-fixtures` recipe (requires a local `claude`).
- [x] `specdrift render --digest FILE --out docs/rules/upstream-spec.md`:
      `title: Upstream spec` frontmatter; one section per artifact kind
      listing documented fields and enums, whether claudelint parses
      each, and the acknowledgement reason where present; a "verified
      against Claude Code vX.Y.Z" line from `meta`; tables emitted in one
      consistent pipe style so `just lint-md` (MD060) passes; the page
      lands in the Starlight `Rules` sidebar group automatically and
      links to the deprecated-and-removed tools table in rules.md rather
      than duplicating it.
- [x] `render --check` (exit 1 when the committed page differs from a
      fresh render) wired per OQ9; commit the first rendered page.
- [ ] `claudelint version` third line per OQ7 and `rules --json`
      `upstream_version` (additive); `docs/rules-json-schema.md` updated;
      `internal/cli` reads `upstream.LoadEmbedded()` (import direction
      `cli → upstream`; `upstream` imports nothing from `cli`, `engine`,
      or `rules`).
- [ ] CLAUDE.md "Project status" paragraph and the `version` output shape
      note in `internal/cli/version.go` updated; DESIGN-0006 gains an
      "Implemented by IMPL-0005" note and moves to Implemented.
- [ ] Dogfood: `just self-check`; confirm `claudelint version` on the
      release binary prints the spec line.
- [ ] Flip this doc to Completed; PR labelled `minor`; `just ci` green.

#### Success Criteria

- On merge day the runtime job reports zero disagreements with the
  latest published CLI, and the `skills/` probe confirms `contents` is
  empty (or the report says the assumption changed).
- `docs/rules/upstream-spec.md` renders in both Starlight and MkDocs,
  passes `just lint-md`, and `render --check` fails when the digest
  changes without a re-render.
- `claudelint version` prints the spec line; `rules --json` carries
  `upstream_version` and still matches `docs/rules-json-schema.md`.
- `just ci` green; minor release ships; DESIGN-0006 Implemented;
  IMPL-0005 Completed.

---

## File Changes

| File | Action | Description |
| --- | --- | --- |
| `internal/upstream/source.go` | Create | Source table (ids, tiers, URLs) and schema probes |
| `internal/upstream/fetch.go` | Create | Context-aware fetcher with retries, timeout, manifest |
| `internal/upstream/table.go` | Create | GFM table scanner and section scoping |
| `internal/upstream/digest.go`, `lock.go`, `diff.go`, `disagreements.go` | Create | Digest types and emitter, lock, structural diff and renderers |
| `internal/upstream/extract_*.go` | Create | One extractor per source (skills, agents, plugins, marketplaces, hooks, mcp, tools, schemastore, portable, changelog) |
| `internal/upstream/command.go` | Create | cobra wiring for `pull`, `digest`, `diff`, `check`, `render`, `validate-fixtures` |
| `internal/upstream/render.go`, `validate.go` | Create | Phase 3: docs page renderer, runtime-validator driver |
| `internal/upstream/guard_test.go` | Create | Phase 2 guardrail |
| `internal/upstream/digest.json`, `sources.lock.json` | Create | Committed last-known digest and lock (embedded) |
| `internal/upstream/acknowledged.json`, `runtime_fixtures.json` | Create | Deliberate deviations; runtime fixture manifest |
| `internal/upstream/testdata/snippets/**` | Create | Golden section snippets per extractor |
| `cmd/specdrift/main.go` | Create | Thin main |
| `internal/artifact/parse_md_kinds.go`, `parse_json.go`, `parse_marketplace.go`, `types.go`, `knowndata.go` | Modify | Exported key lists; archive and command source kinds; hook events and tools refresh; `DeprecatedTools` |
| `internal/rules/agents/toolsknown.go`, `internal/rules/commands/allowedtoolsknown.go` | Modify | Removed/renamed wording from `DeprecatedTools` |
| `internal/rules/marketplace/reservedname.go`, `pluginsourcevalid.go`, `externalsourceskipped.go` | Modify | `claude-tag-plugins`; archive and command handling |
| `internal/rules/version.go` | Modify | Ruleset bump; doc comment covers known-data changes |
| `internal/cli/version.go`, `rules.go` | Modify | Spec line; `upstream_version` |
| `internal/artifact/testdata/ok/**`, `bad/**` | Create | New-event hooks file, new-tool agent and skill, archive-and-command marketplace, runtime fixtures |
| `.github/workflows/spec-drift.yml` | Create | Weekly drift workflow |
| `scripts/spec-drift-issue.sh`, `scripts/labels.sh` | Create / Modify | Issue lifecycle; `spec-drift` label |
| `justfile` | Modify | `spec-check`, `spec-sync`, `spec-validate-fixtures`; `render --check` in `docs-check` |
| `.github/workflows/ci.yml` | Modify | `render --check` step (Phase 3, per OQ9) |
| `docs/rules/upstream-spec.md` | Create | Rendered digest page |
| `docs/rules/rules.md`, `README.md`, `docs/rules-json-schema.md`, `CLAUDE.md` | Modify | Event and tool tables, reserved names, source kinds, JSON field, commands and conventions |
| `docs/design/0006-*.md` | Modify | §7 row for `HookEntryKeys`; implemented-by note |

## Testing Plan

- **Unit (library):** table scanner edge cases; every extractor against
  its golden snippet plus the three negative cases; digest determinism;
  diff goldens; lock diff classification.
- **Fetch:** `httptest.Server` for retry, timeout, redirect, and manifest
  fields; a test asserting no extractor or command test opens a real
  socket (`GOFLAGS=-mod=mod` is irrelevant; simply no network hosts in
  test config).
- **Command:** `check` end to end against `httptest`-served fixtures for
  exit 0, 1, and 2; `--update` writes both files; `render --check` for
  both outcomes; `validate-fixtures` with a fake `claude` on `PATH`.
- **Guardrail:** one subtest per row; message contents asserted; stale
  acknowledgement fails; unknown acknowledgement path fails at load.
- **Parsers and rules (Phase 2):** table-driven tests for archive and
  command sources with ranges; reserved-name addition; regression
  fixtures for unknown event, unknown tool, empty source still flag.
- **Workflow:** `just lint-actions`; the three-run dispatch sequence on
  the branch; `scripts/spec-drift-issue.sh --dry-run` output checked by
  eye for each branch of the state machine.
- **Coverage:** `just coverage-gate` (55% floor) on `internal/upstream`
  and every touched `internal/` package.
- **Docs:** `just lint-md`, `just docs-check`, `just docs-mkdocs-check`
  after Phase 3's rendered page lands.
- **Dogfood:** `just self-check` plus a `donaldgifford/claude-skills`
  checkout after Phases 2 and 3.

## Dependencies

- DESIGN-0006 approved (all twelve questions resolved 2026-09-12).
- GitHub Actions network egress to `code.claude.com`,
  `www.schemastore.org`, `raw.githubusercontent.com`, and the npm
  registry (Phase 3); `gh` on the runner (preinstalled on
  `ubuntu-latest`).
- Node pinned in `mise.toml` (already, for the docs site) for the
  runtime validator install.
- No new Go module dependencies: cobra is already present; the table
  scanner is hand-written.
- Upstream shape as of 2026-09-13: 33 hook events, 45 tools, six
  marketplace source kinds, 17 reserved names, docs markers through
  v2.1.268. Re-run `spec-check` at the start of each phase; the counts
  in this document are the day-one expectations, not fixed targets.

## Resolved Decisions

All eleven resolved by Donald on 2026-09-13, every one on option (a).
Task references such as "per OQ2" point here.

- **OQ1 — Phase 1 release label:** `dont-release`. goreleaser builds
  only `./cmd/claudelint`, and Phase 1 does not touch it.
- **OQ2 — the four undocumented entries in `KnownTools`:** keep `Task`
  with an acknowledgement citing the v2.1.63 rename to `Agent`; drop
  `BashOutput`, `KillShell`, and `MultiEdit`, since the tools-known rules
  exist precisely because the runtime silently ignores unknown names.
  **Addendum:** record deprecated, removed, and renamed tools with the
  version and replacement in a "Deprecated and removed tools" table in
  rules.md, backed by `artifact.DeprecatedTools` so the rules can name
  the replacement and the guardrail can catch a stale row. The Claude
  Code changelog records `BashOutput` (unshipped 2.0.64, replaced by
  `TaskOutput`) and `TaskOutput` (deprecated 2.1.83 in favour of `Read`)
  but has no entry for `KillShell`, `MultiEdit`, or the `Task` rename, so
  the table carries a source column and falls back to the first docs
  marker at which the tool left the reference table.
- **OQ3 — ruleset version for Phase 2:** minor, `v1.5.0` → `v1.6.0`;
  the `RulesetVersion` doc comment gains known-data changes as an
  explicit bump trigger.
- **OQ4 — documented fields no rule consumes:** acknowledge each with
  the reason "no consuming rule; parse when a rule needs it". Parsing
  follows rule demand in the IMPL that adds the rule.
- **OQ5 — runtime fixture layout:** reuse the two existing
  `.claude-plugin` fixtures and add the missing `pass` / `fail` cases as
  `.claude-plugin`-shaped directories under `internal/artifact/testdata`,
  shared with the parser tests.
- **OQ6 — golden snippets:** `specdrift digest --write-snippets DIR`
  writes exactly the section bytes each extractor consumed.
- **OQ7 — version spec line:** `spec       v2.1.268 (a1b2c3d4)`, the
  parenthesised value being the first eight hex characters of the
  digest's sha256. No date.
- **OQ8 — sanity floors:** derived at run time from the committed
  digest as `max(1, ceil(0.75 × committed count))`.
- **OQ9 — where `render --check` runs:** a step in `ci.yml`'s `lint`
  job plus the same command inside `just docs-check`.
- **OQ10 — runtime validator wiring:** one job; the runtime steps run
  after the drift check with `continue-on-error`, their report is
  appended, and the issue script runs once.
- **OQ11 — marketplace `archive` and `command` source kinds:** fix in
  Phase 2 as part of the catch-up.

### Amendments made while implementing

- **Lock fields (Phase 1).** DESIGN §4 specified `last_modified` per
  source and made no exception for the optional probes. Measuring both
  against the live sources showed each one breaks the invariant the lock
  exists for. `code.claude.com` sets `Last-Modified` to the time of the
  request, so two fetches five seconds apart differ for byte-identical
  pages; the two `code.claude.com/schemas` probes resolve to a product
  page whose body carries a per-request nonce. Both were dropped from
  the committed lock and DESIGN §4 was amended in the same commit. The
  header is still recorded in the work-directory `manifest.json`.

## Open Questions

None. All eleven were resolved on 2026-09-13; see Resolved Decisions.

## References

- [DESIGN-0006](../design/0006-upstream-spec-drift-detection.md) — the
  design this implements; §1 sources, §3 extractors, §6 workflow, §7
  guardrail, Resolved Decisions
- [INV-0006](../investigation/0006-rule-coverage-audit-against-current-claude-code-docs.md)
  — the manual audit whose approach the tool automates
- [IMPL-0004](0004-phase-4-ruleset-alignment-and-agent-rules.md) —
  house checklist, dogfood convention, and OQ6 release cadence reused here
- [DESIGN-0005](../design/0005-agent-rules-and-opt-in-rule-mechanism.md)
  — agent enum sets under guard
- `internal/rules/all/fingerprint_test.go` and `cmd/genfp/main.go` —
  guardrail and dev-tool precedents
- Claude Code docs (raw Markdown): [skills](https://code.claude.com/docs/en/skills.md),
  [sub-agents](https://code.claude.com/docs/en/sub-agents.md),
  [plugins reference](https://code.claude.com/docs/en/plugins-reference.md),
  [plugin marketplaces](https://code.claude.com/docs/en/plugin-marketplaces.md),
  [hooks](https://code.claude.com/docs/en/hooks.md),
  [MCP](https://code.claude.com/docs/en/mcp.md),
  [tools reference](https://code.claude.com/docs/en/tools-reference.md)
- SchemaStore: [plugin manifest](https://www.schemastore.org/claude-code-plugin-manifest.json),
  [marketplace](https://www.schemastore.org/claude-code-marketplace.json),
  [settings](https://www.schemastore.org/claude-code-settings.json)
- Agent Skills [specification](https://agentskills.io/specification) and
  [skills-ref validator](https://github.com/agentskills/agentskills/tree/main/skills-ref);
  Anthropic [quick_validate.py](https://github.com/anthropics/skills/blob/main/skills/skill-creator/scripts/quick_validate.py)
