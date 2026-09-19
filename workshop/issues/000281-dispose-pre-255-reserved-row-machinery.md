---
id: 000281
status: open
deps: [pair#255]
github_issue:
created: 2026-09-17
updated: 2026-09-17
estimate_hours:
---

# Dispose the pre-#255 reserved-row machinery reachable only from tests and a probe

## Problem

#255 M3 moved both hosts' chrome (couch's actor strip, `pair term`'s tab strip)
onto `terminal.Presenter.UpdateChrome`. The machinery that used to paint those
rows was left in the tree, and now has no production caller:

| symbol | reachable from |
|---|---|
| `ptychild.Screen` (`SafeToPaint`, `TakeRowDirty`, `HoldsCursorSave`, `MidSequence`, ...) | its own tests and `notification_benchmark_test.go` only. `TestChildHasOneTerminalAuthority` already pins that `Child` no longer holds one. |
| `hostty.Reservation.ReserveAndPaint` / `Release`, and the unexported sequences they compose (`setRegion` and five more in `reserve.go`) | `cmd/probes/couchnestedrows` only. Production uses `Reservation` for `ChildRows` arithmetic (`couchtty/console.go:1019`). |

Dead machinery keeps misleading readers. #262's diagnosis first treated
`hostty.Reservation` as a live second writer to the parent, because the code and
its comments read as if it were. #262 M1 corrected the prose. This issue decides
what the code should become.

## Spec

Per symbol: delete it, with its tests, or keep it with a stated reason. Two
judgements matter:

- `couchnestedrows` measured whether two reserved rows compose, and there are
  no longer two DECSTBM reservations to compose. Decide whether the probe still
  answers a live question, or retire it with its `atlas/index.md` mention.
- `ptychild.Screen`'s notification OSC scanning may still carry value that
  `terminal.Endpoint` re-implements. Check before deleting, so a capability is
  not lost.

## Done when

- Every symbol in the table is deleted or kept with a reason in `## Log`.
- No comment or atlas line describes a deleted mechanism as live. The #262 M1
  sweep vocabulary returns nothing stale: `SafeToPaint`, `TakeRowDirty`,
  `ReserveAndPaint`, `paneWriter`, `owe-and-flush`.

## Plan

- [ ] Disposition each symbol; delete or document; re-run the vocabulary sweep.

## Log

### 2026-09-17

Filed from #262 M1's stale-prose sweep. It is not folded into #262 because
deleting code is separable from the flicker fix. #262 only corrects the prose
that presented this machinery as live.

### 2026-09-18

- pair#289 deleted `Reservation.Paint`, which only tests reached (the table
  above called it probe-reachable). It also moved `SetRegion` and the other
  sequences `ReserveAndPaint`/`Release` compose into `reserve.go`, unexported,
  and deleted `control.go`. Whatever this issue decides for `ReserveAndPaint`
  and `Release` decides for them too. `artifactpath`'s dead-symbol guard now
  covers hostty, so deleting the painters without their sequences fails a test.
