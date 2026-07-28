---
id: 000124
status: working
deps: []
github_issue:
created: 2026-07-28
updated: 2026-07-28
estimate_hours: 0.15
started: 2026-07-28T08:11:35-07:00
---

# Alt+Shift+Enter re-tiles in one blind burst

## Problem

The #123 tiled toggle re-tiles the terminal column via a converge loop:
read geometry → one resize step → 80ms settle (zellij applies resizes
asynchronously) → re-read → repeat until within tolerance. Live it takes
~3 visible steps and ~500ms — the user reports it feels slow.

Zellij 0.44.3's tiled resize step is a fixed fraction of the screen:
5% per `resize increase|decrease left` (RESIZE_PERCENT; measured live
7–8 cols on a 150-col screen — the `* 2.0` source path does not apply
to this action, an earlier misread). The toggle delta (1/2 ↔ ~2/3) is
therefore always exactly three steps — measuring and settling buys
nothing (user decision: "just do 1/3, 2/3 expansion").

## Spec

- `Alt+Shift+Enter` reads pane state once (expanded when the terminal is
  ≥60% of the tiled screen width), then fires exactly three
  `resize increase left` (expand) or `resize decrease left` (collapse)
  actions back-to-back — no settle pauses, no re-reads, no convergence
  loop (`Simplicity First`).
- Expanded lands at 65/35 (the 1/3–2/3 arrangement given zellij's 5%
  step); collapse returns to exactly 50/50. Verified live.
- The pure planner (`ARCH-PURE`) maps (terminalCols, screenCols) → the
  action burst; the executor is a thin fire-and-forget shell. The
  converge-loop machinery (`terminalResizeStep`, tolerance, step cap,
  settle delay) is deleted (`Root Cause`, not a tuned loop).
- If zellij refuses a step (minimum sizes), the extra action is a
  harmless no-op; a future zellij step-size change degrades to a
  different stable pair of widths, still classified by the ≥60%
  threshold.

## Done when

- Live: `Alt+Shift+Enter` toggles 50/50 ↔ ~65/35 with zero pair-added
  delay (the only remaining latency is zellij's own application), both
  split and unsplit.
- Unit tests cover the planner's direction/refusal cases and the
  burst contract; the stateful-resize fake and settle-delay
  tests are removed with the loop.
- `tests/term-pane-shortcuts-test.sh` expects the burst ops.

## Estimate

Method A against `estimate-logic-v3.1` (source stale, same caveat as #123):
a single well-specced simplification inside an existing Go module — the
design is already resolved in the Spec (user decision + #123 live data), so
design hours are minimal.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.05 impl=0.10
total: 0.15
```

## Plan

- [x] RED: planner + executor tests expect the burst; shell test expects the burst ops.
- [x] Implement burst planner, delete converge loop + settle delay; GREEN.
- [x] Rebuild, live-verify toggle speed in a fresh session, close.

## Log

### 2026-07-28
- Implemented the blind burst; live calibration in an isolated session
  corrected the step model: zellij's `resize` step is 5% of the screen
  (not 10% — successive reads during application caused the earlier
  misread; back-to-back actions all apply, so no inter-step gap is
  needed). Burst = 3 steps: expand 75→97/150 (64.7% ≈ 2/3), collapse
  97→75 exactly, identical with a split present. Deleted
  `terminalResizeStep`/target/tolerance/step-cap/settle-delay converge
  machinery and the stateful-resize fake mode. Verified: go test
  layoutcmd/termcmd; tests/term-pane-shortcuts-test.sh; live smoke as
  above.
