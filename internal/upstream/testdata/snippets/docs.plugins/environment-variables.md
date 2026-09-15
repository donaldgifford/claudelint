### Environment variables

| Variable                | Resolves to                                                                                                 | Use it for                                                                                               |
| :---------------------- | :---------------------------------------------------------------------------------------------------------- | :------------------------------------------------------------------------------------------------------- |
| `${CLAUDE_PLUGIN_ROOT}` | Absolute path to the plugin's installation directory                                                        | Scripts, binaries, and config files bundled with the plugin                                              |
| `${CLAUDE_PLUGIN_DATA}` | [Persistent directory](#persistent-data-directory) that survives plugin updates, created on first reference | Installed dependencies such as `node_modules` or Python virtual environments, generated code, and caches |
| `${CLAUDE_PROJECT_DIR}` | The project root                                                                                            | Project-local scripts and config files                                                                   |

| Plugin component                | Fields where placeholders resolve           |
| :------------------------------ | :------------------------------------------ |
| Skill and agent content         | Anywhere the placeholder appears            |
| Hook and monitor commands       | Anywhere the placeholder appears            |
| MCP `stdio` servers             | `command`, `args`, `env`                    |
| MCP `http`, `sse`, `ws` servers | `url`, `headers`, `headersHelper`           |
| LSP servers                     | `command`, `args`, `env`, `workspaceFolder` |

#### Persistent data directory
