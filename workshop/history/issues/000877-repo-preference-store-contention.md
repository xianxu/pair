---
id: 000877
status: done
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '856e2019bbaa6f484d9de680cb4b9fbf61206fd5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T11:11:57-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcore.TestRepoPreferenceConcurrentInstancesPreserveOtherRepositoriesAndAgents]
flow: {kind: quick, provenance: inferred, spec: "7b2652d2", done: "7a67b1e1"}
actual_hours: N/A
---

# Repository preference publication escapes typed store contention

## Problem

During #866 full local Go verification, TestRepoPreferenceConcurrentInstancesPreserveOtherRepositoriesAndAgents failed at repopreferences_store_test.go:123 with lock thread store: thread store busy; retry. The test starts16 independent writers (8 repositories x Claude/Codex) and expects all publications to succeed. The exact test also reproduced alone, failing after1.12s (package1.496s).

Repro: go test ./cmd/internal/couchcore -run ^TestRepoPreferenceConcurrentInstancesPreserveOtherRepositoriesAndAgents$ -count=1 -v -timeout=30s . The terminal period is prose punctuation, not an extra package argument.

rememberRepoConfigurationContext retries only unix.EWOULDBLOCK/EAGAIN at repopreferences_store.go:86-94. withUpdateLockContext uses the fixed1s storeReadLockWait for its nested store lock and can return typed ErrThreadStoreBusy instead. That error escapes unchanged. These source/test files are unchanged from main on #866 candidate b6f9b48d12d5c06eb09f9dbbe5d36a4ea00c3f35. Source log /Volumes/TestScratch/pair866-full-go-local-gr31bcx7/go.log (full run still finishing when filed).

pair:6 confirmed #854/PR590 does not cover this: it caps waits at a caller deadline, but context.Background has none, so the1s busy result remains intentional. #873 only changes startup acquisitions. TL266b90f8 directed separate unclaimed filing; do not fold this into Zellij startup readiness.

## Spec

Resolve the mismatch between concurrent repository-preference publication and the busy-store contract. Define the caller-budget/busy behavior explicitly and preserve context cancellation and the existing bounded store-wait policy. Do not silence the failed publication by deleting assertions or increasing a test timeout blindly.

## Done when

- Concurrent independent preference publications preserve unrelated repositories and per-agent arguments under the documented busy/caller-budget policy.
- Deterministic contention coverage distinguishes retryable contention, caller cancellation/deadline, and a genuine bounded busy refusal.
- The existing concurrent-instance test completes under that contract and targeted store-wait tests still pass.

## Plan

Root cause (measured from history): #728 (4a1d1bf76) wrote the publication loop when withUpdateLockContext's nested store lock was nonblocking, so contention surfaced as EAGAIN and the loop retried it within the caller's budget. The next day #827 (ec8bcd7f4) gave that nested lock a 1 s timed wait that ends in the typed ErrThreadStoreBusy. The loop never learned the typed form, so 16 serialized writers whose queue exceeds 1 s now escape busy instead of retrying. The policy is intact; the loop's contention test is stale.

- [x] RED: tests in repopreferences_store_test.go with storeReadLockWait shortened, store.lock held by the test:
  - retryable contention: lock held past one wait, then released, Background ctx: publication succeeds and preserves other entries;
  - bounded busy refusal: lock never released, ctx deadline: error Is context.DeadlineExceeded and Is ErrThreadStoreBusy, nothing written;
  - cancellation: ctx cancelled while waiting: error Is context.Canceled, nothing written.
- [x] Fix: one predicate isStoreContention(err) (ErrThreadStoreBusy, EWOULDBLOCK, EAGAIN), shared by retentionLockError and the publication loop (ARCH-DRY). The loop retries contention until the caller's ctx ends, then returns ctx.Err joined with the last contention error, so the adoption warning says the store stayed busy. No timeout bump, no assertion change; storeReadLockWait and maintenance wait 0 unchanged.
- [x] TL condition, no unbounded retry: the loop refuses a context without a deadline (a future caller cannot spin), and the test wrapper rememberRepoConfiguration passes a bounded context instead of Background; a test pins the refusal.
- [x] Verify: the concurrent-instance test -count=20 green; store-wait tests; atlas couch store-lock note.

Composition with #854 (PR 590, unlanded): there the inner wait is capped at the deadline and returns DeadlineExceeded, which is not contention, so the loop returns it unchanged; the second lander integrates.

## Log


- 2026-10-24: closed — RED first: RetriesTypedContention and RefusesBusyAtTheDeadline failed with thread store busy; retry, RefusesAnUnboundedCaller failed (caller waited on the store). After the fix all four contention tests pass (-count=3); targeted store/retention/preference/edited-restart set passes; TestRepoPreferenceConcurrentInstancesPreserveOtherRepositoriesAndAgents -count=20 green (32.4 s; pair:9 measured 17/20 failures before). make test-affected-inventory clean. Pre-close: sdlc-local/affected pass on 4e87d68f (25 packages + lua/review/shell groups; known flake #873 rerun passed, attempt 237737ab). --no-actual under TL standing authorization: measurement refused with "measurement snapshot does not serialize: json: error calling MarshalJSON for type *gatestate.MeasurementInputs: measurement exceeds 65536 bytes" (ariadne#593); no window printed.; review verdict: SHIP
Filed unclaimed from #866; no fix made in that branch. Reported exact isolated reproduction exit1 and full-set failure. No flake classification claimed.

pair:9 evidence (TL relayed; registration does not persist it): suite attempt 65d6f942 on 912ce21e failed in full (27m32s); targeted runs of TestRepoPreferenceConcurrentInstancesPreserveOtherRepositoriesAndAgents failed 17 of 20 in 22.5 s.

TL approved the plan with the no-unbounded-retry condition above.

RED (before the fix): RetriesTypedContention and RefusesBusyAtTheDeadline failed with "lock thread store: thread store busy; retry"; RefusesAnUnboundedCaller failed (the caller waited on the store); StopsOnCancellation already held. After the fix all four pass (-count=3), the targeted store/retention/preference set passes, and TestRepoPreferenceConcurrentInstancesPreserveOtherRepositoriesAndAgents passes -count=20 (32.4 s). The deadline case first returned a bare context error when the deadline fell inside the inner wait; the loop now joins the last busy whenever the context ends after contention. editedrestart_publication_test.go's direct Background call now uses the bounded test wrapper.
