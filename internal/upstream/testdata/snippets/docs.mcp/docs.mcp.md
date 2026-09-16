# Connect Claude Code to tools via MCP

A JSON entry that has a `url` but no `type` is a configuration error, because Claude Code reads an entry with no `type` as a stdio server. Claude Code skips that server and reports `MCP server "<name>" has a "url" but no "type"; add "type": "http" (or "sse" / "ws") to this entry`. Before v2.1.202, Claude Code reported this misconfiguration as `command: expected string, received undefined`.
'{"type":"ws","url":"wss://mcp.example.com/socket","headers":{"Authorization":"Bearer YOUR_TOKEN"}}'
* **A `url` with no `type`**: add `"type": "http"`, `"type": "sse"`, or `"type": "ws"` to match the endpoint. Claude Code reads an entry with no `type` as a stdio server, so a `url` entry without a `type` fails.
"type": "http",
"type": "http",
"type": "http",
'{"type":"http","url":"https://mcp.example.com/mcp","oauth":{"clientId":"your-client-id","callbackPort":8080}}' \
'{"type":"http","url":"https://mcp.example.com/mcp","oauth":{"callbackPort":8080}}'
"type": "http",
"type": "http",
"type": "http",
"type": "http",
claude mcp add-json weather-api '{"type":"http","url":"https://api.weather.com/mcp","headers":{"Authorization":"Bearer token"}}'
claude mcp add-json local-weather '{"type":"stdio","command":"/path/to/weather-cli","args":["--api-key","abc123"],"env":{"CACHE_DIR":"/tmp"}}'
claude mcp add-json my-server '{"type":"http","url":"https://mcp.example.com/mcp","oauth":{"clientId":"your-client-id","callbackPort":8080}}' --client-secret
"type": "stdio",
"type": "stdio",
"type": "http",
