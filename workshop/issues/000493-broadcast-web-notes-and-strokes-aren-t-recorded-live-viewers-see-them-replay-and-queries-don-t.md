---
id: 000493
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'f20b77cda8d8e82f9e64c2ac8000146cce0f780b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T14:31:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "3cdae29c", done: "4149f073"}
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

Root cause: `recording.DirWriter` buffered each record file (4 KiB `bufio.Writer`), flushed only when full or at
`Close`. events.jsonl fills its buffer constantly; a cast's few notes (~500 B each) never filled notes.jsonl's, so
they reached disk only when the cast ended. Fix (quick flow):

- [x] `DirWriter.Put` appends each record in one unbuffered write (also crash-safe: whole lines, no lost tail).
- [x] E2E test `TestCastBoardNotesAndStrokesReachDiskWhileLive`: real cast (`StartCast`, its `DirWriter`, the Board),
  a note and a stroke POSTed on the viewer routes, then — without closing — `ReadDir` (replay's reader) holds both
  and `query.Run DIR notes` lists the note. Red before the fix (0 records on disk), green after.
- [x] atlas/webcast-format.md: the unbuffered append rule.
- [x] TL: don't rely on memory for live strokes. `TestLiveStrokeReachesDiskWhileLive` (WEBCAST_LIVE_CHROME, real
  headless Chrome): Board → `Chrome.Emit` → the real recorder (OnDoc check) → `Tee(DirWriter, board)` as
  `recordingOutput` wires it; a stroke during live segment 1 is in pointer.jsonl while the cast runs.

## Log

### 2026-10-12
- **The recording was not missing the notes, they were late:** 20261012T120120-7bc7 now holds 6 note lines
  (12:03:08, 12:03:30, 12:07:50, 13:25:46 + resolve, 13:26:38) and 15 stroke chunks (13:26:23). notes.jsonl and
  pointer.jsonl were last written 14:29, when the cast ended (`stop why=end`). events.jsonl has no note/stroke by
  design (per-kind files: `FileOf`). So at 12:0x the operator read a notes.jsonl still in the buffer.
- Segments: 1 live 12:02:17–12:04:42, 2 live 12:07:31–13:25:41 (pause), 3 from 13:26:42. The 13:26:23 strokes were
  drawn while paused; no stroke from a live segment is in the file. Every Board record goes Board → `Chrome.Emit`
  → recorder (`OnDoc` check) → `Tee(DirWriter, board, live)`, so what other viewers saw was also Put; a viewer's
  own strokes draw locally, so a solo test can "see" a stroke the server refused. Asked the TL whether strokes were
  drawn while live at 12:0x; if so, a second cause sits before the Tee (Board `pageLocked` refusal or the
  recorder's `OnDoc` drop, deliberately uncounted per #470's TestRecorderEmitChecksTheDocument).
- **No second cause (TL's check):** TestLiveStrokeReachesDiskWhileLive passes through the real recorder; with
  dir.go restored to the pre-fix buffered writer (db118ad89~1) BOTH e2e tests fail (cast test: 0 records while live;
  live Chrome: empty pointer.jsonl after 10 s, recorder dropped nothing). So a live stroke is recorded, and the
  buffer was the whole cause. Paused strokes being accepted at all is #499.
- make -k test exit 0 on db118ad89+.
- **Close review round 1 (FIX-THEN-SHIP):** BR-1 (right): a reader racing a write can see a half-written last line,
  and `ReadDir` failed the whole recording on it — now a live hazard. `ReadDir` decodes complete lines only
  (`completeLines`); `Files` keeps raw bytes so `ValidateDir` still flags a torn tail on an ended recording. The
  three "whole lines only" claims corrected (Put, DirWriter doc, atlas). Tests: TestReadDirDecodesCompleteLinesOnly
  (fails with `completeLines` removed), TestDirWriterRecordIsOnDiskBeforeClose (owning-layer pin). Minor: the two
  cast tests share `startRecordedBoardCast`/`open`. Done-when 2 (operator's next live cast) is the pending live check.
