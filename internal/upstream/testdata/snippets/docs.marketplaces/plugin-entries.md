## Plugin entries

### Required fields

| Field | Type | Description |
| :------- | :------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `name` | string | Plugin identifier in kebab-case, with no spaces, control characters, or |
| `source` | string|object | Where to fetch the plugin from (see [Plugin sources](#plugin-sources) be |

### Optional plugin fields

| Field | Type | Description |
| :--------------- | :------ | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `displayName` | string | Human-readable name shown in UI surfaces. When neither the entry nor the |
| `description` | string | Brief plugin description |
| `version` | string | Plugin version. If set (here or in `plugin.json`), the plugin is pinned |
| `author` | object | Plugin author information (`name` required; `email` and `url` optional) |
| `homepage` | string | Plugin homepage or documentation URL |
| `repository` | string | Source code repository URL |
| `license` | string | SPDX license identifier (for example, MIT, Apache-2.0) |
| `keywords` | array | Tags for plugin discovery and categorization |
| `metadata` | object | Free-form object for your own fields, such as entitlement or catalog dat |
| `category` | string | Plugin category for organization |
| `tags` | array | Tags for searchability |
| `strict` | boolean | Controls whether `plugin.json` is the authority for component definition |
| `relevance` | object | Signals that tell Claude Code when to suggest this plugin to users. Take |
| `defaultEnabled` | boolean | Whether the plugin is enabled after install (default: true). Set to |

| Field | Type | Description |
| :----------- | :------------- | :------------------------------------------------------------- |
| `skills` | string|array | Custom paths to skill directories containing `<name>/SKILL.md` |
| `commands` | string|array | Custom paths to flat `.md` skill files or directories |
| `agents` | string|array | Custom paths to agent files |
| `hooks` | string|object | Custom hooks configuration or path to hooks file |
| `mcpServers` | string|object | MCP server configurations or path to MCP config |
| `lspServers` | string|object | LSP server configurations or path to LSP config |

| Field | Type | Description |
| :-------------- | :----- | :-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `headers` | object | HTTP headers Claude Code sends when it downloads this entry's archive. O |
| `headersHelper` | string | Command that prints the HTTP headers for this entry's archive download a |
