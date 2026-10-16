---
id: 000605
status: working
deps: []
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
severity: critical
card_mirror: 'fba2855a78cbdccd83e44e095d48cff05bb41f98' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T23:33:30-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Recovery resume fails workspace recheck: in-memory intent time carries a monotonic reading

## Problem

`couch --resume ariadne:5` (and the UI's Resume) fails every time with `slot start claim changed during workspace readiness` (couchcore/slotlaunch.go:69) on the first recheck, before readiness.Ensure, with nothing else touching the slot (TL op 647b6780). Diagnosis (probe on a temp copy of ariadne:5's real record): the only difference is `Start.RecoveryIntent.CreatedAt`'s monotonic clock reading. Recovery resume builds the intent with `c.Clock.Now()` (SystemClock: `time.Now()`, monotonic included); `CommitStartClaim` returns the in-memory record; the recheck `reflect.DeepEqual`s it with a fresh `GetThread` whose decoded time has no monotonic part. Same wall time, `Equal` true, `DeepEqual` false. Tests use FixedClock (no monotonic), so none caught it; the fresh-reboot path passes only because it re-reads the record.

## Spec

A recovery intent equals its persisted form from the moment it is built, so every in-memory vs persisted comparison of it (start fence, workspace recheck) holds. No check is added or changed.

## Done when

- A test with a real monotonic clock runs a recovery resume through the store and passes the workspace recheck; it is RED on main with the exact error.
- Targeted tests, recorded suite, close review pass; land and report.

## Plan

- Canonicalize the recovery intent through its persisted encoding where `executePreparedRecovery` builds it, before it is first persisted.
- Regression test with a monotonic clock.

## Log

### 2026-10-15

Filed from the read-only diagnosis of ariadne:5's failed resume after the f136934d restart (TL priority 1: slot startup).

### 2026-10-15 — fix (pair:2; TL approval 61111332)

- Class sweep (every reflect.DeepEqual in couchcore, launcher, threadrecord): the in-memory vs persisted comparisons of thread-record values are all of the recovery intent: start fence (recovery_start_fence.go), workspace recheck (slotlaunch.go), fresh replacement invariant (threadstore_replace.go); launcher's compares two decoded copies; previous_session compares two record copies; the rest compare process snapshots. The intent has ONE creation site (executePreparedRecovery). Fix there: `persistedRecoveryIntent` canonicalizes the whole intent through its encoding before it is first persisted, which covers every time in it (CreatedAt, inferred Context times), not only CreatedAt. No check changed.
- Test: TestRecoveryResumeWithMonotonicClockPassesWorkspaceRecheck runs inferred and proven recovery resume with production's SystemClock through the real slot store; both RED on main with the exact error, GREEN with the fix.
