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
- The operator smoke-tested it on a desktop browser and an iPad once the recorder is live (ask the TL to schedule it; it may follow the recorder's landing).

## Plan

Durable plan: `workshop/plans/000460-broadcast-web-live-viewer-sandboxed-script-free-rebuild-scaled-join-mid-cast-plan.md`.

- [ ] `model.js` (JS port of the reducer + canonical) with node tests against every fixture pin
- [ ] `render.js` (sandboxed rebuild, neutralizing, asset/CSS rewriting) + `viewer.js` shell (scale, URL, overlays, hover, pointer, `boxOf`)
- [ ] `webcast.Hub` (real `recording.Live`), `Server` `rec` events, CSP; Go tests
- [ ] Headless fixture replay: canonical equality and screenshot comparison with a measured, stated tolerance; join-mid-cast and segments
- [ ] Atlas
- [ ] Smoke test (operator; via the TL, after the recorder lands)

## Log

### 2026-10-12
- Seams confirmed by the TL (delivery 9d90bc7b): `{k:hover,id}` and `{k:ptr,x,y}` (events.jsonl, outside Canonical; #458 emits them); assets at `<cast>/assets/<hash>` (#459 mounts there); the viewer rewrites `url()`s in streamed CSS; `boxOf(id)` for wave 2's note boxes; frame sandbox `allow-same-origin` only, the shell owns input (#466's reverse channel).

### 2026-10-11
