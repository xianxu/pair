---
id: 000607
status: open
deps: []
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
severity: critical
card_mirror: '50f84b1b342dba75b25b54a7532f7828981fee5a' # card fields mirrored from issue-cards; edit via sdlc
---

# Reboot of a legacy slot leaves the old wrapper tree running

## Problem

`couch --reboot` of a legacy slot (no proven binding) starts the fresh wrapper but leaves the old process tree running: pair:1 (wrapper 86287, couch-9d6fdf5eecace248), pair:6 (34570, 1-pair-47) and ariadne:3 (94921, 1-ariadne-32) kept wrapper, codex, session-watch and engine; #587's monitor flagged a 363 MB orphan; the TL SIGTERMed them by hand. Cause: in executePreparedRecovery, a Fresh selection whose confirmed-terminal preparer fails (legacy readiness/attribution) proceeds with an EMPTY stop set ("unproved remnants left untouched").

## Spec

Operator rule: reboot = kill everything belonging to the slot, then start. A reboot's stop set always includes the slot's live wrapper from Couch's message registry (Binding.PID + Binding.Start for the thread's scope+tag), verified live by exact identity, plus its descendants (agent, session-watch), each by exact PID+identity: SIGTERM, then await exact exit. The slot's zellij session is killed by its exact recorded name and confirmed absent, only when no other live slot's record names that session; otherwise it is left and the result says so. All of it appears in the reboot confirmation's stop list. The only new refusal: the kill is not confirmed. No registered wrapper: behaviour unchanged. (TL approval 016cce48.)

## Done when

- Regression: a legacy slot with an unproven binding (preparer errors) and a registered live wrapper, rebooted, signals and confirms exit of the wrapper and its descendants and kills its own session; nothing old remains. Shared-session and no-wrapper cases leave things untouched; unconfirmed exit refuses.
- Targeted tests, recorded suite, close review; land and report.

## Plan

- [ ]

## Log

### 2026-10-15

Filed from the TL's report (priority 1: slot startup); design approved 016cce48.

### 2026-10-16 — build (pair:2; TL approvals 016cce48, b6cb4e9c)

- Stop set: `withSlotTree` (couchcore/recovery_slot_tree.go), Fresh only, added after the park-helper block: the registry wrapper (new `Couch.SlotWrapper`, wired in couchcmd from the broker binding via the live-restart probe's message service) verified Live by exact identity, its descendants from one `procutil.Table` snapshot (new `Couch.ProcessTable`), and the slot's own zellij session via `QuiesceNamed` unless another live slot's record names it (then left, with a note). Retire: SIGTERM each, await exact exit, then the session; refuses "kill not confirmed" otherwise. No registered wrapper: unchanged. All entries appear in the confirmation's stop list.
- Tab bar: `statusModelLocked` gives no chip to a pane whose address has no inventory row when its tree is a slot's: a slot's chip is its current row, as in the switcher. The old pane itself goes once its client exits, which the session kill now causes.
- Tests: TestRebootLegacySlotStopsRegisteredWrapperTreeAndSession, TestRebootWithoutRegisteredWrapperIsUnchanged, TestRebootRefusesWhenSlotTreeExitIsNotConfirmed, TestSlotSessionSharedWithAnotherLiveSlotIsLeft, TestRebootedSlotShowsOneTabChip; the legacy, unconfirmed and tab-chip tests RED without the change.
- Package run under machine load: TestParkCoordinatorCoalescesExplicitRecoveryAndRetry and TestWorkbenchPerformanceIdleUITick failed once; both pass in isolation (5x, 2x) and touch nothing changed here.
