---
id: 000511
status: done
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-13
estimate_hours:
card_mirror: 'ed592aa6a21142077c9000418cd683efcf461ee1' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T17:18:58-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "6de427fd", done: "dca3f81b"}
actual_hours: N/A
---

# couchcore: TestRealRunnersExitedPrecedesTheGroup/pty flakes on main (Alive = false while the backgrounded group member still runs)

## Problem

`TestRealRunnersExitedPrecedesTheGroup/pty` (`cmd/internal/couchcore/ptyrunner_test.go`, added by #489) fails intermittently on main:

    ptyrunner_test.go:179: Alive = false while the backgrounded group member still runs

Measured 2026-10-12 (pair:5, for ops:0): 5 of 20 runs fail with `go test ./cmd/internal/couchcore -run TestRealRunnersExitedPrecedesTheGroup -count=20` on a clean `origin/main` checkout (38d6cc787). It also failed in #503's full suite. The `exec` subtest passes.

Unverified hypothesis: the survivor is `sh -c '(trap "" HUP; exec sleep 30) & exit 0'`. The parent can exit, and the pty session can send SIGHUP to the group, before the backgrounded subshell has run `trap "" HUP`. The survivor then dies of the hangup, and `Alive` is truthfully false. If so, the test's fixture races, not the runner. A machine under load (#498 starts slots at nice 10) widens the window.

## Spec

- Find the cause by measurement first (e.g. record the survivor's exit signal, or have it write a marker once the trap is set), and fix it at its root: the fixture if it races, `PtyRunner`'s `Alive`/`Exited` if they do.
- Keep the test's purpose: the conformance of `FakeRunner.SetChildExited` against both real runners (#489).

## Done when

- `-count=50` of the test passes on an idle machine and under a parallel load, and the fix's reason is in the Log.

## Plan

- [x] Record the resolution already implemented, reviewed and landed by #542; close this card with its landing pointer.

## Log

### 2026-10-12
- 2026-10-12: unclaimed: filed for ops:0 as an unassigned follow-up; nothing pending

### 2026-10-13 — resolved by #542
- 2026-10-13: closed — Administrative closure only: #511 is resolved by reviewed and landed #542, PR #321, merge eb63b610aade75c040337047459669bf9d8359b9 (confirmed ancestor of this branch). Its FIFO survivor readiness fixture passed 50 runs under natural load 67 and 50 at load 7 to 6.7, focused race checks and full make -k test passed, independent review SHIP/no findings, CI 4 green. This branch only records the pointer; no code changed. Development effort is already accounted in #542, so no separate actual hours apply.; review verdict: SHIP

- Implemented and landed in [PR #321](https://github.com/xianxu/pair/pull/321), merge `eb63b610aade75c040337047459669bf9d8359b9` (verified with git rev-parse). The FIFO handshake in `TestRealRunnersExitedPrecedesTheGroup` proves that the survivor ignores SIGHUP before the leader exits; the parent exit code confirms readiness succeeded. No timing threshold changed.
- The old fixture failed 2 of 5 PTY repetitions with the old liveness behavior too; the raw group probe returned ESRCH (/tmp/pair542-descendant-baseline.log). The ready-before-exit fixture passed 50 repetitions at natural parallel load 67.32, and 50 at load 7.19 → 6.70 (2.393s, /tmp/pair542-survivor-idle50.log). The combined focused tests passed 50 times at load 35 → 34, and race checks passed.
- Full make -k test passed in #542 (/tmp/pair542-make-k-final.log; couchcore 377.409s). Its independent reviewer repeated 50 focused runs and race checks, then returned SHIP with no findings. All four PR CI checks passed.
- This branch records the closure only: implementation, full-suite verification and development effort belong to #542. No additional code, architectural surface or independently chargeable development work is introduced here (ARCH-DRY/ARCH-PURPOSE).

### 2026-10-13 — baseline-count erratum (review BR-1)

The complete /tmp/pair542-descendant-baseline.log contains **3 failing and 2 passing PTY repetitions (3/5 failures)**. Earlier 2/5 statements in this record were counted from a truncated terminal excerpt and are superseded by this correction. The reproduction, readiness fix, and all post-fix qualification results are unchanged. Verified by counting the complete per-subtest FAIL/PASS records.

## Revisions

### 2026-10-13 — close with the landed resolution pointer

TL explicitly folded this fixture correction into #542 and requested closing #511 after that landing. The original acceptance criteria are satisfied by #542's implementation and qualification. This administrative closure preserves the original diagnosis and its evidence rather than reimplementing the fix or duplicating its development hours.

### 2026-10-13 — evidence-count correction

Preserve the append-only Log and record its corrected baseline above: 3/5, not 2/5. This addresses the evidence-count-accuracy family across both #511 and #542; no executable behavior or acceptance criterion changes.
