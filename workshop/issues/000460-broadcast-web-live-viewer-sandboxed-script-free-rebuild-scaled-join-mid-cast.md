---
id: 000460
status: open
deps: [000457]
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: 'fcb51a512d154e60941480c7d5983bd4b0e58b91' # card fields mirrored from issue-cards; edit via sdlc
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

## Done when

- Every #457 fixture replays in the viewer; renders compared to the fixtures' reference screenshots within a stated tolerance (headless).
- Join-mid-cast and segment changes covered by tests.
- The operator smoke-tested it on a desktop browser and an iPad once the recorder is live (ask the TL to schedule it; it may follow the recorder's landing).

## Plan

- [ ] Sandboxed rebuild and change application
- [ ] Scale-to-fit, URL display, mid-cast join, segments
- [ ] Fixture replay tests with screenshot comparison
- [ ] Smoke test

## Log

### 2026-10-11
