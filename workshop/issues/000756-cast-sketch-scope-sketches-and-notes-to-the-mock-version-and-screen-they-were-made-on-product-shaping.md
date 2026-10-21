---
id: 000756
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '4f582ddb176c77f698c6a906a0a19436b661bab5' # card fields mirrored from issue-cards; edit via sdlc
---

# cast-sketch: scope sketches and notes to the mock version and screen they were made on (product-shaping)

## Problem

product-shaping (ops) iterates mocks by sketching on them in the web cast. When a mock changes version or the user
moves to another screen, earlier sketches and notes should not clutter the new view, and a revert should bring them
back. `couch mock serve` (pair, filed alongside) marks each page with `data-mock-version` and `data-mock-screen`.

## Spec

- cast-sketch and notes record the page's `data-mock-version` / `data-mock-screen` when present (falling back to the
  URL on any other page) with each sketch or note.
- The cast shows only the sketches and notes matching the current version and screen; others are kept, not deleted,
  and reappear when that version or screen is shown again.
- The agent-side view (recording queries and export) includes the version and screen of each sketch.

## Done when

- Tests: a sketch on mock-1/screen A is hidden on mock-2 and on screen B, and reappears on returning to mock-1/A;
  a page without the marks behaves as today (URL scoping).

## Plan

- [ ]

## Log

### 2026-10-20
