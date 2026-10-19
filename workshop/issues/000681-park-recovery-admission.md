---
id: 000681
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
severity: high
collaboration-mode: collaboration
card_mirror: '0be6f0c22706ae465bee9eda50e845d2f38b6279' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T18:19:31-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "d2f76ff4", done: "9beb35bc"}
---

# Recover park inventory exceeds CLI admission deadline

## Problem

On the operator build containing #651, confirmed CLI recover-park can time out before receiving a receipt because it builds the full fleet inventory inside a two-second admission exchange. The same inventory during ordinary resume preparation took 3.055 and 5.233 seconds. Two requests returned context deadline exceeded / outcome uncertain, but caller-scoped operation-status returned unknown for both; neither recovery ran. This strands an otherwise recoverable interrupted park behind its named CLI recovery command.

## Spec

Allocate and publish the operation receipt before building the inventory. Queue expensive exact-target preparation under the existing operation budget and report its progress/result through that receipt. Preserve exact address plus park nonce binding, caller authentication, confirmation, changed-target refusal and idempotent duplicate request handling. Do not silently act on a newer park transaction. Do not raise the global two-second message-admission timeout.

## Done when

- [ ] A stateful production-transport regression delays inventory beyond two seconds: CLI receives a receipt promptly, queued preparation completes and exactly one confirmed recovery runs.
- [ ] Duplicate IDs return the same receipt without a second effect; changed/replaced park identity refuses safely before mutation.
- [ ] Preparation error and cancellation remain inspectable outcomes, and no asynchronous effect survives a settled cancellation.
- [ ] Required suite and close review pass; fix lands through the normal gates.

## Plan

- [ ] Bind exact address/nonce from known cached console rows, then admit receipt before full inventory; refuse unavailable/ambiguous cache with refresh advice.
- [ ] Keep queued revalidation, dispatch authorization and lifecycle nonce checks; prove transport beyond2s, duplicates, replacement, cancellation and bounded growth.
- [ ] Required suite, close review, hub-coordinated conformance and normal landing.

Durable design: workshop/plans/000681-park-recovery-admission-plan.md. Pending hub approval; no code changed.

## Log

2026-10-18 — Read-only urgent investigation after operator UI relaunch left pair:0 and ariadne:0 with open awaiting_completion parks. Both helpers published matching success completions and preserved raw/events/capture bundles; original helper processes and terminal sessions are absent. No metadata repair, relaunch, resume or test was performed by the investigating worker. Original UI finalization cause remains a separate investigation.

Evidence: recover-park request IDs c8715942-f76f-4175-a2f7-fdbb85a54d86 and 56e33c49-e58f-4402-814f-99cd313dc886 each timed out at the client; hub caller verified both receipts unknown. Adjacent resume receipts measured preparation at 3.055s and 5.233s. Hub message 55a5b467-e030-4b23-b0c6-b81f15b47320 confirmed unknown receipts and requested this separate filing.

Code: couchcmd/slot_operations.go handle calls preparePark before admit; console_slot_operations.go preparePark calls PrepareSlotOperation; couchcore/slot_operation.go builds ActionableThreadInventoryContext; couchcmd/messages.go gives the entire admission exchange couchmessage.AdmissionTimeout (2s). The normal queued resume performs its preparation after receipt allocation. ARCH-ORDER: receipt admission precedes bounded queued preparation; retain exact transaction authority.

## Revisions

2026-10-18 — Planning after hub dispatch. Full-inventory-first violates the admission budget; moving the first binding to execution would weaken exact selection. Use an existing cached console snapshot for immutable selection before receipt, then queue expensive revalidation. Unknown/ambiguous cache refuses explicitly. Preserve original Done-when and exact-identity contract; #683 diagnostics stays separate.

2026-10-18 — Start-plan refreshed the environment dependency and asked for a rerun; rerun created canonical issue branch from fresh main. Design is awaiting hub approval; no implementation/test/build was initiated except the gate-owned dependency tool refresh.
