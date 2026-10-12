---
id: 000493
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'd816e8ccba32f0cdd1b97437efa90b6396a990d7' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast-web: notes and strokes aren't recorded (live viewers see them; replay and queries don't)

## Problem

Operator smoke test, 10-12 12:0x, Couch a7d9779b (main): viewers left notes and drew strokes on a live cast of the blog and every viewer saw them, but replay showed only the pointer and scrolling. The recording `~/.local/share/pair/couch/webcast/recordings/20261012T120120-7bc7/` has an empty `notes.jsonl`, no `pointer.jsonl`, and no `note`, `note-state` or `stroke` records in `events.jsonl` (which has 1767 records: scroll, hover, ptr, DOM changes, assets, layout, frames). So the Board's records reach viewers but never the recorder's queue, or the recorder drops them (e.g. the OnDoc check: 'the recorder drops a record whose document has gone'). #470 said the Board emits through `Chrome.Emit` → `recorder.Emit`; #458 had noted the Board couldn't reach the recorder without a Chrome-level passthrough. Agents querying a recording see no notes, which breaks the product's core loop (and #483's submission, which reads notes from the recording).

## Spec

Reproduce with a live cast (headless, real recorder, real Board), find where note/stroke records stop (Board emit path, Chrome passthrough, recorder admission, OnDoc document check, file split by kind), and fix the root cause. Add an end-to-end test that fails today: a live cast with the real recorder, a viewer posting a note and a stroke, then the recording on disk holds them, replay shows them, and `couch --broadcast-web-query DIR notes` lists the note.

## Done when

- The end-to-end test passes and failed before the fix.
- The operator's next live cast replays with notes and strokes, and the query CLI lists the notes.

## Plan

- [ ]

## Log

### 2026-10-12
