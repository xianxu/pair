---
id: '000289'
status: done
started: 2026-09-18T17:37:41-07:00
created: 2026-09-18
updated: 2026-09-18
actual_hours: 0.48
---

# Retire hostty's test-only control surface

## Problem

Since #255 M3 (`f32bb4cf`) moved every parent-terminal write into
`terminal.Presenter`, `cmd/internal/hostty/control.go` has no production
consumer. A grep of non-test code on 2026-09-18 found zero references to
`ResetRegion`, `SaveCursor`, `RestoreCursor`, `ClearLine`, `ResetSGR`,
`HomeAndClear`, `LeaveAltScreen`, `ShowCursor`, `HideCursor`,
`ResetInteractiveModes`, `EnableMouseClicks`, `PrivateModes`, `SetRegion` or
`MoveTo`. Only tests read them: `hostty_test.go`, `privatemodes_test.go`,
`couchtty/console_test.go` and `couchtty/core_concepts_contract_test.go`.

The package's own rule, stated at `enterAltScreen`, is that exported surface
needs a consumer outside its own package's tests. The doc comments also still
describe these constants as the live mechanism ("Written on teardown", "couch
asserts its own"). That is how #279's regression hid: `EnableKeyboardDisambiguation`
read as Couch's keyboard policy for three days after nothing wrote it any more.
The #279 close review found it and deleted it, and left the rest here.
