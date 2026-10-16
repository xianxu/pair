---
id: 000590
status: done
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
card_mirror: 'a81c5a0052ab078a519ad1f088b90bf7bd20939f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T01:10:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "855e8091", done: "a8d02e3f"}
actual_hours: 0.44
---

# Diagnose concurrent launch diagnostics lock timeout under load

## Problem

`TestLaunchFailureStoreReopenConcurrent` intermittently fails under full-suite load when one of 17 concurrent diagnostics Save calls times out acquiring the store lock. The later retention assertion also fails because that failure record was not saved. This is independent of #571's Codex adapter; the diagnostics store and its test are unchanged by that branch.

## Spec

The production one-second admission timer conflates scheduling and serialized durable publication with a failed lock owner. Seventeen valid writers already consume about half that budget in measured critical sections. Make admission wait on the actual lease with a caller-supplied context, rather than a store-owned wall-clock guess. Retain safe open, exclusive flock, atomic publication and exact retention.

Diagnostics must not postpone failed-helper cleanup: snapshot immutable evidence in memory, perform cleanup, enrich the record, then save once. The diagnostic caller supplies a separate ten-second context even when the launch was canceled; that bounds admission, not an already admitted filesystem operation. Direct storage callers supply their own context. Cancellable nonblocking flock attempts avoid an uncancellable blocking syscall or orphan waiter goroutine. The existing timer is only a wake mechanism; tests control actual failed-acquisition/release/cancellation events without sleeps.

Removing the pre-cleanup publication permits loss of the current attempt if Couch crashes during cleanup or before its final publication. Existing published history remains durable. No recovery authority depends on this optional record; helper cleanup takes priority. There is no new artifact family or retention policy.

## Done when

- Controlled real-lock regressions prove waiting, release, cancellation and no late acquisition or mutation after refusal; admission uses the caller's context.
- Concurrent valid diagnostics writes keep the exact newest sixteen records. A genuinely held lease refuses on caller cancellation/deadline without changing history.
- Failure capture performs cleanup before its one save, retaining immutable pre-cleanup evidence and separate exit observations; persistence failure preserves the original and cleanup error chains.
- Targeted and recorded suite results report the measured baseline and final failure counts; no test sleep, retry, skip or enlarged test budget.

## Plan

- [x] Add caller-context admission to the storage interface, real store and stateful fake; a small injected wait boundary controls real contention in tests (ARCH-PURE). Preserve the existing safe-open and unlock ownership (ARCH-DRY).
- [x] Red/green controlled held-lease release/cancellation tests and cleanup-before-save/immutable-evidence tests; migrate existing exact retention and unsafe/corrupt/durability coverage without relaxing assertions.
- [x] Snapshot evidence before cleanup, enrich and publish once afterward, using a named bounded diagnostic context independent of launch cancellation. Record the explicit crash-loss tradeoff (ARCH-CONSTRAINTS); enumerate waiter cancellation versus lease release, with no background waiter surviving return (ARCH-STATE).
- [x] Update Atlas and run targeted/race checks and the recorded suite.

After these implementation steps: request the mandatory close review, prepare SHIP and wait for the hub's final merge turn.

## Revisions

### 2026-10-16 — measured contention and cleanup-first policy

The TL accepted measured serialized contention as the bug class and proposed cleanup before a single context-bounded save. Replace the original store-owned one-second timeout contract with caller-owned cancellation, preserving refusal of a genuinely held lease. A larger test timeout alone would leave the production bug; an unbounded blocking flock before cleanup could strand a failed helper. The ten-second production diagnostic context is an explicit caller policy, not the test's scheduling oracle. ARCH-SECURE and ARCH-FUNERAL preserve current safe-path/regular-file checks and sixteen-record/one-MiB bounds; ARCH-MOCK reuses the stateful fake and real temporary-directory integration tests. No new external binary or durable state machine is introduced; the existing flock/publication protocol owns persisted history (ARCH-PURPOSE).

## Log

### 2026-10-16 — bounded read-only measurement before claiming
- 2026-10-16: closed — Production lock admission now follows caller cancellation; cleanup precedes its single diagnostic save. Controlled real-flock virtual-time red/green reproduces the former one-second loss without wall sleeps; held-lock release/cancel/deadline and immutable evidence/error-chain regressions pass. Targeted/race checks and ten concurrent repeats pass. Clean make -k -B build verified vcs.revision fb047bfeb556b0c3e0224be700a14089f70489f0. Recorded full suite 45a1dd14add46c7fa2a88560dc75044a passed all recipes/packages in 14m48 on that source, log SHA256 c3a47b88b22be7efa5bacb61f6bee1dab8f86f21600b6b17ef7b9e83834e587f; subsequent changes are issue evidence only. Atlas documents the crash-loss/admitted-IO bounds. No project write: TL owns ops/test-economy and will tick it. Final merge awaits hub turn.; review verdict: SHIP

- Source verified with git rev-parse: `1a1047c89126c68fa1f9c2651bff36663f6ab656`. Overlay-only instrumentation: ten samples, 170/170 writes passed, package 5.952s. Maximum admission wait 576.162ms; median/max lease hold 26.205/55.326ms; atomic publication accounted for 98.2% of hold time; total held time per seventeen writes 386.897–537.793ms. Measurement includes descheduling and does not separate filesystem latency from scheduling. No failure was reproduced in this bounded probe. O_CLOEXEC and explicit LOCK_UN/Close are intact; inherited retention was not observed or proven.
- Local evidence: `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair590-lock-measure.ueOfd0/` (overlay, source identity, metrics, summaries and run log). No tracked code or main publication changed during the #605 hold.
- Additional main baseline failed this test in 1.14s on `e7b676abb2083791028bed45eda22515361cca4c`, run `6e6e954f55eb8e7ba8dacebde0bf0e07`; log SHA256 `86910833ebc459007a65b3b46b05b4adcdfbd550fb1bd68efd64124da824a013`. Every remaining recipe/package completed. The one-second timer is production policy, not a test-only deadline.
- TL released claim/start-plan while retaining the final merge hold. Released deferred #591, claimed and started #590. Posted the small context-admission/cleanup-first plan to the hub (receipt `46dd25fb-513e-4b9f-9d00-4d8924edc930`); dispatch explicitly permits proceeding after posting. Pre-cleanup persistence has no recovery consumer; its crash-window guarantee is intentionally traded for prompt cleanup.

### 2026-10-15 — first seen during #571 corrected-code full suite

- Exact tested head: `8d79c161e1ba68c07b5884fefb657e13272b145f`.
- Test `TestLaunchFailureStoreReopenConcurrent`, `cmd/internal/couchcore/launchfailure_store_test.go:32,42`, failed after 1.10s: `launch diagnostics lock timeout: resource temporarily unavailable`; retention then lacked a required record.
- Recorded full-suite log: `/Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/fe1008b1860377ef1d9a72d918548b57.log`; SHA256 prefix `73f371f8e9a9`; runtime12m57s, couchcore481.625s. Every recipe/package completed. Its other failure was #571's separately corrected protocol-vocabulary inventory classification; broadcast and review restoration passed.
- One targeted unchanged repeat: `go test ./cmd/internal/couchcore -run '^TestLaunchFailureStoreReopenConcurrent$' -count=10` passed all10 iterations, package5.041s. Log `/tmp/pair571-launchfailure-lock-check.log`.
- No matching active issue found. Filed unassigned at the TL's direction. Do not fold a diagnostics-store fix into #571; its already queued corrected-head suite may proceed, and any further unrelated flake stops retries for a suite-wide test-economy review.

### 2026-10-16 — implementation and deterministic red/green

- Failure capture now snapshots immutable evidence, cleans up, then performs exactly one capture publication. Storage Save accepts the caller context; safe flock admission checks cancellation before attempting and after obtaining the lease. A cancellable timer wakes another attempt without an orphan blocking goroutine. Couch gives optional post-cleanup admission its named ten-second policy; launch cancellation does not suppress the evidence. Admitted atomic publication is not interrupted midway.
- Original cleanup-order red: TestLaunchFailureCaptureSavesAfterCleanup failed with "persistence postponed cleanup" against the prior implementation (package 0.251s). Updated capture passes, including a deliberately blocked save after cleanup, erased/mutated pre-cleanup output, separate before/after exits, and original/cleanup/persistence error identity.
- TestLaunchFailureStoreWaitsBeyondFormerBudget holds a real flock, advances only testing/synctest's virtual clock by two seconds, observes another real failed acquisition, then releases. Green package 0.282s (case 0.04s). Negative control reinstating the former one-second admission check through a temporary source overlay fails with the original "launch diagnostics lock timeout: resource temporarily unavailable" (case 0.01s). Evidence overlay: `/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair590-red.fvEhRw/`. No wall-clock sleep or test-budget increase.
- Additional real-lock cases control release, cancellation, spent deadline and simultaneous-ready release/cancellation. Refusal preserves history and releases every acquired descriptor. Existing concurrent fixture keeps its exact newest-sixteen assertion and supplies its test context instead of receiving a private performance budget.
- Targeted launch-failure coverage across couchcore/couchcmd passed (4.001s / 1.009s); final race coverage passed (5.783s / 2.045s). Ten unchanged concurrent-retention iterations passed (170/170 writes, package 4.546s), compared with the measured ten-sample baseline's 170/170 successes and separately recorded broad-suite failures above. `make -k -B build` passed. Atlas documents context admission, cleanup ordering, durability and the explicit crash-loss window. Full recorded suite and boundary review remain pending; final merge still awaits the hub's landing turn.

### 2026-10-16 — recorded broad suite passed

- Exact source `fb047bfeb556b0c3e0224be700a14089f70489f0` (git rev-parse). `make -k -B build` rebuilt that clean source; bin/pair reports the same vcs.revision and vcs.modified=false.
- Required `sdlc local-check run --name suite`, run `45a1dd14add46c7fa2a88560dc75044a`: PASS, exit 0, wall 888.345766292s (14m48), queue 100.791711209s. All shell/editor recipes and Go packages completed; couchcore 510.066s, couchcmd 148.912s, no failures. The broad run includes the controlled original-budget regression and exact concurrent retention fixture.
- Retained log `/Users/xianxu/workspace/pair/.git/sdlc/local-checks/fb047bfeb556b0c3e0224be700a14089f70489f0/suite.log`; full SHA256 `c3a47b88b22be7efa5bacb61f6bee1dab8f86f21600b6b17ef7b9e83834e587f`. No repeat was needed. The previously cited startup-cancellation, review-resume, observation, fresh-restore and installed-fleet conformance failures did not recur.
- Implementation/verification checklist is complete. Close review, SHIP and final merge are workflow checkpoints; final merge still waits for the hub. The TL owns ops/test-economy, so its project tick is reported to the hub rather than written by this worker.
