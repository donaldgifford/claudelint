# claudelint

A static linter for Claude Code artifacts: `CLAUDE.md`, skills,
slash commands, subagents, hooks, and plugin manifests.

`claudelint` walks a repository, classifies each matching file into a
typed artifact (a `Skill`, a `CommandManifest`, a `HookConfig`, etc.),
and runs a built-in ruleset against it. The design goal is
`golangci-lint` shape: parsers → engine → rules, built-in rules only,
versioned with the binary, no plugin SDK in v1.

See [docs/rfc](docs/rfc/0001-claudelint-linter-for-claude-code-artifacts.md)
for the full rationale, [docs/design](docs/design/0001-claudelint-linter-architecture-and-rule-engine.md)
for the architecture, and [docs/impl](docs/impl/0001-phase-1-core-linter-for-claudemd-skills-plugins-and-hooks.md)
for the phased rollout.

## Install

```bash
go install github.com/donaldgifford/claudelint/cmd/claudelint@latest
```

Prebuilt binaries for darwin/linux/windows ship with each tagged
release on the [GitHub Releases page](https://github.com/donaldgifford/claudelint/releases).

## Quickstart

Lint the current repo:

```bash
claudelint run .
```

Emit GitHub Actions annotations on your PR:

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
claudelint rules
```

## Output formats

- `--format=text` (default) — human-readable; honors `--no-color` / `NO_COLOR`.
- `--format=json` — stable schema documented in [docs/json-output-schema.md](docs/json-output-schema.md).
- `--format=github` — `::error` / `::warning` / `::notice` workflow commands.

## Exit codes

| Code | Meaning |
|------|---------|
| `0`  | Run succeeded; no error-severity diagnostics, no `--max-warnings` overflow. |
| `1`  | Run produced at least one `error`, or warnings exceeded `--max-warnings=N`. |
| `2`  | Usage, config, or I/O problem (`--format=bogus`, unreadable config, etc.). |

## Suppressing diagnostics

Three independent mechanisms, each documented in
[docs/design](docs/design/0001-claudelint-linter-architecture-and-rule-engine.md#suppression-model).

### In-source markers (Markdown kinds only)

`CLAUDE.md`, `SKILL.md`, command, and agent files accept HTML-comment
markers. The marker silences diagnostics on its own line and the next
non-blank line.

```markdown
<!-- claudelint:ignore=skills/require-name -->              <!-- same line or next non-blank line -->
<!-- claudelint:ignore=skills/require-name,style/no-emoji --> <!-- multiple IDs -->

<!-- claudelint:ignore-file=skills/require-name -->        <!-- whole-file -->
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

A typo in a `rule "<id>"` block does *not* silently disable the real
rule. `claudelint` emits a `meta/unknown-rule` warning pointing at the
offending config so the typo is visible.

## Rules (v1)

Every rule is built into the binary. The fingerprint under
`claudelint version` changes whenever rules are added, removed, or
have their ID / category / severity / options changed — a CI guardrail
fails if the drift is not acknowledged.

| ID                              | Category | Default  | Applies to                  | What it checks |
|---------------------------------|----------|----------|-----------------------------|----------------|
| `schema/parse`                  | schema   | error    | every kind                  | Synthetic: emitted by the engine when a file cannot be parsed (bad YAML frontmatter, invalid JSON). |
| `schema/frontmatter-required`   | schema   | error    | skill, command, agent       | Required frontmatter keys are present and non-empty. |
| `skills/trigger-clarity`        | content  | warning  | skill                       | `description:` includes a "Use when …" trigger phrase so Claude can match the skill on intent. |
| `skills/body-size`              | content  | warning  | skill                       | SKILL.md body is under `max_words` (default 2000). |
| `claude_md/duplicate-directives`| content  | warning  | `CLAUDE.md`                 | No repeated directive lines. |
| `claude_md/size`                | content  | warning  | `CLAUDE.md`                 | Length stays within `max_bytes` (default 30 000). |
| `commands/allowed-tools-known`  | schema   | error    | command                     | `allowed-tools:` entries are valid Claude tool names. |
| `hooks/event-name-known`        | schema   | error    | hook                        | Event name matches a known Claude hook event. |
| `hooks/timeout-present`         | content  | warning  | hook                        | Every hook entry has an explicit timeout. |
| `hooks/no-unsafe-shell`         | security | warning  | hook                        | Flags `eval`, unquoted variables, and other shell smells inside `command:`. |
| `plugin/manifest-fields`        | schema   | error    | plugin                      | Plugin manifest has `name`, `version`, etc. |
| `plugin/semver`                 | schema   | warning  | plugin                      | `version:` is a valid semver. |
| `security/secrets`              | security | error    | every kind                  | Flags long high-entropy tokens and known secret prefixes (AWS, GitHub, Slack, etc.). |
| `style/no-emoji`                | style    | info     | every kind                  | Reports emoji in source; surfaces where rule authors want plain text. |

Inspect any rule's metadata with:

```bash
claudelint rules <id>
```

## Profiling

`--profile=<dir>` captures CPU, heap, block, and mutex profiles for a
single run. Use it to investigate scheduler behavior before proposing
runtime changes.

```bash
claudelint run --profile=./profile .
go tool pprof ./profile/cpu.pprof
```

`make profile` is a convenience target that runs the command against
this repo and prints the pprof command you need.

## Contributing

- `make check` runs `lint + test`.
- `make ci` runs the full CI pipeline (`lint + test + build + license-check`).
- `make self-check` dogfoods `claudelint run .` against this repo — the
  CI build fails if new code surfaces any error-severity diagnostic.
- All docs go through [docz](https://github.com/donaldgifford/docz).
  Run `docz update` after editing any doc to refresh index tables and
  in-file ToCs.

Bug reports and RFCs: <https://github.com/donaldgifford/claudelint/issues>.
