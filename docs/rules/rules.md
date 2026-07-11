---
title: Rules
tableOfContents:
  minHeadingLevel: 2
  maxHeadingLevel: 4
---

## Ruleset v1.4

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
| `agents/field-enums`                  | schema   | warning | agent                 |
| `agents/model-policy` (opt-in)        | schema   | error   | agent                 |
| `agents/model-valid`                  | schema   | warning | agent, skill, command |
| `agents/name-format`                  | schema   | warning | agent                 |
| `agents/plugin-ignored-fields`        | content  | warning | agent                 |
| `agents/tools-known`                  | schema   | warning | agent                 |
| `claude_md/duplicate-directives`      | content  | warning | `CLAUDE.md`           |
| `claude_md/size`                      | content  | warning | `CLAUDE.md`           |
| `commands/allowed-tools-known`        | schema   | error   | command, skill        |
| `hooks/event-name-known`              | schema   | error   | hook                  |
| `hooks/type-known`                    | schema   | error   | hook                  |
| `hooks/type-fields`                   | schema   | error   | hook                  |
| `hooks/timeout-present`               | content  | warning | hook                  |
| `hooks/no-unsafe-shell`               | security | warning | hook                  |
| `plugin/manifest-fields`              | schema   | error   | plugin                |
| `plugin/semver`                       | schema   | warning | plugin                |
| `marketplace/name`                    | schema   | error   | marketplace           |
| `marketplace/reserved-name`           | schema   | error   | marketplace           |
| `marketplace/version-semver`          | schema   | error   | marketplace           |
| `marketplace/plugins-nonempty`        | schema   | warning | marketplace           |
| `marketplace/plugin-source-valid`     | schema   | error   | marketplace           |
| `marketplace/plugin-name-unique`      | schema   | error   | marketplace           |
| `marketplace/plugin-name-matches-dir` | style    | warning | marketplace           |
| `marketplace/name-format`             | style    | warning | marketplace           |
| `marketplace/source-path-safety`      | security | error   | marketplace           |
| `marketplace/renames-valid`           | schema   | error   | marketplace           |
| `marketplace/owner-required`          | schema   | warning | marketplace           |
| `marketplace/author-legacy`           | style    | info    | marketplace           |
| `marketplace/version-missing`         | style    | info    | marketplace           |
| `marketplace/external-source-skipped` | schema   | info    | marketplace           |
| `mcp/command-required`                | schema   | error   | mcp_server            |
| `mcp/url-required`                    | schema   | error   | mcp_server            |
| `mcp/server-name-required`            | schema   | error   | mcp_server            |
| `mcp/command-exists-on-path`          | schema   | warning | mcp_server            |
| `mcp/no-unsafe-shell`                 | security | error   | mcp_server            |
| `mcp/no-secrets-in-env`               | security | error   | mcp_server            |
| `mcp/no-secrets-in-headers`           | security | error   | mcp_server            |
| `mcp/timeout-minimum`                 | schema   | warning | mcp_server            |
| `mcp/transport-known`                 | schema   | warning | mcp_server            |
| `mcp/transport-deprecated`            | schema   | info    | mcp_server            |
| `mcp/disabled-commented`              | style    | info    | mcp_server            |
| `mcp/legacy-servers-key`              | schema   | info    | mcp_server            |
| `mcp/server-allowlist` (opt-in)       | security | error   | mcp_server            |
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

> **Stricter than spec, by design.** Claude Code itself tolerates a skill
> without `name` (it falls back to the directory name) and loads one
> without `description`, but a skill the model can't reliably discover is
> a skill that silently never runs — so claudelint keeps both required at
> error severity. The skill diagnostics phrase this as best practice and
> name the runtime fallback.

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

Warns when a `SKILL.md` frontmatter declares a `version` key. The skills
reference documents `version` as accepted-but-ignored metadata; versioning is
load-bearing only at the plugin level (`plugin.json`'s `version` field), so a
`version:` in `SKILL.md` is a second source of truth that drifts over time
without ever taking effect.

**Bad**:

    ---
    name: bar
    description: Does the thing
    version: 1.2.3
    ---

**Fix**: drop the `version:` line — set the version in the enclosing
`plugin.json`. To enforce as a CI gate, override severity in `.claudelint.hcl`:

    rule "skills/no-version-field" { severity = "error" }

#### `agents/field-enums`

Agent frontmatter has five closed-enum fields; a value outside the
documented set silently falls back to the default at runtime. Checked
sets: `permissionMode` (`default`, `acceptEdits`, `auto`, `dontAsk`,
`bypassPermissions`, `plan`, `manual`), `effort` (`low`, `medium`,
`high`, `xhigh`, `max`), `color` (`red`, `blue`, `green`, `yellow`,
`purple`, `orange`, `pink`, `cyan`), `isolation` (`worktree`), and
`memory` (`user`, `project`, `local`). Absent keys are fine — every
field has a default.

**Bad**: `effort: turbo` **Fix**: `effort: high`.

#### `agents/model-policy`

Governance policy over agent `model` declarations, for teams that want
to pin which models agents may request.

**Opt-in.** Without a `rule "agents/model-policy"` block in
`.claudelint.hcl` the rule does not run at all. Once a block exists,
exactly one option must be set:

    rule "agents/model-policy" {
      options = {
        require = "inherit"
      }
    }

- `require = "inherit"` — every agent must inherit the session model:
  compliant when `model` is absent or `inherit`; anything else fires.
- `allowlist = ["opus", "claude-sonnet-5", "inherit"]` — a declared
  `model` must be in the list; an absent key is evaluated as `inherit`
  (the documented default).

Both options set, neither set, `require` set to anything but
`"inherit"`, or an allowlist entry that is not a valid model reference
all produce a loud config-error diagnostic per agent — an explicit
enable that cannot be evaluated is a misconfiguration, not a silent
no-op.

#### `agents/model-valid`

A declared `model` must be a documented reference: an alias (`sonnet`,
`opus`, `haiku`, `fable`), `inherit`, or a full model ID
(`claude-...`). Anything else silently falls back to the inherited
model at runtime — the typo never surfaces. Omitting the key is fine
(omitted means inherit). Skills and commands share the `model` field,
so the rule runs on all three kinds.

**Bad**: `model: sonet` **Fix**: `model: sonnet`.

#### `agents/name-format`

Agent names are documented as lowercase letters and hyphens. Hooks
receive the value as `agent_type`, and duplicate names across scopes
resolve by undocumented filesystem order — nonconforming names misbehave
in hard-to-debug ways.

**Bad**: `name: Code_Reviewer` **Fix**: `name: code-reviewer`.

#### `agents/plugin-ignored-fields`

Claude Code ignores `permissionMode`, `mcpServers`, and `hooks` on
subagents distributed inside a plugin (any agent file with a
`.claude-plugin/plugin.json` in an ancestor directory). Declaring them
there is dead config that reads as if it took effect. Each declared
field gets its own diagnostic anchored at the key. Project-level
(`.claude/agents/`) and user-level agents may declare all three
freely — the rule stays silent for them.

**Fix**: delete the ignored keys, or move the agent out of the plugin
if it genuinely needs them.

#### `agents/tools-known`

Entries in an agent's `tools` and `disallowedTools` must be known tool
names, MCP patterns (`mcp__server`, `mcp__server__tool`), or
permission-rule forms (`Bash(git diff:*)`). Claude Code silently
ignores unknown names, so a typo widens or narrows the agent's tool
access with no runtime signal.

**Bad**: `tools: Read, Wrte` **Fix**: `tools: Read, Write`.

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

#### `hooks/type-known`

A declared hook `type` must be one of the five documented values:
`command`, `http`, `mcp_tool`, `prompt`, `agent`. Claude Code treats
anything else as misconfiguration and the hook never fires. Omitting
the key is fine — it defaults to `command`.

**Bad**: `"type": "webhook"` **Fix**: `"type": "http"`.

#### `hooks/type-fields`

Each hook type has a required payload field, and an entry without it
is silently inert: `command` hooks need `command`, `http` hooks need
`url`, `mcp_tool` hooks need both `server` and `tool`, and `prompt` /
`agent` hooks need `prompt`. One diagnostic per missing field; entries
with an unknown declared `type` are left to `hooks/type-known`.

**Bad**: `{"type": "http", "timeout": 10}` **Fix**: add `"url": "..."`.

#### `hooks/timeout-present`

Every hook entry should declare a `timeout` (seconds). Claude Code applies
documented per-type defaults when it's omitted — 600 s for `command`,
`http`, and `mcp_tool` hooks, 30 s for `prompt`, 60 s for `agent` — so a
missing timeout won't hang the session, but an explicit one fails faster
in CI and records the latency you actually expect.

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

Flags network-download-into-shell pipes (`curl ... | sh` and variants)
inside hook commands. Only shell-form `command` hooks are checked:
non-`command` types have no shell, and entries with `args` run exec-form
(argv spawned directly), so pipe characters there are literal arguments,
not shell syntax.

**Bad**: `command: "curl https://get.example.sh | bash"` **Fix**: vendor
the script into the repo and run it directly, or verify a pinned
checksum before executing.

#### `plugin/manifest-fields`

Plugin manifests must declare `name` and `version`.

> **Stricter than spec, by design.** The docs require only `name`; a
> missing `version` falls back to the installing commit's git SHA.
> claudelint requires a pinned `version` anyway — explicit versions make
> update semantics reviewable for marketplace consumers. Downgrade the
> rule in `.claudelint.hcl` if you prefer the SHA fallback.

**Bad**:

    {"name": "my-plugin"}

**Fix**: add `"version": "1.0.0"`.

#### `plugin/semver`

`version` must be a valid semver string.

**Bad**: `"version": "1"` **Fix**: `"version": "1.0.0"`.

#### `security/secrets`

Matches known prefixes (AWS keys, GitHub tokens, Slack bots, etc.) and
high-entropy strings. False positives are suppressible per-path:

    rule "security/secrets" { paths = ["testdata/**"] }

**Bad**: a literal `AKIA...` string in a CLAUDE.md fixture. **Fix**: delete it,
scrub via `git filter-branch`, rotate the key.

#### `marketplace/name-format`

The marketplace `name` and every `plugins[].name` should be kebab-case
— lowercase alphanumeric segments joined by single hyphens. Names are
public-facing (`/plugin install <plugin>@<marketplace>`) and claude.ai
sync rejects violations. Empty names are left to `marketplace/name`
and `marketplace/plugin-source-valid`.

**Bad**: `"name": "Acme Tools"` **Fix**: `"name": "acme-tools"`.

#### `marketplace/source-path-safety`

Local plugin sources must start with `./` and must not contain `..`
segments — Claude Code's validator rejects both, and a `..` path that
slipped through would read outside the marketplace repo. When the
manifest declares `metadata.pluginRoot`, bare sources (`"formatter"`)
are documented-valid and only the `..` check applies. Remote source
shapes are out of scope.

**Bad**: `"source": "../shared/plugin"` **Fix**: move the plugin into
the repo and reference it with `./`.

#### `marketplace/renames-valid`

Every `renames{}` migration chain must terminate — at `null` (plugin
removed) or at a name listed in `plugins[]` — and no chain may cycle.
A dangling or cyclic rename strands existing installs mid-migration:
Claude Code follows the chain to find the current plugin and never
arrives. One diagnostic per broken link; each cycle is reported once.
The parser stores no per-entry ranges for `renames{}`, so diagnostics
anchor at the manifest's `name` field.

**Bad**: `"renames": {"old-name": "ghost"}` where `ghost` isn't
listed. **Fix**: point the rename at the current entry name, or use
`null` if the plugin was removed.

#### `marketplace/reserved-name`

Sixteen marketplace names are reserved for official Anthropic use
(e.g. `anthropic-plugins`, `claude-code-marketplace`, `agent-skills`,
`healthcare`). Claude Code re-checks the list on every load, so a
manifest shipping one stops loading for every user. Exact match only —
impersonation lookalikes (`official-claude-plugins`) are blocked
server-side by claude.ai, and this rule deliberately does not attempt
those heuristics.

**Fix**: pick a name that identifies you or your org
(`acme-tools`).

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

#### `marketplace/owner-required` and `marketplace/author-legacy`

The docs require a root `owner` object (`{"name": ..., "email": ...}`)
identifying the marketplace maintainer. `owner-required` warns when the
manifest names no maintainer at all; the legacy top-level `author`
string (documented backward-compat) satisfies it. `author-legacy` adds
an info nudge when *only* the legacy string is present — rename it to
the `owner{}` shape. These replace the pre-v1.3.0
`marketplace/author-required` rule.

#### `marketplace/version-semver` and `marketplace/version-missing`

The docs make the root `version` optional, so its absence is only an
info nudge (`version-missing`) — a declared version lets consumers order
releases. A version that **is** declared must be valid semver
(`version-semver`, error): `MAJOR.MINOR.PATCH` with optional prerelease
/ build metadata and an optional leading `v`.

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

#### `mcp/no-secrets-in-headers`

Same detector as `mcp/no-secrets-in-env`, applied to a remote
server's `headers{}` values. Kept as a separate rule (not folded into
the env rule) so the two surfaces can be suppressed independently;
both share the `security/secrets` matcher, so the regex tables live
in one place. Placeholders like `Bearer ${API_KEY}` pass — only
credential-looking literals flag.

**Fix**: use `headersHelper` or an environment reference instead of a
literal token.

#### `mcp/timeout-minimum`

A server's `timeout` is in **milliseconds** with a documented minimum
of 1000. Nearly every sub-1000 value in the wild is someone writing
seconds (`"timeout": 30`), so the diagnostic leads with the unit hint
and suggests the ×1000 value. Absent is fine.

**Bad**: `"timeout": 30` **Fix**: `"timeout": 30000`.

#### `mcp/transport-known` and `mcp/transport-deprecated`

A declared server `type` must be one of the four documented
transports: `stdio`, `http`, `sse`, `ws` — Claude Code cannot connect
over anything else, so the server silently never loads
(`transport-known`, warning). Omitting the key defaults to `stdio`.
`sse` parses as known but is documented-deprecated in favor of
`http`; that advisory is a separate info-severity rule
(`transport-deprecated`) because the engine assigns one severity per
rule.

**Bad**: `"type": "grpc"` **Fix**: `"type": "http"`.

#### `mcp/legacy-servers-key`

`.mcp.json` files historically used a top-level `servers{}` key; the docs
standardized on `mcpServers{}`. claudelint accepts both (when both are
present, `mcpServers` wins and `servers` is ignored), and this rule emits
one info notice per legacy-keyed file. Support for `servers` is scheduled
to drop at the next major ruleset revision — rename the key now.

#### `mcp/server-allowlist`

Restricts MCP servers to a vetted list. Useful for marketplace owners who want
every plugin's MCP server reviewed before it ships.

**Opt-in.** Without a `rule "mcp/server-allowlist"` block in
`.claudelint.hcl` the rule does not run at all. Any block — even an
empty one — activates it; set the `allowlist` option to the vetted
server names:

    rule "mcp/server-allowlist" {
      options = {
        allowlist = ["github", "deepwiki", "jira"]
      }
    }

Behaviour matrix, once a rule block exists:

| `allowlist` value | Effect                                                             |
| ----------------- | ------------------------------------------------------------------ |
| unset             | Loud config error per server: rule is enabled without an allowlist |
| `[]`              | Fires on every server (explicit "block all")                       |
| `["x", "y"]`      | Fires on every server whose name is not in the list                |

An explicit enable without an allowlist is treated as a
misconfiguration, not a silent no-op. To turn the rule back off, remove
the block or set `enabled = false` inside it.

#### `style/no-emoji`

Advisory info-level rule; many internal docs prefer plain text. Runs on every
artifact kind.

**Fix**: replace the emoji with a short phrase, or disable the rule globally in
`.claudelint.hcl`.
