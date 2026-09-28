---
id: 000224
status: open
created: 2026-09-09
updated: 2026-09-09
estimate_hours:
github_issue:
---

# couch's console writes have no typed single-writer door

## Problem

Found while closing `#209`, and recorded there as I-2 rather than fixed there,
because the change is couch-wide.

`Console.takeOverScreen` carried the sentence *"It is still Run-goroutine-only,
like every other writer."* It is false. `switchTo` — which calls it — is reached
from the **operationQueue** goroutine as well as `Run`:
`ExecuteConsoleOperation`'s `switch` case, wired through `couchcore`'s
`EffectConsole` dispatch. That is the switcher's Enter and the status-chip
click, i.e. the operator's primary way of changing threads, not an edge.

`#209` C2 found this while fixing the repaint nudge's ordering and fixed the
half it owned: `ptychild.Child` now owns its geometry, so no caller has an
ordering obligation about resizes. The WRITER's half is untouched. `c.host` is a
bare `io.Writer` with no serialization, and couch writes to it from several
sites (`console.go` around the takeover, the row paint, the diagnostics; plus
`console_menu.go`'s hide-cursor and cursor-position writes). Two goroutines
writing escape sequences to one terminal interleave at byte granularity, which
is the corruption class `#199` M2 spent a milestone on for `pair term`.

**The answer already exists next door.** `termcmd.paneWriter` is deliberately
NOT an `io.Writer`: `io.WriteString(m.pane, …)` and `m.pane.Write(…)` are
compile errors, so a new door cannot skip the gate's reasoning. `#199` M2 landed
that after a scanning test proved narrower than its claim — the type replaced
the test. couch runs the same primitives (`hostty.Reservation`,
`ptychild.Screen.SafeToPaint`) through an untyped writer.

This is the divergence `BR-77` and `BR-82` are both instances of: one shared
primitive, two consumers, and the reasoning lands at one of them.
