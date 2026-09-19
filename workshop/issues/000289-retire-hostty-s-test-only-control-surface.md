---
id: 000289
status: working
deps: []
github_issue:
created: 2026-09-18
updated: 2026-09-18
estimate_hours:
started: 2026-09-18T17:37:41-07:00
flow: {kind: quick, provenance: inferred, spec: "226b98c6", done: "6a1ec0e6"}
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

### Disposition (measured 2026-09-18)

No symbol is re-homed. The presenter already has its own spellings, and they are
not the same bytes: its release sends one DECRST per mode where
`ResetInteractiveModes` groups them, and it has its own `altLeave` / `altEnter`.
Pointing it at hostty's constants would change wire bytes for no consumer's
benefit. It is the single parent writer now, which is what #127's "one spelling"
rule protects (ARCH-DRY).

| symbol | disposition |
|---|---|
| `HomeAndClear`, `LeaveAltScreen`, `ShowCursor`, `HideCursor`, `ResetInteractiveModes`, `EnableMouseClicks`, `PrivateModes`, `enterAltScreen` | delete, with `privatemodes_test.go`, `TestHomeAndClearResetsColourBeforeErasing` and their `TestControlSequences` rows. `enterAltScreen`'s "withdrawal test" no longer exists either. |
| `ResetRegion`, `SaveCursor`, `RestoreCursor`, `ClearLine`, `ResetSGR`, `SetRegion`, `MoveTo` | only consumer is `reserve.go`'s paint, which only `cmd/probes/couchnestedrows` uses (pair#281). Move them there, unexported. `control.go` goes; its package doc moves to `host.go`, corrected. |
| `Reservation.Paint` | test-only (#281's table says probe-reachable, which is wrong). Delete it, fold `drawRow` into `ReserveAndPaint`, and move its bracket check into the `ReserveAndPaint` tests. Strike it from #281's table. |
| `EdgeTop` | keep, allowlisted. It is deliberately "represented and refused" with the measured DECOM reason (#199 close review). |

Test oracles that name these symbols:
- `console_test.assertConsoleRestored` spells `\x1b[r` / `\x1b[?25h` inline, like its neighbours.
- `console_newest_page_test`: both "no takeover" checks look for `HomeAndClear`, which nothing writes since #255, so they cannot fail. Replace them with the presenter's selection `Token`. `Select` bumps it synchronously inside `switchTo`, before the notice the test waits on. #279 lesson: await the state the oracle reads.
- `console_notice_expiry_test.lastPaintedRow` has no caller. Delete it.

Guard: `deadsymbols_test.go` takes a list of scopes, each with its own
allowlist. It also counts `const`/`var`: hostty's dead surface is constants,
which the funcs-and-types scan cannot see. It skips `fake.go` as well as
`*_fake.go`, and skips an enum's `= iota` zero value. That value is reached by an
unset field, never by name. couchcore's five `*Unknown` values are the measured
instances.

ARCH: PURE/MOCK/CONSTRAINTS/SECURE N/A. This deletes pure strings, touches no
seam and changes no production write path. ORDER: holds no state between events;
the new oracle reads a state machine's counter after the handler has returned.
FUNERAL: creates nothing durable.

## Done when

- [ ] No exported `hostty` symbol is referenced only by tests, except
      allowlist entries that state why. A guard test fails otherwise.
- [ ] `core_concepts_contract_test.go`'s PURE row names only symbols that exist.
- [ ] The newest-page "no takeover" checks fail under a mutation that re-selects
      the current actor. The `HomeAndClear` check they replace passed under it.
- [ ] `go test ./...` is green.

## Plan

- [ ] Guard: multi-scope, `const`/`var`, `fake.go`, iota-zero; add hostty.
      Watch it flag the ten hostty orphans before the fix.
- [ ] Delete the dead symbols and `Reservation.Paint`; move the seven sequences to `reserve.go`.
- [ ] Rewrite the test oracles; override the contract row; update the atlas
      `couch.md` line ("the control constants") and #281's table.
- [ ] Mutation-check the newest-page oracle; `go test ./...`.

## Log

### 2026-09-18

- Filed from the #279 close review (finding `dead-exported-surface`). #279
  deleted `EnableKeyboardDisambiguation`, the #251 mechanism whose removal it
  diagnosed, and deferred the remainder of the sweep to this issue.

## Revisions

### 2026-09-18 — design

- The first Done-when row gains "except allowlist entries that state why", for
  `EdgeTop`. A newest-page oracle row is added, because the takeover checks
  turned out to be vacuous consumers of `HomeAndClear`.
