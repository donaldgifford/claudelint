---
title: Changelog
---

All notable changes to this project are documented here. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) and
this project adheres to [Semantic Versioning](https://semver.org/).

## [unreleased]

### Features

- *(docs)* Dual-output docs site — MkDocs (TechDocs) + Starlight (claudelint.dev) ([#40](https://github.com/donaldgifford/claudelint/issues/40))
- *(artifact)* Accept mcpServers as primary .mcp.json key
- *(artifact)* Parse MCP transport fields on server entries
- *(artifact)* Parse marketplace object sources into typed MarketplaceSource
- *(artifact)* Parse marketplace owner{} and renames{} at the root
- *(artifact)* Parse hook entry type and per-type fields
- *(artifact)* Add shared tool-list splitter for string frontmatter forms
- *(artifact)* Add Agent/Skill to KnownTools and tool-pattern classifier
- *(artifact)* Expand KnownHookEvents to the full documented event set
- *(artifact)* Parse merged skill/command frontmatter model fields
- *(rules)* Suggest exact event casing in hooks/event-name-known
- *(rules)* Accept tool patterns and skills in commands/allowed-tools-known
- *(rules)* Add mcp/url-required for remote transports
- *(rules)* Validate object sources in marketplace/plugin-source-valid
- *(rules)* Rework marketplace/external-source-skipped for object sources
- *(rules)* Split marketplace version checks into semver and missing
- *(rules)* Replace marketplace/author-required with owner-required
- *(rules)* Add mcp/legacy-servers-key deprecation notice
- *(artifact)* Parse the full 16-field subagent frontmatter spec
- *(rules)* Add agents package with agents/model-valid
- *(rules)* Add agents/name-format
- *(rules)* Add agents/tools-known
- *(rules)* Add agents/plugin-ignored-fields
- *(rules)* Add agents/field-enums
- *(engine)* Add opt-in rule mechanism, migrate mcp/server-allowlist
- *(rules)* Add agents/model-policy governance rule
- *(rules)* Add hooks/type-known and hooks/type-fields
- *(rules)* Add mcp/transport-known + mcp/transport-deprecated
- *(rules)* Add mcp/no-secrets-in-headers
- *(rules)* Add mcp/timeout-minimum
- *(rules)* Add marketplace/reserved-name
- *(rules)* Add marketplace/name-format
- *(rules)* Add marketplace/source-path-safety
- *(rules)* Add marketplace/renames-valid
- *(rules)* Add skills/description-length
- *(rules)* Add skills/fork-agent-pairing
- *(rules)* Add claude_md/import-exists

### Bug Fixes

- *(rules)* Scope mcp/command-required to stdio transport
- *(rules)* Skip non-stdio transports in mcp/command-exists-on-path
- *(rules)* Cite documented per-type defaults in hooks/timeout-present
- *(rules)* Scope hooks/no-unsafe-shell to shell-form command hooks
- *(rules)* Best-practice phrasing for skill frontmatter diagnostics

### Documentation

- *(impl-0003)* Close out — production live at claudelint.dev ([#51](https://github.com/donaldgifford/claudelint/issues/51))
- INV-0006 rule coverage audit + IMPL-0004 implementation tracker ([#52](https://github.com/donaldgifford/claudelint/issues/52))
- *(rules)* Cite current docs in no-version-field and manifest-fields
- Sync README rule reference with the v1.3 ruleset
- *(design)* Record the mcpServers resolution in DESIGN-0002
- *(impl)* Record clean Phase 2 dogfood pass
- *(design)* Add DESIGN-0005 agent rules and opt-in rule mechanism
- *(impl)* Complete IMPL-0004 Phase 4 — coverage + dogfood
- *(impl)* Complete IMPL-0004 — Phase 5 dogfood, flip to Completed

### Testing

- *(artifact)* Fixture sweep covering every doc-valid Phase 1 shape
- *(rules)* Add Phase 4 agent fixtures + five-rule sweep
- *(rules)* Phase 5 fixture coverage

### Miscellaneous Tasks

- *(rules)* Bump ruleset to v1.4.0
- *(rules)* Bump ruleset to v1.5.0

## [0.2.3] - 2026-05-30

### Bug Fixes

- *(release)* Install syft before goreleaser ([#33](https://github.com/donaldgifford/claudelint/issues/33))

## [0.2.2] - 2026-05-30

### Miscellaneous Tasks

- *(security)* SBOM generation + grype scanning ([#32](https://github.com/donaldgifford/claudelint/issues/32))

## [0.2.1] - 2026-05-23

### Miscellaneous Tasks

- Tooling cleanup — just task runner, docker-bake, git-cliff, renovate ([#23](https://github.com/donaldgifford/claudelint/issues/23))

## [0.2.0] - 2026-04-26

### Features

- Range-aware secrets + mcp/server-allowlist + skills/no-version-field ([#19](https://github.com/donaldgifford/claudelint/issues/19))

## [0.1.1] - 2026-04-26

### Bug Fixes

- *(hooks)* Parse nested shape uniformly; drop unsupported flat shape ([#18](https://github.com/donaldgifford/claudelint/issues/18))

### Documentation

- Rename Phase 2 ship from v0.2.0 to v0.1.0 ([#10](https://github.com/donaldgifford/claudelint/issues/10))

## [0.1.0] - 2026-04-25

### Features

- Phase 2 — marketplaces, MCP rules, SARIF, Docker, companion Action

### Miscellaneous Tasks

- *(claude-md)* Reflect Phase 1 ship + label-driven release + GPG gotcha

## [0.0.1] - 2026-04-21

### Features

- *(cli)* Scaffold cobra CLI with run/rules/init/version stubs
- *(diag)* Add Diagnostic, Severity, Range, Position, Fix types
- *(artifact)* Add ArtifactKind enum and Artifact interface
- *(discovery)* Walk repo with gitignore semantics and classify artifacts
- *(reporter)* Add minimal text formatter
- *(cli)* Wire run end-to-end — discovery + text reporter
- *(cli,rules)* Move ruleset version constants into internal/rules
- *(artifact)* Add typed Artifact structs and positional helpers
- *(artifact)* Add ParseError type for parser failures
- *(artifact)* Parse Markdown + YAML frontmatter via goccy/go-yaml
- *(artifact)* Parse Hook and Plugin JSON with byte-offset ranges
- *(artifact)* Index Skill companion files by kind
- *(config)* HCL v1 loader, resolver, and init scaffolder
- *(rules,engine)* Ship Rule interface, registry, and concurrent runner
- *(rules)* Ship every MVP rule, wire registry, add fingerprint gate
- *(engine)* Implement three-mechanism suppression for Phase 1.6
- *(cli)* Phase 1.7 output formats, flags, exit codes, E2E tests
- *(cli,ci)* Phase 1.8 — pprof, bench, coverage gate, release polish
- *(discovery)* Classify plugin-root layouts without .claude/ parent

### Other

- Settings

### Documentation

- Add RFC/ADR/DESIGN/IMPL/INV for claudelint linter
- Reframe DESIGN-0001 around engine+rules, rename lint→run
- *(impl-0001)* Align with DESIGN-0001 and add Open Questions
- Fold resolved open questions into DESIGN-0001 and IMPL-0001
- *(inv-0002)* Diagnose ralph-loop permission matcher bug, plan fork
- *(inv-0002)* Fix ToC truncation by removing raw backtick fences
- *(inv-0002)* Add upstream git history, issues, and unmerged PRs
- *(inv-0002)* Add copy-pasteable reference implementation for fork
- Add CLAUDE.md with architecture, commands, and doc workflow
- *(README,impl)* Add per-rule examples and fixes
- *(release)* Finalize Phase 1 testing plan + add RELEASE.md
- *(inv)* Record donald-loop stop hook silent-promise-detection bug

### Testing

- *(artifact)* Add testdata/ok + testdata/bad parser fixtures

### Miscellaneous Tasks

- Allow ralph-loop setup script in project permissions
- Remove ineffective ralph-loop allow entry
- Add LICENSE (Apache-2.0) and CHANGELOG for v0.1.0
- Ignore dist/ and untrack accidentally-committed goreleaser output
- Drop docker-build job — no Docker distribution in Phase 1
- Drop docker release job; pin codeql-action to v4.35.1

