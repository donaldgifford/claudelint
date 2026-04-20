---
id: INV-0002
title: "Ralph-loop permission matcher bug and fork plan"
status: Concluded
author: Donald Gifford
created: 2026-04-20
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0002: Ralph-loop permission matcher bug and fork plan

**Status:** Concluded
**Author:** Donald Gifford
**Date:** 2026-04-20

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Environment](#environment)
- [Findings](#findings)
  - [Observation 1 — The plugin's allowed-tools uses ${CLAUDEPLUGINROOT} plus colon-argument syntax](#observation-1--the-plugins-allowed-tools-uses-claudepluginroot-plus-colon-argument-syntax)
  - [Observation 2 — The error "pattern" is the raw fenced code block](#observation-2--the-error-pattern-is-the-raw-fenced-code-block)
<!--toc:end-->

## Question

Why does invoking the `/ralph-loop:ralph-loop` slash command from the
official `ralph-loop` Claude Code plugin always fail with
`"Shell command permission check failed"` — even after adding an
explicit allow entry to `.claude/settings.json` — and what is the
minimal change needed to fork the plugin into a working variant that
can live alongside other plugins in a personal plugins repo?

## Hypothesis

The plugin's `allowed-tools` frontmatter pattern uses
`${CLAUDE_PLUGIN_ROOT}` and the colon-argument syntax
(`Bash(...sh:*)`). The likely failure mode was one of:

1. `${CLAUDE_PLUGIN_ROOT}` is not expanded at permission-check time.
2. The plugin's body uses a triple-backtick-bang (` ```! `) auto-exec
   block whose raw markdown-fenced content is being fed to the
   permission matcher instead of the extracted shell command.
3. Multiple cached plugin versions (`1.0.0` and `61c0597779bd`) cause
   Claude Code to resolve to different absolute paths on successive
   invocations, so even a concrete path allow-entry goes stale.

## Context

While working on the claudelint project we wanted to drive the
implementation loop with the official ralph-loop plugin. Every
invocation of `/ralph-loop:ralph-loop` fails at the permission check,
blocking automation. We need to either fix the plugin, fork it, or
abandon it. A fork is preferable because the plugin concept is useful
and we'd like it to live in the user's personal plugins repository
next to the other skills/plugins already maintained there.

**Triggered by:** interactive session, 2026-04-20. Related to
IMPL-0001 Phase 1 bring-up (we wanted ralph-loop to drive phase
completion).

## Approach

1. Reproduce the failure with the real slash command.
2. Inspect the plugin command definition, frontmatter, and body.
3. Compare the two cached versions on disk to rule out a code
   difference.
4. Compare user-scope vs project-scope install records in
   `~/.claude/plugins/installed_plugins.json`.
5. Add an explicit allow-entry for the absolute script path to
   `.claude/settings.json` and re-test.
6. Delegate to the `claude-code-guide` agent to get authoritative
   answers on `allowed-tools` semantics, `${CLAUDE_PLUGIN_ROOT}`
   expansion, and recent Claude Code matcher changes.

## Environment

| Component | Value |
|-----------|-------|
| Claude Code CLI | version present as of 2026-04-20 |
| Plugin | `ralph-loop@claude-plugins-official` |
| Cached versions | `1.0.0` (user scope), `61c0597779bd` (project scope, different project) |
| Host project | `/Users/donaldgifford/code/claudelint` |
| Project settings | `.claude/settings.json` with explicit allow entry |

## Findings

### Observation 1 — The plugin's `allowed-tools` uses `${CLAUDE_PLUGIN_ROOT}` plus colon-argument syntax

`~/.claude/plugins/cache/claude-plugins-official/ralph-loop/1.0.0/commands/ralph-loop.md`
frontmatter:

```yaml
---
description: "Start Ralph Loop in current session"
argument-hint: "PROMPT [--max-iterations N] [--completion-promise TEXT]"
allowed-tools: ["Bash(${CLAUDE_PLUGIN_ROOT}/scripts/setup-ralph-loop.sh:*)"]
hide-from-slash-command-tool: "true"
---
```

Command body contains a triple-backtick-bang block that auto-executes
on invocation:

    ```!
    "${CLAUDE_PLUGIN_ROOT}/scripts/setup-ralph-loop.sh" $ARGUMENTS
    ```

### Observation 2 — The error "pattern" is the raw fenced code block

Claude Code's permission-check error message shows:

```
Shell command permission check failed for pattern "```!
"/Users/donaldgifford/.claude/plugins/cache/claude-plugins-official/ralph-loop/1.0.0/scripts/setup-ralph-loop.sh" "<args>"
```": This command requires approval
```

The "pattern" starts with ` ```! ` (the markdown fence with bang) and
ends with ` ``` `. That is the entire code-block content, not a
stripped shell command. No pattern in `permissions.allow` that
targets a script path can match a string that begins with
` ```! `.

### Observation 3 — `${CLAUDE_PLUGIN_ROOT}` is not expanded at permission-check time

Authoritative answer from the `claude-code-guide` agent (citing
`plugins-reference.md`): `${CLAUDE_PLUGIN_ROOT}` is expanded at
execution time, not at permission evaluation time. The permission
matcher has no knowledge of the variable and never sees the expanded
path. This means any `allowed-tools` pattern that references
`${CLAUDE_PLUGIN_ROOT}` can never match the concrete command Claude
Code actually runs. The pattern is effectively dead.

### Observation 4 — The two cached versions are functionally identical

`diff -r` between `ralph-loop/61c0597779bd/` and `ralph-loop/1.0.0/`
shows:

- `commands/ralph-loop.md` — identical
- `scripts/setup-ralph-loop.sh` — identical
- `hooks/hooks.json` — differs only in wrapping the `stop-hook.sh`
  command in `bash "..."`
- `.claude-plugin/plugin.json` — new version adds a `version` field

The permission failure is **not** a plugin-version regression. Both
versions have the same failing frontmatter pattern and same
` ```! ` body block.

### Observation 5 — Version resolution alternates between runs

`installed_plugins.json` records two installs:

- `scope: "project"` — `projectPath:
  /Users/donaldgifford/code/wiz-go-gen`, version `61c0597779bd`.
- `scope: "user"` — version `1.0.0`, installed 2026-03-21.

Observed behavior on two back-to-back invocations from
`/Users/donaldgifford/code/claudelint`:

- First invocation resolved to `.../1.0.0/scripts/setup-ralph-loop.sh`
- Second invocation resolved to
  `.../61c0597779bd/scripts/setup-ralph-loop.sh`

There is no documented mechanism to pin which cached version a given
working directory uses — it's a caching implementation detail. Even
if the matcher worked, a single absolute-path allow-entry would go
stale across runs.

### Observation 6 — No recent Claude Code changelog entry fixes this

Relevant recent permission-related changes:

- **v2.1.113** — Bash permission matching hardened (wrapping
  detection, `find -exec`).
- **v2.1.90** — added `disableSkillShellExecution` + PowerShell
  permission checks.
- **v2.1.85** — conditional hook `if` field improvements.

None address triple-backtick-bang block parsing for permission
checks, and none change `${CLAUDE_PLUGIN_ROOT}` expansion semantics
in `allowed-tools`.

### Observation 7 — Adding a concrete allow-entry does not help

We added
`"Bash(/Users/donaldgifford/.claude/plugins/cache/claude-plugins-official/ralph-loop/1.0.0/scripts/setup-ralph-loop.sh:*)"`
to `.claude/settings.json`. Next invocation still failed — both
because (a) the pattern being matched is the fenced block, not the
script path, and (b) the invocation resolved to the
`61c0597779bd` version directory, a different absolute path.

## Conclusion

**Answer:** Yes — this is a Claude Code harness bug (or pair of
bugs), not a plugin bug that settings-level workarounds can solve:

1. The permission matcher evaluates raw `` ```!... ``` `` code-block
   content as the "command pattern." Any `allowed-tools` or
   `permissions.allow` entry written as a bare command (which is all
   the documentation describes) cannot match a fenced block.
2. `${CLAUDE_PLUGIN_ROOT}` in `allowed-tools` is never expanded, so
   plugin-declared patterns that use it are dead.

The plugin's own configuration is consistent with what the
documentation suggests should work. The permission-check behavior is
inconsistent with it.

A fork can work around the bug without waiting on an upstream fix by
eliminating both triggers: stop using `` ```! `` auto-execute in the
command body, and stop relying on `${CLAUDE_PLUGIN_ROOT}` in
`allowed-tools`.

## Recommendation

### Fork plan

Create `ralph-loop@donaldgifford-claude-skills` as a replacement that
lives in the user's personal plugins repo alongside the existing
skills. Minimal changes vs the upstream plugin:

#### 1. Remove the `` ```! `` auto-exec block from the command body

Replace the code-block with **instructions** for the LLM to run the
script via the normal `Bash` tool. The command markdown becomes:

```markdown
---
description: "Start Ralph Loop in current session"
argument-hint: "PROMPT [--max-iterations N] [--completion-promise TEXT]"
allowed-tools: ["Bash"]
hide-from-slash-command-tool: "true"
---

Run the setup script located at
`${CLAUDE_PLUGIN_ROOT}/scripts/setup-ralph-loop.sh` and pass it the
arguments: $ARGUMENTS

Then follow the Ralph-loop operating instructions ...
```

When Claude Code expands `$ARGUMENTS` and `${CLAUDE_PLUGIN_ROOT}` in
the prompt body, the LLM sees the concrete path and invokes the
`Bash` tool with it. That path goes through the regular permission
matcher (not the `` ```! `` path), which does work for
`Bash(path:*)` patterns.

#### 2. Pre-approve the setup script via a `PreToolUse` hook

Add a hook to `hooks/hooks.json` that auto-approves `Bash` invocations
whose command matches the setup-script path (expanded at hook time
via `${CLAUDE_PLUGIN_ROOT}`, which *is* available in hook
environments):

```json
{
  "PreToolUse": [
    {
      "matcher": "Bash",
      "hooks": [
        {
          "type": "command",
          "command": "bash \"${CLAUDE_PLUGIN_ROOT}/hooks/approve-setup.sh\""
        }
      ]
    }
  ]
}
```

`hooks/approve-setup.sh` reads the tool-input JSON from stdin,
extracts the command, and exits with an allow-decision when the
command invokes the setup script. This bypasses the broken static
allow-list pattern.

#### 3. Keep `scripts/setup-ralph-loop.sh` and `hooks/stop-hook.sh`
verbatim from upstream

No change needed to the script or stop hook — the bug is entirely in
how the command is invoked and authorized, not in what the script
does.

#### 4. Re-name the command and plugin

Use a new name (`ralph-loop` → `ralph` or `loop`, plugin slug
`ralph-loop-fork`) so Claude Code doesn't collide with the official
plugin during cached-version resolution. Advertise via the user's
marketplace config.

### Other actions

- **File an upstream issue** against
  `claude-plugins-official/ralph-loop` describing Observations 1–7
  and linking this doc. The plugin's `allowed-tools` pattern is dead
  as written and the ` ```! ` body will fail on any user who has not
  already approved it.
- **File an upstream issue (or docs PR) against Claude Code** to
  clarify that `${CLAUDE_PLUGIN_ROOT}` is not expanded in
  `allowed-tools`, and that ` ```! ` blocks are matched in a
  different codepath than the `Bash` tool. Pick one behavior and
  document it.
- **Do not attempt** any more allow-list workarounds in
  `.claude/settings.json` — they cannot match the fenced block.

## References

- Official plugin (broken):
  `https://github.com/anthropics/claude-plugins/tree/main/ralph-loop`
  (or equivalent — see `~/.claude/plugins/marketplaces/`).
- User's personal plugins repo (target for the fork):
  `donaldgifford-claude-skills` marketplace.
- Claude Code plugin authoring reference: `plugins-reference.md`.
- Claude Code permissions reference: `permissions.md`.
- Changelog entries referenced above: v2.1.113, v2.1.90, v2.1.85.
- Session transcript in which this was diagnosed (2026-04-20).
- INV-0001 — claudelint format conversion investigation (unrelated
  but same doc type).
- IMPL-0001 — claudelint Phase 1 plan (the work ralph-loop was
  intended to drive).
