---
id: 000260
status: open
deps: [pair#255]
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
---

# Restore Ctrl+Return notification tab jump

## Problem

During operator smoke testing after #255, `Ctrl+Return` no longer jumps to the
right-pane tab with a pending notification. The shortcut previously provided a
quick path to the tab that needs attention.

## Spec

Trace the complete `Ctrl+Return` path from terminal input through Pair's chord
dispatcher and notification/tab selection. Restore the intended jump without
changing ordinary Return handling, agent passthrough, or the right-pane tab
model.

## Done when

- `Ctrl+Return` selects the right-pane tab with a pending notification.
- No-notification behavior remains documented and does not corrupt input.
- A regression test covers the production dispatch boundary.

## Plan

- [ ] Reproduce the regression and identify where the chord is dropped or misrouted.
- [ ] Restore the shared dispatch path and add focused regression coverage.
- [ ] Run shortcut, notification and integration tests; record smoke evidence.

## Log

### 2026-09-15

Filed as a follow-up to #255 from operator smoke testing. The former
`Ctrl+Return` notification-tab jump is no longer working.
