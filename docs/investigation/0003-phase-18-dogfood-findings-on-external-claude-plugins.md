---
id: INV-0003
title: "Phase 1.8 dogfood findings on external Claude plugins"
status: Closed
author: Donald Gifford
created: 2026-04-21
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0003: Phase 1.8 dogfood findings on external Claude plugins

**Status:** Closed
**Author:** Donald Gifford
**Date:** 2026-04-21

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Environment](#environment)
- [Findings](#findings)
  - [Discovery: plugin layouts without .claude/ were invisible](#discovery-plugin-layouts-without-claude-were-invisible)
  - [Plugin A: donaldgifford-claude-skills/go-development@2.0.1](#plugin-a-donaldgifford-claude-skillsgo-development201)
  - [Plugin B: donaldgifford-claude-skills/docz](#plugin-b-donaldgifford-claude-skillsdocz)
- [Conclusion](#conclusion)
- [Recommendation](#recommendation)
- [References](#references)
<!--toc:end-->

## Question

Does the Phase 1 MVP of `claudelint` produce useful diagnostics on
real Claude plugin distributions, and are there gaps in discovery
that prevent it from finding artifacts in the wild?

## Hypothesis

Running against the plugin cache under `~/.claude/plugins/cache/`
should surface:

- a small number of skills/commands/agents per plugin
- low-severity content warnings (trigger-clarity, no-emoji) but no
  schema errors

## Context

Phase 1.8 of the implementation plan requires dogfooding on at least
two external plugin repos before declaring the MVP complete. The goal
is to surface real-world usability issues that synthetic fixtures
cannot.

**Triggered by:** IMPL-0001 Phase 1.8.

## Approach

1. Run `claudelint run <plugin>` against two plugins extracted to the
   local Claude plugins cache.
2. Note every file discovered and every diagnostic surfaced.
3. Compare against the plugin source tree to see if any artifacts were
   missed.
4. Fix any discovery gaps before repeating.

## Environment

| Component        | Version / Value |
|------------------|-----------------|
| `claudelint`     | pre-v0.1.0 (branch `docs/claudelint-linter-rfc`) |
| Go               | 1.26.1          |
| Plugin cache     | `~/.claude/plugins/cache/donaldgifford-claude-skills/` |

## Findings

### Discovery: plugin layouts without `.claude/` were invisible

First run against `go-development/2.0.1`:

    $ claudelint run ~/.claude/plugins/cache/donaldgifford-claude-skills/go-development/2.0.1
    0 diagnostics, 1 files checked

Only the plugin's top-level `README.md` was inspected; the eight files
under `skills/`, `commands/`, and `agents/` were silently ignored.

**Root cause.** `internal/discovery/classify.go` required every
`skills/`, `commands/`, `agents/`, and `hooks/` path to live below a
`.claude/` segment. That assumption holds for user repositories
(`.claude/skills/foo/SKILL.md`) but breaks for plugin distributions,
where the plugin IS the `.claude/` root and its kind directories sit
at the plugin top level.

**Fix.** Added a `classifyPluginLayout` fallback that matches the
plugin-root shape (`.../skills/foo/SKILL.md`,
`.../commands/foo.md`, etc.) when no `.claude/` parent is present.
Re-run of the same command:

    $ claudelint run ~/.claude/plugins/cache/donaldgifford-claude-skills/go-development/2.0.1
    agents/go-style.md: info: artifact contains emoji [style/no-emoji]
    commands/review.md: info: artifact contains emoji [style/no-emoji]
    commands/scaffold.md: info: artifact contains emoji [style/no-emoji]
    skills/go/SKILL.md:3:1: warning: skill description has no trigger phrase [skills/trigger-clarity]
    4 diagnostics, 8 files checked

### Plugin A: `donaldgifford-claude-skills/go-development@2.0.1`

- **Files checked:** 8 (3 commands, 1 agent, 1 skill, 1 CLAUDE.md-equivalent, 1 plugin manifest, 1 README).
- **Diagnostics:** 4 total.
  - 3× `style/no-emoji` (info) on agents and commands.
  - 1× `skills/trigger-clarity` (warning) on the top-level Go skill's
    `description:` — the plugin does not use a "Use when …" phrase.
- No schema errors. The plugin is structurally valid.

### Plugin B: `donaldgifford-claude-skills/docz`

- **Files checked:** 27 (across two versions: `1.1.1/` and `1.2.0/`).
- **Diagnostics:** 12 total.
  - 12× `skills/trigger-clarity` (warning) — every skill description
    across both versions is missing a "Use when …" phrase.
- Again no schema errors. The finding is a real, actionable content
  concern: docz skills would benefit from clearer activation triggers.

## Conclusion

**Answer:** Yes, claudelint produces useful diagnostics on real
plugins. The dogfood run surfaced exactly one structural gap
(classifier missing plugin-root layouts) and a consistent content
finding (`skills/trigger-clarity` on every surveyed plugin) that an
upstream PR could address.

The classifier gap is the kind of issue dogfooding is designed to
expose. Left in place, it would have meant claudelint worked on its
own fixtures but failed silently on 100% of real-world plugin
distributions — the opposite of the tool's purpose.

## Recommendation

- **Landed:** Extend `internal/discovery/classify.go` with a
  plugin-root fallback (this investigation's commit).
- **Follow-up:** Open issues on `donaldgifford-claude-skills/docz`
  (or submit a PR) proposing "Use when …" trigger phrases on every
  skill description.
- **Follow-up:** Add the plugin-layout cases to the Phase 2 broader
  dogfood matrix when more external plugins come online.

## References

- [IMPL-0001](../impl/0001-phase-1-core-linter-for-claudemd-skills-plugins-and-hooks.md) — Phase 1.8 acceptance criteria
- [DESIGN-0001](../design/0001-claudelint-linter-architecture-and-rule-engine.md) — discovery-model section
- Commit adding `classifyPluginLayout`
