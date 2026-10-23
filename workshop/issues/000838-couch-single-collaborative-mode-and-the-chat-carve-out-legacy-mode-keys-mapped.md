---
id: 000838
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '89f8f4158893503d075815b90d209d0f816e2476' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch: single collaborative mode and the @@chat@@ carve-out; legacy mode keys mapped or ignored

## Problem

## Spec

Implements ariadne's single-collaborative-mode spec (filed alongside, from the operator's 10-23 pivot) in Couch,
simplifying pair#823's per-section mode machinery:
- Rounds follow the one collaborative mode; presentation is always highlight for agent edits.
- `@@chat@@` above a heading marks a chat section (numbered inputs always get side-doc replies); Couch recognises it,
  protects the tag as operator-owned, and enforces the chat rule there.
- Legacy keys: `collaboration-mode: chat` (document-wide or per-section map) maps to the chat carve-out; the
  presentation and processing keys and instructions are ignored with one deprecation notice per round; nothing new
  writes them. pair#823's migration command converts old maps to `@@chat@@` tags (operator-run).
- Remove the per-section mode compiler paths that only served the retired keys, keeping the legacy reads above.

## Done when

- A round on a document with a `@@chat@@` section follows the chat rule there and the collaborative rule elsewhere
  (test against the spec's fixtures).
- Documents with legacy keys resolve as mapped, with one notice and no refusal (test; plus a read-only audit of the
  real ops and brain session docs like pair#823's).
- Brain's chat document keeps working unchanged.

## Plan

- [ ]

## Log

### 2026-10-23
