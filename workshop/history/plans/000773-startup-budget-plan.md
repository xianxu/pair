---
collaboration-mode: discussion
---

# Caller-owned terminal startup budget

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Execute this single-pass plan with the executing-plans skill; SDLC owns the boundary review.

**Goal:** Birth, exact-generation readiness and registration consume one caller-owned startup deadline, without making that deadline the admitted interactive client's lifetime.

**Architecture:** Select an absolute deadline once at the launch owner, transport it through the existing blocked helper, and use it above the existing short native-observation budgets. Keep the terminal lease, readiness identity checks, client cleanup and uncertain recovery outcomes authoritative. No retry or numeric timeout increase.

**Tech stack:** Go contexts, existing launcher Runtime and Couch FakeRunner seams, testing/synctest, existing native disposable-fleet harness once #744 lands.

## Investigation and alternatives

The actual production boundary `launcher/session_handoff_os.go:runTerminalHandoffContext` supplies `birthBound` to the coordinator. `session_handoff.go` starts a relative timeout under the interactive context, even when the caller has substantially more time. `LaunchOptions.BirthBound` only controls a different outer watchdog. Couch starts its two-minute registration wait after acknowledgement (`couchcore/launch_existing.go`), while the blocked helper has a separate ten-second acknowledgement timeout. The postbirth native query starts another short budget from interactive lifetime. Fixing one constant cannot make these coherent.

1. **Recommended:** one startup deadline, separate from interactive lifetime and short native-query budgets. All phases consume the same remaining window; preserve earlier cancellation.
2. Removing only the inner ten-second timeout fixes the reported truncation but leaves registration renewal, no-deadline callers and subprocess propagation unspecified.
3. Raising ten seconds or putting the overall deadline on the interactive client either repeats the defect at another load or terminates healthy sessions. Reject both.

The original #744 missing-readiness incident is retained as evidence, not proof of why its wrapper was late. Load, AMFI, inheritance and early child failure remain unproved causes of that particular incident.

## Core concepts

### Pure values

| Name | Lives in | Status |
|---|---|---|
| Startup deadline selection / encoding | `cmd/internal/startupbudget/budget.go` | new |
| OperationProgress | `cmd/internal/couchmessage/operation.go` | new |
| OperationReceipt / ReceiptEvent | `cmd/internal/couchmessage/operation.go` | modified |
| LaunchOptions startup context | `cmd/internal/launcher/runtime.go` | modified |

The selector takes explicit current time, an optional caller deadline, and a fallback duration; it returns the original deadline when present, otherwise now plus the fallback. A supplied child deadline is capped by an earlier local caller deadline and never renewed. Reject invalid/expired serialized deadlines before launch effects. Share the existing two-minute registration hang ceiling as the fallback; move its declaration to startupbudget rather than inventing a second default. Test selection, strict bounded timestamp parsing/encoding, expiration and earlier-parent composition directly.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Tracked launch owner | `cmd/internal/couchcore/launch_existing.go` | modified | helper, thread store, registration |
| Blocked helper | `cmd/internal/couchcore/launchhelper.go` | modified | ack pipe and exec |
| Native launcher entry | `cmd/internal/launcher/runcli.go` | modified | consumed launch environment |
| Create/attach orchestration | `cmd/internal/launcher/createflow.go`, `runtime.go`, `osruntime.go` | modified | Runtime handoff |
| Birth and admission | `cmd/internal/launcher/birthwatch.go`, `session_handoff.go`, `session_handoff_os.go` | modified | local evidence and native Zellij queries |

Reuse FakeRunner, the channel-controlled handoff world and fake launcher runtime. They model helper ack/exit, evidence publication, readiness and client lifetime, not merely a list of expected calls. Add virtual-time boundary cases through the same budget-selection function called by the OS handoff, never a test-supplied replacement timeout. Real subprocess tests separately qualify transport and helper expiry with private files. No new executor, persistent store or lease.

## Deadline contract

- **Couch owner:** at entry to `launchTrackedThread`, before preparation/spawn, select the startup deadline. An explicit caller deadline wins; absent one, use the existing two-minute ceiling (retain explicit test policy overrides). All normal startup preparation, birth, readiness and registration consume that window. Preserve the original operation context separately for post-attempt diagnosis/recovery.
- **Helper boundary:** export the exact absolute deadline as a consumed internal `PAIR_STARTUP_DEADLINE` value alongside the existing generation-bearing Couch intent. Override inherited values. Pass remaining time to StartBlocked, and have the helper check the absolute deadline when waiting for acknowledgement and again before exec, so process-start delay cannot renew it. Unrelated helper callers retain their current API. The helper's ordinary orphan/EOF handling remains unchanged.
- **Pair entry:** read and unset the internal deadline before launching descendants. Require a valid selected Couch intent when it is supplied; malformed/expired values refuse before startup effects. Turn it into a separate startup context. It is not authority to change a session name, nonce or admission result. Existing helpers without this value use the explicit fallback, not an unbounded wait. Coordinated Couch/Pair builds are the rollout contract; do not claim older binaries gain the fix.
- **Standalone/programmatic caller:** an explicitly supplied startup context/deadline owns the attempt; otherwise inherit any earlier lifetime deadline, or select the shared two-minute fallback at the selected effect boundary. Do not spend it while a human is choosing a tag/configuration. A restart-loop iteration is a new attempt; it gets a new fallback only if no externally supplied deadline owns it. Never reuse an expired Couch deadline for a later attempt.
- **Runtime boundary:** carry lifetime, startup and short precreation admission contexts explicitly through create/attach handoff. Update Runtime implementations/fakes together; do not hide startup authority in context values or OS globals. Direct OS entrypoints also use this shared selector, eliminating the fixed production birthBound path.
- **Short observations:** precreation checks share their existing five-second query window, capped by startup. Readiness survives expiry/cancellation of that completed query window, but not expiry/cancellation of startup. After exact readiness, native admission gets its existing fresh five-second observation window capped by the same startup deadline. No native query before positive local birth/readiness evidence except the existing birth-watch diagnostic policy.
- **Birth watchdog:** ten seconds remains the existing initial liveness-probe threshold, not a readiness failure deadline. Its backoff and proof-of-death policy stay; its work is bounded by startup cancellation. A live or unknown session is never treated as dead merely because startup time expired.
- **Registration:** `whileHelperAlive` remains an early-exit observer under the already selected startup context. Do not create a fresh two-minute window after ack. Keep existing final read-only recovery/exit looks explicitly outside startup admission: they may reconcile the exact already-started attempt but never create/retry one or renew its startup deadline.
- **After admission:** cancelling/expiring the startup context must not end the interactive client. Its original lifetime context still owns it. An unsuccessful handoff cancels/reaps only its owned client and joins observation before lease release, preserving existing generation-qualified server/agent cleanup rules.

## Startup progress and final diagnosis

The existing operation progress surface must report the current startup stage and elapsed stage time, for example “waiting for agent readiness, 25 s”. Propagate bounded typed stage facts through the existing launch-output/progress seam; never infer readiness from arbitrary stderr text. Final budget exhaustion names the consuming stage and its elapsed time, while preserving the underlying error and existing retained attempt. Test stage transitions and stale/late progress with controlled time; no extra polling loop, timer renewal or free-form native content in status. Inspect the actual progress owner before choosing the smallest wire representation.

## State projection and ordering

Actions: wait, inspect the retained failed/unconfirmed attempt through the existing recovery action, detach/quit an admitted client.

| Internal state / failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Preparing, awaiting birth/readiness or registration, budget remains | existing starting operation | wait | none |
| Exact registration admitted | existing live row | detach/quit | normal existing quit semantics |
| Deadline/cancellation before target exec | existing failed start with phase/cause | inspect/recover using existing row advice | no new native work |
| Target started but readiness/registration absent or unknown | existing failed/unconfirmed attempt, retaining phase and underlying error | existing exact-attempt recovery action | native work may exist; no automatic relaunch |
| Owned client exits during admission | existing Pair-exited diagnosis and retained recovery state | existing row recovery action | startup incomplete; preserve acknowledged evidence |

No new user-visible state or recovery mechanism. Existing thread transitions and per-attempt lease remain the one lifecycle owner. Admission success racing startup cancellation rechecks the startup context before commit. Join the observer/client before releasing the lease on refusal; never infer absence from expiry. An admitted interactive client survives later startup deadline expiry.

## Operating envelope and architecture

- **ARCH-DRY/PURE:** one pure deadline/default owner and codec, separate from context/timer/IO glue. Existing readiness and registration validators continue to decide authority.
- **ARCH-PURPOSE:** sweep create, cold resume, fresh recovery, warm attach and direct OS handoff paths; no inner fixed startup timeout or registration renewal may remain. Optional saved-default persistence is a separate existing settings operation, not registration authority.
- **ARCH-MOCK:** stateful existing handoff/FakeRunner/native-query fixtures, plus landed #744 native conformance; private roots and real registrations only.
- **ARCH-CONSTRAINTS:** startup interaction; latency budget supplied by caller, absent-caller default reuses existing two minutes. Poll cadence remains 100 ms. For elapsed ticks T, at most O(T) bounded local metadata checks, no growth with fleet count. Assert poll/query counts at N and 2N virtual intervals and no native-query hot-loop on absent evidence. Existing short observation budgets remain five seconds. Memory/concurrency unchanged, one observer and client per admitted per-slot operation; network N/A.
- **ARCH-SECURE:** timestamp/env crosses exec and is strictly parsed with a small length bound; consume it once, never trust it as identity. Private fixtures never touch live fleet/native data. No credentials or native session content logged.
- **ARCH-ORDER:** preserve exact generation/lease and existing thread state transitions. Tests control ack, readiness, query completion, cancellation, helper/client exit and lease release independently. Most dangerous case: startup expiry after readiness but before registration; it cannot renew admission or kill an already adopted interactive session.
- **ARCH-FUNERAL:** no durable artifacts added. Contexts/timers/observers end with their phase; owned child handles are joined by existing handoff/helper cleanup. Fixture files/children are cleaned and asserted absent.
- **ARCH-EVOLVE:** no persisted format change. One ephemeral consumed environment value; existing readiness, failure, ledger and thread formats untouched.

## Implementation and verification steps

- [x] Add red production-boundary virtual-time regressions in `launcher/session_handoff_budget_test.go` (or a focused sibling): same production selector/entry used by runTerminalHandoffContext; ready at virtual 12 s under caller 30 s succeeds, while old ten-second production selection fails. No wall-clock sleep or custom short bound can stand in for the production selection.
- [x] Add controls: caller expiry/cancel before readiness; absent readiness reaches exactly the selected deadline; wrong nonce/session never admits; precreate-query cancellation does not end readiness; postbirth query is freshly short but capped by remaining startup; expiry during query refuses; early client exit joins observer before release; admitted client survives startup expiry. Record red results before implementation.
- [x] Add pure deadline/codec table tests and implement the shared deadline owner. Context/timer construction uses the selected absolute deadline; no per-phase renewal.
- [x] Wire explicit startup context through Runtime, birth watcher and handoff; preserve all existing short-query and cleanup tests. Capture exact deadlines at acquire/check/readiness/postbirth boundaries, not only a successful final return.
- [x] Add red Couch/helper transport tests in `couchcore/registration_liveness_test.go`, `launchhelper_test.go` and `runner_contract_test.go`: preparation and ack consume budget; exact timestamp reaches helper and Pair; delayed ack cannot extend it; early cancel/helper exit prevents admission; registration gets only remaining time; stale inherited value overridden; admitted subprocess survives deadline. Exercise both ExecRunner and PtyRunner shared boundary where applicable.
- [x] Wire tracked startup selection before helper launch and reuse it for registration. Adjust only tests whose old expectation depended on independently renewed windows, recording that reason in Log. Keep final-recovery observations separate.
- [x] Check the mechanical mistake class against existing Pair-local Go lint rules. Add a narrowly testable guard against using the probe-only birthBound as terminal admission lifetime, with positive/negative controls, if the existing lint catalog has no equivalent. Map its owner and limits in `atlas/go-lint-catalog.md`; do not introduce a blanket ban on context.WithTimeout.
- [x] Stage new reader tests before the fast affected-inventory audit; declare exact per-reader certificates and fixed-preflight consumers when needed. Run `make test-affected-inventory`, all launcher/handoff/birth and Couch helper/registration/recovery targets, including real subprocess boundary tests.
- [x] Update `atlas/couch.md`/index and relevant README startup-budget prose: caller deadline, fallback, phase bounds, admitted lifetime and failure action. Retain #744 incident limitations in issue Log.
- [x] After #744 lands on main, integrate its shared `testcouch` harness, never copy its unlanded source. Run one bounded disposable cold native Claude/Codex acceptance through real production registration and assert exact cleanup. Retain phase/readiness/error evidence on failure; no same-head blind retry and no live fleet/pair:0 operation. This native acceptance remains owed if the harness is still pending; ask the hub before changing it.
- [x] Push final head and pass recorded affected/full-suite verification. Mandatory boundary review, merge on SHIP and the verified hub outcome report remain required checkpoints. Hub owns project updates.

Targeted commands (adjust exact new test names as authored):

```sh
go test ./cmd/internal/startupbudget -count=1
go test ./cmd/internal/launcher -run 'Test(TerminalHandoff|Couch(Create|Attach)|.*Birth|.*StartupBudget)' -count=1
go test ./cmd/internal/couchcore -run 'Test(ColdResume|LaunchHelper|.*Registration|.*StartupBudget|.*Runner)' -count=1
make test-affected-inventory
```

The native harness's landed command and current API will be read before use; its existing bounded opt-in fleet target is the acceptance owner, not a second fixture.

## Revisions

### 2026-10-21 — approved budget and required progress

Hub approved the absolute deadline/default/exec/lifetime contract and added stage plus elapsed-time progress and final failure attribution because an actually broken agent may now consume the existing two-minute ceiling. Implement this on the existing operation progress seam, with deterministic tests, before close. Native acceptance still waits for landed #744.

### 2026-10-21 — progress seam as implemented

The existing per-operation context reporter feeds an explicit per-request callback into the receipt reducer. Added optional immutable stage/start time there, preserving same-stage age and rejecting terminal or retired-incarnation updates; existing CLI cadence formats it. No launcher stdout parsing, new polling loop or persisted format. Native settled-composer waiting after adoption remains its existing distinct post-restart confirmation contract; terminal birth/readiness/registration use the shared startup budget.


### 2026-10-22 — implementation and verification completed

Resumed from the checkpoint after #738 and #739 preparation. All implementation and test steps above are now evidenced in the issue Log: integrated targeted tests, one real private cold Claude/Codex acceptance through the landed shared harness, exact cleanup, and corrected recorded full suite PASS10m48s. Ticks record completed work, not a review or landing claim. The boundary review and normal merge are still mandatory. Related missing-ready/restart-registration captures remain separately classified and do not prove a common cause.


### 2026-10-22 — round1 consumer sweep correction

Reopen the warm-attach, console-progress and failure-fixture verification claims after the review's measured counterexamples. Tracked lifecycle launches and public borrowed-client attachment are distinct production consumers of the same selected deadline. The latter must select once before its helper preparation/retry loop, serialize the absolute value, cap existing short native queries, and consume remaining time without bounding the admitted client's lifetime. Preserve all terminal/registry ownership and existing helper retry rules.

Remote receipts and the local Console are distinct progress consumers: each retains a stage-start timestamp and projects that age on repeated current-stage reports; no new timer or IO. The console's operation-start time stays as fallback for ordinary no-stage work. General warm-adoption failure fixtures must trigger the named observation, not expire during filesystem preparation; dedicated budget-propagation controls use controlled time and assert remaining absolute budget. Add/regress each class, then repeat full recorded verification and boundary review. No scope change, new recovery state, executor, timeout increase or native-startup-cause claim.


### 2026-10-22 — correction evidence ready for the gate

All three round1 consumer corrections now have red/green regression proof, full core/tty package passes, focused race controls and a new successful shared native Claude/Codex matrix including detach/restart and joined exact cleanup. See issue Log and native-warm-88f7326491b1 evidence. Prior recorded green predates these runtime changes; the corrected recorded suite and new boundary review are still required. The ticks describe completed implementation/targeted/native work, never waive that final recorded gate.
