---
id: 000499
status: done
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '63db9a0e0a3a9fe909d1b1c23aa7a8dcbc136a97' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T15:11:40-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "e03b774e", done: "604cca88"}
actual_hours: 0.84
---

# broadcast-web: viewers can still draw and leave notes while the cast is paused; disable every viewer action while paused

## Problem

Operator smoke test, 10-12: after pausing the web cast, viewers could still draw strokes and add notes, and those notes appeared when the cast resumed. #470 had decided 'notes allowed while paused, refused after stop end'; the operator reverses that: while paused, viewers do nothing.

## Spec

While a cast is paused, every viewer action is disabled:
- **Server (the guard):** the Board refuses `stroke`, `note` and `note-state` posts while the cast is paused (a fixed sentence: 'The cast is paused.'), as the drive routes already do (`ErrNotLive`); nothing posted during a pause is ever recorded or shown after resume.
- **Viewer:** the pen, swatch rail and 💭 controls are disabled while the stream says paused, with the paused state shown; re-enabled on resume.
- Same rule for the terminal broadcast's pointer and notes if it has a paused state.
- Update the atlas (the #470 decision is revised) and the board's tests.

## Done when

- A test: post a note and a stroke while paused → refused; resume → nothing from the pause appears; viewer controls disabled while paused.
- The operator smoke-tested it.

## Plan

One rule at two layers, plus the viewer (quick flow):

- [x] Board admission: `pageLocked` refuses while `!State.Live` with `ErrPaused`; the routes answer 409 "The cast is
  paused." (`refusePaused`). Covers stroke, note and note-state.
- [x] Writer admission: `recording.State.OnDoc` requires the segment live (`Live && DocSeg == doc`), so a request the
  board admitted just before the pause and queued after the `stop` is dropped by the recorder (and by the reference
  `board.Writer`) — one rule for both writers (ARCH-DRY).
- [x] Viewer (annotate.js): `paused()`; a `stop` puts the pen down (watch mode), the rail's buttons are disabled and
  say "The cast is paused." (CSS greys them), a `seg` re-enables them; pointerdown is ignored while paused.
- [x] Terminal broadcast: no paused state exists (no pause anywhere in cmd/internal/broadcast), so nothing to do.
- [x] atlas/webcast-format.md (#470's decision revised) and README.
- [x] Tests: TestDocumentRules (note/stroke/note-state → ErrPaused; after resume only the pre-pause note, no strokes),
  TestRecorderEmitDropsARecordDuringAPause (fails with the old OnDoc), TestBoardRoutesRefuseWhilePaused (409 + the
  sentence on all three routes), TestAnnotateLiveTwoViewers (WEBCAST_LIVE_CHROME: every rail button disabled while
  paused, none after resume; fails with the disable line removed).

## Log

### 2026-10-12
- 2026-10-12: closed — make -k test exit 0 on b07d91a4 (+ main merge: issue files only). While paused, viewers do nothing: Board refuses stroke/note/note-state (ErrPaused, 409 'The cast is paused.'); OnDoc requires a live segment (recorder and board.Writer drop a record admitted before the pause; Writer answers ErrPaused); one paused rule: Go recording.State.Paused, JS Model.paused; annotate.js disables the rail and holds the note dialog (collab.js noteDialog.hold). Tests: TestDocumentRules, TestBoardRoutesRefuseWhilePaused, TestRecorderEmitDropsARecordDuringAPause, TestWriterRefusesARecordDuringAPause, node notes.test.mjs hold case (default run; fails without held in Save's rule), TestAnnotateLiveTwoViewers under WEBCAST_LIVE_CHROME (rail + dialog off while paused, on after resume; mutation-checked). Operator smoke test pending.; review verdict: SHIP
- The 12:01 recording showed the bug: strokes at 13:26:23 were filed during the pause 13:25:41–13:26:42 (#493 Log).
- **Close review round 1 (FIX-THEN-SHIP):** BR-1 (right): the note dialog's Save and Resolve stayed live while
  paused, including on a pin opened during the pause. Added `noteDialog.hold(reason)` (collab.js): Save and Resolve
  disabled with the reason shown, a draft keeps its text; annotate.js holds it whenever it renders the paused rail.
  TestAnnotateLiveTwoViewers checks Resolve on a viewed note and Save on a typed draft while paused (fails without
  the hold) and both enabled after resume. Minor: `board.Writer.Emit` answers a pause drop with ErrPaused
  (TestWriterRefusesARecordDuringAPause); the recorder's drop comment names the pause. README/atlas updated.
- **Close review round 2:** BR-3 (right): `hold` had only the opt-in Chrome test. Added a default-run node case to
  tests/broadcast-viewer/notes.test.mjs (notes.js re-exports collab.js's noteDialog; webcast/web/collab.js is a
  symlink to it): held draft can't save, says why, keeps text; held view's Resolve posts nothing; release restores
  both. Fails with `held` removed from Save's rule. Minors by rule: comments re-read on every touched function
  (collab.js `held`, annotate.js rail comment now names pause/resume); one paused predicate: Go
  `recording.State.Paused()` (board `pageLocked`, `board.Writer.Emit`), JS `Model.paused` (annotate.js).
  viewer.js's `s.page && !s.live` reads the show-state shared by live casts and replays, not a Model, so it stays.
