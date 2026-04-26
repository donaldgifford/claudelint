# claudelint rules JSON schema

`claudelint rules --json` emits a single JSON object to stdout describing
the built-in ruleset. `claudelint rules <id> --json` emits the same
envelope but with exactly one entry in `rules`. This document is the
stable contract that external consumers — editor integrations,
documentation generators, policy registries — can pin on. Any breaking
change bumps `schema_version` and lands in a separate release note.

The primary consumer of this output is the companion `claudelint-action`
repo (Phase 2.9); keeping the shape narrow and stable means CI-facing
tools can discover the ruleset without parsing human-targeted text.

## Top-level shape

```json
{
  "schema_version":  "1",
  "ruleset_version": "v1.2.0",
  "fingerprint":     "e7f26796",
  "rules": [
    { ... }
  ]
}
```

| Field             | Type    | Description                                                                                      |
| ----------------- | ------- | ------------------------------------------------------------------------------------------------ |
| `schema_version`  | string  | Incremented on any breaking change to this output. Currently `"1"`.                              |
| `ruleset_version` | string  | SemVer of the registered ruleset at binary build time (mirrors `rules.RulesetVersion`).          |
| `fingerprint`     | string  | Truncated sha256 of the registry. Changes only when rules are added / removed / retyped.         |
| `rules`           | array   | Rule descriptors, sorted by `id`. Always an array, never `null`. For `rules <id>`, length == 1.  |

`schema_version`, `ruleset_version`, and `fingerprint` are identical in
shape and semantics to `claudelint run --format=json`, so a downstream
tool can cache ruleset metadata by fingerprint without special-casing
per-command.

## Rule shape

Each entry in `rules` has this shape:

```json
{
  "id":               "mcp/no-unsafe-shell",
  "category":         "security",
  "default_severity": "error",
  "applies_to":       ["mcp_server"],
  "help_uri":         "https://github.com/donaldgifford/claudelint/blob/main/README.md#rule-mcp-no-unsafe-shell",
  "default_options":  {}
}
```

| Field              | Type   | Description                                                                                            |
| ------------------ | ------ | ------------------------------------------------------------------------------------------------------ |
| `id`               | string | Stable rule identifier (`category/name`) used by config and `<!-- claudelint:ignore=<id> -->`.         |
| `category`         | string | One of `schema`, `content`, `security`, `style`, `meta`.                                               |
| `default_severity` | string | One of `error`, `warning`, `info`. Matches the engine's severity vocabulary.                           |
| `applies_to`       | array  | Artifact kinds this rule analyzes (e.g. `claude_md`, `skill`, `plugin`, `marketplace`, `mcp_server`).  |
| `help_uri`         | string | URL to documentation for this rule. Rules without bespoke docs point at the README anchor.             |
| `default_options`  | object | Option keys and their default values. Empty object (never `null`) when the rule takes no options.      |

## Stability

- Field names and types are part of the contract — never renamed within a
  schema version.
- New fields may be added in a minor release; consumers must ignore
  unknown fields.
- `fingerprint` is suitable as a cache key: identical output for the same
  registered rules, regardless of binary version.
- `applies_to` preserves the order registered by the rule, not sorted;
  consumers that need a stable order should sort it themselves.
