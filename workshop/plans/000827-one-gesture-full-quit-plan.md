---
collaboration-mode: discussion
---

# One-gesture full quit implementation plan

> **For agentic workers:** Follow AGENTS.md §3; use the existing lifecycle fakes and shared disposable fleet. This is one atomic landing, with one close review, no milestone tags.

**Goal:** One confirmed switcher Alt+x parks the selected fleet, automatically handles safe stragglers within one budget, or gives one exact unresolved-slot report.

**Architecture:** Retain the thread gate, park transaction/nonce, recovery protocol and half-CPU fan-out. Add a bounded park-leave pass around existing per-slot work; it never treats timeout as absence or uncertain publication as permission for a fresh transaction. Detach retains its behavior.

**Tech stack:** Existing Go context/clock/store, stateful Pair lifecycle fake, Console reducer and testcouch; no new dependency or persisted format.

## Evidence and boundaries

- park.go:20 defines a100ms diagnostic soft target and1s commit threshold. parkExpected times synchronous BeginPark and checks AFTER the write; an overrun records failure and rolls back before publishing quit. It is not a cancellable1s IO timeout.
- Leave currently snapshots once, drives one attempt per address at LifecycleParallelism, preserves partial progress, and joins errors. Console stops only on success; another Alt+x repeats the fleet pass.
- Normal completion wait is15s. Existing Recover reconciles completions and requires the exact session before another trigger; worker address+nonce coalescing prevents duplicate active ownership.
- parkCommitDeadline also bounds relaunch's diagnostic observation. Separate that existing1s observation timeout before changing the park commit threshold; do not silently extend relaunch.
- The operator's loaded30-slot incident is an unverified causal lead. Neither source inspection nor a fast private run proves that the old threshold caused that particular shutdown.
- #759 remains held on its own branch at6019be87f98b434429c17d2787802abf6c98ad2e. No changes, dependency work or enrollment there.

## Core concepts

| Pure concept | Location | Status |
| --- | --- | --- |
| Park-leave retry decision: settled, fresh-safe, recover-existing, refusal, budget-exhausted | cmd/internal/couchcore/leave_park.go (new) | new |
| LeaveResult with ordered unresolved addresses and existing recovery advice | cmd/internal/couchcore/park.go | modified runtime result only |
| Park commit overrun error carrying observed duration and selected threshold | cmd/internal/couchcore/park.go | new typed evidence; not text classification |

| Integration | Location | Status | Wraps |
| --- | --- | --- | --- |
| Couch.Leave / per-address drain | couchcore/park.go, leave_park.go | modified | existing thread holds, PairLifecycle Park/Recover, one context deadline |
| BeginPark measurement | couchcore/park_commit_timing_test.go | new opt-in fixture | real private ThreadStore durability with fake external processes/sessions |
| Console final/partial report | couchtty/console.go and console_run_menu_test.go | modified | existing confirmation, operation result and full copyable notice |
| Physical full quit | testcouch/fleet.go and cmd/couch/fleet_acceptance_test.go | additive helper/test | shared isolated PTY, installed Zellij, Claude/Codex fixtures, exact cleanup |

The pure decision consumes outcome evidence; it performs no IO. The transaction is the lifetime authority. Fresh retry needs positive rollback/no-publication proof and the same guarded incarnation. Existing transactions keep their nonce and go through Recover; no new recovery authority or force-kill path.

## Operating envelope and proposed policy (requires TL approval)

- **Commit threshold:** provisional10s, inside the operator's5–10s choice. Preserve100ms as a soft diagnostic. Confirm with measurements before setting it; if measured tails exceed10s, stop and report rather than expanding silently.
- **Overall park-leave budget:** propose60s from executing the confirmed leave, including gate waits, queueing, queries, attempts and reconciliation. Every retry uses the same absolute deadline; completion waits use the smaller remaining parent budget.
- **Attempts:** at most3 per target, including its first drive. Do not repeat successful slots. Keep current fan-out; no increase or extra timer/poller. Check remaining budget before every next attempt and before starting another target.
- **Important limit:** BeginPark and some durability calls are synchronous and cannot be preempted by context. The60s budget stops new work/retries and cancels context-aware waits; owned writes/cleanup must still settle/join. It is not a promised hard60s process-exit ceiling. A hard wall-time limit would require a larger ownership design; not proposed here.
- **Scale:**30 slots is the functional contract; test15 and30 for counted growth. One initial fleet snapshot; at most3N slot drives, no fleet rescan per retry. Existing reconciliation cost also depends on each retained attempt history A; no more than3 new attempts per slot. Count snapshot calls, slot drives, publications/triggers and peak concurrency at N/2N.
- **Resources:** no synthetic machine-wide CPU stress. Measurement writes at most64MiB in an explicit private primary-SSD root because RAM-disk fsync cannot qualify the operator's storage latency. Other tests prefer APFS TestScratch. Native4/8-total-slot fixtures retain their existing harness bounds and cleanup.

## State and recovery

Actions: confirm full quit; inspect named slot; run its existing Recover park; retry full quit after correction.

| State / failure | Visible state | Recovery | Possible loss |
| --- | --- | --- | --- |
| Still within budget / safe retry | Quitting, counts in existing operation state | wait | agents may already be stopped |
| All selected targets verified parked | One summary then Couch exits | none | mid-turn work was explicitly authorized to stop |
| Confirmed unpublished rollback, same source | internal bounded retry | wait | no new quit was published |
| Existing uncertain transaction | reconcile/recover same transaction only where existing protocol permits | wait, or inspect on refusal | partial durable/teardown progress retained |
| Unsafe/unreadable/replaced source, unknown incarnation, exhausted attempts or budget | One partial summary naming ALL unresolved/skipped/not-started addresses and their recovery; Couch stays open | inspect named slot, then existing recovery/retry | unresolved agents may still run; never label shutdown complete |
| Console cancellation / host process death | existing interrupted-park recovery | restart Couch and inspect/recover | retains existing transaction history; no forced rollback of uncertain effects |

No new durable states: existing park transitions remain authoritative. Per-address retries hold the existing thread gate, re-read authoritative state, and compare the captured source before any fresh attempt. Duplicate Alt+x cannot start another active leave while the existing operation runs. Late completion is reconciled before any resend; a changed incarnation is refused, never substituted. Permanent validation/ownership/storage-uncertainty failures are not blindly retried. Snapshot-order reporting is stable regardless of completion order.

## Tasks

### 1. Measure and preserve the baseline before changing budgets

Files: new opt-in couchcore/park_commit_timing_test.go; workshop/evidence/000827/.

- [ ] Build on the real ThreadStore/controller with existing pairlifecycletest stateful completion/owner model and real clock. Do not create a second fake protocol.
- [ ] Run one15-slot and one30-slot bounded commit/park pass with existing fan-out in an empty-base private environment. Record per-slot RequestedCommitDuration, full operation duration, phase/error and write volume. Record filesystem/compiler/build and contemporaneous machine load/co-tenancy.
- [ ] Use a private primary-SSD root for this measurement and an APFS scratch control if useful; label healthy-load evidence honestly. No live fleet, operator files or artificial unbounded stress. Preserve all results, including failed/rolled-back commits, then delete owned data.
- [ ] Record observed p50/p95/max, not a statistically inferred tail guarantee. If the operator's load is absent, say that the loaded incident remains unproven. Confirm provisional10s or propose5s with headroom; stop if neither is supported.

### 2. Pin the new commit threshold with existing RED/GREEN tests

Files: couchcore/park.go, relaunch.go, park_rollback_test.go, park_wait_test.go.

- [ ] Extend existing fake-clock test first:2–3s commits proceed; selected-boundary/over-bound commits roll back before publication and return typed duration evidence. Failed rollback retains uncertainty and denies fresh retry.
- [ ] Run targeted tests RED. Separate relaunch observation's existing1s timeout and pin its behavior unchanged.
- [ ] Set the measured operator-approved threshold; derive diagnostics from that constant, retaining100ms diagnostic semantics. No hidden duplicate budget or generic Park behavior redesign.
- [ ] Run the complete relevant park/relaunch test cohort GREEN.

### 3. One bounded full-quit operation

Files: couchcore/leave_park.go, leave_park_test.go; park.go/leave_test.go.

- [ ] Add deterministic policy tests before implementation for30 targets, modeled2–3s commits, a safe first-attempt overrun, recoverable retained transaction, late completion, failed rollback, permanent refusal, source replacement, cancellation and budget exhausted before a queued target begins.
- [ ] Represent retry disposition as explicit typed outcomes. No error-string matching. Use one absolute deadline and at most3 drives per target under existing fan-out/hold; do not create a second nonce for an unconfirmed transaction.
- [ ] Preserve already parked targets, reconcile existing completion before re-trigger, and report exact failures/not-started/skipped targets once in snapshot order.
- [ ] Controlled tests cancel held waits and prove ownership/worker joins. Test exact concurrent duplicate/coalescing order; no sleeps as ordering assertions.
- [ ] At15/30 targets assert one fleet snapshot, bounded attempts/publications, unchanged concurrency and no retry of a verified success. Include Claude/Codex profiles; Qoder if available.

### 4. One confirmation and one honest final report

Files: couchtty/console.go, console_run_menu_test.go, console_test.go; testcouch/fleet.go; cmd/couch/fleet_acceptance_test.go.

- [ ] RED: one physical Alt+x plus existing confirmation finishes a30-fake-slot quit, including slow commits/stragglers. No second gesture is needed for eligible automatic retries.
- [ ] On full success retain current report-and-exit; on failure preserve partial progress and put one complete copyable summary into the existing notice/menu error surface. Names/recoveries must survive clipped visual rows. Keep Couch open on unresolved shutdown, not silent partial success.
- [ ] Extend the existing testcouch owner with a physical ParkAndQuit helper mirroring Detach keyboard negotiation and confirmation; no improvised Couch fixture or production CLI shortcut.
- [ ] One declared private4- and8-total-slot native flow across Claude and Codex, respecting existing harness caps. Prove console exit and verified inactive history for every targeted incarnation, and Close/AssertCleaned for all owned processes/IO. Preserve setup failures rather than retry around them.

### 5. Verification, documentation and landing

- [ ] Record before/after commit and full-quit measurements; distinguish controlled slow-stimulus evidence from naturally loaded native measurements.
- [ ] Run targeted park/leave/relaunch, console and testcouch tests, appropriate race cohorts and internal/repolint. Full/affected evidence follows the repo's declared SDLC contract on the pushed final head.
- [ ] Update atlas/couch.md and atlas/index.md for budgets, retry boundary and partial recovery. No project-file edits; report to TL.
- [ ] Before close run make test-affected-inventory and audit any named reader; no bulk hash refresh. Use normal SDLC close review and merge, with live/operator check coordinated through TL if required.

## Architecture review

- ARCH-DRY: reuse gates, worker, Recover, lifecycle fake, notices and shared native harness; no second quit protocol.
- ARCH-PURE / ARCH-ORDER: typed retry decisions and explicit outcomes; persisted Park remains sole authority, with identity/nonce fences across late completion and cancellation.
- ARCH-PURPOSE: one confirmed Alt+x owns safe retries and exact unresolved reporting, not merely a larger timeout.
- ARCH-MOCK: real private store plus existing stateful Pair lifecycle model; native shared-harness conformance.
- ARCH-CONSTRAINTS: measured5–10s commit choice, proposed60s shared scheduling budget,3-drive cap, current fan-out and N/2N counted checks; synchronous IO limit stated.
- ARCH-SECURE: no new credentials or trusted inputs; preserve existing strict lifecycle decoding and exact process/session identity. Fixtures start from empty environment and own private roots.
- ARCH-FUNERAL: retry state dies with the joined leave; no new worker/cache/file family in production. Measurement artifacts have bounded retained text; owned scratch/data/processes removed.
- ARCH-EVOLVE: no persisted format/schema or existing park-state meaning changes; runtime result additions only.
