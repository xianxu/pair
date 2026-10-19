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

- [x] A stateful production-transport regression delays inventory beyond two seconds: CLI receives a receipt promptly, queued preparation completes and exactly one confirmed recovery runs.
- [x] Duplicate IDs return the same receipt without a second effect; changed/replaced park identity refuses safely before mutation.
- [x] Preparation error and cancellation remain inspectable outcomes, and no asynchronous effect survives a settled cancellation.
- [ ] Required suite and close review pass; fix lands through the normal gates.

## Plan

- [x] Bind exact address/nonce from known cached console rows, then admit receipt before full inventory; refuse unavailable/ambiguous cache with refresh advice.
- [x] Keep queued revalidation, dispatch authorization and lifecycle nonce checks; prove transport beyond2s, duplicates, replacement, cancellation and bounded growth.
- [ ] Required suite, close review, hub-coordinated conformance and normal landing.

Durable design: workshop/plans/000681-park-recovery-admission-plan.md. Pending hub approval; no code changed.

## Log

2026-10-18 — Read-only urgent investigation after operator UI relaunch left pair:0 and ariadne:0 with open awaiting_completion parks. Both helpers published matching success completions and preserved raw/events/capture bundles; original helper processes and terminal sessions are absent. No metadata repair, relaunch, resume or test was performed by the investigating worker. Original UI finalization cause remains a separate investigation.

Evidence: recover-park request IDs c8715942-f76f-4175-a2f7-fdbb85a54d86 and 56e33c49-e58f-4402-814f-99cd313dc886 each timed out at the client; hub caller verified both receipts unknown. Adjacent resume receipts measured preparation at 3.055s and 5.233s. Hub message 55a5b467-e030-4b23-b0c6-b81f15b47320 confirmed unknown receipts and requested this separate filing.

Code: couchcmd/slot_operations.go handle calls preparePark before admit; console_slot_operations.go preparePark calls PrepareSlotOperation; couchcore/slot_operation.go builds ActionableThreadInventoryContext; couchcmd/messages.go gives the entire admission exchange couchmessage.AdmissionTimeout (2s). The normal queued resume performs its preparation after receipt allocation. ARCH-ORDER: receipt admission precedes bounded queued preparation; retain exact transaction authority.

## Revisions

2026-10-18 — Planning after hub dispatch. Full-inventory-first violates the admission budget; moving the first binding to execution would weaken exact selection. Use an existing cached console snapshot for immutable selection before receipt, then queue expensive revalidation. Unknown/ambiguous cache refuses explicitly. Preserve original Done-when and exact-identity contract; #683 diagnostics stays separate.

2026-10-18 — Start-plan refreshed the environment dependency and asked for a rerun; rerun created canonical issue branch from fresh main. Design is awaiting hub approval; no implementation/test/build was initiated except the gate-owned dependency tool refresh.

2026-10-18 — Hub approved cached exact selection in delivery53e81e0b-ada3-4d7b-b830-7f30238b7ef2; change-code passed inferred quick flow. Added pure SelectCachedSlotRow (two bounded linear traversals for numbered selection, existing SelectSlotRow reused) and moved slow PrepareSlotOperation behind cached binding/receipt to the owned queue. Shared confirmation validation now precedes binding. Pure selector RED failed on missing function, GREEN targeted tests passed0.248s. Behavior RED caught unconfirmed requests binding once and unknown cache lacking refresh advice; GREEN TestRecoverPark passed0.396s, broader RecoverPark|SlotOperation passed0.996s. Real Unix transport >2s/adversarial proof remains in progress; no full-suite or live acceptance claimed. No staging or operator recovery run.

2026-10-18 — Actual Unix transport/production console queue proof passed with full preparation blocked beyond AdmissionTimeout: a receipt/status stays readable, duplicate request joins one job and one retirement effect. Old-binding Go overlay failed at the real two-second exchange; red log /tmp/pair-681-park-admission-overlay-red.log SHA256347364df68daca9f546f55b7ed2b21a507b3bfdcc4a1ff760184f4c08e54d398. Private hub wiring exercises the effect-time guard; a nil-hub test shortcut would bypass that guard. Replacement nonce before queued preparation and before dispatch refuses without changing the replacement; cancellation joins the worker, and a canceled observation retains its failed receipt and releases the pending key. At8/16 cached rows, both run2 workspace probes/2 discoveries/1 effect, with0 discoveries/effects while blocked. Focused package pass6.640s; focused race pass7.708s; core SelectCachedSlotRow|SelectSlotRow|RecoverPark pass7.642s. No full-suite/live acceptance claimed.

An ad-hoc cached-binding review found retained duplicates still depended on fresh alias lookup. Added a red regression that produced unknown-slot on a now-unavailable catalog; moved retained receipt/conflict lookup before IO in shared admission, rechecking under lock afterward. Reviewer confirmed resolution with no additional findings. This implements the existing duplicate-first plan requirement, not a scope change. Added the rule to workshop/lessons.md.

Hub-approved next items remain #683 part2 (durable lifecycle error messages, keep root cause open) AFTER #681 lands, then #684 (Fresh fallback must prove old actor/session gone before archive). Leftover Zellij session1-188 is operator-owned; no action taken against it. Live #681 fixture may use pair:8; request hub scheduling before staging :0.

2026-10-18 — Hub revised the follow-up queue in verified delivery c3ea26f5-c568-4ea7-9a49-893a8be886c4: after #681 lands, hand off a short #683/#684 evidence summary for a separately approved lifecycle project; do not implement either next. In verified delivery cedf3674-7d87-4875-b420-7475d250532f the hub approved landing on private production-service proof, required suite and SHIP, with no :0 staging. Pair:8 live acceptance remains OWED after the next operator Couch restart; the hub tracks it. Staging alone would not replace the running server. Own-slot make build passed; bin/couch reports clean vcs.revision 695142390005e031b523cac545d0eefb04a8982b and trimpath=true. Recorded suite attempt de830da7014ac37467ee7cf0626f8953 is queued on that pushed source, no result yet.
