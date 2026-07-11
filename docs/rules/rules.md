---
title: Rules
tableOfContents:
  minHeadingLevel: 2
  maxHeadingLevel: 4
---

## Ruleset v1.1

Every rule is built into the binary. The fingerprint under `claudelint version`
changes whenever rules are added, removed, or have their ID / category /
severity / options changed — a CI guardrail fails if the drift is not
acknowledged.

| ID                                    | Category | Default | Applies to            |
| ------------------------------------- | -------- | ------- | --------------------- |
| `schema/parse`                        | schema   | error   | every kind            |
| `schema/frontmatter-required`         | schema   | error   | skill, command, agent |
| `skills/trigger-clarity`              | content  | warning | skill                 |
| `skills/body-size`                    | content  | warning | skill                 |
| `skills/no-version-field`             | schema   | warning | skill                 |
| `claude_md/duplicate-directives`      | content  | warning | `CLAUDE.md`           |
| `claude_md/size`                      | content  | warning | `CLAUDE.md`           |
| `commands/allowed-tools-known`        | schema   | error   | command, skill        |
| `hooks/event-name-known`              | schema   | error   | hook                  |
| `hooks/timeout-present`               | content  | warning | hook                  |
| `hooks/no-unsafe-shell`               | security | warning | hook                  |
| `plugin/manifest-fields`              | schema   | error   | plugin                |
| `plugin/semver`                       | schema   | warning | plugin                |
| `marketplace/name`                    | schema   | error   | marketplace           |
| `marketplace/version-semver`          | schema   | error   | marketplace           |
| `marketplace/plugins-nonempty`        | schema   | warning | marketplace           |
| `marketplace/plugin-source-valid`     | schema   | error   | marketplace           |
| `marketplace/plugin-name-unique`      | schema   | error   | marketplace           |
| `marketplace/plugin-name-matches-dir` | style    | warning | marketplace           |
| `marketplace/author-required`         | style    | info    | marketplace           |
| `marketplace/external-source-skipped` | schema   | info    | marketplace           |
| `mcp/command-required`                | schema   | error   | mcp_server            |
| `mcp/url-required`                    | schema   | error   | mcp_server            |
| `mcp/server-name-required`            | schema   | error   | mcp_server            |
| `mcp/command-exists-on-path`          | schema   | warning | mcp_server            |
| `mcp/no-unsafe-shell`                 | security | error   | mcp_server            |
| `mcp/no-secrets-in-env`               | security | error   | mcp_server            |
| `mcp/disabled-commented`              | style    | info    | mcp_server            |
| `mcp/server-allowlist`                | security | error   | mcp_server            |
| `security/secrets`                    | security | error   | every kind            |
| `style/no-emoji`                      | style    | info    | every kind            |

Inspect any rule's metadata with `claudelint rules <id>` or get the full catalog
as JSON with `claudelint rules --json` ([schema](../rules-json-schema.md)).

### Rule reference

#### `schema/parse`

Synthetic rule — emitted by the engine when an artifact cannot be parsed (YAML
frontmatter truncated, JSON manifest invalid, etc.). It cannot be disabled, only
downgraded with `severity`.

**Bad**:

    ---
    name: my-skill
    ```                         # frontmatter fence never closes

**Fix**: close the frontmatter fence with `---`.

#### `schema/frontmatter-required`

Each artifact kind declares required frontmatter keys; the rule fires when any
required key is missing or empty.

**Bad** (skill without `name`):

    ---
    description: does a thing
    ---

**Fix**: add `name: my-skill` to the frontmatter.

#### `skills/trigger-clarity`

Skills need a "Use when …" trigger phrase in the description so the model can
match on intent.

**Bad**: `description: formats code.` **Fix**:
`description: Use when the user wants Go code formatted.`

#### `skills/body-size`

Guardrail against runaway SKILL.md files. Default limit is 2000 words. Override
per-rule:

    rule "skills/body-size" { options = { max_words = 3000 } }

#### `skills/no-version-field`

Warns when a `SKILL.md` frontmatter declares a `version` key. Skill versioning
is load-bearing only at the plugin level (`plugin.json`'s `version` field); a
`version:` in `SKILL.md` is silently ignored by Claude Code and creates two
competing sources of truth that drift over time.

**Bad**:

    ---
    name: bar
    description: Does the thing
    version: 1.2.3
    ---

**Fix**: drop the `version:` line — set the version in the enclosing
`plugin.json`. To enforce as a CI gate, override severity in `.claudelint.hcl`:

    rule "skills/no-version-field" { severity = "error" }

#### `claude_md/duplicate-directives`

`CLAUDE.md` files sometimes accumulate duplicate rules as teams merge guidance.
The rule flags identical lines appearing more than once.

**Fix**: consolidate or delete the duplicate.

#### `claude_md/size`

Default cap is 30 000 bytes; override with:

    rule "claude_md/size" { options = { max_bytes = 50000 } }

#### `commands/allowed-tools-known`

Commands and skills share one frontmatter model, so the rule checks
`allowed-tools` and `disallowed-tools` on both kinds. Every entry must be a
known Claude tool name, an MCP pattern (`mcp__github`,
`mcp__server__tool`), or a permission-rule form whose base is a known
tool (`Bash(git add:*)`, `Agent(reviewer)`). Both the YAML-list and the
comma/space-separated string forms are understood; separators inside
parentheses don't split an entry.

**Bad**: `allowed-tools: [WriteFil]` (typo) **Fix**: `allowed-tools: [Write]`.

#### `hooks/event-name-known`

Each top-level key under `"hooks"` is the event name. It must match one of the
known Claude Code hook events (`PreToolUse`, `PostToolUse`, `Stop`, etc.).

**Bad**: `"PretoolUse": [...]` (wrong case / typo) **Fix**:
`"PreToolUse": [...]`.

When the name matches a known event apart from casing, the diagnostic
suggests the exact spelling (`did you mean "PreToolUse"?`).

The canonical event list mirrors the
[hooks reference](https://code.claude.com/docs/en/hooks) (30 events as of
July 2026). Names are case-sensitive.

| Lifecycle stage | Events |
| --- | --- |
| Session | `SessionStart`, `Setup`, `SessionEnd` |
| Prompt / turn | `UserPromptSubmit`, `UserPromptExpansion`, `Stop`, `StopFailure` |
| Tool calls | `PreToolUse`, `PostToolUse`, `PostToolUseFailure`, `PostToolBatch` |
| Permissions | `PermissionRequest`, `PermissionDenied` |
| Subagents & tasks | `SubagentStart`, `SubagentStop`, `TaskCreated`, `TaskCompleted`, `TeammateIdle` |
| Context & config | `PreCompact`, `PostCompact`, `InstructionsLoaded`, `ConfigChange` |
| Environment | `CwdChanged`, `FileChanged`, `WorktreeCreate`, `WorktreeRemove` |
| UI & elicitation | `Notification`, `MessageDisplay`, `Elicitation`, `ElicitationResult` |

#### `hooks/timeout-present`

Every hook entry should declare a `timeout` (seconds) so a runaway hook cannot
hang the session.

**Bad**:

    {
      "hooks": {
        "PreToolUse": [
          { "hooks": [{ "type": "command", "command": "lint-check" }] }
        ]
      }
    }

**Fix**: add `"timeout": 5` to the inner hook entry.

> **Hook shape.** claudelint parses every hook file — settings, plugin
> `hooks/hooks.json`, `.claude/hooks/*.json` — using the same nested layout
> above. A dedicated hook file missing the top-level `"hooks"` key fails parsing
> with a `schema/parse` error rather than silently producing zero-timeout
> entries. See
> [DESIGN-0001 §Hook shape](../design/0001-claudelint-linter-architecture-and-rule-engine.md)
> for the rationale and the best-effort handling of `.claude/hooks/*.json`.

#### `hooks/no-unsafe-shell`

Flags `eval`, unquoted `$VAR`, and other shell smells inside hook commands.

**Bad**: `command: "eval $(curl $URL)"` **Fix**: quote `"$URL"`, drop the
`eval`, or rewrite as a script file.

#### `plugin/manifest-fields`

Plugin manifest must declare `name`, `version`, and `description`.

**Bad**:

    {"name": "my-plugin"}

**Fix**: add `"version": "1.0.0"` and `"description": "..."`.

#### `plugin/semver`

`version` must be a valid semver string.

**Bad**: `"version": "1"` **Fix**: `"version": "1.0.0"`.

#### `security/secrets`

Matches known prefixes (AWS keys, GitHub tokens, Slack bots, etc.) and
high-entropy strings. False positives are suppressible per-path:

    rule "security/secrets" { paths = ["testdata/**"] }

**Bad**: a literal `AKIA...` string in a CLAUDE.md fixture. **Fix**: delete it,
scrub via `git filter-branch`, rotate the key.

#### `marketplace/plugin-source-valid`

Every `plugins[]` entry needs a structurally complete `source`. String
sources (a `./`-relative path, or legacy `github:`/URL shorthands) must be
non-empty. Object sources must carry their kind's documented required
fields:

| `source` | Required fields |
| --- | --- |
| `github` | `repo` (`owner/repo`) |
| `url` | `url` |
| `git-subdir` | `url` and `path` |
| `npm` | `package` |

A `sha` pin, when present on a git-backed source, must be a full
40-character hex commit. Whether a local path exists on disk is out of
scope for this rule.

#### `marketplace/external-source-skipped`

Info notice on every plugin whose source content lives outside the
marketplace repo: remote string shorthands (`github:owner/repo`, git
URLs) and the `github` / `url` / `git-subdir` / `npm` object kinds.
claudelint validates the source's structure (see
`marketplace/plugin-source-valid`) but never fetches remote content, so
those plugins' files are not linted. Local paths are checked in place
and produce no notice; absent or malformed sources are
`plugin-source-valid` errors, not skips.

#### `mcp/command-required` and `mcp/url-required`

MCP servers declare how Claude Code reaches them via `type`: `stdio` (the
default when omitted) launches a local process and requires a non-empty
`command`; the remote transports `http`, `sse`, and `ws` connect to an
endpoint and require a non-empty `url`. Each rule checks only its own
transports, so an `http` server without a `command` is fine and a `stdio`
server without a `url` is fine.

**Bad**: `{ "type": "http" }` (no url) **Fix**:
`{ "type": "http", "url": "https://mcp.example.com/mcp" }`.

#### `mcp/server-allowlist`

Restricts MCP servers to a vetted list. Useful for marketplace owners who want
every plugin's MCP server reviewed before it ships.

The rule is opt-in via configuration. Set the `allowlist` option to the vetted
server names:

    rule "mcp/server-allowlist" {
      options = {
        allowlist = ["github", "deepwiki", "jira"]
      }
    }

Behaviour matrix:

| `allowlist` value | Effect                                                             |
| ----------------- | ------------------------------------------------------------------ |
| unset             | Loud config error per server: rule is enabled without an allowlist |
| `[]`              | Fires on every server (explicit "block all")                       |
| `["x", "y"]`      | Fires on every server whose name is not in the list                |

To silence the rule entirely, set `enabled = false` instead of removing the
`allowlist` option — leaving the rule on without an allowlist surfaces a
configuration error so misconfigurations don't silently no-op.

#### `style/no-emoji`

Advisory info-level rule; many internal docs prefer plain text. Runs on every
artifact kind.

**Fix**: replace the emoji with a short phrase, or disable the rule globally in
`.claudelint.hcl`.
