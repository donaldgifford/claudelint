# Changelog

## 2.1.270

- Fixed read-only git commands in Bash unexpectedly asking for permission after a session had been running for a while (regression in 2.1.269)

## 2.1.269

- Added `claude plugin eval`: run a plugin's eval suite against Claude Code and get scored, reproducible results (JSON + HTML report); see `claude plugin eval --help`
- Added `/output-style [name]` to list and switch output styles, including over Remote Control and in cloud and other headless sessions
- Added a diff of the files a Bash command changed to the Bash tool result when the Bash tool handles file edits (setting `bashEditDiffEnabled`)
- Added `OTEL_METRICS_INCLUDE_REPOSITORY` to tag OpenTelemetry metrics and events with `vcs.*` repository attributes; commit events get `vcs.ref.head.*` with `OTEL_LOG_TOOL_DETAILS`
- Added `CLAUDE_CODE_GATEWAY_MODEL_DISCOVERY_TIMEOUT_MS` to extend the LLM gateway `/v1/models` discovery timeout (default 3s)
- Added a spinner tip suggesting `/focus` for a view with just your prompt, a one-line work summary, and the response
- Added `CLAUDE_CODE_WORKFLOW_MAX_CONCURRENT_AGENTS` (1–256) to raise the Workflow tool's per-run concurrent agent limit for inference-bound fan-outs
- Fixed the prompt cache being partially invalidated on the turn after a response was cut off at the output-token limit and automatically resumed
- Fixed a case where resuming a session after interrupting Claude mid-thought could change how earlier context was re-sent, hurting prompt-cache reuse
- Fixed F1/F2/F4 not working in kitty-protocol terminals and Delete in st, Alt+arrows acting as Escape in rxvt-unicode, and Shift+punctuation typing the unshifted key in WezTerm (regression in 2.1.247)
- Fixed remote and headless sessions reporting "waiting for your input" while background agents were still running (set `CLAUDE_CODE_BG_TASKS_REPORT_RUNNING=0` to restore the old behavior)
- Fixed the terminal's replies to capability queries (`^[[?1;2c`) appearing as stray text at startup in some terminals
- Fixed rows at the top or bottom of the transcript going blank in fullscreen after resizing the terminal
- Fixed a deny or ask permission rule starting with `!` applying beyond the settings source that wrote it; such a rule now applies only within its own source, and a bare `!` negation is ignored
- Fixed the git status Claude is told after a compaction: it is now the current status, not the one from the start of the session
- Fixed synced plugin MCP servers not connecting when a remote session resumes
- Fixed resumed headless sessions losing a turn's replies when the model was switched or a request was retried mid-turn
- Fixed terminal escape codes, line breaks and oversized text from a background task's on-disk record reaching the task list and task notifications when work is resumed
- Fixed CMYK JPEG images failing to attach with "cannot decode"; they are now converted and resized like other JPEGs
- Fixed the managed settings approval dialog not naming the collector for a gRPC telemetry endpoint set without a scheme
- Fixed plugin `headersHelper` consent prompts showing a URL path that could be misread as a different host
- Fixed plugin errors showing `[redacted URL]` in place of a relative Windows path with a folder name that starts with `@`
- Fixed missing cursor in the permission-rule, auto-mode-rule, add-directory, session-rename and feedback-review text fields when the terminal's native cursor is enabled
- Fixed repeated clicks on a `/fork` receipt, each under a second apart, never backgrounding the session right away while it waited for the current tool to finish
- Fixed plugin LSP servers that reject `shutdown` params (e.g. rust-analyzer) being left running at session end; `exit` is now sent even if `shutdown` fails
- Fixed the attribution reminder overriding a CLAUDE.md or memory rule against commit and pull request attribution; lines set by managed settings still apply
- Fixed prompt suggestions being dropped for text in Japanese, Chinese, Thai and other languages written without spaces between words
- Fixed synchronized output being assumed from the terminal's name in GNOME Terminal and Konsole versions that do not support it
- Fixed `permission_denials` in `--output-format stream-json` results omitting Read, Edit and Write calls blocked by a path-scoped deny rule
- Fixed sessions run through the SDK or the desktop app showing an unknown status in other sessions' agent list
- Fixed `/insights` failing on Bedrock, Vertex, Foundry, and gateway deployments whose account can't reach the default Opus model by using the session model there instead
- Fixed organization policy limits not loading for the session when another Claude Code process refreshed the login at the same moment
