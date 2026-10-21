---
type: continuation
slug: couch-scheduler-m2
agent: codex
created: 2026-10-20T22:22:37
branch: 000737-couch-scheduler-run-each-hub-s-scheduled-tasks-from-its-operations-repo-due-notices-direct-script-run-skip-if-missed-schedule-status
worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
issues: [000737]
---

# Continuation: couch-scheduler-m2

## NEXT ACTION

Run `tl-worker --help`, `sdlc state`, and inspect recorded check session 56669 (if still available). It is the full-suite fallback for pushed code head 7d65d5420a16b002097d496691af3ca47a5aaba0, queued because the fresh affected graph lacks runtime assets. Do not cancel or blindly rerun it. The TL's acceptance sequence is now settled: wait for pair#742's stage to clear after #747, finish M2 boundary review and suite, integrate current main, then `sdlc stage --build` and tell the hub readiness. TL coordinates the restart. Qualify native exit-10 and timeout using pair#744's disposable fleet once it lands, with scratch hub/tasks only. The required real day uses ops#1 testscratch. Keep the issue open through live acceptance; no independent :0 edit/restart.

## State of play

pair#737 is working on its issue branch in pair:1. M1 closed SHIP after the TL-authorized fourth round; M2 has no boundary review yet. All approved transaction corrections are implemented and pushed at code head 7d65d5420a16b002097d496691af3ca47a5aaba0; later commits only record acceptance sequencing/approval. Scheduling/skill/help/message-skill race targets pass without exclusions (23.369s). Component packages also passed; exact evidence lives in issue Log. Recorded suite remains queued, not passing. No stage or live day has occurred. All three former subagents are completed; no active delegated work.

## Thread arc & user model

The user wants a small scheduler that survives crashes conservatively without replay or mechanisms that turn its cache into a database. The latest transaction rule supersedes incremental patches: durable reservation/consumption first, immutable history from durable facts, history before sole Active/Last overwrite, canonical same-occurrence history on retry. They also want fleet-safe acceptance: isolated synthetic qualification, coordinated operator restart, then a real day on testscratch. Do not trade live acceptance for unit-test volume or change live ops tasks.

## Artifact map

Read the active issue `workshop/issues/000737-couch-scheduler-run-each-hub-s-scheduled-tasks-from-its-operations-repo-due-notices-direct-script-run-skip-if-missed-schedule-status.md` and `workshop/plans/000737-couch-scheduler-plan.md`, especially final Revisions. They carry approved recovery, complete transaction inventory, tests and acceptance sequence. `atlas/couch.md` maps implementation. `schedule_records.go` owns passive/unknown history repair; `schedule_automatic.go` commits due barriers; `schedule_manual.go` shares reserved execution; `couchschedule/model.go` adopts canonical summaries with additive revision; `operationalrun/occurrence.go` returns canonical immutable history per ID while retaining strict legacy Publish. Tests alongside these files pin crash-order, canonical winners, unknown stability and no replay.

TL project/task repo is `/Users/xianxu/workspace/ops`; do not edit its project or tasks. ops#1 supplies `scripts/scheduled/testscratch`. pair#744 supplies the disposable native-fleet harness when landed. The TL owns stage/restart coordination and project updates.

## Decisions & dead ends

Canonical same-ID timestamps win; do not compare proposed timestamps or rewrite saved facts/output. Source/revision structural checks remain. Unknown classification commits before repair and preserves its original observation across outages. History damage/lock contention refuses only that task, preserving unknown authority, rather than globally failing healthy hubs. Broad tests exposed old global-error assertions; those now assert task-local refusal while retaining no-replay and stable-fact checks.

M1 threat model excludes a deliberately hostile same-user swapper inside private 0700 data; crashes/cooperating processes/stale partial files remain in scope. M1-only cap4 authorization is consumed and does not apply to M2. If M2's next review finds another ordering gap, stop and bring it to TL against the approved rule; do not add a mechanism. Gate owns fresh-context review, so do not run a redundant ad-hoc review at boundary.

## Lessons learned

A stable occurrence ID cannot repair facts that were never durable. The state lock and commit barrier establish facts; same-ID canonical adoption makes a crash retry converge. Preserve the distinction between state-store failure and isolated task-history failure. Keep pending/unknown history protected and execution/output immutable while notice finalization is monotonic.
