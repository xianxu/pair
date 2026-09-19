---
id: 000289
status: working
deps: []
github_issue:
created: 2026-09-18
updated: 2026-09-18
estimate_hours:
started: 2026-09-18T17:37:41-07:00
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

## Spec

Each symbol is either deleted with its tests, or re-homed where a production
consumer actually lives (the presenter spells several of these sequences inline).
The core-concepts contract row that names them is updated to match. A guard like
`artifactpath`'s `TestNoProductionSymbolIsReferencedOnlyByTests` is extended to
`hostty`, so the next migration cannot strand surface silently.

## Done when

- [ ] No exported `hostty` symbol is referenced only by tests. A guard test fails
      if one is.
- [ ] `core_concepts_contract_test.go`'s PURE row names only symbols that exist.
- [ ] `go test ./...` is green.

## Plan

- [ ] Per symbol: delete it, or move it to its production consumer.
- [ ] Extend the dead-symbol guard to `hostty`.

## Log

### 2026-09-18

- Filed from the #279 close review (finding `dead-exported-surface`). #279
  deleted `EnableKeyboardDisambiguation`, the #251 mechanism whose removal it
  diagnosed, and deferred the remainder of the sweep to this issue.
