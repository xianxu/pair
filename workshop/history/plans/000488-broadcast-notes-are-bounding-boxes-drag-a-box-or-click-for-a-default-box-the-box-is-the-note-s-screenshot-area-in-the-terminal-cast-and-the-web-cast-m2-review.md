# Boundary Review — pair#488 (milestone M2)

| field | value |
|-------|-------|
| issue | 488 — broadcast: notes are bounding boxes — drag a box (or click for a default box); the box is the note's screenshot area, in the terminal cast and the web cast |
| repo | pair |
| issue file | workshop/issues/000488-broadcast-notes-are-bounding-boxes-drag-a-box-or-click-for-a-default-box-the-box-is-the-note-s-screenshot-area-in-the-terminal-cast-and-the-web-cast.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 1bd525429a569a2e567153267c82705d31d115d8..ff0d7010ed6913fa576b925579b4895a4cd7d7bc |
| command | sdlc milestone-close --issue 488 --milestone M2 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T12:21:50-07:00 |
| verdict | REWORK |

## Review

Reviewing the M2 diff now — rest of Go tests, viewer.js, then running tests.

Found a likely integer-overflow in `CellBox.within`; verifying with a probe test, then running the suites.

Confirmed: `ParseNote` hangs on `box:[1,0,9223372036854775807,1]`. Now running the Go and node test suites.

Generating the embedded runtime bundle so the couchtty note tests can build, then running them.

```verdict
verdict: REWORK
confidence: high
```

M2 does what the plan asks for. A terminal note now carries `box:[col,row,w,h]`. The board picks the pin inside the box: top-right first, then leftward, then the rows below, never on the controls. `DrawNotes` tints the box's edge cells in the author's color, and `NoteCrop` gives #455 a crop. The viewer handles both drag and tap through the shared `gestureBox`/`dragMin`, with a larger threshold for touch. README and atlas are updated. Tests pass: `go test ./cmd/internal/broadcast/...` is ok; the couchtty note tests pass once the runtime bundle is generated (the bundle is missing in the checkout, which is an environment issue, not this diff); and the node viewer tests show 70 pass, 0 fail.

One Critical bug blocks SHIP. `CellBox.within` checks `b.Col+b.W <= cols` without guarding against overflow. A body with `"box":[1,0,9223372036854775807,1]` passes that check, and then `pinSpots` counts down from `MinInt-2`, which wraps to `MaxInt-1`, through about 9×10¹⁸ iterations. I reproduced it: a probe test calling `ParseNote` on that body was still running after 3 s and failed on the timeout. Anyone on the pointer link can therefore pin one Couch CPU core per request, indefinitely. The fuzz oracle repeats the same unguarded addition, so it can't catch this.

### 1. Strengths
- **Pin placement is pure and well tested.** `pinSpots` plus `NoteBoard.Add(n, cols, rows)` (`cmd/internal/broadcast/notes.go:84-138`) implement the hub's "first left, then down" guard rule. `TestNoteBoardPlacesThePin` covers top-right, next-left, the row below, the last row next to the controls, and the case where nowhere is free.
- **The page never sends the pin.** `ParseNote` rejects a body with both `box` and `col/row`, and the old point format, because the decode is strict. The seeds cover null, short and long arrays, floats, strings and truncated bodies.
- **The outline reuses the glow's tint merge** through the `take` closure (`notes.go:281-291`). It changes backgrounds only and never touches the controls (`boxEdge` checks `guarded`), and it is tested for edge versus interior cells.
- **Viewer gesture tests cover both pointer types and cancellation:** tap gives the default box, a drag follows the outline, a 1-cell mouse drag counts as a box, a 1-cell finger wobble counts as a tap, and pointercancel or turning pointing off drops the box and opens no draft (`tests/broadcast-viewer/pointermode.test.mjs`).
- **The wire change is a clean cut.** Page and server ship together, which the plan accepted. `parseNotes` on the page now requires a well-formed `box`.

### 2. Critical findings
- **`cmd/internal/broadcast/notes.go:78-80` and `ParseNote` (around line 205): integer overflow in the box bounds check lets a viewer request run forever (ARCH-SECURE, ARCH-CONSTRAINTS).**
  - `within` computes `b.Col+b.W` and `b.Row+b.H` from viewer-controlled int64 values. When the sum reaches 2⁶³ it wraps negative, so `<= cols` is true. `pinSpots` then loops with `b.Col+b.W-2`, which wraps to about `MaxInt`.
  - Reproduced with `{"cols":80,"rows":24,"box":[1,0,9223372036854775807,1],"color":0,"text":"x"}`.
  - **Rule to apply:** bound each untrusted integer to its range before doing arithmetic on it. Check `0 <= Col < cols`, `2 <= W <= cols-Col` (subtract, don't add), and the same for Row and H, or cap each component at `maxPointGrid` first.
  - **Test change:** add the overflowing body to `noteRefusals`, and give it a deadline so it can't hang the suite. Rewrite the `FuzzParseNote` oracle (`notes_test.go`, around line 259) in the subtraction form, so it is an independent statement of the invariant rather than a copy of the implementation's arithmetic.
  - **Sweep:** I checked the sibling `webcast/board.ParseNote`. It parses floats through `boxOf`/`coord` bounds and isn't affected.

### 3. Important findings
None.

### 4. Minor findings
- **ARCH-DRY:** `notes.js` `boxHasPin` copies Go's `pinSpots` loop by hand. This crosses a language boundary and the comment says so, but nothing ties the two together. A shared table of cases run by both the Go and node tests would keep them from drifting.
- **ARCH-ORDER:** `pointerMode` keeps `gesture` and `stroke` as two nullable fields that are only mutually exclusive because `end()` runs first. A single tagged `drag` state (`{kind:'box'|'stroke'}`) would make that explicit.
- `ErrNoteOverlap` ("a note is already there") is also returned when every pin spot in the box is taken. The wording still fits, so no change is needed.

### 5. Test coverage notes
- `go test ./cmd/internal/broadcast/...` is ok (54 s).
- `go test ./cmd/internal/couchtty -run 'Note|WebNote'` is ok after generating `runtimebundle/assets/runtime`.
- `node --test tests/broadcast-viewer/*.test.mjs`: 70 pass, 0 fail.
- Gap: no refusal seed or fuzz invariant would detect integer overflow (see the Critical finding). The `9999999999` seed is large but doesn't wrap.

### 6. Architectural notes for upcoming work (M3)
- `AnchorFor` will read `Note.Box`. Once the Critical is fixed, `Box` is guaranteed to be on the grid it was accepted on. M3 still has to handle a grid that shrank afterwards, which `boxEdge` and `NoteCrop` already do by cutting the box.
- The plan's ORDER items for the async anchor resolver are the real state-machine work. Keep the single-flight pass, generation and epoch checks behind an injected resolver with controlled completion order, as the PQ-1 revision specifies.
- ARCH-MOCK / ARCH-FUNERAL: M2 introduces no external dependency and nothing durable; notes stay in memory and are cleared when pointing turns off or the broadcast ends. Both pass.
- ARCH-PURPOSE: passes for M2's scope.

### 7. Plan revision recommendations
- Under ARCH notes → SECURE, add: "Viewer box components are bounded before arithmetic (no `Col+W` on untrusted ints); the fuzz oracle states the bound independently."

```findings
findings:
  - id: new
    severity: Critical
    family: bound-untrusted-ints-before-arithmetic
    title: |
      ParseNote box check overflows: box [1,0,MaxInt64,1] passes within() and pinSpots spins ~9e18 iterations (viewer-triggered CPU hang)
    detail: |
      cmd/internal/broadcast/notes.go within() adds untrusted Col+W / Row+H without overflow guards; Col+W reaching 2^63 wraps negative and passes, then pinSpots starts at Col+W-2 which wraps to about MaxInt. Reproduced: ParseNote on that body still running after 3s. Fix: compare by subtraction (W <= cols-Col, H <= rows-Row) after range-checking Col/Row, or cap each component at maxPointGrid first; add the body to noteRefusals with a deadline, and rewrite the FuzzParseNote oracle so it doesn't repeat the same overflowing addition.
  - id: new
    severity: Minor
    family: cross-language-mirror-unpinned
    title: |
      notes.js boxHasPin hand-mirrors Go pinSpots with no shared case table
    detail: |
      A shared table of cases run by both Go and node tests would catch the two drifting apart (ARCH-DRY).
  - id: new
    severity: Minor
    family: state-as-nullable-constellation
    title: |
      pointerMode keeps gesture and stroke as two nullable fields, exclusive only because end() runs first
    detail: |
      Collapse them into one tagged drag state (ARCH-ORDER).
```

---

## Re-review — 2026-10-12T12:36:54-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 488 — broadcast: notes are bounding boxes — drag a box (or click for a default box); the box is the note's screenshot area, in the terminal cast and the web cast |
| repo | pair |
| issue file | workshop/issues/000488-broadcast-notes-are-bounding-boxes-drag-a-box-or-click-for-a-default-box-the-box-is-the-note-s-screenshot-area-in-the-terminal-cast-and-the-web-cast.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 1bd525429a569a2e567153267c82705d31d115d8..f622966e2e280698d0a8a3b0d75d8afec3ec38a8 |
| command | sdlc milestone-close --issue 488 --milestone M2 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-12T12:36:54-07:00 |
| verdict | SHIP |

## Review

I've read the full diff; now checking the plan's M2 scope and running the tests.

Broadcast Go and node tests pass. Now trying to generate the runtime bundle so the couchtty integration tests can run.

```verdict
verdict: SHIP
confidence: high
```

**Verdict: SHIP.** M2 delivers what the plan promised for terminal boxes:
- The wire now carries `box`, and Couch places the pin itself.
- Comment mode supports dragging a box and tapping for a default one, with a dashed `.drag-box` while drawing.
- `DrawNotes` outlines the box, and `NoteCrop` is in place.
- README and atlas are updated.

All three prior findings are fixed. For BR-10 I reverted the fix in the disposable checkout (both the per-component bound in `ParseNote` and the subtraction form of `within()`). `TestParseNoteRefusesQuickly` then failed at its 5 s deadline, and with the fix restored it passes. All tests ran green on the pinned head:
- `go test ./cmd/internal/broadcast/`
- `go test ./cmd/internal/couchtty/ -run 'Note|Viewer'`, after generating the runtime bundle with `make runtimebundle-generate`'s command
- all 70 node tests in `tests/broadcast-viewer/*.test.mjs`

What remains is two Minor findings.

### 1. Strengths
- **Two layers against the overflow.** `ParseNote` checks each of the box's four numbers against `maxPointGrid` before any arithmetic (`notes.go`, the box loop). `within()` then compares by subtraction (`W <= cols-Col`). Either layer alone blocks the BR-10 hang.
- **The fuzz test checks bounds independently.** `FuzzParseNote` now states its own limits without adding the request's numbers, so it no longer shares the parser's bug. The overflow bodies are in `noteRefusals`, guarded by a deadline test.
- **One case table for Go and the viewer.** `tests/broadcast-viewer/pinspots.json` is run by both the Go test and the node test, and the Go side asserts that `guard == StatusGuardCols`, so the two sides can't drift silently.
- **`pointerMode` keeps one `drag` state** tagged `stroke` or `box`, and every path out goes through `end()`. Tests cover a cancelled pointer and turning pointing off mid-drag.
- **Pin placement is a pure function** (`pinSpots` plus `NoteBoard.Add(n, cols, rows)`) with table tests, including the controls on the last row.

### 2. Critical findings
None.

### 3. Important findings
None.

### 4. Minor findings
- **`notes.go` `pinSpots`: the work is proportional to the request's box area.** `ParseNote` only needs to know whether any pin spot exists, but it builds the full list. On a 1000×1000 request that is about 1M cells, roughly 16–32 MB allocated. This happens before `landsNow` rejects a grid that doesn't match the screen. Fix: add an early-exit `hasPinSpot` for the parser, or have `Add` stop at the first free spot (ARCH-CONSTRAINTS).
- **`CellBox.UnmarshalJSON` has no production caller**, only the round-trip test. Keep it or drop it, but its comment ("Couch's own notes event") suggests a reader that doesn't exist.
- **`notes.js` `isBox` accepts a width of 1, but Go requires `W >= 2`.** It's harmless because the data comes from Couch, but the bound differs from Go.

### 5. Test coverage notes
- **BR-10:** the regression test fails without the fix. Reverting only `within()` stays green because the per-component bound still holds; the two layers are intentional.
- **Viewer:** node tests cover a mouse tap, a dragged box, the touch threshold, a box over only the controls, cancel, and pointing turned off. The Go tests cover the outline (edge cells tinted, text kept, controls never outlined), the clip on a shrunk grid, the wire round trip, and `NoteCrop`'s clamping.

### 6. Architectural notes
| Principle | Result | Note |
|---|---|---|
| ARCH-DRY | pass | `gestureBox` is reused from M1, and the shared table pins `boxHasPin` against `pinSpots`. |
| ARCH-PURE | pass | `pinSpots`, `boxEdge`, `NoteCrop` and `within` are pure; the IO glue only adds the `cols, rows` parameters. |
| ARCH-PURPOSE | pass | The M2 scope is fully delivered; the anchor is M3, as planned. |
| ARCH-MOCK | N/A | No external dependency in M2. |
| ARCH-CONSTRAINTS | minor | The `pinSpots` allocation above. |
| ARCH-SECURE | pass | The page never sends the pin; the box is bounds-checked component by component; errors are fixed values that don't echo the input. |
| ARCH-ORDER | pass | The single tagged `drag` state (BR-12). On the server, `addNote(…, gen)` still checks the pointer generation, and it now places the pin under the same lock. |
| ARCH-FUNERAL | pass | Nothing durable is created: boxes live on in-memory notes, which are cleared on pointer off. |

For M3: the anchor resolver should read `Note.Box` as `NoteCrop` does, and treat a box that no longer fits after a resize as partial.

### 7. Plan revision recommendations
- **Clamp vs refuse:** the plan's Terminal-cast "Wire" bullet says "The box is clamped to the grid", but the code refuses an off-grid box with `ErrNoteRange`. That matches the later "never a clamped fabrication" rule. Add a "## Revisions" entry: "M2 as built: off-grid boxes are refused, not clamped; the pin is placed by `NoteBoard.Add(n, cols, rows)` via `pinSpots`."

```findings
dispose:
  - id: BR-10
    disposition: addressed
    note: |
      Per-component bound in ParseNote plus a subtraction-based within(); reverting both makes TestParseNoteRefusesQuickly fail at its 5 s deadline; FuzzParseNote now states its bounds without addition.
  - id: BR-11
    disposition: addressed
    note: |
      tests/broadcast-viewer/pinspots.json is run by TestPinSpotsCaseTable (Go, which also checks guard == StatusGuardCols) and by notes.test.mjs (boxHasPin).
  - id: BR-12
    disposition: addressed
    note: |
      viewer.js pointerMode has one tagged drag state (stroke or box), cleared only by end(); cancel and pointing-off are covered in pointermode.test.mjs.
findings:
  - id: new
    severity: Minor
    family: untrusted-size-drives-work
    title: |
      pinSpots builds the full spot list (about 1M cells) just so ParseNote can check that one exists
    detail: |
      A 1000x1000 request allocates roughly 16-32 MB in ParseNote before landsNow rejects the mismatched grid. Use an early-exit hasPinSpot in the parser, and stop at the first free spot in Add (ARCH-CONSTRAINTS).
  - id: new
    severity: Minor
    family: plan-names-match-code
    title: |
      The plan says the terminal box is clamped to the grid; the code refuses it with ErrNoteRange
    detail: |
      This is the 2nd finding in family plan-names-match-code. The rule: every M-close adds an "as built" Revisions entry that reconciles each Design bullet the milestone touched. Add "M2 as built: off-grid boxes are refused; the pin is placed by NoteBoard.Add via pinSpots."
```

My first attempt at the revert check, the one that hung and moved to the background, has now finished. Its last step put `notes.go` back. `git status` and `git diff` are both empty, so the review checkout matches the pinned head exactly.

The verdict stays **SHIP**. Separately from that run, I re-ran the check with a 30 s timeout. Without the BR-10 fix, `TestParseNoteRefusesQuickly` fails at its 5 s deadline, and with the fix it passes.
