# Boundary Review — pair#447 (milestone M4)

| field | value |
|-------|-------|
| issue | 447 — Broadcast sticky notes: viewers pin comments on the screen |
| repo | pair |
| issue file | workshop/issues/000447-broadcast-sticky-notes-viewers-pin-comments-on-the-screen.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | 89a1d7b4f9e1f51febf14a9de217b609150272e7..9050ec292e52fc3e03accb86a2aa4f088e81c589 |
| command | sdlc milestone-close --issue 447 --milestone M4 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T16:22:35-07:00 |
| verdict | REWORK |

## Review

Reviewing the M4 notes-menu diff against the plan; now reading the plan's Chunk 4 and running the tests.

Possible bug: the hit-test layout persists while a private panel (the switcher) is shown, since `noteOverlay` skips private frames. Let me confirm with a quick throwaway test.

```verdict
verdict: REWORK
confidence: high
```

**Verdict: REWORK.** The M4 notes menu mostly does what the plan asks. The pure renderer is cleanly separated and well tested, the `💭N` span works as a click target, routing runs before the presenter and the child, and the lifecycle is cleared in all three places. One confirmed correctness bug blocks it. The menu remembers its click area from the last screen it was drawn on. If a private panel replaces that screen, the menu still claims those clicks. I reproduced this with a throwaway test in the checkout, since removed: open the notes menu, call `showMenu()` (the switcher panel, a private frame), then click where the menu used to be. A note the operator can't see was copied, "Note copied." appeared, and the panel never got the click. Also, the atlas and plan say the menu is drawn after the pointer marks; in the code the marks are drawn after it.

1. **Strengths**
   - `notes_menu.go` is a real pure core. `RenderNotesMenu`, `Contains`, `ItemAt`, `CloseAt` and `Draw` are tested with no IO. That covers placement at 80×24 and 40×10, the 3-line cap with `…`, "+k more", escape text drawn as plain characters, a wide character cut at the box edge, and the input frame not being changed.
   - The "never reaches the child" test (`console_notes_test.go:301`) is built properly. The child is first put into mouse-tracking mode (1003/1006), and the test checks that a click outside the menu still arrives. The lesson recorded in `workshop/lessons.md` is a good one.
   - Lock order is sound. `routeNotesMenu` releases `pnotes.mu` before calling `refreshNotes`, `toggleNotesMenu` or `copyLink`, so nothing waits on the presenter while holding the lock. The race-detector run of the notes, status-row, collaboration and pointer tests passed 3×.
   - `settleMenuLocked` is the single place that enforces "closed with no notes, no hover on a note that's gone", and both `applyNotes` and `resyncNotes` call it.
   - The `Notes` span in `RenderedStatusRow` is only set when the count was actually drawn (`used > start`), so a count cut off a narrow row isn't clickable.

2. **Critical findings**
   - `cmd/internal/couchtty/console_notes.go:178-193` together with `routeNotesMenu`: `noteOverlay` returns early on a private frame but leaves `pnotes.menu` and `menuOpen` set. `routeNotesMenu` (`terminal_input.go:131`) then hit-tests against a box that isn't on screen. It swallows presses, motion and wheel in that area before the panel sees them, and copies notes the operator can't see. **Fix sketch:** in `noteOverlay`'s non-public branch, take `pnotes.mu` and reset `pnotes.menu = NotesMenu{}`. The menu stays logically open and its layout comes back on the next public frame. Add a console test: open the menu, `showMenu()`, click in the old box, assert no copy and that the panel or child received the click. **ARCH-ORDER:** `menuOpen`, `menu`, `pressed` and `Hover` are loose fields. The state "open, but laid out on a frame no longer shown" is reachable and has no defined meaning. Treat it as `closed | open(layout of the visible frame)`.

3. **Important findings**
   - `console_pointer.go:41`: `screenOverlay = markOverlay(noteOverlay(f))`, so marks are drawn after the menu and tint its cells. `atlas/broadcast.md` ("drawn last in `screenOverlay`") and the plan ("notes, then marks, then the menu") both say otherwise. **Fix:** split the menu drawing into its own overlay step that runs after `markOverlay`, and add a test that a mark inside the box leaves the menu's cells untinted.
     - This is the 2nd finding in family `comment-claims-unenforced-invariant`. The rule that covers both: a draw-order or ownership claim in the atlas or plan needs a test that would fail if the order were wrong; otherwise the claim must not be made.

4. **Minor findings**
   - **Plan doesn't match the code** (2nd in `plan-tracks-code`). The plan says `RenderNotesMenu(notes, cols, rows, hover)`; the code takes `hover` in `Draw` instead. The plan says the hovered item is drawn in reverse video; the code uses a 239 background. The rule: a commit that departs from a Core concepts signature or visual spec updates that row in the same commit.
   - **Save-and-restore of `Hover` repeated three times** (3rd in `shared-helper-not-reused`). `applyNotes`, `resyncNotes` and `restyleNotes` each save `Hover`, replace `style`, then put `Hover` back. The rule: state that belongs to the view must not live inside a value that gets replaced wholesale. Move hover to `pnotes.hover` and add it to the style at draw time; that removes all three sites.
   - **Broadcast-end path untested** (3rd in `untested-cleanup-path`). The menu closing when the broadcast ends (`forgetNotes`) has no test; only resolving the last note and pointer-off are covered. The rule: every teardown that resets menu state is run as one row of a single table of teardown events (resolve-last, pointer-off, broadcast-end) in one test.
   - **Drag passing over the menu.** Motion while dragging across the box is swallowed, so a presenter drag started outside the menu misses the motion events over it.
   - **Palette wiring untested.** `TestNotesTakeANewPalette` calls `restyleNotes` directly, so the wiring in `terminal_input.go:66` is not tested.
   - **Weak span test.** `TestStatusRowNotesSpan` doesn't check that the pointer and remote spans are unchanged, which Task 17 lists.

5. **Test coverage notes**
   - The full `couchtty` run had one failure, `TestCouchProductionSoak`. Starting the pseudo-terminal child failed with "operation not permitted", and that test's cleanup then reported a data race in `vt.Emulator.Close`. Nothing in that trace touches M4 code, so I treat it as a problem with this environment, not this change.
   - `artifactpath` also failed, on stale entries in `known-artifact-violations.txt`, because the generated runtime bundle was missing from the checkout. Not caused by M4.
   - The console tests only check that `Hover` is set. The "💭 held at full glow" behaviour is covered only by `notehalo_test.go:185`.

6. **Architectural notes**
   - **ARCH-DRY:** flagged as Minor (the repeated save-and-restore of `Hover`).
   - **ARCH-PURE:** passes; the renderer is pure and the console is a thin shell around it.
   - **ARCH-PURPOSE:** the Done-when items are delivered, apart from the Critical bug.
   - **ARCH-MOCK:** N/A. There's no external dependency; OSC 52 goes through the presenter.
   - **ARCH-CONSTRAINTS:** passes. The layout is rebuilt each frame for at most 48 notes, and motion only repaints when the hover actually changes.
   - **ARCH-SECURE:** passes. Note text from viewers goes through `rowtext.Sanitize`, and the escape test pins that. The copy goes out base64-encoded in OSC 52.
   - **ARCH-ORDER:** flagged; this is the Critical finding.
   - **ARCH-FUNERAL:** passes. The feature creates nothing durable; menu state lives in memory and `forgetNotes` clears it.

7. **Plan revisions** — add these entries under `## Revisions`:
   - "M4: the menu is drawn after marks; its hit-test layout exists only while a public frame shows it."
   - "M4: `RenderNotesMenu(notes, cols, rows)`; hover is passed to `Draw` and highlighted with a background color, not reverse video."

```findings
findings:
  - id: new
    severity: Critical
    family: hit-region-tracks-drawn-frame
    title: |
      The notes menu's click area outlives its drawing: under a private panel it swallows clicks and copies a note nobody can see
    detail: |
      noteOverlay returns early on FramePrivate but leaves pnotes.menu and menuOpen set, so routeNotesMenu (terminal_input.go:131) keeps hit-testing the old box. Reproduced: open the menu, showMenu(), click in the old box: the note is copied, "Note copied." appears, and the panel gets nothing. Fix: reset pnotes.menu on non-public frames (the menu stays open and the layout returns on the next public frame), and add a regression test. ARCH-ORDER: replace the loose open/layout fields with closed | open(layout of the visible frame).
  - id: new
    severity: Important
    family: comment-claims-unenforced-invariant
    title: |
      Marks are drawn over the notes menu, though the atlas and plan say the menu is drawn last
    detail: |
      console_pointer.go:41 runs markOverlay after noteOverlay, and noteOverlay draws the menu, so marks tint the menu's cells. 2nd in this family. Rule: a draw-order claim in docs needs a test that fails if the order is wrong. Move the menu into its own overlay after markOverlay and test that a mark inside the box leaves it untinted.
  - id: new
    severity: Minor
    family: plan-tracks-code
    title: |
      Plan Chunk 4 gives a RenderNotesMenu signature with hover and a reverse-video hover; the code takes hover in Draw and uses a background color
    detail: |
      2nd in this family. Rule: a commit that departs from a Core concepts signature or visual spec updates that row in the same commit.
  - id: new
    severity: Minor
    family: shared-helper-not-reused
    title: |
      Saving and restoring Hover around a style replacement is repeated in applyNotes, resyncNotes and restyleNotes
    detail: |
      3rd in this family. Rule: view state must not live inside a value that is replaced wholesale. Keep hover in pnotes.hover and add it to the style at draw time, which removes all three sites.
  - id: new
    severity: Minor
    family: untested-cleanup-path
    title: |
      No test that the notes menu closes when the broadcast ends (forgetNotes)
    detail: |
      3rd in this family. Rule: every teardown that resets menu state (resolve-last, pointer-off, broadcast-end) is one row of a single table-driven lifecycle test.
```

---

## Re-review — 2026-10-11T16:29:56-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 447 — Broadcast sticky notes: viewers pin comments on the screen |
| repo | pair |
| issue file | workshop/issues/000447-broadcast-sticky-notes-viewers-pin-comments-on-the-screen.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | 89a1d7b4f9e1f51febf14a9de217b609150272e7..d55a23857c233ff13a07b1645ee7a818a03d707a |
| command | sdlc milestone-close --issue 447 --milestone M4 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T16:29:56-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Checking the prior findings now: I'm running the tests and then reverting each claimed fix to confirm its test goes red.

All five prior fixes went red when reverted. I'm now checking whether the menu's hover works when the child app doesn't track the mouse.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All five prior findings were checked by reverting each fix in this checkout. Four are fixed: with BR-20, BR-21 or BR-24's fix taken out, its regression test fails, and BR-23 is a structural change with no hover bookkeeping left to revert. BR-22 is only partly fixed: the plan's Core concepts row was corrected, but the Task 16 checklist in the same chunk still describes the old design. The menu code itself is sound. `RenderNotesMenu` and `Draw` are pure and tested without IO; the hit-test layout is now only the layout of the frame on screen; and the menu has its own overlay step after the marks. One Important gap remains: the hover glow. The issue's Done-when lists it as tested, but if the console stops passing `pnotes.hover` into the draw style, every notes, menu, pointer and mark test still passes. Shipping should wait for a test that pins it, not for a code change.

**Test environment:** the `couchtty` package needed `cmd/internal/runtimebundle/assets/runtime` generated first (`go run ./cmd/internal/runtimebundle/generatecmd`). After that, `broadcast` and `artifactpath` pass. `couchtty` passes apart from four PTY and `/tmp` tests that fail here with "operation not permitted", which is this environment, not the diff. The M4 tests pass under `-race`.

### 1. Strengths
- **Pure core:** `notes_menu.go` is a pure layout (`RenderNotesMenu`) plus a pure `Draw`. Tests cover placement, wrapping and the 3-line cap, "+k more", that the status row is never covered, that escape characters stay literal, and that a wide character cut at the box edge still gives a valid frame. None of these tests need IO (ARCH-PURE).
- **Hit-testing follows the drawn frame:** `menuOverlay` (`console_notes.go:190`) records the layout only from a public frame it actually draws, and clears it on any other. Because item IDs are stable, a click between a note change and the next frame copies the note that was on screen.
- **Routing tests can actually fail:** the child is fed tracking mode 1003 before the test asserts that nothing reaches it, and a click outside the menu checks that clicks still do arrive. The lesson in `workshop/lessons.md` records why.
- **One lifecycle table:** `TestNotesMenuLifecycle` covers resolving the last note, pointer-off and broadcast end in one table.
- **Hover is separate from style:** `drawStyleLocked` builds the style at draw time. This removed all three places that saved and restored hover around a style replacement.

### 2. Critical findings
None.

### 3. Important findings
- **The hover glow, and the console's hover highlight, are not pinned by any test.**
  - At `console_notes.go:204`, changing `style.Now, style.Hover = c.now(), c.pnotes.hover` to `style.Now = c.now()` leaves every notes, menu, pointer and mark test green. Yet the issue's Done-when says "hovering an item highlights it and makes its 💭 glow (tests)".
  - `TestNotesMenuOpenHoverCopy` only checks that `pnotes.hover != 0`, not what is drawn. The pure `notehalo_test.go:185` pins `DrawNotes` with `Hover` set, but not the console passing it in.
  - **This is the 3rd finding in family `comment-claims-unenforced-invariant`.** The rule that covers all three: *every behavior that the Done-when, atlas or plan claims for this milestone has a console-level test that goes red when that behavior's wiring is removed.*
  - Fix: go through the M4 Done-when clause by clause (opens with colors, hover highlight, hover glow, copy, close by `💭N` and by `×`, the three teardowns, no child reach) and map each clause to a test.
  - The two unmapped today are the glow and the highlight. Add one test that hovers an item at a time off the flash beat, then asserts two things: the 💭 chip and glow cells match `DrawNotes` with `Hover` set, and the item's rows use `notesMenuHover`.

### 4. Minor findings
- **BR-22 is partly unfixed (plan-tracks-code):**
  - Plan Task 16 still lists "Hover in reverse video".
  - The `RenderNotesMenu` bullet still says the function "returns the box's cells", but `NotesMenu` holds lines.
- **A press inside the menu followed by a drag outside it** passes motion events with the button held to the presenter or child, which never saw the press (`routeNotesMenu` returns false outside the box even while `pressed` is set).
- **Sleep-based negative check:** `TestNotesMenuTakesNoClickUnderAPanel` uses `time.Sleep(100ms)` before asserting no copy happened. It is gated on the layout being cleared first, so this is acceptable.

### 5. Test coverage notes
Each claimed fix was reverted and its test re-run:

| Finding | Fix reverted | Test that went red |
|---|---|---|
| BR-20 | layout no longer cleared on private frames | `TestNotesMenuTakesNoClickUnderAPanel` |
| BR-21 | menu and mark overlays swapped | `TestNotesMenuDrawnOverMarks` |
| BR-24 | menu reset dropped from `forgetNotes` | `TestNotesMenuLifecycle/broadcast_end` |

The status-row span is pinned by `TestStatusRowNotesSpan`. The only gap is the hover glow and highlight wiring above.

### 6. Architectural notes
- **ARCH-DRY: pass.** `NoteChipColor` reuses `paletteTint`, and copying reuses `copyLink`.
- **ARCH-PURE: pass.** The layout and drawing are pure; `routeNotesMenu` is a thin shell that does the copy and the notice.
- **ARCH-PURPOSE: pass, apart from the Important finding.** All M4 clauses are delivered; the glow is implemented but its test is missing.
- **ARCH-MOCK: N/A.** No new external dependency; copying goes through the presenter seam that already exists.
- **ARCH-CONSTRAINTS: pass.**
  - Couch always asks the terminal for any-motion reports (`terminal.AnyMotion`), so hover works whether or not the child tracks the mouse.
  - Motion only triggers a repaint when the hovered item changes.
  - The layout is recomputed on each frame while the menu is open, which is at most 48 short wraps.
- **ARCH-SECURE: pass.** Note text, which comes from viewers, goes through `rowtext.Sanitize`, and a test asserts no escape reaches a cell. Copying only happens when the operator clicks.
- **ARCH-ORDER: pass, with a note.**
  - The state is still four fields (`menuOpen`, `menu`, `hover`, `pressed`) rather than the `closed | open(layout)` type BR-20 suggested.
  - Every reachable combination I traced is legal: hover is cleared on close, on settle and in `forgetNotes`; the layout is cleared on any frame that doesn't show the menu.
  - If the menu grows (keyboard navigation, resolving notes from the terminal), collapse these fields into a tagged state with a pure `(state, mouse event) -> (state, effect)` function. That would also fix the drag-out leak.
- **ARCH-FUNERAL: pass.** Nothing durable is created; all menu state is in memory and reset in `forgetNotes`.

### 7. Plan revision recommendations
- Task 16: change "Hover in reverse video" to "hovered item's rows on `notesMenuHover`".
- Chunk 4 `RenderNotesMenu` bullet: change "returns the box's cells" to "returns the box's lines and placement".
- Add a `## Revisions` entry: "M4 close review: a hover-glow and highlight test at console level; every M4 Done-when clause mapped to a test that fails without its wiring."

```findings
dispose:
  - id: BR-20
    disposition: addressed
    note: |
      menuOverlay clears the layout on non-public frames; TestNotesMenuTakesNoClickUnderAPanel goes red with the clearing reverted. The tagged-state refactor was not adopted, but every reachable field combination is legal.
  - id: BR-21
    disposition: addressed
    note: |
      menuOverlay runs after markOverlay (console_pointer.go:42); TestNotesMenuDrawnOverMarks goes red when the order is swapped.
  - id: BR-22
    disposition: not-addressed
    note: |
      The Core concepts row was fixed, but Task 16 still says "Hover in reverse video" and the bullet still says it "returns the box's cells".
  - id: BR-23
    disposition: addressed
    note: |
      Hover lives in pnotes.hover and is added to the style at draw time by drawStyleLocked; no hover save or restore is left in applyNotes, resyncNotes or restyleNotes.
  - id: BR-24
    disposition: addressed
    note: |
      TestNotesMenuLifecycle is one table covering resolve-last, pointer-off and broadcast end; the broadcast_end row goes red when the reset in forgetNotes is removed.
findings:
  - id: new
    severity: Important
    family: comment-claims-unenforced-invariant
    title: |
      The Done-when says the hover glow and highlight are tested, but removing their console wiring leaves every test green
    detail: |
      Dropping style.Hover in drawStyleLocked (console_notes.go:204) passes all notes, menu, pointer and mark tests; TestNotesMenuOpenHoverCopy only checks pnotes.hover. 3rd in this family. Rule: every behavior a milestone's Done-when, atlas or plan claims has a console-level test that goes red when its wiring is removed. Map each M4 Done-when clause to a test and add the missing one: hover an item, then assert the note's chip and glow cells match DrawNotes with Hover set (at a time off the flash beat) and the item rows use notesMenuHover.
  - id: new
    severity: Minor
    family: hit-region-tracks-drawn-frame
    title: |
      A press inside the menu followed by a drag outside it sends button-held motion to the presenter or child, which never saw the press
    detail: |
      routeNotesMenu returns false for motion outside the box even while pnotes.pressed is set. 2nd in this family. Rule: a gesture that starts in a hit region belongs to that region until its release. Take all motion while pressed is set.
```

---

## Re-review — 2026-10-11T16:35:46-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 447 — Broadcast sticky notes: viewers pin comments on the screen |
| repo | pair |
| issue file | workshop/issues/000447-broadcast-sticky-notes-viewers-pin-comments-on-the-screen.md |
| boundary | milestone M4 |
| milestone | M4 |
| window | 89a1d7b4f9e1f51febf14a9de217b609150272e7..826bd20866b4eec74189ab3c63377679ccdc7484 |
| command | sdlc milestone-close --issue 447 --milestone M4 |
| reviewer | claude |
| isolation | checkout |
| timestamp | 2026-10-11T16:35:46-07:00 |
| verdict | SHIP |

## Review

Tests run; checking per-package results, then mutation-checking the claimed fixes.

The checkout lacks generated (gitignored) embed assets; I'll add a throwaway stub in this disposable checkout so the package compiles.

Notes tests pass. Now reverting each claimed fix to check the regression tests go red.

All three claimed fixes go red when reverted. Now checking gesture ownership in the other direction: a drag that starts outside the menu and crosses it.

Wrapping up: checking the issue's M4 Done-when against tests, then writing the verdict.

```verdict
verdict: SHIP
confidence: high
```

All three open findings (BR-22, BR-25, BR-26) are fixed. I checked the two behavior fixes by reverting each one in the checkout and confirming its new test fails. Each clause of the M4 Done-when now has a console-level test, and the README, atlas and plan describe the code as it is. One new Minor finding: the gesture-ownership rule from BR-26 was applied only to presses that start inside the menu, not to presses that start outside it. It does not block shipping.

**How I verified:** The checkout has no `runtimebundle/assets` folder; it is generated and gitignored. Without it `couchtty` won't compile, so I added a throwaway stub and deleted it afterwards. With the stub, `go test -run 'Notes|NotesMenu|StatusRow|Collaboration'` passes and `broadcast` passes. The other `couchtty` failures in this sandbox are PTY and `/tmp` permission errors (`TestCouchPressure*`, `TestCouchProductionSoak`, `TestNotificationPTYConformance`). `artifactpath` fails on the same missing assets. These failures have nothing to do with this diff.

1. **Strengths**
   - `console_notes.go:186-198`: the menu's click area is whatever was last drawn on a public frame. On any other frame it is cleared, so a menu that isn't on screen can't take a click. `TestNotesMenuTakesNoClickUnderAPanel` covers this.
   - `notes_menu.go`: `RenderNotesMenu` and `Draw`/`ItemAt`/`CloseAt` are pure functions with tests that need no IO. The console code around them stays small.
   - `TestNotesMenuHoverGlowsItsNote` compares a hovered note against an unhovered one at a moment off the shared flash beat. A full-strength glow can only come from the hover, so the test checks the wiring itself.
   - `TestNotesMenuLifecycle` covers all three ways the menu must close (last note resolved, pointing off, broadcast end) and checks that the layout and hover are cleared too.
   - `broadcast.NoteChipColor` reuses `paletteTint`, so the menu chip and the note chip are always the same color.

2. **Critical findings:** none.

3. **Important findings:** none.

4. **Minor findings**
   - **Drags that start outside the menu get cut when they cross it.** At `console_notes.go:255-275`, motion with a button held over the menu is taken as hover whenever the press happened elsewhere. A child's drag-select that passes over the box loses its motion events there. It also bypasses the presenter's comment at `terminal_input.go:134` that release and motion "always visit the presenter".
     - **This is the 3rd finding in family `hit-region-tracks-drawn-frame`.** The rule covering all three: the press decides who owns a gesture, and that owner keeps it until the release. Fix the rule, not this case: turn `pnotes.pressed` into an owner value (none / menu / elsewhere). Set it to "elsewhere" when a press falls outside the box (`button < 64`). While it is "elsewhere", pass motion and release through. Add the matching case to `TestNotesMenuDragStaysTheMenus`.
   - `copyLink` now copies note text as well as links; a more general name would fit (`console_pointer.go:103`).
   - `pnotes.hover` survives while a private panel hides the menu, so the note keeps its full glow until the next mouse motion.

5. **Test coverage notes:** Each Done-when clause maps to a test: open, chip color, hover highlight and glow, copy plus notice, close by `💭N` or `×`, the lifecycle closes, and nothing reaching the child. Reverting each fix turned its test red:
   - dropping `Hover` from `drawStyleLocked` fails `TestNotesMenuHoverGlowsItsNote`
   - drawing the hovered row on the normal background fails that test and `TestNotesMenuDrawAndHitTests`
   - removing the `motion && pressed` guard fails `TestNotesMenuDragStaysTheMenus`

6. **Architectural notes**
   - ARCH-DRY: pass. The one duplication is the menu's own press tracking next to the presenter's gesture state (see ARCH-ORDER).
   - ARCH-PURE: pass. Layout and drawing are pure; the console does the IO.
   - ARCH-PURPOSE: pass on the M4 scope. The gesture rule was applied to only one side, which is the Minor above.
   - ARCH-MOCK: N/A. There is no new external dependency; the copy goes through the existing presenter and OSC 52.
   - ARCH-CONSTRAINTS: pass. The menu is capped at 48 columns and 3 lines per item. Hover repaints only when the hover changes.
   - ARCH-SECURE: pass. Note text comes from viewers and is cleaned again with `rowtext.Sanitize` before it reaches the operator's terminal, and tested literally. The clipboard path goes through the presenter's copy.
   - ARCH-ORDER: flag, Minor. The menu's state is four separate fields (`menuOpen`, `menu`, `hover`, `pressed`) rather than one enumerated state. They are settled in several places (`settleMenuLocked`, `toggleNotesMenu`, `forgetNotes`, `menuOverlay`), and press tracking repeats state the presenter already keeps. If this widget grows, collapse them into one state type.
   - ARCH-FUNERAL: pass. The menu creates nothing durable; its state is in memory and cleared in `forgetNotes`.

7. **Plan revision recommendations:** none needed. The plan's Core concepts (`RenderNotesMenu(notes, cols, rows)`, `Draw(f, hover, style)`, the `notesMenuHover` background) match the code. Optionally, note the gesture-owner rule in the routing bullet when the Minor is fixed.

```findings
dispose:
  - id: BR-22
    disposition: addressed
    note: |
      Plan Core concepts now state RenderNotesMenu(notes, cols, rows) plus NotesMenu.Draw(f, hover, style) and the notesMenuHover background, matching notes_menu.go.
  - id: BR-25
    disposition: addressed
    note: |
      TestNotesMenuHoverGlowsItsNote drives hover via console mouse input; reverting style.Hover in drawStyleLocked or the hover bg in Draw turns it red (verified).
  - id: BR-26
    disposition: addressed
    note: |
      motion-while-pressed guard at console_notes.go:250; removing it fails TestNotesMenuDragStaysTheMenus (verified).
findings:
  - id: new
    severity: Minor
    family: hit-region-tracks-drawn-frame
    title: |
      A drag that starts outside the notes menu loses its motion while crossing the box
    detail: |
      3rd in this family. routeNotesMenu takes button-held motion over the box as hover whenever the press was elsewhere, so a child's drag-select passing over the menu loses those events. Rule: the press decides who owns a gesture, and that owner keeps it until the release. Make pnotes.pressed an owner value (none/menu/elsewhere), set elsewhere on a press outside the box, pass motion and release through while it is elsewhere, and extend TestNotesMenuDragStaysTheMenus.
```
