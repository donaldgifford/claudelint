---
id: INV-0004
title: "Donald-loop stop hook silently fails promise detection on transcripts with unescaped newlines"
status: Closed
author: Donald Gifford
created: 2026-04-21
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0004: Donald-loop stop hook silently fails promise detection on transcripts with unescaped newlines

**Status:** Closed
**Author:** Donald Gifford
**Date:** 2026-04-21

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Environment](#environment)
- [Findings](#findings)
  - [Symptom: state file never deleted](#symptom-state-file-never-deleted)
  - [Root cause: jq -rs aborts on a single bad line](#root-cause-jq--rs-aborts-on-a-single-bad-line)
  - [Why bad lines exist in the transcript](#why-bad-lines-exist-in-the-transcript)
- [Conclusion](#conclusion)
- [Recommendation](#recommendation)
- [References](#references)
<!--toc:end-->

## Question

Why does the `donald-loop` stop hook keep blocking session exit even
after the assistant emits the exact completion promise
(`<promise>MVP COMPLETE</promise>`), and what is the minimum fix?

## Hypothesis

The hook's promise-extraction pipeline is brittle against one of:

1. a stray character around the `<promise>` tag,
2. multiple text blocks in a turn where the last one is empty, or
3. malformed JSONL in the transcript.

(3) turned out to be correct.

## Context

During the Phase 1 MVP build of `claudelint`, a `donald-loop` session
was left running with `--completion-promise "MVP COMPLETE"`. The
assistant emitted `<promise>MVP COMPLETE</promise>` at the end of a
turn, but the loop continued to block every subsequent session exit
with the same prompt. The hook neither deleted the state file (which
would have indicated a clean detection) nor surfaced an error to the
user — the failure is silent.

**Triggered by:** `donald-loop` stop hook blocking error surfaced
during the Phase 1.8 wrap-up of IMPL-0001.

## Approach

1. Inspect the state file at
   `.claude/donald-loop.local.md` — confirm `iteration` is
   incrementing (hook is running and falling through) and
   `completion_promise` matches the emitted tag verbatim.
2. Read the hook at
   `~/.claude/plugins/cache/donaldgifford-claude-skills/donald-loop/1.0.0/hooks/stop-hook.sh`
   to understand the promise-extraction pipeline.
3. Replay the pipeline against the exact transcript snapshot at the
   moment the promise was emitted (truncate the JSONL at that line,
   run the same `grep | jq -rs | perl` the hook runs).
4. Isolate the first command in the pipeline that behaves differently
   than expected.

## Environment

| Component         | Version / Value |
|-------------------|-----------------|
| `donald-loop`     | `donaldgifford-claude-skills@1.0.0` (fork of upstream `ralph-loop` — see INV-0002) |
| `jq`              | bundled with macOS (`jq-1.7.x`) |
| Claude Code       | 2.1.112 |
| Shell             | bash (`/bin/bash`) |
| State file        | `.claude/donald-loop.local.md` with `completion_promise: "MVP COMPLETE"` |

## Findings

### Symptom: state file never deleted

`iteration: 4` at the time of diagnosis; state file still on disk.
The hook's happy-path promise branch (`rm "$STATE_FILE"; exit 0`)
never ran even though the transcript unambiguously contained
`<promise>MVP COMPLETE</promise>` as the last assistant text block:

    $ grep '"role":"assistant"' "$TRANSCRIPT" | tail -n 100 \
        | jq -rs 'map(.message.content[]? | select(.type == "text") | .text) | last // ""' \
        | tail -c 200
    (no output — jq aborted)

### Root cause: jq -rs aborts on a single bad line

The hook at `stop-hook.sh:112-117` runs:

    set +e
    LAST_OUTPUT=$(echo "$LAST_LINES" | jq -rs '
      map(.message.content[]? | select(.type == "text") | .text) | last // ""
    ' 2>&1)
    JQ_EXIT=$?
    set -e

Two properties of this pipeline combine to silently swallow the
promise:

1. **`-rs` slurps every input line into one JSON array before
   evaluating the filter.** A single malformed line makes the entire
   batch fail; there is no per-line error tolerance.
2. **`2>&1` redirects jq's stderr into `$LAST_OUTPUT`.** On failure
   the variable holds `"jq: parse error: ..."` — not empty, not a
   valid text block. The subsequent check
   `if [[ $JQ_EXIT -ne 0 ]]` would bail out, but the hook then uses
   the variable *anyway* in the promise branch at line 134:

       PROMISE_TEXT=$(echo "$LAST_OUTPUT" | perl -0777 -pe 's/.*?<promise>(.*?)<\/promise>.*/$1/s; ...')

   The perl regex finds no `<promise>` in the jq error text, returns
   empty, the equality check fails, and the hook falls through to the
   "continue loop" branch — indistinguishable from a legitimate
   no-promise turn.

Reproduction in our transcript: of the last 100 assistant lines at
the moment the promise was emitted, **27 failed `jq -e .`**. The
first failure reported:

    jq: parse error: Invalid string: control characters from U+0000
    through U+001F must be escaped at line 11, column 440

The offending bytes were literal `\n` characters (0x0A) embedded
inside a JSON string — which is invalid JSON (must be escaped as
`\\n`).

### Why bad lines exist in the transcript

Claude Code writes assistant content blocks as JSONL lines in
`~/.claude/projects/<slug>/<session>.jsonl`. When the `text` field of
a message contains long multi-line prose (especially code fences,
long diff output, or nested snippets), occasional lines end up with
raw newlines inside the string rather than the required `\n` escape.
This is ultimately a Claude Code transcript-writer robustness issue,
not something donald-loop can fix upstream — but the hook can still
tolerate it.

## Conclusion

**Answer:** The donald-loop stop hook's promise-extraction pipeline
is brittle against malformed JSONL lines in the Claude Code
transcript. A single bad line poisons the `jq -rs` batch and causes
silent loop continuation. The hook needs a per-line parser with
error tolerance.

## Recommendation

Patch `donald-loop/1.0.0/hooks/stop-hook.sh` (and the upstream source
of the fork) to parse transcript lines individually, skipping any
line that fails to decode:

    LAST_OUTPUT=$(echo "$LAST_LINES" | perl -MJSON::PP -e '
      my $last = "";
      while (<STDIN>) {
        my $obj = eval { JSON::PP->new->decode($_) };
        next unless $obj && ref($obj->{message}{content}) eq "ARRAY";
        for my $blk (@{$obj->{message}{content}}) {
          $last = $blk->{text} if ($blk->{type}//"") eq "text";
        }
      }
      print $last;
    ')
    JQ_EXIT=$?

`JSON::PP` is part of core Perl on macOS and Linux, so no new
dependency is introduced. The `eval` block skips malformed lines
instead of aborting the whole pipeline.

**Immediate workaround** for a stuck loop (no patch required):

    rm .claude/donald-loop.local.md

The next Stop hook sees no state file and allows the session to
exit normally.

**Follow-up:**

- Open an issue / PR against the upstream `donald-loop` fork carrying
  this fix and this investigation as the reproduction record.
- Consider whether the hook should warn-log when `jq` / `perl` parse
  fails instead of silently continuing, so future silent-failure
  modes surface early. Today the only visible signal is the missing
  `rm` of the state file, which requires careful reading to spot.

## References

- Related: [INV-0002](0002-ralph-loop-permission-matcher-bug-and-fork-plan.md)
  (prior ralph-loop → donald-loop fork motivation).
- `stop-hook.sh` source:
  `~/.claude/plugins/cache/donaldgifford-claude-skills/donald-loop/1.0.0/hooks/stop-hook.sh`.
- [IMPL-0001](../impl/0001-phase-1-core-linter-for-claudemd-skills-plugins-and-hooks.md) —
  the Phase 1.8 completion-promise emission that triggered the
  investigation.
