---
id: 000470
status: working
deps: [000458]
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours: 5.22
card_mirror: '14aa43f624c9c83f81bd498d341c2c8b6e8c699c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T02:54:30-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# broadcast-web: pointer and notes in the viewer — anchored to element and time, multi-viewer

## Problem

Wave 2 of couch-broadcast-web (ops `teams/devinfra/projects/couch-broadcast-web/project.md`). The live viewer (pair#460) shows the cast; viewers can't yet point or leave notes on it. The terminal broadcast already has pointer colors (pair#446), the swatch rail (pair#449) and sticky notes (pair#447); the web cast needs the same, anchored to elements and moments instead of grid cells, using the notes model (pair#461).

## Spec

- **Pointer:** viewers pick a color from the swatch rail (#446/#449's six colors, same rail) and their strokes stream to everyone as `stroke` records (top-page CSS px), drawn over the scaled page.
- **Notes:** a viewer clicks an element (or drags a region) and types a note; the note is a `note` record anchored with the notes model (`notes.Model.Anchor`): node id, stable selector, text, and the box the viewer computed at click time (`couchViewer.boxOf`, approved 10-12). Everyone sees it pinned on its element; resolved notes hide; the operator sees them in Couch (#447's 💭N box). Live notes clear on a new document, per the project's Direction.
- **Region notes:** fix the seam gap pair:5 reported: `notes.Model.Crops` takes `[]NoteAt`, which can't carry a dragged region; revise the seam additively so a region note's region reaches crops.
- Reuse #447's note UI pieces where they fit; desktop Chrome and iPad Safari (touch: tap to anchor, drag to region).
- No submission to an agent yet (wave 3's DOM submission).

## Done when

- Two viewers point and leave notes on the blog cast (once the recorder #458 lands; fixtures and the fake hub until then); notes pin to the right element, survive pause/resume, clear on navigation, and are in the recording.
- Region notes reach crops.
- Tests against fixtures; the operator's smoke test (desktop + iPad), scheduled by the TL.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.04 impl=0.2
item: greenfield-go-module   design=0.1 impl=0.32
item: smaller-go-module      design=0.04 impl=0.2
item: smaller-go-module      design=0.04 impl=0.2
item: cross-cutting-refactor design=0.1 impl=0.3
item: greenfield-go-module   design=0.1 impl=0.24
item: tui-screen             design=0.2 impl=0.4
item: smaller-go-module      design=0.06 impl=0.2
item: tui-screen             design=0.1 impl=0.4
item: atlas-docs             design=0.1 impl=0.08
item: milestone-review       design=0.0 impl=0.4
item: milestone-review       design=0.0 impl=0.4
item: milestone-review       design=0.0 impl=0.4
item: ux-rename-iteration    design=0.3 impl=0.12
design-buffer: 0.15
total: 5.22
```

Items, in order: `NoteCrops` + contract; `board` package (wire, Board, OnDoc, Writer); server routes + admission;
`CastConfig.Board` wiring; `collab.js` extraction from the terminal viewer; `annotate.js` helpers; viewer shell UI
(modes, rail, strokes canvas, pins, dialog, touch); headless two-viewer e2e; Couch 💭N + notes menu for web notes;
atlas; three milestone reviews; a UX tuning round. Design ×0.2 where the plan pre-resolves decisions. Impl at 40% of
v2 ranges.

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* (Calibration source is flagged stale; numbers provisional.)

## Plan

Durable plan: `workshop/plans/000470-broadcast-web-pointer-and-notes-in-the-viewer-anchored-to-element-and-time-multi-viewer-plan.md`.

- [x] M1 — Go: `notes.Model.NoteCrops` (region crops); `board` package (wire, Board as a Sink with an emit func, rules); `POST <cast>/stroke|note|note-state` with admission; `CastConfig.Board`; atlas
- [x] M2 — Viewer: `collab.js` shared with the terminal viewer; `annotate.js`; mode button, swatch rail, strokes canvas, pins, note dialog, touch; headless two-viewer e2e on the blog fixture
- [ ] M3 — Couch: `💭N` after `Chrome ⏸`, #447's notes menu for web notes (copy, resolve)

## Log

### 2026-10-12
- 2026-10-12: closed M1 — go test ./cmd/internal/webcast/... ./cmd/internal/broadcast/... ./cmd/internal/artifactpath green; board -race + fuzz; mutation checks; cast-level note to notes.jsonl + resolve; review round-1 fixes in HEAD; review verdict: FIX-THEN-SHIP

- TL approved the seams (1f98c952): self-chosen viewer id + name (attribution, not authentication; #467 swaps in
  seats); one always-on link; strokes stream as several records sharing an id; the Board follows the stream as a Sink
  and emits through a func (#458 wires it to the recorder queue); `Model.NoteCrops` with region crops, no snap; notes
  allowed while paused, refused after `stop end`.
- plan-quality round 1 (PQ-1, Important): a note checked by the Board, then emitted outside its lock, could be filed
  on a page a navigation brought in between. Fix: the emit seam carries the checked document and the writer
  re-checks at ordering time with one rule (`board.OnDoc`); pending entries end on emit error or a new document.
  `operator` reserved; deps gains #458.
- M1 built. #458 landed mid-milestone (TL, 110c6da0): merged main; `recorder.Emit` gained `Doc` and drops a record
  whose document has gone (the PQ-1 rule now has one statement, `recording.State.OnDoc`, used by the recorder and
  `board.Writer`); `Chrome.Emit` (`webcast.Emitter`) forwards into the queue; `StartCast` makes the board for a
  recorded cast and tees it into the recording beside the directory and the Hub; `Cast.Notes`/`ResolveNote` for
  Couch. `Board.Put` never fails the recording's Tee (a divergence stops the board instead). Mutation checks: pending
  not counted → 40 of 40 concurrent notes land; pending not ended at a new document → the silent-drop test fails.
- M1 review round 1: BR-1 (plan drift) → plan Revisions. Minors fixed: board before the Hub in the Tee; `ErrBroken`
  (503, logged once) instead of a silent ErrStale; Put never fails after Close; the board keeps no strokes; atlas states
  the composed stroke id. Deferred to wave 3 (DOM submission): clamping a viewer-supplied box before `NoteCrops`
  widens the page with it. Not done: a shared contract over `board.Writer` and `recorder.Emit` — the one rule they
  share is `State.OnDoc`, tested in both (`TestWriterStamps…`, `TestRecorderEmitChecksTheDocument`); their difference
  (synchronous ErrStale vs a silent drop) is the `Emit` contract's stated allowance.
- M2 built. `collab.js` extracted (terminal node tests 60/60, broadcast Go tests green; the storage policy test now
  reads collab.js). `annotate.js` + `annotate.css`, mounted from viewer.js by one call; viewer.js otherwise gains only
  `couchViewer.fromPage` (TL b33316a8: #471 edits the same files). `TestAnnotateLiveTwoViewers` (headless, two
  Chromes, B by emulated touch): A's stroke on B's canvas, A's note pinned on its element in B within 2 px, B's region in
  A, both kept while paused, cleared by a navigation, recording reduces with both authors. Chrome needs the sandbox off.
