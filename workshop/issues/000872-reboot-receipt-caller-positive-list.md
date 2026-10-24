---
id: 000872
status: open
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
card_mirror: '39bb72c98c0769cf2a93969facb71293c3164613' # card fields mirrored from issue-cards; edit via sdlc
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
