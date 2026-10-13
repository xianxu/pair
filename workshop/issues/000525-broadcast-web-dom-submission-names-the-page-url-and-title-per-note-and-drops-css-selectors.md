---
id: 000525
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '4ce88a8117bddb169689212c2f7654b578615afa' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-12

## Revisions

- 2026-10-12 — from pair#526 (privacy flag, pair:5): the page URL must be bound to the **captured document** (the one each
  crop and box text came from), including a same-document origin change, not the recording's head URL. Expose it as the
  source identity (URL + title per captured document, referenced by each note) so #526's origin policy reads the same
  value.
