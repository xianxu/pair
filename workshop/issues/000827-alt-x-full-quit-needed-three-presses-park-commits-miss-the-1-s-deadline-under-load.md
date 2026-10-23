---
id: 000827
status: codecomplete
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-23
estimate_hours: 2.691
collaboration-mode: collaboration
card_mirror: '96f8fdb7f1fe16017903761889f9b0ce9fd548dd' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T23:31:37-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 2.29
---

# Alt+x full quit needed three presses: park commits miss the 1 s deadline under load

## Problem

## Spec

Operator report (10-22 full shutdown): **Alt+x (park everything and quit Couch) timed out on some slots; it took
three Alt+x presses to shut everything down.** The machine was loaded (load average 40+, 30-odd slots).

Lead (TL reading, unverified): `couchcore/park.go` sets `parkCommitSoftTarget = 100ms` and `parkCommitDeadline =
1s` for each slot's park commit; under load some slots likely miss the 1 s commit and are left unparked, so the
operator has to repeat Alt+x.

Fix: measure park commit times across a full fleet under realistic load (pair#738-style counted timing), derive the
deadline from the measurement with a re-measure note (ARCH-CONSTRAINTS), and make one Alt+x finish the job:
slots that miss the first deadline get retried within one bounded overall budget, and anything still unparked is
named in a single summary (with its recovery) instead of silently needing another press.


**Operator decision (10-22, night):** "there's no reason to set this this tight. I think 5 to 10 seconds is perfectly
fine." Set the per-slot park commit deadline in that range (measurement still recorded to confirm it). Current
parallelism, for context: parks fan out up to `LifecycleParallelism` = half the CPU cores (operator decision 10-07),
so slots park in parallel batches, not all at once nor one by one.

## Done when

- One Alt+x parks a fleet of 30 fake slots where some commits take 2–3 s (test), or names exactly the ones it
  couldn't park.
- The per-slot deadline is derived from measured park commit times, with the measurement recorded.

## Plan

Durable plan: [one-gesture full quit](../plans/000827-one-gesture-full-quit-plan.md).

- [x] Measure private real-store commit timings at15/30 slots before changing the threshold; choose within operator5–10s, stop if evidence contradicts it.
- [x] Retain fan-out/gates/nonce recovery; one proposed60s scheduling budget and at most3 safe drives per target. Synchronous durability cannot be forcibly interrupted.
- [x] Confirmed Alt+x automatically handles eligible stragglers, exits only after verified success, or leaves one exact unresolved-slot/recovery report.
- [x] Deterministic30-slot/N2N tests, physical shared-harness4/8-slot Claude/Codex check, normal recorded verification and close.


## Log

### 2026-10-22


### 2026-10-22 — claimed and plan proposed

TL ee975c2b dispatched #827 while #759 awaits operator review. Card was unowned;
claimed pair:6 and ran start-plan. #759 remains untouched at
6019be87f98b434429c17d2787802abf6c98ad2e. Start-plan catch-up refreshed sibling
ariadne and seeded an unrelated merge-check workflow diff (ariadne#557 class).
Saved it at /Volumes/TestScratch/p827-plan.ogi77579/merge-check-seeded.patch,
restored only that workflow, and reran start-plan successfully. No seeded change
is included on this issue branch.

Read-only trace confirms the1s limit is a POST-COMMIT elapsed check, followed by
known-unpublished rollback; it is not a preemptive timer. Leave has one pass and
no whole-operation deadline. Existing thread/park nonce ownership and Recover
must govern any retries. Raising the shared constant must not accidentally
change relaunch's separate1s diagnostic observation window.

Proposed full-flow plan records a measurement-first10s provisional threshold,
one60s overall scheduling/context budget, at most3 safe per-target drives with
current fan-out, same-identity/nonce fences, exact partial reporting and no force
quit on unresolved state. Non-cancellable durability remains joined and can
exceed the logical deadline; no hard wall-time guarantee. No measurement numbers
or operator-incident causal conclusion are invented. No implementation or native
run started. Plan goes to TL before change-code.


### 2026-10-22 — plan approved

TL3cbb3359 approved the measurement-first plan, conditional10s commit threshold,
60s overall scheduling budget and up to3 safe drives, exact partial recovery
summary, unchanged fan-out and separate1s relaunch diagnostics. The60s is not a
hard process-exit ceiling; synchronous durability stays owned/joined. Proceed to
change-code, then measure before changing the threshold. No #759 changes.


### 2026-10-22 — full plan-quality round1 findings

Pinned full flow because the planned controller exceeds the quick shell. Gate
round1 raised PQ-1 (cancellation ownership) and PQ-2 (function-level test strategy).
Both are valid. Existing parkFuture.Await can return before its producer; the
proposed Leave-only parkJoined/recoverJoined paths retain the exact submitted
future and use AwaitJoined, keeping the thread hold/fan-out unit until done.
Ordinary callers keep existing cancellation behavior. Updated the durable plan
with named risky functions, adversarial class strategies and explicit unreadable
snapshot reporting. No code/measurement started while the gate is blocked.



### 2026-10-23 — capacity handoff checkpoint; stop for model switch
- 2026-10-23: closed — Recorded sdlc-local/suite (affected) exit 0 on 289ef0c5d (review-fix head). Baseline: real-store BeginPark 15/30 slots at load 65, max 709ms vs old 1s. RED->GREEN: 2s/3s commits refused under 1s, now proceed under 10s; fleet 15/30 Leave test RED on nonblocking store.lock vs preview reads, GREEN after TL-approved bounded wait; store_lock_wait tests; exhaustive retry-policy invariants; AwaitJoined test; BR-1 fixed (skipped thread keeps full quit open, test). Native make test-couch-fleet: one Alt+x + confirm parks Claude/Codex 4 and 8 slots, quit 0.48-1.40s, every record VerifiedPark, AssertCleaned. Inventory audited.; review verdict: SHIP

TL eb69966d/d767565b directs commit/push and stop after the third model-capacity
failure; TL will move pair:6 to Claude. All work is on #827; #759 remains untouched
at6019be87f98b434429c17d2787802abf6c98ad2e, waiting for the operator's tsnet decision.

Approved plan: workshop/plans/000827-one-gesture-full-quit-plan.md. Plan-quality
round3 CLEAN; prior PQ-1/PQ-2 addressed. Estimate-quality INFO and change-code
completed exit0 on full flow. Card estimate2.691h is derived in the Estimate
section; advisory concerns: integration includes controlled/race tests, and
native setup cost assumes shared harness reuse. Do not repeat planning/gates
unless design changes. TL f4a757ee approved Leave-only joined adapters retaining
the exact submitted future and thread/fan-out ownership through producer done;
ordinary Park/Recover/Retry waits remain unchanged.

WIP: new cmd/internal/couchcore/park_commit_timing_test.go compiles. Exact targeted
command GOTOOLCHAIN=local go test ./cmd/internal/couchcore -run '^TestTimingNearestRank$' -count=1
PASS0.255s. The opt-in TestParkCommitTimingRealStore has NOT been run; no actual
commit-time samples exist yet. It seeds15/30 records, times real BeginPark on a
new private os.UserCacheDir SSD directory, uses current LifecycleParallelism,
counts staged payload bytes and emits PARK_COMMIT_TIMING JSON. It creates no
agents, requests or external side effects. Review the WIP fixture before its
first run; its64MiB counter currently applies per subtest, so enforce the plan's
whole-measurement bound if retaining that interpretation. No production budget,
retry controller or UI behavior has changed.

EXACT NEXT STEP: review the opt-in fixture, then run it once with an explicit
empty-base child environment (HOME/PATH plus GOTOOLCHAIN=local/GOWORK=off, no
COUCH/PAIR identity). Enable PAIR_PARK_COMMIT_TIMING=1 and run go test
./cmd/internal/couchcore -run '^TestParkCommitTimingRealStore$' -count=1 -v -timeout 3m.
Capture actual uptime/load, compiler, SSD filesystem and complete stdout into
workshop/evidence/000827/; parse p50/p95/max and state whether incident load was
present. Verify owned cache directory removed after the test. Do not change the
1s commit threshold until measurement is recorded. Then execute approved plan:
10s if supported, separate relaunch1s diagnostics,60s scheduling budget/max3 safe
drives with exact-future joins, exact copyable partial summary, deterministic
30-slot/N2N tests and shared native4/8-total-slot Claude/Codex qualification.

No owned suite, merge, review or targeted-test runner is still active. No live
Couch/slot or :0 action was taken. No continuation/restart/model change from this
worker; stop after the committed/pushed receipt so TL can switch the agent.

2026-10-23 (claude/opus-5.5, pair:6) — Task 1 measured. Real-store BeginPark, 15/30
slots, fan-out 6, load avg 64.8: p50 106/122ms, p95 342/367ms, max 342/709ms, 0 errors,
112KB staged. Evidence workshop/evidence/000827/commit-timing-baseline.{md,txt}. Loaded
but not incident load; incident cause unproven. Max at 30 slots used 71% of 1s ->
confirm 10s per plan. Next: Task 2 RED tests for threshold + relaunch 1s separation.

2026-10-23 — Task 2 GREEN. parkCommitDeadline 1s -> 10s; typed ParkCommitOverrunError
{Observed, Threshold} replaces the text cause; relaunch's post-failure read keeps its own
relaunchObservationTimeout=1s. RED seen: 2s/3s commits refused under the old 1s. Failed
rollback retains Park (no fresh nonce). AtOneSecond boundary test made threshold-relative.
Park|Relaunch|Leave|Recover cohort ok (unsandboxed: several fixtures mkdir /tmp, which the
sandbox denies). Overlap: open #580 (slow park commit) is the same 1s defect; this
change fixes its stated symptom — flagged to TL. Next: Task 3 bounded leave pass.

2026-10-23 — #580 assessed per TL: NOT fully met by #827. Its Spec forbids "merely
raise the timeout" and Done-when requires a successful slow BeginPark never fail solely
from elapsed time; #827 still rolls back on elapsed time (at 10s). Leave #580 open;
remaining: distinguish committed-slow from failed/uncertain commit instead of a wall-time
cutoff.

2026-10-23 — Task 3 implemented (uncommitted→checkpoint): leave_park.go (pure
decideLeaveParkRetry, leaveParkOne joined drives, LeaveUnresolved, FormatLeaveParkSummary),
AwaitJoined + parkJoined/recoverJoined via typed await, parkRolledBackError (positive
no-publication proof, no text change), LeaveParkBudget 60s, Leave park routes to
leaveParkOne; leaveOne's dead LeavePark branches removed. Existing Park|Relaunch|Leave|
Recover cohort green.

DISCOVERY (likely incident root cause, needs TL design decision): the 15/30-target
fleet test on a COORDINATED store (as production, couch.go:197) fails with "lock thread
store: resource temporarily unavailable". Traced (temporary instrumentation, removed):
ctx-aware writes (AdvancePark/FinalizePark/rollback) run under the coordinator but take
the nested store.lock NONBLOCKING (withStoreLockChecked check!=nil); routing reads
(recoveryBackend -> storeForAddressContext -> slotRepositoryManifestContext) and any
preview read take store.lock via retentionReadLock OUTSIDE the coordinator. A sibling
park's routing read, or Console's preview poll, makes a park write fail at once. Under
fan-out this is exactly "needs another Alt+x". leave_park_test RetriesProvenRollback
is the RED fixture (23 EWOULDBLOCK in 3 runs). Waiting on TL before changing the lock.

2026-10-23 — TL approved option A (bd8f4bf3 reply). Foreground ctx-aware writes
(withUpdateLockContext) now poll the nested store.lock within storeReadLockWait (1s),
honoring ctx/coordinator checks, then fail typed ErrThreadStoreBusy; retention
maintenance (archive_gc) keeps wait 0 / immediate yield. Tests: store_lock_wait_test.go
(brief preview hold waited out via contention hook, typed busy past bound, cancellation,
maintenance still yields without polling). Under -race + load ~65 one rollback then hit
its fixed 1s cleanup ctx -> uncertain; rollback cleanup now uses parkCommitDeadline (same
budget as the commit it undoes). Fleet 15/30 green; isolated wall ~0.7-1.2s, 0 contended
polls. Full couchcore package ok (315s, unsandboxed). Next: Task 4 console summary.

2026-10-23 — Task 4 (console) done: Leave(LeavePark) failure returns typed
LeaveParkIncompleteError whose text is FormatLeaveParkSummary, so the existing error
notice (and its copy key) carries every unresolved thread + next action; Couch stays
open; success path unchanged. Console test: one Alt+x + confirmation dispatches leave
once, notice ends with the whole summary (surface prefixes "Couch: "). Next: testcouch
physical ParkAndQuit helper + native 4/8-slot Claude/Codex fleet acceptance.

2026-10-23 — Task 4 native done: testcouch Fleet.ParkAndQuit (physical Alt+x + confirm,
incomplete summary via copy key) and ListThreads; TestDisposableCouchFleetFullQuit
(claude/codex x 4/8 slots) in make test-couch-fleet: all PASS, quit 0.48-1.40s, every
record VerifiedPark, Close/AssertCleaned clean. Evidence native-full-quit.txt. Finding
for TL: codex rows can list "binding lost — repairable" after a verified park when quit
lands before transcript discovery (resume-binding, not park). Atlas couch.md + index
updated. Next: verification sweep, inventory audit, pushed-head suite, close.

2026-10-23 — Recorded affected run on 0178970cc failed on two branch defects (not
flakes): artifactpath exhaustive inventory lacked leave_park.go (added beside
parkworker.go), and onStoreLockContended was inserted between
acquireThreadStoreLockWithin's godoc and its func (moved above). Both targeted tests
green; requeued. couchtty TestNativeAgentExitDiagnostics/hup failed once in my
unrecorded targeted run and passed 3x on the same head; register refused for lack of
recorded evidence, it did not recur in the recorded suite.

2026-10-23 — Close review round 1 (FIX-THEN-SHIP, window ce5146c9..81b47b37): BR-1
Important fixed as a class: under LeavePark every outcome short of verified park is
unresolved (skipped/unvouched incarnation now names the thread and keeps Couch open);
TestLeaveParkSkippedThreadKeepsCouchOpen. Minors fixed: advice no longer duplicated in
Reason (leaveParkOne stops appending it; test asserts once); unresolved advice lookups
run at LifecycleParallelism instead of serially; recoveryBackend comment updated.
Lessons added. Re-running recorded suite on the fix head.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against
baseline-v3.1.md. Method A only; calibration is marked stale/provisional by
sdlc estimate-source. Existing controller, store, lifecycle model, console and
native harness make this familiar territory; no novel library/stack discovery
or new dependency is needed.

Derivation from v2/v2.1 table, with thorough-spec design discount0.2 and v3.1
implementation scale0.40 applied once:

- Measurement/threshold extension: smaller Go base design0.2/impl0.5 ->0.04/0.20.
- Joined lifecycle retry integration: API integration base2.0/1.5 ->0.40/0.60.
- Existing console completion/reporting: TUI base1.0/0.8 ->0.20/0.32.
- Shared native fixture extension: smaller Go base0.3/0.5 ->0.06/0.20.
- Native key/park conformance discovery: real API base0/0.6 ->0/0.24.
- Atlas and measurement documentation: docs base0.1/0.2 ->0.02/0.08.
- One close review/fixes: review base0.1/0.5 ->0.02/0.20.

Design subtotal0.74 with15% buffer=0.851; implementation subtotal1.84 at familiar1.0;
combined2.691 focused ship hours. Machine FIFO time is not a primitive estimate.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.04 impl=0.20
item: api-integration design=0.40 impl=0.60
item: tui-screen design=0.20 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: real-api-discovery design=0 impl=0.24
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 2.691
```

## Revisions

### 2026-10-22 — plan gate and approved joined ownership refinement

TL f4a757ee confirmed the Leave-only exact-future joined adapters; ordinary waits
and recovery authority remain unchanged. Plan-quality round2 CLEAN, both PQ-1 and
PQ-2 addressed. Derivation above follows only after that plan acceptance. Full
implementation remains measurement-first; no production budget changed yet.
