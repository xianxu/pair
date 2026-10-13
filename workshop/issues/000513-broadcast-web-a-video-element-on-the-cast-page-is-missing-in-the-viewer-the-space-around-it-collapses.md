---
id: 000513
status: codecomplete
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'd2459a69decfdc32f9423d3d1f9d2ce9696df0f3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T18:21:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 0.61
---

# broadcast-web: a video element on the cast page is missing in the viewer (the space around it collapses)

## Problem

The operator, 10-12 ~18:30, casting the parley blog in the cast Chrome and watching in Safari: after "How does it work? One video is worth a thousand words. Take a look." the cast page shows a demo video (a poster of a terminal session, a play button, a caption "navigate away while waiting for answer"). The viewer shows nothing there: the next heading ("What Parley isn't") follows directly, with no gap, so the element is either not recorded, not rebuilt, or rebuilt with zero size. Screenshot: the TL scratchpad image 117 (left: cast Chrome; right: Safari viewer).

## Spec

Find where the video is lost (recorder snapshot or mutation, asset fetch for its poster or source, or the viewer's rebuild), with a test that reproduces it from a page holding a `<video>` with a poster (and, if the blog uses one, the embed's actual markup). Fix so the viewer shows at least the element at its size with its poster frame; playing the video in the viewer is out of scope unless it falls out. Check the other replaced elements the same way (`<iframe>`, `<canvas>`, `<picture>`/`srcset`, `<object>`) and fix or file what the test finds. Measure the real page first (the operator's cast tab, read-only; never steal focus).

## Done when

- The parley blog's demo video shows in the viewer at its place and size, with its poster.
- A test with a video page fails before the fix; the replaced-element sweep is recorded in the Log.
- (Revision) What the recording can't hold (canvas, video, object/embed, an unrecorded frame) keeps the operator's
  live size in the viewer (also a video sized only by metadata, the operator's iPad case; also inside shadow roots;
  never un-hiding a hidden one) and carries a "kind: not captured" mark that follows its size.

## Plan

- [x] Recorder: `DOM.childNodeCountUpdated` for children the mirror doesn't hold → `Expand` (`DOM.requestChildNodes`
  → `setChildNodes`). Root cause, measured below.
- [x] Viewer: a drawn `<canvas>` takes the operator's live layout box (else its attributes) at zero specificity
  (`drawLiveSizes`, since generalized to every uncaptured kind): with scripts off a canvas draws its empty fallback (0x18, measured in the sweep).
- [x] Revision: placeholders for what the recording can't hold (canvas, video, object/embed, an unrecorded frame):
  `Renderer.uncaptured()` + `placeholders.js` (dashed outline + "kind: not captured" at the drawn box, redrawn on
  every record, so it follows the live size). #515 (canvas snapshots) filed unassigned.
- [x] Tests: TestMirrorExpandsANodeFilledAfterItsInsert (default run, red before), TestLiveReplacedElements
  (WEBCAST_LIVE_CHROME: video, iframe, canvas resized live to 400x100, picture, object, a node filled after its insert;
  sizes match the cast, labels exactly canvas/object/video; fails without the recorder fix and without the canvas
  sizing), tests/webcast/placeholders.test.mjs (default run).
- [x] atlas/webcast.md, README.

## Log

### 2026-10-12
- 2026-10-12: closed — make -k test exit 0 (no FAIL lines) on 4ffda71f + main. Round-1 fixes: liveSizeRule never applies to [hidden] and gives nothing to an element the operator's layout didn't render; rules live in a per-root live-size sheet (document or shadow root) adopted before the page's sheets; expansions de-duplicated while queued; pure liveSizeRule/isUncaptured with default-run node tests (tests/webcast/placeholders.test.mjs, 5 pass). TestLiveReplacedElements (WEBCAST_LIVE_CHROME) adds a hidden canvas (stays display:none, unmarked) and a shadow-hosted canvas (200x60, marked); routing rules to the document fails it (mutation-checked). Root cause and earlier evidence unchanged: childNodeCountUpdated expansion (TestMirrorExpandsANodeFilledAfterItsInsert), the real blog embed shown with poster frame, the operator's iPad video at live size marked 'video: not captured' (its mp4 isn't recorded). #515 filed unassigned.; review verdict: SHIP
- 2026-10-12: flow upgraded quick → full — 190 added lines in code files (limit 100); an earlier round of this close already ran the full review
- **TL, from the blog source (~/workspace/xianxu.dev, read-only):** it is not a `<video>`. The post has
  `<div class="cast-embed" data-cast="/casts/parley-nvim-v1.cast" data-poster="npt:0:41"></div>`;
  `src/components/blog/CastEmbed.astro` hydrates it lazily (IntersectionObserver, 600px margin) into
  asciinema-player 3.17.0 with `fit:'width'`, the player's CSS imported dynamically, plus a JS-placed caption
  overlay. The player renders `div.ap-term` = `<canvas>` (background and colour blocks via `getContext('2d')`) +
  `<svg class="ap-term-symbols">` + `<pre class="ap-term-text">` of spans. Check: canvas pixels (never in the DOM),
  the dynamically inserted stylesheet, and the lazy mutation after the first snapshot.
- **The operator's recording 20261012T171139-ddad, measured:** both full-document captures (`kids` at 621634 and
  2639691) hold `div.cast-embed` (with `data-mounted`) and no children; at 2557975 an `ins` adds an empty `div` and
  the caption, and nothing is ever inserted into that `div`. The player's CSS (dynamic `<style>`) is recorded.
- **The live page (separate headless Chrome, read-only GETs to the dev server):** `div.cast-embed > div >
  div.ap-wrapper > div.ap-player > div.ap-term{canvas, svg, pre}`, controls and the play overlay, 720x640, plain DOM
  (no shadow root). Through the real recorder: the inner `div` recorded, none of its children; no drops, no desync.
- **Cause:** CDP sends `DOM.childNodeInserted` only under a parent whose children it has sent. The inner `div` came
  as an empty insert, so when the player filled it Chrome sent only `DOM.childNodeCountUpdated`, which the mirror
  ignored. With the fix the whole player subtree is recorded and the viewer shows the embed at 708x629 (cast 720x640
  at a slightly different width) with its poster frame, play button, caption and Screenkey box; the canvas-drawn
  cursor and background tint are the only difference (canvas pixels: #515).
- **Replaced-element sweep (TestLiveReplacedElements):** video (poster), iframe (recorded), picture/srcset, object:
  shown at the cast's size. canvas: collapsed to 0x18 in the viewer (scripts off → fallback) → fixed above.

## Revisions

- 2026-10-12 (operator via TL): scope addition. Any content the recorder can't capture (canvas pixels, video frames,
  cross-origin iframes, plugins) shows in the viewer as a placeholder at its live rendered size, updated when the
  size changes, with a small label (e.g. "canvas: not captured"), so layout never collapses and agents know
  something is there (the rrweb/FullStory practice). A low-rate canvas snapshot (toDataURL when not tainted) is a
  follow-up, filed unassigned as #515, not built here.
- **Operator's second case (TL): the ariadne-2-5 post's iPad video** (`<video src="/videos/couch-demo-viewer.mp4"
  controls preload="metadata">`, no width/height; sized by metadata) had no box in Safari. Measured on the page
  (headless, read-only): the recorder stores no `.mp4` asset, so the viewer's video has no `src` (it never loads)
  and is marked "video: not captured"; its size must come from the live box. Live sizing now covers every
  uncaptured kind, not just canvas, border-box (iframe was 4 px over) and with the box's `aspect-ratio` (the page's
  CSS sets `height:auto`, which beat the rule's height: 500x150 against the cast's 500x375). The marks are drawn on
  every paint too (a replay applied before they mounted drew none). TestLiveReplacedElements adds a video sized only
  by metadata from a source the viewer can't load (a MediaRecorder clip, `height:auto`): without the aspect ratio it
  shows 346x150 against 346x174 and fails.
- **Close review round 1 (FIX-THEN-SHIP):** BR-1 (right): the rule's `display:inline-block` (author origin) beat the
  UA's `[hidden]{display:none}`, and a hidden canvas fell back to its attributes. Fixed by class ("the viewer's sizing
  never un-hides"): `:not([hidden])`, and once a layout has arrived an element without a box gets no rule. BR-2
  (right): document rules don't pierce shadow trees; rules now go to a live-size sheet per root (`getRootNode()`),
  adopted before the page's sheets. BR-3: Done when gains the revision's bullet. Minors: expansions de-duplicated
  while queued; stale names fixed; pure `liveSizeRule`/`isUncaptured` with default-run node tests. Live test adds a
  hidden canvas (stays `display:none`, unmarked) and a shadow-hosted canvas (200x60, marked); routing rules to the
  document instead of the root fails it (mutation-checked).
