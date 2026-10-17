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

- [ ] Measure the original path with a gated stateful Release endpoint: withhold promotion cleanup, observe the shared mutex and independent Send/Status completion, then open the gate and retain timings. This is the first TDD step; no production fix precedes the measurement.
- [ ] Drop b.mu only around promotion Release IO. Keep the existing actor reservation for the cleanup lifetime; reacquire the mutex and clear only this attempt's reservation. Preserve exact endpoint/ID, bounded RPC context, already-committed recovery, and PromoteRetry pacing.
- [ ] Regress independent recipient progress, same-receiver reservation ownership, disconnect/replacement during cleanup and stale completion. Sweep all broker Release calls for the same shared-lock class; run messaging race and existing promotion/duplicate/reconnect controls.
- [ ] Record required full suite, obtain close review, follow the landing line and report the PR.

The existing actor reservation is the single resource excluding conflicting same-receiver work while cleanup runs (ARCH-ORDER). Keeping cleanup under the shared mutex couples unrelated recipients; detaching a goroutine adds lifetime ownership without helping this bounded worker. Reuse the existing endpoint/fake seam and pure model transitions (ARCH-DRY / ARCH-PURE / ARCH-MOCK). No protocol, status, queue, timeout, durable store or task owner is added (ARCH-CONSTRAINTS / ARCH-FUNERAL). Sweep the lock/Release class rather than only the named site (ARCH-PURPOSE).

## Log

- Filed separately at the TL's direction during #629 planning; unclaimed for later dispatch.

- Started from fresh main after #629 landed. start-plan refreshed the ariadne tool layer and refused until rerun with the refreshed binary; the rerun created the #631 branch successfully. Static sweep finds four broker Release call sites; promotion failure cleanup is the one inside b.mu. First collect gated contention evidence, then confirm the measured plan with the hub before production edits.
