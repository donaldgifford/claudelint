---
title: Documentation
---

## Documentation

The same `docs/` tree feeds two rendered outputs:

- **Public site** — <https://claudelint.dev>, served by Astro + Starlight from
  `site/` and deployed via Cloudflare Pages.
- **Backstage TechDocs** — built by MkDocs from `mkdocs.yml` (managed by
  `docz update`) for internal/embedded consumption.

See
[DESIGN-0003](../design/0003-dual-output-docs-site-shared-source-mkdocs-for-techdocs.md)
for the dual-pipeline architecture. To preview the Starlight site locally:

```bash
just docs-install   # one-time
just docs-dev       # http://localhost:4321
```

`just docs-build` runs the full Astro build; `just docs-check` runs type +
content collection diagnostics. `just lint-md` enforces CommonMark + GFM only —
no MkDocs-only syntax (admonitions, collapsible blocks, or other
MkDocs extensions) so the same source renders in both pipelines.
