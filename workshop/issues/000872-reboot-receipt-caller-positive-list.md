---
id: 000872
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: 'a0ae1aa1aa7fdc9bfae2779b246362d6a1612101' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T16:04:42-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Admitted worker reboot loses steward positive-list authentication during receipt polling

## Problem

In the approved #865 disposable Claude scale collection, N10 failed while starting worker5. The CLI first safely retried a catalog-warming refusal, then admitted reboot request c7f11c5f-10b2-472f-822d-ced172133631. Subsequent operation-status polling failed with unavailable: caller steward thread9223742121301770243-fixture-1 was not positively listed. The CLI correctly reported outcome uncertain, and no reboot was resent. This is a captured symptom; actual session disappearance versus discovery/projection error is unresolved. Do not assert a confirmed scale threshold or merge this with #866's earlier startup-probe panic.

## Spec

Classify and correct post-admission loss of positive-list caller authentication without weakening caller ownership, resending an uncertain mutation, increasing timeouts, or mutating the live fleet. Follow the accepted reboot to pollSlotOperation in cmd/internal/couchcmd/messages.go and the shared validated listing through messageThreadResolver in message_service.go and ListedPairSessionContext in cmd/internal/couchcore/artifactcollision.go.

The retained capture is /Volumes/TestScratch/tmp/pair865-scale.HgmR9S/evidence/collection.log; partial manifest claude-N10/manifest.json. Source1eb2135667e8b608cd369529ffdff19353f8d486, compiled vcs.modified=false; exact executable retained with SHA256a44dd8dfa3d6161fe3d7981ac12c649372496bc71f474a96346b322eddd001ff. The N10 subtest failed142.50s, before detach/workloads/profile. All owned generations were proved absent and direct child/IO joined; native root was removed by owner. Four worker pane captures and bounded server log tails exist; the failing caller steward pane, raw listing, receipt and binding census were not retained. Absence of #866's lib.rs1462 panic in truncated logs does not exclude every earlier server event.

## Done when

- Bounded evidence identifies whether exact steward session, discovery listing or binding projection changed between accepted reboot and receipt status.
- A deterministic accepted-to-status-auth-failure regression exercises the existing stateful seam and preserves conservative uncertain outcome, ownership and no blind resend.
- Correct the measured root cause and qualify via the shared disposable harness with exact cleanup; no live-fleet manipulation.

## Plan

- Preserve the existing capture, add bounded admission/status/listing-generation/steward-binding evidence at the refusal seam, and design only after classifying the loss.
- Follow normal issue gates; this filing grants no implementation or native rerun approval to another slot.

## Log

Filed unclaimed by pair:9 per TL ops:0 direction. Largest completed original cohort N5 (its waveform was separately invalid because startup system submissions stopped redraw before capture). #865 retains workload repair and measurement; #871 separately lands the opt-in profiler. No retained native owner remains from this failure.

### 2026-10-24 — claim and initial causal trace

Dispatched by verified ops:0 receipt 2fab61cc-ee8e-4641-abad-09e875dc45b2 after #851 landed, while #874's approved planning-only contract waits for #737. Claimed and start-plan completed on this separate main-based branch. The first start-plan correctly refused while on the parked #874 branch; its committed/pushed clean checkpoint was preserved, switched to main-slot2, then reran successfully. No stacked commits or runtime changes.

Read retained collection.log and traced CLI admission -> pollSlotOperation -> handleSlotOperation/liveCaller -> messageAuthority.current -> messageThreadResolver -> ListedPairSessionContext. The exact captured error is the resolver's absent/non-present projection branch, not a stale-poller error, timeout, duplicate attribution error, wrapper PID mismatch or nonce mismatch. That narrows which evidence failed but does not prove why the session was absent. The capture lacks the raw listing/caller pane/binding census needed to distinguish actual disappearance from an erroneous listing. Preserve conservative uncertain outcome and no resend.

Source investigation is read-only, including a bounded delegated listing/projection analysis. No native rerun, code, new timeout or authentication relaxation. Next: finish tracing listing parsing and namespace effects, inspect retained manifest and exact original source, then send the hub a causal classification and bounded diagnostic/correction plan before change-code.
