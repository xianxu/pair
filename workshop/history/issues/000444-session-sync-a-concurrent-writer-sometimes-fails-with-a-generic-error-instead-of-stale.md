---
id: 000444
status: done
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: 'cb77114bb72ee6b6170a85f75c012e0f54a54fa0' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T10:58:27-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "fb8f300a", done: "bfdd3abf"}
actual_hours: 0.21
---

# session-sync: a concurrent writer sometimes fails with a generic error instead of stale

## Problem

`TestTwoProcessWritersExactlyOneWins` (pair#439, `cmd/internal/sessionsync/race_test.go`) fails about 1 run in 10 on main (TL, 10-11, 10 runs; pair:2 also listed it as pre-existing): `exit codes [0 1], want one winner and one stale`. The losing writer returned an error that is not a `Refusal` (exit 1) instead of `stale` (exit 6). The compare-and-replace may still be safe (the test stops before its body check), but an agent can't tell a lost race from a real failure, so it won't retry.

## Spec

Find the non-Refusal error the loser hits under contention (suspects: creating the `.<file>.flock` file, the shadow or ledger temp file in `writeAtomic`, the ledger read while the winner writes) and make every concurrent path either wait on the flock or return `stale`. No sleep-based fix.

## Done when

- The race test passes 200 consecutive runs (`-count=200`), and it prints the loser's error when it fails.
- The cause is named in the Log.

## Plan

- [x] Reproduce with `-count=200`, capture the loser's error; fix the path; prove with the repeated run.

## Log

### 2026-10-11
- 2026-10-11: closed — race test -count=200: 200/200 pass twice (before the fix it failed within 200; loser error: Refusal "not a session-sync file" from the unlocked pre-flock load reading the in-place rewrite torn). TestAWriteJudgesATornFileUnderTheFlock fails on the old sync.go; TestAFailedCheckDefersToAFlockCreatedMeanwhile fails when the fallback is removed (mutation checked). make -k test: only test-review (8) fails, identical on origin/main. go test ./...: sandbox socket/pty failures; unsandboxed couchcmd only fails TestContinuationWriter (no origin remote, unrelated). --no-atlas: bugfix, no new surface.; review verdict: SHIP
- Cause (measured, `-count=200` with the race helper now printing its error): the loser got `*Refusal` "tl.md is not a session-sync file". `locked` validated the file with an **unlocked** `load` before taking the flock (so a wrong path leaves no flock file), but the winner rewrites the file in place (`os.WriteFile` truncates, then writes) for parley's inode watch, so the loser's unlocked read could catch it torn. `ExitRefused` is 1, the same exit as the helper's generic-error branch, which is why it looked like a "non-Refusal" error.
- Fix: `openFlock` (sync.go). When the flock file already exists, skip the unlocked check: only the read under the lock judges the file, and nothing would be left behind. Otherwise validate, then create it. If that validation fails, open the flock file again without O_CREATE: a writer that created it meanwhile may be mid-write, so the lock judges. This is sound because a torn read implies a writer holding the flock file, and the flock file is never removed. No sleeps.
- Tests: `TestAWriteJudgesATornFileUnderTheFlock` (holds the flock over a truncated file; it fails on the old code); the race helper prints each writer's error on failure. `-count=200` race: 200/200 pass (it failed within 200 before the fix).
