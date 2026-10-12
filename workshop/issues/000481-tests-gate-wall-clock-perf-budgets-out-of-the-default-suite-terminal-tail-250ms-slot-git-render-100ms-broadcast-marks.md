---
id: 000481
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'a1e62a3caf06b2572cee64658d71fb71d8f5c545' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T07:50:39-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# tests: gate wall-clock perf budgets out of the default suite (terminal tail 250ms, slot-git render 100ms, broadcast marks)

## Problem

Found by the #476 sweep (fixed real-time windows asserted under load, the #473/#476 flake family). Ranks are a reading of the code, not measured failures. Paths are under `cmd/internal/`.

These default-suite tests assert a wall-clock performance budget, which a loaded `-race` suite can miss:
- wrapcmd/terminal_tail_test.go:213: a 200-line tail of a 12k-line model over a unix socket in under 250 ms.
- couchtty/console_slotgit_test.go:109: render plus RenderMenu with a hung git probe in under 100 ms.
- broadcast/marks_test.go:259,274: 20 ms per batch and 50 ms total.
- broadcast/hub_test.go:292: 10k offers in under 1 s (low).
Repository precedent: slotreconcile and sessioninventory budgets run only under `PAIR_PERF_BUDGETS`.

## Spec

Split each test into two parts. The behaviour (the tail is right, the render doesn't wait on the hung probe, offers never block) stays in the default suite with no wall-clock check; "doesn't wait" is asserted through a fake or a sync point. The budget moves behind `PAIR_PERF_BUDGETS`, like the existing perf tests.

## Done when

- No default-suite test asserts a wall-clock budget from this list; the budgets still run under `PAIR_PERF_BUDGETS`.

## Plan

The TL's rule (2026-10-12): each test asserts its outcome. A performance check stays only where performance is the
subject, measured so that load can't fake it: a count of operations, or a budget run in isolation
(`make test-perf-budgets`).

- [ ] **One switch.** `cmd/internal/perfbudget`: `Check(t, what, elapsed, budget)` logs every time and fails only when
  `PAIR_PERF_BUDGETS=1`. couchcore's `perfBudgetsEnforced` routes through it (ARCH-DRY).
- [ ] **Terminal tail** (perf is the subject): the default suite asserts the 200 lines; the 250 ms budget runs through
  `perfbudget.Check` and is added to `make test-perf-budgets`.
- [ ] **Slot-git render behind a hung probe** (the outcome is "doesn't wait"): render in a goroutine while the probe
  is held, and fail if it doesn't return before the gate opens (a hang guard, not a budget). Drop the 100 ms.
- [ ] **Marks** (the cost is the subject, an operation count): extract the stroke walk into
  `walkStrokes(points, cols, rows, put)`. The worst-case batch's walk stays within the grid-bounded count. A huge
  coordinate walks no line. The cap holds. No clock.
- [ ] **Hub offers** (the outcome is "never blocks"): 10k offers finish while the hub loop is stalled (a hang guard).
  Drop the 1 s.
- [ ] The default suite and `make test-perf-budgets` both pass. The waitUpTo windows belong to #479, which is in
  flight on another slot, so they're left alone.

## Log

### 2026-10-12
