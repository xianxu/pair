---
id: 000737
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours: 6.67
collaboration-mode: collaboration
card_mirror: 'f1cd606d671fc818c16be23372a438043db66b70' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T17:17:42-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
---

# couch: scheduler: run each hub's scheduled tasks from its operations repo (due notices, direct script-run, skip-if-missed, --schedule --status)

## Problem

Recurring operational work has no scheduler: it runs when someone remembers, or on private timers inside pair
(backup freshness, the Drive report). Project couch-scheduling (ops, `teams/devinfra/projects/couch-scheduling/`)
settled the contour with the operator on 10-20; its `design.md` is the reference.

## Spec

- At start and on change, the supervisor reads each hub repo's `scripts/scheduled/<task>/` folders: a Go program
  (`main.go` package, built to a content-keyed executable cache) plus a schedule declaration (cron-like time,
  timeout, scope `machine` default or `team` with an owner,
  optional run-once-if-missed).
- Persisted next-due per task, per machine. Missed runs **skip** by default and wait for the next slot; opt-in
  run-once on start.
- When due: if the task offers `script-run`, run it in the hub repo as the operator, with its timeout,
  process-group stop and bounded output. There is no general sandbox; network is allowed. Tasks declaring
  themselves heavy take the existing machine test lock (0 quiet/logged, 10 wake the hub with output, other
  failure notice). Otherwise deliver a due notice directing the hub to the shared helper for task help.
- Team-scoped tasks run only on the hub of their owner operator; an owner not listed in `team.md`'s Operators
  section refuses with a clear reason.
- `couch --schedule --status`: tasks, scope, last run and result, next due. Run history stays local and bounded.
- Each hub only ever runs its own repo's tasks. Fall-back repeated local times run once; spring-forward missing times skip.
- Offline hub: notify-only due tasks skip/advance; direct tasks execute anyway, dropping offline notices with an explicit status/history reason.
- Per-run records under Couch data `schedule/<hub>/<task>/runs/` retain revision, source, start/end, exit, outcome, notice-delivery state and 256 KiB head/tail output. Reuse diagnostic-log retention as operational data (30 days/300 runs per task); `couch --schedule history <task>` lists runs and prints selected output.
- Manual/help entrypoint: `couch --schedule run <task> -- <args>`, sharing the content-keyed cached Go build and
  execution path with the automatic scheduler (TL clarification, 2026-10-20).

## Done when

- Fake-clock tests pin: due notice delivery, `script-run` exit codes 0/10/other, timeout, skip-if-missed and the
  opt-in run-once, team-owner refusal, a hub never running another hub's tasks.
- Live: a trivial ops task (the landing-plan refresh) runs on schedule for a day and shows in `--schedule --status`.

## Plan

- [ ] M1 — Shared manual task path, durable state, and read-only status; implementation tasks 1–4 in [the durable plan](../plans/000737-couch-scheduler-plan.md).
- [ ] M2 — Automatic per-hub scheduling, authenticated notices, recovery, and day-long live acceptance; tasks 5–6 in the durable plan.

## Log

### 2026-10-20

- Claimed and started in pair:1; planning only. Read the settled ops project/design, the updated Operators format, current Couch composition/notice/protocol seams, persistence/process APIs and relevant lessons.
- TL confirmed team-owner validation belongs in this issue despite the older project reservation: absent Operators refuses team tasks, not machine tasks; no real team task/election added.
- TL clarified cached Go-package entrypoints and a shared manual helper. General sandbox isolation is explicitly out of v1; trusted operator execution/network access, timeout/group cleanup/bounded output, and heavy-only machine admission are the contract.
- Two read-only exploration agents returned code-backed integration digests. Existing broker/service/store primitives are reusable; generic scheduled execution, cached build ownership, occurrence persistence and per-task heavy admission need explicit new adapters. No production code, tests, task declarations, ops artifacts or estimates changed.
- Durable implementation plan written to workshop/plans/000737-couch-scheduler-plan.md for TL/operator review before change-code. It keeps the current live Done when: a staged trivial ops task must run for a day; ops task provisioning and the staged restart require TL coordination.
- One fresh-context plan edit identified six incomplete contracts, now resolved in the revised plan: broker expiry, lost system receipts, watched controller/group lifetime, manual/global execution permits, captured build inputs, and disconnected-hub reservation. Broader fleet drain is explicitly outside the proposed v1 shutdown guarantee. Operator review still precedes implementation; no tests were run for these documentation-only changes.

- Operator approved the plan through ops:0 with the three revisions below. Proceed to change-code and M1; no acceptance requirement was waived.

- Plan gate round 1 held on PQ-1 retention integration and PQ-2 repeated test inventories. Verified actual APIs: diagnostic generations and session transactions cannot collect run bundles directly. Added focused operationalrun adapter plus gcruntime phase/report and compressed test strategies; rerunning the gate before code.

- Plan-quality passed after PQ-1/PQ-2 were addressed; change-code completed with estimate-quality INFO (implementation/rollout allowances may be tight). Task 1 pure declaration/owner/calendar/transition/status files are implemented. Watched the no-op transition stub fail to arm a due slot, then passed focused tests; status exit-pointer alias reproduced red and fixed. Explicit-file kernel suite passed while parallel Task 2 tests were intentionally red. Fake-clock traces pin offline skip/direct execution, dropped notices, exit meanings, stale events, unknown/no-replay, manual schedule independence, and N/2N recurrence counts. Spec/calendar DST and fuzz qualification passed in the bounded subtask. M1 remains open; persistence, runtime execution and CLI wiring are in progress.

- M1 implementation checkpoint: shared manual/controller/build path, immutable per-run operational data and shared GC phase, catalogue/state recovery, status/history and help/skill/atlas integration are implemented. Real disposable Couch acceptance passed help/argv/exit10/history/cache/no-schedule-advance/timeout/group cleanup. It first exposed the parent-deadline EOF race (timeout misreported cancelled), now fixed with native regression; output streaming/backpressure/uncertainty boundaries also passed. Saved-result authority recovery and nullable missing exit were reproduced red and fixed. Root verification: full couchschedule/couchmessage/artifactpath packages PASS; full operationalrun/gcruntime/gccmd PASS; all TestSchedule plus help-cost checks PASS (13.790s); real isolated machine helper lifetime check PASS (1.105s). Catalogue/store race/fuzz and service recovery race checks passed in bounded subtasks. No live operator Couch or :0 was touched. M1 recorded check and boundary review remain; M2 has not started.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; calibration currently marked stale/provisional by estimate-source. Already-scaled implementation hours use 40% of v2/v2.1 ranges; design uses the thorough-spec 0.2 multiplier, plus 0.5 library credit only for cron/kernel and bounded durable catalog/store. Familiarity 1.0; thorough-plan design buffer 15%. Passive day-long live observation is not focused ship time.

Rows follow implementation scope: kernel, catalog/store, operational bundles, GC integration, captured builds, native execution/admission, CLI/status/history, scheduler, broker notices, docs, two review boundaries, and two native API qualifications. The cron library, durablefile/strictjson, broker and machine helper are available; no existing library supplies the operational bundle authority/recovery or frozen Go source view, so those retain full design credit before the spec discount.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: greenfield-go-module design=0.10 impl=0.32
item: greenfield-go-module design=0.10 impl=0.32
item: greenfield-go-module design=0.40 impl=0.32
item: api-integration design=0.20 impl=0.40
item: greenfield-go-module design=0.40 impl=0.32
item: api-integration design=0.30 impl=0.60
item: smaller-go-module design=0.06 impl=0.20
item: greenfield-go-module design=0.40 impl=0.32
item: api-integration design=0.20 impl=0.40
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0 impl=0.20
item: milestone-review design=0 impl=0.20
item: real-api-discovery design=0 impl=0.24
item: real-api-discovery design=0 impl=0.24
design-buffer: 0.15
total: 6.67
```

Design subtotal 2.18 × 1.15 = 2.507; implementation subtotal 4.16; total 6.667, rounded 6.67 focused ship hours.

## Revisions

### 2026-10-20 — clarified execution and owner scope (ops:0)

Original Spec described a Go program named main and execution “under the sandbox and machine test budget.” TL clarified main is an abstract entrypoint: main.go packages are built once per content hash into a cache, and the shared helper is couch --schedule run <task> -- <args>. No general sandbox is promised in v1: reviewed code runs as the operator with network access, timeout, process-group stop and bounded output; only declared heavy tasks acquire the machine test lock. The current Spec reflects that superseding contract.

The project/design previously reserved team scope beyond its owner field. TL confirmed the small deterministic owner check now: scope team requires an owner slug listed under team.md Operators, using lines of the form - <slug> (<name>): …; slug is normalized from git user.name. Missing section is tolerated but refuses a team task. No real team task or election is added.

### 2026-10-20 — operator-approved plan revision (ops:0)

Fall-back repeated local times run once. Remove waiting-hub: offline notify-only tasks skip/advance; direct tasks run regardless, with offline notices dropped and recorded. Replace embedded bounded history with per-run operational data under Couch data schedule/<hub>/<task>/runs, 256 KiB head/tail output, shared diagnostic retention (30 days/300 runs per task), and a history list/output command. Two execution permits are machine-wide. Other plan contracts, including one-minute notice admission and unknown/no-replay, remain approved.

- 2026-10-20: M1 boundary review returned REWORK with six findings (refusal isolation, crash-recoverable cache mutation, persisted-state invariants, concept traceability, runnable skill examples, additive build metadata). M1 remains open. Cancelled this checkout's queued old-head suite through `sdlc local-check cancel`; four other checkout runs were left alone. Fixes and final-head verification precede a new boundary attempt.
- 2026-10-20: TL approved the M2 notice revision after M1 review: immutable execution/output with one atomic, locked, monotonic pending-to-terminal notice finalization; same final is idempotent, conflicting final refuses, uncertain never retries, pending protected from retention. Appended approval and scope to the durable plan; no M2 implementation before M1 passes.
