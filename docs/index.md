---
title: claudelint
description: A static linter for Claude Code artifacts — CLAUDE.md, skills, slash commands, subagents, hooks, plugins, marketplaces, and MCP servers.
---

A static linter for Claude Code artifacts: `CLAUDE.md`, skills, slash commands,
subagents, hooks, plugin manifests, marketplaces, and MCP servers.

`claudelint` walks a repository, classifies each matching file into a typed
artifact (a `Skill`, a `CommandManifest`, a `HookConfig`, etc.), and runs a
built-in ruleset against it. The shape is `golangci-lint`-inspired: parsers →
engine → rules, built-in rules only, versioned with the binary, no plugin SDK
in v1.

## Install

```bash
go install github.com/donaldgifford/claudelint/cmd/claudelint@latest
```

Prebuilt binaries for darwin/linux/windows ship with each tagged release on the
[GitHub Releases page](https://github.com/donaldgifford/claudelint/releases).
Multi-arch container images (linux/amd64, linux/arm64) ship alongside each
release at `ghcr.io/donaldgifford/claudelint`. The full walkthrough lives in
[Quickstart](./install/quickstart.md); CI integration is covered in
[GitHub Action](./install/github-action.md).

## Quickstart

Lint the current repo:

```bash
claudelint run .
```

Emit GitHub Actions annotations on your PR (preferred: use the companion
[`claudelint-action`](https://github.com/donaldgifford/claudelint-action), which
handles install + SARIF upload):

```yaml
- uses: donaldgifford/claudelint-action@v1
```

Or invoke the binary directly:

```yaml
- run: claudelint run --format=github .
```

Fail the build on any warning:

```bash
claudelint run --max-warnings=0 .
```

Write a starter `.claudelint.hcl`:

```bash
claudelint init
```

List every rule shipped in the binary:

```bash
claudelint rules            # human-readable table
claudelint rules --json     # machine-readable catalog
claudelint rules <id>       # detail view for one rule
```

## Output formats

- `--format=text` (default) — human-readable; honors `--no-color` / `NO_COLOR`.
- `--format=json` — stable schema documented in
  [JSON output schema](./json-output-schema.md).
- `--format=github` — `::error` / `::warning` / `::notice` workflow commands.
- `--format=sarif` — SARIF 2.1.0 log, suitable for GitHub Code Scanning and
  other SARIF-aware tools. Pair with `--sarif-file=<path>` to write to a file
  instead of stdout.

## Linted artifacts

| Kind | Examples |
| --- | --- |
| `CLAUDE.md` | repo root, nested directories |
| Skills | `**/skills/*/SKILL.md` |
| Slash commands | `**/commands/*.md` |
| Subagents | `**/agents/*.md` |
| Hooks | `settings.json`, plugin `hooks/hooks.json`, `.claude/hooks/*.json` |
| Plugin manifests | `.claude-plugin/plugin.json` |
| Marketplaces | `.claude-plugin/marketplace.json` |
| MCP servers | project-scoped `.mcp.json`, user-scope MCP entries |

## What's in the box

- **Built-in ruleset.** Every rule ships with the binary; the version
  fingerprint changes whenever a rule's id / category / severity / options
  change. CI breaks if the drift is not acknowledged. See the
  [Rules reference](./rules/rules.md) for the full catalog and the
  [Exit codes](./rules/exit-codes.md) page for what they mean.
- **HCL config.** `claudelint init` writes a starter `.claudelint.hcl`
  in your repo. Per-rule severity, options, and path scoping all live
  there. See [ADR-0001](./adr/0001-use-hcl-as-the-linter-configuration-format.md)
  for the format decision.
- **SARIF + GitHub Actions output** so it drops cleanly into existing CI
  surfaces (Code Scanning, PR annotations).
- **Suppression model** with in-source markers for Markdown kinds and
  config-level toggles for JSON kinds. Three independent mechanisms — see
  [DESIGN-0001 §Suppression model](./design/0001-claudelint-linter-architecture-and-rule-engine.md#suppression-model).

## Where to go next

- **Install + run** — start with [Quickstart](./install/quickstart.md), then
  wire CI via [GitHub Action](./install/github-action.md).
- **What gets flagged** — browse the full
  [Rules reference](./rules/rules.md) and [Exit codes](./rules/exit-codes.md).
- **Design + history** — the [Development](./development/docs.md) section
  collects the architecture docs, dev guides, and every RFC / ADR / DESIGN /
  IMPL / Plan / Investigation behind the project.
- **What changed** — the [Changelog](./changelog.md) is regenerated from
  conventional commits via git-cliff.
- **Source + issues** —
  [github.com/donaldgifford/claudelint](https://github.com/donaldgifford/claudelint).
