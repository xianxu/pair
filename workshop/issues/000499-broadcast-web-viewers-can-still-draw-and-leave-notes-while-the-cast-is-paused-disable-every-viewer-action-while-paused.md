---
id: 000499
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '3feb82ea1dfbfe7b975503bf47334cdce86ec8d7' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-12
