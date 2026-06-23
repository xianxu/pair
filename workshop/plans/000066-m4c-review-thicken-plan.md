---
issue: 000066
milestone: M4c
created: 2026-06-21
---

# M4c Review Thickening Plan

## Goal

Make the review workbench smoke-testable as a modeful collaboration surface: the
operator can see the current review mode, request a mode switch with optional
instructions, and see when the agent is expected to respond.

## Scope

- Add the `review-<tag>.mode` seam helpers and display labels.
- Show the active mode in the draft review bar and review pane statusline.
- Add a Parley-shaped review mode menu: mode selector plus optional instruction
  buffer, submitted as a poke to the agent.
- Wire the existing spinner helper so user pokes mark the pane as awaiting the
  agent and agent handoffs clear it.

Deferred from the broader M4c bucket until after smoke: `voice:` frontmatter
loading, fact-check fold polish, pending-marker quickfix, diagnostics polish, and
`xx-fix` rename.

## Steps

- [x] TDD the mode seam and label defaults.
- [x] TDD the draft bar's mode display.
- [x] TDD the review pane keymap/statusline/poke behavior for the mode menu path.
- [x] Implement the seam, menu, statusline, and spinner wiring.
- [x] Run focused tests, then `make test-lua`, `make test-review`, and
  `git diff --check`.

## Revisions
