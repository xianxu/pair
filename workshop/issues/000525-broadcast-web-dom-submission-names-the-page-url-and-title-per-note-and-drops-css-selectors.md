---
id: 000525
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-13
estimate_hours: 2.544
card_mirror: '0392da6fd224d7fcffadd43e2fbeec7f8ba45ea6' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T10:20:26-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# broadcast-web: DOM submission names the page (URL and title per note) and drops CSS selectors

## Problem

Reviewing pair#483's first real submission (2026-10-12, the operator's blog), the TL found the bundle never says which page the notes were on: `source` carries a tab id and a document number, no URL or title. An agent mapping notes to source files needs the page's route first. Meanwhile every box-text entry and anchor carries a long `nth-child`/`nth-of-type` CSS selector, two formats mixed, which nobody uses: in an agentic development environment the agent edits its own site; it isn't reverse-engineering one.

## Spec

The operator (2026-10-12):
1. **Page URL and title per note**, the cast tab's real URL (e.g. `http://localhost:4321/...`), not the proxied viewer URL. The cast Chrome is local, so take it from the recording's document record (the viewer already shows the page's title and URL) or from CDP at submission time; record which.
2. **Drop CSS selectors** from box text and anchors. Keep what helps find source: the element's tag and classes (the `element` markup line), its text, and the recording handle `@N` (needed for `couch --broadcast-web-query DIR dom --sel @N`).
3. **Keep box text's structure** (one entry per element, reading order); don't merge it into one string: the crop already covers what the user saw.
4. Update `how_to_read` and the README schema example.

## Done when

- A submission from the blog names each note's page URL and title, carries no CSS selectors, keeps `@N`, tag/classes and structured box text; tests pin the schema; the README example matches.

## Plan

- [ ] Implement and verify [the captured-page plan](../plans/000525-captured-page-plan.md).

## Log

### 2026-10-12

## Revisions

- 2026-10-12 — from pair#526 (privacy flag, pair:5): the page URL must be bound to the **captured document** (the one each
  crop and box text came from), including a same-document origin change, not the recording's head URL. Expose it as the
  source identity (URL + title per captured document, referenced by each note) so #526's origin policy reads the same
  value.

### 2026-10-13 — design clarification

Hub approved preserving box_text's existing per-text-node granularity and order, with tag/classes replacing selectors; no flattening or regrouping. One source page (URL, title, provenance) is bound to the capture batch and referenced by each note. #526 landed first; this issue derives privacy from that same captured URL. The durable plan proposes a bounded CDP probe in the admitted context, version-2 output with v1 read compatibility, and explicit unavailable-page behavior. Awaiting hub review of the concrete plan before change-code.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; stale calibration means provisional rates. Scope includes implementation, tests, docs, one close review and staging for the operator's blog check.

Incurred issue/spec design is 0.5h plus 0.2h base implementation. Schema/projection extends existing Go seams (0.3 design, 0.5 impl); CDP capture/fence integration uses the API integration primitive (1.0 design, 1.0 impl). Collector refactor is 0.5 design/0.5 impl, docs 0.1/0.2, guide 0.3/0.3, review 0.1/0.5, and live API discovery 0/0.4. Resolved plan discounts downstream design by 0.2; incurred authoring is not discounted. v3.1 scales every implementation figure by 0.4. Familiarity is 1.0, design buffer 15%. Existing CDP/JSON/URL machinery provides the libraries; no novel runtime or greenfield service is assumed.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.08
item: smaller-go-module design=0.06 impl=0.2
item: api-integration design=0.2 impl=0.4
item: cross-cutting-refactor design=0.1 impl=0.2
item: atlas-docs design=0.02 impl=0.08
item: skill-or-dispatcher design=0.06 impl=0.12
item: milestone-review design=0.02 impl=0.2
item: real-api-discovery design=0 impl=0.16
design-buffer: 0.15
total: 2.544
```
