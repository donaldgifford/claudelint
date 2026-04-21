# claudelint JSON output schema

`claudelint run --format=json` emits a single JSON object to stdout.
This document is the stable contract that external consumers can pin
on; any breaking change bumps `schema_version` and lands in a separate
release note.

## Top-level shape

```json
{
  "schema_version":    "1",
  "ruleset_version":   "v1.0.0",
  "fingerprint":       "d97ca099",
  "files_checked":     42,
  "diagnostic_count":  3,
  "severity_count": {
    "error":   1,
    "warning": 1,
    "info":    1
  },
  "diagnostics": [
    { ... }
  ]
}
```

| Field              | Type    | Description                                                                                   |
| ------------------ | ------- | --------------------------------------------------------------------------------------------- |
| `schema_version`   | string  | Incremented on any breaking change. Currently `"1"`.                                          |
| `ruleset_version`  | string  | SemVer of the registered ruleset at binary build time.                                        |
| `fingerprint`      | string  | Truncated sha256 of the registry. Changes only when rules are added / removed / retyped.      |
| `files_checked`    | integer | Count of artifacts parsed, regardless of whether each produced a diagnostic.                  |
| `diagnostic_count` | integer | `len(diagnostics)`. Matches the number of elements for convenience.                           |
| `severity_count`   | object  | Per-severity counts; keys `error`, `warning`, `info` always present, zero when none.          |
| `diagnostics`      | array   | Sorted `(path, line, col, rule_id)`; deduped on exact equality. Always `[]`, never `null`.    |

## Diagnostic shape

Each entry in `diagnostics` has this shape:

```json
{
  "rule_id":  "skills/require-name",
  "severity": "error",
  "path":     "skills/a/SKILL.md",
  "range": {
    "start": {"line": 2, "column": 1, "offset": 5},
    "end":   {"line": 2, "column": 10, "offset": 14}
  },
  "message": "frontmatter missing \"name\""
}
```

- `severity` encodes as the lowercase string `"error" | "warning" | "info"`.
- `range.start` and `range.end` share the same shape. When a diagnostic
  is file-level (for example a JSON parse error) the range is a zero
  value `{start: {line: 0, column: 0, offset: 0}, end: ...}` and
  consumers should use `path` alone for location.
- Line and column are 1-based. `offset` is 0-based into the raw source.
- The optional `detail` field carries a longer explanation when a rule
  produces one; omitted when empty.
- `fix` is reserved for a future `claudelint fix` subcommand. It will
  be populated starting in v2 of this schema; it is always omitted in
  schema v1.

## Exit codes

The JSON output is independent of the exit code. `claudelint` exits:

- `0` when no error-severity diagnostics were produced and any warning
  count is within `--max-warnings`.
- `1` when any `error` appears, or `--max-warnings=N` was exceeded.
- `2` for usage / config / I/O problems (bad `--format`, missing config
  file, unreadable artifact, etc.).

## Stability guarantees

- Adding new top-level keys or new diagnostic fields is non-breaking;
  consumers should ignore unknown keys.
- Removing or renaming keys, or changing a field's type, bumps
  `schema_version`.
- `severity_count` will always have the same keys (even as new
  severities are added, they are additive).
