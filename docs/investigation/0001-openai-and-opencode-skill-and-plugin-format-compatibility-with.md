---
id: INV-0001
title: "OpenAI and OpenCode skill/plugin format compatibility with Claude"
status: Open
author: Donald Gifford
created: 2026-04-18
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0001: OpenAI and OpenCode skill/plugin format compatibility with Claude

**Status:** Open
**Author:** Donald Gifford
**Date:** 2026-04-18

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Environment](#environment)
- [Findings](#findings)
  - [Observation 1 — Field overlap matrix](#observation-1--field-overlap-matrix)
  - [Observation 2 — Round-trip loss quantification](#observation-2--round-trip-loss-quantification)
  - [Observation 3 — Semantic divergences that cannot be bridged](#observation-3--semantic-divergences-that-cannot-be-bridged)
- [Conclusion](#conclusion)
- [Recommendation](#recommendation)
- [References](#references)
<!--toc:end-->

## Question

Can claudelint provide a useful bidirectional `convert` subcommand
between Claude skill/plugin artifacts and the equivalent artifacts in
(a) OpenAI's Apps / Custom GPTs ecosystem and (b) OpenCode? Specifically:

1. Is there a meaningful schema overlap (name, description, trigger,
   instructions, tool allowlist, attachments) such that best-effort
   translation is useful?
2. Which fields are *lossy* in each direction, and can we detect and
   report loss deterministically?
3. Is a pure static translator sufficient, or do semantics diverge
   enough that round-tripping is not achievable even in principle?

## Hypothesis

- **Claude ↔ OpenCode:** high overlap. OpenCode's skill/plugin model
  was influenced by Claude's, and both use Markdown-with-frontmatter.
  Expect round-trippable for ≥80% of non-trivial skills, with small
  fixed-shape losses on tool identifiers and hook semantics.
- **Claude ↔ OpenAI (Apps / Custom GPTs):** partial overlap. GPT
  "Actions" map loosely to tool use, "Instructions" to system prompts,
  and "Knowledge" to skill reference files, but the hook/subagent model
  has no direct equivalent. Expect useful one-way import (OpenAI → Claude)
  and a diagnostics-heavy export (Claude → OpenAI).

## Context

This investigation gates **Phase 3** of IMPL-0001 (`claudelint convert`)
and the stretch goal in RFC-0001. Before spending engineering time on a
translator, we need to know whether the overlap is large enough to be
useful, and where the unavoidable losses are.

**Triggered by:** RFC-0001, IMPL-0001 (Phase 3)

## Approach

1. **Collect canonical specs and sample artifacts.**
   - Claude: `SKILL.md` frontmatter, `.claude/settings.json` hooks,
     plugin manifests, slash-command frontmatter, subagent frontmatter.
   - OpenCode: its skill and plugin schemas (frontmatter + manifest).
   - OpenAI: Custom GPT "Instructions", "Conversation starters",
     "Knowledge" files, "Actions" (OpenAPI), and the newer "Apps"
     platform manifest if available.
2. **Build a field-level mapping table** (source field → target field →
   transform → loss class). Loss classes: `lossless`, `defaulted`,
   `approximated`, `dropped`.
3. **Prototype translators** for the three most common skill shapes:
   (a) pure-prompt skill, (b) skill with reference files, (c) skill
   with tool allowlist.
4. **Run round-trip experiments.** For each direction, encode in source,
   convert to target, convert back, and diff. Quantify loss.
5. **Document divergent semantics.** Identify concepts present in one
   ecosystem but not the others (e.g., Claude hooks → no GPT
   equivalent; OpenAI Knowledge files → Claude skill `references/`).
6. **Summarize** in a findings matrix and recommend a scope.

## Environment

| Component | Version / Value |
|-----------|----------------|
| Claude Code CLI | latest at investigation time |
| OpenAI platform | Custom GPTs + Apps manifest as of investigation date |
| OpenCode | latest release at investigation time |
| Go (prototype) | 1.23+ (matches claudelint) |

## Findings

> Fill in as experiments run. Expected structure below.

### Observation 1 — Field overlap matrix

Placeholder for the mapping table. Populate with rows of the form:

| Claude field | OpenCode field | OpenAI field | Loss class | Notes |
|--------------|----------------|--------------|------------|-------|
| `name` (frontmatter) | `name` | GPT name | lossless | |
| `description` | `description` | GPT description | lossless | |
| `allowed-tools` | `tools` | Action operationIds | approximated | tool naming differs |
| `hooks` (settings.json) | `hooks` | — | dropped (→ OpenAI) | no GPT equivalent |
| `references/` | `resources/` | Knowledge files | approximated | upload mechanics differ |
| slash command `argument-hint` | — | conversation starter | approximated | |
| subagent `tools` list | — | — | dropped (→ OpenAI) | |

### Observation 2 — Round-trip loss quantification

Prototype results per skill shape (pure-prompt, with-refs, with-tools)
and per direction. Expected format: % of fields preserved, % approximated,
% dropped.

### Observation 3 — Semantic divergences that cannot be bridged

- Claude hooks execute shell in the user's harness; GPT has no hook
  concept. Translation direction Claude → OpenAI will drop hooks with
  a mandatory diagnostic.
- OpenAI Actions are OpenAPI specs; Claude's `allowed-tools` lists
  built-in tool names. Bidirectional mapping only exists for a small
  set of tools with obvious equivalents (web search, code execution).
- OpenCode and Claude differ on plugin command syntax in ways that
  require a translation table rather than structural mapping.

## Conclusion

> Fill in at investigation close. Draft expectations below.

**Answer:** *Inconclusive — pending experiment runs.*

Working thesis:

- **Yes** for a useful Claude ↔ OpenCode translator with a documented
  small set of approximated fields.
- **Partial** for Claude ↔ OpenAI: one-way import (OpenAI → Claude) is
  tractable; export (Claude → OpenAI) requires dropping hooks,
  subagents, and most plugin semantics with clear diagnostics.

## Recommendation

Tentative, pending experiments:

1. **Proceed with Phase 3**, but scope to two translators:
   `convert claude-to-opencode` and `convert opencode-to-claude`
   as the primary flow, with `convert openai-import` as a secondary
   one-way tool.
2. **Always emit a loss report.** The subcommand must print (or write
   alongside the output) a structured summary of every approximated or
   dropped field. Non-zero exit if loss exceeds a configurable budget.
3. **Freeze the translation table.** Check in the field-mapping matrix
   as data (`internal/convert/mappings/*.hcl` or `.yaml`) so that
   additions can be reviewed without code changes.
4. **Revisit OpenAI export** after the OpenAI Apps manifest stabilizes;
   until then, only one-way import is supported.

## References

- RFC-0001 — Claudelint
- IMPL-0001 — Phase 1 core linter (this investigation gates Phase 3)
- DESIGN-0001 — Architecture (see `internal/convert/` stub)
- OpenAI GPT Actions spec (OpenAPI): link TBD
- OpenAI Apps platform docs: link TBD
- OpenCode skill/plugin docs: link TBD
- Claude plugin, skill, hook references in this project's `.claude/`
