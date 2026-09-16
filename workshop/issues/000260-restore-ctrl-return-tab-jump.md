---
id: 000260
status: open
deps: [pair#251, pair#255]
github_issue:
created: 2026-09-15
updated: 2026-09-15
estimate_hours:
---

# Investigate Ctrl+Return notification-tab regression

## Problem

`pair#251` already fixed `Ctrl+Return` notification jumps and closed with
focused tests plus operator smoke confirmation. During later operator smoke
testing after #255, the same behavior appeared broken again: `Ctrl+Return` no
longer jumps to the right-pane tab with a pending notification. This issue is a
regression investigation, not a second implementation of #251.

## Spec

Compare the current path with the #251 implementation and evidence, then
identify whether #255 introduced a regression or the smoke observation came
from a different terminal/input mode. Restore the intended jump only if the
regression is reproducible, without changing ordinary Return handling, agent
passthrough, or the right-pane tab model.

## Done when

- The report is either explained as a false observation or `Ctrl+Return` again
  selects the right-pane tab with a pending notification.
- No-notification behavior remains documented and does not corrupt input.
- A regression test covers the production dispatch boundary.

## Plan

- [ ] Reproduce the report using the #251 test and smoke conditions, identifying
  any difference in terminal mode or dispatch state.
- [ ] If reproducible, restore the shared dispatch path and add focused
  regression coverage; otherwise record why #251 remains authoritative.
- [ ] Run shortcut, notification and integration tests; record smoke evidence.

## Log

### 2026-09-15

Filed as a follow-up to #255 after an apparent recurrence of the behavior fixed
by #251. #251's close record says the shortcut passed tests and operator smoke,
so this issue must first establish whether the current report is a regression.
