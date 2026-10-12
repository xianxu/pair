---
id: 000475
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '915286de6266aa3b6909c12633633bb336e258f2' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T04:34:24-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "3df9d70f", done: "b5c21435"}
---

# broadcast-web: a segment's linked stylesheets bind after its first paint (viewers see the page unstyled)

## Problem

Found by pair#472 (real recorder fixtures). At a segment's start the recorder emits the `doc` snapshot at once, but a
linked stylesheet's `asset` record only when its body has been fetched, tens of ms later (`rec-stress`: seg at 0,
first frame at 84, `stress.css` bound at 103, photo at 152). The tab was already styled; a viewer, live or replaying,
draws the page unstyled until the record arrives. `rec-stress` frame 0 replays 24% off its screenshot (tolerance 3%).
Every navigation and tab switch repeats it, briefly.

`replay_test.go` lists the frame as a known gap (`knownGaps`), and fails if it starts passing, so the fix removes the
entry.

## Spec

To decide (TL): the recorder holding back the segment's first records until the snapshot's stylesheet bodies are
bound or a short deadline passes, or the viewer holding the previous paint until a new document's stylesheets are
bound. Either way a viewer never draws an unstyled page that the tab never showed.

**Decided (TL, 2026-10-12): the viewer holds the previous paint**, as a browser's render-blocking CSS does. The
recording stays an honest record of when bytes arrived, and the recorder's single-writer queue stays simple. Live and
replay both.

- The renderer (`render.js`, "paint holding") draws a new top document (`head`/`seg`) into a second frame placed
  under the first. The first frame keeps the previous paint on top until every render-blocking stylesheet of the new
  document is bound, or `HOLD_MS` (500 ms) after the hold began; then the first frame is hidden and blanked.
- A render-blocking stylesheet is a `<link rel=stylesheet>` that isn't alternate or disabled and whose media matches.
  The snapshot's own sheets (adopted, owned) are in the model before or with the snapshot (`adopt` of an unknown sheet
  is refused), so links are all that can be late.
- `#page` is always the frame with the current document: `canonical()`, `boxOf()`, the shell's layout and the
  tests read it. The shell and the harness look it up instead of keeping an element.
- The new document's frame is never hidden itself. Chrome throttles a hidden frame's rendering, which starts its
  animations late (measured: rec-blog frame 0 moved 2.92% → 3.22%). Both frames are opaque white, a tab's default
  canvas.

## Done when

- ~~`rec-stress` frame 0 replays within tolerance at its own time and `knownGaps` is empty.~~ (see Revisions)
- A live test shows the previous paint held while a linked stylesheet's bytes are still to come, a print-only link
  blocking nothing, and the page let through after `HOLD_MS` when a stylesheet never arrives.
- `rec-stress` frame 0 stays a known gap, re-reasoned as non-blocking asset lag, with a ceiling (4%) on its best try.
  The test tries it at its hold's release time, where it is 3.55% (24.45% unstyled before the hold), so the frame
  still proves #475.

## Plan

- [x] TL picks the side: the viewer holds the previous paint.
- [x] Implement the hold in `render.js`; the shell and the harness look `#page` up.
- [x] Live test `TestLiveRenderHoldsPaintUntilStylesheetsBind` (mutation-checked: a no-op `holdPaint` fails it).
- [x] Replay test: try a frame at its hold's release (`releaseAt`) when its own times fail; known gaps carry a ceiling.

## Log

### 2026-10-12

- Built the hold; rec-stress frame 0 then went from 24.45% unstyled to blank at 84 ms (held, first segment) and 3.55%
  at the release (103 ms). The rest is photo.png: bytes recorded at 152 ms, image not render-blocking; 2.2% at 152.
  Asked TL (couch); TL chose to keep a re-reasoned known gap with a ceiling (see Revisions).
- The replay test asks for the release time only after a frame's own times fail: replaying ahead warms the asset
  cache and shifts transitions (rec-blog frame 0 2.92% → 3.22% otherwise).
- The harness wires `onDirty` to a scheduled flush as the shell does: a scroll applied before the stylesheet gave its
  box a height was clamped and never redrawn (rec-stress scroller at 0, not 49).
- Side find, fixed: `rewriteSrcset` returns `''` when no candidate has its asset, and `srcset=""` made the img a
  broken one with its alt text, against the renderer's own rule. It now draws no srcset (tested).
- Side find, fixed: `doc.open()` erases a document's listeners, but `openDoc`'s load-redraw listener sat behind a flag
  that survives it, so only the first document a frame ever drew redrew on image or stylesheet loads. Now re-added on
  each open; the `doc.fonts` one (not erased) is added once, both routed to the frame's current renderer.
- Merged main mid-flight: #478 moved the harness's replay into `web/replayer.js` (shared with the fidelity check). The
  hold's harness pieces moved there: `inBrowser` draws into `currentFrame(frame)`, wires `onDirty`, and a fresh
  `Replayer` (or a seek) blanks the held paint (`blankFrames`); `replayAt` waits for stylesheets already in hand
  when holding. `releaseAt` stays in the harness.
- Pre-existing flake, not #475: #478's fidelity check puts rec-blog frame 0 at exactly 6.75% intermittently (main 1/8
  runs, this branch 2/8; otherwise 0.10%). Reported to TL.

## Revisions

### 2026-10-12: Done when amended (TL decision, via couch)

Reason: with the viewer hold, frame 0 at its own time (84 ms) correctly shows the held paint, and at the release it
is 3.55%, the remaining gap being an image whose bytes arrive after the screenshot (not render-blocking). A test
offset reaching 152 ms would pass on main too and stop proving #475. Delta: frame 0 stays in `knownGaps`
(non-blocking asset lag) with a 4% ceiling, tried at its hold's release; plus the synthetic hold test.

