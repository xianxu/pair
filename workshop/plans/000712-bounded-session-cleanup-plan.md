---
collaboration-mode: discussion
---

# Bound failed-start session cleanup

> **For agentic workers:** Consult AGENTS.md Section 3 for execution/delegation. SDLC owns the boundary review.

**Goal:** A failed session observation cannot keep a launch caller waiting forever; retain unresolved ownership, record and surface the failure, and repair the fake that exposed it.

**Architecture:** Prove the exact helper dead using the existing process cleanup owner. Give the following session-quiescence phase one independent five-second deadline, including retries and native deletion. On expiration retain the exact thread/start reservation as unknown and return a typed unproven cleanup failure through the existing lifecycle recorder and receipt/display path. No background retry task or replacement launch is created.

**Tech Stack:** Go contexts, existing Couch thread transactions/lifecycle diagnostics, stateful artifact and Zellij doubles, disposable filesystem roots.

## Evidence and operating envelope

- Retained attempt d963805085758c53cfbbab3363ad6432 on f2333e342cc325ded2344fe69dd0d7309224dca7 spent 1172.59s in fresh-slot. At that revision couch.go:966 is the session retry, after helper cleanup completed.
- The unchanged composed test passes today in 2.71s (ordinary 1.32s, fresh-slot 1.39s); no unchanged replay proves the hang fixed.
- Owned probe measured 129 consecutive session calls, zero deadline-bearing contexts, and 128 retry sleeps; it returned only when the probe changed refusal to success. The loop explicitly has no total deadline. A real persistent error has the same behavior.
- A real fixture subprocess queried after PAIR_SESSION_NAME restoration emitted ` [Created 1s ago]` with exit 0. The listing depends on caller environment and a stale marker, not server identity/lifetime.
- The five-second total session budget reuses the existing recovery/native observation scale. It begins after exact helper quiescence, is independent of cancellation of the failed launch, and is not extended per retry. A slow/unknown observation becomes unproven, never absent. UI/receipt completes after the bounded session phase and bounded existing diagnostic storage, rather than hours of retries.
- Each attempt is sequential: at most one native observation/deletion in flight. Retries wait the existing 100ms between attempts; a timed-out context ends that delay. Work is O(attempts), with the one shared deadline as its bound; there is no queue or fan-out.
- This issue bounds the observed session retry. Exact helper termination continues retaining its handle until reaped; that distinct process owner cannot be abandoned by an arbitrary timeout. The dump proves it had already completed in this incident. A permanently unkillable helper is not represented as a successful session cleanup.

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| SessionCleanupUnconfirmed | cmd/internal/couchcore/startcleanup.go | new |
| StartCleanupInput/DecideStartCleanup | cmd/internal/couchcore/startcleanup.go | modified |

- SessionCleanupUnconfirmed names exact thread/session and wraps the deadline and last refusal. It implements existing LifecycleFailureDetails with phase session-quiescence and outcome unproven; it never authorizes deletion. Pure tests cover error identity/details without IO.
- Cleanup knowledge must distinguish proved quiescence from an expired/unproven cleanup. Add an input representing that knowledge to the existing decider, so no owning shape can roll back/retire while its cleanup is unconfirmed. Default/unknown fails closed. Unit table tests cover every shape, claim/live phase, helper and presence combination. Existing successful cleanup behavior remains explicit.

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| quiescePostAckStart | cmd/internal/couchcore/couch.go | modified | exact helper drain, bounded session phase |
| applyStartCleanup | cmd/internal/couchcore/couch.go | modified | retained thread transaction |
| QuiesceNamed | cmd/internal/couchcore/artifactcollision.go | modified | native owner revalidation/deletion |
| launch failure projection | cmd/internal/couchcore/launchfailure_capture.go | modified | existing bounded operator line |
| registration terminal fake | cmd/internal/couchcore/terminal_registration_fixture_test.go | modified | persisted fake server identity/lifetime, real owned children |

## State and recovery

Actions: inspect with couch --show; use only the existing slot-recovery outcomes continue, resume from disk, or start fresh when current evidence permits them. No new recovery outcome or unconditional retry/relaunch is introduced.

| Internal result | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| helper dead, session cleanup proved | existing launch failure/disposition | inspect original failure | only this start's proved resources removed |
| helper dead, session error persists to deadline | cleanup unconfirmed; ownership retained; diagnostic nonce | inspect, then existing recovery after evidence permits | no session or reservation discarded |
| unrelated/replaced server observed | same unproven refusal | inspect exact diagnostic | no foreign server signalled |
| diagnostic publication fails | named failure plus diagnostics-unavailable detail | inspect Slots/process state | diagnostic history may be unavailable; ownership retained |

The thread gate remains held through cleanup disposition. At the deadline, a claim keeps its nonce/proposed binding and an unknown incarnation; a live record keeps the exact incarnation marked unknown. Do not globally reconcile or perform fresh native reads after this deadline merely to decide whether to discard it. Existing later reconciliation/recovery may resolve it using new evidence. No new artifact family, worker, retry queue or automatic replay is introduced (ARCH-FUNERAL).

## Function strategies and mechanical guards

| Function/seam | Strategy | Adversarial regression/guard |
|---|---|---|
| quiescePostAckStart session phase | one context for all attempts and waits, fixed five-second maximum | stateful QuiesceNamed refuses forever; controlled cancellation after entry must return typed error; every retry sees identical deadline |
| helper phase | preserve one Wait owner and exact KILL/reap contract | existing delayed sidecar/three-KILL and one-waiter regressions remain passing |
| QuiesceNamed | pass supplied context into contextual SessionDeleter; production OSRuntime supports it | block deleter until ctx.Done; cancellation must reach the native deletion seam; no background-context reset |
| applyStartCleanup | unproven knowledge routes to existing unknown-state writes only | matrix over spawn/cold/fresh/warm, claim/live, absent/present/unknown; retained exact nonce/session and no rollback/reconcile calls on unproven |
| lifecycle recorder/operator line | retain CleanupError and typed unproven phase/outcome; concise line names cleanup uncertainty and inspect action | public failed launch returns; history contains original launch error and cleanup refusal; receipt and Slots/show expose same nonce and unproven diagnosis |
| fake listing | derive exact name from persisted server/readiness record, never querying PAIR_SESSION_NAME | unset and poisoned caller session env produce same exact listing; missing/ended server produces canonical no-sessions reply |
| fake lifetime | release/reap ends authoritative listing; saved readiness is not liveness | stale marker with removed/reaped server, delayed release, injected refusal, ordinary/fresh composed cleanup all join |
| fixture publication | retain #698 prerequisite-before-ready ordering | existing synchronous between-write test remains passing; both failed write cases preserve ordering |

No production format fields are required: reuse existing incarnation unknown state and lifecycle Phase/CleanupError/Outcome. Existing codecs preserve unknown members and strict known checks (ARCH-EVOLVE). Thread names and native observations remain untrusted; failure does not grant cleanup authority (ARCH-SECURE). Fakes exercise the production entrypoints rather than replacing registration or ownership policy (ARCH-MOCK).

## Tasks

### 1. Deterministic failure tests

- [ ] Preserve the probe results in the issue Log, then remove the throwaway measurement test.
- [ ] Add failing persistent-error/deadline tests with the existing stateful artifact fake, plus literal cleanup decision tables. Use controlled entry/cancellation; no sleep-based speed assertion.
- [ ] Add a fake listing regression with restored/poisoned caller environment and ended server state. Reproduce the composed failed launch under ordinary and fresh-slot starts; capture its repeated cleanup error before repair.
- [ ] Run the named tests and retain expected RED diagnostics; fixtures release and join on every path.

### 2. Bounded session phase and retained ownership

- [ ] Add SessionCleanupUnconfirmed and cleanup-knowledge input/decider cases.
- [ ] Refactor only the session phase to use one bounded context. Keep helper drain unchanged. Retain last refusal and exact identity on expiration.
- [ ] Use the contextual native deleter and retain exact unknown thread/start state on unproven cleanup without further session reads.
- [ ] Surface the named error through existing lifecycle details and bounded operator projection. Test public receipt/history/show behavior using isolated service composition.
- [ ] Run focused tests and source guards, and prove omission of the shared deadline or unknown-state retention makes the new regressions fail.

### 3. Fixture lifecycle and verification

- [ ] Implement persisted session identity/lifetime in the existing terminal fake; preserve canonical reply rendering and #698 ordering.
- [ ] Run the original composed test and deterministic failed-launch variants with -race. Verify all owned server/wrapper/agent children are reaped before test return.
- [ ] Run the composed family 20 times under bounded owned CPU load (20 workers, watchdog, kill-and-Wait in defer/cleanup on every exit). No native Zellij or live Couch.
- [ ] Update atlas/README for bounded failure/recovery behavior and lessons. Add any applicable mechanical API guard at the shared lint owner; do not invent a syntax rule for a semantic fixture lifecycle bug.
- [ ] Push, early draft PR, recorded suite, SDLC close review, normal merge gates; report the verified merge to the hub.

## Revisions

2026-10-19 R1: hub receipt ba6e71aa-42d2-416a-a7ab-fe8ee2cf9829 approved test-side work and required the real production retry to be measured and bounded if unbounded. The measurement confirmed it is unbounded, so this design supersedes the earlier test-only scope; implementation waits for hub approval of the production behavior. The initial quick-flow gate admitted only diagnostic/test investigation. Re-enter full-flow change-code before production edits.

2026-10-19 R1 approval: canonical hub receipt 8937d10b-c9c6-4bbd-a243-b5163bdc0a36 approves the five-second session phase and retained unknown ownership. Recovery wording explicitly reuses continue, resume from disk and start fresh; it creates no fourth outcome. Re-enter the full-flow gate now.
