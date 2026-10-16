---
id: 000605
status: open
deps: []
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
severity: critical
card_mirror: '0cbc2726266b786e6c7d39954a0c683c991b206f' # card fields mirrored from issue-cards; edit via sdlc
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
