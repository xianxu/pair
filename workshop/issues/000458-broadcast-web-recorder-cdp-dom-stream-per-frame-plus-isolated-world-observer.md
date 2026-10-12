---
id: 000458
status: working
deps: [000457]
github_issue:
created: 2026-10-11
updated: 2026-10-12
estimate_hours: 4.12
card_mirror: '927ac96d02e7601d5fdbfbd4b47662c279132738' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T00:53:17-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# broadcast-web: recorder — CDP DOM stream per frame plus isolated-world observer

## Problem

Wave 1 of couch-broadcast-web (ops `teams/devinfra/projects/couch-broadcast-web/project.md`; read its Direction, Development approach and Principles). Depends on pair#457 (recording format and component seams): build against its format, Go interfaces, fakes and fixtures, not against the other wave-1 components, which are being built at the same time. Your component sits behind its #457 interface; test it against fakes and the fixtures, and add contract tests the other side can run. If the format or a seam is wrong for you, don't change it locally: report it to the TL (ops:0), who revises #457's contract. Browser runs are headless and never take focus from Couch. The real test app is the operator's blog (`~/workspace/xianxu.dev`, `npm run dev`).

## Spec

Implement #457's recorder interface on real Chrome, through pair#454's browser-control component (`webcast`):
- DOM snapshot and changes per frame from CDP's DOM domain (`DOM.getDocument` pierce, mutation events), cross-origin iframes through auto-attached targets, closed shadow roots; map CDP node ids to the format's stable ids.
- The isolated-world observer #453 needed: CSS rule text, form values (including values set by script), scroll offsets, selection. Nothing in the page's main world unless a measured gap requires it (record why).
- #453's CDP limits: text nodes truncated at 10k characters (Tailwind's runtime `<style>`; use `CSS.getStyleSheetText`), user-agent shadow noise filtered.
- Segments: a full snapshot on go-live, on each new document, on each tab switch (the cast follows the active tab).
- Asset references handed to the asset store's interface (capture and storage are pair#459's).
- Plugs into `webcast.Cast` in place of the stub stream.
- **Format additions approved by the operator with #457 (10-12), not yet in #457's page; add them to `atlas/webcast-format.md` as additive v1 kinds/fields (a Revisions note there), with validator and reducer support:**
  - password values are never recorded: for `input type=password` the `input` record says a value exists, never what it is;
  - a `hover` record: the element under the operator's pointer (hover menus and tooltips are CSS `:hover` state the DOM doesn't hold), plus a sampled operator pointer position, so the viewer can show them and the agent sees hovered-then-clicked.

## Done when

- Casting the blog and #453's stress page produces recordings that #457's validator accepts and its reducer rebuilds; fidelity against tab screenshots (reuse #453's harness) at least matches #453's own-on-CDP prototype.
- Unit tests against fakes; a live check against headless Chrome.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module      design=0.1 impl=0.2
item: greenfield-go-module   design=0.5 impl=0.32
item: greenfield-go-module   design=0.4 impl=0.32
item: smaller-go-module      design=0.2 impl=0.2
item: greenfield-go-module   design=0.3 impl=0.32
item: real-api-discovery     design=0.0 impl=0.24
item: atlas-docs             design=0.1 impl=0.08
item: milestone-review       design=0.0 impl=0.2
item: milestone-review       design=0.0 impl=0.2
item: milestone-review       design=0.0 impl=0.2
design-buffer: 0.15
total: 4.12
```

Items in order: M1 format additions; the pure mirror; observer + Recorder shell; Chrome loop/Cast plug-in and CDP sequence numbers; the fidelity tool; CDP discovery; atlas; three milestone reviews.

> *Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Plan

Durable plan: `workshop/plans/000458-broadcast-web-recorder-cdp-dom-stream-per-frame-plus-isolated-world-observer-plan.md`.

- [x] M1 — format: `hover {id}`, `ptr {x,y}` (TL-coordinated shapes, outside Canonical), `input.secret` and the password rules in reducer + validator; `ValidateDir` reports strays in `frames/` (#457 advisory); format page Revisions
- [x] M2 — recorder: pure mirror (CDP DOM per session → records, path-resolved observer reports), isolated-world observer (sync clicks), Recorder shell, Chrome loop (iframe auto-attach) and Cast plug-in; unit tests on a stateful fake CDP page
- [ ] M3 — live headless checks (stress page, blog) + fidelity tool vs `frames/`, compared with #453's own-on-CDP numbers; atlas

## Log



- 2026-10-12: closed M2 — review round 2: BR-19 Critical fixed (bury after the update), each finding now has a regression test that fails when its fix is reverted (mutation-checked: graveyard, unknown sibling, unknown frame, DOM-only bound, Close losses, unreadable tab stop, one frame doc per seg, frame offsets OOP + nested in-process, Logf reaches recorder); non-DOM events bounded and counted; go test -race -count=5 webcast/... cdp green; WEBCAST_LIVE_CHROME=1 -run Live green; review verdict: FIX-THEN-SHIP
- 2026-10-12: closed M1 — go test ./cmd/internal/webcast/... green; new tests: hover/ptr tracked, reset by seg, outside Canonical (pins unchanged); password value refused in input/attr/ins/kids/doc incl. type flip with value attr; ValidateDir reports frames/.tmp-* and unnamed pngs; review verdict: SHIP
### 2026-10-11
- Claimed and started; plan written. TL (10-12) fixed the shapes: `{k:hover,id}` and `{k:ptr,x,y}` in events.jsonl, reducer-tracked, outside Canonical.
