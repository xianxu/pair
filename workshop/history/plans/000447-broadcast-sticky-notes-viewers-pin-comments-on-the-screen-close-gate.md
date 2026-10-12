---
gate: boundary-review
issue: 447
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T14:10:32-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Task 3 Step 3 says to blank cells with uv.EmptyCell; Core concepts says EmptyCell drops the background and blanking must keep style
          detail: |-
            2nd finding in this family. Rule: a task step names what to build and points to its Core concepts entry for how, without restating the mechanism. Follow Core concepts (set content " " and width 1, keep the style) and drop the uv.EmptyCell wording in Task 3.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: plan-self-consistency
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-10-11T14:10:32-07:00"
      dispatched: "2026-10-11T14:00:03-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: "README update appears missing for the \U0001F4ADN tab-bar count and notes cleared on pointing off"
          detail: "README.md:876-891 documents the live status row as LIVE ⏸ \U0001F446 \U0001F47D; M1 adds an operator-visible \U0001F4ADN after \U0001F47D (reserve.go:227-230) and deletes all notes when pointing turns off. Add a sentence or two to the Remote pointer paragraph."
          family: docs-gate-user-surface
          round: 2
        - id: BR-3
          severity: Minor
          title: ParsePointBatch duplicates the strict-decode logic now in decodeStrict (ARCH-DRY)
          detail: point.go:52-58 repeats DisallowUnknownFields plus the trailing-token check that notes.go:140 decodeStrict provides; have ParsePointBatch call decodeStrict.
          family: shared-helper-not-reused
          round: 2
        - id: BR-4
          severity: Minor
          title: forgetNotes at broadcast end is not exercised with notes held
          detail: TestNotesIgnoredAfterBroadcastEndsOrFromAnotherSession stops a broadcast that holds no notes, so plan Task 6's count-is-0-after-stop case with notes present is not tested.
          family: untested-cleanup-path
          round: 2
        - id: BR-5
          severity: Minor
          title: Per-color note limit uses a client-supplied color, so one viewer can use all six colors
          detail: ParseNote accepts any color 0-5 that is not tied to the stream's claimed color; the 48-note total cap and the rate limit still hold, and the setting is collaborative by design.
          family: client-asserted-identity
          round: 2
        - id: BR-6
          severity: Minor
          title: drawnNotes comment says empty unless pointing is on, but the function does not check
          detail: This holds only because every pointer-phase change resyncs the snapshot (console_notes.go); reword the comment to say that.
          family: comment-claims-unenforced-invariant
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 3
      timestamp: "2026-10-11T14:22:15-07:00"
      dispatched: "2026-10-11T14:12:51-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan line 147 still says "f.Clone() and uv.EmptyCell for blanks"; the code is correct (notes.go:205 blank keeps style), only the plan step is stale.
          round: 3
        - id: BR-2
          disposition: addressed
          note: "README.md:894-901 adds the Sticky notes paragraph covering the \U0001F4ADN count after the controls and that pointing off deletes notes; matches reserve.go:227-230 and pointer.go:103."
          round: 3
        - id: BR-3
          disposition: addressed
          note: point.go:50 now calls decodeStrict; the broadcast point tests pass unchanged.
          round: 3
        - id: BR-4
          disposition: addressed
          note: TestNotesForgottenWhenBroadcastEnds (console_notes_test.go:113) fails when the forgetNotes call in resetPointer is removed ("console still holds 1 notes (v1)"), so the fix is reachable.
          round: 3
        - id: BR-5
          disposition: withdrawn
          note: 'Consistent with the spec''s trust model: the pointer link is a shared capability, point colors (#446) are client-asserted too, and the 48-note total cap plus rate limit are the real bound.'
          round: 3
        - id: BR-6
          disposition: addressed
          note: console_notes.go:78-80 comment now says emptiness relies on the resync at each pointer-phase change; applyNotes also gates on pointer==pointerOn.
          round: 3
      findings:
        - id: BR-7
          severity: Minor
          title: README describes viewer-page note actions that only land in M2
          detail: "README.md:894-901 says pointer-link viewers can pin notes and anyone can tap a \U0001F4AD to read it, but M1 ships no viewer UI (notes are curl-only until M2 Tasks 8-10). This is the 2nd finding in this family; the rule: README changes describe only the surface present at the boundary's head, so tag M2-only behavior as upcoming or add it with M2."
          family: docs-gate-user-surface
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: checkout
      reviewed: 1c3a0b160976359ea6565092c6ff5dd8017f3feb
      blocked: false
    - "n": 4
      timestamp: "2026-10-11T15:12:41-07:00"
      dispatched: "2026-10-11T15:09:56-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: Plan line 147 (Task 3 Step 3) still reads "using f.Clone() and uv.EmptyCell for blanks"; code blank() at notes.go:208 keeps style, so only the plan text remains wrong.
          round: 4
      findings:
        - id: BR-8
          severity: Important
          title: Viewer keeps notes, pointing-on, mode button and open dialog after event end (viewer.js:200-206, 380-395)
          detail: deps.reset only resets the terminal; server.go selects end/caps/notes in random order, so stale notes stay tappable (with Resolve) on the blank ended screen and comment mode still drafts. Reset all stream-derived state (notes list, setOn(false)) on the terminal event and add a test. ARCH-ORDER.
          family: stream-end-resets-derived-state
          round: 4
        - id: BR-9
          severity: Minor
          title: noteDialog applies a save/resolve response to whatever dialog is open now, not the one that sent it (notes.js:74-81)
          detail: 'Cancel during an in-flight save and open a new draft: the old 204 closes the new draft (text lost), or an old 409 shows on it. Capture a token per request and drop mismatched completions. ARCH-ORDER.'
          family: late-completion-identity
          round: 4
        - id: BR-10
          severity: Minor
          title: "README Sticky notes paragraph never says how a viewer places a note (the \U0001F446/\U0001F4AD mode button)"
          detail: '3rd finding in this family. Rule: the README feature paragraph names every control a person uses for the feature, and each milestone that adds or removes a control updates it; fold into M3''s README task (hint bar removal, tooltips).'
          family: docs-gate-user-surface
          round: 4
        - id: BR-11
          severity: Minor
          title: Escape does not close a viewed note; aria-modal dialog has no focus move or trap in view mode
          detail: 'The keydown listener is on #note, and only a draft moves focus into it (notes.js:89, draft focuses input).'
          family: dialog-keyboard-reach
          round: 4
        - id: BR-12
          severity: Minor
          title: Plan Task 8 gives noteAt(notes, col, row, cols, rows, guard); code takes noteAt(list, col, row, cols, rows) with list.guard
          detail: '2nd in family. Rule: task steps cite Core concepts for signatures rather than restating them, which also covers BR-1; update Task 8 to match.'
          family: plan-self-consistency
          round: 4
      boundary: M2
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 5
      timestamp: "2026-10-11T15:15:54-07:00"
      dispatched: "2026-10-11T15:14:12-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: plan.md:147 Task 3 Step 3 still reads "using f.Clone() and uv.EmptyCell for blanks", contradicting Core concepts line 37.
          round: 5
        - id: BR-8
          disposition: addressed
          note: viewer.js:208-209 delivers caps-off and an empty notes list on end; connect.test.mjs BR-8 test fails with the pre-fix viewer.js (verified by revert). New view-link side effect raised separately.
          round: 5
        - id: BR-9
          disposition: addressed
          note: notes.js:62-91 generation token; notes.test.mjs "an answer applies only to the dialog that sent it" fails on revert (verified).
          round: 5
        - id: BR-10
          disposition: addressed
          note: "README.md:895-896 now names the \U0001F446/\U0001F4AD mode button under the swatches."
          round: 5
        - id: BR-11
          disposition: addressed
          note: notes.js:141 focuses Close on view; test fails on revert. No focus trap yet, acceptable at Minor.
          round: 5
        - id: BR-12
          disposition: addressed
          note: plan.md:232 Task 8 now cites Core concepts for the noteAt signature instead of restating it.
          round: 5
      findings:
        - id: BR-13
          severity: Minor
          title: End sends a made-up caps-off to view links, which never get caps, so they show the "Pointing is off" hint band
          detail: 'viewer.js:208 calls deps.caps({pointer:false}) on every page. server.go:325-340 sends caps only to pointer pages, so on a view link setOn(false) un-hides #hint and band() lifts the stage after the end. 2nd in family. Rule: an end puts stream-derived state back to the page variant''s join state and never replays events that variant does not receive. Fix: only send caps-off when a caps event was seen (or use one ended() dep), and add a view-link end test.'
          family: stream-end-resets-derived-state
          round: 5
      boundary: M2
      recipe: milestone-review
      isolation: checkout
      reviewed: 706a32676d10c63478c5fdae9f61984e62812423
      blocked: false
    - "n": 6
      timestamp: "2026-10-11T15:47:45-07:00"
      dispatched: "2026-10-11T15:38:20-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Plan Task 3 Step 3 now reads "Implement DrawNotes per Core concepts" (diff 75f2e1ac..f7a574e5); the uv.EmptyCell wording appears only in Core concepts' rationale (plan line 37).
          round: 6
      findings:
        - id: BR-14
          severity: Important
          title: 'Task 13 tests missing: flash-timer re-arm after firing and notes timer cleared on broadcast stop'
          detail: '2nd finding in this family. Rule: every timer the console arms needs a test for each way it is torn down plus a test that it re-arms after firing; today that is pmarks.timer and pnotes.timer, each on pointer-off and on broadcast end (resetPointer). Only notes-on-pointer-off is tested (console_notes_test.go:177). Dropping the scheduleNotes call from the AfterFunc callback, or the timer stop from forgetNotes, keeps the suite green. Add one table-driven test over both timers and both teardown paths, plus a re-arm check through the test clock or a delay seam.'
          family: untested-cleanup-path
          round: 6
        - id: BR-15
          severity: Minor
          title: Blend predicate Known && TrueColor && !NoColor duplicated in noteStyleLocked and applyPoints
          detail: '2nd finding in this family. Rule: when a new helper encodes a predicate that already exists inline, the old site calls the new helper in the same change. A grep finds exactly two sites: console_notes.go:69 and console_pointer.go:180. applyPoints should take its blend from noteStyleLocked, or both should use a shared paletteBlendLocked.'
          family: shared-helper-not-reused
          round: 6
        - id: BR-16
          severity: Minor
          title: Note.At stamped with wall-clock time.Now in session.go while the console draws with injectable c.now
          detail: With a fake console clock, note ages come out negative, are clamped to 0, and the note looks permanently flashing; flash timing can't be driven from tests. Stamp At from the same clock the drawing uses.
          family: single-clock-source
          round: 6
        - id: BR-17
          severity: Minor
          title: Notes flash timer repaint ceiling at the 48-note bound (about 35 full repaints/s to every viewer) is not stated
          detail: The plan states the per-note cost (about 11 repaints per 15s) but not the total at the 6x8 note limit. Write that ceiling into the plan, or snap flash phases to a coarser grid so repaints for several notes coincide.
          family: envelope-unstated-at-bound
          round: 6
      boundary: M3
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 7
      timestamp: "2026-10-11T15:56:38-07:00"
      dispatched: "2026-10-11T15:51:47-07:00"
      agent: claude
      dispose:
        - id: BR-14
          disposition: addressed
          note: TestCollaborationTimersRearmAndTearDown covers both timers on re-arm, pointer-off and broadcast end; reverting the AfterFunc re-arm or the forgetNotes Stop turns it red (verified).
          round: 7
        - id: BR-15
          disposition: addressed
          note: blendable(Palette) at console_notes.go:73 is now the only site; applyPoints (console_pointer.go:180) and noteStyleLocked both call it, and a grep finds no other inline copy.
          round: 7
        - id: BR-16
          disposition: not-addressed
          note: Config.Now is wired to c.now, but reverting both the wiring and the stamp leaves every broadcast and couchtty test green; no regression test drives stamp and draw through the fake clock.
          round: 7
        - id: BR-17
          disposition: addressed
          note: The plan Flash bullet states the 48-note ceiling (about 35/s per-note) and the shared beat bounds it; TestNoteFlashRepaintsDontGrowWithNotes pins it.
          round: 7
      findings:
        - id: BR-18
          severity: Minor
          title: Plan Core concepts describe paletteTint with a strength argument (1 = full) but the code takes t (0 = full)
          detail: Plan line 285 and the glow strengths (0.5 and 0.25) are the inverse of notehalo.go:29-30 and marks.go:98 (t 0.5 and 0.75). The colors match but the stated API is inverted; add a Revisions entry and fix Core concepts. The atlas already says t.
          family: plan-tracks-code
          round: 7
        - id: BR-19
          severity: Minor
          title: The notes' blend background is captured only when a snapshot is accepted, so a later OSC 11 report or theme change is ignored until the next note change
          detail: console_notes.go:49,107 cache noteStyleLocked() into pnotes.style; marks re-read the palette on every batch. Read the style in noteOverlay at draw time (check the c.mu lock order on the Presenter goroutine).
          family: snapshot-captures-mutable-input
          round: 7
      boundary: M3
      recipe: milestone-review
      isolation: checkout
      reviewed: dede99b74396ddbf01d0c52dd008126121560b52
      blocked: false
    - "n": 8
      timestamp: "2026-10-11T16:22:35-07:00"
      dispatched: "2026-10-11T16:15:38-07:00"
      agent: claude
      findings:
        - id: BR-20
          severity: Critical
          title: 'The notes menu''s click area outlives its drawing: under a private panel it swallows clicks and copies a note nobody can see'
          detail: 'noteOverlay returns early on FramePrivate but leaves pnotes.menu and menuOpen set, so routeNotesMenu (terminal_input.go:131) keeps hit-testing the old box. Reproduced: open the menu, showMenu(), click in the old box: the note is copied, "Note copied." appears, and the panel gets nothing. Fix: reset pnotes.menu on non-public frames (the menu stays open and the layout returns on the next public frame), and add a regression test. ARCH-ORDER: replace the loose open/layout fields with closed | open(layout of the visible frame).'
          family: hit-region-tracks-drawn-frame
          round: 8
        - id: BR-21
          severity: Important
          title: Marks are drawn over the notes menu, though the atlas and plan say the menu is drawn last
          detail: 'console_pointer.go:41 runs markOverlay after noteOverlay, and noteOverlay draws the menu, so marks tint the menu''s cells. 2nd in this family. Rule: a draw-order claim in docs needs a test that fails if the order is wrong. Move the menu into its own overlay after markOverlay and test that a mark inside the box leaves it untinted.'
          family: comment-claims-unenforced-invariant
          round: 8
        - id: BR-22
          severity: Minor
          title: Plan Chunk 4 gives a RenderNotesMenu signature with hover and a reverse-video hover; the code takes hover in Draw and uses a background color
          detail: '2nd in this family. Rule: a commit that departs from a Core concepts signature or visual spec updates that row in the same commit.'
          family: plan-tracks-code
          round: 8
        - id: BR-23
          severity: Minor
          title: Saving and restoring Hover around a style replacement is repeated in applyNotes, resyncNotes and restyleNotes
          detail: '3rd in this family. Rule: view state must not live inside a value that is replaced wholesale. Keep hover in pnotes.hover and add it to the style at draw time, which removes all three sites.'
          family: shared-helper-not-reused
          round: 8
        - id: BR-24
          severity: Minor
          title: No test that the notes menu closes when the broadcast ends (forgetNotes)
          detail: '3rd in this family. Rule: every teardown that resets menu state (resolve-last, pointer-off, broadcast-end) is one row of a single table-driven lifecycle test.'
          family: untested-cleanup-path
          round: 8
      boundary: M4
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 9
      timestamp: "2026-10-11T16:29:56-07:00"
      dispatched: "2026-10-11T16:25:10-07:00"
      agent: claude
      dispose:
        - id: BR-20
          disposition: addressed
          note: menuOverlay clears the layout on non-public frames; TestNotesMenuTakesNoClickUnderAPanel goes red with the clearing reverted. The tagged-state refactor was not adopted, but every reachable field combination is legal.
          round: 9
        - id: BR-21
          disposition: addressed
          note: menuOverlay runs after markOverlay (console_pointer.go:42); TestNotesMenuDrawnOverMarks goes red when the order is swapped.
          round: 9
        - id: BR-22
          disposition: not-addressed
          note: The Core concepts row was fixed, but Task 16 still says "Hover in reverse video" and the bullet still says it "returns the box's cells".
          round: 9
        - id: BR-23
          disposition: addressed
          note: Hover lives in pnotes.hover and is added to the style at draw time by drawStyleLocked; no hover save or restore is left in applyNotes, resyncNotes or restyleNotes.
          round: 9
        - id: BR-24
          disposition: addressed
          note: TestNotesMenuLifecycle is one table covering resolve-last, pointer-off and broadcast end; the broadcast_end row goes red when the reset in forgetNotes is removed.
          round: 9
      findings:
        - id: BR-25
          severity: Important
          title: The Done-when says the hover glow and highlight are tested, but removing their console wiring leaves every test green
          detail: 'Dropping style.Hover in drawStyleLocked (console_notes.go:204) passes all notes, menu, pointer and mark tests; TestNotesMenuOpenHoverCopy only checks pnotes.hover. 3rd in this family. Rule: every behavior a milestone''s Done-when, atlas or plan claims has a console-level test that goes red when its wiring is removed. Map each M4 Done-when clause to a test and add the missing one: hover an item, then assert the note''s chip and glow cells match DrawNotes with Hover set (at a time off the flash beat) and the item rows use notesMenuHover.'
          family: comment-claims-unenforced-invariant
          round: 9
        - id: BR-26
          severity: Minor
          title: A press inside the menu followed by a drag outside it sends button-held motion to the presenter or child, which never saw the press
          detail: 'routeNotesMenu returns false for motion outside the box even while pnotes.pressed is set. 2nd in this family. Rule: a gesture that starts in a hit region belongs to that region until its release. Take all motion while pressed is set.'
          family: hit-region-tracks-drawn-frame
          round: 9
      boundary: M4
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 10
      timestamp: "2026-10-11T16:35:46-07:00"
      dispatched: "2026-10-11T16:32:09-07:00"
      agent: claude
      dispose:
        - id: BR-22
          disposition: addressed
          note: Plan Core concepts now state RenderNotesMenu(notes, cols, rows) plus NotesMenu.Draw(f, hover, style) and the notesMenuHover background, matching notes_menu.go.
          round: 10
        - id: BR-25
          disposition: addressed
          note: TestNotesMenuHoverGlowsItsNote drives hover via console mouse input; reverting style.Hover in drawStyleLocked or the hover bg in Draw turns it red (verified).
          round: 10
        - id: BR-26
          disposition: addressed
          note: motion-while-pressed guard at console_notes.go:250; removing it fails TestNotesMenuDragStaysTheMenus (verified).
          round: 10
      findings:
        - id: BR-27
          severity: Minor
          title: A drag that starts outside the notes menu loses its motion while crossing the box
          detail: '3rd in this family. routeNotesMenu takes button-held motion over the box as hover whenever the press was elsewhere, so a child''s drag-select passing over the menu loses those events. Rule: the press decides who owns a gesture, and that owner keeps it until the release. Make pnotes.pressed an owner value (none/menu/elsewhere), set elsewhere on a press outside the box, pass motion and release through while it is elsewhere, and extend TestNotesMenuDragStaysTheMenus.'
          family: hit-region-tracks-drawn-frame
          round: 10
      boundary: M4
      recipe: milestone-review
      isolation: checkout
      reviewed: 826bd20866b4eec74189ab3c63377679ccdc7484
      blocked: false
    - "n": 11
      timestamp: "2026-10-11T19:34:23-07:00"
      dispatched: "2026-10-11T19:31:39-07:00"
      agent: claude
      dispose:
        - id: BR-7
          disposition: addressed
          note: M2 viewer UI (notes.js, mode button, dialog) is present at head, so README's viewer-page note actions now describe shipped surface.
          round: 11
        - id: BR-13
          disposition: addressed
          note: viewer.js:182,195,215 gate caps-off on sawCaps; connect.test.mjs view-page end test asserts no caps, red without the guard.
          round: 11
        - id: BR-16
          disposition: not-addressed
          note: Config.Now still wired (console_broadcast.go:155) but no test drives a note stamp and its drawing through the console's fake clock.
          round: 11
        - id: BR-18
          disposition: addressed
          note: Core concepts plan:285 now states paletteTint in t (0 = full) with a Revisions entry at plan:389; Task 12's stale wording is covered by the new plan-tracks-code finding.
          round: 11
        - id: BR-19
          disposition: addressed
          note: restyleNotes runs on every palette reply (terminal_input.go:66) and the test at console_notes_test.go:249 pins the immediate restyle.
          round: 11
        - id: BR-27
          disposition: not-addressed
          note: pnotes.pressed is still a bool; button-held motion from a press outside the box is taken as hover at console_notes.go:276-283.
          round: 11
      findings:
        - id: BR-28
          severity: Minor
          title: README says notes are resolved from a viewer's page, but the terminal menu's right click now resolves them too
          detail: '4th in this family. README.md:903-904 contradicts the right-click sentence added at 902. Rule: a commit that changes a capability sweeps every README/atlas sentence describing it (grep the verb) in the same commit.'
          family: docs-gate-user-surface
          round: 11
        - id: BR-29
          severity: Minor
          title: Plan Flash and NoteNextChange bullets still state a 1.5s hold, 10 steps and about 11 repaints per beat; code is 300ms, 25x20ms, about 26
          detail: '3rd in this family. plan:287-288 and Task 12 (297, 301) are stale; only Revisions plan:401 records the smoke change, and the peak of about 50 repaints/s during a fade is unstated. Rule: a commit that changes a constant a Core concepts bullet states rewrites that bullet in the same commit.'
          family: plan-tracks-code
          round: 11
      recipe: milestone-review
      isolation: fenced
      reviewed: cd5911909669681c42d2620b732b1e313ff4d9ae
      blocked: false
---

# Gate ledger — pair#447 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T14:10:32-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `plan-self-consistency` Task 3 Step 3 says to blank cells with uv.EmptyCell; Core concepts says EmptyCell drops the background and blanking must keep style
  2nd finding in this family. Rule: a task step names what to build and points to its Core concepts entry for how, without restating the mechanism. Follow Core concepts (set content " " and width 1, keep the style) and drop the uv.EmptyCell wording in Task 3.
  (carried from plan-quality PQ-7, deferred to the boundary review)

## Round 2 — 2026-10-11T14:10:32-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `docs-gate-user-surface` README update appears missing for the 💭N tab-bar count and notes cleared on pointing off
  README.md:876-891 documents the live status row as LIVE ⏸ 👆 👽; M1 adds an operator-visible 💭N after 👽 (reserve.go:227-230) and deletes all notes when pointing turns off. Add a sentence or two to the Remote pointer paragraph.
- **BR-3** [Minor] `shared-helper-not-reused` ParsePointBatch duplicates the strict-decode logic now in decodeStrict (ARCH-DRY)
  point.go:52-58 repeats DisallowUnknownFields plus the trailing-token check that notes.go:140 decodeStrict provides; have ParsePointBatch call decodeStrict.
- **BR-4** [Minor] `untested-cleanup-path` forgetNotes at broadcast end is not exercised with notes held
  TestNotesIgnoredAfterBroadcastEndsOrFromAnotherSession stops a broadcast that holds no notes, so plan Task 6's count-is-0-after-stop case with notes present is not tested.
- **BR-5** [Minor] `client-asserted-identity` Per-color note limit uses a client-supplied color, so one viewer can use all six colors
  ParseNote accepts any color 0-5 that is not tied to the stream's claimed color; the 48-note total cap and the rate limit still hold, and the setting is collaborative by design.
- **BR-6** [Minor] `comment-claims-unenforced-invariant` drawnNotes comment says empty unless pointing is on, but the function does not check
  This holds only because every pointer-phase change resyncs the snapshot (console_notes.go); reword the comment to say that.

## Round 3 — 2026-10-11T14:22:15-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — Plan line 147 still says "f.Clone() and uv.EmptyCell for blanks"; the code is correct (notes.go:205 blank keeps style), only the plan step is stale.
- BR-2 — addressed — README.md:894-901 adds the Sticky notes paragraph covering the 💭N count after the controls and that pointing off deletes notes; matches reserve.go:227-230 and pointer.go:103.
- BR-3 — addressed — point.go:50 now calls decodeStrict; the broadcast point tests pass unchanged.
- BR-4 — addressed — TestNotesForgottenWhenBroadcastEnds (console_notes_test.go:113) fails when the forgetNotes call in resetPointer is removed ("console still holds 1 notes (v1)"), so the fix is reachable.
- BR-5 — withdrawn — Consistent with the spec's trust model: the pointer link is a shared capability, point colors (#446) are client-asserted too, and the 48-note total cap plus rate limit are the real bound.
- BR-6 — addressed — console_notes.go:78-80 comment now says emptiness relies on the resync at each pointer-phase change; applyNotes also gates on pointer==pointerOn.

### Raised

- **BR-7** [Minor] `docs-gate-user-surface` README describes viewer-page note actions that only land in M2
  README.md:894-901 says pointer-link viewers can pin notes and anyone can tap a 💭 to read it, but M1 ships no viewer UI (notes are curl-only until M2 Tasks 8-10). This is the 2nd finding in this family; the rule: README changes describe only the surface present at the boundary's head, so tag M2-only behavior as upcoming or add it with M2.

## Round 4 — 2026-10-11T15:12:41-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — not-addressed — Plan line 147 (Task 3 Step 3) still reads "using f.Clone() and uv.EmptyCell for blanks"; code blank() at notes.go:208 keeps style, so only the plan text remains wrong.

### Raised

- **BR-8** [Important] `stream-end-resets-derived-state` Viewer keeps notes, pointing-on, mode button and open dialog after event end (viewer.js:200-206, 380-395)
  deps.reset only resets the terminal; server.go selects end/caps/notes in random order, so stale notes stay tappable (with Resolve) on the blank ended screen and comment mode still drafts. Reset all stream-derived state (notes list, setOn(false)) on the terminal event and add a test. ARCH-ORDER.
- **BR-9** [Minor] `late-completion-identity` noteDialog applies a save/resolve response to whatever dialog is open now, not the one that sent it (notes.js:74-81)
  Cancel during an in-flight save and open a new draft: the old 204 closes the new draft (text lost), or an old 409 shows on it. Capture a token per request and drop mismatched completions. ARCH-ORDER.
- **BR-10** [Minor] `docs-gate-user-surface` README Sticky notes paragraph never says how a viewer places a note (the 👆/💭 mode button)
  3rd finding in this family. Rule: the README feature paragraph names every control a person uses for the feature, and each milestone that adds or removes a control updates it; fold into M3's README task (hint bar removal, tooltips).
- **BR-11** [Minor] `dialog-keyboard-reach` Escape does not close a viewed note; aria-modal dialog has no focus move or trap in view mode
  The keydown listener is on #note, and only a draft moves focus into it (notes.js:89, draft focuses input).
- **BR-12** [Minor] `plan-self-consistency` Plan Task 8 gives noteAt(notes, col, row, cols, rows, guard); code takes noteAt(list, col, row, cols, rows) with list.guard
  2nd in family. Rule: task steps cite Core concepts for signatures rather than restating them, which also covers BR-1; update Task 8 to match.

## Round 5 — 2026-10-11T15:15:54-07:00 (claude) — passed

### Disposed

- BR-1 — not-addressed — plan.md:147 Task 3 Step 3 still reads "using f.Clone() and uv.EmptyCell for blanks", contradicting Core concepts line 37.
- BR-8 — addressed — viewer.js:208-209 delivers caps-off and an empty notes list on end; connect.test.mjs BR-8 test fails with the pre-fix viewer.js (verified by revert). New view-link side effect raised separately.
- BR-9 — addressed — notes.js:62-91 generation token; notes.test.mjs "an answer applies only to the dialog that sent it" fails on revert (verified).
- BR-10 — addressed — README.md:895-896 now names the 👆/💭 mode button under the swatches.
- BR-11 — addressed — notes.js:141 focuses Close on view; test fails on revert. No focus trap yet, acceptable at Minor.
- BR-12 — addressed — plan.md:232 Task 8 now cites Core concepts for the noteAt signature instead of restating it.

### Raised

- **BR-13** [Minor] `stream-end-resets-derived-state` End sends a made-up caps-off to view links, which never get caps, so they show the "Pointing is off" hint band
  viewer.js:208 calls deps.caps({pointer:false}) on every page. server.go:325-340 sends caps only to pointer pages, so on a view link setOn(false) un-hides #hint and band() lifts the stage after the end. 2nd in family. Rule: an end puts stream-derived state back to the page variant's join state and never replays events that variant does not receive. Fix: only send caps-off when a caps event was seen (or use one ended() dep), and add a view-link end test.

## Round 6 — 2026-10-11T15:47:45-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Plan Task 3 Step 3 now reads "Implement DrawNotes per Core concepts" (diff 75f2e1ac..f7a574e5); the uv.EmptyCell wording appears only in Core concepts' rationale (plan line 37).

### Raised

- **BR-14** [Important] `untested-cleanup-path` Task 13 tests missing: flash-timer re-arm after firing and notes timer cleared on broadcast stop
  2nd finding in this family. Rule: every timer the console arms needs a test for each way it is torn down plus a test that it re-arms after firing; today that is pmarks.timer and pnotes.timer, each on pointer-off and on broadcast end (resetPointer). Only notes-on-pointer-off is tested (console_notes_test.go:177). Dropping the scheduleNotes call from the AfterFunc callback, or the timer stop from forgetNotes, keeps the suite green. Add one table-driven test over both timers and both teardown paths, plus a re-arm check through the test clock or a delay seam.
- **BR-15** [Minor] `shared-helper-not-reused` Blend predicate Known && TrueColor && !NoColor duplicated in noteStyleLocked and applyPoints
  2nd finding in this family. Rule: when a new helper encodes a predicate that already exists inline, the old site calls the new helper in the same change. A grep finds exactly two sites: console_notes.go:69 and console_pointer.go:180. applyPoints should take its blend from noteStyleLocked, or both should use a shared paletteBlendLocked.
- **BR-16** [Minor] `single-clock-source` Note.At stamped with wall-clock time.Now in session.go while the console draws with injectable c.now
  With a fake console clock, note ages come out negative, are clamped to 0, and the note looks permanently flashing; flash timing can't be driven from tests. Stamp At from the same clock the drawing uses.
- **BR-17** [Minor] `envelope-unstated-at-bound` Notes flash timer repaint ceiling at the 48-note bound (about 35 full repaints/s to every viewer) is not stated
  The plan states the per-note cost (about 11 repaints per 15s) but not the total at the 6x8 note limit. Write that ceiling into the plan, or snap flash phases to a coarser grid so repaints for several notes coincide.

## Round 7 — 2026-10-11T15:56:38-07:00 (claude) — passed

### Disposed

- BR-14 — addressed — TestCollaborationTimersRearmAndTearDown covers both timers on re-arm, pointer-off and broadcast end; reverting the AfterFunc re-arm or the forgetNotes Stop turns it red (verified).
- BR-15 — addressed — blendable(Palette) at console_notes.go:73 is now the only site; applyPoints (console_pointer.go:180) and noteStyleLocked both call it, and a grep finds no other inline copy.
- BR-16 — not-addressed — Config.Now is wired to c.now, but reverting both the wiring and the stamp leaves every broadcast and couchtty test green; no regression test drives stamp and draw through the fake clock.
- BR-17 — addressed — The plan Flash bullet states the 48-note ceiling (about 35/s per-note) and the shared beat bounds it; TestNoteFlashRepaintsDontGrowWithNotes pins it.

### Raised

- **BR-18** [Minor] `plan-tracks-code` Plan Core concepts describe paletteTint with a strength argument (1 = full) but the code takes t (0 = full)
  Plan line 285 and the glow strengths (0.5 and 0.25) are the inverse of notehalo.go:29-30 and marks.go:98 (t 0.5 and 0.75). The colors match but the stated API is inverted; add a Revisions entry and fix Core concepts. The atlas already says t.
- **BR-19** [Minor] `snapshot-captures-mutable-input` The notes' blend background is captured only when a snapshot is accepted, so a later OSC 11 report or theme change is ignored until the next note change
  console_notes.go:49,107 cache noteStyleLocked() into pnotes.style; marks re-read the palette on every batch. Read the style in noteOverlay at draw time (check the c.mu lock order on the Presenter goroutine).

## Round 8 — 2026-10-11T16:22:35-07:00 (claude) — BLOCKED

### Raised

- **BR-20** [Critical] `hit-region-tracks-drawn-frame` The notes menu's click area outlives its drawing: under a private panel it swallows clicks and copies a note nobody can see
  noteOverlay returns early on FramePrivate but leaves pnotes.menu and menuOpen set, so routeNotesMenu (terminal_input.go:131) keeps hit-testing the old box. Reproduced: open the menu, showMenu(), click in the old box: the note is copied, "Note copied." appears, and the panel gets nothing. Fix: reset pnotes.menu on non-public frames (the menu stays open and the layout returns on the next public frame), and add a regression test. ARCH-ORDER: replace the loose open/layout fields with closed | open(layout of the visible frame).
- **BR-21** [Important] `comment-claims-unenforced-invariant` Marks are drawn over the notes menu, though the atlas and plan say the menu is drawn last
  console_pointer.go:41 runs markOverlay after noteOverlay, and noteOverlay draws the menu, so marks tint the menu's cells. 2nd in this family. Rule: a draw-order claim in docs needs a test that fails if the order is wrong. Move the menu into its own overlay after markOverlay and test that a mark inside the box leaves it untinted.
- **BR-22** [Minor] `plan-tracks-code` Plan Chunk 4 gives a RenderNotesMenu signature with hover and a reverse-video hover; the code takes hover in Draw and uses a background color
  2nd in this family. Rule: a commit that departs from a Core concepts signature or visual spec updates that row in the same commit.
- **BR-23** [Minor] `shared-helper-not-reused` Saving and restoring Hover around a style replacement is repeated in applyNotes, resyncNotes and restyleNotes
  3rd in this family. Rule: view state must not live inside a value that is replaced wholesale. Keep hover in pnotes.hover and add it to the style at draw time, which removes all three sites.
- **BR-24** [Minor] `untested-cleanup-path` No test that the notes menu closes when the broadcast ends (forgetNotes)
  3rd in this family. Rule: every teardown that resets menu state (resolve-last, pointer-off, broadcast-end) is one row of a single table-driven lifecycle test.

## Round 9 — 2026-10-11T16:29:56-07:00 (claude) — BLOCKED

### Disposed

- BR-20 — addressed — menuOverlay clears the layout on non-public frames; TestNotesMenuTakesNoClickUnderAPanel goes red with the clearing reverted. The tagged-state refactor was not adopted, but every reachable field combination is legal.
- BR-21 — addressed — menuOverlay runs after markOverlay (console_pointer.go:42); TestNotesMenuDrawnOverMarks goes red when the order is swapped.
- BR-22 — not-addressed — The Core concepts row was fixed, but Task 16 still says "Hover in reverse video" and the bullet still says it "returns the box's cells".
- BR-23 — addressed — Hover lives in pnotes.hover and is added to the style at draw time by drawStyleLocked; no hover save or restore is left in applyNotes, resyncNotes or restyleNotes.
- BR-24 — addressed — TestNotesMenuLifecycle is one table covering resolve-last, pointer-off and broadcast end; the broadcast_end row goes red when the reset in forgetNotes is removed.

### Raised

- **BR-25** [Important] `comment-claims-unenforced-invariant` The Done-when says the hover glow and highlight are tested, but removing their console wiring leaves every test green
  Dropping style.Hover in drawStyleLocked (console_notes.go:204) passes all notes, menu, pointer and mark tests; TestNotesMenuOpenHoverCopy only checks pnotes.hover. 3rd in this family. Rule: every behavior a milestone's Done-when, atlas or plan claims has a console-level test that goes red when its wiring is removed. Map each M4 Done-when clause to a test and add the missing one: hover an item, then assert the note's chip and glow cells match DrawNotes with Hover set (at a time off the flash beat) and the item rows use notesMenuHover.
- **BR-26** [Minor] `hit-region-tracks-drawn-frame` A press inside the menu followed by a drag outside it sends button-held motion to the presenter or child, which never saw the press
  routeNotesMenu returns false for motion outside the box even while pnotes.pressed is set. 2nd in this family. Rule: a gesture that starts in a hit region belongs to that region until its release. Take all motion while pressed is set.

## Round 10 — 2026-10-11T16:35:46-07:00 (claude) — passed

### Disposed

- BR-22 — addressed — Plan Core concepts now state RenderNotesMenu(notes, cols, rows) plus NotesMenu.Draw(f, hover, style) and the notesMenuHover background, matching notes_menu.go.
- BR-25 — addressed — TestNotesMenuHoverGlowsItsNote drives hover via console mouse input; reverting style.Hover in drawStyleLocked or the hover bg in Draw turns it red (verified).
- BR-26 — addressed — motion-while-pressed guard at console_notes.go:250; removing it fails TestNotesMenuDragStaysTheMenus (verified).

### Raised

- **BR-27** [Minor] `hit-region-tracks-drawn-frame` A drag that starts outside the notes menu loses its motion while crossing the box
  3rd in this family. routeNotesMenu takes button-held motion over the box as hover whenever the press was elsewhere, so a child's drag-select passing over the menu loses those events. Rule: the press decides who owns a gesture, and that owner keeps it until the release. Make pnotes.pressed an owner value (none/menu/elsewhere), set elsewhere on a press outside the box, pass motion and release through while it is elsewhere, and extend TestNotesMenuDragStaysTheMenus.

## Round 11 — 2026-10-11T19:34:23-07:00 (claude) — passed

### Disposed

- BR-7 — addressed — M2 viewer UI (notes.js, mode button, dialog) is present at head, so README's viewer-page note actions now describe shipped surface.
- BR-13 — addressed — viewer.js:182,195,215 gate caps-off on sawCaps; connect.test.mjs view-page end test asserts no caps, red without the guard.
- BR-16 — not-addressed — Config.Now still wired (console_broadcast.go:155) but no test drives a note stamp and its drawing through the console's fake clock.
- BR-18 — addressed — Core concepts plan:285 now states paletteTint in t (0 = full) with a Revisions entry at plan:389; Task 12's stale wording is covered by the new plan-tracks-code finding.
- BR-19 — addressed — restyleNotes runs on every palette reply (terminal_input.go:66) and the test at console_notes_test.go:249 pins the immediate restyle.
- BR-27 — not-addressed — pnotes.pressed is still a bool; button-held motion from a press outside the box is taken as hover at console_notes.go:276-283.

### Raised

- **BR-28** [Minor] `docs-gate-user-surface` README says notes are resolved from a viewer's page, but the terminal menu's right click now resolves them too
  4th in this family. README.md:903-904 contradicts the right-click sentence added at 902. Rule: a commit that changes a capability sweeps every README/atlas sentence describing it (grep the verb) in the same commit.
- **BR-29** [Minor] `plan-tracks-code` Plan Flash and NoteNextChange bullets still state a 1.5s hold, 10 steps and about 11 repaints per beat; code is 300ms, 25x20ms, about 26
  3rd in this family. plan:287-288 and Task 12 (297, 301) are stale; only Revisions plan:401 records the smoke change, and the peak of about 50 repaints/s during a fade is unstated. Rule: a commit that changes a constant a Core concepts bullet states rewrites that bullet in the same commit.

## Open findings

- **BR-16** [Minor] `single-clock-source` Note.At stamped with wall-clock time.Now in session.go while the console draws with injectable c.now
- **BR-27** [Minor] `hit-region-tracks-drawn-frame` A drag that starts outside the notes menu loses its motion while crossing the box
- **BR-28** [Minor] `docs-gate-user-surface` README says notes are resolved from a viewer's page, but the terminal menu's right click now resolves them too
- **BR-29** [Minor] `plan-tracks-code` Plan Flash and NoteNextChange bullets still state a 1.5s hold, 10 steps and about 11 repaints per beat; code is 300ms, 25x20ms, about 26
