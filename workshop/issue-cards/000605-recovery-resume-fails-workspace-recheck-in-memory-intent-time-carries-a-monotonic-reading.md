---
id: 000605
status: codecomplete
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
github_issue:
severity: critical
started: 2026-10-15T23:33:30-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 0.04
tracker:
    version: 1
    completion:
        token: close-f7acb2315060
        repository: github.com/xianxu/pair
        reviewed_head: 1447f7681ac0bf15d06624febae95dd346e17429
        evidence_commit: 20bf5b5251755796e3845e99a15b044524e1d812
---

# Recovery resume fails workspace recheck: in-memory intent time carries a monotonic reading

## Problem

`couch --resume ariadne:5` (and the UI's Resume) fails every time with `slot start claim changed during workspace readiness` (couchcore/slotlaunch.go:69) on the first recheck, before readiness.Ensure, with nothing else touching the slot (TL op 647b6780). Diagnosis (probe on a temp copy of ariadne:5's real record): the only difference is `Start.RecoveryIntent.CreatedAt`'s monotonic clock reading. Recovery resume builds the intent with `c.Clock.Now()` (SystemClock: `time.Now()`, monotonic included); `CommitStartClaim` returns the in-memory record; the recheck `reflect.DeepEqual`s it with a fresh `GetThread` whose decoded time has no monotonic part. Same wall time, `Equal` true, `DeepEqual` false. Tests use FixedClock (no monotonic), so none caught it; the fresh-reboot path passes only because it re-reads the record.
