---
id: 000260
status: working
deps: [pair#251, pair#255]
github_issue:
created: 2026-09-15
updated: 2026-09-28
estimate_hours:
card_mirror: 'a0169c70ed114a8eb9b5141b7790cf1a82282a78' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-28T09:47:22-07:00
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

### 2026-09-28 — Investigation on current main

- Claimed and prepared the issue branch through SDLC. No runtime changes made.
- Historical cause is explicitly recorded in `75cd04a3` (#279): #255 M3
  replaced #251's keyboard reassertion with an initial push on the main screen.
  Zellij then entered the alternate screen, which has its own keyboard stack;
  Ctrl+Return lost its distinct encoding there. #279 pushes/pops keyboard state
  at the screen transition and fixes the test to await actual alternate paint.
- Current `TestKeyboardPhysicalNotificationJump` exercises child resets and
  alternate-screen transitions through the real console input dispatch. Existing
  `TestNewestPage*` tests cover newest-target selection, acknowledgement,
  no-notification notices, exited targets, and switcher handling.
- Ran `go test ./cmd/internal/couchtty ./cmd/internal/couchkeys -run
  'Test(Keyboard|NewestPage|.*CtrlReturn.*)' -count=1`: couchtty passed;
  couchkeys compiled but had no matching tests. This is focused evidence, not
  a full integration or operator smoke claim.
- Terminology mismatch: the existing implementation and public help target a
  Couch notification thread, not a shell/TUI tab within Pair's right pane.
  Preserve that existing dispatch authority (ARCH-DRY, ARCH-PURPOSE); ask the
  operator to confirm the intended target and smoke current behavior before
  closing as already fixed by #279 or proposing another change.
