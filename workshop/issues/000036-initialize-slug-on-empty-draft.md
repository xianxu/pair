---
id: 000036
status: done
deps: []
created: 2026-06-01
updated: 2026-06-01
estimate_hours: 0.5
actual_hours: 0.3
---

# Initialize slug on empty draft

## Done when

- A proposed slug applies immediately when draft line 1 is empty, even if the cursor is on line 1.
- The draft becomes slug line 1 plus blank line 2, with the cursor moved to line 2 for composing.
- Existing protections still hold for structured slug edits, freeform manual headers, and prompt text on line 1.

## Spec

After Codex slugging started working through subscription auth, the first live
proposal did not immediately populate the winbar when the draft buffer was
empty. The proposal existed, but nvim deferred applying it because the cursor
was on line 1. That guard is correct when line 1 already contains editable slug
text, but wrong for an empty draft: initializing the header is safe and should
move composition down to a blank second line.

## Plan

- [x] Let the live reconcile path bypass cursor-row deferral when line 1 is empty.
- [x] Move the cursor to line 2 when applying a slug into an empty buffer.
- [x] Add headless nvim coverage for the empty-buffer cursor behavior.
- [x] Update atlas and run verification.

## Log

### 2026-06-01

- Filed from live Codex session behavior: `slug-proposed-pair` appeared, but
  the winbar did not populate until the user moved off the empty first line.
- Implemented empty-line initialization: live reconcile no longer defers on an
  empty first line, and `slug.lua` moves the cursor to line 2 after adding the
  blank composition line.
- Verification: `nvim -l nvim/slug_test.lua` and `make test` pass.
- Closed: `nvim -l nvim/slug_test.lua` and `make test` pass; empty-draft slug
  apply now inserts line 1 header, blank line 2, and moves the cursor to line 2.
