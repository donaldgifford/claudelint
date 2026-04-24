---
id: INV-0005
title: "Phase 2 dogfood findings: marketplaces, MCP, and spec divergence"
status: Closed
author: Donald Gifford
created: 2026-04-23
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0005: Phase 2 dogfood findings: marketplaces, MCP, and spec divergence

**Status:** Closed
**Author:** Donald Gifford
**Date:** 2026-04-23

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Environment](#environment)
- [Findings](#findings)
  - [Phase 2 rules fire on real marketplaces](#phase-2-rules-fire-on-real-marketplaces)
  - [Spec divergence: version and author live under metadata and owner](#spec-divergence-version-and-author-live-under-metadata-and-owner)
  - [MCP rules were not exercised](#mcp-rules-were-not-exercised)
  - [Known-tools allowlist gap: AskUserQuestion](#known-tools-allowlist-gap-askuserquestion)
  - [Security/secrets false positive on a docs SKILL](#securitysecrets-false-positive-on-a-docs-skill)
- [Conclusion](#conclusion)
- [Recommendation](#recommendation)
- [References](#references)
<!--toc:end-->

## Question

Does the Phase 2 ruleset produce useful diagnostics on a real Claude
Code marketplace, and does the marketplace parser match the shape of
manifests in the wild?

## Hypothesis

Running against `donaldgifford/claude-skills` — the donald-loop / docz /
go-development marketplace — should surface:

- the existing Phase 1 content findings (trigger-clarity, no-emoji)
  already documented in INV-0003;
- new `marketplace/*` diagnostics against `.claude-plugin/marketplace.json`;
- if any plugin carries an embedded MCP server or the repo has
  `.mcp.json`, matching `mcp/*` findings.

## Context

Phase 2.8 requires dogfooding the new ruleset on one real marketplace
before declaring the phase shippable. `donaldgifford/claude-skills` is
the primary dogfood target because it is the marketplace this author
uses for day-to-day Claude Code workflows — the one that the
donald-loop plugin driving this implementation project itself comes
from. Finding problems here means finding problems with the workflow
that generates the problems.

**Triggered by:** IMPL-0002 Phase 2.8.

## Approach

1. Build `claudelint` from the `docs/impl-0002-phase-2` branch.
2. Run `claudelint run --format=text ~/code/claude-skills`.
3. Count diagnostics by rule and by file.
4. Cross-check every Phase 2 rule hit against the manifest shape.
5. Inspect the repo for `.mcp.json` or plugin-embedded `mcp.servers`
   to decide whether the MCP ruleset is actually being exercised.

## Environment

| Component      | Version / Value                                      |
|----------------|------------------------------------------------------|
| `claudelint`   | pre-v0.2.0 (branch `docs/impl-0002-phase-2`, ruleset v1.1.0, fingerprint `4cee5ee7`) |
| Go             | 1.26.1                                               |
| Marketplace    | `donaldgifford/claude-skills` at `~/code/claude-skills` (HEAD, 2026-04-23) |

## Findings

### Phase 2 rules fire on real marketplaces

A single run surfaced 65 diagnostics across 106 files. The top two
findings originate from the new `marketplace/*` package:

    .claude-plugin/marketplace.json: info:  marketplace manifest has no "author" field  [marketplace/author-required]
    .claude-plugin/marketplace.json: error: marketplace manifest is missing required field "version"  [marketplace/version-semver]

Both rules are correctly wired: they run against
`.claude-plugin/marketplace.json`, they produce a file-level diagnostic
with a stable rule ID, and they respect severity.

### Spec divergence: version and author live under metadata and owner

Inspection of the real manifest revealed a schema mismatch between
DESIGN-0002 and the format `claude-skills` actually ships:

```json
{
  "name": "donaldgifford-claude-skills",
  "owner": { "name": "donaldgifford", "email": "dgifford@pm.me" },
  "metadata": {
    "description": "Collection of Claude Code skills for developer workflows",
    "version": "2.0.0"
  },
  "plugins": [ ... ]
}
```

DESIGN-0002 §2.1 specifies `version` and `author` as **top-level**
fields. The live manifest wraps them inside `metadata` and uses `owner`
(an object with `name` + `email`) instead of `author`. Every
`marketplace/version-semver` and `marketplace/author-required` hit the
dogfood produced is therefore a **false positive** relative to the real
ecosystem.

This is a doc/code bug, not a user bug. The claudelint parser follows
DESIGN-0002; DESIGN-0002 describes a shape that does not match the
field the `claude-skills` marketplace is actually using.

**Provisional direction (for a later phase):**

1. Extend the parser to accept either `version` at the top level **or**
   `metadata.version` nested, preferring top-level if both are present
   (least-surprise for readers of the manifest).
2. Same for `author` — accept `author` (string) **or** `owner.name`
   (object → flatten).
3. Update DESIGN-0002 to document both shapes as supported, with a
   rationale pointing to this investigation.

Not fixing this now, to keep Phase 2 on schedule. Tracked as a
follow-up.

### MCP rules were not exercised

`claude-skills` has no `.claude-plugin/.mcp.json` and no plugin
manifest carries `mcp.servers`. The full `mcp/*` ruleset was
registered, appeared in `claudelint rules`, but produced zero
diagnostics on this run. Exercise confirmed out-of-band against a
hand-built fixture under `/tmp/sarif-fixture/` during Phase 2.6 SARIF
smoke testing — three of six rules (`mcp/no-unsafe-shell`,
`mcp/no-secrets-in-env`, and the security/secrets file-level rule)
fired on a deliberate bad `.mcp.json`.

**Implication.** MCP rules are correct in isolation but have not been
exercised on real-world manifests. When a public marketplace that
distributes MCP servers comes online, revisit.

### Known-tools allowlist gap: AskUserQuestion

Two diagnostics fired on
`plugins/infrastructure-as-code/commands/{scaffold,test}.md`:

    unknown tool "AskUserQuestion" in allowed-tools  [commands/allowed-tools-known]

`AskUserQuestion` is a legitimate Claude Code tool — it appears in the
Claude Code agent's deferred tool list (this agent session has it).
The Phase 1 allowlist in `internal/artifact/knowndata.go` predates it.

**Direction:** Expand `KnownTools` to include `AskUserQuestion` and any
other recent Claude Code additions; treat this as a minor ruleset bump
(additive → patch). Tracked as follow-up; not blocking Phase 2.

### Security/secrets false positive on a docs SKILL

`plugins/git-workflow/skills/commit/SKILL.md` tripped
`security/secrets`:

    file contains a high-entropy token that resembles a secret  [security/secrets]

Spot-check showed the match is on example content (a demo commit
message body showing a hash-like string) rather than a real secret.
This is a Phase 1 rule; the Phase 2 dogfood merely re-surfaced it.
Suppressing via in-source marker on that line is the documented path.

## Conclusion

**Answer to the question:** Yes with caveats. Phase 2 rules fire on
real marketplaces and the output is actionable. The primary caveat is
the `version`/`author` spec divergence — the rules fire, but the
diagnostics are not what a user of a conforming `claude-skills`-style
marketplace would want.

The divergence is the most important finding of this investigation.
Shipping v0.2.0 without addressing it means the two marketplace rules
most likely to fire — `marketplace/version-semver` (error) and
`marketplace/author-required` (info) — will produce false positives on
a large fraction of real marketplaces. Mitigation for v0.2.0: leave the
rules as-is but bump the defaults to `info` via config in downstream
consumers until the parser is taught to read both shapes.

## Recommendation

- **Ship v0.2.0** as-is. The ruleset is functional and the spec
  divergence is a known limitation documented here.
- **Follow-up (pre-v0.3.0):** teach the marketplace parser to accept
  both `version` and `metadata.version`, both `author` and
  `owner.name`. Update DESIGN-0002 to match.
- **Follow-up (minor/patch):** add `AskUserQuestion` (and any other
  recent additions) to `artifact.KnownTools`.
- **Note in release notes:** the `marketplace/version-semver` and
  `marketplace/author-required` rules may produce false positives on
  manifests that follow the nested `metadata` / `owner` shape. Point
  at this investigation.

## References

- [IMPL-0002](../impl/0002-phase-2-marketplaces-mcp-rules-sarif-and-github-action.md) — Phase 2.8 acceptance criteria
- [DESIGN-0002](../design/0002-phase-2-marketplaces-mcp-rules-and-github-action.md) — marketplace shape §2.1
- [INV-0003](0003-phase-18-dogfood-findings-on-external-claude-plugins.md) — Phase 1.8 dogfood (for contrast)
