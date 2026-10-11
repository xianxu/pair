---
id: 000447
status: open
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: 'dc94db19dad87fc820fe9f0e1dcf11553c8672b0' # card fields mirrored from issue-cards; edit via sdlc
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
7. A 💬 is drawn where they clicked.
8. The operator's terminal shows the 💬 but can't interact with it (keeps the design simple).
9. In any viewer, clicking a 💬 opens a fixed-size dialog with the note's content, in a typical sticky-note color.
10. The author can open their own note the same way.
11. A note is visible to others only once saved: the draft is the author's private space (it never leaves their browser before Save).
12. The note dialog has a Resolve button that removes the note.
13. Any viewer can resolve anyone's note.

Open design questions (the TL's, for the operator):
- **Anchoring under scroll:** a note outlives the screen it was placed on. Pin it to an absolute scrollback line (scrolls with content), or pin it to the grid cell and mark it stale once that line's content changes?
- **Lifetime:** in memory per broadcast (simple; a Couch restart clears notes) vs persisted.
- **Operator visibility:** with rule 8 the operator can't read notes from the terminal; an open-notes count in the broadcast indicator, or the operator keeps a viewer page open?
- **Attribution:** tag a note with its author's pointer color (pair#446)?
- **Safety:** anyone with the link can write: a length cap (about 1 KB), plain-text rendering only, and rate limits like pointing.
- **Occlusion:** the 💬 is two cells wide and covers terminal text.

Depends on pair#446 (viewer icon bar, colors) landing first; no stacked branch.

## Done when

- To be fixed with the operator during design. At minimum: save, see, open and resolve work across two live viewers and show in the operator's terminal; drafts never reach Couch before Save; note text is length-capped and rendered as plain text.

## Plan

- [ ] Design session with the operator in the slot: settle the open questions, then fix Spec and Done when.
- [ ] Implementation plan (durable plan if it leaves the quick-flow shell), after pair#446 lands.

## Log

### 2026-10-11
