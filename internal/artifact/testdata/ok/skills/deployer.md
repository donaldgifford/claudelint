---
name: deployer
description: Deploy the service to staging or production.
when_to_use: When the user asks to ship, deploy, or release.
context: fork
agent: shipper
user-invocable: false
allowed-tools: Bash(just deploy:*) Read, mcp__github
disallowed-tools: Write, Edit
---

# Deployer

Run the deploy pipeline and report the release URL.
