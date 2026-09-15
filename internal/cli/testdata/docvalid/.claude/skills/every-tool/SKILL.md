---
name: every-tool
description: Use when verifying that a skill may declare every documented tool without tripping commands/allowed-tools-known.
allowed-tools:
  - EnterPlanMode
  - SendMessage
  - TaskCreate
  - Read
  - Write
disallowed-tools:
  - Bash
---

## Purpose

Fixture skill covering the tool names added to the canonical list in
ruleset v1.6.0. It pairs with the agent and command fixtures beside it:
between them every documented tool appears in `tools`, `allowed-tools`,
and `disallowed-tools`.
