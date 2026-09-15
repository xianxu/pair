---
id: 000250
status: codecomplete
deps: []
github_issue:
created: 2026-09-14
updated: 2026-09-14
estimate_hours: 5.056
started: 2026-09-14T10:31:03-07:00
actual_hours: 7.25
---

# Recover stale Couch threads without losing live sessions or checkpoints

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only; calibration source is marked stale by SDLC.
Derived after plan-quality passed in round 2. Mapping in order: issue/spec;
recovery policy/coordinator; request authority extension; exact readiness/target
generation wiring; archive reconciliation; operation dispatch; recovery menu;
cross-package acceptance; disposable live acceptance; docs; closing review.

Reuse existing ThreadStore, checkpoint, readiness, operation queue, supervisor
lease and Zellij fixtures. No external library implements this ownership policy.
Design uses thorough-plan 0.2 discount: modules 1h, smaller modules 0.3h,
wiring 0.6h, dispatcher 0.5h, TUI 2h, docs/review 0.2h. Issue/spec uses 0.75h
without another discount. Implementation uses v3.1's 40% of v2 hours:
modules 0.8h, smaller/wiring/dispatcher/review 0.5h, TUI 1h, docs 0.2h,
spec 0.3h. Familiar stack multiplier 1.0; design buffer 15%.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.75 impl=0.12
item: greenfield-go-module design=0.2 impl=0.32
item: smaller-go-module design=0.06 impl=0.2
item: cross-cutting-refactor design=0.12 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: skill-or-dispatcher design=0.1 impl=0.2
item: tui-screen design=0.4 impl=0.4
item: greenfield-go-module design=0.2 impl=0.32
item: greenfield-go-module design=0.2 impl=0.32
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.2
design-buffer: 0.15
total: 5.056
```

## Problem

A failed hosted restart or lost Couch helper can leave a thread recorded as
live while Couch refuses to host it. The operator cannot resume work, and
archive can also refuse because the persisted incarnation is still occupied.
Recovery must be available without hand-editing the thread store or discarding
a surviving agent conversation.

Reported 2026-09-14 for Pair address
`e108517d46ab4575/couch-36b623f6869ebaa2`, following the continuation failure
tracked in #249. Read-only `couch --show pair` reported recorded live PID 5330
and `unusable: stale — couch exited unexpectedly`. At inspection time,
`zellij list-sessions --no-formatting` showed no live Pair session. The operator
had reported an archive refusal attributed to a live Zellij session; the exact
UI error was not captured, so distinguish that report from current observation.

Source inspection found `archivableRecord` rejects occupied persisted
incarnations before `Couch.ArchiveThread` reaches session quiescence. Thus a
stale live record can block the documented manual escape even when its process
has died. Recheck runtime evidence and the exact refusal before repair.

The archived proposal
`workshop/history/issues/000171-reconcile-stale-incarnations-after-crash.md`
was punted because manual archive was considered sufficient. This incident
invalidates that premise. Refer to that file by path: another archived issue
also uses ID 000171. #214 covers a related launch-race recovery case; #249
prevents the continuation failure. This issue owns recovery from an already
stranded thread, independently of those triggering bugs.

## Spec

Provide an operator-accessible recovery path from a stale/unusable Couch row,
with a diagnosis and available next action. Recover ownership through Couch's
existing store transactions and session observation mechanisms; do not ask the
operator to edit JSON or bypass ownership checks (ARCH-DRY, ARCH-SECURE).

Distinguish the dead Couch helper from its potentially surviving Zellij and
agent processes. Retire only incarnations whose exact PID and process-start
identity prove they are dead. Unknown observations, active clients, ambiguous
ownership and open lifecycle transactions require an actionable explanation,
not an assumption of death. Recheck evidence at execution and serialize recovery
with start, park, resume, relaunch, archive and other recovery attempts
(ARCH-ORDER).

When an exactly owned live detached session survives, restore Couch attachment
to that session, preserving the running agent. Warm reattachment must not require
a native transcript binding or silently fall back to a new agent (#248).
When the source session is gone, preserve and offer recovery from the exact
saved continuation, including one authored in another worktree. Explicitly
distinguish a fresh conversation seeded by that checkpoint from native
conversation resume; only offer native resume when its binding is verified.
Missing or unreadable recovery evidence must be reported without silently
launching an empty conversation.

Make archive a reachable explicit escape from a reconciled stale row. Preserve
checkpoint, native transcript references and Pair history. A genuinely live
session must not be terminated as a side effect of inspection or reconciliation;
any stop/archive must follow the operator's explicit action and verified scope.
Do not let one stale row permanently prevent work in its repository.

Prove recovery through a disposable Couch thread with deliberately induced
helper/session failure and an exact checkpoint. The original Pair incarnation
was already manually retired; do not damage an existing working thread to
recreate the incident. Record automated integration and live fixture evidence,
and provide the disposable fixture for operator smoke testing.

## Done when

- A stale row exposes a working recovery action and truthful diagnosis; a dead
  helper with a surviving detached session reattaches to the same agent.
- A dead session with a valid checkpoint can recover into usable Couch-hosted
  work seeded from that exact checkpoint, including across worktrees.
- A stale persisted live incarnation cannot permanently block both recovery
  and explicit archive. Missing recovery evidence still leaves a clear escape.
- Stateful tests cross inventory, UI eligibility, process observation, store
  transition and attachment/archive. Cover live/dead/unknown processes, PID
  reuse, absent native binding, occupied sessions, missing checkpoint, open
  transactions, concurrent actions, interruption and retry without duplicate
  ownership or silent loss of conversation context.
- A disposable live Couch fixture demonstrates usable recovery, with an
  operator smoke path; docs describe recovery choices and failure handling.

## Plan

- [x] Reproduce the stale-record/archive dead end and surviving/dead-session variants with stateful fixtures.
- [x] Design recovery choices, ownership proofs and interruption handling in a durable plan.
- [x] Implement the shared recovery path, verify UI-to-store behavior and update the Couch atlas.
- [x] Verify recovery with a disposable live fixture, preserve existing operator threads, record the outcome and close through SDLC review; pause for operator smoke.

## Log

### 2026-09-14
- 2026-09-14: closed — BR1/BR2 accepted in round2; BR3 atomic model retirement passes final pure/interruption tests and focused race12.096s; rebuilt binaries. Earlier full repo and affected packages green. Latest broad core run only failed an interim invalid test fixture; corrected fixture and recovery regressions rerun green2.566s, command package green25.070s. Real Zellij and interactive stand-in recovery passed. Plan-check waiver only this gate and post-review operator smoke; Astro untouched.; review verdict: SHIP

Created at operator request specifically for recovery, after locating the
punted reconciliation proposal and checking the current stale Pair record.
The checkpoint was found at
`/Users/xianxu/workspace/worktree/pair/000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it/workshop/continuation/20260913T210234-pair-storage-retention.md`.
Runtime addresses and paths are incident evidence, not authority to act later
without fresh verification. No sessions or runtime metadata were modified;
implementation has not started.


### 2026-09-14 — Operator-authorized manual unblock

Detach-all failed partway through on exact address
`e108517d46ab4575/couch-36b623f6869ebaa2`, reporting no live Pair session.
Operator explicitly requested manual state repair before implementing #250.
Rechecked: recorded PID 5330 absent, exact Zellij session `📁pair-couch-27`
absent, revision 22, identity `1789341245.53810`, one live incarnation,
no open start/park transaction. Backed up the entire threadstore and exact
#239 continuation to `/tmp/pair-250-manual-recovery-57qbkgg9`.

A temporary Go helper called the existing `ThreadStore.RetireIncarnation`
transaction with revision 22 and that exact identity, immediately rechecking
`OSProcOps.Exists == Dead`. Passed the existing LastActiveAt to preserve
historical activity rather than claiming the absent session ran today.
Result revision 23, no incarnation, no fabricated VerifiedPark. Raw comparison
proved only this record changed, and only its revision/incarnations fields;
all other thread records unchanged. Helper source retained in backup and
removed from checkout. `couch --show pair` now truthfully reports `session gone`.
`Couch.Leave` skips records without an active incarnation, so the identified
stale-record detach blocker is removed. No process/session was signalled or
archived. Operator must retry detach-all and confirm usable access; the absent
Pair conversation is not restored by this repair. General #250 remains open.

## Revisions

### 2026-09-14T13:28:00-07:00 — Astro exit investigation

Operator reported another spontaneous exit and asked for trigger analysis.
Read-only evidence: Astro address `fcff31946c0ac9d2/couch-057762b29e5f0952`
still records helper PID 20234/start identity `1789416513.384925` live at
revision 56, but that PID is absent and the registry has dropped Astro.
Claude PID 51537, wrap PID 51530 and Zellij server PID 51529 remain alive
since September 12. Exact session `📁astro-couch-2` is live with zero clients.
Registry mtime is 13:16:41.601764; this is not proof of the exit timestamp.

Confirmed stale-state mechanism: `Console.onExit` calls `Couch.Forget`, which
removes the registry actor without retiring the durable incarnation;
`Couch.PruneDead` likewise leaves that incarnation. The current label claiming
the supervisor crashed is not justified by this evidence. Attach has no
post-start timeout. Wrapper output continues through 13:12 with no agent exit;
available Zellij/macOS logs do not establish the helper's exit trigger.
The exit notice is transient and stores only an integer code, losing the signal
distinction. Do not assert a particular signal, keypress or cleanup operation
caused this incident without more evidence.

Snapshot of record, registry, process identities and relevant logs:
`/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair-astro-exit-sescyfqn`.
No process was signalled, session attached, or runtime record repaired during
inspection. Astro is a real surviving-session instance of the #250 gap; preserve
its live agent when implementing recovery. Fresh spec review approved #250;
the durable plan is drafted and its first review requested exact target-generation
correlation on retry plus explicit absence phase rules, now appended by its author.

### 2026-09-14T13:20:00-07:00 — Controlled recovery acceptance

The operator noted there is no longer a naturally broken thread and approved
disposable fault fixtures for #250. Replace the obsolete real-incident recovery
requirement with controlled helper-gone/session-alive, helper-and-session-gone
with checkpoint, and stale/no-checkpoint cases. Stateful tests additionally
cover unknown ownership, PID reuse, concurrency and interruption. Historical
manual repair evidence remains above; it does not by itself prove the new flow.

#249 shipped in PR #135 after SHIP review and a successful operator brain-thread
continuation smoke. #250 planning has begun via `sdlc start-plan`. Initial source
inspection confirms that archive still refuses stale occupied records, while
#249 correctly requires an actual park receipt and cannot yet recover an
already-dead source. Reuse its execution and delivery machinery with explicit
absence authority; do not fabricate a park receipt (ARCH-DRY, ARCH-ORDER).

### 2026-09-14T10:40:00-07:00 — Reconcile then reuse recovery operations

Operator approved design and implementation of #248 → #249 → #250, pausing for
smoke after #248. Coordinated contract is recorded in
`workshop/plans/000248-couch-warm-reattach-gate-plan.md`; #250 receives its own
executable plan before implementation. Its responsibility is exact stale-owner
reconciliation and recovery choices, reusing #248 reattachment and #249 durable
checkpoint execution rather than building another launcher. Diagnose observed
helper/session state without asserting the whole Couch supervisor crashed.
The operational acceptance baseline includes the manual incarnation retirement
already logged above; usable conversation recovery remains to be confirmed.


### 2026-09-14T14:00:00-07:00 — Implementation integration and telemetry follow-up

Schema/readiness unit implemented and focused checkpoint/readiness/threadrecord
tests pass; optional launch ordinal and absence/generation witnesses are present.
Recovery menu/dispatch unit implemented, with full Couch TTY race tests passing
before wiring the pending core method. Disposable process/session acceptance and
interactive smoke fixtures are written; integration is still red because the
RecoverThread execution method is being implemented. No end-to-end recovery
success is claimed yet. Core execution and owner archive reconciliation are
active independent implementation tasks. Preserve all live operator sessions.

Operator asked whether Astro can be recovered: surviving detached Zellij plus
dead helper is the intended warm recovery case, preserving the existing Claude
conversation after fresh ownership checks. Operator also requested durable exit
telemetry; tracked separately in #253 so unknown-trigger investigation remains
distinct from #250's recovery contract.


### 2026-09-14T15:13:44-07:00 — Core complete; acceptance integration pending

Core RecoverThread execution, target-generation correlation, shared dead-helper
reconciliation, owner archive revision fencing, menu/CLI and documentation are
implemented. Full core package passed (83.120s); focused recovery race tests and
archive race tests passed. Schema/readiness/threadrecord/TTY packages passed.
The combined command suite remains red on the four new disposable acceptance
fixtures; fixture integration resumed to correct session setup and confirmation
driving and rerun real Zellij conformance. See /tmp/pair250-integrated-tests.log.
Implementation remains uncommitted; full repository verification, SDLC boundary
review, build and operator smoke are pending. Astro runtime is untouched.


### 2026-09-14T15:24:15-07:00 — Acceptance progress; live empty-session blocker

All four disposable command/menu acceptance cases now pass, including race:
warm recovery, checkpoint recovery, retired checkpoint import, and archive.
Real Zellij warm recovery preserves the source agent PID and accepts input.
PTY smoke exercised warm/checkpoint recovery and exact NEXT ACTION/echo behavior.

Live source-absent conformance remains blocked by a discovered production
observer mismatch: Zellij 0.45.1 returns exit 1 with "No active zellij sessions
found." after deleting the last session, but the session-quiescence observer
treats that as an error and loops. Fixture owner is now implementing a shared
observer fix plus regression test before rerunning live conformance and cleanup.
Disposable smoke shutdown also exposed blocking raw stdin reads on Darwin;
fixture uses a cancelable reader, so do not claim this proves production stdin
shutdown is fixed. Full final checks, implementation commit and SDLC boundary
review remain pending. Operator Astro session remains untouched.


### 2026-09-14 — Evidence-backed implementation checklist audit

Shared recovery, owner archive, generation-correlated absence continuation,
menu/dispatch and atlas work are implemented. The plan revision records actual
concept/file mappings and the layered fixture scope. Portable four-mode race
acceptance passed (2.686s); real-Zellij warm and source-absent conformance and
both interactive warm/checkpoint smoke commands exited 0 with echo, exact
`RECOVERY-EXACT-250` checkpoint output and clean fixture Ctrl+D. These use
stand-in agents/deterministic target attachments and do not establish a complete
real coding-agent end-to-end recovery. The missing-checkpoint archive drill is
portable acceptance. The empty-Zellij-inventory blocker above is fixed through
shared observation normalization and its regression tests pass (1.046s).
Final core/TTY race passed (94.086s/9.204s); current binaries built successfully.
Root is still completing repository checks and SDLC review. Operator smoke and
the plan's remaining explicit verification obligations remain open; no real
operator thread was faulted or restarted for this fixture.


### 2026-09-14T18:11:53-07:00 — Final verification before close review

All repository Go tests pass (/tmp/pair250-final-go-test.log). make test completed
all shell/Lua checks; its first Go pass found only missing source-inventory
classification for recovery.go/recovery_execute.go. Added the two non-artifact
consumer entries, reran artifactpath tests successfully, then reran the full Go
suite successfully. Full Couch core/TTY race tests pass (94.086s/9.204s,
/tmp/pair250-final-race.log). make build passes and rebuilt bin/pair, bin/couch
and pair-launch-helper (/tmp/pair250-final-build.log). Disposable real-Zellij
conformance and both interactive warm/checkpoint smoke drivers passed; fixture
limitations remain explicitly documented in the plan. Final cancellation/deadline
and bad-document no-spawn regression results are recorded separately below.

The remaining close/checklist items include the review invocation itself and
operator smoke after review, not unimplemented recovery behavior. Preserve that
pending operator acceptance; do not label it already performed. No live Astro
process/session has been changed. Implementation is being committed for the
mandatory SDLC boundary review.


### 2026-09-14 — Boundary review round 1: REWORK

SDLC reviewed 2e4b2df7..75962c63. BR-1 Critical: newly unconditional recovery
observation blocks readable empty never-bound threads from non-signalling
archive. BR-2 Important: retry/generation session observation drops caller
context. Both findings are accepted for targeted reproduction and class-wide
fixes; cancellation sweep includes archive's final observation. No gate bypass
or operator smoke claim. Review ledger preserves stable findings for round 2.


### 2026-09-14 — BR-1 and BR-2 fixes verified locally

BR-1 reproduced through Couch.ArchiveThread using a real readable scoped index
with no matching binding. Added an explicit missing-binding error and a narrow
empty/no-park/no-continuation archive path that performs no Quiesce or signal,
rechecks the index, and uses the final record revision fence. Corrupt index,
retained request and concurrent replacement tests refuse. Focused archive and
PairSession race tests passed (5.388s).

BR-2 reproduced lost caller cancellation during continuation retry. Added a
context-aware presence observer, used it in retry and archive's final observation,
and wired OSOrientationStatusReader.SessionContext in production. Generation,
Read and Registered share the cancellable binding reader. Focused race tests
passed (3.197s), proving deadlines reach the observer without record mutation or
extra launch. Existing cleanup with no caller context keeps its deliberate
post-cancellation reconciliation contract. Review-fix build passes; full affected
core/command packages are running before round 2. No operator session changed.

Full affected packages pass after both fixes: couchcore 95.010s and couchcmd
20.593s (/tmp/pair250-review-fix-tests.log). Real Zellij recovery conformance
passes again (3.142s, /tmp/pair250-review-fix-live.log). Build and diff check
pass. These fixes are ready for the second SDLC boundary review.


### 2026-09-14 — Boundary review round 2: prior fixes accepted; BR-3

Review disposed BR-1/BR-2 as addressed. New Important BR-3 identifies direct
Unknown-to-Live assignment during exact-receipt target reconciliation, before
retiring that dead helper. Accepted: move receipt-backed reconciliation into the
owned pure transition model with a revision-checked store operation and test
interruption before reattachment. Prefer truthful atomic retirement over a
synthetic live intermediate state. Gate remains unfinalized; no smoke-ready
claim or operator-session mutation.


### 2026-09-14 — BR-3 atomic transition implementation

Added RegisteredTargetProof and pure ReconcileRegisteredTarget in the existing
start transition model, with a revision-checked store operation. Exact request,
attempt, agent/session and helper identity must match; only Running/Failed
continuations with one settled Unknown incarnation and no start/park qualify.
The IO caller rechecks registration/death; session observation no longer fakes
a Live state even in its in-memory projection. Retirement preserves activity,
checkpoint/history and all unrelated fields in one revision.

Value-only model tests cover invalid proofs/states. Stateful tests interrupt
immediately after retirement, prove no synthetic Live incarnation or extra
launch, then reattach with the same attempt. Receipt loss, revived helper and
competing revision refuse unchanged. Final focused race passed 12.096s (agent
exec session65339); build passes. Full affected package rerun is in progress
before submitting BR-3 for round 3. No operator runtime changes.

The full BR-3 core run compiled an interim pure-test fixture during relocation
and failed only TestReconcileRegisteredTargetRetiresUnknownAtomically (invalid
source park receipt). The final value-only fixture uses valid SourceAbsence;
fresh root targeted tests pass 2.566s (/tmp/pair250-br3-final-targeted.log), as
did final focused race 12.096s. The same broad run passed couchcmd 25.070s and
reported no other failed test. Build passes (/tmp/pair250-br3-build.log). These
results supplement the earlier green full repository and full affected suites;
do not mislabel the interim full core run as green.


### 2026-09-14 — SHIP review; operator smoke handoff

Round 3 returned SHIP, with BR-1/BR-2/BR-3 addressed and no open findings.
Reviewed window 2e4b2df7..0e702a9d. A clean full-core run on the final fixed tree
also passed 110.875s (/tmp/pair250-br3-fixed-core.log), resolving the interim
fixture-run caveat. Binaries are built; codecomplete is the local acceptance
state, not shipped/done. Preserve the unchecked operator-acceptance item.

Operator smoke: after using the rebuilt Couch, select Astro and choose Recover
session or retained checkpoint. Its surviving session should reattach to the
same Claude conversation after fresh ownership checks. Disposable alternatives:
sh tests/couch-recovery-smoke.sh warm and checkpoint. Stop for operator smoke;
do not publish or modify Astro's runtime on the operator's behalf. The separately
investigated mouse-selection mode leak is recorded under #254 and remains unfixed.


### 2026-09-14 — Operator acceptance and ship authorization

Operator reports Astro loaded automatically at startup without selecting Recover
and appears good. This verifies successful startup reattachment, not the
explicit Recover menu path; that path remains covered by automated and disposable
interactive acceptance. Operator then authorized closing #250. Smoke checkpoint
is accepted; proceed through PR/merge to done and archive.
