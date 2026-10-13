---
id: 000511
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'f6fb52569cc86bbc427b46129f4102ce2bff1bb3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T17:18:58-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
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

- [ ]

## Log

### 2026-10-12
