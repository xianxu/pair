---
type: continuation
slug: couch-scheduler-one-writer
agent: codex
created: 2026-10-21T11:31:05
branch: 000737-couch-scheduler-run-each-hub-s-scheduled-tasks-from-its-operations-repo-due-notices-direct-script-run-skip-if-missed-schedule-status
worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
issues: [737]
---

# Continuation: couch-scheduler-one-writer

## NEXT ACTION

Continue #737 M3 implementation by adding a counted 8/16-task regression for terminal receipt sweeps, then batch those owner transitions into constant full-ledger commits. Due/startup batching is implemented and the due guard is green (56/112 encoded task rows, seven publications at either size); `schedule_owner_dispatch.go` still invokes `o.apply` once per terminal delivery and therefore can reintroduce quadratic ledger encoding. Keep canonical terminal evidence ahead of competing edits/finalization and preserve reservation-before-history. This is unfinished implementation of the approved plan, not a new design or reason to ask for approval.

## State of play

Active issue pair#737 in pair:1 under ops:0. M1 closed SHIP. Operator selected option A after M2's authorized fourth full-model review returned BR-14; TL explicitly retired M2 and approved a genuinely new M3 boundary. The issue Plan and final durable-plan revisions record the re-scope; no M2 review cap reset or gate waiver. M3 change-code accepted plan-quality CLEAN and estimate-quality INFO. Incremental M3 allowance was derived after that result; card estimate is 10.42, not actuals.

Implementation is substantial but incomplete. New namespace-lifetime owner, pure settlement, additive manual intent digests, service-mediated manual commands, fact-only worker, gate-ready actor handoff and live message-service composition are present. All current TestScheduleOwner race tests pass (15.836s), including remote manual execution through the actor. Core scheduler/operationalrun race outcomes and checkpoint SHA are in the issue Log / latest Git commits. There is no recorded full-suite pass for M3 and no running recorded check; exact superseded lease-1820886732 was cancelled successfully. All delegated agents finished. No stage, restart of live Couch, disposable acceptance, live day or merge has occurred.

## Thread arc & user model

The operator wants the small scheduler shipped with reliable facts and explicit uncertainty, not another accumulating recovery mechanism. Repeated M2 history/authority ordering findings led to selecting one actual writer. TL approved the concrete model, bounded completion-time manual output, and a new milestone for the redesign. Continue autonomously through approved implementation; do not re-ask about settled choices or patch the old M2 boundary. TL owns the project file, ops main task, stage/restart coordination and operator acceptance.

## Artifact map

Read the issue Log and final Revisions in `workshop/issues/000737-couch-scheduler-run-each-hub-s-scheduled-tasks-from-its-operations-repo-due-notices-direct-script-run-skip-if-missed-schedule-status.md`, then the final option-A/M3 revisions in `workshop/plans/000737-couch-scheduler-plan.md`. Earlier plan sections and M2 reviews are historical and intentionally preserved; final revisions supersede their concurrency model. The close-gate and m2-review sidecars retain BR-14 and prior dispositions; M3 gets its own fresh full-model boundary.

`cmd/internal/couchschedule/owner.go` adds StateStore.Own -> *StateOwner with Update/ViewDurable/Close, self-copy refusal, retained descriptor and existing permanent flock, context-safe method/Close lifetime, CLOEXEC. store.go factors updateHeld; store_recovery tests now explicitly own/close through the whole M1 ledger matrix. Legacy StateStore.Update/WithAuthority remain temporarily and must be retired/restricted from production once all callers migrate.

`settlement.go` contains pure PlanSettlement(state, canonical *RunSummary, event) Settlement. Supplied canonical is durably confirmed IO evidence; raw ResultRecovered cannot bypass it. Candidate Next actions require publication/finalization before ledger acknowledgement. Catalogue edits are busy across retained Active; canonical terminal history wins first. `manual.go` defines typed intent/request/status, digest, argument/page limits; model/state_json and operationalrun/record add optional digest fields with legacy-absence, strict malformed-field and additive preservation tests.

The new shell is split: schedule_owner.go (owner capability, apply/canonical, catalogue refresh/status, controller authorization); schedule_owner_batch.go (due/startup and retained passive/unknown batches); schedule_owner_dispatch.go (tick/complete; terminal receipt sweeps still per-task); schedule_owner_loop.go (bounded typed commands, 2 execution + 2 notice workers, result retention/retry, joined shutdown and gate handoff); schedule_owner_manual.go (internal submit/observe/cancel). schedule_execution.go is fact-only worker; no ledger/history writer. message_service.go now composes this owner and routes schedule-run operations to handleScheduleRun.

schedule_rpc.go + protocol additions + schedule_status.go route manual CLI over the existing private service socket. Completion output is paged at16KiB under the frame bound, capped256KiB. Submit once; timeout/lost ack prints exact run ID and never resubmits. Public handler configured-root validates and reports uncertain submit/cancel errors. Internal manual methods retain argv in o.intents only after durable reservation and inspection success. A manual reserved row without matching in-memory ID/digest must never execute after uncertainty/restart.

## Live deliberations

Remaining implementation inventory, without new operator questions:

- Batch/count terminal receipt sweeps (and any other O(N) per-tick collection), not only due/startup. Each full-ledger encode is O(tasks); N individual apply calls become O(N²). Current due publication barrier is one batch commit before new history; history adoption follows in another batch when needed. At most two notice candidates advance per tick; remaining tasks retain committed reserved rows without a pending Last. Keep tests asserting occurrence identity/due advancement and two effects, not requiring premature completion for every reserved task.
- Catalogue recovery must repair retained unknown diagnostic history under existing task lock using already committed unknown facts, then adopt canonical evidence, without N ledger commits. Do not recreate absent native completed output. Current refresh canonically reads active records but does not yet run the full unknown repair sweep.
- Enforce the sole writer structurally: remove/restrict old production writer APIs and retire legacy automatic/manual helper implementations (many old tests still call them). Move reusable pure/routing helpers before moving legacy code to test-only or replacing tests with production-owner paths. Add a source/AST dependency guard covering all scheduler ledger writes and PublishOccurrence/FinalizeNotice consumers, not a comment-only claim.
- Finish actor fairness and counted hub lookup work, current automation getter integration under the existing #739 agreement, native crash/transport/group cleanup and history/retention conformance, atlas/help/artifact inventory. Do not reach across unlanded branches. Whichever #737/#739 lands second wires the service-owned predicate and total-order test.
- Run appropriate combined targets and final recorded affected/full check on pushed final head, M3 fresh full-model boundary, then TL-coordinated staging and required acceptance. No full suite should start while implementation has known gaps.

## Decisions & dead ends

M2 is explicitly superseded by the one-writer re-plan (operator, 10-21). Approval deliveries: option A re-plan request `8935cb14-0bb2-47d2-a8db-5325ca4023aa`, concrete plan/M3 approval `29bde47e-ecea-4daf-bf0f-0278b7fa6145`; authenticated submitted. No new journal, queue, election, sandbox or retention worker. Reuse state.lock as lifetime fence and requalify affected M1 recovery. Keep operationalrun task lock, cache/capture leases, machine permits/heavy lock, hub-effect gate, broker and automation locks.

Gate acquisition happens outside the owner. Gate-ready callback queues a typed command and retains gate until definitive acknowledgement, even after cancellation. The owner consumes canonical evidence then validates tuple/deadline and invokes bounded send. Owner never waits for gate or a long worker result. Total runtime wait order is hub gate -> owner turn -> broker -> automation; namespace lifetime flock is a startup fence, never reacquired inside admission.

Live acceptance remains real ops#1 testscratch for one day. Exit10 and timeout qualification use pair#744 disposable fleet with scratch hub/Go tasks, never fake tasks on ops main. TL coordinates :0 staging/restart; no unilateral live change. Milestone review may precede the day under prior TL sequencing; whole-issue close/landing requires acceptance.

## Lessons learned

Use exact occurrence facts and durable evidence; a readable renamed record is not a sync confirmation. One writer still needs settlement ordering after partial persistence. Canonical terminal history must settle before edits, cancellation and gate admission. Duplicate manual IDs need intent identity as well as occurrence identity; legacy missing digest is not evidence of matching argv. Output facts remain immutable and uncertain never replays.

Avoid shared-tree source mutations during concurrent tests; use go test -overlay for mutation probes. Initial component tests temporarily fail compilation while sibling agents introduce APIs; capture a clean behavioral red once symbols exist. Public actor queues must return busy rather than accumulating unbounded caller waiters. Keep completed facts independently from worker capacity; shared ledger failure blocks new dispatch, task-local history failure does not.

Every reported SHA must come from current Git output. Use bare couch/sdlc commands as whole shell calls. Commit trailers: Agent: codex/gpt-6.1-sol/medium and Co-Authored-By: GPT-6.1 Sol <noreply@openai.com>. SDLC owns fresh milestone reviews; no redundant ad-hoc gate reviewer. No recorded check is live, so the worker no-restart-while-own-recorded-check rule does not block this checkpoint.
