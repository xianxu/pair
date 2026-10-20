---
id: 000733
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '69c94a144912c45c78979d022886c650197f7b46' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T14:54:44-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "a85081df", done: "05d0bfec"}
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

- Production-startup regression seeds a valid legacy attention/list.json, starts the real message service, verifies the empty-list notice, observes/clears a fresh event, and verifies legacy bytes unchanged. Passed on the implementation; failed against a temporary overlay of the original persistence path (missing empty-list notice), proving restart regression coverage.
- Local-check affected fell back to suite because its disposable package-graph checkout lacks generated runtime assets; canceled that queued old-head suite to include the new startup test. Requeue on the pushed test-complete head.

- Red/green: original selection left condition rows visited and dismissal re-raised them; both regressions now pass. Added 12h boundary/live-condition preservation, clear-all with keyboard and mouse under overflow, restart reset, fresh-occurrence, and existing-activity-tick expiry tests. Removed persistence/saver path and obsolete store tests.

### 2026-10-20

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
