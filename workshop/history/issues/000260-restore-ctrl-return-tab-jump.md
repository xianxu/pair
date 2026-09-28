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

- [x] Reproduce the report using the #251 test and smoke conditions, identifying
  any difference in terminal mode or dispatch state.
- [x] If reproducible, restore the shared dispatch path and add focused
  regression coverage; otherwise record why #251 remains authoritative.
- [x] Run shortcut, notification and integration tests; record smoke evidence.

## Log

### 2026-09-15

Filed as a follow-up to #255 after an apparent recurrence of the behavior fixed
by #251. #251's close record says the shortcut passed tests and operator smoke,
so this issue must first establish whether the current report is a regression.

### 2026-09-28 — Investigation on current main
- 2026-09-28: closed — Operator confirmed Ctrl+Return works after #279 and authorized closure; historical regression resolved by 75cd04a3, no new runtime changes. Fresh go test ./cmd/internal/couchtty ./cmd/internal/couchkeys ./cmd/internal/terminal ./cmd/internal/workbenchshortcut -count=1 passed, including production keyboard dispatch and newest-page/no-notification handling. Documentation-only investigation; no new architectural surface.; review verdict: SHIP

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

### 2026-09-28 — Verification after resuming investigation

- Full uncached suites passed: `go test ./cmd/internal/couchtty
  ./cmd/internal/couchkeys ./cmd/internal/terminal
  ./cmd/internal/workbenchshortcut -count=1`.
- Focused race checks passed: `go test -race ./cmd/internal/couchtty
  ./cmd/internal/terminal -run 'Test(Keyboard|NewestPage|PresenterKeyboard)'
  -count=1`.
- README's Couch keyboard contract agrees with the existing dispatch: answer
  the newest paging Couch thread, acknowledge it, and preserve previous-thread
  navigation. With no pending notification, remain in place and display
  "nothing is paging". Plain Return is forwarded unchanged; terminals without
  distinct Ctrl+Return encoding use Ctrl+Space followed by Return.
- No new runtime change is justified by these results. Pending acceptance:
  in a freshly started Couch, have a second thread produce a notification,
  press Ctrl+Return from the first thread, confirm it selects/acknowledges the
  paging thread, then verify the no-notification notice and ordinary Return.
  Do not close this issue on automated evidence alone while the operator's
  originally reported live behavior remains unconfirmed.

### 2026-09-28 — Restart handoff

- Operator requested a durable checkpoint and a stop because repeated sandbox
  approvals were disruptive; they will restart with the intended Codex flags
  and say "continue on #260". Do not restart the conversation automatically.
- Branch: `000260-restore-ctrl-return-tab-jump` in `pair:0`. Issue is already
  claimed; do not claim again. No implementation or close gate has run, and
  no runtime code changed for #260. Investigation and test results are above.
- `make build` initially failed on sandboxed Go-cache access, then completed
  successfully with permission (including explicit Pair and Couch builds).
  The interrupted build was polled to exit 0; no build remains running.
- Resume with `sdlc state`, read this issue, and check working-tree state.
  Existing evidence identifies #279 (`75cd04a3`) as the already-landed fix.
  Next obtain live notification-jump smoke evidence with the rebuilt Couch;
  distinguish a Couch-thread jump from a right-pane shell/TUI tab change.
  If smoke succeeds, record that #260 was resolved by #279 and run the normal
  close/review workflow. If it fails, capture terminal/key encoding and trace
  production dispatch before proposing any new fix. Do not repeat completed
  investigation merely because the conversation restarted.
- Side tickets #335 (same-conversation relaunch parameter editor) and #336
  (slot-shell `home` alias) are fully filed on remote main through
  `issue move-detail`; their absence in this branch is intentional. Do not
  recreate them or implement them as part of #260.
- Relaunch/park-resume uses the thread's saved arguments, not changed path
  preferences. Switch coding agent can reread preferences but starts a fresh
  conversation even for Codex to Codex. The operator owns the restart choice.

### 2026-09-28 — Operator acceptance

- Operator confirmed the shortcut works now and explained that they had not
  retried it after #279; explicitly authorized closing #260. The historical
  regression was resolved by #279, with no additional runtime change needed.
- Existing production-dispatch regression coverage and the recorded automated
  no-notification/plain-Return checks remain the acceptance evidence for those
  paths; the operator confirmation establishes the reported live jump works.
- All three investigation steps are complete. Preserve the existing Couch-thread
  shortcut semantics; the original right-pane-tab wording does not require a
  new navigation feature.
