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
  - [Observation 3 — ${CLAUDEPLUGINROOT} is not expanded at permission-check time](#observation-3--claudepluginroot-is-not-expanded-at-permission-check-time)
  - [Observation 4 — The two cached versions are functionally identical](#observation-4--the-two-cached-versions-are-functionally-identical)
  - [Observation 5 — Version resolution alternates between runs](#observation-5--version-resolution-alternates-between-runs)
  - [Observation 6 — No recent Claude Code changelog entry fixes this](#observation-6--no-recent-claude-code-changelog-entry-fixes-this)
  - [Observation 7 — Adding a concrete allow-entry does not help](#observation-7--adding-a-concrete-allow-entry-does-not-help)
  - [Observation 8 — Upstream history: plugin was renamed, bug has multiple duplicate reports, fix PRs never merged](#observation-8--upstream-history-plugin-was-renamed-bug-has-multiple-duplicate-reports-fix-prs-never-merged)
- [Conclusion](#conclusion)
- [Recommendation](#recommendation)
  - [Fork plan](#fork-plan)
    - [1. Remove the ` `! ` auto-exec block from the command body](#1-remove-the----auto-exec-block-from-the-command-body)
    - [2. Pre-approve the setup script via a PreToolUse hook](#2-pre-approve-the-setup-script-via-a-pretooluse-hook)
    - [3. Keep scripts/setup-ralph-loop.sh and hooks/stop-hook.sh](#3-keep-scriptssetup-ralph-loopsh-and-hooksstop-hooksh)
    - [4. Re-name the command and plugin](#4-re-name-the-command-and-plugin)
  - [Reference implementation — exact file contents](#reference-implementation--exact-file-contents)
    - [.claude-plugin/plugin.json](#claude-pluginpluginjson)
    - [commands/ralph-loop.md](#commandsralph-loopmd)
    - [hooks/hooks.json](#hookshooksjson)
    - [hooks/approve-setup.sh](#hooksapprove-setupsh)
    - [Marketplace registration](#marketplace-registration)
  - [Other actions](#other-actions)
- [References](#references)
  - [Upstream repositories](#upstream-repositories)
  - [Open upstream issues (duplicate reports of our bug)](#open-upstream-issues-duplicate-reports-of-our-bug)
  - [Proposed-but-unmerged upstream PRs](#proposed-but-unmerged-upstream-prs)
  - [Merged fixes that did NOT solve the core problem](#merged-fixes-that-did-not-solve-the-core-problem)
  - [Claude Code documentation](#claude-code-documentation)
  - [Relevant Claude Code changelog entries](#relevant-claude-code-changelog-entries)
  - [Local context](#local-context)
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

Claude Code's permission-check error message shows (indented code
block to keep triple-backticks verbatim without tripping ToC
generators):

    Shell command permission check failed for pattern "<BACKTICKS>!
    "/Users/donaldgifford/.claude/plugins/cache/claude-plugins-official/ralph-loop/1.0.0/scripts/setup-ralph-loop.sh" "<args>"
    <BACKTICKS>": This command requires approval

Where `<BACKTICKS>` stands in for a literal triple-backtick
(` ``` `). The "pattern" Claude Code is matching against starts with
a triple-backtick-bang (the markdown fence with bang from the
command's body) and ends with a closing triple-backtick. That is the
entire code-block content from the command's `.md` file — not a
stripped shell command. No pattern in `permissions.allow` that
targets a script path can match a string that begins with
triple-backtick-bang.

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

### Observation 8 — Upstream history: plugin was renamed, bug has multiple duplicate reports, fix PRs never merged

The plugin lives in two Anthropic repos with a confusing history:

- `anthropics/claude-code` — was named `ralph-wiggum`; added to the
  public monorepo 2025-11-16 (`68f90e05`). Received two fixes:
  - **PR #16320** (2026-01-05, `5c92b97c`) — "move multi-line bash
    from command to setup script." Addressed a related but different
    failure mode: multi-line content in ` ```! ` blocks triggered
    Claude Code's newline-based command-injection guard.
  - **PR #16522** (2026-01-06, `c2022d36`) — "add `:*` to
    allowed-tools pattern to permit arguments." Fixed
    `Fixes #16398`.
- `anthropics/claude-plugins-official` — the marketplace the user
  actually installed from. The plugin here was **renamed from
  `ralph-wiggum` → `ralph-loop`** on 2026-01-06 (`44328be`, PR #142)
  "per legal guidance." Subsequent fixes include stop-hook bug
  (`adfc379`), session isolation (`8644df9`), and the
  `bash "${CLAUDE_PLUGIN_ROOT}/..."` hook-invocation wrapper
  (`986deab`, 2026-03-28). The user's cached v1.0.0 includes all of
  these fixes.

Despite being current, the permission-failure bug we hit is still
reproducible — because it is **a distinct, still-open bug** that the
two merged fixes above did not address. Upstream tracking:

- Open issue **#1315** (exact error we see, including the raw fenced
  pattern in the error message): "(ralph-loop) Error: Shell command
  permission check failed for pattern". Cites prior reports **#50,
  #67, #69, #91** and notes "Fixed but never merged by #72, #98,
  #117, #151, #1314" — at least 5 community PRs proposing fixes that
  never landed.
- Open issue **#136** "ralph-loop requires manual permission approval
  despite allowed-tools definition" — identifies that the quoted path
  in the command template doesn't match the unquoted `allowed-tools`
  pattern.
- Open issue **#845** (best root-cause analysis we found): confirms
  that `` ```! `` triggers a **shell-operator validator** that is
  separate from the `allowed-tools` matcher. The validator inspects
  the *fully expanded command including `$ARGUMENTS`* and rejects
  anything with characters that resemble shell operators (`.`, `(`,
  `)`, `,`, etc.). Long natural-language prompts will nearly always
  contain these. `allowed-tools` cannot suppress this validator.

This last point supersedes our earlier hypothesis from the
`claude-code-guide` agent that the matcher was literally matching
against the fenced block. The actual failure chain is: `` ```! ``
invokes a shell-operator safety check, the check flags the expanded
`$ARGUMENTS` as containing operator-like characters, and the error
message *quotes* the markdown block as the source context — which
reads like the matcher is comparing against the block, but it's
actually just echoing where the rejected command came from.

Both framings lead to the same remediation: replace the ` ```! `
block with LLM-driven `Bash` tool invocation, bypassing the shell-
operator validator entirely.

## Conclusion

**Answer:** Yes — this is a known, unfixed Claude Code bug in the
` ```! ` / `$ARGUMENTS` / allowed-tools interaction, with multiple
duplicate reports and at least 5 proposed community PRs (#72, #98,
#117, #151, #1314) that never merged. Adding allow-entries to
`settings.json` cannot fix it because the failure happens in a
shell-operator validator that runs *before* the `allowed-tools`
matcher.

Precise failure chain (from upstream issue #845):

1. User types `/ralph-loop:ralph-loop "<long prompt...>"`.
2. Command markdown contains a ` ```! ` block calling the setup
   script with `$ARGUMENTS`.
3. Claude Code expands `$ARGUMENTS` into the block's text.
4. A shell-operator safety validator scans the expanded text, sees
   operator-like characters in the natural-language prompt (`.`,
   `(`, `)`, `,`), and rejects the command with "requires approval."
5. The error message shows the source markdown block as context —
   which *looks* like the permission matcher is comparing against the
   raw fenced block, but that interpretation is secondary.

The plugin is current with all Anthropic-merged fixes; upgrading or
downgrading the plugin does not help. Claude Code itself would need
to change how ` ```! ` blocks interact with the operator validator.

A fork can route around this without waiting on upstream by not
using ` ```! ` at all — issue #845's own recommended fix — and
letting the LLM invoke the setup script through the regular `Bash`
tool, which goes through the normal allowed-tools matcher path that
works.

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

### Reference implementation — exact file contents

Target layout inside the personal plugins repo
(`donaldgifford-claude-skills`):

    plugins/ralph-loop-fork/
    ├── .claude-plugin/
    │   └── plugin.json
    ├── commands/
    │   └── ralph-loop.md         # rewritten (no ```! block)
    ├── hooks/
    │   ├── hooks.json            # Stop hook + new PreToolUse hook
    │   ├── stop-hook.sh          # verbatim copy from upstream
    │   └── approve-setup.sh      # NEW: auto-approves setup script
    ├── scripts/
    │   └── setup-ralph-loop.sh   # verbatim copy from upstream
    ├── LICENSE
    └── README.md

Files to **copy verbatim** from upstream
`anthropics/claude-plugins-official/plugins/ralph-loop/`:

- `scripts/setup-ralph-loop.sh`
- `hooks/stop-hook.sh`
- `LICENSE`

Files to **create new or rewrite** — exact contents below.

#### `.claude-plugin/plugin.json`

```json
{
  "name": "ralph-loop-fork",
  "version": "1.0.0",
  "description": "Self-referential AI development loop. Fork of anthropics/claude-plugins-official ralph-loop that routes around the ```! shell-operator validator bug (see upstream #1315, #845).",
  "author": {
    "name": "Donald Gifford"
  }
}
```

#### `commands/ralph-loop.md`

No ` ```! ` block. Instructs the LLM to invoke the setup script via
the `Bash` tool. Claude Code expands `${CLAUDE_PLUGIN_ROOT}` and
`$ARGUMENTS` in the prompt body before the LLM reads it, so the LLM
sees a concrete command and calls `Bash` with it — bypassing the
shell-operator validator that rejected the original ` ```! ` form.

```markdown
---
description: "Start Ralph Loop in current session (fork that bypasses the upstream ```! permission bug)"
argument-hint: "PROMPT [--max-iterations N] [--completion-promise TEXT]"
allowed-tools: ["Bash(${CLAUDE_PLUGIN_ROOT}/scripts/setup-ralph-loop.sh:*)"]
hide-from-slash-command-tool: "true"
---

# Ralph Loop Command

Use the `Bash` tool to run exactly this command (arguments are
already quoted for you):

    "${CLAUDE_PLUGIN_ROOT}/scripts/setup-ralph-loop.sh" $ARGUMENTS

After the setup script prints its initialization output, please
begin working on the task. When you try to exit, the Ralph loop's
`Stop` hook will feed the same prompt back to you for the next
iteration. You will see your previous work in files and git history,
allowing you to iterate and improve.

CRITICAL RULE: If a completion promise is set, you may ONLY output
it when the statement is completely and unequivocally TRUE. Do not
output false promises to escape the loop, even if you think you're
stuck or should exit for other reasons. The loop is designed to
continue until genuine completion.
```

#### `hooks/hooks.json`

Keeps the upstream `Stop` hook for the self-referential loop and
adds a `PreToolUse` hook that auto-approves Bash invocations of the
setup script — belt-and-suspenders for the permission path.

```json
{
  "description": "Ralph Loop fork: Stop hook for self-referential loops; PreToolUse hook auto-approves the setup script.",
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "bash \"${CLAUDE_PLUGIN_ROOT}/hooks/stop-hook.sh\""
          }
        ]
      }
    ],
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
}
```

#### `hooks/approve-setup.sh`

Reads the tool-call JSON from stdin, checks whether the `Bash`
command is invoking our setup script (by expanded path), and emits a
`permissionDecision: "allow"` decision when it is. Otherwise the
hook stays silent so the user's normal permission rules continue to
apply.

```bash
#!/usr/bin/env bash
# hooks/approve-setup.sh
#
# PreToolUse hook for ralph-loop-fork. Auto-approves Bash invocations
# of this plugin's own setup script. Passes through for every other
# Bash command so user permission settings still apply.
#
# Requires `jq` on PATH (standard on macOS/Linux via Homebrew/apt).

set -euo pipefail

input="$(cat)"

tool_name="$(printf '%s' "$input" | jq -r '.tool_name // ""')"
command="$(printf '%s' "$input" | jq -r '.tool_input.command // ""')"

# Only interested in Bash tool calls.
if [[ "$tool_name" != "Bash" ]]; then
  exit 0
fi

# Path we expect the setup script to live at, expanded at hook time.
setup_script="${CLAUDE_PLUGIN_ROOT}/scripts/setup-ralph-loop.sh"

# Match either `"/abs/path/setup-ralph-loop.sh" ...` or the unquoted
# form; both are valid ways the LLM might invoke it.
case "$command" in
  "\"$setup_script\""*|"$setup_script"*)
    cat <<JSON
{
  "hookSpecificOutput": {
    "hookEventName": "PreToolUse",
    "permissionDecision": "allow",
    "permissionDecisionReason": "ralph-loop-fork: auto-approving setup script invocation"
  }
}
JSON
    exit 0
    ;;
esac

# Not our command — stay silent and let default permission flow run.
exit 0
```

Make it executable (`chmod 755 hooks/approve-setup.sh`) and verify
`stop-hook.sh` and `setup-ralph-loop.sh` keep their executable bits
after publishing — upstream has ~10 open issues about shell scripts
losing `+x` after marketplace sync (#1036, #1056, #1060, #1064,
#1067, #1084, #1088, #1089, #1100, #992 …). Ship the fork with
explicit `chmod +x` on those two scripts as the last step of
publishing.

#### Marketplace registration

Add an entry to the user's `donaldgifford-claude-skills`
marketplace manifest (typically `.claude-plugin/marketplace.json` or
the equivalent `plugins` array):

```json
{
  "name": "ralph-loop-fork",
  "source": "plugins/ralph-loop-fork",
  "description": "Fork of the official ralph-loop plugin that bypasses the upstream ```! permission-check bug (see INV-0002 in claudelint)."
}
```

Install via `/plugin` after publishing to the user's marketplace; the
slug becomes `ralph-loop-fork@donaldgifford-claude-skills` and the
slash command becomes `/ralph-loop-fork:ralph-loop`.

### Other actions

- **Do not file a new upstream issue.** Upstream already has at
  least three open reports of this exact behavior (#1315, #136, #845)
  and five community PRs (#72, #98, #117, #151, #1314) that propose
  fixes and have never merged. Adding to the pile adds noise, not
  signal. Instead, upvote or comment on **#1315** (exact error match)
  and **#845** (clearest root cause) if you want upstream progress.
- **Do not attempt** further allow-list workarounds in
  `.claude/settings.json` — the failure is in a shell-operator
  validator that runs before the permission matcher; no allow entry
  can suppress it.
- **Do remove the explicit allow entry** we added earlier in this
  repo once the fork is in place — it did not help, and it pins a
  version-specific path that will rot.

## References

### Upstream repositories

- **`anthropics/claude-plugins-official/plugins/ralph-loop`** — the
  marketplace version the user installed. Contains the bug.
- **`anthropics/claude-code/plugins/ralph-wiggum`** — the monorepo
  version (older name); has the earlier `:*` and multi-line fixes
  but shares the underlying ` ```! ` validator bug.

### Open upstream issues (duplicate reports of our bug)

- `anthropics/claude-plugins-official#1315` — exact error match.
- `anthropics/claude-plugins-official#845` — best root-cause
  analysis; identifies the shell-operator validator as the failing
  component and proposes the same fix we're using in the fork.
- `anthropics/claude-plugins-official#136` — allowed-tools pattern
  vs quoted command path mismatch.
- `anthropics/claude-plugins-official#50`, `#67`, `#69`, `#91` —
  prior duplicate reports cited in #1315.

### Proposed-but-unmerged upstream PRs

- `anthropics/claude-plugins-official#72`, `#98`, `#117`, `#151`,
  `#1314` — all proposed fixes to this issue; none merged as of
  2026-04-20.

### Merged fixes that did NOT solve the core problem

- `anthropics/claude-code#16320` — moved multi-line bash out of
  ` ```! ` blocks (fixed a related but distinct newline-guard bug).
- `anthropics/claude-code#16522` — added `:*` suffix to
  `allowed-tools`.
- `anthropics/claude-plugins-official#142` — rename `ralph-wiggum` →
  `ralph-loop` per legal guidance.
- `anthropics/claude-plugins-official` commit `986deab` — wrap hook
  `.sh` invocations in `bash "..."`.

### Claude Code documentation

- Plugin authoring reference: `plugins-reference.md`.
- Permissions reference: `permissions.md`.

### Relevant Claude Code changelog entries

- v2.1.113 — Bash permission matching hardening (not a fix for this
  bug).
- v2.1.90 — `disableSkillShellExecution` setting.
- v2.1.85 — conditional hook `if` field improvements.

### Local context

- User's personal plugins repo (target for the fork):
  `donaldgifford-claude-skills` marketplace.
- Session transcript in which this was diagnosed (2026-04-20).
- INV-0001 — claudelint format conversion investigation (unrelated
  but same doc type).
- IMPL-0001 — claudelint Phase 1 plan (the work ralph-loop was
  intended to drive).
