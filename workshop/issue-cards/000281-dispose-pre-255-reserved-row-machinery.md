---
id: 000281
status: open
created: 2026-09-17
updated: 2026-09-17
estimate_hours:
github_issue:
---

# Dispose the pre-#255 reserved-row machinery reachable only from tests and a probe

## Problem

#255 M3 moved both hosts' chrome (couch's actor strip, `pair term`'s tab strip)
onto `terminal.Presenter.UpdateChrome`. The machinery that used to paint those
rows was left in the tree, and now has no production caller:

| symbol | reachable from |
|---|---|
| `ptychild.Screen` (`SafeToPaint`, `TakeRowDirty`, `HoldsCursorSave`, `MidSequence`, ...) | its own tests and `notification_benchmark_test.go` only. `TestChildHasOneTerminalAuthority` already pins that `Child` no longer holds one. |
| `hostty.Reservation.ReserveAndPaint` / `Release`, and the unexported sequences they compose (`setRegion` and six more in `reserve.go`) | `cmd/probes/couchnestedrows` only. Production uses `Reservation` for `ChildRows` arithmetic (`couchtty/console.go:1019`). |

Dead machinery keeps misleading readers. #262's diagnosis first treated
`hostty.Reservation` as a live second writer to the parent, because the code and
its comments read as if it were. #262 M1 corrected the prose. This issue decides
what the code should become.
