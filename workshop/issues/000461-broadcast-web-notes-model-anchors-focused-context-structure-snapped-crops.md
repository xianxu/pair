---
id: 000461
status: working
deps: [000457]
github_issue:
created: 2026-10-11
updated: 2026-10-12
estimate_hours:
card_mirror: 'c46d4499251e069a84ca8018baa046ee274fbc99' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T00:54:11-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "e6ce3ae1", done: "17982254"}
---

# broadcast-web: notes model — anchors, focused context, structure-snapped crops

## Problem

Wave 1 of couch-broadcast-web (ops `teams/devinfra/projects/couch-broadcast-web/project.md`; read its Direction, Development approach and Principles). Depends on pair#457 (recording format and component seams): build against its format, Go interfaces, fakes and fixtures, not against the other wave-1 components, which are being built at the same time. Your component sits behind its #457 interface; test it against fakes and the fixtures, and add contract tests the other side can run. If the format or a seam is wrong for you, don't change it locally: report it to the TL (ops:0), who revises #457's contract. Browser runs are headless and never take focus from Couch. The real test app is the operator's blog (`~/workspace/xianxu.dev`, `npm run dev`).

## Spec

Implement #457's notes-model interface as a pure component over the reducer's DOM at time t (no UI; pointer and notes in the viewer come later):
- **Anchor:** a note at (time, element) stored as node id, a stable selector, the element's text and its box; re-resolved within the same document (notes clear on reload, per the project's Direction).
- **Focused context** (the operator, 10-11): the element, its nearest meaningful ancestor (ARIA landmarks, roles, layout boxes) and nearby text; not the whole DOM.
- **Crop rectangle:** the bounding box of the submitted notes plus a generous margin, snapped to structure (a nav bar or header at full width, a card, a form, a dialog); notes far apart become one crop per cluster. A hook for per-app/URL crop hints, interface only.
- Keep the note shape compatible with pair#455 (terminal notes), so both submissions can share a format.
- A viewer-made note's box is the one the viewer computed at click time (approved with #457, 10-12); the model takes it as given rather than re-capturing it in the operator's tab.

## Done when

- Anchoring, focused context and crops are tested on #457's fixtures and the blog fixture, including clustering and snapping cases.
- The note and context shapes are documented next to #457's format.

## Plan

Durable plan: `workshop/plans/000461-broadcast-web-notes-model-anchors-focused-context-structure-snapped-crops-plan.md`.

- [ ] Selectors: make and match a stable CSS subset (ids, test attributes, nth-of-type paths, ` >>> ` across shadow/frame)
- [ ] Landmarks and focused context (container rule table; 500-char window around the anchor)
- [ ] Crops: pad, per-axis snap, cluster, clip, hints
- [ ] `Structural` model: contract, fixture tests; atlas section

## Log

### 2026-10-11

### 2026-10-12

- Claimed at pair:5 (TL ops:0 dispatch). Seam question for the TL: `Crops` takes `[]NoteAt`, which can't carry a region note's `region`; not built here.
