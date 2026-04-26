# Changelog

All notable changes to `claudelint` are documented here. The format
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Fixed

- `hooks/timeout-present` no longer false-fires on plugin
  `hooks/hooks.json` files that declare `timeout` per inner entry. The
  hook parser previously dispatched non-`settings.json` files to a flat
  `{event, matcher, command, timeout}` shape that does not appear in
  the Claude Code docs, causing `Timeout` to read as 0 for every entry
  in a real plugin hook file. The parser now uses the canonical nested
  `{"hooks": {"<EventName>": [{"matcher": "...", "hooks": [...]}]}}`
  shape uniformly. (#14)

### Changed

- **Breaking:** the hook parser no longer accepts the flat
  `{event, matcher, command, timeout}` top-level shape. A dedicated
  hook file (`.claude/hooks/*.json`, plugin `hooks/hooks.json`) that
  is missing the `hooks` key now fails parsing with `*ParseError`
  rather than silently producing an entry with `Timeout == 0`.
  Settings files (`.claude/settings{,.local}.json`) may still omit
  the `hooks` key. The flat shape was a parser-author assumption that
  did not match Claude Code's documented hook schema; see
  `docs/design/0001-*.md` "Hook shape" for the rationale and the
  best-effort handling of `.claude/hooks/*.json`.

## [v0.1.0] — 2026-04-25

### Added

- Phase 2: two new artifact kinds — `KindMarketplace` for
  `.claude-plugin/marketplace.json` and `KindMCPServer` for MCP server
  declarations (standalone `.mcp.json` or plugin-embedded
  `mcp.servers{}`).
- Phase 2: eight `marketplace/*` rules (schema, author, plugin
  uniqueness, plugin-source validity, external-source handling,
  versioning) and six `mcp/*` rules (command required, known runner,
  no secrets in env, no unsafe shell, disabled-but-commented, server
  name required).
- Phase 2: `Rule.HelpURI() string` method with `rules.DefaultHelpURI`
  helper. Every built-in rule now exposes a documentation URL.
- Phase 2: `claudelint rules --json` emits the rule catalog in a stable
  schema documented at `docs/rules-json-schema.md`.
- Phase 2: `--format=sarif` renders diagnostics as SARIF 2.1.0, suitable
  for GitHub Code Scanning. `--sarif-file=<path>` redirects SARIF
  output to a file.
- Phase 2: multi-arch container image at
  `ghcr.io/donaldgifford/claudelint` (linux/amd64 + linux/arm64),
  published via goreleaser on every release.
- Ruleset version bumped to `v1.1.0` (minor, additive) to reflect the
  new rule packages.

### Fixed

- `marketplace/version-semver` and `marketplace/author-required` no
  longer produce false positives on marketplace manifests that nest
  `version` under `metadata.version` or express `author` as an
  `owner{name,email}` object. The parser now accepts both the shape
  DESIGN-0002 documents and the shape real marketplaces (e.g.
  `donaldgifford/claude-skills`) actually use. See INV-0005.
- `commands/allowed-tools-known` now recognizes `AskUserQuestion` as a
  Claude Code built-in tool.

## [v0.0.1]

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

[Unreleased]: https://github.com/donaldgifford/claudelint/compare/v0.1.0...HEAD
[v0.1.0]: https://github.com/donaldgifford/claudelint/compare/v0.0.1...v0.1.0
[v0.0.1]: https://github.com/donaldgifford/claudelint/compare/v0.0.0...v0.0.1
