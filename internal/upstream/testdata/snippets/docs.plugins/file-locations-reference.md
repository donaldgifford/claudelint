### File locations reference

| Component | Default Location | Purpose |
| :---------------- | :--------------------------- | :----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **Manifest** | `.claude-plugin/plugin.json` | Plugin metadata and configuration (optional) |
| **Skills** | `skills/` | Skills with `<name>/SKILL.md` structure |
| **Commands** | `commands/` | Skills as flat Markdown files. Use `skills/` for new plugins |
| **Agents** | `agents/` | Subagent Markdown files |
| **Workflows** | `workflows/` | [Workflow](/docs/en/workflows) script files |
| **Output styles** | `output-styles/` | Output style definitions |
| **Themes** | `themes/` | Color theme definitions |
| **Hooks** | `hooks/hooks.json` | Hook configuration |
| **MCP servers** | `.mcp.json` | MCP server definitions |
| **LSP servers** | `.lsp.json` | Language server configurations |
| **Monitors** | `monitors/monitors.json` | Background monitor configurations |
| **Executables** | `bin/` | Executables added to the Bash tool's `PATH` and invokable as bare comman |
| **Settings** | `settings.json` | Default configuration applied when the plugin is enabled. Only the [ |
