---
title: Quickstart
---

## Install

```bash
go install github.com/donaldgifford/claudelint/cmd/claudelint@latest
```

Prebuilt binaries for darwin/linux/windows ship with each tagged release on the
[GitHub Releases page](https://github.com/donaldgifford/claudelint/releases).

Multi-arch container images (linux/amd64, linux/arm64) ship alongside each
release at `ghcr.io/donaldgifford/claudelint`. See
[the GitHub Action guide](./github-action.md) for CI integration.

## Quickstart

Lint the current repo:

```bash
claudelint run .
```

Emit GitHub Actions annotations on your PR (preferred: use the companion
[`claudelint-action`](https://github.com/donaldgifford/claudelint-action) which
also handles install + SARIF upload):

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
claudelint rules --json     # machine-readable catalog (see docs/rules-json-schema.md)
claudelint rules <id>       # detail view for one rule
```

## Output formats

- `--format=text` (default) — human-readable; honors `--no-color` / `NO_COLOR`.
- `--format=json` — stable schema documented in
  [json-output-schema.md](../json-output-schema.md).
- `--format=github` — `::error` / `::warning` / `::notice` workflow commands.
- `--format=sarif` — SARIF 2.1.0 log, suitable for GitHub Code Scanning and
  other SARIF-aware tools. Pair with `--sarif-file=<path>` to write to a file
  instead of stdout.
