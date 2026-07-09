---
title: GitHub Action
---

## Running in CI

### GitHub Actions

```yaml
- run: claudelint run --format=github .
```

Or upload SARIF to Code Scanning:

```yaml
- run: claudelint run --format=sarif --sarif-file=claudelint.sarif .
- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: claudelint.sarif
```

### Docker (GitLab CI, Jenkins, generic)

```bash
docker run --rm -v "$PWD:/src" -w /src ghcr.io/donaldgifford/claudelint:latest run .
```

The image pins a tag per release (`:0.1.0`, `:v0`, `:v0.1`, `:latest`). Use a
pinned tag in scheduled pipelines to avoid surprises when claudelint ships a new
rule.
