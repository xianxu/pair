---
id: 000535
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '337ecbaf8392ddce7ac7d8c3a345102356dcf3ed' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T15:20:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "cd594eed", done: "61ed8974"}
---

# main red after #531: launch-failure rendering drops cleanup diagnostics (2 couchcore tests)

## Problem

#531 preserved wrapped errors but replaced their full text with a concise operator line, hiding cleanup failures from three lifecycle tests and diagnostic callers.

## Spec

Restore full launch/cleanup/persistence error text and errors.Is/As access. Keep the existing sanitized one-line explanation and one inspect action at operator presentation boundaries. Durable diagnostics keep separate launch, cleanup and final switch errors. Inventory repair belongs to #533.

## Done when

- All three reported cleanup-diagnostic tests pass without weakening their assertions.
- Wrapped causes remain discoverable and complete in Error text and stored diagnostics; operator output remains one concise sanitized line with one action.
- Investigate the cancellation observation failure under quiet load and record the result; remove timing assumptions if confirmed.
- Full couchcore and make -k test run, with every failure accounted for.

## Plan

- [ ] Reproduce failures, separate full error text from shared operator formatting (ARCH-PURE, ARCH-DRY); sweep launch presentation callers.
- [ ] Test cleanup text, cause identity, stored diagnostics and concise operator output; investigate cancellation timing.
- [ ] Run full couchcore and make -k test, close and land.

## Log

### 2026-10-13

- Quick flow authorized by TL. Three existing cleanup tests reproduce in /tmp/pair535-red.log; cancellation case passed in that run. Root cause: launchFailureError.Error omits CleanupError and truncates the launch cause. No cleanup ownership or lifecycle change intended.
- ARCH-PURPOSE: cover all three cleanup sites plus full-text consumers and concise presentation. ARCH-MOCK: retain existing stateful fixtures. ARCH-CONSTRAINTS / ARCH-FUNERAL: existing bounded diagnostic store unchanged, no new durable family. ARCH-STATE: no product state transition change; cancellation test ordering investigated separately. ARCH-DRY / ARCH-PURE: one pure operator formatter reused at UI/receipt boundaries.

- Quiet cancellation reproduction: load 5.37/13.08/36.13, original test passed 10/10 (4.975s package). The reported failure had observation=false and no state mutation: the caller's 20ms deadline expired before reaching observation. Changed the test to cancel from inside the observer, retaining an outer test-budget deadline and checks for propagated deadline/cancellation and unchanged state.
- Targeted core regression run passed (4.958s); command presentation/show tests passed (0.769s). Added direct full text + errors.Is/As + stored switch final/cleanup coverage and real failed-spawn CLI/receipt/show coverage. Full couchcore running; make -k test waiting on the machine-wide two-suite gate.
- Presentation sweep: menu operation result, continuation notice, command renderError, remote receipt outcome, and reconcile cause use OperatorError. All diagnostic producers retain Error text. No launch or cleanup state transitions changed.

- Full couchcore completed in 421.472s: all four reported #535 tests passed. Three unrelated tests returned workspace host creation is busy: TestAcceptanceReportBudgetTenSlots, TestObserveHealthySlot, TestConvergeWorktreeAddOverAStaleRegistration (/tmp/pair535-couchcore.log). Isolated rerun passed all three in 19.163s (/tmp/pair535-host-busy-rerun.log). Existing hostCreationSettle documents a 100ms inherited-descriptor fork/exec window (#436); contention is plausible, not proven. No lease behavior changed.
