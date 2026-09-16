## Plugin manifest schema

### Complete schema

### Required fields

| Field | Type | Description | Example |
| :----- | :----- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :------------------- |
| `name` | string | Unique identifier in kebab-case, with no spaces, control characters, or | `"deployment-tools"` |

### Unrecognized fields

### Metadata fields

| Field | Type | Description | Example |
| :--------------- | :------ | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :---------------------------------------------------------------- |
| `$schema` | string | JSON Schema URL for editor autocomplete and validation. Claude Code igno | `"https://json.schemastore.org/claude-code-plugin-manifest.json"` |
| `displayName` | string | Human-readable name shown in the `/plugin` picker and other UI surfaces. | `"Deployment Tools"` |
| `version` | string | Optional. Semantic version. Setting this pins the plugin to that version | `"2.1.0"` |
| `description` | string | Brief explanation of plugin purpose | `"Deployment automation tools"` |
| `author` | object | Author information | `{"name": "Dev Team", "email": "dev@company.com"}` |
| `homepage` | string | Documentation URL | `"https://docs.example.com"` |
| `repository` | string | Source code URL | `"https://github.com/user/plugin"` |
| `license` | string | License identifier | `"MIT"`, `"Apache-2.0"` |
| `keywords` | array | Discovery tags | `["deployment", "ci-cd"]` |
| `metadata` | object | Free-form object for your own data, such as entitlement or catalog field | `{"catalogId": "cat-123"}` |
| `defaultEnabled` | boolean | Whether the plugin starts in an enabled state when the user has not set | `false` |

### Default enablement

### Component path fields

| Field | Type | Description | Example |
| :---------------------- | :-------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | :--------------------------------------------------- |
| `skills` | string|array | Custom skill directories containing `<name>/SKILL.md`. Adds to the defau | `"./custom/skills/"` |
| `commands` | string|array | Custom flat `.md` skill files or directories (replaces default | `"./custom/cmd.md"` or `["./cmd1.md"]` |
| `agents` | string|array | Custom agent files (replaces default `agents/`) | `"./custom/agents/reviewer.md"` |
| `workflows` | string|array | Custom [workflow](/docs/en/workflows) script files or directories (repla | `"./custom/workflows/"` |
| `hooks` | string|array|object | Hook config paths or inline config | `"./my-extra-hooks.json"` |
| `mcpServers` | string|array|object | MCP config paths or inline config | `"./my-extra-mcp-config.json"` |
| `outputStyles` | string|array | Custom output style files/directories (replaces default `output-styles/` | `"./styles/"` |
| `lspServers` | string|array|object | [Language Server Protocol](https://microsoft.github.io/language-server-p | `"./.lsp.json"` |
| `experimental.themes` | string|array | Color theme files/directories (replaces default `themes/`). See [Themes] | `"./themes/"` |
| `experimental.monitors` | string|array | Background [Monitor](/docs/en/tools-reference#monitor-tool) configuratio | `"./monitors.json"` |
| `experimental.evals` | string|array | Directory below the plugin root that holds the plugin's [eval cases](/do | `"quality/evals"` |
| `userConfig` | object | User-configurable values prompted at enable time. See [User configuratio | See below |
| `channels` | array | Channel declarations for message injection (Telegram, Slack, Discord sty | See below |
| `dependencies` | array | Other plugins this plugin requires, optionally with semver version const | `[{ "name": "secrets-vault", "version": "~2.1.0" }]` |

### Experimental components

### User configuration

| Field | Required | Description |
| :------------ | :------- | :--------------------------------------------------------------------------------------- |
| `type` | Yes | One of `string`, `number`, `boolean`, `directory`, or `file` |
| `title` | Yes | Label shown in the configuration dialog |
| `description` | Yes | Help text shown beneath the field |
| `sensitive` | No | If `true`, masks input and stores the value in secure storage instead of |
| `required` | No | If `true`, validation fails when the field is empty |
| `default` | No | Value used when the user provides nothing |
| `multiple` | No | For `string` type, allow an array of strings |
| `min` / `max` | No | Bounds for `number` type |

| Rejected field | How to pass the value |
| :--------------------------------------------------------------------------- | :-------------------------------------------------------------------------------------------------------------------------------- |
| Shell-form hook commands | Use [exec form](/docs/en/hooks#exec-form-and-shell-form) with `args`, or |
| [Monitor](#monitors) commands | Read the value from a config file in the script |
| MCP [`headersHelper`](/docs/en/mcp#use-dynamic-headers-for-custom-authentication) | Read the value from a config file in the script |

### Channels

### Path behavior rules

### Environment variables

| Variable | Resolves to | Use it for |
| :---------------------- | :---------------------------------------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------- |
| `${CLAUDE_PLUGIN_ROOT}` | Absolute path to the plugin's installation directory | Scripts, binaries, and config files bundled with the plugin |
| `${CLAUDE_PLUGIN_DATA}` | [Persistent directory](#persistent-data-directory) that survives plugin | Installed dependencies such as `node_modules` or Python virtual environm |
| `${CLAUDE_PROJECT_DIR}` | The project root | Project-local scripts and config files |

| Plugin component | Fields where placeholders resolve |
| :------------------------------ | :------------------------------------------ |
| Skill and agent content | Anywhere the placeholder appears |
| Hook and monitor commands | Anywhere the placeholder appears |
| MCP `stdio` servers | `command`, `args`, `env` |
| MCP `http`, `sse`, `ws` servers | `url`, `headers`, `headersHelper` |
| LSP servers | `command`, `args`, `env`, `workspaceFolder` |

#### Persistent data directory
