---
id: DESIGN-0003
title: "Dual-output docs site — shared source, MkDocs for TechDocs, Starlight for public"
status: Implemented
author: Donald Gifford
created: 2026-05-31
---
<!-- markdownlint-disable-file MD025 MD041 -->

# DESIGN 0003: Dual-output docs site — shared source, MkDocs for TechDocs, Starlight for public

**Status:** Implemented
**Author:** Donald Gifford
**Date:** 2026-05-31

<!--toc:start-->
- [Overview](#overview)
- [Goals and Non-Goals](#goals-and-non-goals)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Background](#background)
- [Detailed Design](#detailed-design)
  - [Source layout (unchanged)](#source-layout-unchanged)
  - [Build pipelines (two new + one existing)](#build-pipelines-two-new--one-existing)
  - [MkDocs side (existing — no changes)](#mkdocs-side-existing--no-changes)
  - [Starlight side (new)](#starlight-side-new)
  - [Build + deploy](#build--deploy)
  - [Frontmatter compatibility](#frontmatter-compatibility)
- [API / Interface Changes](#api--interface-changes)
- [Data Model](#data-model)
- [Testing Strategy](#testing-strategy)
- [Migration / Rollout Plan](#migration--rollout-plan)
- [Resolved Decisions](#resolved-decisions)
- [Open Questions](#open-questions)
- [References](#references)
<!--toc:end-->

## Overview

Build the public-facing claudelint docs site with [Starlight](https://starlight.astro.build/) (Astro-based) while keeping the existing MkDocs + Material setup intact for Backstage TechDocs consumption. Both generators read from the **same `docs/` source tree** so authors write Markdown once and the build pipelines decide how to render it.

## Goals and Non-Goals

### Goals

- Ship a public docs site with a modern look and feel that doesn't read as "another MkDocs Material site".
- Preserve the existing Backstage TechDocs integration (TechDocs embeds the MkDocs Python build pipeline; replacing MkDocs would break it).
- Maintain a **single source of truth** for documentation content: one `docs/` tree, one set of `.md` files, two outputs.
- Keep `docz` as the lifecycle tool — creating new RFC/ADR/DESIGN/IMPL/PLAN/INV docs continues to work unchanged.
- Avoid maintaining two parallel nav configs by relying on Starlight's filesystem-based sidebar auto-discovery (MkDocs nav stays driven by `docz update` writing `mkdocs.yml` as today).

### Non-Goals

- Replacing MkDocs entirely. MkDocs stays on disk because Backstage TechDocs requires it.
- Migrating to MkDocs 2.0 or Zensical now. MkDocs 1.x + Material is sufficient for the TechDocs feed and we don't want to chase an in-progress upstream.
- Versioned docs (Docusaurus-style snapshots per release tag). Out of scope until Phase 3+ when the CLI's stable API surface justifies it.
- A blog. Starlight doesn't ship one natively, and we have no immediate need.
- MDX / React components in `.md` files. Sticking to CommonMark + frontmatter keeps both renderers happy.

## Background

Three forces are pushing this decision:

1. **MkDocs 2.0 deliberately removes plugin support** ([squidfunk 2026-02-18](https://squidfunk.github.io/mkdocs-material/blog/2026/02/18/mkdocs-2.0/)). That breaks Material for MkDocs as a project and the wider plugin ecosystem. Material's maintainer is building **Zensical** as a drop-in replacement for MkDocs 1.x with plugin support preserved. So MkDocs 1.x is functional today but is no longer the active upstream path.
2. **"Material feels dated"** — the default Material skin is widely recognizable and looks ~2018-era to a designer's eye. For a public-facing CLI tool docs site, the first impression matters.
3. **Backstage TechDocs requires MkDocs**. TechDocs (the Backstage docs plugin) builds its index by invoking MkDocs on a repo's `mkdocs.yml`. If we want to stay discoverable inside a Backstage instance, we can't drop MkDocs entirely.

The combination means: keep MkDocs for TechDocs, add a modern generator for the public site, and don't duplicate the doc source.

Why Starlight over Docusaurus:

- **Simpler mental model.** Starlight is an Astro integration — one `starlight({ ... })` block in `astro.config.mjs`. Docusaurus is a React app with a docs plugin loaded, lots of plugins, MDX everywhere, sprawling TS config.
- **Filesystem-based sidebar discovery.** Drop a new `.md` file into `docs/`, Starlight picks it up. No second nav config to maintain alongside `mkdocs.yml`.
- **Plain Markdown by default.** MDX is opt-in. Our `docz`-generated files use CommonMark + frontmatter; Starlight reads them without translation.
- **Astro Islands** for selective interactivity if we ever need it, without committing to a full React app.

## Detailed Design

### Source layout (unchanged)

The existing `docs/` tree stays as-is. `docz` continues to scaffold new docs into `docs/<type>/NNNN-slug.md` with the established frontmatter shape.

```text
docs/
├── adr/                # ADR-0001, ADR-0002, ...
├── design/             # DESIGN-0001, DESIGN-0002, DESIGN-0003 (this doc), ...
├── impl/
├── investigation/
├── plan/
├── rfc/
└── README.md
```

### Build pipelines (two new + one existing)

```text
                    ┌──────────────────────────┐
                    │     docs/ (one tree)     │
                    │   .md + frontmatter      │
                    └────────────┬─────────────┘
                                 │
                  ┌──────────────┴──────────────┐
                  │                             │
                  ▼                             ▼
        ┌───────────────────┐         ┌─────────────────────┐
        │  mkdocs.yml       │         │  astro.config.mjs   │
        │  (docz-managed)   │         │  starlight({...})   │
        └────────┬──────────┘         └──────────┬──────────┘
                 │                               │
                 ▼                               ▼
        ┌───────────────────┐         ┌─────────────────────┐
        │  MkDocs + Material │         │  Astro + Starlight  │
        │  build pipeline   │         │  build pipeline     │
        └────────┬──────────┘         └──────────┬──────────┘
                 │                               │
                 ▼                               ▼
        ┌───────────────────┐         ┌─────────────────────┐
        │  TechDocs feed    │         │  claudelint.dev     │
        │  (Backstage)      │         │  (Cloudflare Pages) │
        └───────────────────┘         └─────────────────────┘
```

### MkDocs side (existing — no changes)

- `mkdocs.yml` continues to be written by `docz update` (the `wiki.auto_update: true` setting in `.docz.yaml`).
- TechDocs consumers continue to read `mkdocs.yml` and the `docs/` tree as today.
- No new tooling, no migration.

### Starlight side (new)

Add an Astro project alongside the existing repo:

```text
.
├── astro.config.mjs        # starlight({ sidebar: 'auto', ... })
├── package.json            # astro + @astrojs/starlight
├── docs/                   # SHARED with MkDocs (existing)
└── site/                   # Starlight wrapper: a thin Astro shell that
                            # mounts ../docs/ as content collection
```

The Starlight config uses **filesystem-based auto-discovery** for the sidebar, so authors don't update a second config when adding docs. Optional per-doc frontmatter (`sidebar_label`, `sidebar_order`) lets us override defaults when needed.

Markdown content is read directly from the existing `docs/` tree via Astro's content collections (or a `srcDir` pointed at `docs/`). The `<!--toc:start-->` blocks that docz emits are harmless to Starlight (it generates its own TOC); MkDocs uses them.

### Build + deploy

- **MkDocs**: stays manual / Backstage-driven. No new CI work.
- **Starlight**: new GitHub Actions workflow that runs `astro build` on push to `main` and publishes the `dist/` output to **Cloudflare Pages**, served from **claudelint.dev** (domain already registered on Cloudflare). PR builds produce preview deploys via the Cloudflare Pages GitHub integration.
- **Search**: Pagefind — Starlight's default, client-side index, zero ops. Good enough for v1; we can revisit Algolia DocSearch later if scale demands it.
- Both pipelines validate in CI on PRs touching `docs/**` to catch render failures before merge.

### Frontmatter compatibility

`docz`'s generated frontmatter (`id`, `title`, `status`, `author`, `created`) is plain YAML. MkDocs ignores unknown keys. Starlight uses `title` natively for the page heading; the rest are pass-through metadata we can surface via custom Astro components later if useful.

The single risk is **MkDocs-specific Markdown extensions** (e.g. `!!! note "title"` admonitions, `pymdownx` features). Starlight renders standard CommonMark + GFM. We resolve this by:

- Sticking to plain Markdown in new docs.
- Auditing existing docs for MkDocs-only syntax; convert to plain Markdown where found.
- Adding a CI check that runs `markdownlint-cli2` against the canonical CommonMark profile to prevent regression.

## API / Interface Changes

No public API or CLI surface changes. This is purely doc-site tooling.

- **`docz`**: no changes. Continues to write `mkdocs.yml`. Future enhancement (not in this design): teach `docz` to optionally emit a Starlight sidebar override file if filesystem auto-discovery isn't sufficient.
- **CONTRIBUTING / README**: add a brief note that the public docs site is built by Starlight from the same `docs/` tree; authors don't need to touch the Astro project.

## Data Model

Not applicable — no schema or storage changes.

## Testing Strategy

- **Per-PR**: CI workflow runs `astro check` + `astro build` on any PR touching `docs/**`, `astro.config.mjs`, `site/**`, or `package.json`. Build failure = PR red.
- **Markdown linting**: extend the existing `markdownlint-cli2` config to enforce CommonMark-compatible patterns repo-wide, catching MkDocs-only syntax at lint time.
- **Render parity smoke**: a small CI script renders a representative doc through both pipelines and asserts both succeed (no semantic diff — just build success).
- **Mermaid parity check**: pick one doc with a Mermaid block, render it through both pipelines, eyeball the output. MkDocs uses `pymdownx.superfences`; Starlight uses `@astrojs/markdown-remark` plugins. We want the same fenced-block syntax to render in both — if not, we normalize to whichever syntax both support and lint against deviation.
- **Manual review**: spot-check a handful of docs in the rendered Starlight site after first deploy to confirm look and feel.

## Migration / Rollout Plan

Phased on this `docs/site` branch:

1. **Scaffold Starlight** alongside the existing MkDocs setup. New `astro.config.mjs`, `package.json`, `site/` shell. `docs/` untouched.
2. **Wire content collection** to read from the shared `docs/` tree. Verify auto-discovery produces a sensible sidebar.
3. **Audit existing docs** for MkDocs-specific Markdown. Convert any non-CommonMark patterns. Add lint enforcement.
4. **CI workflow**: add Astro build job. Initially `fail-build: false` while we shake out parity issues; flip to required check once stable.
5. **Deploy target**: Cloudflare Pages, project bound to this repo's `main` branch. PR preview deploys via the Cloudflare GitHub integration.
6. **Cutover**: point `claudelint.dev` at the Starlight output via Cloudflare DNS. MkDocs / TechDocs feed continues to work unchanged.

Rollback is trivial: if Starlight breaks, the public site stays on the last-good deploy; nothing about MkDocs/TechDocs is touched.

## Resolved Decisions

- **Deploy target**: Cloudflare Pages. Domain already registered on Cloudflare; PR preview deploys are a meaningful quality-of-life win.
- **Domain**: `claudelint.dev` (already owned).
- **`docz` Starlight nav emission**: not now. Filesystem auto-discovery covers the case. Revisit only if we need fine-grained ordering that frontmatter overrides can't provide.
- **Search**: Pagefind for v1. Zero ops, client-side, ships with Starlight. Algolia DocSearch deferred.

## Open Questions

- **Mermaid render parity**: confirmed as a goal but the syntax compatibility between MkDocs (`pymdownx.superfences`) and Starlight (`@astrojs/markdown-remark`) still needs hands-on verification. Tracked in Testing Strategy as a parity check before declaring the migration complete.

## References

- [MkDocs 2.0 announcement — squidfunk, 2026-02-18](https://squidfunk.github.io/mkdocs-material/blog/2026/02/18/mkdocs-2.0/) — context for why MkDocs is no longer the active upstream path
- [Zensical](https://zensical.org/) — Material maintainer's MkDocs 1.x successor
- [Starlight docs](https://starlight.astro.build/) — Astro integration we're adopting
- [Backstage TechDocs](https://backstage.io/docs/features/techdocs/) — why MkDocs has to stay
- [Astro Content Collections](https://docs.astro.build/en/guides/content-collections/) — mechanism for reading shared `docs/` tree
- [Pagefind](https://pagefind.app/) — default Starlight search
- DESIGN-0001 — Phase 1 claudelint architecture (for the broader context this docs site documents)
- DESIGN-0002 — Phase 2 architecture
