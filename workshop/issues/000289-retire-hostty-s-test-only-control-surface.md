---
id: 000289
status: codecomplete
deps: []
github_issue:
created: 2026-09-18
updated: 2026-09-18
estimate_hours:
started: 2026-09-18T17:37:41-07:00
flow: {kind: quick, provenance: inferred, spec: "226b98c6", done: "6a1ec0e6"}
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

- [x] No exported `hostty` symbol is referenced only by tests, except
      allowlist entries that state why. A guard test fails otherwise.
- [x] `core_concepts_contract_test.go`'s PURE row names only symbols that exist.
- [x] The newest-page "no takeover" checks fail under a mutation that re-selects
      the current actor. The `HomeAndClear` check they replace passed under it.
- [x] `go test ./...` is green.

## Plan

- [x] Guard: multi-scope, `const`/`var`, `fake.go`, iota-zero; add hostty.
      Watch it flag the nine hostty orphans before the fix.
- [x] Delete the dead symbols and `Reservation.Paint`; move the seven sequences to `reserve.go`.
- [x] Rewrite the test oracles; override the contract row; update the atlas
      `couch.md` line ("the control constants") and #281's table.
- [x] Mutation-check the newest-page oracle; `go test ./...`.

## Log

### 2026-09-18
- 2026-09-18: closed — dead-symbol guard (now multi-scope, consts/vars) red before fix on 9 hostty orphans, green after; newest-page Token oracle FAILS under a re-select mutation where the old HomeAndClear oracle PASSED on git-archive main; go test ./... exit 0 (unsandboxed, retention env scrubbed); make test exit 0 with TMPDIR=/private/tmp (test-changelog fails only under the /var/folders symlinked TMPDIR, unrelated); review verdict: SHIP

- Filed from the #279 close review (finding `dead-exported-surface`). #279
  deleted `EnableKeyboardDisambiguation`, the #251 mechanism whose removal it
  diagnosed, and deferred the remainder of the sweep to this issue.
- Guard red before the fix, as designed. The hostty scope flagged eight
  control.go symbols plus `Reservation.Paint`, with `EdgeTop` allowlisted.
  couchcore stayed green: the iota-zero rule absorbs its five `*Unknown` values.
- Mutation check, both newest-page tests. The mutation re-selects the current
  actor in the "nothing paging" branch and re-presents it in the "already
  there" branch.
  - On a `git archive main` copy with the old `HomeAndClear` oracle, both tests
    PASS: the oracle was vacuous.
  - On this branch both FAIL at the selection check ("selection 2 -> 3"), and
    `landingOf` does not catch it.
  - `console.go` restored from git afterwards.
- `go test ./...` surfaced two consumers the design sweep missed. Both name
  the deleted file or a deleted symbol, not a live one.
  - `artifactpath.NonArtifactSources` listed `hostty/control.go`: entry removed.
  - #152's delivered-concepts contract resolves `ResetInteractiveModes`. It
    gains `issue152RetiredDeliveredConcepts`, after #149's retired-concept
    precedent. A retired row still counts toward the 21, and its declaration
    must be absent.
  - A symbol grep misses the path-shaped references. Grep the file path too.
- Also removed an orphaned comment in `console_test.go`. It described a
  differential against `hostty.RepaintFor`, which no longer exists. Left alone:
  the adjacent "C2's consumer half" block. It is stale prose about
  `RequestRepaint`, not hostty, so it falls outside this class.
- Verified: `go test ./...` exit 0 (unsandboxed, retention env scrubbed).
  `make test` exit 0 with `TMPDIR` on `/private/tmp`. Under the default
  `/var/folders` TMPDIR, `test-changelog` fails with "process target is outside
  selected owner directory". That is the symlinked tmp path, not this diff:
  the same script passes with a non-symlinked `TMPDIR`.

## Revisions

### 2026-09-18 — design

- The first Done-when row gains "except allowlist entries that state why", for
  `EdgeTop`. A newest-page oracle row is added, because the takeover checks
  turned out to be vacuous consumers of `HomeAndClear`.

### 2026-09-18 — close review (4 Minor, SHIP)

- The Plan said "ten hostty orphans"; the guard flags nine (eight `control.go`
  symbols and `Reservation.Paint`). `EdgeTop` is allowlisted, not flagged. The
  guard comment and the lesson both said "for a week"; it was three days after
  #255 M3 (2026-09-15). #281's table said "five more" sequences; it is six.
- The guard counted fakes as references while skipping them as declarations.
  Both sides now share `isProductionSource`. That surfaced couchcore's
  `joinArgs`, which only the two fakes use, so it moved into `runner_fake.go`.
- The guard's rules gained a fixture test. Disabling the iota-zero rule fails
  it, and so does dropping the fake filter.
- Open issues #217 and #241 cited deleted hostty symbols as live. Both point
  at the presenter now.
