---
id: DESIGN-0006
title: "Upstream spec drift detection"
status: Draft
author: Donald Gifford
created: 2026-09-12
---

<!-- markdownlint-disable-file MD025 MD041 -->

# DESIGN-0006: Upstream spec drift detection

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-09-12

<!--toc:start-->
- [Overview](#overview)
- [Goals and Non-Goals](#goals-and-non-goals)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Background](#background)
  - [The gap today](#the-gap-today)
  - [Prior art in this org](#prior-art-in-this-org)
  - [Upstream sources survey](#upstream-sources-survey)
- [Detailed Design](#detailed-design)
  - [1. Sources and tiers](#1-sources-and-tiers)
  - [2. Pipeline and CLI](#2-pipeline-and-cli)
  - [3. Extractors](#3-extractors)
  - [4. Digest and lock format](#4-digest-and-lock-format)
  - [5. Diff semantics](#5-diff-semantics)
  - [6. Workflow](#6-workflow)
  - [7. Guardrail test](#7-guardrail-test)
  - [8. Runtime validator job](#8-runtime-validator-job)
  - [9. Human-facing outputs](#9-human-facing-outputs)
- [API / Interface Changes](#api--interface-changes)
- [Data Model](#data-model)
- [Testing Strategy](#testing-strategy)
- [Migration / Rollout Plan](#migration--rollout-plan)
- [Resolved Decisions](#resolved-decisions)
- [Open Questions](#open-questions)
- [References](#references)
<!--toc:end-->

## Overview

claudelint encodes Anthropic's Claude Code artifact specs — skill, agent,
and command frontmatter; plugin and marketplace manifests; hook and MCP
config shapes; the built-in tool list — as hand-maintained Go constants and
parser key lists. Anthropic ships Claude Code several times a week and the
public docs are the only authoritative spec, so those constants go stale
silently until a dogfood pass or a user report catches a false positive.

This design specifies `specdrift`, a Go dev tool that fetches the upstream
sources at CI time, extracts a small, deterministic **spec digest**, diffs
it against the committed last-known digest, and reports the delta to a
single tracking issue. A guardrail test then ties the digest to the
known-data sets in `internal/artifact`, so code drift becomes a failing
test that names the identifier to update instead of a surprise. It is the
same regenerate-and-diff shape as the ruleset fingerprint guardrail
(internal drift) and the CHANGELOG drift check, pointed upstream.

Only the digest and a source lockfile are committed. The raw docs are
fetched fresh on every run and retained as a workflow artifact for
forensics; they are never checked in.

## Goals and Non-Goals

### Goals

- Detect additions, removals, and changes to the documented field tables,
  enum sets, and numeric limits for every artifact kind claudelint lints,
  within one week of publication.
- Commit only a derived digest and a source lockfile, never raw docs.
- Produce a diff that names the digest paths that changed and, through the
  guardrail test, the exact Go identifiers to update.
- Surface drift through one tracking issue that an INV or IMPL doc can
  link to; never fail an unrelated PR.
- Cross-check claudelint's own fixtures against the runtime's validator
  (`claude plugin validate --strict --json`) so runtime-versus-linter
  disagreements are visible alongside docs drift.
- Keep `just ci` and `just test` network-free.

### Non-Goals

- Generating rules or parser code from the digest. The tool says *that*
  and *what* changed; whether a new field deserves a rule, and at what
  severity, remains a human design decision (the INV-0006 pattern).
- Tracking ecosystems other than Claude Code and the Agent Skills spec.
  OpenCode and OpenAI compatibility is INV-0001 territory.
- Deep extraction for surfaces the docs mark experimental (monitors,
  themes, channels, LSP servers). Their field names are recorded so an
  addition is visible; nothing more.
- Diffing docs prose. Wording changes that alter no field, enum, or limit
  are noise by design and are reported only as "source changed, digest
  unchanged".
- Validating user artifacts against the SchemaStore JSON Schemas at lint
  time. Those schemas lag the docs (see Background) and are used here only
  as a cross-check.

## Background

### The gap today

[INV-0006](../investigation/0006-rule-coverage-audit-against-current-claude-code-docs.md)
(2026-07-09) audited the ruleset against the docs by hand and found 4 rules
producing wrong results and 7 needing updates;
[IMPL-0004](../impl/0004-phase-4-ruleset-alignment-and-agent-rules.md)
closed that gap. INV-0006's four-step approach — dump the catalog,
inventory what the parsers check, fetch the docs, diff — is exactly the
algorithm this design automates. Nothing recurs today: no scheduled
workflow touches upstream, no spec is vendored, there is no `go:generate`
and no `just` recipe for it. The only staleness markers in the tree are
`(2026-07)` comments in `internal/artifact/knowndata.go`, and no test
reads them.

A spot check on 2026-09-11, nine weeks after the audit, against the same
docs:

| Surface | Documented | In code | Effect today |
| --- | --- | --- | --- |
| Hook events | 33 | 30 (`KnownHookEvents`) | `hooks/event-name-known` errors on `DirectoryAdded`, `PreModelSwitch`, `PostModelSwitch` |
| Built-in tools | 45 (tools reference table) | 18 (`KnownTools`) | `agents/tools-known` and `commands/allowed-tools-known` warn on 31 documented tools (`EnterPlanMode`, `SendMessage`, `TaskCreate`, `Workflow`, ...); 4 entries (`BashOutput`, `KillShell`, `MultiEdit`, `Task`) are no longer in the table |
| Skill frontmatter fields | 20 | 10 parsed | `arguments`, `effort`, `paths`, `shell`, `hooks`, `background`, `metadata`, `license`, `compatibility`, `argument-hint` invisible to rules |
| Plugin manifest fields | 25 | 6 parsed | `plugin/manifest-fields` cannot see `author`, `hooks`, `mcpServers`, `dependencies`, `userConfig`, ... |
| Docs version markers | through v2.1.268 | audit at v2.1.205 | roughly 60 releases unobserved |

The hook-event and tool-list rows are live false positives against
doc-valid input. The frontmatter rows are coverage gaps: nothing flags an
unknown key, so new fields are silently ignored rather than surfaced.

### Prior art in this org

- `docz-site/.github/workflows/spec-drift.yml` — weekly cron plus
  `workflow_dispatch` and `pull_request`; fetches an upstream OpenAPI spec,
  diffs it against the vendored copy, emits a `::warning` on PRs (never
  fails them), and on scheduled runs opens or comments on a single
  `spec-drift`-labelled tracking issue. This is the reporting model
  adopted here.
- `wiz-sdk/.github/workflows/schema-check.yml` — daily pull of a live
  GraphQL schema, generated `SCHEMA_CHANGELOG.md`, auto-pushed
  `auto/schema-update-<date>` branch. The pull-and-diff shape is adopted;
  the auto-commit is not (see OQ3).
- `internal/rules/all/fingerprint_test.go` — the in-repo precedent for a
  committed expected value plus a test that fails with instructions when
  the computed value drifts. The guardrail in this design is the same
  mechanism with the digest as the expected value.
- `.github/workflows/changelog.yml` — regenerate-and-diff as a CI check.

### Upstream sources survey

Verified 2026-09-11 and 2026-09-12.

| Source | Format | Freshness | Notes |
| --- | --- | --- | --- |
| `https://code.claude.com/docs/en/<page>.md` | Raw Markdown (Mintlify export) | Tracks releases within days; pages carry `v2.1.NNN` markers and a `last-modified` header | Byte-stable across fetches. Field tables are GFM table rows with a backticked field name in the first column and Yes/No in the second; hook events are H3 headings under the "Hook events" H2. Index at `/docs/llms.txt`. **Primary source.** |
| `https://www.schemastore.org/claude-code-plugin-manifest.json` and `.../claude-code-marketplace.json` | JSON Schema draft-07 | Generated once on 2026-04-23 (SchemaStore PR 5603, `$comment: Generated on ...`), not refreshed since | Hook enum has 29 events versus 33 in the docs. Machine-readable but stale; cross-check only. `json.schemastore.org` 301s to `www`. |
| `https://www.schemastore.org/claude-code-settings.json` | JSON Schema draft-07 | Synced roughly monthly by Anthropic-affiliated contributors, last to v2.1.220 (2026-07-27) | Hook enum has 31 events; still missing `PreModelSwitch` / `PostModelSwitch`. Cross-check only. |
| `https://code.claude.com/schemas/{plugin,marketplace}.json` | — | — | Appears as the `$schema` example in the docs but 302s to the product page. Not usable; re-probed on every run so it is picked up if it goes live. |
| `agentskills/agentskills` `docs/specification.mdx` and `skills-ref/src/skills_ref/validator.py` | MDX + Python constants | Stable | Portable skill spec: six allowed fields, name regex, `MAX_SKILL_NAME_LENGTH = 64`, `MAX_DESCRIPTION_LENGTH = 1024`, `MAX_COMPATIBILITY_LENGTH = 500`. |
| `anthropics/skills` `skills/skill-creator/scripts/quick_validate.py` | Python | Stable | Anthropic's own strict allowlist (`name`, `description`, `license`, `allowed-tools`, `metadata`, `compatibility`), kebab-case regex, no `<`/`>` in description. **Not referenced anywhere in this repo today** (zero hits); it becomes a digest section. |
| `anthropics/claude-code` `CHANGELOG.md` | Markdown | Per release | Top `## X.Y.Z` heading is the latest release; recorded as metadata. |
| `claude plugin validate --strict --json` | Local CLI | Per installed version | Works with an empty `CLAUDE_CONFIG_DIR` (no auth). Validates `plugin.json` / `marketplace.json` including `Unknown field '<x>'` warnings, and agent files when pointed at an `agents/` directory. **Does not walk `skills/`** at v2.1.259 (returns empty `contents`). Manifest-and-agent oracle only. |

Two consequences shape the design: the docs are the only source fresh
enough to be authoritative, and because they are prose, the extractable
facts (tables, enums, limits) are what get digested, not the pages.

## Detailed Design

### 1. Sources and tiers

Each source has a stable id, a tier, and the digest sections it feeds.

| Id | Tier | URL | Feeds |
| --- | --- | --- | --- |
| `docs.skills` | A | `code.claude.com/docs/en/skills.md` | `skills.frontmatter`, `skills.substitutions`, `skills.portable_fields` |
| `docs.agents` | A | `code.claude.com/docs/en/sub-agents.md` | `agents.frontmatter`, `agents.enums` |
| `docs.plugins` | A | `code.claude.com/docs/en/plugins-reference.md` | `plugins.manifest_fields`, `plugins.locations`, `plugins.substitution_fields`, `hooks.types` |
| `docs.marketplaces` | A | `code.claude.com/docs/en/plugin-marketplaces.md` | `marketplace.fields`, `marketplace.owner_fields`, `marketplace.plugin_entry_fields`, `marketplace.sources`, `marketplace.reserved_names` |
| `docs.hooks` | A | `code.claude.com/docs/en/hooks.md` | `hooks.events`, `hooks.handler_fields`, `hooks.timeout_defaults` |
| `docs.mcp` | A | `code.claude.com/docs/en/mcp.md` | `mcp.transports` (see OQ5) |
| `docs.tools` | A | `code.claude.com/docs/en/tools-reference.md` | `tools.builtin` |
| `docs.memory` | A | `code.claude.com/docs/en/memory.md` | `claude_md.import_depth`, `claude_md.rules_frontmatter` (Phase 3) |
| `schemastore.plugin` | B | `www.schemastore.org/claude-code-plugin-manifest.json` | `schemastore.plugin_manifest.{properties,hook_events,hook_types,mcp_types}` |
| `schemastore.marketplace` | B | `www.schemastore.org/claude-code-marketplace.json` | `schemastore.marketplace.{properties,plugin_entry_properties,source_kinds}` |
| `schemastore.settings` | B | `www.schemastore.org/claude-code-settings.json` | `schemastore.settings.{hook_events,default_modes}` |
| `agentskills.spec` | C | `raw.githubusercontent.com/agentskills/agentskills/main/docs/specification.mdx` | `portable.frontmatter` |
| `agentskills.validator` | C | `raw.githubusercontent.com/agentskills/agentskills/main/skills-ref/src/skills_ref/validator.py` | `portable.limits`, `portable.allowed_fields` |
| `anthropic.quick_validate` | C | `raw.githubusercontent.com/anthropics/skills/main/skills/skill-creator/scripts/quick_validate.py` | `portable.anthropic_allowed_fields`, `portable.name_pattern`, `portable.description_constraints` |
| `claudecode.changelog` | D | `raw.githubusercontent.com/anthropics/claude-code/main/CHANGELOG.md` | `meta.claude_code_latest` |

Tier semantics:

- **A** is authoritative. The guardrail test (§7) compares Go constants
  against Tier A sections only.
- **B** is cross-checked. Where B disagrees with A the disagreement is
  recorded in `digest.disagreements` and reported as information, never as
  drift (OQ9). They disagree today.
- **C** defines the portable Agent Skills spec, kept separate from the
  Claude Code superset so a future `skills/portable-fields` rule can read
  it directly.
- **D** is metadata used in reports.

`docs/en/commands.md` is deliberately not a source: custom commands share
the skill frontmatter model, and that page is the built-in slash-command
list.

### 2. Pipeline and CLI

`specdrift` is a thin `main` in `cmd/specdrift` over a library package
`internal/upstream` (OQ1). The precedent is `cmd/genfp`: a dev tool
inside the module, run with `go run`, excluded from goreleaser.

```text
specdrift pull   [--out DIR]
    Fetch every source into DIR/<id>.<ext>; write DIR/manifest.json
    (url, status, sha256, etag, last-modified, bytes) per source.

specdrift digest --in DIR [--out FILE] [--lock FILE]
    Run every extractor over DIR; write the digest and the lock.

specdrift diff   --base FILE --head FILE [--format text|json|markdown] [--out FILE]
    Structural diff of two digests. Exit 0 = identical, 1 = drift.

specdrift check  [--format ...] [--out FILE] [--update]
    pull → digest → diff against the committed digest. CI entry point.
    --update rewrites the committed digest and lock (just spec-sync).

specdrift render --digest FILE --out docs/rules/upstream-spec.md
    Human-readable page (Phase 3, §9).
```

Exit codes follow claudelint's own convention: 0 clean, 1 drift, 2
fetch or extraction failure. Fetching uses `net/http` with three retries
and backoff, a 30-second per-source timeout, a `User-Agent` naming the
repo, and honours redirects (the `www.schemastore.org` hop). No source
needs authentication.

Determinism: object keys sorted, arrays sorted, two-space indent, LF line
endings, trailing newline. Timestamps never enter the digest, so running
`digest` twice on the same input is byte-identical (tested).

### 3. Extractors

One extractor per (source, section). Each declares a source id, an
**anchor** (a heading regex), the table columns it reads, a **sanity
floor** (minimum row count), and a golden snippet under
`internal/upstream/testdata/`. Anchors verified against the live pages on
2026-09-12:

| Digest section | Source | Anchor | Extracts |
| --- | --- | --- | --- |
| `skills.frontmatter` | `docs.skills` | `### Frontmatter reference` | field, required (`yes` / `no` / `recommended`) |
| `skills.portable_fields` | `docs.skills` | `#### Using skill frontmatter outside Claude Code` | backticked field names in the section |
| `skills.substitutions` | `docs.skills` | `#### Available string substitutions` | variable names |
| `agents.frontmatter` | `docs.agents` | `### Write subagent files`, first table whose first row is `name` | field, required |
| `agents.enums` | `docs.agents` | same table; `model`, `permissionMode`, `effort`, `memory`, `isolation`, `color` rows | backticked tokens in the description cell, per field |
| `plugins.manifest_fields` | `docs.plugins` | `## Plugin manifest schema` → `### Required fields`, `### Metadata fields`, `### Component path fields` | field, type, group |
| `plugins.locations` | `docs.plugins` | `### File locations reference` | component, default path |
| `plugins.substitution_fields` | `docs.plugins` | `### Environment variables` | component, fields where placeholders resolve |
| `hooks.events` | `docs.hooks` | `## Hook events` → every `###` child heading until the next `##` | event names |
| `hooks.types` | `docs.hooks` | `### Hook handler fields`, `type` row | `command`, `http`, `mcp_tool`, `prompt`, `agent` |
| `hooks.handler_fields` | `docs.hooks` | `### Hook handler fields` → per-type tables | field, required, per type |
| `hooks.timeout_defaults` | `docs.hooks` | `timeout` row in the common-fields table | numbers keyed by type and by event override |
| `marketplace.fields` / `owner_fields` | `docs.marketplaces` | `## Marketplace schema` → `### Required fields`, `### Owner fields`, `### Optional fields` | field, required |
| `marketplace.plugin_entry_fields` | `docs.marketplaces` | `## Plugin entries` → `### Required fields`, `### Optional plugin fields` | field, required |
| `marketplace.sources` | `docs.marketplaces` | `## Plugin sources` → each `###` subsection's table | kind → required and optional fields |
| `marketplace.reserved_names` | `docs.marketplaces` | paragraph beginning `**Reserved names**` | backticked names |
| `tools.builtin` | `docs.tools` | first table after the page H1 | tool names |
| `mcp.transports` | `docs.mcp` | `type` field enumeration (OQ5) | `stdio`, `http`, `sse`, `ws` |
| `schemastore.*` | Tier B | JSON paths, not headings | `properties` keys and enums via a small JSON-pointer walker |
| `portable.*` | Tier C | `^(MAX_[A-Z_]+) = (\d+)` and `ALLOWED_FIELDS`/`ALLOWED_PROPERTIES` literals; the spec's frontmatter table | limits, allowed fields, regex |
| `meta.claude_code_latest` | `claudecode.changelog` | first `^## \d+\.\d+\.\d+` | version string |

Rules every extractor follows:

- **Scope to the section.** Rows are read from the first GFM table after
  the anchor until the next heading of equal or higher level. Whole-page
  matching is not allowed: `plugins-reference.md` alone has over 100
  backticked first-column cells across unrelated tables.
- **Line-oriented table parsing.** GFM tables are line-based: header row,
  delimiter row, body rows split on unescaped `|`, first cell stripped of
  backticks. No Markdown library is added; goldmark would be a new
  dependency for one construct, and the repo has none today.
- **Enums are explicit, not heuristic.** Enum values inside a description
  cell are read only for fields the extractor names, as the backticked
  tokens in that cell.
- **Hard failure on missing structure** (OQ11). A missing anchor, a table
  with unexpected headers, or a row count under the floor is an error
  naming the source and anchor; the run exits 2 and no digest is written.
  A partial digest would diff as mass removal, which is worse than no
  digest. The floors start at the current counts minus a margin (hook
  events ≥ 25, tools ≥ 30, skill fields ≥ 12).
- **Golden snippets, not golden pages.** Each extractor's fixture is the
  relevant section only (a few KB), hand-trimmed from the live page and
  refreshed only when the extractor changes. This keeps `just test`
  offline and the fixtures reviewable.

### 4. Digest and lock format

`internal/upstream/digest.json` (illustrative, abbreviated):

```json
{
  "digest_version": 1,
  "meta": {
    "claude_code_latest": "2.1.268",
    "docs_max_marker": "v2.1.268"
  },
  "tools": {
    "builtin": ["Agent", "Artifact", "AskUserQuestion", "Bash", "..."],
    "deprecated": []
  },
  "hooks": {
    "events": ["ConfigChange", "CwdChanged", "DirectoryAdded", "..."],
    "types": ["agent", "command", "http", "mcp_tool", "prompt"],
    "handler_fields": {
      "common": [{"name": "type", "required": "yes"}, {"name": "if", "required": "no"}],
      "command": [{"name": "command", "required": "yes"}, {"name": "args", "required": "no"}]
    },
    "timeout_defaults": {
      "by_type": {"agent": 60, "command": 600, "http": 600, "mcp_tool": 600, "prompt": 30},
      "by_event": {"MessageDisplay": 10, "PostModelSwitch": 30, "PreModelSwitch": 30, "UserPromptSubmit": 30}
    }
  },
  "skills": {
    "frontmatter": [
      {"name": "allowed-tools", "required": "no"},
      {"name": "description", "required": "recommended"},
      {"name": "name", "required": "no"}
    ],
    "portable_fields": ["allowed-tools", "compatibility", "description", "license", "metadata", "name"],
    "substitutions": ["$ARGUMENTS", "$ARGUMENTS[N]", "$N", "$name", "${CLAUDE_SKILL_DIR}", "..."]
  },
  "agents": {
    "frontmatter": [{"name": "description", "required": "yes"}, {"name": "name", "required": "yes"}],
    "enums": {
      "color": ["blue", "cyan", "green", "orange", "pink", "purple", "red", "yellow"],
      "effort": ["high", "low", "max", "medium", "xhigh"],
      "isolation": ["worktree"],
      "memory": ["local", "project", "user"],
      "model": ["fable", "haiku", "inherit", "opus", "sonnet"],
      "permissionMode": ["acceptEdits", "auto", "bypassPermissions", "default", "dontAsk", "manual", "plan"]
    }
  },
  "plugins": {
    "manifest_fields": [{"name": "name", "type": "string", "group": "required"}, {"name": "hooks", "type": "string|array|object", "group": "component"}],
    "locations": {"agents": "agents/", "hooks": "hooks/hooks.json", "manifest": ".claude-plugin/plugin.json"}
  },
  "marketplace": {
    "fields": [{"name": "name", "required": "yes"}, {"name": "owner", "required": "yes"}, {"name": "plugins", "required": "yes"}, {"name": "renames", "required": "no"}],
    "plugin_entry_fields": [{"name": "name", "required": "yes"}, {"name": "source", "required": "yes"}, {"name": "strict", "required": "no"}],
    "sources": {
      "archive": {"required": ["source", "url"], "optional": ["sha256"]},
      "command": {"required": ["source", "command"], "optional": ["mode", "timeout"]},
      "git-subdir": {"required": ["source", "url", "path"], "optional": ["ref", "sha"]},
      "github": {"required": ["source", "repo"], "optional": ["ref", "sha"]},
      "npm": {"required": ["source", "package"], "optional": ["registry", "version"]},
      "url": {"required": ["source", "url"], "optional": ["ref", "sha"]}
    },
    "reserved_names": ["agent-skills", "anthropic-marketplace", "claude-code-marketplace", "..."]
  },
  "mcp": {"transports": ["http", "sse", "stdio", "ws"]},
  "portable": {
    "allowed_fields": ["allowed-tools", "compatibility", "description", "license", "metadata", "name"],
    "limits": {"compatibility": 500, "description": 1024, "name": 64},
    "name_pattern": "^[a-z0-9]+(-[a-z0-9]+)*$"
  },
  "schemastore": {
    "plugin_manifest": {"hook_events": ["..."], "properties": ["$schema", "agents", "author", "..."]},
    "settings": {"default_modes": ["acceptEdits", "auto", "bypassPermissions", "default", "dontAsk", "plan"], "hook_events": ["..."]}
  },
  "disagreements": [
    {"docs_only": ["DirectoryAdded", "MessageDisplay", "PostModelSwitch", "PreModelSwitch"], "source": "schemastore.plugin", "source_only": [], "topic": "hooks.events"}
  ]
}
```

`internal/upstream/sources.lock.json` records content-derived facts only,
so it changes when upstream changes and at no other time:

```json
{
  "docs.hooks": {
    "sha256": "a6f4f8...",
    "url": "https://code.claude.com/docs/en/hooks.md",
    "version_marker": "v2.1.267"
  }
}
```

A lock change without a digest change means a prose-only edit; the report
lists it under "sources changed without affecting the digest".

**Amended during IMPL-0005 Phase 1.** Two fields this section originally
specified were removed after measuring them against the live sources,
because both break the invariant the lock exists for:

- `last_modified`. On `code.claude.com` the header carries the time of
  the request, not of the content: two fetches five seconds apart report
  timestamps five seconds apart for byte-identical pages. Committing it
  would make every `just spec-sync` a diff. The header is still recorded
  in the work-directory `manifest.json`, where a clock reading is
  diagnostic rather than committed.
- Optional probe entries. The two `code.claude.com/schemas` URLs do not
  exist yet, and the site answers them with its product page, whose body
  carries a per-request nonce. A probe answers a yes-or-no question about
  a URL; its body is not a specification until an extractor reads it, so
  it stays out of the committed record.

**Why the digest is embedded.** The CI comparison in §5 is always file
against file: the digest freshly generated from the live docs against the
digest committed in the tree. Embedding does not change that comparison;
the embedded bytes are the committed file at that commit. Embedding exists
for two readers that cannot conveniently open the tree: the guardrail test
(§7), where it is a convenience that follows from co-locating the file
with the package, and the release binary (§9, OQ12), where
`claudelint version` prints the spec the release was verified against.
Each release therefore carries its own upstream digest, bug reports can
quote it, and diffing what two releases believed about upstream is a diff
of their embedded digests. `go:embed` can only reach files inside the
package directory, which is why the file lives under `internal/upstream/`
(OQ2).

### 5. Diff semantics

The diff is structural over the digest tree, not textual:

- Sorted string arrays diff as sets: `added` / `removed`.
- Arrays of objects with a `name` key diff by name: `added` / `removed` /
  `changed` (with the old and new value of each differing key).
- Maps recurse; scalars compare directly.
- `meta.*`, `schemastore.*`, and `disagreements` are reported in a separate
  informational block and never count toward the exit code (OQ9).

Rendered as Markdown, each change is one line with its leaf path so a
reader can go straight to the constant:

```markdown
### hooks.events
- added: `DirectoryAdded`, `PostModelSwitch`, `PreModelSwitch`

### tools.builtin
- added: `Artifact`, `CronCreate`, `CronDelete`, ... (31)
- removed: `BashOutput`, `KillShell`, `MultiEdit`, `Task`

### skills.frontmatter
- changed: `description.required` no → recommended
```

The JSON form is the same tree with `added` / `removed` / `changed`
arrays, consumed by the workflow for its issue-dedupe marker.

### 6. Workflow

`.github/workflows/spec-drift.yml`, modelled on docz-site's:

```yaml
name: Upstream Spec Drift
on:
  schedule:
    - cron: "0 6 * * 1"   # Mondays 06:00 UTC, after the 00:00 CodeQL/govulncheck slot
  workflow_dispatch:
  pull_request:
    paths:
      - internal/upstream/**
      - cmd/specdrift/**
      - .github/workflows/spec-drift.yml

permissions:
  contents: read

jobs:
  drift:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      issues: write
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with: { go-version-file: go.mod }
      - name: Check upstream
        id: check
        run: |
          set +e
          go run ./cmd/specdrift check --work /tmp/upstream \
            --format markdown --out /tmp/report.md --json /tmp/diff.json
          echo "code=$?" >> "$GITHUB_OUTPUT"
      - name: Upload fetched sources and report
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: upstream-${{ github.run_id }}
          path: |
            /tmp/upstream
            /tmp/report.md
            /tmp/diff.json
          retention-days: 90
      - name: Step summary
        if: always()
        run: cat /tmp/report.md >> "$GITHUB_STEP_SUMMARY"
      - name: Warn on PR (never fails)
        if: steps.check.outputs.code == '1' && github.event_name == 'pull_request'
        run: echo "::warning title=Upstream spec drift::see the job summary; run 'just spec-sync' when ready"
      - name: Open, update, or close the tracking issue
        if: github.event_name != 'pull_request'
        env:
          GH_TOKEN: ${{ github.token }}
          CODE: ${{ steps.check.outputs.code }}
        run: ./scripts/spec-drift-issue.sh "$CODE" /tmp/report.md /tmp/diff.json
      - name: Fail on fetch/extract error
        if: steps.check.outputs.code == '2'
        run: exit 1
```

Issue handling (`scripts/spec-drift-issue.sh`, shell so it stays readable
in the workflow log):

- Exactly one open issue carries the `spec-drift` label. The body starts
  with a hidden marker `<!-- specdrift:diff-sha256=<hash of diff.json> -->`.
- Exit 1 with no open issue → create it. Exit 1 with an open issue whose
  marker differs → post a comment with the new report and update the
  body's marker. Same marker → no-op (no weekly nag).
- Exit 0 with an open issue → comment "resolved by upstream or by a sync
  on <date>" and close it (OQ3).
- Exit 2 → the job fails (a red scheduled run emails the maintainer) and
  the report, which names the broken extractor, is attached; no issue
  churn.
- The `spec-drift` label is added to `scripts/labels.sh`.

Only this workflow reaches the network; `ci.yml` is untouched. A
pull-request run exists so extractor changes are exercised against live
pages before merge, but it can only warn.

### 7. Guardrail test

`internal/upstream/guard_test.go` loads the embedded digest and compares
it with the constants rules actually use. It is the upstream analogue of
`TestRulesetFingerprint` and runs under `just test` with no network.

| Digest path | Go identifier | Relation |
| --- | --- | --- |
| `hooks.events` | `artifact.KnownHookEvents` | equal sets |
| `hooks.types` | `artifact.KnownHookTypes` | equal sets |
| `tools.builtin` | `artifact.KnownTools` | equal sets, modulo acknowledgements |
| `agents.enums.model` | `artifact.KnownModelAliases` + `inherit` | equal sets |
| `agents.enums.permissionMode` / `effort` / `color` / `memory` | `AgentPermissionModes` / `AgentEffortLevels` / `AgentColors` / `AgentMemoryScopes` | equal sets |
| `agents.frontmatter[].name` | `artifact.AgentFrontmatterKeys` (new export) | every documented key is parsed or acknowledged |
| `skills.frontmatter[].name` | `artifact.SkillFrontmatterKeys`, `CommandFrontmatterKeys` (new exports) | same |
| `plugins.manifest_fields[].name` | `artifact.PluginManifestKeys` (new export) | same |
| `marketplace.sources` keys | `artifact.SourceKind` values | every documented kind has a `SourceKind` |
| `marketplace.reserved_names` | the list in `internal/rules/marketplace/reservedname.go` | equal sets |
| `mcp.transports` | the set used by `mcp/transport-known` | equal sets |
| `hooks.timeout_defaults` | the defaults `hooks/timeout-present` cites in its message | equal |

The **acknowledgement file** `internal/upstream/acknowledged.json` is how
a deliberate deviation stays deliberate:

```json
{
  "tools.builtin": {
    "Task": "renamed to Agent in v2.1.63; still accepted by the runtime, kept as an alias (INV-0006)",
    "BashOutput": "undocumented since v2.1.2xx; kept one minor for compatibility, remove at next ruleset major"
  },
  "skills.frontmatter": {
    "hooks": "presence parsed; contents left to the hook rule package (DESIGN-0005 non-goal)",
    "paths": "not parsed; no rule consumes it (IMPL-0004 out of scope)"
  }
}
```

An acknowledged item suppresses the failure for that item and is rendered
with its reason on the docs page (§9). An acknowledgement whose item no
longer exists upstream *fails* the test, so the file self-cleans. The
failure message follows the fingerprint test's shape: the identifier, the
digest path, the delta, and the two ways to resolve it (update the code,
or acknowledge with a reason).

Exporting the parser key lists is a small refactor: `ParseSkill`,
`ParseCommand`, `ParseAgent`, and `ParsePlugin` read keys through the
exported slice instead of string literals, so the list cannot drift from
the parser. No behaviour change.

### 8. Runtime validator job

A second job in the same workflow cross-checks claudelint's fixtures
against the Claude Code runtime's validator, so a runtime that starts
rejecting what claudelint calls valid (or accepting what it rejects) is
visible in the same issue:

- Install the CLI (OQ8) and record `claude --version` in the report.
- Run `claude plugin validate --strict --json <path>` with an empty
  `CLAUDE_CONFIG_DIR` for every entry in
  `internal/upstream/runtime_fixtures.json`:

  ```json
  [
    {"path": "internal/artifact/testdata/ok/pluginroot", "kind": "plugin", "expect": "pass"},
    {"path": "internal/artifact/testdata/ok/marketplaces/object_sources", "kind": "marketplace", "expect": "pass"},
    {"path": "internal/artifact/testdata/bad/plugins/missing-name", "kind": "plugin", "expect": "fail"}
  ]
  ```

- Compare `success` against `expect`; list every disagreement with the
  runtime's `errors` / `warnings` verbatim. `Unknown field` warnings on an
  `ok` fixture are the signal that a field was removed upstream.
- Coverage is bounded by the runtime: manifests and agent directories
  only, since the validator does not walk `skills/` today. The job
  re-tests that assumption each run by validating a `skills/` fixture and
  reporting if `contents` becomes non-empty.

### 9. Human-facing outputs

- **The tracking issue** is the pointer INV and IMPL docs cite. Its body
  leads with the version delta (`v2.1.205 → v2.1.268`), then the digest
  diff grouped by section, then runtime-validator disagreements, then
  "sources changed without affecting the digest", then the workflow run
  link and the artifact name.
- **`docs/rules/upstream-spec.md`** (Phase 3) is generated by
  `specdrift render` from the digest and the acknowledgement file: one
  section per artifact kind listing documented fields and enums, whether
  claudelint parses each, and the acknowledgement reason where relevant,
  with a "verified against Claude Code vX.Y.Z on <lock date>" line. It
  lives under `docs/rules/` so the Starlight sidebar picks it up, needs
  `title:` frontmatter, and is checked for staleness by `just docs-check`
  through a `render --check` mode (OQ2).
- **`claudelint version`** may print the digest's Claude Code version so
  bug reports carry it (OQ12).

## API / Interface Changes

- New `cmd/specdrift` (dev tool; not a release artifact; added to the
  goreleaser ignore list alongside `genfp`).
- New `internal/upstream` package. Exported: `Digest`, `LoadEmbedded()`,
  `Fetch`, `Extract`, `Diff`, `Render`. Extractors are unexported.
- `internal/artifact`: new exported key lists `SkillFrontmatterKeys`,
  `CommandFrontmatterKeys`, `AgentFrontmatterKeys`, `PluginManifestKeys`.
  Parsers read through them. No behaviour change.
- `justfile`: `spec-check` (network; pull → digest → diff, prints the
  report), `spec-sync` (network; rewrite digest and lock), and
  `spec-validate-fixtures` (requires a local `claude`). None join `ci` or
  `check`.
- `.github/workflows/spec-drift.yml`, `scripts/spec-drift-issue.sh`, and
  the `spec-drift` label in `scripts/labels.sh`.
- Phase 3 only: `docs/rules/upstream-spec.md` and a `render --check` step
  in `just docs-check`.
- No claudelint CLI flags, config schema, or ruleset fingerprint changes.
  The digest is not a rule.

## Data Model

```go
// Package upstream — internal/upstream

type Digest struct {
    DigestVersion int                    `json:"digest_version"`
    Meta          Meta                   `json:"meta"`
    Tools         Tools                  `json:"tools"`
    Hooks         Hooks                  `json:"hooks"`
    Skills        Skills                 `json:"skills"`
    Agents        Agents                 `json:"agents"`
    Plugins       Plugins                `json:"plugins"`
    Marketplace   Marketplace            `json:"marketplace"`
    MCP           MCP                    `json:"mcp"`
    Portable      Portable               `json:"portable"`
    SchemaStore   SchemaStore            `json:"schemastore"`
    Disagreements []Disagreement         `json:"disagreements"`
}

type Field struct {
    Name     string `json:"name"`
    Required string `json:"required,omitempty"` // yes | no | recommended
    Type     string `json:"type,omitempty"`
    Group    string `json:"group,omitempty"`
}

type Source struct {
    ID   string // "docs.hooks"
    Tier byte   // 'A'..'D'
    URL  string
}

// Fields are declared in alphabetical order of their json tag, because
// encoding/json emits struct fields in declaration order and sorts only
// map keys. LastModified was dropped; see the amendment in section 4.
type LockEntry struct {
    SHA256        string `json:"sha256"`
    URL           string `json:"url"`
    VersionMarker string `json:"version_marker,omitempty"`
}

type Extractor interface {
    Section() string                       // digest path this fills
    Source() string                        // source id
    Extract(page []byte, d *Digest) error  // hard error on missing structure
}

type Change struct {
    Path    string      `json:"path"`     // "hooks.events"
    Kind    string      `json:"kind"`     // added | removed | changed
    Item    string      `json:"item"`     // "PreModelSwitch" or "description.required"
    Old, New any        `json:"old,omitempty"` // changed only
}

// Acknowledgements: digest path -> item -> reason.
type Acknowledged map[string]map[string]string
```

Files committed under `internal/upstream/`: `digest.json`,
`sources.lock.json`, `acknowledged.json`, `runtime_fixtures.json`, and
`testdata/snippets/<source>/<section>.md`. `digest.json` and
`acknowledged.json` are embedded with `go:embed` so `LoadEmbedded` needs
no filesystem at test time.

## Testing Strategy

- **Extractors:** golden tests from the vendored section snippets, one per
  extractor, asserting the exact extracted set. Negative tests: missing
  anchor, wrong table headers, and under-floor row count each return a
  hard error naming the source and anchor.
- **Table parser:** unit tests for escaped pipes, alignment rows, trailing
  whitespace, cells with nested backticks, and tables ending at a heading
  versus a blank line.
- **Digest:** determinism (digest twice → identical bytes), sorted output,
  `digest_version` round-trip.
- **Diff:** golden pairs for add, remove, change, no-op, and the
  informational-only sections; exit-code mapping.
- **Guardrail:** every row of the §7 table is one subtest; the failure
  message is asserted to contain the identifier and the digest path;
  stale acknowledgements fail.
- **Workflow:** `just lint-actions` (actionlint) on the YAML; a
  `workflow_dispatch` run from the branch must produce no drift against
  the freshly committed digest; a second run with a deliberately stale
  digest must open the issue with the expected body; a third clean run
  must close it. The issue script is exercised against the `spec-drift`
  label on a scratch issue before merge.
- **Runtime job:** each `runtime_fixtures.json` entry asserted once with
  the pinned CLI version; the `skills/` probe fixture asserted to return
  empty `contents` so the assumption in §8 is re-verified every run.
- **Coverage:** `internal/upstream` must clear the 55% floor
  (`just coverage-gate`).

## Migration / Rollout Plan

Three PRs, each independently useful, each with its own IMPL doc phase.

1. **Phase 1 — tool, digest, workflow** (branch `chore/spec-drift-tool`,
   label `dont-release`; no binary change). `internal/upstream` with
   Tier A–D extractors, `cmd/specdrift` with `pull` / `digest` / `diff` /
   `check`, the initial committed `digest.json` and `sources.lock.json`,
   `spec-drift.yml` with the issue script and label, `spec-check` /
   `spec-sync` recipes. Exit criterion: a dispatch run against the merged
   digest reports no drift, and a run against a hand-staled digest opens
   an issue whose body lists the three missing hook events.
2. **Phase 2 — guardrail and catch-up** (branch `fix/upstream-drift-2026-09`,
   label `minor` — the ruleset changes). Exported parser key lists,
   `guard_test.go`, `acknowledged.json` seeded with today's deliberate
   deviations, and the code fixes the guardrail demands: `KnownHookEvents`
   +3, `KnownTools` refreshed against the tools table with the four
   undocumented names acknowledged or dropped, the unparsed skill and
   plugin fields either parsed or acknowledged with a reason. Ruleset
   version bump and rules-doc updates per house policy; one dogfood pass
   against `donaldgifford/claude-skills` per IMPL-0004 convention.
3. **Phase 3 — runtime validator and rendered page** (branch
   `chore/spec-drift-runtime`, label `dont-release`).
   `runtime_fixtures.json`, the second workflow job, `specdrift render`,
   `docs/rules/upstream-spec.md`, the `render --check` step in
   `docs-check`, and the `claudelint version` line if OQ12 is accepted.

After Phase 1 the weekly issue exists; after Phase 2 drift is a failing
test; after Phase 3 the runtime is in the loop and there is a page to link
to. INV-0006's "deliberately not chasing yet" list moves into
`acknowledged.json`, where it is checked rather than remembered.

Rollback is trivial at every phase: the workflow is additive, the digest
is data, and the guardrail can be skipped with an acknowledgement while a
fix is in flight.

## Resolved Decisions

All twelve resolved by Donald on 2026-09-12, every item on option (a).

- **OQ1 — tool location:** `cmd/specdrift` thin `main` over an
  `internal/upstream` library, matching the `cmd/genfp` precedent and
  excluded from goreleaser the same way.
- **OQ2 — digest location:** `internal/upstream/digest.json` embedded via
  `go:embed`, co-located with the guardrail, plus a generated
  `docs/rules/upstream-spec.md` for humans (Phase 3) kept fresh by
  `render --check` in `just docs-check`. One file, three readers (CI diff,
  test, binary), no build-time plumbing; rationale in §4. The root `spec/`
  alternative with an ldflags-injected version string was rejected because
  it wires the version in two places and a plain `go build` reports `dev`.
- **OQ3 — reporting channel:** one `spec-drift` tracking issue, created on
  first drift, commented only when the diff hash changes, closed
  automatically on a clean run. Raw sources and the report are kept as a
  90-day workflow artifact. No bot commits.
- **OQ4 — cadence:** weekly, Mondays 06:00 UTC, plus `workflow_dispatch`.
- **OQ5 — MCP fields:** `mcp.transports` from the `mcp.md` prose;
  `mcp.server_fields` from the SchemaStore plugin-manifest `mcpServers`
  definition, promoted to Tier A for that section only and cross-checked
  against the `plugins-reference.md` MCP section.
- **OQ6 — raw docs:** never committed; workflow artifact only.
- **OQ7 — known data:** guarded, not generated. `knowndata.go` stays
  hand-written; deliberate deviations live in `acknowledged.json`.
- **OQ8 — runtime validator install:**
  `npm install -g @anthropic-ai/claude-code@latest`, with `claude --version`
  recorded in the report.
- **OQ9 — SchemaStore disagreements:** docs win. Disagreements are recorded
  under `digest.disagreements` and reported as information only.
- **OQ10 — phase order:** tool and workflow, then guardrail and catch-up
  fixes, then runtime validator and rendered page.
- **OQ11 — extraction failure:** hard fail. Exit 2, no digest written, the
  scheduled run goes red with the report naming the extractor.
- **OQ12 — digest version in the binary:** yes, in Phase 3.
  `claudelint version` prints the spec line and `rules --json` gains an
  additive `upstream_version` field.

## Open Questions

None. All twelve were resolved on 2026-09-12; see Resolved Decisions.

## References

- [INV-0006](../investigation/0006-rule-coverage-audit-against-current-claude-code-docs.md)
  — the manual audit this design automates; its Approach section is the
  algorithm
- [INV-0005](../investigation/0005-phase-2-dogfood-findings-marketplaces-mcp-and-spec-divergence.md)
  — prior upstream-shape false positives (marketplace nesting, stale
  `KnownTools`) and the "one-line bump" monitoring plan this replaces
- [IMPL-0004](../impl/0004-phase-4-ruleset-alignment-and-agent-rules.md)
  — remediation of INV-0006; out-of-scope list that seeds
  `acknowledged.json`
- [DESIGN-0005](0005-agent-rules-and-opt-in-rule-mechanism.md) — agent
  enum sets and the parser fields the guardrail compares against
- [DESIGN-0001](0001-claudelint-linter-architecture-and-rule-engine.md)
  — ruleset fingerprint guardrail, the in-repo precedent
- `internal/artifact/knowndata.go`, `internal/artifact/parse_md_kinds.go`,
  `internal/artifact/parse_json.go`, `internal/artifact/parse_marketplace.go`,
  `internal/artifact/parse_mcp.go`, `internal/rules/marketplace/reservedname.go`
  — the constants and key lists under guard
- `cmd/genfp/main.go` and `internal/rules/all/fingerprint_test.go` — dev
  tool and guardrail precedents
- `docz-site/.github/workflows/spec-drift.yml` and
  `wiz-sdk/.github/workflows/schema-check.yml` — org prior art
- Claude Code docs (raw Markdown): [skills](https://code.claude.com/docs/en/skills.md),
  [sub-agents](https://code.claude.com/docs/en/sub-agents.md),
  [plugins reference](https://code.claude.com/docs/en/plugins-reference.md),
  [plugin marketplaces](https://code.claude.com/docs/en/plugin-marketplaces.md),
  [hooks](https://code.claude.com/docs/en/hooks.md),
  [MCP](https://code.claude.com/docs/en/mcp.md),
  [tools reference](https://code.claude.com/docs/en/tools-reference.md),
  [memory](https://code.claude.com/docs/en/memory.md),
  [docs index](https://code.claude.com/docs/llms.txt)
- SchemaStore: [plugin manifest](https://www.schemastore.org/claude-code-plugin-manifest.json),
  [marketplace](https://www.schemastore.org/claude-code-marketplace.json),
  [settings](https://www.schemastore.org/claude-code-settings.json),
  [catalog](https://www.schemastore.org/api/json/catalog.json)
- Agent Skills: [specification](https://agentskills.io/specification),
  [skills-ref validator](https://github.com/agentskills/agentskills/tree/main/skills-ref)
- Anthropic: [quick_validate.py](https://github.com/anthropics/skills/blob/main/skills/skill-creator/scripts/quick_validate.py),
  [Claude Code CHANGELOG](https://github.com/anthropics/claude-code/blob/main/CHANGELOG.md)
