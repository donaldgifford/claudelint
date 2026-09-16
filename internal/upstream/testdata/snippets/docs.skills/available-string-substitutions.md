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
