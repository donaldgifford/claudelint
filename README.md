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

Multi-arch container images (linux/amd64, linux/arm64) ship alongside
each release at `ghcr.io/donaldgifford/claudelint`. See
[Running in CI](#running-in-ci) below.

## Quickstart

Lint the current repo:

```bash
claudelint run .
```

Emit GitHub Actions annotations on your PR (preferred: use the
companion [`claudelint-action`](https://github.com/donaldgifford/claudelint-action)
which also handles install + SARIF upload):

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
- `--format=json` — stable schema documented in [docs/json-output-schema.md](docs/json-output-schema.md).
- `--format=github` — `::error` / `::warning` / `::notice` workflow commands.
- `--format=sarif` — SARIF 2.1.0 log, suitable for GitHub Code Scanning
  and other SARIF-aware tools. Pair with `--sarif-file=<path>` to write
  to a file instead of stdout.

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

The image pins a tag per release (`:0.1.0`, `:v0`, `:v0.1`, `:latest`).
Use a pinned tag in scheduled pipelines to avoid surprises when
claudelint ships a new rule.

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

## Rules (ruleset v1.1)

Every rule is built into the binary. The fingerprint under
`claudelint version` changes whenever rules are added, removed, or
have their ID / category / severity / options changed — a CI guardrail
fails if the drift is not acknowledged.

| ID                                    | Category  | Default  | Applies to                      |
|---------------------------------------|-----------|----------|---------------------------------|
| `schema/parse`                        | schema    | error    | every kind                      |
| `schema/frontmatter-required`         | schema    | error    | skill, command, agent           |
| `skills/trigger-clarity`              | content   | warning  | skill                           |
| `skills/body-size`                    | content   | warning  | skill                           |
| `claude_md/duplicate-directives`      | content   | warning  | `CLAUDE.md`                     |
| `claude_md/size`                      | content   | warning  | `CLAUDE.md`                     |
| `commands/allowed-tools-known`        | schema    | error    | command                         |
| `hooks/event-name-known`              | schema    | error    | hook                            |
| `hooks/timeout-present`               | content   | warning  | hook                            |
| `hooks/no-unsafe-shell`               | security  | warning  | hook                            |
| `plugin/manifest-fields`              | schema    | error    | plugin                          |
| `plugin/semver`                       | schema    | warning  | plugin                          |
| `marketplace/name`                    | schema    | error    | marketplace                     |
| `marketplace/version-semver`          | schema    | error    | marketplace                     |
| `marketplace/plugins-nonempty`        | schema    | warning  | marketplace                     |
| `marketplace/plugin-source-valid`     | schema    | error    | marketplace                     |
| `marketplace/plugin-name-unique`      | schema    | error    | marketplace                     |
| `marketplace/plugin-name-matches-dir` | style     | warning  | marketplace                     |
| `marketplace/author-required`         | style     | info     | marketplace                     |
| `marketplace/external-source-skipped` | schema    | info     | marketplace                     |
| `mcp/command-required`                | schema    | error    | mcp_server                      |
| `mcp/server-name-required`            | schema    | error    | mcp_server                      |
| `mcp/command-exists-on-path`          | schema    | warning  | mcp_server                      |
| `mcp/no-unsafe-shell`                 | security  | error    | mcp_server                      |
| `mcp/no-secrets-in-env`               | security  | error    | mcp_server                      |
| `mcp/disabled-commented`              | style     | info     | mcp_server                      |
| `mcp/server-allowlist`                | security  | error    | mcp_server                      |
| `security/secrets`                    | security  | error    | every kind                      |
| `style/no-emoji`                      | style     | info     | every kind                      |

Inspect any rule's metadata with `claudelint rules <id>` or get the
full catalog as JSON with `claudelint rules --json`
([schema](docs/rules-json-schema.md)).

### Rule reference

#### `schema/parse`

Synthetic rule — emitted by the engine when an artifact cannot be
parsed (YAML frontmatter truncated, JSON manifest invalid, etc.). It
cannot be disabled, only downgraded with `severity`.

**Bad**:

    ---
    name: my-skill
    ```                         # frontmatter fence never closes

**Fix**: close the frontmatter fence with `---`.

#### `schema/frontmatter-required`

Each artifact kind declares required frontmatter keys; the rule fires
when any required key is missing or empty.

**Bad** (skill without `name`):

    ---
    description: does a thing
    ---

**Fix**: add `name: my-skill` to the frontmatter.

#### `skills/trigger-clarity`

Skills need a "Use when …" trigger phrase in the description so the
model can match on intent.

**Bad**: `description: formats code.`
**Fix**: `description: Use when the user wants Go code formatted.`

#### `skills/body-size`

Guardrail against runaway SKILL.md files. Default limit is 2000 words.
Override per-rule:

    rule "skills/body-size" { options = { max_words = 3000 } }

#### `claude_md/duplicate-directives`

`CLAUDE.md` files sometimes accumulate duplicate rules as teams merge
guidance. The rule flags identical lines appearing more than once.

**Fix**: consolidate or delete the duplicate.

#### `claude_md/size`

Default cap is 30 000 bytes; override with:

    rule "claude_md/size" { options = { max_bytes = 50000 } }

#### `commands/allowed-tools-known`

Slash-command manifests declare `allowed-tools`; the rule checks every
entry is a valid Claude tool name from the shipping set.

**Bad**: `allowed-tools: [WriteFil]` (typo)
**Fix**: `allowed-tools: [Write]`.

#### `hooks/event-name-known`

Each top-level key under `"hooks"` is the event name. It must match
one of the known Claude Code hook events (`PreToolUse`, `PostToolUse`,
`Stop`, etc.).

**Bad**: `"PretoolUse": [...]` (wrong case / typo)
**Fix**: `"PreToolUse": [...]`.

#### `hooks/timeout-present`

Every hook entry should declare a `timeout` (seconds) so a runaway
hook cannot hang the session.

**Bad**:

```json
{
  "hooks": {
    "PreToolUse": [
      { "hooks": [{ "type": "command", "command": "lint-check" }] }
    ]
  }
}
```

**Fix**: add `"timeout": 5` to the inner hook entry.

> **Hook shape.** claudelint parses every hook file — settings,
> plugin `hooks/hooks.json`, `.claude/hooks/*.json` — using the same
> nested layout above. A dedicated hook file missing the top-level
> `"hooks"` key fails parsing with a `schema/parse` error rather
> than silently producing zero-timeout entries. See
> [DESIGN-0001 §Hook shape](docs/design/0001-claudelint-linter-architecture-and-rule-engine.md)
> for the rationale and the best-effort handling of `.claude/hooks/*.json`.

#### `hooks/no-unsafe-shell`

Flags `eval`, unquoted `$VAR`, and other shell smells inside hook
commands.

**Bad**: `command: "eval $(curl $URL)"`
**Fix**: quote `"$URL"`, drop the `eval`, or rewrite as a script file.

#### `plugin/manifest-fields`

Plugin manifest must declare `name`, `version`, and `description`.

**Bad**:

    {"name": "my-plugin"}

**Fix**: add `"version": "1.0.0"` and `"description": "..."`.

#### `plugin/semver`

`version` must be a valid semver string.

**Bad**: `"version": "1"`
**Fix**: `"version": "1.0.0"`.

#### `security/secrets`

Matches known prefixes (AWS keys, GitHub tokens, Slack bots, etc.) and
high-entropy strings. False positives are suppressible per-path:

    rule "security/secrets" { paths = ["testdata/**"] }

**Bad**: a literal `AKIA...` string in a CLAUDE.md fixture.
**Fix**: delete it, scrub via `git filter-branch`, rotate the key.

#### `mcp/server-allowlist`

Restricts MCP servers to a vetted list. Useful for marketplace owners
who want every plugin's MCP server reviewed before it ships.

The rule is opt-in via configuration. Set the `allowlist` option to
the vetted server names:

    rule "mcp/server-allowlist" {
      options = {
        allowlist = ["github", "deepwiki", "jira"]
      }
    }

Behaviour matrix:

| `allowlist` value | Effect                                                              |
|-------------------|---------------------------------------------------------------------|
| unset             | Loud config error per server: rule is enabled without an allowlist  |
| `[]`              | Fires on every server (explicit "block all")                        |
| `["x", "y"]`      | Fires on every server whose name is not in the list                 |

To silence the rule entirely, set `enabled = false` instead of
removing the `allowlist` option — leaving the rule on without an
allowlist surfaces a configuration error so misconfigurations don't
silently no-op.

#### `style/no-emoji`

Advisory info-level rule; many internal docs prefer plain text. Runs
on every artifact kind.

**Fix**: replace the emoji with a short phrase, or disable the rule
globally in `.claudelint.hcl`.

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
