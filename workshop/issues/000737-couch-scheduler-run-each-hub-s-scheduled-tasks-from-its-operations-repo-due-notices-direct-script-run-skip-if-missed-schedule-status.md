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

- [x] M1 — Shared manual task path, durable state, and read-only status; implementation tasks 1–4 in [the durable plan](../plans/000737-couch-scheduler-plan.md).
- [ ] M2 — Automatic per-hub scheduling, authenticated notices, recovery, and day-long live acceptance; tasks 5–6 in the durable plan.

## Log

### 2026-10-20
- 2026-10-20: closed M1 — Manual cached execution, history and status verified in prior M1 checks; BR-8 regression red before fix on refusal, unchanged authority and preserved replacement, now passes with Ledger/Store race coverage (2.181s). Explicit crash/cooperating-process threat model appended; only ledger cell requalified per TL. Recorded affected check queued on pushed head; TL authorized exactly round four, with no ledger waiver.; review verdict: SHIP

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

- 2026-10-20 M1 review fixes BR-1/BR-3: diagnostic-only invalid-name rows preserve routable task checks and isolate malformed declarations through real socket/CLI status, history and manual execution. Invalid filesystem bytes use injective persisted identities; pure/reopen tests cover byte names APFS refuses to create (native EILSEQ is reported as a skip). Phase/source/summary invariants now guard encode, decode and reducer inputs/outputs; OwnerRecorded owns controller identity; cancellation terminates pending notice authority. Race checks for Status/Catalog/StateJSON and Advance/StateJSON/Store pass, plus native manual acceptance and recovery. Root's combined Schedule/Skill/HelpAndSkill/MessageSkill tests pass (9.822s); artifact-path package passes (8.336s). Cache BR-2/BR-6 still being finalized; M1 remains open.

- 2026-10-20 M1 review fixes BR-2/BR-6: TL-directed rename-only cache now recovers aged current/legacy publication and manifest/binary staging, evicts through trash, and rebuilds known corrupt/partial entries under held key leases. Hits touch mtime without metadata rewrite; unknown envelope/input members survive reuse and rebuild byte-for-byte. Fourteen native abrupt process-death checkpoints cover all mutation shapes; young residues count against capacity for one minute, unknown/symlink occupants remain safe refusals. Final inspection found zero-byte partial entries incorrectly treated as absent; task-only/manifest-only/both red regressions now pass with existence tracked independently of bytes.
- 2026-10-20 final M1 fix verification: scheduler/message/artifact packages pass (11.962s/27.002s/6.035s), targeted Schedule/Skill/HelpAndSkill/MessageSkill CLI tests pass (8.567s); after zero-byte correction, cache/manifest/concurrent-build race tests pass (2.059s). Actual machine test-lock conformance passes (0.638s), proving owned cleanup before native slot release. `git diff --check` clean. Generated runtime bundle locally so the affected graph is loadable. Push and recorded final-head check precede the M1 re-review; no M2 code yet.

- 2026-10-20: Interrupted the second M1 review attempt after discovering one missed BR-4 table name (scheduleService versus implemented scheduleCoordinator), before a verdict. No boundary finalized. Corrected the table and audited every delivered core symbol/path before restarting review. Queue the recorded check on the corrected documentation head as well.

- 2026-10-20 M1 round 2: all six original findings disposed; FIX-THEN-SHIP did not finalize because BR-7 (Important, repeated durable-artifact-crash-recovery) found build-capture and ledger staging with defer-only cleanup. Whole writer inventory additionally found operational collection-stage residue. TL approved the complete staging revision: owner-held build scratch slots, exact ledger-temp cleanup under the state lock, collection-stage cleanup under task lock; no journal/worker/loop. Appended the full artifact table and tests. Cancelled only this checkout's queued old-head recorded suite before code changes; other checkout runs remained running. M1 and M2 remain open. Per TL, any further staging gap in the next review is a whole-inventory escalation, not another local patch.

- 2026-10-20 BR-7 ledger slice: native abrupt created/written/synced/pre-rename/post-rename and repeated-crash recovery pass; live-writer cancellation, read-only View, replacement/hardlink/symlink/type/mode/size/inventory/sync refusals are retained. Normal returned-error cleanup also preserves a substituted stage inode. Store/Ledger race tests pass (3.128s), repeated crash subset passes, and manual/catalogue/recovery/status caller checks pass. Build-capture and operational staging slices remain in progress under the approved full inventory.

- 2026-10-20 completed BR-7 sweep: capture slots inherit lifetime leases into both Go adapters; abrupt parent death, live writer, ignored error/GC, concurrent/context-bounded Join, cancellation, path replacement and oversized owned binary cleanup are qualified. Operational orphan journal/run stages recover under their task lock; ledger, cache, run and shared atomic publisher ordinary-error cleanup preserve substituted roots/payloads and verify identities before rename. Returned-error and pre-rename replacement regressions failed before fixes; all pass now. No new journal/worker/retention loop. Root combined packages pass: scheduler 15.311s, operationalrun 3.165s, durablefile 0.116s, messages 26.016s, artifacts 6.420s, GC runtime 1.848s, GC CLI 0.574s; scheduled/skill CLI targets 8.673s; native test-lock conformance 0.535s. Cache race matrix passes 2.070s; helper race suites cover capture/ledger/operational/shared publisher. Diff check clean. Commit/push then recorded final-head check and third M1 review follow; M1 remains open.


- 2026-10-21 04:17 UTC M1 round 3: BR-7 disposed; REWORK on Critical BR-8. A pre-rename pathname substitution is published as ledger authority despite captured-identity protection in the cleanup path, allowing retained authority to disappear with a successful Update. Accepted the finding. The previous whole-family claim missed the publication consumer. Per TL's explicit next-gap instruction, stopped production edits and appended a family-by-operation inventory and proposed proof plan; requesting whole-inventory approval and the proper continuation after three review rounds. No gate waiver/cap change and no M2 work. Current recorded check remains on the pre-correction head.


- 2026-10-21 TL decision: explicit crash/cooperating-process/stale-leftover threat model, excluding deliberate same-user substitution in private 0700 storage. Approved only the cheap ledger publication identity check and three-part regression, with cell-local requalification. Authorized one additional M1 boundary round (cap 4 for that invocation), no waiver; any new crash-class gap returns to TL. M2 remains paused. Cancelled only this checkout's pre-correction recorded check.


- 2026-10-21 BR-8 correction: `TestLedgerPublicationRefusesReplacement` failed before the fix on all three required assertions (nil success, changed previous authority, consumed replacement). Extracted the existing descriptor-relative ledger ownership predicate and reused it immediately before Renameat and in ordinary-error cleanup. `go test ./cmd/internal/couchschedule -run '^Test(Ledger|Store)' -race -count=1` passes (2.181s), including abrupt-death/live-owner/cleanup regressions for this cell; diff check clean. No other matrix cells changed or requalified. Push/recorded check and the TL-authorized fourth boundary round follow; M2 remains paused.


- 2026-10-21 M2 checkpoint: M1 closed SHIP in authorized round four (evidence `3eaef955395b67863f0f8824ba53d587ef8a948c`), BR-8 addressed/no new findings, reported to TL with submitted receipt. M2 adds clock-driven due dispatch, frozen authenticated notices, monotonic finalization/retention protection, shared reserved execution and saved-result recovery. Red/green regressions cover cancellation, stale completion, ledger-finalization failure, startup refusal, unknown history, binding aliasing and bounded retry; counted 8/16-task hub lookups now stay at two. Real broker plus fake clock verifies pending-before-delivery, canonical submitted finalization and no duplicate after refresh. Live one-day acceptance remains required/open; no stage/restart/ops edits. Final combined checks and new pushed-head recorded check follow; the obsolete M1 recorded suite was cancelled through SDLC.


- 2026-10-21 04:56 UTC M2 paused for transaction re-plan: final crash audit reproduced online notify-only and offline skipped record publication preceding the first ledger occurrence commit. Injected BeforePersist failure leaves immutable history; retry one second later fails `immutable operational run already exists` because reconstructed start/end differ. Preserved intentional red `TestScheduleDueHistorySurvivesFailedOccurrenceCommit` (both cases). Appended full transaction inventory and proposed reservation barrier + additive Last revision + passive-history-before-overwrite invariant; requesting TL decision before another recovery correction. No M2 gate/stage/restart/landing or passing recorded suite is claimed. Existing targeted core/operational packages and scheduling integration cases pass apart from this explicit new blocker; final remaining targets are being measured separately with the blocker named.


- 2026-10-21 checkpoint verification: scheduling/skill/help/message-skill targets excluding the explicitly retained `TestScheduleDueHistorySurvivesFailedOccurrenceCommit` pass under race (35.710s). The blocker itself fails in both online/offline cases (0.954s). Core scheduler, operational runs, durable publisher, artifact inventory, GC runtime and GC CLI packages pass (33.991s/44.531s/0.143s/15.091s/11.093s/6.260s); final focused automatic/clock/result/budget race checks pass (2.763s). No passing full-suite/affected record is claimed, and no gate bypass is used. Diff whitespace check clean. Commit/push this known-failing checkpoint before the TL re-plan question.


- 2026-10-21 TL approved the whole M2 transaction re-plan: durable occurrence facts first, history before sole-fact overwrite, additive summary revision. Existing same-ID occurrence history is canonical and adopted without timestamp comparison/rewrite; preserve unknown/no replay. Implementing the barrier and shared occurrence publication/recovery; any further ordering gap at review returns to TL against this rule. M2 remains open, no staging/restart authorized here.

- 2026-10-21 approved M2 correction checkpoint: due reservation/consumption commits before publication; shared occurrence publication adopts canonical same-ID history without comparing timestamps; Last retains its declaration revision; automatic/manual replacement first preserves passive history; unknown classification commits before protected history repair and never replays. Component packages pass: couchschedule 34.886s, operationalrun 51.306s, durablefile 0.313s, artifactpath 16.757s, gcruntime 28.599s, gccmd 2.697s. Broad scheduling race verification exposed task-history damage escaping as a global refusal and mismatching source/revision adoption. Recovery now isolates that refusal to its task while preserving uncertain authority and structural checks; damaged-history/manual-unknown race regressions pass (2.407s). Canonical timestamp and history-outage tests were updated to the approved adoption and existing task-local refusal contracts. Final broad scheduling race run remains in progress; no passing recorded check or M2 boundary/live acceptance is claimed. Real one-day ops task, exit-10/timeout qualification and TL-coordinated stage/restart remain outstanding.

- 2026-10-21 final checkpoint targets: `go test ./cmd/internal/couchcmd -run 'Test(Schedule|Skill|HelpAndSkill|MessageSkill)' -race -count=1` passes (23.369s), including the occurrence-commit regression with no exclusions. The preceding run exposed one remaining old global-error assertion in the scheduled unknown outage test; it now checks a task-local refusal and preserves the original unknown facts on retry. Diff whitespace check clean. Commit/push, then queue the recorded affected check on that head; live acceptance and M2 review remain open.

- 2026-10-21 TL acceptance sequence received and verified: real day uses ops#1 testscratch; wait for #742 staging to clear after #747, finish M2 review/suite, integrate main then stage build. TL owns coordinated restart. Proposed isolated exit-10/timeout qualification with pair#744 native disposable-fleet harness once landed, using scratch hub/tasks only. Code checkpoint pushed at `7d65d5420a16b002097d496691af3ca47a5aaba0`; recorded affected fell back to full suite (missing fresh runtime assets), queued at position 7. Runner session 56669 remains active. No M2 review or stage attempted.

- 2026-10-21 resume: reread Log after operator resumed the exited agent. The old runner session is gone; no matching local attempt/lease or remote status remains for that head, so no pass is claimed. Requeue the recorded check on the current pushed head and continue M2 boundary review, with live acceptance still open under TL's sequence.

- 2026-10-21 TL agreed #739 integration seam (ARCH-DRY/ARCH-ORDER): messageService owns one per-hub automation predicate injected at newAutomaticScheduler construction, shared with scheduleNotice. Consult before consuming each due occurrence and at reserved-script authorization; off records skipped and advances NextDue through existing durable passive history, with no queue/catch-up or immediate run on enable. Explicit manual runs remain independent. Already-running scripts finish; scheduleNotice.Admit consults the same current service-owned predicate before SendSystem, dropping their wake when off. No scheduler-owned automation state. Whichever of #737/#739 lands second wires the concrete getter; no cross-branch development. TL relayed this contract to pair:7; this entry records the agreed future wiring, not an implemented automation switch.

- 2026-10-21 M2 first boundary REWORK: BR-9 admission outlives persisted deadline/committed cancellation; BR-10 task-history errors still globally block startup/consumers; BR-11 canonical summary adoption loses unknown members. Verified named paths and stopped production edits for TL decision. TL approved hub-effect gate then authority lock through bounded broker admission, deadline spanning complete wait, preserving receipt authority; documented ordering/acquirer sweep and independent authority progress regression required. TL directs visible per-task quarantine and complete unknown-field rewrite preservation. Full inventories appended to plan. This is M2's second ordering correction; a third requires stopping for a whole concurrency re-plan. No M2 close, stage or live acceptance claimed.
