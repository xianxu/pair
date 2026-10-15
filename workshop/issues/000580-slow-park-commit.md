---
id: 000580
status: open
deps: []
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
severity: high
card_mirror: 'c30a43360ae0618929cae7dcc9e8b9d4d180b27b' # card fields mirrored from issue-cards; edit via sdlc
---

# Successful slow park commit is converted into an incomplete park under load

## Problem

During the operator's 2026-10-15 14:57 staged #544/#556 restart acceptance, couch --relaunch ariadne:5 --confirm failed park-incomplete: park requested commit exceeded 1s deadline; transaction still open; recover with retry / recover / abandon. Reported prepare10.9s / operation29.5s. The live wrapper and native Codex remained running; no lifecycle request was published. The success of BeginPark's durable commit is converted into a hard failure solely because its wall time reaches one second.

## Spec

Assess and correct the elapsed-time-only park commit failure policy under machine load. Distinguish successful committed state from a failed/unknown write and caller cancellation. A latency target must not mislabel a committed requested transition as request publication failure when PublishRequest was never called. Keep serialized exact-attempt ownership, honest uncertain outcomes and safe explicit recovery; do not merely raise the timeout or add live-slot retries. Scope the latency budget where it can actually bound work; retain a deterministic slow-success/cancellation/failure matrix (ARCH-ORDER, ARCH-CONSTRAINTS).

## Done when

- A successful slow BeginPark does not create an avoidable park-incomplete row solely from elapsed time; actual cancellation/write failure and uncertain effects remain explicit and safe.
- Tests cover commit below/at/above budget, exact rechecks, cancellation and genuine write failures; no duplicate quit/park attempt and no loss of uncertain authority.
- Retry of a committed requested attempt remains executable, preserving the native conversation; operator acceptance uses scratch only.

## Plan

## Log

Read-only evidence at staged source e13111215270762629c27e860584429b7ae66a98: couchcore/park.go constants parkCommitSoftTarget100ms, parkCommitDeadline1s. parkExpected calls synchronous Threads.BeginPark, records elapsed, and when elapsed>=1s (despite nil commit error), calls AdvancePark(FailureRequestPublishFailed) then returns BEFORE runActiveAttempt/PublishRequest. BeginPark goes through root coordinator/store locks, journal atomic writes and synchronization. This is a post-return elapsed threshold, not a one-second interruptible write deadline. Lock wait, IO and descheduling count; the reported total timings cannot isolate these costs.

Durable /Users/xianxu/workspace/worktree/ariadne-slot5/.couch/thread.json revision107: park nonce park-0d1013c0e43ed049, phase requested, attempt1 not closed/timed-out, failure request_publish_failed / requested commit exceeded 1s deadline. Target helper45960 identity1792101431.210756 remains alive. No lifecycle directory for that nonce exists. Wrapper84122 and child84260 both alive since Oct14 11:08:17, native resume01a136db-6cd1-7a31-87f3-271411613545. Couch peek remains source live / Continue. Park Retry reconciles and drives this existing requested attempt without BeginPark's elapsed threshold; Recover is for unknown/absent/completion evidence and Abandon gives up while leaving running resources.

Existing deterministic TestParkWorkerBeginDeadlineHasZeroExternalEffectsAtOneSecond explicitly pins current999ms-vs1s behavior: this is a production policy defect exposed by load, not a newly introduced #556 regression. Source assessment only; no live park, retry, recover, abandon, signals, lock manipulations or other slot writes performed. At15:04 uptime load37.32/105.71/107.35; not a measurement of incident-time commit IO. Old operation receipt750d54f6-b333-49c5-90ec-13c851ab2703 is unavailable after restart, so no exact commit duration is claimed.
