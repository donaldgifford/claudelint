# Tools reference

| Tool | Description | Permission required |
| :--------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | :------------------ |
| `Agent` | Spawns a [subagent](/docs/en/sub-agents) with its own context window to | No |
| `Artifact` | Publishes an HTML or Markdown file as an [artifact](/docs/en/artifacts): | Yes |
| `AskUserQuestion` | Asks multiple-choice questions to gather requirements or clarify ambigui | No |
| `Bash` | Executes shell commands in your environment. See [Bash tool behavior](#b | Yes |
| `CronCreate` | Schedules a recurring or one-shot prompt within the current session. Tas | No |
| `CronDelete` | Cancels a scheduled task by ID | No |
| `CronList` | Lists all scheduled tasks in the session | No |
| `Edit` | Makes targeted edits to specific files. See [Edit tool behavior](#edit-t | Yes |
| `EndConversation` | Ends the session, in rare cases of sustained abusive input or when you a | No |
| `EnterPlanMode` | Switches to plan mode to design an approach before coding | No |
| `EnterWorktree` | Creates an isolated [git worktree](/docs/en/worktrees) and switches into | Yes |
| `ExitPlanMode` | Presents a plan for approval and exits plan mode | Yes |
| `ExitWorktree` | Exits a worktree session and returns to the original directory. Not avai | No |
| `Glob` | Finds files based on pattern matching. Absent by default on macOS, Linux | No |
| `Grep` | Searches for patterns in file contents. Absent by default on macOS, Linu | No |
| `ListAgents` | Lists the agents Claude can message with `SendMessage`: subagents in the | No |
| `ListMcpResourcesTool` | Lists resources exposed by connected [MCP servers](/docs/en/mcp) | No |
| `LSP` | Code intelligence via language servers: jump to definitions, find refere | No |
| `Monitor` | Runs a command in the background and feeds each output line back to Clau | Yes |
| `NotebookEdit` | Modifies Jupyter notebook cells. See [NotebookEdit tool behavior](#noteb | Yes |
| `PowerShell` | Executes PowerShell commands natively. See [PowerShell tool](#powershell | Yes |
| `PushNotification` | Sends a desktop notification, and a phone push when [Remote Control](/do | No |
| `Read` | Reads the contents of files. See [Read tool behavior](#read-tool-behavio | No |
| `ReadMcpResourceTool` | Reads a specific MCP resource by URI | No |
| `RemoteTrigger` | Creates, updates, runs, and lists [Routines](/docs/en/routines) on claud | No |
| `ReportFindings` | Reports code-review findings as a structured list, with a file, summary, | No |
| `ScheduleWakeup` | Reschedules the next iteration of a [self-paced `/loop`](/docs/en/schedu | No |
| `SendFeedback` | Drafts a feedback report about Claude Code, covering a product problem o | No |
| `SendMessage` | Sends a message to another agent: an [agent team](/docs/en/agent-teams) | No |
| `SendUserFile` | Sends files from the session to you with an optional caption, so a gener | No |
| `ShareOnboardingGuide` | Uploads `ONBOARDING.md` and returns a share link teammates can open in C | Yes |
| `Skill` | Executes a [skill](/docs/en/skills#control-who-invokes-a-skill) within t | Yes |
| `TaskCreate` | Creates a new task in the task list. Provided by default only on the mod | No |
| `TaskGet` | Retrieves full details for a specific task. Provided by default only on | No |
| `TaskList` | Lists all tasks with their current status. Provided by default only on t | No |
| `TaskOutput` | Retrieves output from a background task. Deprecated in favor of `Read` o | No |
| `TaskStop` | Stops a running background task by ID. It also accepts an [agent-team te | No |
| `TaskUpdate` | Updates task status, dependencies, details, or deletes tasks. Provided b | No |
| `TodoWrite` | Manages the session task checklist. Disabled by default in favor of | No |
| `ToolSearch` | Searches for and loads deferred tools when [tool search](/docs/en/mcp#sc | No |
| `WaitForMcpServers` | Waits for one or more [MCP servers](/docs/en/mcp) that are still connect | No |
| `WebFetch` | Fetches content from a specified URL. See [WebFetch tool behavior](#webf | Yes |
| `WebSearch` | Performs web searches. See [WebSearch tool behavior](#websearch-tool-beh | Yes |
| `Workflow` | Runs a [dynamic workflow](/docs/en/workflows): a script that orchestrate | Yes |
| `Write` | Creates or overwrites files. See [Write tool behavior](#write-tool-behav | Yes |

## Configure tools with permission rules and hooks

| Rule format | Applies to | Details |
| :----------------------------- | :------------------------ | :--------------------------------------------------------------- |
| `Bash(npm run *)` | Bash, Monitor | [Command pattern matching](/docs/en/permissions#bash) |
| `PowerShell(Get-ChildItem *)` | PowerShell | [Command pattern matching](/docs/en/permissions#powershell) |
| `Read(~/secrets/**)` | Read, Grep, Glob, LSP | [Path pattern matching](/docs/en/permissions#read-and-edit) |
| `Edit(/src/**)` | Edit, Write, NotebookEdit | [Path pattern matching](/docs/en/permissions#read-and-edit) |
| `Skill(deploy *)` | Skill | [Skill name matching](/docs/en/skills#restrict-claude’s-skill-access) |
| `Agent(Explore)` | Agent | [Subagent type matching](/docs/en/permissions#agent-subagents) |
| `WebFetch(domain:example.com)` | WebFetch | [Domain matching](/docs/en/permissions#webfetch) |
| `WebSearch` | WebSearch | No specifier; allow or deny the tool as a whole |

## Agent tool behavior

## AskUserQuestion tool behavior

### Question auto-continue timeout

## Bash tool behavior

### What persists between commands

### Timeout and output limits

#### Output limits

| Result | What Claude gets |
| :------ | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Valid | Inline up to roughly 30,000 characters by default; past that, the path o |
| Failure | Inline up to roughly 10,000 characters; past that, a head-and-tail excer |

### Background commands

### Memory limit on Linux and WSL

## Edit tool behavior

## EndConversation tool behavior

## Glob tool behavior

## Grep tool behavior

## LSP tool behavior

## Monitor tool

### WebSocket source

| Field | Required | Description |
| :---------- | :------- | :--------------------------------------------------------------------------------------------------------------------------------------------- |
| `url` | Yes | The endpoint to connect to. Must be a `ws://` or `wss://` URL with no em |
| `protocols` | No | WebSocket subprotocol names to offer during the handshake. Each entry mu |

## NotebookEdit tool behavior

## PowerShell tool

| Match `Bash | PowerShell |

### Enable the PowerShell tool

### Shell selection in settings, hooks, and skills

### Windows encoding and exit codes

### Preview limitations

## Read tool behavior

## SendFeedback tool behavior

### What you see when Claude drafts

### Review and edit a draft

### Send a draft

### Discard or keep a draft

### Turn Claude-drafted feedback off

### Sessions without Claude-drafted feedback

## Task tool availability

## WebFetch tool behavior

## WebSearch tool behavior

### Session search limit

## Write tool behavior

## Check which tools are available

## See also
