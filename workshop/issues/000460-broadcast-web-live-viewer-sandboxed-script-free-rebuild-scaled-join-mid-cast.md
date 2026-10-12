---
id: 000460
status: working
deps: [000457]
github_issue:
created: 2026-10-11
updated: 2026-10-12
estimate_hours:
card_mirror: '9dc9993518206e425a782d4e752bdb169ba69bd3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T00:53:33-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "51261daf", done: "e4ee71b2"}
---

# broadcast-web: live viewer — sandboxed, script-free rebuild, scaled, join mid-cast

## Problem

Wave 1 of couch-broadcast-web (ops `teams/devinfra/projects/couch-broadcast-web/project.md`; read its Direction, Development approach and Principles). Depends on pair#457 (recording format and component seams): build against its format, Go interfaces, fakes and fixtures, not against the other wave-1 components, which are being built at the same time. Your component sits behind its #457 interface; test it against fakes and the fixtures, and add contract tests the other side can run. If the format or a seam is wrong for you, don't change it locally: report it to the TL (ops:0), who revises #457's contract. Browser runs are headless and never take focus from Couch. The real test app is the operator's blog (`~/workspace/xianxu.dev`, `npm run dev`).

## Spec

Implement #457's viewer contract as the live viewer page (replacing pair#454's stub page):
- Rebuild the DOM from the stream in a sandboxed frame without `allow-scripts`: no script or event handler from the recorded page runs.
- Apply changes by node id; handle iframes and shadow roots as the format represents them.
- Lay out once at the operator's viewport and scale to fit the viewer, never relayout (same rule as the terminal broadcast); show the page's URL.
- Join mid-cast from the latest snapshot; survive segments (new document, tab switch, pause).
- Assets resolved through the proxy's URLs (pair#459); against fixtures, a static fixture server stands in.
- Works in desktop Chrome and iPad Safari.
- Show the operator's hover (the `hover` record, added by pair#458) as a stand-in for `:hover` state, and the operator's sampled pointer position.
- A note made in the viewer carries the box the viewer computed at click time (its layout equals the operator's, only scaled), not a later capture in the operator's tab (approved with #457, 10-12).

## Done when

- Every #457 fixture replays in the viewer; renders compared to the fixtures' reference screenshots within a stated tolerance (headless).
- Join-mid-cast and segment changes covered by tests.
- ~~The operator smoke-tested it on a desktop browser and an iPad once the recorder is live.~~ Moved to a follow-up the TL schedules after #458 lands (see Revisions).

## Plan

Durable plan: `workshop/plans/000460-broadcast-web-live-viewer-sandboxed-script-free-rebuild-scaled-join-mid-cast-plan.md`.

- [x] `model.js` (JS port of the reducer + canonical) with node tests against every fixture pin
- [x] `render.js` (sandboxed rebuild, neutralizing, asset/CSS rewriting) + `viewer.js` shell (scale, URL, overlays, hover, pointer, `boxOf`)
- [x] `recording.Hub` (FakeHub promoted, TL decision), `Server` `rec` events + Presence, CSP; Go tests
- [x] Headless fixture replay: canonical equality and screenshot comparison with a measured, stated tolerance; join-mid-cast and segments
- [x] Atlas
- [ ] ~~Smoke test~~ moved to the TL's follow-up after #458 (see Revisions)

## Revisions

### 2026-10-12 — smoke test moves after #458 (TL, delivery 16887036)
- Reason: the desktop + iPad smoke test needs a live recorder, which lands with #458.
- Delta: `## Done when` drops the smoke test, and `## Plan` marks it moved. The TL runs it with the operator together with #458's first live cast. #460 lands on the headless evidence.

## Log

### 2026-10-12
- Seams confirmed by the TL (delivery 9d90bc7b): `{k:hover,id}` and `{k:ptr,x,y}` (events.jsonl, outside Canonical; #458 emits them); assets at `<cast>/assets/<hash>` (#459 mounts there); the viewer rewrites `url()`s in streamed CSS; `boxOf(id)` for wave 2's note boxes; frame sandbox `allow-same-origin` only, the shell owns input (#466's reverse channel).
- #466 additions (delivery d44edcc4): `?viewer=ID` presence hook on `events`; shell `nodeAt`/`toPage`; `ctl` rendering waits for #467 (hook left). Folded into the plan's Revisions.
- `model.js` matched every fixture pin's hash on the first run: the JS port of `Reduce`/`Canonical` emits Go's exact bytes (one-space indent, HTML-escaped `<>&`).
- Renderer bugs found by the headless replay, each fixed at the cause: Chrome's `Document.replaceChildren` checks a new doctype against the root it is about to drop (empty first, then append); a frame window keeps its scroll across documents (reset per segment); `scroll-behavior: smooth` animated replayed scrolls behind the tab (scroll `instant`); a scroll applied before images and fonts loaded was clamped and never re-applied (the renderer re-flushes on load events and `loadingdone`).
- Render fidelity (`replay_test.go`, `WEBCAST_LIVE_CHROME=1`): canonical equal to the reducer at every pin of all four fixtures; renders 0.2–2.3% of pixels off `frames/` (tolerance 3%, pixel = a channel off by >48/255). Two timing effects are the fixtures', not the viewer's, and the test models them instead of loosening the tolerance: transitions (the last 1 s before a frame is replayed in real time, then animations pause) and the prototype's polled observer (scroll records trail the tab by up to 150 ms; blog t=5810 is 12.9% at t, 2.3% at +150 ms). Set `fixtureObserverLag` to 0 once #458's recordings replace the fixtures.
- `boxOf` against the tab's own layout boxes (`viewer_live_test.go`, blog at t=10041): 45% at first. Causes: entry animations mid-flight when measured (the test finishes them), and an image the tab never loaded (lazy, below the fold) drawn as broken with 72px of alt text. Now an image with nothing to load is drawn as unloaded (no alt; alt returns with the asset). Result: 215 of 217. Out: Astro's dev toolbar (its shadow content was never recorded) and one SVG shape whose stroke adds 2.02px (DOMSnapshot boxes include half the stroke; SVG shapes get 2px).
- Recorder gaps passed to the TL for #458: blog's `astro-dev-toolbar` shadow root recorded empty; scroll records lag up to ~150 ms.
- TL decision (delivery a3360305): `recording.FakeHub` promoted to `recording.Hub`, the production fan-out, with `FakeHub`/`NewFakeHub` kept as deprecated aliases for the parallel wave-1 branches; #458 constructs it next to its `Tee`.
- No README/CHANGELOG entry: until #458 feeds the Hub, viewers still see only the live state, as the README says.

### 2026-10-11
