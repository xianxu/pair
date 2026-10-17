---
id: 000631
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
card_mirror: 'edacdc16553c1f815c726bcefbcfcc85c21a2843' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T01:16:20-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "94d91913", done: "3d36ed4e"}
---

# Peer promotion holds broker lock during wrapper release RPC

## Problem

During #629 investigation, Broker.promote was found calling the wrapper endpoint's Release RPC while holding the shared broker mutex on its error path. A slow Release can delay independent recipients' Send and Status calls until the RPC timeout.

This is a separate static finding. No retained evidence ties this lock contention to the 2026-10-16 17:37–18:22 hub outage; #629's confirmed mechanism is serial pre-paste expiry.

## Spec

Move wrapper release IO outside the shared broker lock while preserving exact actor/reservation ownership and promotion retry pacing. Use a gated stateful endpoint to prove another recipient's admission and status can progress while Release is withheld.

## Done when

- A gated Release test fails on the original locking path and passes after the change.
- Another recipient completes Send/Status while the first receiver's Release is blocked.
- Existing promotion/reconnect/duplicate-ID tests and targeted race checks pass.

## Plan

- [x] Measure the original path with a gated stateful Release endpoint: withhold promotion cleanup, observe the shared mutex and independent Send/Status completion, then open the gate and retain timings. This is the first TDD step; no production fix precedes the measurement.
- [x] Drop b.mu only around promotion Release IO. Keep the existing actor reservation for the cleanup lifetime; reacquire the mutex and clear only this attempt's reservation. Preserve exact endpoint/ID, bounded RPC context, already-committed recovery, and PromoteRetry pacing.
- [x] Regress independent recipient progress, same-receiver reservation ownership, disconnect/replacement during cleanup and stale completion. Sweep all broker Release calls for the same shared-lock class; run messaging race and existing promotion/duplicate/reconnect controls.
- [ ] Record required full suite, obtain close review, follow the landing line and report the PR.

The existing actor reservation is the single resource excluding conflicting same-receiver work while cleanup runs (ARCH-ORDER). Keeping cleanup under the shared mutex couples unrelated recipients; detaching a goroutine adds lifetime ownership without helping this bounded worker. Reuse the existing endpoint/fake seam and pure model transitions (ARCH-DRY / ARCH-PURE / ARCH-MOCK). No protocol, status, queue, timeout, durable store or task owner is added (ARCH-CONSTRAINTS / ARCH-FUNERAL). Sweep the lock/Release class rather than only the named site (ARCH-PURPOSE).

## Log

- Filed separately at the TL's direction during #629 planning; unclaimed for later dispatch.

- Started from fresh main after #629 landed. start-plan refreshed the ariadne tool layer and refused until rerun with the refreshed binary; the rerun created the #631 branch successfully. Static sweep finds four broker Release call sites; promotion failure cleanup is the one inside b.mu. First collect gated contention evidence, then confirm the measured plan with the hub before production edits.

- Measurement on unchanged production code: TestPromotionReleaseDoesNotBlockOtherReceiver, -timeout=6s, failed as intended. A stateful fake withheld the promotion Release until the test deadline budget opened its gate after 2.9996s; 0/2 unrelated operations completed while it was held. Status took 2.999679s and Send 2.999699s, both completing only after Release was allowed to return (/tmp/pair631-release-contention-red.log). This is injected contention evidence, not a claim that a real bounded RPC takes 3s or that it caused the old #629 outage. Confirms the shared mutex couples independent receivers. Hub approved the plan and operator-restart qualification, no pair:0 staging (delivery 8a90dbfa-ac48-4138-97ce-190ffba778cc).

- Fixed contention: the same gated test passes under -race while Release remains withheld. Status completed in 31.459 microseconds and Send in 159.458 microseconds, both before Release returned (/tmp/pair631-release-contention-green.log). The mutex is dropped only for the bounded RPC, and the actor reservation remains owned until cleanup completes.
- Ordering regressions pass under -race (/tmp/pair631-release-ordering-green.log, 1.208s): sender cancellation cannot expose the receiver to another reservation before cleanup; replacement admission can hold a new reservation while the old worker retires, and old queued work cannot move to the replacement. Restoring early reservation clearing alone fails the ownership test (empty reservation, 2 wrapper reservation attempts instead of 1; /tmp/pair631-early-reservation-mutant-red.log). Temporary mutation restored.
- Shared-lock Release sweep: Send cleanup, runActor shutdown cleanup and delivery completion already perform Release outside b.mu. Only promotion failure cleanup required a change. The already-committed path performs no Release and keeps its existing recovery/wake behavior. Targeted godoc guard passed (0.453s); full messaging race package is running.

## Revisions

### 2026-10-17 — approved operator qualification

The hub approved measurement-first implementation and explicitly moved the live check alongside #629 to the operator-run restart (delivery 8a90dbfa-ac48-4138-97ce-190ffba778cc). No pair:0 staging. After restarting with the landed code, run #629's busy-hub A/B check and send a separate message to an idle connected receiver while the hub remains busy. That recipient's Send and Status should answer normally, with both FIFO hub messages preserved. The deterministic fake supplies the deliberately stalled Release ordering; do not freeze a live operator wrapper to reproduce it. Preserve unexpected receipts/timings and fix forward.

- Full couchmessage race package passed after the fix and all ordering tests: 23.799s, /tmp/pair631-couchmessage-race.log. The package run includes existing promotion pacing, already-committed/duplicate-ID recovery, queued cancellation and reconnect controls. Proceeding to the required recorded repository suite on the pushed head.

- Required suite reserved on pushed head e337cb6f95dfa1b709cd8503c7298cabd6637de1 as run a3bdcffb501a6de0fbfd71c49c815b01, initially position 7 in the shared machine queue. Active log .git/sdlc/test-stats/inflight/a3bdcffb501a6de0fbfd71c49c815b01/run.log. No executable edits remain planned; retain this reservation through the wait, then close/review/land under the normal gates.
