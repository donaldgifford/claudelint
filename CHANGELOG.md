# Changelog

All notable changes to `claudelint` are documented here. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Phase 1 MVP of the linter (`run`, `rules`, `init`, `version`
  subcommands).
- 14 built-in rules across `schema`, `content`, `security`, and
  `style` categories.
- Three independent suppression mechanisms: in-source HTML-comment
  markers (Markdown kinds only), config-level `rule { enabled = false }`
  / `paths = […]` toggles, and a `meta/unknown-rule` warning for typos.
- Three output formats: text (colorized, `NO_COLOR`-aware), JSON (stable
  schema v1 documented in `docs/json-output-schema.md`), and GitHub
  Actions workflow commands.
- Exit-code contract: `0` clean, `1` diagnostics failed the run,
  `2` usage / config / I/O error. `--max-warnings=N` promotes warning
  overflow into exit 1.
- `--profile=<dir>` flag captures cpu/heap/block/mutex pprof for a
  single run.
- CI coverage gate (`make coverage-gate`) at `COVERAGE_MIN=55`, with
  the eventual target documented as 80%.
- Dogfood investigation ([INV-0003](docs/investigation/0003-phase-18-dogfood-findings-on-external-claude-plugins.md))
  against two external Claude plugins; surfaced a plugin-layout
  classifier gap that is now fixed.

### Fixed

- `internal/discovery/classify.go` now recognizes plugin-distribution
  layouts where `skills/`, `commands/`, `agents/`, and `hooks/` sit at
  the plugin root (no `.claude/` parent). Prior versions silently
  ignored every plugin artifact outside the `.claude/` convention.

[Unreleased]: https://github.com/donaldgifford/claudelint/compare/v0.0.0...HEAD
