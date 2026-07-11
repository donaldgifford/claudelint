---
name: full-agent
description: Exercises every documented subagent frontmatter field.
tools: Read, Grep, Bash(git diff:*), mcp__github
disallowedTools: Write, Edit
model: claude-sonnet-5
permissionMode: acceptEdits
maxTurns: 12
skills:
  - deployer
  - writer
mcpServers:
  github:
    command: gh-mcp
hooks:
  PreToolUse:
    - matcher: Bash
      hooks:
        - type: command
          command: ./check.sh
memory: project
background: true
effort: high
isolation: worktree
color: cyan
initialPrompt: Summarize the repo state before starting.
---

Doc-valid subagent covering the full 16-field merged frontmatter model,
pinned by TestFixturesOK.
