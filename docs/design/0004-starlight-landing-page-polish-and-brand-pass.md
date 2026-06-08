---
id: DESIGN-0004
title: "Starlight landing page polish and brand pass"
status: Draft
author: Donald Gifford
created: 2026-06-08
---
<!-- markdownlint-disable-file MD025 MD041 -->

# DESIGN 0004: Starlight landing page polish and brand pass

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-06-08

<!--toc:start-->
- [Overview](#overview)
- [Goals and Non-Goals](#goals-and-non-goals)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Background](#background)
- [Detailed Design](#detailed-design)
  - [1. Landing page Hero + CardGrid](#1-landing-page-hero--cardgrid)
  - [2. Brand identity (color, logo, favicon)](#2-brand-identity-color-logo-favicon)
  - [3. Information architecture refinement](#3-information-architecture-refinement)
- [API / Interface Changes](#api--interface-changes)
- [Data Model](#data-model)
- [Testing Strategy](#testing-strategy)
- [Migration / Rollout Plan](#migration--rollout-plan)
- [Open Questions](#open-questions)
- [References](#references)
<!--toc:end-->

## Overview

DESIGN-0003 / IMPL-0003 shipped the dual-output docs site: Starlight on
Cloudflare for `https://claudelint.dev` and MkDocs for Backstage TechDocs,
both reading the shared root `docs/` tree. The structure is sound — the
remaining gap is **presentation**. The default Starlight chrome with a
prose-only index page reads like a wall of markdown, not a landing page
for a CLI tool. This design pass turns `/` into a real landing surface
(hero + cards + tabbed install), gives the site a small brand identity
(accent color, logo, favicon), and tunes the sidebar so the most
important content isn't buried under collapsed groups.

## Goals and Non-Goals

### Goals

- Replace the prose-only index with a scannable landing page using
  Starlight's built-in `hero` frontmatter, `CardGrid`, and `Tabs`
  components. Visitor should know what claudelint is and have an
  install command within ~3 seconds of arrival.
- Add a minimal but distinct brand identity: custom accent color,
  small SVG logo (mark + wordmark variants), and matching favicon.
  Goal is "this is claudelint", not "this is a Starlight template".
- Reshape the sidebar so architecture docs (DESIGN / ADR) are
  reachable in one click for curious users, while keeping the
  history-track docs (RFC / IMPL / Plan / Investigation) one level
  deeper for contributors.
- Keep the MkDocs / TechDocs pipeline working — every change must
  stay in the shared CommonMark+GFM authoring contract from
  DESIGN-0003.

### Non-Goals

- No content rewrite of the existing rule reference, ADRs, DESIGN
  docs, or other in-tree material.
- No new search backend (Pagefind stays).
- No multi-language / i18n.
- No custom Starlight overrides beyond `<Hero>`, `<CardGrid>`,
  `<LinkCard>`, `<Tabs>`, and the `<head>` slot for favicons.
  Components ship with Starlight — we're using, not authoring, them.

## Background

What's live today (commit `37b99cd` on `docs/site`, PR
[#40](https://github.com/donaldgifford/claudelint/pull/40)):

- Default Starlight 0.39 chrome (purple accent, Inter typography).
- Sidebar: Install → Rules → Development (collapsed, six docz subgroups
  nested) → Reference → Changelog.
- `docs/index.md` is a long prose page with H2 sections for Install,
  Quickstart, Output formats, Linted artifacts, What's in the box,
  Where to go next.
- README-as-Overview pattern: each docz README is `order: 0, label:
  Overview` so the first child of every group is a clickable index.
- Header social row: `cloud-download` → `/releases/latest`, then
  `github` → repo.

What still feels off:

- The landing page reads as documentation, not as a tool's home page.
  Compare `https://ruff.rs`, `https://biome.dev`,
  `https://golangci-lint.run` — every one of those leads with a hero
  + tagline + install command + 3-5 feature cards.
- No project identity. Default purple, no logo, no favicon — visitors
  see a Starlight template, not claudelint.
- The Development group hides six subgroups behind one collapse, which
  buries the two most useful surfaces (DESIGN docs and ADRs) for
  anyone trying to understand the architecture.

## Detailed Design

Three independent workstreams. Each is roughly half a day of work and
can ship in its own commit; the goal is incremental polish, not a big
bang.

### 1. Landing page Hero + CardGrid

Replace `docs/index.md` body with Starlight's `hero` frontmatter and
component imports. The page becomes:

```text
+------------------------------------------------------+
| HERO                                                 |
|   title:    claudelint                               |
|   tagline:  A static linter for Claude Code...       |
|   actions:  [ Quickstart -> ] [ GitHub -> ]          |
|   image:    /logo.svg (right side, ~280px)           |
+------------------------------------------------------+
| Install one-liner (Tabs)                             |
|   [ go install ] [ Docker ] [ Releases ] [ Action ]  |
+------------------------------------------------------+
| CardGrid (2x2)                                       |
|   [ Built-in ruleset ]  [ HCL config           ]    |
|   [ SARIF + CI       ]  [ Suppression model    ]    |
+------------------------------------------------------+
| LinkCard row -> Quickstart / Rules / Changelog       |
+------------------------------------------------------+
```

Concretely:

- Move the existing Install / Quickstart / Output formats / Linted
  artifacts / What's in the box prose into a new
  `docs/install/quickstart.md`-adjacent page (or extend the existing
  Quickstart). The landing page should not duplicate them in long form.
- New `hero` block uses `actions[]` for `Quickstart` (primary, links to
  `/install/quickstart/`) and `GitHub` (secondary, external).
- `Tabs` component for Install, four panels: `go install`, Docker
  (`docker run --rm -v $PWD:/work ghcr.io/donaldgifford/claudelint:latest run /work`),
  GitHub Releases (one-line `curl | tar` style), GitHub Action
  (`uses: donaldgifford/claudelint-action@v1`).
- `CardGrid` of 4 `Card` components for the headline features:
  Built-in ruleset, HCL config, SARIF + CI, Suppression model. Each
  card is 2-3 sentences max, with a "Learn more →" link to the
  matching deep page.
- `LinkCard` row at the bottom for the three highest-traffic next
  steps: Quickstart, Rules reference, Changelog.

Markdown caveat: `<Hero>`, `<CardGrid>`, `<Card>`, `<Tabs>`,
`<LinkCard>` are MDX-only — MkDocs cannot render them. Either:

- (a) Make `docs/index.md` a `.mdx` for Starlight, and ship a separate
  `docs/index.md` (or skip the index entirely in MkDocs nav) for
  TechDocs. The shared-source contract holds for every other page;
  this is a single deliberate exception for the home page.
- (b) Keep `.md` and degrade gracefully — MDX components become raw
  text in MkDocs. Worse UX for TechDocs users but no fork.

Pick (a). Backstage TechDocs viewers are internal Donald-and-future-
contributors; they will rarely visit the index page (they land via
the nav). Public visitors hit `claudelint.dev/`. Optimize the public
surface; let TechDocs render a minimal text-only home.

### 2. Brand identity (color, logo, favicon)

Three small artifacts. All can ship behind one PR / commit.

**Accent color.** Override Starlight's `--sl-color-accent` in
`site/src/styles/custom.css`. Candidate palette: a sharp teal
(`#06b6d4` / cyan-500 range) signals "linter / scanner" without being
yet-another-purple. Add a `colorScheme` override in
`astro.config.mjs`'s `starlight()` call to point at the CSS.

**Logo.** Small SVG mark — a stylized check / bracket motif that
evokes "lint" + "Claude Code". Two variants:

- `site/public/logo.svg` — mark + wordmark, ~280px wide, used by the
  hero image.
- `site/public/logo-mark.svg` — square mark only, used by favicon
  generation.

Authoring approach: hand-build in Figma or Inkscape; commit the
SVG. Do not depend on a runtime SVG library.

**Favicon.** `site/public/favicon.svg` (modern browsers) + a 32x32
PNG fallback. Add via Starlight's `head` config so it gets injected
into every page.

Stretch (out of scope for v1): OG image (`og.png`) for link previews.
Track as a follow-up.

### 3. Information architecture refinement

Current sidebar buries DESIGN and ADRs. Two options:

**Option A — Promote Architecture.** Make Architecture a top-level
sidebar group containing the README/Overview for DESIGN and ADRs.
Leave RFC / IMPL / Plan / Investigation under Development as a
"History" subgroup.

```text
- Install
- Rules
- Architecture        <-- NEW
  - DESIGN
  - ADRs
- Development
  - Docs / Profiling guides
  - History
    - RFCs
    - Implementation
    - Plans
    - Investigations
- Reference
- Changelog
```

**Option B — Status quo.** Keep the single Development collapse.
Lower-friction; the README-as-Overview pattern already gives every
group a clickable landing.

Recommendation: **Option A**. The cost is one autogenerate entry in
`astro.config.mjs` and a re-shuffle of the existing groups. The
benefit is that DESIGN-0001 / DESIGN-0002 / DESIGN-0003 / ADR-0001 —
the documents that explain *why* the tool is the way it is — are
visible without expanding a tree. RFC / IMPL / Plan / INV are
process artifacts; second-level nesting is fine for those.

## API / Interface Changes

None. Pure docs-site work; no Go binary, CLI, or config surface
changes.

## Data Model

No persisted state changes. New SVG / CSS assets land in
`site/public/` and `site/src/styles/`. The `astro.config.mjs`
sidebar shape changes per §3.

## Testing Strategy

- `just docs-check` (`astro check`) must stay green — catches content
  schema regressions on the new `.mdx` index.
- `just docs-build` (Starlight build) must stay green.
- `just docs-mkdocs-check` must stay green — verifies the TechDocs
  side still renders even though the index degrades.
- `just lint-md` must stay green — the new index file (whether `.md`
  or `.mdx`) still has to pass markdownlint where applicable.
- Manual: Cloudflare Pages preview deploy must show the new hero +
  cards rendering correctly on desktop and mobile.
- Manual: Lighthouse pass on the preview (target: 90+ accessibility;
  CardGrid + Hero must not regress contrast against the new accent).

## Migration / Rollout Plan

Three commits on a new branch `docs/landing-polish` (or sequenced
directly on `docs/site` if PR #40 is still open):

1. **Landing page** — convert `docs/index.md` to `docs/index.mdx`,
   wire Hero + CardGrid + Tabs + LinkCards. Add MkDocs-side
   placeholder index if needed.
2. **Brand pass** — commit `logo.svg`, `logo-mark.svg`, `favicon.svg`,
   `custom.css` with accent override; wire into `astro.config.mjs`.
3. **Sidebar reshape** — promote Architecture, re-nest History under
   Development.

Each commit is independently revertable. None of them require
Cloudflare dashboard changes — the existing preview-on-push pipeline
covers verification.

Defer until after PR #40 merges if Phase 5 Cloudflare DNS cutover
hasn't happened yet — better to land a polished `claudelint.dev`
launch than to ship default chrome and polish later.

## Open Questions

1. **Logo design**: hire it out, generate via AI tool, or hand-draw?
   Smallest viable: a single-glyph wordmark in a monospace font with a
   custom color. Decide before starting §2.
2. **Accent color**: cyan-500 (`#06b6d4`) is the recommendation; alt
   candidates are emerald-500 (`#10b981`) or amber-500 (`#f59e0b`).
   Pick before §2.
3. **`.mdx` vs `.md` for index**: §1 recommends `.mdx`. Confirm
   acceptable to ship a single MkDocs/TechDocs degradation on the
   home page only.
4. **Architecture group**: confirm Option A from §3. If the answer is
   "leave it as Development", §3 collapses to a no-op.

## References

- [DESIGN-0003: Dual-output docs site](./0003-dual-output-docs-site-shared-source-mkdocs-for-techdocs.md)
- [IMPL-0003: Phase 3 implementation tracker](../impl/0003-phase-3-dual-output-docs-site-with-starlight.md)
- [PR #40 — dual-output docs site](https://github.com/donaldgifford/claudelint/pull/40)
- [Starlight Hero docs](https://starlight.astro.build/reference/frontmatter/#hero)
- [Starlight components — CardGrid / Tabs / LinkCard](https://starlight.astro.build/components/card-grids/)
- Comparable landing pages: [ruff.rs](https://ruff.rs),
  [biome.dev](https://biome.dev),
  [golangci-lint.run](https://golangci-lint.run)
