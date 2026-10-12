---
id: 000481
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'c51b335300fb91db0de7c222afa3711942160b3e' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ] Split each listed test: behaviour stays in the default suite without a wall-clock check; the budget moves behind `PAIR_PERF_BUDGETS`.
- [ ] Run the default suite and the `PAIR_PERF_BUDGETS=1` run; both pass.

## Log

### 2026-10-12
