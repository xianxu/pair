---
id: 000704
status: working
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
severity: medium
collaboration-mode: collaboration
card_mirror: 'c1e9d9430bb44e8de96404e7c382af686cb712cb' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T08:12:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "6e340b9c", done: "9d5fcc5c"}
---

# TestParkLifecycleLive fails locally on main: quit intent does not match observed session owner

## Problem

TestParkLifecycleLive fails deterministically on the operator's machine on unchanged main, while CI's macOS conformance job passes it. It fails on the operator's machine on every run so far: 5/5 on the #692 branch (`-count=5`) and 3/3 on origin/main 04e2216a in a clean detached worktree (2026-10-19). The sibling TestParkLifecycleLiveIntentOnlyMutation passes 5/5 in the same runs. #669 covers a different native test (continuation registration timeout) and does not cover this one.

Exact output (first failure; each repetition is identical apart from temp paths and timing):

```text
=== RUN   TestParkLifecycleLive
    park_lifecycle_live_test.go:237: quit intent does not match observed session owner
    park_lifecycle_live_test.go:481: owned helper PTY diagnostics: "...zellij screen with the owner stub pane: /tmp/pair-owner-stub-1189795670/pair wrap --scrollback-log /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/TestParkLifecycleLive2876366904/..."
    recovery_query_capture_test.go:249: recovery query capture: {"Records":[{"Kind":"lifecycle","Name":"fixture-roots home=\"/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/TestParkLifecycleLive2876366904/001/home\" ...
--- FAIL: TestParkLifecycleLive (0.67s)
```

Command: `PAIR_LIVE_COUCH=1 env -u COUCH_THREAD_SCOPE -u COUCH_THREAD_TAG -u ZELLIJ -u ZELLIJ_SESSION_NAME go test ./cmd/internal/couchcore -run '^TestParkLifecycleLive$' -count=3 -v`, run unsandboxed from a Couch slot (pair:2). Host: macOS 27.0.1 arm64, zellij 0.45.1, go1.27.1, TMPDIR=/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/.

Full retained log (both tests, five repetitions, complete query captures): /Users/xianxu/workspace/pair/.git/pair-parklive-evidence/local-TestParkLifecycleLive-count5.log, SHA256 af9d32b462c4e5068e00b910277ce90a5d070ee78126b1299f74299800fcfaf5.

The refusal is `ScopedThreadArtifactCollisionChecker.TriggerBoundQuit` (cmd/internal/couchcore/artifactcollision.go): the observed binding's Owner is nil, or its owner DataDir/RepoScope/Tag differs from the intent's. Not yet diagnosed. One candidate to check, not a conclusion: the owner observation and the intent may spell the data dir differently (/var/folders versus its canonical /private/var/folders), or the owner probe may not see the owner stub's process on this host. An environment-dependent difference between the operator's machine and CI is still a wrong assumption in Pair or its fixture.

## Spec

## Done when

- The mismatching field (owner missing, or which of DataDir/RepoScope/Tag differs, with both values) is identified from retained evidence on the operator's machine.
- The cause is fixed in production or the fixture with a deterministic regression, and TestParkLifecycleLive passes locally and in CI.

## Plan

### Diagnosis (2026-10-19, pair:2)

Temporary instrumentation (reverted) on the operator's machine:
- At the refusal the probed owner is `State: Absent` with an empty owner. Neither `list-sessions` nor the process scan sees the fixture session: no zellij process carries its session name or private socket root. The owner fields are not mismatched; the session is gone.
- A probe in `DeliverTrigger` shows the order. Delivery 1: the session is listed, its server is alive, the helper is live. Delivery 2: "No active zellij sessions found.", no server, the helper is dead.

`RunConformanceScenario` delivers the trigger twice (redelivery after restart). The live `TriggerBoundQuit` quiesces the session (`DeleteSession`) to return control to Pair's handoff, and here Pair's quit cleanup finishes before delivery 2. Delivery 2 then probes an absent session, and the guard reports it as "quit intent does not match observed session owner". CI passes only because its server is still visible at delivery 2. This is not an ownership or layout bug: the outcome depends on how fast the first trigger's quiescence finishes.

Production already handles redelivery. `park.go` triggers only when the observed binding is `Present`, and an absent session goes straight to awaiting completion. The test drivers diverge from production in two ways:
- The live driver triggers unconditionally.
- The fake's `DeliverTrigger` leaves the session in place until cleanup, so it never shows the vanished-session state that the real trigger produces (ARCH-MOCK).

### Fix

- [x] Fake fidelity (deterministic red). `Fake.DeliverTrigger` quiesces the session, as the production trigger does. Under today's drivers the fake conformance scenario then fails on every run at delivery 2 (session absent), reproducing the live failure without timing.
- [x] Drivers follow production's delivery rule: observe the session and trigger only when it is present; a delivery that finds it absent does nothing, as park.go does. The live driver observes through `NamedPairSessionContext` and triggers with `TriggerBoundQuit` on that binding. The intent-only variant keeps its write-only trigger behind the same gate.
- [x] Production diagnostic. `TriggerBoundQuit` given an absent binding says the session is absent and there is nothing to trigger, instead of "quit intent does not match observed session owner". Production never calls it on an absent binding, so this changes no behavior.
- [x] The fake unit test that expected two raw trigger effects now expects the second to be refused as absent. Verify with pairlifecycletest and couchcore tests, the live park tests locally (TestParkLifecycleLive should pass here now), and the full suite.

This is not slot ownership, so pair:1 is not affected.

## Log


- 2026-10-19: closed — Test-driver and diagnostic fix, no new architectural surface. sdlc-local/suite green on 647a52a5 (14m6s). Diagnosis by temporary instrumentation (reverted): the second scenario trigger probed an already-quiesced session (owner Absent, no server), not an owner mismatch. Red: without the present-only gate the fake conformance scenario fails every run at delivery 2. Live on the operator machine: TestParkLifecycleLive 5/5 and TestParkLifecycleLiveIntentOnlyMutation 5/5 (previously 0/8). TestTriggerBoundQuitNamesAnAbsentSession; pairlifecycletest and couchcore park tests green. (done-when: contract moved since change-code; judged by the close review); review verdict: SHIP
2026-10-19 — Filed by pair:2 at the hub's request while landing #692; observed during #692's live verification.

2026-10-19 (pair:2) — Built. Moving the quiescence into the shared `Fake.DeliverTrigger` broke TestParkInterruptionBoundaryMatrix/after-trigger: Couch's session view there comes from the separate `Artifacts` fake, so the two views diverged. The quiescence therefore lives in `fakeConformanceDriver.DeliverTrigger`, the adapter that models production's trigger, and the shared Fake and its unit test are unchanged (see Revisions).
- Red: without the presence gate, TestQuitLifecycleConformanceScenarioFake fails on every run with `exact session "pair-work" is absent` at delivery 2, the live symptom.
- The live driver observes through `NamedPairSessionContext` and triggers only a present binding with `TriggerBoundQuit`.
- `TriggerBoundQuit` names an absent binding (TestTriggerBoundQuitNamesAnAbsentSession).
- Live, on the operator's machine: TestParkLifecycleLive 5/5 and TestParkLifecycleLiveIntentOnlyMutation 5/5 (`-count=5`). TestParkLifecycleLive was 0/8 before.
- pairlifecycletest and couchcore park/session-owner tests are green.

## Revisions

2026-10-19 — Fix bullet 1 narrowed: the trigger's session quiescence is modeled in the conformance driver, not the shared Fake, so the unit-test harnesses that pair the Fake with a separate Artifacts view keep one consistent session view. Bullet 4 (fake unit test change) was not needed.
