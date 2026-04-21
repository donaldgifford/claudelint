# claudelint

Framework and tooling for linting, auditing, and formatting Claude plugins, skills, subagents, and more.

## Getting Started

TODO: Add getting started instructions (Phase 1.8).

## Suppressing diagnostics

claudelint offers three independent suppression mechanisms. Pick the
one that matches the scope of the exception.

### In-source markers (Markdown kinds only)

`CLAUDE.md`, `SKILL.md`, command, and agent files accept HTML-comment
markers:

```markdown
<!-- claudelint:ignore=skills/require-name -->      # same line or next non-blank line
<!-- claudelint:ignore=skills/require-name,style/no-emoji --> # multiple IDs

<!-- claudelint:ignore-file=skills/require-name --> # whole-file
```

JSON-backed artifacts (`hooks.json`, plugin manifests) do **not**
recognize in-source markers — JSON has no standard comment syntax. Use
config-level suppression for those kinds.

### Config-level rule toggles

In `.claudelint.hcl`, disable a rule for every artifact or scope it to
a subset of paths:

```hcl
rule "skills/require-name" {
  enabled = false
}

rule "style/no-emoji" {
  # Only ignore this rule inside vendored docs.
  paths = ["vendor/**/*.md"]
}
```

### Unknown-ID safety net

A typo in a `rule "<id>"` block silently does *not* disable the
real rule; instead, claudelint emits a `meta/unknown-rule` warning
pointing at the offending config. Fix the typo or the warning stays.

