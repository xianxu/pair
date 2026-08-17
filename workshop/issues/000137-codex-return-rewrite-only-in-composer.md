---
id: 000137
status: working
deps: []
github_issue:
created: 2026-08-16
updated: 2026-08-16
estimate_hours:
started: 2026-08-16T21:35:17-07:00
---

# Codex Return rewrite only in composer

## Problem

Pair currently rewrites plain Return in the Codex pane by default, then tries to
discover every menu/permission picker that needs bare Return. That inverts the
safer rule: the multiline rewrite only belongs in Codex's visible composer
box. Codex has multiple user-facing menu systems, so chasing every menu footer
will keep drifting.

## Spec

- For Codex, plain Return should rewrite to LF only when Pair positively
  identifies the live bottom-anchored Codex composer/input box.
- Known overlay/menu markers still take precedence and force the next plain
  Return to pass through as bare CR.
- Positive composer detection should use raw terminal state, not stripped text:
  current rows/cols, cursor visibility/position, and Codex's bottom-band
  composer background `48;2;57;57;57`.
- If Codex composer state is unknown, hidden, or not bottom-anchored, plain
  Return should pass through as bare CR instead of being rewritten.
- Other agents keep their existing Return behavior for this issue.
- Update `atlas/how-to-bring-up-a-new-harness-cli.md` so new harness
  integrations prefer positive composer detection over enumerating every menu
  variant.

## Done when

- Codex plain Return rewrites to LF when the bottom-anchored composer is
  detected.
- Codex plain Return passes through as bare CR when no composer is detected.
- Existing Codex overlay bypass behavior still wins over composer detection.
- Tests cover composer detection from raw ANSI, non-bottom/hidden negatives,
  and Return translation.
- The harness bring-up guide describes the positive-composer model.

## Plan

- [ ] Write the durable implementation plan in `workshop/plans/000137-codex-return-rewrite-only-in-composer-plan.md`.
- [ ] Implement the plan with TDD.
- [ ] Verify focused wrapcmd tests, full Go tests, issue validation, and diff whitespace.

## Log

### 2026-08-16

- Raw Codex scrollback confirms the composer signal: rows near the bottom are
  painted with `48;2;57;57;57`, and Codex leaves the cursor visible at row 36
  col 3 in a 38-row pane while the composer is visible.
- Plan-quality round 1 asked for named unit-tested functions and adversarial
  input strategies; updated the durable plan to name tracker and Return routing
  surfaces explicitly.
