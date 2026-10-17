---
id: 000630
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
card_mirror: '08874e17f671bac7f053b0d85dba2cdc1ff62b6b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T05:43:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# couch session-sync: carry tl-lead's stale-section list to the TL when no event would show it

## Problem

## Spec

Drafted by ariadne:4 (from ariadne#414), filed by the TL.

ariadne#414 makes tl-lead stamp each status-file section and adds `tl-lead status check FILE`, which lists stale sections: Threads and Slots older than 2 h or than the newest landing; Quota older than 12 h; Needs you when unstamped. The TL sees that list only when it runs a tl-lead status verb, which happens on an event. In a quiet stretch, or while the operator holds the lock and writes are queued, nothing prompts it. That is how Threads went 20 h stale on 10-16.

For a registered session-sync file whose owner is a hub, Couch runs `tl-lead status check FILE` at most every 30 min while the file has stale lines and the owner is idle. It sends the output to the owner as one message, and none when the output is fresh. It skips this while the operator holds the lock, then sends once after their submit. It never writes the file.

## Done when

- A fixture file with a stale Threads produces one message to its owner per interval, and none when fresh.
- A held lock suppresses it.
- tl-lead missing or failing produces no message and logs the reason once.

## Plan

- [ ] Add baseline-neutral registered session inspection and a bounded serialized stale-section reminder worker, with tests for cooldown, idle/hub identity, operator locks, command failures and cancellation.
- [ ] Verify real isolated Couch/tl-lead conformance, update atlas/help-facing documentation, and pass the recorded suite and close review.

Detailed design: [stale-session reminder plan](../plans/000630-stale-session-reminder-plan.md). Landing follows the normal SDLC gates.

## Log

### 2026-10-16

### 2026-10-17

- Claimed and started planning in pair:6 after #643 landed. `tl-lead status check --help` confirms that the command owns section budgets and reports without writing. Existing wait-watcher ticks every 15 seconds; external IO must run in a separate serialized worker so it cannot delay wait notices or supervision.
- ARCH-DRY/ARCH-PURE: keep freshness rules in tl-lead; use existing registered-document transaction for an inspection that never acknowledges edits, and existing hub identity, wrapper liveness and broker self-delivery seams. No timer-based inference of idleness.
- Verified TL approval receipt 7792da29-cd2b-4de6-becc-7aeda30b307d: in-memory cooldown may reset on restart, costing at most one extra reminder; isolated real Couch acceptance here is sufficient, installed-hub smoke deferred to restart, no pair:0 staging.
- Verified TL approval receipt 5b8479be-4ca3-41b8-9745-586c7b791da1: full flow approved for the separate serialized worker and safe IO envelope, same feature scope. No code written yet.

- Plan gate PQ-1 requested an authoritative pure transition model; the durable plan now defines advanceStaleReminder and event/effect transitions for every retained state mutation. PQ-2 recurring conformance triggers and the real Git-backed fixture are specified. No implementation has started.

## Revisions

- 2026-10-17: quick-flow sizing exceeded the shell once baseline-neutral inspection and bounded asynchronous command execution were included. TL approved full flow before implementation; product behavior is unchanged.
