---
id: 000458
status: open
deps: [000457]
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '5e5c7776d2e4d1bf5f3302b3423d32713bac1779' # card fields mirrored from issue-cards; edit via sdlc
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

## Plan

- [ ] CDP DOM stream per frame, node-id mapping
- [ ] Observer in the isolated world
- [ ] Segments, tab switch, navigation
- [ ] Plug into webcast.Cast; live check; fidelity run

## Log

### 2026-10-11
