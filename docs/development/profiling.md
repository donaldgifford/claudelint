---
title: Profiling
---

## Profiling

`--profile=<dir>` captures CPU, heap, block, and mutex profiles for a single
run. Use it to investigate scheduler behavior before proposing runtime changes.

```bash
claudelint run --profile=./profile .
go tool pprof ./profile/cpu.pprof
```

`just profile` is a convenience target that runs the command against this repo
and prints the pprof command you need.
