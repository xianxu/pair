---
id: 000301
status: open
deps: []
github_issue:
created: 2026-09-20
updated: 2026-09-20
estimate_hours:
---

# Sync renamed thread labels to tab bars

## Problem

Renaming a thread in the switcher changes the switcher's label but can leave
the active Pair/Zellij tab bar showing the old name. The two user-facing
surfaces then disagree about which thread is open.

## Spec

When a thread rename is accepted in the switcher, propagate the canonical new
label to the currently attached tab bar and any other tab-bar presentation of
that thread. The rename must use the same durable identity and persistence
path as the switcher, so a reload or reattach cannot restore the stale label.
If the tab-bar update fails, preserve the durable rename and report the
presentation failure without reverting the accepted name.

## Done when

- Renaming an attached thread updates its switcher row and tab-bar label in
  the same operation.
- Reloading or reattaching the renamed thread presents the new label.
- Renaming a non-active or detached thread does not alter another thread's tab
  bar.
- A tab-bar presentation failure leaves the durable rename intact and produces
  an actionable diagnostic.
- Tests cover accepted rename, persistence/reload, detached or non-active
  threads, and the presentation-failure path.

## Plan

- [ ] Trace the switcher rename, durable record, attach/reload, and tab-bar
      label owners to identify the canonical propagation boundary.
- [ ] Propagate the accepted name through that boundary with explicit failure
      handling.
- [ ] Add regression coverage for active, detached, reload, and failure cases;
      update operator documentation if the rename behavior is user-visible.

## Log

### 2026-09-20

Created from the operator report that switcher renames leave the tab bar stale.
