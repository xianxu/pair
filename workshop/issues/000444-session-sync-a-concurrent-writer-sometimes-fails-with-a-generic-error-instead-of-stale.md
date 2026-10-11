---
id: 000444
status: open
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: 'b923ebd93545651dc72a808b60b908c22a75fee0' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ] Reproduce with `-count=200`, capture the loser's error; fix the path; prove with the repeated run.

## Log

### 2026-10-11
