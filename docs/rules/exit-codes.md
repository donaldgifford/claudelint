---
title: Exit Codes
---

## Exit codes

| Code | Meaning                                                                     |
| ---- | --------------------------------------------------------------------------- |
| `0`  | Run succeeded; no error-severity diagnostics, no `--max-warnings` overflow. |
| `1`  | Run produced at least one `error`, or warnings exceeded `--max-warnings=N`. |
| `2`  | Usage, config, or I/O problem (`--format=bogus`, unreadable config, etc.).  |

## Suppressing diagnostics

Three independent mechanisms, each documented in
[DESIGN-0001 §Suppression model](../design/0001-claudelint-linter-architecture-and-rule-engine.md#suppression-model).

### In-source markers (Markdown kinds only)

`CLAUDE.md`, `SKILL.md`, command, and agent files accept HTML-comment markers.
The marker silences diagnostics on its own line and the next non-blank line.

```markdown
<!-- claudelint:ignore=skills/require-name -->              <!-- same line or next non-blank line -->
<!-- claudelint:ignore=skills/require-name,style/no-emoji --> <!-- multiple IDs -->

<!-- claudelint:ignore-file=skills/require-name -->        <!-- whole-file -->
```

JSON-backed artifacts (`hooks.json`, plugin manifests) do **not** recognize
in-source markers — JSON has no standard comment syntax. Use config-level
suppression for those kinds.

### Config-level rule toggles

In `.claudelint.hcl`, disable a rule for every artifact or scope it to a subset
of paths:

```hcl
rule "skills/require-name" {
  enabled = false
}

rule "style/no-emoji" {
  # Only silence this rule inside vendored docs.
  paths = ["vendor/**/*.md"]
}

ignore {
  paths = [
    "testdata/**",
    "vendor/**",
  ]
}
```

### Unknown-ID safety net

A typo in a `rule "<id>"` block does _not_ silently disable the real rule.
`claudelint` emits a `meta/unknown-rule` warning pointing at the offending
config so the typo is visible.
