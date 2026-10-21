---
id: 000756
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
---

# cast-sketch: scope sketches and notes to the mock version and screen they were made on (product-shaping)

## Problem

product-shaping (ops) iterates mocks by sketching on them in the web cast. When a mock changes version or the user
moves to another screen, earlier sketches and notes should not clutter the new view, and a revert should bring them
back. `couch mock serve` (pair, filed alongside) marks each page with `data-mock-version` and `data-mock-screen`.
