---
id: 000447
status: working
deps: [000446, 000449]
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours: 6.3
card_mirror: 'a5ca5962240f2637e8522d6c0e41f218b96ffb65' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T12:25:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# Broadcast sticky notes: viewers pin comments on the screen

## Problem

Broadcast viewers can point (fading marks, #412; per-viewer colors, pair#446) but can't say anything about what they see. The operator (10-11, during a live demo) wants viewers to leave sticky notes on the screen.

## Spec

The operator's requirements (10-11), to be refined with the operator in the slot before implementation:

1. Pointer mode stays the default: highlight with a trail/fade.
2. A pointer-color icon (pair#446).
3. A second icon switches between pointer mode and comment mode.
4. In comment mode, a click on the screen places a sticky note there.
5. The author gets a dialog to type the comment.
6. Save publishes it; Couch remembers it.
7. A 💭 is drawn where they clicked.
8. The operator's terminal shows the 💭 but can't interact with it (keeps the design simple).
9. In any viewer, clicking a 💭 opens a fixed-size dialog with the note's content, in a typical sticky-note color.
10. The author can open their own note the same way.
11. A note is visible to others only once saved: the draft is the author's private space (it never leaves their browser before Save).
12. The note dialog has a Resolve button that removes the note.
13. Any viewer can resolve anyone's note.

Decisions from the design session with the operator (10-11):
- **Anchoring: none (option A).** A note stays on the screen cell where it was clicked, whatever scrolls underneath. Scrollback stays closed. Following the text and a dock were discussed and dropped to keep it simple; they can come later.
- **Drawing:** Couch draws the 💭 in its overlay, like pointer marks, so the operator's terminal and every viewer get the same picture from the stream. The viewer page draws nothing itself: it gets the list of notes (id, cell, text) as a stream event and turns a tap on a 💭 into "open that note". No tint: the emoji keeps its own colors.
- **Two layers:** the view-only link stays read-only (no request body, #395) but sees every 💭 and can open and read notes. The pointer link adds pointing, notes and Resolve. Within that layer anyone can add or resolve any note; the setting is collaborative. Length cap, plain text and rate limits stay as hygiene.
- **Lifetime:** in memory, one broadcast. Never on disk. Turning pointing off (click or the 👆-hidden watch) turns off the whole collaboration layer: every note is deleted, as marks are cleared. The broadcast ending clears them too.
- **Operator visibility:** a `💭N` open-notes count in the broadcast group of the Couch tab bar, between 👽 and the slot name. Not interactive (rule 8).
- **Mode switch:** the pointer/comment icon sits directly below the color swatches, in the same column at the screen's right edge (pair#449).
- **Taps on a 💭:** in pointer mode a tap on a 💭 opens the note instead of pointing; in comment mode it opens the note instead of placing a second one on top.
- **View-only dialog:** the same note dialog, read-only, with no Resolve button.
- **Grid resize:** a note whose cell falls off the grid is hidden until it fits again; it never moves to another cell.
- **Limits:** a note carries its author's pointer color (0–5, pair#446), stored but not drawn. At most 8 open notes per color, so at most 48 per broadcast. Text is at most 1 KB of plain text. A Save over a limit fails with a short message in the dialog.
- **Occlusion:** the 💭 covers two cells; never on the guarded control cells.

Depends on pair#446 (colors) and pair#449 (swatch column at the right edge) landing first; no stacked branch.

## Done when

- With two live viewers on the pointer link: one places a 💭 in comment mode, types, saves; the 💭 appears at that cell on the operator's terminal and on both viewers' screens, and the operator's tab bar shows `💭1` between 👽 and the slot name.
- Either viewer opens the note by tapping the 💭 (in pointer or comment mode) and sees its text in a fixed-size sticky-note dialog; either one resolves it, and the 💭 and the count go away everywhere.
- A view-only viewer sees the 💭, opens it read-only, and has no Resolve; the view-only link still refuses every request body (test).
- A draft never reaches Couch before Save (viewer.js test: no request until Save).
- Note text renders as plain text; over 1 KB, a 9th note in one color, or a malformed note request is refused, with a short message in the dialog (tests).
- Notes are never drawn on the guarded control cells or on a private frame, are hidden when their cell is off the grid, and are gone when pointing turns off or the broadcast ends (tests).
- (M3) A 💭 sits on a chip of its author's pointer color with a round glow (about 5 rows by 10 columns, stronger near the chip) that blends into the operator's background, or uses the 256-color ladders without truecolor; the glow never covers the controls; it flashes at full color on arrival and every 15s until resolved (tests on the pure styling and the timing).
- (M3) The viewer page has no hint bar: the screen uses the whole window, and the rail controls carry their explanation as tooltips.
- (M4) Clicking `💭N` opens a notes menu over the screen listing each note with its color; hovering an item highlights it and makes its 💭 glow; clicking an item copies its text and says so; right-clicking an item copies and resolves it; clicking `💭N` or the menu's close mark closes it; the menu closes when the last note goes or pointing turns off; mouse events over the open menu never reach the child (tests).

## Plan

Durable plan: `workshop/plans/000447-broadcast-sticky-notes-viewers-pin-comments-on-the-screen-plan.md`.

- [x] Design session with the operator in the slot: settle the open questions, then fix Spec and Done when.
- [x] M1 — Notes in Couch: note board and strict parsing, `note`/`resolve` routes, `notes` event to every stream, session checks, 💭 overlay and `💭N` in the tab bar.
- [x] M2 — The viewer page: tap routing, comment mode, the sticky-note dialog and the mode switch below the swatches; live check with two viewers and a view-only one.
- [x] M3 — Visibility: each 💭 on a chip of its author's color with a soft round glow, flashing on arrival and every 15s until resolved; the viewer's hint bar becomes tooltips on the rail controls.
- [x] M4 — Notes in the terminal: click `💭N` to open a notes menu over the screen; hovering an item highlights it and makes its 💭 glow; clicking copies the text; right-click copies and resolves.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec            design=0.8 impl=0.04
item: smaller-go-module     design=0.2 impl=0.2
item: smaller-go-module     design=0.2 impl=0.2
item: smaller-go-module     design=0.1 impl=0.16
item: tui-screen            design=0.5 impl=0.32
item: atlas-docs            design=0.1 impl=0.04
item: milestone-review      design=0.0 impl=0.16
item: milestone-review      design=0.0 impl=0.16
item: ux-rename-iteration   design=0.3 impl=0.08
item: scope-pivot           design=0.3 impl=0.08
item: smaller-go-module     design=0.1 impl=0.2
item: smaller-go-module     design=0.1 impl=0.16
item: tui-screen            design=0.3 impl=0.32
item: ux-rename-iteration   design=0.2 impl=0.08
item: atlas-docs            design=0.05 impl=0.04
item: milestone-review      design=0.0 impl=0.16
item: milestone-review      design=0.0 impl=0.16
design-buffer: 0.15
total: 6.30
```

Revision (2026-10-11, operator smoke): the second block of items is the scope revision — the revision itself, the glow and flash timing (`notehalo.go` + `paletteTint`), the menu renderer, the menu interaction (routing, hover, copy; a screen-plus-state-machine), the tooltip change on the viewer, atlas, and two more milestone reviews. Items, in order: the design session with the operator and the plan with three plan-quality rounds; the note board, parsing and drawing (`notes.go`); notes in `PointerState`, the routes and session wiring; the console overlay and the `💭N` count; the viewer page (tap routing, comment mode, dialog, mode switch — the closest primitive is a screen with a state machine and tests); atlas; two milestone reviews; one live-smoke iteration round with the operator. Impl values are at 40% of the v2 table (v3.1); +15% design buffer for a thorough plan doc.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* The calibration doc is flagged stale by `sdlc estimate-source`; per-primitive hours are provisional.

## Log

### 2026-10-11
- 2026-10-11: closed — Full suite on bb9a2bc5+godoc fix: make -k test green (default TMPDIR, after npm ci in tests/terminal-oracle for #412 addon-unicode11); TMPDIR=scratchpad make test-changelog green; env -i go test ./... green except 2 couchcore provision-recovery tests that time out under full parallel load and pass alone (couchcore untouched by this branch). Broadcast+couchtty notes tests green under -race. Headless Chrome smoke (dialog, 390px, full-window screen, tooltips, glow). Operator smoke on pair:0 of M1-M4 plus tuning (300ms hold, 20ms fade steps, inner-ring-only light rest, right-click resolve+copy): "looks good, ship this".; review verdict: FIX-THEN-SHIP
- 2026-10-11: closed M4 — Round 2 fixes: BR-25 TestNotesMenuHoverGlowsItsNote drives hover by mouse motion through the console and asserts the drawn note glow equals the full chip color and the item row uses notesMenuHover, with an unhovered control off the beat (red when style.Hover is dropped from drawStyleLocked); drag rule: TestNotesMenuDragStaysTheMenus asserts routeNotesMenu takes drag motion after a press in the menu and its release, and releases afterwards (red without the rule); BR-22 plan wording. couchtty notes/menu/collaboration -race green; round-1 fixes and evidence as before.; review verdict: SHIP
- 2026-10-11: closed M3 — As round 1 (full suite green at f7a574e5; unit/property/console/CDP evidence). Round 1 fixes in 725bfa3a + atlas: BR-14 table test over marks and notes timers x {re-arm after firing, pointer-off, broadcast end}, teardown asserted immediately; mutation-checked (dropping scheduleNotes in the timer callback fails re-arm; dropping the stop in forgetNotes fails broadcast end). Shared 15s flash beat + arrival flash; TestNoteFlashRepaintsDontGrowWithNotes pins repaints per beat equal for 1 and 48 notes. Config.Now single clock; blendable shared. broadcast + couchtty (-race x3) green.; review verdict: FIX-THEN-SHIP
- 2026-10-11: closed M2 — As in round 1 (node tests via TestViewerNode, page test, headless-Chrome CDP smoke incl. 390px, operator smoke on pair:0). Round 1 fixes in 706a3267: BR-8 end delivers pointer-off and empty notes through the same handlers (connect test, order-independent); dialog answers bound to their open generation (test: old 204 does not close a new draft); view focuses Close so Escape works (test); README names the mode button; plan Task 8 cites Core concepts. 59 node tests green.; review verdict: FIX-THEN-SHIP
- 2026-10-11: closed M1 — As before (unit+property+fuzz; PointerState ordering, injected interleavings, -race stress; real-session route tests; console integration test with real session+console+viewer emulator). Review round 1 fixes in 1c3a0b16: README notes paragraph (BR-2), ParsePointBatch uses decodeStrict (FuzzParsePointBatch 10s green), forget-at-broadcast-end tested with notes held, drawnNotes comment. broadcast + couchtty (-race, notes/pointer/status) green after the fixes. Client-chosen color for the per-color limit is the accepted collaborative design (Spec).; review verdict: SHIP
- Design session with the operator: anchoring option A (fixed cell), 💭 not 💬 (💬 is the review threads'), view link read-only but sees notes, pointing off deletes every note, 8 notes per color, `💭N` after 👽, mode switch under the #449 swatch rail.
- change-code: plan-quality took 3 rounds, all ordering (ARCH-ORDER): the board moved inside `PointerState` so off and clear are one critical section; `addNote` carries the flip generation; no foreign call under `p.mu` (`Hub.Current` waits on the hub loop, which takes `p.mu`).
- M1 discoveries: (1) Go's JSON decoder turns invalid UTF-8 into U+FFFD, so `ParseNote` refuses a non-UTF-8 body up front. (2) The renderer draws a row's text only up to its used columns, so a 💭 on a blank row vanished while marks (background only) showed; `DrawNotes` widens them (test added). (3) A 👆 click runs on the Run loop and `DisablePointer` reports the emptied board through `OnNotes`; a waiting `onNotes` deadlocked it, so it hands off without waiting. (4) New sources need entries in `artifactpath/manifest.go`.
- M1 verification: `make -k test` (default TMPDIR, five-var scrub, unsandboxed) runs every shell/lua target green; its only failure was the Go recipe's `artifactpath` inventory (fixed). `TMPDIR=<scratchpad> make test-changelog` passes. `go test ./... -count=1`: green except `couchsingleton`'s selection-size fixture, which embeds TMPDIR and passes under the default TMPDIR. The end-to-end check is `TestNoteDrawnOnOperatorAndViewers`: a real session and console, a note POSTed over HTTP, the 💭 on the operator's screen and in a viewer's emulator, `💭1` in the tab bar, resolve removes both. The live two-browser check is M2's.
- M2: the page test allowed exactly one `fetch`, the pointer POST; following the #446 lesson it was narrowed, not dropped: one `postJSON` limited to `POST_ROUTES`, and HTML sinks banned in `viewer.js` and `notes.js`. Real-browser smoke via `BROADCAST_MANUAL_POINTER=1 TestManualViewerServer` + headless Chrome driven over CDP: a curl-posted note shows as 💭 and opens read-only text (`<b>` literal), the mode button switches to 💭 and the hint follows, a tap drafts, typed text counts bytes (31/1024), Save closes and a second 💭 appears in the stream, it opens, Resolve removes it; no console errors. At 390px the dialog sits inside the viewport (55–335px), no horizontal scroll.

## Revisions

- 2026-10-11 — operator smoke on pair:0 (scope revision, folded in as M3 and M4, from "1. in a full page of text, 💭 is really not quite visible", "2. … we should display those sticky notes on the terminal as well"):
  - Visibility reverses "no tint": the 💭's two cells get its author's color as a chip, with a radial glow around it blended into the background, an arrival flash, and a flash every 15s until resolved (staggered naturally by each note's age). Notes now record when they were saved (not sent on the wire).
  - Rule 8 is lifted for reading and copying: `💭N` opens a notes menu drawn over the screen (viewers see it, as they see everything the operator sees). Hover highlights an item and makes its 💭 glow; a click copies the note's text (OSC 52, as links are copied). Read and copy only: no Resolve from the terminal (operator didn't ask for it; the default). The menu does not take Esc, which agents use; it closes by clicking `💭N` or its close mark. Free-form selection works with the terminal's own selection override (Shift- or Option-drag).
  - The viewer's hint bar goes: its text moves into tooltips on the swatches and the mode button, so the screen gets the whole window. Touch devices show no tooltips; the icons stand alone there.
- 2026-10-11 — operator smoke of M3/M4 on pair:0 ("the animation is not smooth, needs more steps", "no need to stay full for 1 second … maybe just stay full glow for 300ms", "the most faded color, can fade more", "operator can resolve one with right click. which also copies"):
  - Flash: hold 1.5s → 300ms; fade 500ms in 25 steps (20ms) instead of 10; resting glow fainter (inner ring t 0.5 → 0.7, outer 0.75 → 0.88).
  - Terminal menu: a right click on an item copies its text and resolves it (`Session.ResolveNote`), reversing the earlier read-and-copy-only default.
- 2026-10-11 — operator: "the resting quiet state should just have one shade (the inner one sized, and very light tint)". At rest only the inner ring shows, at t 0.9; the outer ring fades out to nothing (t 1, not drawn) and appears only during a flash.
