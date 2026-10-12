---
id: 000473
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'a3b61f0713973d476456218ada68f2a7c8f06fa5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T03:19:28-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# tests: two flaky pair tests (couchtty collaboration timers; couchcore launch-helper post-ack retries)

## Problem

Two pair tests fail intermittently on main: `couchtty` `TestCollaborationTimersRearmAndTearDown` (about 1 in 8, reported by pair:1 on 10-12) and `couchcore` `TestPostAckHandleRetriesReuseOneWaiterUntilReap` (failed once in a full run, then passed 8/8, reported by pair:2 on 10-12). Flaky tests cost every slot's full-suite run and erode trust in failures.

## Spec

Reproduce each under load (`-count` and `-race`, and inside a full `make -k test`), find the race or timing assumption, and fix the root cause (an injectable clock, deterministic ordering, waiting on the event instead of time). No retries or longer sleeps as the fix. If the cause is in product code rather than the test, fix the product.

## Done when

- Each test passes 200 runs under `-race` and in repeated full-suite runs; the cause is recorded in the Log.

## Plan

- [x] couchtty: the `pointer off` subtests wait for the Run loop to finish the step that turned pointing off (`consoleFixture.settle`) before checking the timers at once.
- [x] couchcore: the post-ack cleanup takes its timeouts from an injectable `Couch.after` (time.After by default); the two post-ack tests give it a clock that follows the fake child (gives up at once while it lives, waits for the reap once it is dead).
- [x] Prove each cause by widening its window (a sleep injected in product code / the fake), then 200 `-race` runs of each under parallel load and repeated full-suite runs.

## Log

### 2026-10-12

- **couchtty cause (measured).** `pointerOff` sets `c.pointer = pointerOff` under `c.mu`, unlocks, and only then runs `resyncNotes` + `clearMarks` in the same Run-loop step. The test polled the phase from its own goroutine, so it could see "off" between the flip and the teardown, and its at-once check found the timer still armed. A 20ms sleep after the unlock fails both `*/pointer off` subtests 3/3; with the fix they pass 5/5 with that sleep in place. The intermediate state isn't a product bug: every product reader of the phase runs on the same loop (input, timer callbacks via `runTerminalCommand`), so only the test saw it. The fix is a barrier: a no-op `runTerminalCommand` returns only after the loop finishes the step in progress. `broadcast end` was not exposed: `stopBroadcast` tears down before `broadcastEnded` resets the phase.
- **couchcore cause (measured).** The fake child dies on its 3rd KILL; `handleCleanup.attempt` then gives the `Wait` goroutine `postAckQuiesceTimeout` (2ms in the test) of real time to deliver the reap. When the scheduler runs the waiter late, the attempt times out, retries, and sends a 4th KILL, so `kills != 3`. A 5ms delay in the fake's `Wait` after death fails both post-ack tests (`TestPostAckHandleRetriesReuseOneWaiterUntilReap` and its sibling `TestPostAckCleanupEndsItsHelperWithoutOwningTheSession`, which had the same flake) 3/3 with `kills:4`; with the injected clock they pass 5/5 with the delay in place. Not reproduced unaided in 4,800 parallel `-race` runs; the reported rate (once in a full run) fits a scheduler stall.
- **Proof.** 200 runs of each test under `-race`, 4 processes per test, both packages at once: 0 failures.
