# Contributing to claudelint

Thanks for your interest in helping. This project follows a small set
of conventions that keep the codebase, the docs, and the release
pipeline coherent.

## Quick start

```bash
mise install                      # toolchain pinned in mise.toml
just check                        # lint + test
just self-check                   # dogfood claudelint against this repo
```

`just --list` enumerates every recipe. The same automation is available
via `make` — both target sets are equivalent.

## Code

- Go style follows Uber's Go Style Guide (enforced by `golangci-lint`
  via `.golangci.yml`). Don't relax rules casually — prefer fixing the
  code.
- Run `just fmt` before committing. `gofmt -s` + `goimports` are wired
  in.
- New rules live under `internal/rules/<kind>/` as a single file
  (~50 LOC) implementing the `Rule` interface. Each rule's `init()`
  calls `Register()`. Blank-imported from `internal/rules/all/` so the
  binary picks them up.
- Architecture notes are in `docs/design/0001-*.md` and
  `docs/design/0002-*.md`. Read those before changing the engine,
  parser, or rule API.

## Documentation

The same `docs/` tree feeds two pipelines:

1. **Astro + Starlight** publishes the public site at
   <https://claudelint.dev>. Source lives under `site/`. Local
   preview:

   ```bash
   just docs-install
   just docs-dev          # http://localhost:4321
   ```

2. **MkDocs** drives Backstage TechDocs from `mkdocs.yml`, which
   `docz update` keeps in sync.

Authoring rules:

- Use CommonMark + GFM only. No MkDocs-only syntax (`!!! note`, `???`,
  `pymdownx.*`) — `just lint-md` enforces this.
- Fenced code blocks need a language hint (`text` is fine for ASCII
  diagrams).
- Wrap bare URLs in `<>`.
- Cross-doc links: write Markdown-style relative paths with `.md`
  extensions (e.g. `[DESIGN-0001](../design/0001-foo.md)`). MkDocs
  handles them natively; Starlight's custom remark plugin rewrites
  them to absolute lowercase routes.
- All docs go through [`docz`](https://github.com/donaldgifford/docz).
  Run `docz update` after editing any doc to refresh index tables and
  in-file ToCs. Six doc types: `rfc`, `adr`, `design`, `impl`, `plan`,
  `investigation`.

## Pull requests

- Branch prefixes drive auto-labeling via `.github/labeler.yml`:
  `feat/`, `fix/`, `chore/`, `docs/`, `bug/`.
- Releases are **label-driven**, not manual. The PR's release label
  (`major` / `minor` / `patch` / `dont-release`) determines the
  version bump on merge. Don't run `git tag` by hand.
- The squash-merge subject must keep the `(#N)` suffix — accept
  GitHub's default subject or include `(#N)` manually if overriding.
- PRs that touch docs trigger a Starlight build + markdownlint check
  via `.github/workflows/docs.yml`.

## Filing issues

- Bug reports and RFCs: <https://github.com/donaldgifford/claudelint/issues>.
- For larger proposals, open an RFC document via `docz create rfc` and
  link it from the issue.
