---
id: 000447
status: working
deps: [000446, 000449]
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours: 3.89
card_mirror: '6e3adbb1b0520eb3ce0b53ce451c34b87a3e81c7' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T12:25:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
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

## Plan

Durable plan: `workshop/plans/000447-broadcast-sticky-notes-viewers-pin-comments-on-the-screen-plan.md`.

- [x] Design session with the operator in the slot: settle the open questions, then fix Spec and Done when.
- [x] M1 — Notes in Couch: note board and strict parsing, `note`/`resolve` routes, `notes` event to every stream, session checks, 💭 overlay and `💭N` in the tab bar.
- [ ] M2 — The viewer page: tap routing, comment mode, the sticky-note dialog and the mode switch below the swatches; live check with two viewers and a view-only one.

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
design-buffer: 0.15
total: 3.89
```

Items, in order: the design session with the operator and the plan with three plan-quality rounds; the note board, parsing and drawing (`notes.go`); notes in `PointerState`, the routes and session wiring; the console overlay and the `💭N` count; the viewer page (tap routing, comment mode, dialog, mode switch — the closest primitive is a screen with a state machine and tests); atlas; two milestone reviews; one live-smoke iteration round with the operator. Impl values are at 40% of the v2 table (v3.1); +15% design buffer for a thorough plan doc.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* The calibration doc is flagged stale by `sdlc estimate-source`; per-primitive hours are provisional.

## Log

### 2026-10-11
- 2026-10-11: closed M1 — As before (unit+property+fuzz; PointerState ordering, injected interleavings, -race stress; real-session route tests; console integration test with real session+console+viewer emulator). Review round 1 fixes in 1c3a0b16: README notes paragraph (BR-2), ParsePointBatch uses decodeStrict (FuzzParsePointBatch 10s green), forget-at-broadcast-end tested with notes held, drawnNotes comment. broadcast + couchtty (-race, notes/pointer/status) green after the fixes. Client-chosen color for the per-color limit is the accepted collaborative design (Spec).; review verdict: SHIP
- Design session with the operator: anchoring option A (fixed cell), 💭 not 💬 (💬 is the review threads'), view link read-only but sees notes, pointing off deletes every note, 8 notes per color, `💭N` after 👽, mode switch under the #449 swatch rail.
- change-code: plan-quality took 3 rounds, all ordering (ARCH-ORDER): the board moved inside `PointerState` so off and clear are one critical section; `addNote` carries the flip generation; no foreign call under `p.mu` (`Hub.Current` waits on the hub loop, which takes `p.mu`).
- M1 discoveries: (1) Go's JSON decoder turns invalid UTF-8 into U+FFFD, so `ParseNote` refuses a non-UTF-8 body up front. (2) The renderer draws a row's text only up to its used columns, so a 💭 on a blank row vanished while marks (background only) showed; `DrawNotes` widens them (test added). (3) A 👆 click runs on the Run loop and `DisablePointer` reports the emptied board through `OnNotes`; a waiting `onNotes` deadlocked it, so it hands off without waiting. (4) New sources need entries in `artifactpath/manifest.go`.
- M1 verification: `make -k test` (default TMPDIR, five-var scrub, unsandboxed) runs every shell/lua target green; its only failure was the Go recipe's `artifactpath` inventory (fixed). `TMPDIR=<scratchpad> make test-changelog` passes. `go test ./... -count=1`: green except `couchsingleton`'s selection-size fixture, which embeds TMPDIR and passes under the default TMPDIR. The end-to-end check is `TestNoteDrawnOnOperatorAndViewers`: a real session and console, a note POSTed over HTTP, the 💭 on the operator's screen and in a viewer's emulator, `💭1` in the tab bar, resolve removes both. The live two-browser check is M2's.
