# Boundary Review — pair#447 (milestone M3)

| field | value |
|-------|-------|
| issue | 447 — Broadcast sticky notes: viewers pin comments on the screen |
| repo | pair |
| issue file | workshop/issues/000447-broadcast-sticky-notes-viewers-pin-comments-on-the-screen.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 75f2e1ac3e41edd226abbe451a19b8365b64ff42..f7a574e58055544fbf5ee84cf70a5bb79a59e96c |
| command | sdlc milestone-close --issue 447 --milestone M3 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T15:47:45-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

BR-1 is addressed in the plan. Next I'm reading the M3 code diff.

Checking how the renderer treats cells past a row's used columns (affects whether glow shows on short rows).

Renderer paints backgrounds past used columns, so the glow renders. Now the viewer-side diff and tests.

Running the Go suites (with -race) and the viewer node tests.

PTY tests fail on environment permissions (unrelated). Checking whether the soak-test data race touches the notes code.

Unrelated (soak-test cleanup after environment failure); note/pointer/mark tests pass 3× under -race. Running the viewer node tests.

Checking whether the glow can tint a guarded control cell via the wide-char back-step.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M3 delivers what the plan describes. The note's 💭 now sits on a chip in its author's color, with a round glow that flashes when the note arrives and every 15s after. `paletteTint` is shared by marks and notes, the console runs one flash timer, and the viewer's hint bar has been replaced by tooltips. The atlas and README cover the new surface.

I checked the marks refactor by hand. The old ladder index `k*5/10` and the new `int(5·(k+1)/11)` give the same index for every k in 0..9, so marks look exactly as before.

Tests:
- `go test -race ./cmd/internal/broadcast` passes.
- The couchtty Note, Pointer and Mark tests pass three times in a row under `-race`.
- `node --test tests/broadcast-viewer/*.test.mjs` passes, 60 of 60.
- The couchtty failures that remain don't come from this diff. The PTY tests fail with `operation not permitted` when this machine tries to start a pty child. The soak test's data race happens in its cleanup, after that same start failure.

BR-1 is addressed. One gap holds this back from SHIP: two of the tests Task 13 promised are missing — one for the timer re-arming after it fires, and one for the timer being cleared when the broadcast stops. It's not blocking.

1. **Strengths**
   - `cmd/internal/broadcast/marks.go:98` `paletteTint` and `tintCell` are now the only place a pointer color is weakened or a wide character's background is set. `Marks.Overlay` was reduced to call them, and the existing mark tests pin that nothing changed (ARCH-DRY done well).
   - `notehalo.go` is pure: `noteFlash`, `noteGlow` and `NoteNextChange` take the time as an argument. The tests check the exact glow cells, the flash phases at each boundary, and that the soonest of several notes wins.
   - `DrawNotes` is order-independent even with the glow. Ties are broken by strength and then by color, and both halves of a wide character get the stronger tint. `TestDrawNotesOrderIndependentWithGlow` (200 random cases with wide text) checks this.
   - `scheduleNotes` always works out the next timer from the current snapshot. A callback that fires late after pointer-off or broadcast end therefore re-arms nothing.
   - On the viewer, the `sawCaps` fix in `connect` has its own regression test for the view-page case. The asset check in `server_test.go` confirms the page has no `#hint` or `#controls`.

2. **Critical findings**
   - None.

3. **Important findings**
   - **Two Task 13 tests are missing** (`cmd/internal/couchtty/console_notes_test.go:177`). The plan asked for a test that the timer repaints at the flash boundary and for `pnotes.timer == nil` after a broadcast stop. `TestNoteChipAndFlashTimer` only checks that the timer is armed once and is gone after pointer-off. Nothing fails if:
     - the AfterFunc callback stops calling `scheduleNotes`, so the flash runs once and never again; or
     - `forgetNotes` stops clearing the timer.

     This is the **2nd finding in family `untested-cleanup-path`**. The rule: every timer the console arms needs a test for each way it is torn down, plus a test that it re-arms after it fires. Today that means `pmarks.timer` and `pnotes.timer`, each on pointer-off and on broadcast end (`resetPointer`). Neither timer has a broadcast-end test. A table-driven test would cover all four cases. For the re-arm case, check the delay through a seam or use the console's test clock.

4. **Minor findings**
   - **The blend predicate is written twice:** `Known && TrueColor && !NoColor` appears in `noteStyleLocked` (`console_notes.go:69`) and inline in `applyPoints` (`console_pointer.go:180`). This is the **2nd finding in family `shared-helper-not-reused`**. The rule: when a new helper encodes a predicate that already exists inline, the old site calls the new helper in the same change. `applyPoints` should derive its blend from `noteStyleLocked()`, or both should use a shared `paletteBlendLocked()`. A grep found no other copies.
   - **Note times use a different clock from drawing (ARCH-PURE):** `Note.At` is stamped with wall-clock `time.Now()` in `session.go:432`, while the console draws with its injectable `c.now`. With a fake console clock, ages come out negative and are clamped to 0, so the note looks permanently flashing. It's fine in production, but stamp both from one clock source.
   - **Worst-case repaint rate isn't stated (ARCH-CONSTRAINTS):** the plan says about 11 repaints per note every 15s. At the 48-note limit that is about 35 full-screen repaints a second, each sent to every viewer. Write that ceiling down, or snap flash phases to a coarser grid (e.g. 100 ms) so repaints for several notes coincide.
   - Stale comment: `drawnNotes` says the overlay reads it, but `noteOverlay` now reads `pnotes` directly.

5. **Test coverage notes**
   - The plan's "overlap: the stronger glow wins" case isn't asserted directly. Only determinism is covered, through the order-independence property.
   - The plan's "truecolor fixture" for the console chip test was swapped for the indexed color `markTint`.
   - `TestDrawNotesBlankKeepsStyle` now checks that a blanked half keeps its foreground rather than its background. That is correct, because the glow always sets the background of a blanked half.
   - I checked one possible bug and found none: the wide-character back-step in `tintCell` can't reach a guarded cell, because the guard ends on 👽's continuation.

6. **Architectural notes for upcoming work (M4)**
   - `pnotes.style.Hover` is overwritten with 0 every time a snapshot is accepted. M4's hover state needs to live outside `noteStyleLocked`'s result, or be merged in at draw time, or every new snapshot will drop the hover.
   - When M4 takes the notes timer into account, give the menu's hover hold a test for the timer being re-armed when the hover ends.

7. **Plan revision recommendations**
   - None needed. The plan matches the code, apart from the missing Task 13 tests, which are a code fix rather than a plan change.

Architecture markers, one by one:
- **ARCH-DRY:** passes, with one Minor finding (the duplicated blend predicate).
- **ARCH-PURE:** passes, with one Minor finding (two clocks).
- **ARCH-PURPOSE:** passes. Chip, glow, flash and tooltips are all delivered and documented.
- **ARCH-MOCK:** not applicable. No new external dependency.
- **ARCH-CONSTRAINTS:** passes, with one Minor finding (repaint ceiling not stated).
- **ARCH-SECURE:** passes. `At` is `json:"-"`, and no input crosses a new boundary.
- **ARCH-ORDER:** passes. All timer state sits under the `pnotes.mu` leaf lock, and each schedule is worked out again from the current snapshot.
- **ARCH-FUNERAL:** passes. There is one in-memory timer, stopped on pointer-off and on broadcast end, but the broadcast-end path is untested (see the Important finding).

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Plan Task 3 Step 3 now reads "Implement DrawNotes per Core concepts" (diff 75f2e1ac..f7a574e5); the uv.EmptyCell wording appears only in Core concepts' rationale (plan line 37).
findings:
  - id: new
    severity: Important
    family: untested-cleanup-path
    title: |
      Task 13 tests missing: flash-timer re-arm after firing and notes timer cleared on broadcast stop
    detail: |
      2nd finding in this family. Rule: every timer the console arms needs a test for each way it is torn down plus a test that it re-arms after firing; today that is pmarks.timer and pnotes.timer, each on pointer-off and on broadcast end (resetPointer). Only notes-on-pointer-off is tested (console_notes_test.go:177). Dropping the scheduleNotes call from the AfterFunc callback, or the timer stop from forgetNotes, keeps the suite green. Add one table-driven test over both timers and both teardown paths, plus a re-arm check through the test clock or a delay seam.
  - id: new
    severity: Minor
    family: shared-helper-not-reused
    title: |
      Blend predicate Known && TrueColor && !NoColor duplicated in noteStyleLocked and applyPoints
    detail: |
      2nd finding in this family. Rule: when a new helper encodes a predicate that already exists inline, the old site calls the new helper in the same change. A grep finds exactly two sites: console_notes.go:69 and console_pointer.go:180. applyPoints should take its blend from noteStyleLocked, or both should use a shared paletteBlendLocked.
  - id: new
    severity: Minor
    family: single-clock-source
    title: |
      Note.At stamped with wall-clock time.Now in session.go while the console draws with injectable c.now
    detail: |
      With a fake console clock, note ages come out negative, are clamped to 0, and the note looks permanently flashing; flash timing can't be driven from tests. Stamp At from the same clock the drawing uses.
  - id: new
    severity: Minor
    family: envelope-unstated-at-bound
    title: |
      Notes flash timer repaint ceiling at the 48-note bound (about 35 full repaints/s to every viewer) is not stated
    detail: |
      The plan states the per-note cost (about 11 repaints per 15s) but not the total at the 6x8 note limit. Write that ceiling into the plan, or snap flash phases to a coarser grid so repaints for several notes coincide.
```

---

## Re-review — 2026-10-11T15:56:37-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 447 — Broadcast sticky notes: viewers pin comments on the screen |
| repo | pair |
| issue file | workshop/issues/000447-broadcast-sticky-notes-viewers-pin-comments-on-the-screen.md |
| boundary | milestone M3 |
| milestone | M3 |
| window | 75f2e1ac3e41edd226abbe451a19b8365b64ff42..dede99b74396ddbf01d0c52dd008126121560b52 |
| command | sdlc milestone-close --issue 447 --milestone M3 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T15:56:37-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Read the diff and prior findings; now checking the pointer/timer code paths and running the tests (including revert checks for BR-14).

Note tests pass under `-race` (PTY failures are environmental). Now reverting each BR-14 fix to confirm the new test goes red.

Both reverts go red, so BR-14 is confirmed. Checking BR-16/BR-17 evidence and the plan.

BR-14 confirmed by revert. Now checking whether anything pins BR-16 (the note clock wiring).

Nothing pins BR-16. One last check: whether the glow's wide-character lead shift can reach the guarded controls.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

M3 delivers what the plan asks for. Each 💭 gets a chip in its author's color and a round glow that changes only backgrounds. Notes flash on arrival and on one shared 15s beat. `paletteTint` is now the only place marks and notes fade, and the hint bar is gone in favour of tooltips. The atlas and README were updated in the same range.

Of the four open findings, three are fixed with real evidence. **BR-14:** I reverted each fix on its own and the new test failed both times. **BR-15:** fixed by a shared helper. **BR-17:** the plan now states the ceiling, and a test pins it. **BR-16** is fixed in the code, but no test catches it. With the fix reverted, the whole broadcast and couchtty note suites still pass, so under the regression-evidence rule it stays open. It is Minor and cheap to fix. Nothing here blocks the boundary.

The verification was done in this checkout:
- `go test ./cmd/internal/broadcast/` passes.
- The couchtty note and timer tests pass three times under `-race`.
- All 60 node viewer tests pass.
- couchtty only built after I generated the gitignored runtime bundle assets. Its four PTY tests fail with `operation not permitted` (an environment limit, unrelated to this diff).
- After the revert experiments I restored the tree; `git status` shows only that generated bundle.

**1. Strengths**
- **Fixed repaint cost.** One shared beat (`notehalo.go:59` `beatPhase`) means the number of repaints doesn't grow with the number of notes. `TestNoteFlashRepaintsDontGrowWithNotes` (`notehalo_test.go:143`) pins this at the 48-note limit (ARCH-CONSTRAINTS).
- **One fade path.** `paletteTint` (`marks.go:98`) and `tintCell` (`marks.go:293`) are the single place a color fades and a cell is tinted. `Marks.tint` is now "age → t → paletteTint", and the unchanged mark tests pin the refactor (ARCH-DRY).
- **Drawing is pure and deterministic.** `DrawNotes` resolves overlapping glows by "stronger t, then lower color" and folds both halves of a wide character onto its lead. So the output doesn't depend on map order, and `TestDrawNotesOrderIndependentWithGlow` covers this with a 200-iteration property test that includes backgrounds.
- **The timer tests are honest.** `TestCollaborationTimersRearmAndTearDown` (`console_notes_test.go:196`) checks teardown immediately after the layer ends, not eventually. Removing the re-arm in the callback or the `Stop` in `forgetNotes` turns it red; I confirmed both.
- **The view page stays correct at stream end.** The `sawCaps` guard means a view page no longer receives made-up `caps` when the stream ends, and `connect.test.mjs` has a test for it.

**2. Critical findings**
None.

**3. Important findings**
None.

**4. Minor findings**
- **BR-16 (not addressed, `single-clock-source`, 2nd in the family).** No test catches it if the fix is removed. The class rule: any timestamp a component compares against its injectable clock must come from that clock, and one test must drive both stamp and draw through the fake clock. A cheap way to pin it: a console-fixture test with `testClock`. Set the clock an hour back, save a note, and assert it is not in its arrival flash a few seconds later on the fake clock. Today the stamp comes from the real clock, so its age reads as negative and the note stays at "full".
- **Plan drifts from the code on `paletteTint` (new family `plan-tracks-code`).** Core concepts (plan line 285) says `paletteTint(c, strength, …)` with "strength 1 is the full color", and gives glow strengths 0.5 and 0.25. The code takes `t`, where 0 is full color, and uses 0.5 and 0.75 (`notehalo.go:29-30`). The two describe the same colors, but someone reading the plan would call it the wrong way round. The atlas already uses `t` correctly.
- **The note style goes stale (new family `snapshot-captures-mutable-input`).** `pnotes.style` (BG and blend) is copied only when a notes snapshot is accepted (`console_notes.go:49,107`). If OSC 11 reports the background later, or the operator switches theme while notes are open, chips and glows keep blending into the old background until the next note change. Marks re-read the palette on every batch. The fix is to read `noteStyleLocked()` in `noteOverlay` rather than caching it. This needs a small lock-order check, because `c.mu` would then be taken on the Presenter goroutine.
- `NoteStyle.Hover` is never set yet. That is M4 scope, so it's fine, but it is surface with no consumer at M3.
- The notes re-arm subtest waits on real time: a 1.5s hold against a roughly 3s `waitFor`. It may flake under a loaded `-race` CI run. Driving it through `testClock` would make it deterministic.

**5. Test coverage notes**
- Glow geometry, ring strengths, staying off the guarded cells, flash timing, and `NoteNextChange` are each tested at exact boundaries.
- `TestDrawNotesBlankKeepsStyle` was rewritten to check the foreground, because the glow now sets the background. The test explains why, and that is the right call.
- The plan's Task 14 headless-Chrome check at 1200×800 and 390×844 isn't evidenced in the diff. The page-asset check for no `#hint`/`#controls` (`server_test.go:157`) covers the structural part.

**6. Architectural notes**
- **ARCH-DRY:** pass. `blendable` and `paletteTint` are shared. `scheduleNotes` and `refreshMarks` repeat the same "stop, nil, compute next, AfterFunc" pattern, and stop-and-nil appears 5 times. If M4 adds a third timer, pull out `rearm(&t, d, ok, fn)` and `stopTimer(&t)`.
- **ARCH-PURE:** pass. Glow, flash, and next-change logic are pure in `notehalo.go`; the console only arms the timer and repaints.
- **ARCH-PURPOSE:** pass. Chip, glow, arrival flash, 15s beat and tooltips are all delivered. Hover glow is properly left to M4.
- **ARCH-MOCK:** N/A. No new external dependency.
- **ARCH-CONSTRAINTS:** pass. The repaint ceiling is stated and tested.
- **ARCH-SECURE:** pass. `Note.At` is `json:"-"` and set by the server, and `Color` is still checked by `ParseNote` before it indexes `markPalette`.
- **ARCH-ORDER:** pass with notes. Every way the layer ends goes through `resyncNotes` or `forgetNotes`, and both stop the timer. A late callback reschedules from an empty board and arms nothing. The open gap is the real-time re-arm test (above).
- **ARCH-FUNERAL:** pass. There is one timer, owned by `pnotes`, and it is stopped on pointer-off, broadcast end, and an empty board. Nothing durable is created.

**7. Plan revision recommendations**
- Add a "## Revisions" entry: "2026-10-11 — M3 close review: `paletteTint` takes `t` (0 = full color, 1 = background), not strength. The glow rings are t 0.5 (near) and 0.75 (far). Core concepts updated to match the code."
- Task 13's Step 1 says to drive the flash timer with the console's test clock. Either do that, or revise the step to say the re-arm is checked against real time with a wait budget.

```findings
dispose:
  - id: BR-14
    disposition: addressed
    note: |
      TestCollaborationTimersRearmAndTearDown covers both timers on re-arm, pointer-off and broadcast end; reverting the AfterFunc re-arm or the forgetNotes Stop turns it red (verified).
  - id: BR-15
    disposition: addressed
    note: |
      blendable(Palette) at console_notes.go:73 is now the only site; applyPoints (console_pointer.go:180) and noteStyleLocked both call it, and a grep finds no other inline copy.
  - id: BR-16
    disposition: not-addressed
    note: |
      Config.Now is wired to c.now, but reverting both the wiring and the stamp leaves every broadcast and couchtty test green; no regression test drives stamp and draw through the fake clock.
  - id: BR-17
    disposition: addressed
    note: |
      The plan Flash bullet states the 48-note ceiling (about 35/s per-note) and the shared beat bounds it; TestNoteFlashRepaintsDontGrowWithNotes pins it.
findings:
  - id: new
    severity: Minor
    family: plan-tracks-code
    title: |
      Plan Core concepts describe paletteTint with a strength argument (1 = full) but the code takes t (0 = full)
    detail: |
      Plan line 285 and the glow strengths (0.5 and 0.25) are the inverse of notehalo.go:29-30 and marks.go:98 (t 0.5 and 0.75). The colors match but the stated API is inverted; add a Revisions entry and fix Core concepts. The atlas already says t.
  - id: new
    severity: Minor
    family: snapshot-captures-mutable-input
    title: |
      The notes' blend background is captured only when a snapshot is accepted, so a later OSC 11 report or theme change is ignored until the next note change
    detail: |
      console_notes.go:49,107 cache noteStyleLocked() into pnotes.style; marks re-read the palette on every batch. Read the style in noteOverlay at draw time (check the c.mu lock order on the Presenter goroutine).
```
