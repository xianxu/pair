---
id: 000487
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '6471c45fb9c4b3086c599ce202303631f99ad17c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T13:10:07-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "efc89ad3", done: "9e38085a"}
---

# tests: broadcast TestReapOrphans flakes under load (pid still alive after 2s)

## Problem

pair:4 saw `broadcast` `TestReapOrphans` fail once in a full package run at load average ~125 (the pid still alive after 2 s), passing 3 of 3 alone. Same family as #473/#476/#480: a real-time window asserted under load.

## Spec

Same rules as #480: wait on the process's exit (or an injected clock) bounded only by the test deadline, not a 2 s window; fix the product if reaping itself has the race.

## Done when

- 200 runs under -race and load, green; the cause in the Log.

## Plan

- [x] Reproduce deterministically; find whether reaping or the fixture races.
- [x] Fixture: startOrphan's fake stays a shell named cloudflared (no `exec`), settled when its sleep child exists.
- [x] waitDead bounded by `t.Deadline()`, not a 2 s window (all callers).
- [x] 200 runs under -race, plus the delayed-reap reproduction, green.

## Log

### 2026-10-12
- **Cause: a fixture race, not a reaping race.** startOrphan's fake ran `exec sleep 300`, and exec rewrites the process's command to `sleep 300`. startOrphan returned as soon as ps showed `/bin/sh …/cloudflared` (before the exec), and `killIfOurs` re-reads the command and correctly spares a non-cloudflared process. When idle, ReapOrphans' ps won the race against the exec (traced: it saw `/bin/sh …/cloudflared`). Under load the exec won, the orphan was spared, and waitDead's 2 s expired. Measured: a 500 ms pause before `ReapOrphans` fails every run with "pid … still alive after 2s".
- The product is right: a real cloudflared is a binary whose command stays cloudflared, and sparing a process whose command changed is the identity check working. No product change.
- The "live owner", "recycled pid" and "no owner identity" subtests used the same orphan, so under load they could pass for the wrong reason (spared as "not cloudflared"). The fixed fixture makes their reason the intended one.
- Fix: the fake runs `sleep 300` without exec, so the shell stays `/bin/sh …/cloudflared` and a group kill takes both. startOrphan waits for the sleep child (the settled state) and asserts the command. Mutation: restoring `exec` fails with "orphan never started its sleep child" at the test deadline instead of hanging. waitDead now waits until `t.Deadline()` minus 1 s (#480 rule) for all its callers in the package.
- Verification: the 500 ms delayed-reap variant passes 3/3 (it failed 1/1 before). `go test -race -count=200 -run TestReapOrphans$` passed 200/200 (274 s; load average 8–14, ambient). Per the TL's machine rules I generated no extra load, so the delay variant stands in for the load in Done-when.
