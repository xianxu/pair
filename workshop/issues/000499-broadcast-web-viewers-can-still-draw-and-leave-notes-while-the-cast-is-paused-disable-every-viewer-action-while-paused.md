---
id: 000499
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '568c67f60c11184344bf8433c5556837da5cd2a3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T15:11:40-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
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
- The 12:01 recording showed the bug: strokes at 13:26:23 were filed during the pause 13:25:41–13:26:42 (#493 Log).
