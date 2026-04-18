---
id: ADR-0001
title: "Use HCL as the linter configuration format"
status: Proposed
author: Donald Gifford
created: 2026-04-18
---
<!-- markdownlint-disable-file MD025 MD041 -->

# 0001. Use HCL as the linter configuration format

<!--toc:start-->
- [Status](#status)
- [Context](#context)
- [Decision](#decision)
- [Consequences](#consequences)
  - [Positive](#positive)
  - [Negative](#negative)
  - [Neutral](#neutral)
- [Alternatives Considered](#alternatives-considered)
- [References](#references)
<!--toc:end-->

## Status

Proposed

## Context

Claudelint (see RFC-0001) needs a configuration file that:

- Supports **typed, labeled blocks** for per-artifact rule groups
  (`rules "skills" { ... }`, `rules "hooks" { ... }`).
- Allows **comments** so teams can explain why a rule is disabled or tuned.
- Has a mature, well-maintained Go library with stable parsing and good
  error messages (line/column diagnostics are essential for a linter's
  own config).
- Feels familiar to infrastructure-adjacent users, which is our target
  audience (teams already running CI, Terraform, Packer, Nomad).
- Can represent nested structure without significant whitespace ambiguity.

The leading candidates are YAML, TOML, JSON, Starlark, and HCL.

## Decision

Use **HCL v2** (`github.com/hashicorp/hcl/v2`) as the canonical config
format for `.claudelint.hcl`.

- Config schema is versioned via a top-level `claudelint { version = "1" }`
  block; unknown top-level blocks are a warning, unknown fields inside
  known blocks are errors.
- No other config format is supported in v1. A future ADR may add JSON as
  a machine-generated alternative (HCL parses into a JSON-compatible AST,
  so this is cheap if needed).
- Inline suppressions in source artifacts (Markdown/JSON) use a
  `// claudelint:ignore=<rule-id>` comment convention, independent of HCL.

## Consequences

### Positive

- Typed blocks with labels map cleanly onto per-artifact rule groups.
- Comments are first-class — users can document *why* a rule is off.
- Good Go library: `hashicorp/hcl/v2` is production-quality and gives us
  precise diagnostics with file/line/column out of the box.
- Familiar to the Terraform/Packer/Nomad user base and easy to learn for
  others thanks to the simple syntax.
- HCL AST round-trips to JSON, so tooling can emit config programmatically.

### Negative

- HCL is less common than YAML in the AI-tooling ecosystem; some users
  will need to learn it. Mitigation: `claudelint init` scaffolds a
  commented default config.
- Adds a non-trivial dependency (`hashicorp/hcl/v2` and its transitive
  deps) to an otherwise small Go binary.
- HCL has its own expression language; we must pick a *subset* to support
  and explicitly disable dynamic features (e.g., functions, variables
  across files) to keep config predictable.

### Neutral

- Users who prefer YAML can generate HCL from YAML via a small helper;
  we may ship one but do not commit to parity.
- Schema validation errors surface with HCL-style diagnostics, which
  differ slightly from the diagnostics claudelint emits for artifacts.

## Alternatives Considered

- **YAML.** Ubiquitous and widely known, but weakly typed, whitespace-
  sensitive, and awkward for labeled blocks. Error messages from the
  common Go YAML parsers are noticeably worse than HCL's, which matters
  because config errors are our users' first experience with the tool.
- **TOML.** Good for flat key-value config; awkward for nested/labeled
  rule blocks. No natural way to express `rules "skills" { ... }`.
- **JSON.** No comments. Verbose. Acceptable as a machine-generated
  secondary format, not as the hand-authored primary.
- **Starlark.** Expressive and typed, but executable config is a large
  complexity and security cost for a linter. Users expect declarative.
- **Custom DSL.** Rejected. Maintenance burden with no clear upside over
  HCL.

## References

- RFC-0001 — Claudelint
- HashiCorp HCL: https://github.com/hashicorp/hcl
- HCL language spec: https://github.com/hashicorp/hcl/blob/main/hclsyntax/spec.md
