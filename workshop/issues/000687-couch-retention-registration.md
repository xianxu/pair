---
id: 000687
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'd85cf6d4120f96ddda67c84273ff068c26e8d80e' # card fields mirrored from issue-cards; edit via sdlc
---

# Recognize Couch supervisor retention registration

## Problem

During #677's real diagnostic preview, the running Couch supervisor PID45023 was classified as an ownerless legacy runtime and blocked253 current diagnostic rows containing23,462,164,824bytes. Those live current generations were correctly retained; no process intervention was performed. Closed unheld old generations now have separate reader-only proof.

Couch already calls storagegc.Coordinator.RegisterRuntime in couchcore/couch.go, but diagnosticlog.OSInspection/VerifyWriters recognizes PAIR_RETENTION_PROTOCOL=1 or the particular log's writer registrations. The supervisor has no per-thread PAIR_DATA_DIR/tag and its runtime lease is not consumed by that proof, so being registered in the coordinator does not establish protocol awareness there. Missing protocol means unrecognized registration, not an old start time.

## Spec

Make the supervisor's retention protocol participation recognizable through the existing lifecycle/identity-fenced registration boundary, as its clients' participation is. Do not broadly exempt processes by executable name or weaken unknown/open-holder protection. Decide whether to consume the exact registered runtime lease or publish a protocol capability through the existing launch boundary. Preserve birth identity and cancellation fences.

## Done when

- A stateful regression in the real supervisor shape (no per-thread tag or PAIR_DATA_DIR, valid runtime registration) is recognized as protocol-aware without a global current-log block.
- A missing/replaced/unreadable registration stays blocked; a supervisor actually holding a diagnostic payload still protects it.
- The next operator-coordinated Couch restart verifies the supervisor is registered; no process is stopped or relaunched as part of filing this follow-up.

## Plan

- [ ]

## Log

Filed unclaimed at the hub's request in receipt42a427eb-c28b-4124-9ef2-8c09ed6b56cd. #677 PR442 retains the current-log block and does not fold this supervisor lifecycle change into its retention policy implementation.
