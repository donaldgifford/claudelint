---
id: IMPL-0003
title: "Phase 3 — Dual-output docs site with Starlight"
status: Draft
author: Donald Gifford
created: 2026-05-31
---
<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL 0003: Phase 3 — Dual-output docs site with Starlight

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-05-31

<!--toc:start-->
- [Objective](#objective)
- [Scope](#scope)
  - [In Scope](#in-scope)
  - [Out of Scope](#out-of-scope)
- [Implementation Phases](#implementation-phases)
  - [Phase 1 — Scaffold the Starlight project](#phase-1--scaffold-the-starlight-project)
    - [Tasks](#tasks)
    - [Success Criteria](#success-criteria)
  - [Phase 2 — Wire shared docs/ source into Starlight](#phase-2--wire-shared-docs-source-into-starlight)
    - [Tasks](#tasks-1)
    - [Success Criteria](#success-criteria-1)
  - [Phase 3 — Markdown audit + CommonMark lint enforcement](#phase-3--markdown-audit--commonmark-lint-enforcement)
    - [Tasks](#tasks-2)
    - [Success Criteria](#success-criteria-2)
  - [Phase 4 — CI workflow for Starlight builds](#phase-4--ci-workflow-for-starlight-builds)
    - [Tasks](#tasks-3)
    - [Success Criteria](#success-criteria-3)
  - [Phase 5 — Cloudflare Pages deploy + PR previews](#phase-5--cloudflare-pages-deploy--pr-previews)
    - [Tasks](#tasks-4)
    - [Success Criteria](#success-criteria-4)
  - [Phase 6 — Cutover to claudelint.dev + polish](#phase-6--cutover-to-claudelintdev--polish)
    - [Tasks](#tasks-5)
    - [Success Criteria](#success-criteria-5)
- [File Changes](#file-changes)
- [Testing Plan](#testing-plan)
- [Dependencies](#dependencies)
- [Resolved Decisions](#resolved-decisions)
- [References](#references)
<!--toc:end-->

## Objective

Implement the dual-output docs site: keep the existing MkDocs setup intact for Backstage TechDocs and add an Astro + Starlight pipeline that renders the same `docs/` tree as a modern public site served from `https://claudelint.dev` via Cloudflare Pages.

**Implements:** [DESIGN-0003](../design/0003-dual-output-docs-site-shared-source-mkdocs-for-techdocs.md) — refer there for architecture, deploy decisions (Cloudflare Pages + Pagefind + claudelint.dev), and rollback semantics.

## Scope

### In Scope

- New `site/` directory housing the Astro + Starlight project (`astro.config.mjs`, `package.json`, Astro shell pages).
- Starlight integration reads from the shared root `docs/` tree (no doc relocation).
- Node toolchain pinned via `mise.toml`; `package-lock.json` committed.
- CommonMark audit of existing docs; `markdownlint-cli2` config extended to block MkDocs-only syntax.
- New `.github/workflows/docs.yml` running `astro build` on PRs and pushes to `main` that touch `docs/**`, `site/**`, `astro.config.mjs`, `package.json`, or `package-lock.json`.
- Cloudflare Pages project bound to this repo's `main` branch with PR preview deploys.
- DNS cutover: `claudelint.dev` → Cloudflare Pages project.
- Mermaid render parity verified across MkDocs and Starlight.
- README and CONTRIBUTING updated to document the dual-pipeline workflow.
- CLAUDE.md updated with the new `site/` directory and docs commands.

### Out of Scope

- Replacing or migrating MkDocs (Backstage requirement; explicit non-goal in DESIGN-0003).
- Versioned docs, blog, or MDX/React components (deferred per DESIGN-0003 non-goals).
- Teaching `docz` to emit a Starlight nav config (deferred; filesystem auto-discovery covers v1).
- Algolia DocSearch (Pagefind only for v1).
- Migration to MkDocs 2.0 or Zensical (future ADR/RFC).

---

## Implementation Phases

Each phase builds on the previous one. A phase is complete when all its tasks are checked off and its success criteria are met.

---

### Phase 1 — Scaffold the Starlight project

Stand up a working Astro + Starlight skeleton in `site/` that anyone can run locally with `npm run dev` and see the default Starlight landing page. No content wiring yet — the goal is to prove the toolchain works.

#### Tasks

- [x] Pin Node LTS in `mise.toml` (`# renovate:` annotation included so Renovate tracks it).
- [x] Create `site/` directory.
- [x] Run `npm create astro@latest site -- --template starlight --yes --no-install` (or equivalent) inside the worktree.
- [x] Move/rewrite generated files so the project lives at `site/` (Astro project root) with `package.json` at `site/package.json`.
- [x] Add `.gitignore` entries: `site/node_modules/`, `site/dist/`, `site/.astro/`, `site/.astro-cache/`.
- [x] Pin Astro + `@astrojs/starlight` in `site/package.json` with caret ranges; commit `site/package-lock.json`.
- [x] Add a Renovate config entry (or rely on the default `npm` manager) so Astro/Starlight bumps land as PRs. *(default npm manager handles it per Q8)*
- [x] Add npm scripts: `npm run dev`, `npm run build`, `npm run preview`, `npm run check`.
- [x] Run `cd site && npm install && npm run dev` locally to verify the default Starlight site renders at `http://localhost:4321`. *(verified via `npm run check` + `npm run build`; dev server starts cleanly)*
- [x] Add a `justfile` recipe (`just docs-dev`) that wraps `cd site && npm run dev` so contributors don't have to remember the directory.

#### Success Criteria

- `just docs-dev` (or `cd site && npm run dev`) serves the default Starlight site on `http://localhost:4321` with no errors.
- `cd site && npm run build` produces a `site/dist/` directory with HTML.
- `cd site && npm run check` (Astro's type-check) passes.
- No Node-toolchain artifacts (`node_modules/`, `dist/`) are committed.

---

### Phase 2 — Wire shared `docs/` source into Starlight

Configure Starlight to read Markdown from the existing root `docs/` tree, not from `site/src/content/docs/`. The sidebar should auto-discover docs and group them by type so the navigation is sensible without a hand-maintained config.

#### Tasks

- [x] Configure Astro's content collection (or `srcDir`) in `astro.config.mjs` to point at `../docs/` from `site/`. *(uses `glob({ pattern, base: '../docs' })` from `astro/loaders` — Starlight's `docsLoader()` is hardcoded to `src/content/docs/`)*
- [x] Configure Starlight's `sidebar` for filesystem auto-discovery, grouping by directory (`rfc/`, `adr/`, `design/`, `impl/`, `plan/`, `investigation/`). *(Starlight v0.39 requires `items: [{ autogenerate }]` wrapping)*
- [x] Verify each doc type's README renders as the group landing page. *(renders at `/<type>/readme/`; sidebar autogenerate exposes it)*
- [x] Verify all existing RFC/ADR/DESIGN/IMPL/PLAN/INV docs are reachable from the sidebar.
- [x] Confirm `<!--toc:start-->` / `<!--toc:end-->` blocks (docz-generated) don't render as visible content in Starlight. *(HTML comments pass through, invisible)*
- [x] Confirm relative cross-doc links (e.g. `[DESIGN-0001](../design/0001-...)` from an IMPL doc) resolve correctly in Starlight. *(custom remark plugin `site/src/plugins/remark-md-link-rewriter.mjs` rewrites `.md` relative links to absolute Starlight routes)*
- [x] Add a `docs/index.md` landing page (or wire Starlight to use the existing `docs/README.md`) as the public site homepage. *(existing `docs/index.md` + minimal `title:` frontmatter)*
- [x] Add `editLink` config so each doc page links to its GitHub source for "Edit on GitHub".
- [x] Spot-check one doc per type by browsing locally; fix any render issues. *(verified via build output + grep on rendered HTML)*

#### Success Criteria

- Sidebar shows all six doc-type groups (RFC, ADR, DESIGN, IMPL, PLAN, Investigation) with all docs listed under each.
- Clicking any doc renders its Markdown content with frontmatter title as the H1 and the body rendered correctly.
- The site homepage (root `/`) shows the landing page content (not a 404).
- "Edit on GitHub" link on a doc page resolves to the correct file in the repo.
- No relative-link breakage observed during spot checks.

---

### Phase 3 — Markdown audit + CommonMark lint enforcement

Sweep the existing `docs/` tree for MkDocs-specific Markdown syntax (`!!! note`, `???`, `pymdownx` features) that Starlight won't render. Convert any findings to CommonMark + GFM. Then extend `markdownlint-cli2` to block reintroduction at lint time.

#### Tasks

- [x] `grep -RE '^\s*!!!|^\s*\?\?\?' docs/` — flag admonitions. *(no hits)*
- [x] `grep -R 'pymdownx' docs/` — flag pymdownx-specific syntax. *(no hits)*
- [x] Convert admonitions to standard `> [!NOTE]` / `> [!WARNING]` (GFM) or plain blockquotes. *(none needed)*
- [x] Convert any tabs/snippets to plain Markdown. *(none present)*
- [ ] Verify Mermaid fenced blocks (` ```mermaid ... ``` `) render in both MkDocs and Starlight after picking a plugin (see Phase 4). *(deferred to Phase 4)*
- [x] Update `.markdownlint.json` / `.markdownlint-cli2.yaml` config to enforce CommonMark + GFM only (no MkDocs extensions). *(`.markdownlint.yaml` + custom grep checks in `just lint-md`; only legitimate relaxations are MD025 `front_matter_title: ""` for docz title convention and MD024 `siblings_only: true` for repeated phase sub-headings)*
- [x] Add `just lint-md` recipe (or fold into `just lint`) running `markdownlint-cli2` on `docs/**/*.md`. *(runs markdownlint plus three regex grep checks for `!!!`, `???`, and `pymdownx`)*
- [x] Run `markdownlint-cli2 --fix` on `docs/`; commit any auto-fixed nits.
- [x] Manually fix any remaining violations. *(MD040 language hints, MD034 bare URLs, MD038 space-in-code, MD046 mixed code-block style, MD051 broken anchors, MD018 `#issue` at line start — fixed in source rather than disabling rules)*

#### Success Criteria

- `grep -RE '^\s*!!!|^\s*\?\?\?|pymdownx' docs/` returns zero hits.
- `markdownlint-cli2 'docs/**/*.md'` exits 0 with the CommonMark profile.
- All existing docs render in the local Starlight dev server with no visible syntax artifacts.
- MkDocs still builds locally (`mkdocs build --strict`) after the syntax conversions.

---

### Phase 4 — CI workflow for Starlight builds

Add a new GitHub Actions workflow that runs `astro check` + `astro build` on PRs and pushes to `main` that touch the docs surface. Start non-blocking so it doesn't gate work while we shake out parity issues, then flip to required after two clean PRs.

#### Tasks

- [x] Create `.github/workflows/docs.yml`.
- [x] Triggers: `push` to `main` and `pull_request` to `main` with `paths` filter on `docs/**`, `site/**`, `astro.config.mjs`, `package.json`, `package-lock.json`, `.github/workflows/docs.yml`. *(paths cover docs/, site/, .markdownlint.yaml, justfile, mise.toml, the workflow itself; astro config + package.json live under site/)*
- [x] Steps: checkout, `actions/setup-node@v6` reading version from `mise.toml` (or `package.json` `engines`), `npm ci`, `npm run check`, `npm run build`. *(uses `jdx/mise-action@v4` to pin Node from `mise.toml` — single source of truth — rather than a separate setup-node step)*
- [x] Cache `~/.npm` and `site/node_modules` to keep CI fast. *(actions/cache@v4 keyed on package-lock.json)*
- [x] Upload `site/dist/` as a workflow artifact for inspection on failure. *(actions/upload-artifact@v4, 7-day retention)*
- [x] Add a markdown-lint step (`markdownlint-cli2` on `docs/**/*.md`). *(separate `lint-md` job running `just lint-md`)*
- [x] Also: extend `.github/labeler.yml` so PRs touching `site/**` get the `documentation` label.
- [ ] First-pass: do **not** add to branch protection. Observe two PRs running clean.
- [ ] After two clean runs: add `Docs / build` (or chosen job name) as a required status check.
- [x] Update CLAUDE.md "Git / PR conventions" section to mention the docs check. *(added in 6fdacb2 — `Docs CI` bullet in CLAUDE.md L126)*

#### Success Criteria

- A PR that adds or edits any file under `docs/` runs the new workflow and surfaces build success/failure.
- A PR that touches only Go code skips the workflow (paths filter working).
- `npm run check` and `npm run build` both pass for the current `main` state.
- After two clean runs, the workflow is a required check in branch protection.

---

### Phase 5 — Cloudflare Pages deploy + PR previews

Stand up the Cloudflare Pages project, wire it to the repo, configure build settings, and verify production + preview deploys work end-to-end. At this point the site is live at the Pages-assigned subdomain (e.g. `claudelint.pages.dev`), not yet at `claudelint.dev`.

> **Blocked on Donald.** Every task below requires Cloudflare dashboard access and cannot be performed from a code-only session. Donald to walk through these in the Cloudflare UI; once the project is up, mark each box and re-enter the loop for the DNS cutover in Phase 6.

#### Tasks

- [ ] In Cloudflare dashboard: create a Pages project named `claudelint` bound to the `donaldgifford/claudelint` GitHub repo.
- [ ] Configure build settings: framework preset = Astro, build command = `cd site && npm install && npm run build`, build output directory = `site/dist`, root directory = repo root.
- [ ] Set Node version env var in Cloudflare Pages to match `mise.toml` Node pin (currently `22.20.0`).
- [ ] Set production branch to `main`.
- [ ] Enable preview deployments for all branches with open PRs.
- [ ] Trigger first deploy by pushing a no-op to `main` (or by manually deploying); verify the Pages-assigned subdomain works.
- [ ] Open a test PR that touches a doc; verify Cloudflare Pages auto-comments with the preview URL.
- [ ] Verify Pagefind search works on the deployed preview (Cloudflare may need an extra build step to run `pagefind`; Starlight handles this by default but double-check).
- [ ] Save the Cloudflare Pages project ID and account ID as repo secrets (`CLOUDFLARE_ACCOUNT_ID`, `CLOUDFLARE_API_TOKEN`) in case we want a custom deploy workflow later.

#### Success Criteria

- `https://claudelint.pages.dev` (or assigned Pages subdomain) serves the rendered docs site.
- A PR touching `docs/` produces an automatic preview deploy with a unique URL.
- Pagefind search returns results for queries like "MCP" or "marketplace" against the deployed site.
- No build errors in the Cloudflare Pages build log for the latest production deploy.

---

### Phase 6 — Cutover to claudelint.dev + polish

Point `claudelint.dev` at the Cloudflare Pages project, verify HTTPS, and do final documentation/onboarding work so future contributors know about the dual-pipeline setup.

#### Tasks

- [ ] In Cloudflare DNS for `claudelint.dev`: add the Pages project as a custom domain (Cloudflare handles cert provisioning automatically). *(blocked on Phase 5 + dashboard access)*
- [ ] Verify `https://claudelint.dev` resolves to the Starlight site with a valid cert. *(blocked on Phase 5)*
- [ ] Test redirect behavior: HTTP → HTTPS, `www.claudelint.dev` → `claudelint.dev` (or whichever direction you prefer). *(blocked on Phase 5)*
- [x] Confirm `mkdocs build --strict` still passes against current `docs/`. *(passes via new `just docs-mkdocs-check` recipe — runs `mkdocs build --strict -d build/mkdocs` via uvx; fixed one pre-existing broken cross-link in `impl/0001` (`../../RELEASE.md` → absolute GitHub URL) along the way; documented `site_dir` collision gotcha in CLAUDE.md)*
- [ ] Mermaid parity check: pick (or create) a doc with a Mermaid block, render it via both MkDocs and Starlight, compare visually. *(deferred — no Mermaid blocks in `docs/` today; revisit when one lands)*
- [x] Update `README.md` with a "Documentation" section linking to `https://claudelint.dev`.
- [x] Update `CONTRIBUTING.md` (or create one) explaining: docs source = `docs/`; MkDocs and Starlight both read from there; `just docs-dev` for local Starlight preview; `mkdocs serve` for local MkDocs preview. *(created)*
- [x] Update `CLAUDE.md` with: the `site/` directory, the new `just docs-dev` recipe, the dual-output design pointer, and a note that the markdown lint config enforces CommonMark.
- [x] Update IMPL-0003 phase checkboxes as work completes; flip status to "Implemented" when all phases are done. *(checkboxes current; final status flip waits on Phase 5)*
- [ ] Move DESIGN-0003 status from "Draft" → "Implemented". *(waits on Phase 5 — code side complete, deploy pending)*

#### Success Criteria

- `https://claudelint.dev` serves the docs site with a valid TLS cert and acceptable load time.
- MkDocs `mkdocs build --strict` still passes; TechDocs feed is unaffected.
- Mermaid parity check passes (same fenced syntax renders in both pipelines).
- README links to `https://claudelint.dev`.
- CONTRIBUTING and CLAUDE.md document the dual-pipeline workflow.
- DESIGN-0003 + IMPL-0003 status moved to Implemented.

---

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `site/astro.config.mjs` | Create | Astro + Starlight integration config; points content collection at `../docs/`; configures `editLink`, sidebar auto-discovery, Mermaid plugin. |
| `site/package.json` | Create | Astro + `@astrojs/starlight` + Mermaid plugin pinned with caret ranges; npm scripts (`dev`, `build`, `preview`, `check`). |
| `site/package-lock.json` | Create | Committed lockfile for reproducible installs. |
| `site/src/content.config.ts` | Create | Astro content collections config (if needed alongside `srcDir` approach). |
| `site/src/pages/` | Create | Astro shell pages: index landing page (if not using `docs/README.md`). |
| `.gitignore` | Modify | Add `site/node_modules/`, `site/dist/`, `site/.astro/`. |
| `mise.toml` | Modify | Pin Node LTS with Renovate annotation. |
| `.github/workflows/docs.yml` | Create | PR + push CI workflow for Astro build + markdownlint. |
| `.markdownlint.json` (or `.markdownlint-cli2.yaml`) | Modify | Enforce CommonMark + GFM; disallow MkDocs-specific syntax. |
| `justfile` | Modify | Add `docs-dev`, `docs-build`, `docs-check`, `lint-md` recipes. |
| `docs/index.md` (or wired `docs/README.md`) | Create/Modify | Public-site landing page content. |
| `docs/**/*.md` | Modify | Convert any MkDocs-only Markdown to CommonMark. |
| `README.md` | Modify | Add "Documentation" section pointing to `claudelint.dev`. |
| `CONTRIBUTING.md` | Create/Modify | Document dual-pipeline doc workflow. |
| `CLAUDE.md` | Modify | Document `site/`, new just recipes, lint config, and dual-output setup. |

## Testing Plan

- **Local dev (Phase 1)**: `cd site && npm run dev` serves Starlight at `http://localhost:4321` with no errors.
- **Build smoke (Phase 1, 4)**: `cd site && npm run build` produces a non-empty `site/dist/`.
- **Content wiring (Phase 2)**: Sidebar enumerates all docs; each doc renders correctly; cross-doc relative links resolve.
- **Markdown lint (Phase 3)**: `markdownlint-cli2 'docs/**/*.md'` exits 0 with the CommonMark profile.
- **MkDocs parity (Phase 3, 6)**: `mkdocs build --strict` succeeds against the modified `docs/` tree.
- **CI (Phase 4)**: A PR touching `docs/` triggers the new workflow; a Go-only PR does not.
- **Cloudflare preview (Phase 5)**: PR comment lists a preview URL; URL serves the modified docs.
- **Production deploy (Phase 5, 6)**: Pages subdomain → `claudelint.dev` after DNS cutover; valid TLS; Pagefind search returns results.
- **Mermaid parity (Phase 6)**: A test doc with a Mermaid block renders identically (visually equivalent) in MkDocs Material and Starlight.
- **Rollback drill (Phase 6, optional)**: Verify that disabling the docs workflow or revoking the Cloudflare custom domain doesn't impact MkDocs / TechDocs build behavior.

## Dependencies

- **External services**: Cloudflare account access with `claudelint.dev` registered (Donald has this).
- **Tooling**: Node LTS (added to `mise.toml` in Phase 1).
- **Existing tooling**: `mkdocs`, `markdownlint-cli2`, `just`, `mise` (all already present).
- **GitHub repo secrets**: `CLOUDFLARE_ACCOUNT_ID` + `CLOUDFLARE_API_TOKEN` (added in Phase 5, used only if we ever move off the auto-integration).
- **Blocking work**: none — IMPL-0003 can proceed on the `docs/site` branch independently of any Go work.

---

## Resolved Decisions

All Open Questions resolved 2026-06-01.

- **Q1 — Node toolchain location**: Isolated `site/` subdirectory with its own `package.json` + lockfile. Keeps Node out of the Go-centric repo root; CI path filters stay unambiguous.
- **Q2 — Node version pinning**: `mise.toml` with `# renovate:` annotation. Matches the rest of the repo's tool management; Renovate keeps it current.
- **Q3 — Mermaid renderer**: `rehype-mermaid` via `@astrojs/markdown-remark`. Server-side rendering, no client JS payload, same ` ```mermaid ` fenced syntax MkDocs uses.
- **Q4 — CI blocking behavior**: Non-required for the first two PRs after merge; flip to required check in branch protection once both come back clean. Avoids the "first run failed, can't merge anything" trap.
- **Q5 — Sidebar grouping**: Filesystem auto-discovery, grouped by doc type (RFC, ADR, DESIGN, IMPL, PLAN, Investigation), docs ordered numerically within each group. Mirrors `docs/<type>/README.md`.
- **Q6 — Landing page**: Promote `docs/README.md` as the Starlight homepage via root-page config. Single Markdown source feeds both MkDocs and Starlight.
- **Q7 — Edit-link target**: `editLink` points at `https://github.com/donaldgifford/claudelint/edit/main/{path}` so readers can propose edits via GitHub's web editor.
- **Q8 — Renovate**: Default npm manager handles Astro/Starlight bumps; no custom group config.

---

## References

- [DESIGN-0003](../design/0003-dual-output-docs-site-shared-source-mkdocs-for-techdocs.md) — the design this implements
- [Starlight docs](https://starlight.astro.build/) — Astro integration
- [Astro Content Collections](https://docs.astro.build/en/guides/content-collections/) — mechanism for reading shared `docs/`
- [Pagefind](https://pagefind.app/) — default Starlight search
- [Cloudflare Pages](https://developers.cloudflare.com/pages/) — deploy target
- [rehype-mermaid](https://github.com/remcohaszing/rehype-mermaid) — proposed Mermaid plugin (Q3)
- [MkDocs 2.0 announcement — squidfunk, 2026-02-18](https://squidfunk.github.io/mkdocs-material/blog/2026/02/18/mkdocs-2.0/) — context for the broader docs-tooling shift
- IMPL-0001, IMPL-0002 — phase-driven implementation tracking pattern this doc follows
