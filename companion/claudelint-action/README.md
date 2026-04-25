# claudelint-action

A GitHub Action that runs [claudelint](https://github.com/donaldgifford/claudelint)
against a Claude Code repository — CLAUDE.md files, skills, commands,
agents, hooks, plugin manifests, marketplace manifests, and MCP server
declarations — and optionally uploads results to GitHub Code Scanning.

## Quickstart

```yaml
name: claudelint
on: [push, pull_request]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: donaldgifford/claudelint-action@v1
```

The default format is `github` — diagnostics render as workflow
annotations on the PR. See **Inputs** below for the full set of knobs.

## Upload SARIF to Code Scanning

```yaml
name: claudelint
on: [push, pull_request]

permissions:
  contents: read
  security-events: write

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: donaldgifford/claudelint-action@v1
        with:
          format: sarif
          upload-sarif: "true"
```

The Action writes SARIF to `${{ runner.temp }}/claudelint.sarif` and
calls `github/codeql-action/upload-sarif@v4` internally; you do not
need a second step.

## Pin to a specific claudelint version

```yaml
      - uses: donaldgifford/claudelint-action@v1
        with:
          version: v0.1.0
```

`latest` resolves to the most recent release tag at runtime via the
GitHub API. Pinning is recommended for deterministic CI.

## Inputs

| Name             | Default      | Description |
|------------------|--------------|-------------|
| `version`        | `latest`     | claudelint release tag to download (e.g. `v0.1.0`). |
| `path`           | `.`          | Space-separated paths to lint. |
| `format`         | `github`     | `text`, `json`, `github`, or `sarif`. |
| `config`         | *(discovery)*| Explicit path to `.claudelint.hcl`. |
| `max-warnings`   | `-1`         | Fail when warnings exceed this count. `-1` disables. |
| `upload-sarif`   | `false`      | If `true` and `format=sarif`, upload to Code Scanning. |
| `sarif-category` | `claudelint` | Category name for the uploaded SARIF. |

## Outputs

| Name                 | Description |
|----------------------|-------------|
| `diagnostics-count`  | Total diagnostics (errors + warnings + info). |
| `error-count`        | Error-severity diagnostic count. |
| `sarif-path`         | Path to the generated SARIF file (if `format=sarif`). |

## Checksum verification

Every release of claudelint publishes `checksums.txt` alongside the
binaries. The Action downloads both and verifies the archive's
SHA-256 before extraction. A tampered release asset fails the run.

## Platform support

- **Linux (x64 / arm64)**: full support.
- **macOS (x64 / arm64)**: full support.
- **Windows (x64)**: best-effort; please open an issue if you hit a
  bug. Shell is `bash` (pwsh is not configured).

## License

Apache-2.0 — see `LICENSE`.

## See also

- [claudelint](https://github.com/donaldgifford/claudelint) — the
  linter this Action wraps.
- [DESIGN-0002](https://github.com/donaldgifford/claudelint/blob/main/docs/design/0002-phase-2-marketplaces-mcp-rules-and-github-action.md)
  §3 — original design for this Action.
