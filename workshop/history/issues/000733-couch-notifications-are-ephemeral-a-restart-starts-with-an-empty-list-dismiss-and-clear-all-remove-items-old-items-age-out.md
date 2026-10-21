---
id: 000733
status: done
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '016b7ffec82d35ddae3aeab2c505695d097a342d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T14:54:44-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 0.48
---

# couch: notifications are ephemeral: a restart starts with an empty list, dismiss and clear-all remove items, old items age out

## Problem

Couch's "Waiting on you" list keeps every `couch --notify` forever, across restarts: after the 10-20 reboot it showed
11 items going back to 21:03 the night before, all long handled (operator, 10-20, screenshot in ops's session file).
A list that only grows stops being read, which defeats notifications.

## Spec

- Notifications are **ephemeral**: a Couch restart starts with an empty list (they are a nudge, not a record; the
  durable record is the hub's Needs-you section).
- Opening or dismissing an item removes it; a "clear all" control empties the list.
- While Couch runs, items older than a bound (e.g. 12 h) drop off on their own.

## Done when

- Tests pin: restart starts empty; dismiss removes one; clear-all empties; age-out drops an old item.
- Operator live look: after the next restart the list is empty.

## Plan

- [x] Pin restart, select/dismiss suppression and re-occurrence, clear-all keyboard/mouse, notification-only 12h expiry, and live expiry wiring with targeted tests.
- [x] Remove attention persistence/restore/saver, add pure suppression/expiry transitions and panel clear-all, update README/atlas, queue recorded affected verification and request close review. A recorded pass remains required before landing.

## Log

- 2026-10-20: blocker #749 landed in PR #499; resumed merge. Once the landing line became empty, #733 remained the only waiter because trains require a recorded affected vouch before admission, but admission precedes automatic main integration. TL explicitly authorized one manual integration of current main to bring in #749, then push and record the suite while preserving the merge runner. This catch-22 is tracked by ariadne#515. The merge was clean; no gate bypass or unmodified-old-fixture retry.

- 2026-10-20 18:36 PDT: branch full suite passed on 09df3e13 (10m52s, log sha256 ad2e7f04e1ff). Merge integrated main at 13bd7df51123cc82c79953ca2e8423511c0a5b49; hosted conformance passed. Integrated suite attempt 0082d1dc13f7dff65c18d760ca43ba06 stopped in tests/copy-on-select-test.sh: detached copy orchestrator did not finish (4m46s, log sha256 8b7c811b0f82). Isolated fresh-binary TestCopyOnSelectShellFixture also fails hook timeout=3. TL assigned blocker pair#749 separately and explicitly directed stopping the merge runner, preserving this SHIP branch, then re-entering after #749 lands. Merge runner interrupted cleanly; no suite/check bypass.

- Close full review returned SHIP with zero blocking findings. Advisory BR-1 was the atlas/index.md summary still calling attention durable; corrected the summary and captured the index-sweep lesson. Reviewer passed normal/race attention tests and full affected-package tests; authoritative recorded suite remains queued.

- Production-startup regression seeds a valid legacy attention/list.json, starts the real message service, verifies the empty-list notice, observes/clears a fresh event, and verifies legacy bytes unchanged. Passed on the implementation; failed against a temporary overlay of the original persistence path (missing empty-list notice), proving restart regression coverage.
- Local-check affected fell back to suite because its disposable package-graph checkout lacks generated runtime assets; canceled that queued old-head suite to include the new startup test. Requeue on the pushed test-complete head.

- Red/green: original selection left condition rows visited and dismissal re-raised them; both regressions now pass. Added 12h boundary/live-condition preservation, clear-all with keyboard and mouse under overflow, restart reset, fresh-occurrence, and existing-activity-tick expiry tests. Removed persistence/saver path and obsolete store tests.

### 2026-10-20
- 2026-10-20: closed — Targeted Attention/Notify tests and concept/artifact guards passed in couchcore, couchtty, couchcmd and artifactpath. Production startup ignores valid legacy list, observes/clears new event without disk writes; original persistence overlay fails this regression. Select/dismiss/clear suppress live occurrences until clear; only notification rows expire at 12h, including existing activity-tick wiring. Recorded affected check queued, required pass before landing; operator live look at next restart after landing per TL dispatch.; review verdict: SHIP
- 2026-10-20: flow upgraded quick → full — 106 added lines in code files (limit 100)

- Claimed and started planning. TL approved notification-only expiry and episode-scoped suppression; clear-all/open/dismiss do not change the underlying slot state. #730 generated-workflow-only stash dropped as authorized.

## Revisions

### 2026-10-20 14:58 PDT — TL clarifies event versus live condition

- Reason: prompts/questions describe current state, while notifications are events.
- Delta: notification rows expire at 12 hours; live conditions never expire. Selecting, dismissing or clearing a live condition suppresses it until that kind/thread leaves the source set; a later occurrence is fresh. Restart forgets rows and suppressions, then derives current live conditions again. Operator live look is after landing at the next restart, as dispatched.

## Design

Use AttentionList as the one pure transition owner (ARCH-PURE, ARCH-DRY). Remove its restore/store/saver wiring entirely; old namespace attention files become unused and are not rewritten or decoded. No persistent format is changed (ARCH-EVOLVE). Keep the existing 50-item and byte bounds, with the byte encoder remaining an in-memory size guard. Add a Clear all panel row (click/Enter) plus c shortcut. Expire notification rows on the existing activity tick and before new facts/panel opening; no extra worker/timer (ARCH-FUNERAL).

Actions: open, dismiss, clear all, wait, resolve live condition.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
| --- | --- | --- | --- |
| Fresh notification | Listed until opened/dismissed/cleared or 12h | Open or dismiss | Nudge only; hub Needs-you is durable |
| Live prompt/question, even older than 12h | Listed | Open or resolve live condition | None in the underlying slot |
| Opened/dismissed/cleared live condition | Hidden until source clears | Resolve live condition; later occurrence reappears | Current nudge only |
| Couch restart / old list.json present | Empty event list; current conditions re-derived | Review current live conditions | Previous ephemeral nudges intentionally lost |

Cost: N listed items, L current live conditions, S suppressed conditions. Expiry is one O(N) pass per existing activity tick. Source-set reconciliation is O(N+L+S); suppression is removed when a condition clears, so it stays bounded by current live conditions. Count the expiry predicate visits at N and 2N in a test if a helper seam is warranted; directly bound scans in the single filter rather than introducing test-only production counters. No IO on these transitions.

### 2026-10-20 — verification/review overlap

- Reason: tl-worker explicitly permits close review during a queued recorded check; code changes from review require fresh verification.
- Delta: plan row records queueing and review request; completion still requires the recorded pass before landing. Startup regression is in pushed head 09df3e13caf75ac076481a5bb7f84d51d392555a.

### 2026-10-20 — admission-loop recovery

- Reason: the failed pre-#749 fixture prevents train readiness, which prevents the automatic integration that would bring in the fix.
- Delta: TL-authorized one current-main integration, then fresh recorded verification on that combined code before admission/landing. Preserve the existing runner and all review/CI gates.
