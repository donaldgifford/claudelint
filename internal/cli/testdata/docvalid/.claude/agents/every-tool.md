---
name: every-tool
description: Use when a change to the documented tool list needs a fixture that exercises every accepted name at once.
tools:
  - Agent
  - Artifact
  - AskUserQuestion
  - Bash
  - CronCreate
  - CronDelete
  - CronList
  - Edit
  - EndConversation
  - EnterPlanMode
  - EnterWorktree
  - ExitPlanMode
  - ExitWorktree
  - Glob
  - Grep
  - LSP
  - ListAgents
  - ListMcpResourcesTool
  - Monitor
  - NotebookEdit
  - PowerShell
  - PushNotification
  - Read
  - ReadMcpResourceTool
  - RemoteTrigger
  - ReportFindings
  - ScheduleWakeup
  - SendFeedback
  - SendMessage
  - SendUserFile
  - ShareOnboardingGuide
  - Skill
  - TaskCreate
  - TaskGet
  - TaskList
  - TaskOutput
  - TaskStop
  - TaskUpdate
  - TodoWrite
  - ToolSearch
  - WaitForMcpServers
  - WebFetch
  - WebSearch
  - Workflow
  - Write
---

This agent exists as a fixture. It declares every tool the Claude Code
reference documents, so adding a name to the canonical list without
adding it here is caught as a false positive rather than discovered by
a user.
