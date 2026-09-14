### Frontmatter reference

| Field | Required | Description |
| :------------------------- | :---------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `name` | No | Display name shown in skill listings. Defaults to the directory name. Se |
| `description` | Recommended | What the skill does and when to use it. Claude uses this to decide when |
| `when_to_use` | No | Additional context for when Claude should invoke the skill, such as trig |
| `argument-hint` | No | Hint shown during autocomplete to indicate expected arguments. Example: |
| `arguments` | No | Named positional arguments for [`$name` substitution](#available-string- |
| `disable-model-invocation` | No | Set to `true` to prevent Claude from automatically loading this skill. U |
| `user-invocable` | No | Set to `false` when only Claude should invoke the skill: Claude Code hid |
| `allowed-tools` | No | Tools Claude can use without asking permission during the turn that invo |
| `disallowed-tools` | No | Tools removed from Claude's available pool while this skill is active. U |
| `model` | No | Model to use when this skill is active. The override applies for the res |
| `effort` | No | [Effort level](/docs/en/model-config#adjust-effort-level) when this skil |
| `context` | No | Set to `fork` to run in a forked subagent context. See [Run skills in a |
| `agent` | No | Which subagent type to use when `context: fork` is set. |
| `background` | No | Only applies with `context: fork`. Set to `false` to wait for the forked |
| `hooks` | No | Hooks that Claude Code registers when the skill is invoked and keeps run |
| `paths` | No | Glob patterns that limit when this skill is activated. Accepts a comma-s |
| `shell` | No | Shell to use for `` !`command` `` and ` ```! |
| `metadata` | No | Free-form YAML map for your own key-value data, such as entitlement or c |
| `license` | No | License covering the skill. Part of the [Agent Skills](https://agentskil |
| `compatibility` | No | Environment requirements for the skill, such as intended products or sys |

#### Using skill frontmatter outside Claude Code

| Distribution path | Frontmatter fields you can use |
| :-------------------------------------------------------------------------------------------------------------------------------------------- | :----------------------------------------------------------------------------- |
| Claude Code skills at [any level](#where-skills-live), including [plugin](/docs/en/plugins) skills | Every field in the table above |
| claude.ai skill uploads, the Skills API, and packaging with `package_skill.py` from [anthropics/skills](https://github.com/anthropics/skills) | `name`, `description`, `license`, `compatibility`, `metadata`, |

#### How a skill gets its command name

| Skill location | Command name source | Example |
| :------------------------------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------------ | :----------------------------------------------------------------------------------------------------------------------------------- |
| Skill directory under `~/.claude/skills/` or `.claude/skills/` | Directory name | `.claude/skills/deploy-staging/SKILL.md` → `/deploy-staging` |
| [Nested](#where-skills-live) `.claude/skills/` directory, when the name clashes with another skill | Subdirectory path relative to the working directory, then the skill dire | `apps/web/.claude/skills/deploy/SKILL.md` → `/apps/web:deploy` |
| File under `.claude/commands/` | File name without extension | `.claude/commands/deploy.md` → `/deploy` |
| File in a subdirectory of `.claude/commands/` | Subdirectory path relative to `commands/` with each `/` replaced by `:`, | `.claude/commands/frontend/component.md` → `/frontend:component` |
| Plugin `skills/` subdirectory | Frontmatter `name` or the directory name, namespaced by plugin | `my-plugin/skills/review/SKILL.md` → `/my-plugin:review`, or |
| Plugin root `SKILL.md` | Frontmatter `name`, with the plugin directory name as a fallback | `my-plugin/SKILL.md` with `name: review` → `/my-plugin:review`. See [P |

#### Available string substitutions

| Variable | Description |
| :---------------------- | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `$ARGUMENTS` | All arguments passed when invoking the skill. When no placeholder receiv |
| `$ARGUMENTS[N]` | Access a specific argument by 0-based index, such as `$ARGUMENTS[0]` for |
| `$N` | Shorthand for `$ARGUMENTS[N]`, such as `$0` for the first argument or |
| `$name` | Named argument declared in the [`arguments`](#frontmatter-reference) fro |
| `${CLAUDE_SESSION_ID}` | The current session ID. Useful for logging, creating session-specific fi |
| `${CLAUDE_EFFORT}` | The current effort level: `low`, `medium`, `high`, `xhigh`, or `max`. Ul |
| `${CLAUDE_SKILL_DIR}` | The directory containing the skill's `SKILL.md` file. For plugin skills, |
| `${CLAUDE_PROJECT_DIR}` | The project root directory. This is the same path [hooks](/docs/en/hooks |
| `${CLAUDE_PLUGIN_ROOT}` | The plugin's installation directory. Substituted only in plugin skills. |
| `${CLAUDE_PLUGIN_DATA}` | The plugin's [persistent data directory](/docs/en/plugins-reference#pers |
