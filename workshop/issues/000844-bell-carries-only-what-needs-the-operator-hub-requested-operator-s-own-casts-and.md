---
id: 000844
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-24
estimate_hours:
collaboration-mode: collaboration
card_mirror: '830c90aaaec9940181ff7be99db3735ffaac4728' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T10:30:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Bell carries only what needs the operator (hub-requested, operator's own casts and drives, no hub able to act); the rest goes to the hub

## Problem

## Spec

**The bell carries only what needs the operator; everything else goes to the hub (operator, 10-23).** pair#812
routed 34 producer classes to attention (list: ops `session.side/2026-10-23-11-bell-contents.md`). Many are things
the hub already receives and acts on. New default, **deny by default, add as we operate**: an event reaches the bell
only if one of these holds:
1. **A hub asked for it** (`couch --notify`). The hub decides what the operator must see.
2. **It's about something the operator is directly doing**: a cast or Chrome cast they started failing or ending
   unexpectedly, a cast code lockout, remote driving they're using failing, a cast or drive action they took failing.
3. **No hub can act on it**: Couch itself crashed or couldn't start its hub, or no live hub exists.

Everything else (supervision lines such as "N min with no later hub message", worker "waiting on a prompt" and
judged questions, lowered-priority relaunch, lifecycle cleanup and metadata risks, history-backup and continuation
problems, hub claim conflicts, trace and crash-capture setup failures) goes to the owning hub as a Couch system
message, and to the feed; the hub notifies the operator when a human is needed. Keep the routing table in one place
so adding a class back is a one-line change.

## Done when

- Only classes 1 to 3 add bell items; every other class reaches the owning hub and the feed (table-driven test over
  every producer class).
- With no live hub, class-3 events still reach the bell (test).

## Plan

Durable plan: [human attention routing](../plans/000844-human-attention-routing-plan.md).

- [ ] Implement the pure typed routing table and bounded delivery reducer after TL approval.
- [ ] Integrate Console startup/shutdown and existing guarded system-message delivery.
- [ ] Sweep every producer, direct prompt/question/archive/stall path and duplicate-effect case.
- [ ] Update atlas, audit reader certificates, run recorded pre-close, close and land.

## Log

### 2026-10-23


### 2026-10-24 — claim, audit and proposed plan

Started independently from main after #812 landed; #853 is blocked on ariadne#598 and contributes no unlanded commits to this branch. Two bounded read-only audits enumerated all warning producers and existing hub-routing seams. Findings: status/warning Kind is not a policy class; two sites directly call publishWarningLocked; prompt/question and archive/stall paths bypass Warn; wait questions already send a hub message; startup warnings precede service creation. The plan addresses the full enumerable class, preserving existing admission/automation guards (ARCH-DRY, ARCH-PURPOSE).

TL approved policy in delivery 84c76ec1-c90f-40f0-885d-bfeafba9b898: automation OFF gives feed only/no replay; connected disabled hub is not absent; missing connected owning hub gives actionable fallback while routine feedback stays feed-only. The detailed design proposes typed classes, one table, buffered startup and bounded asynchronous hub delivery with exact occurrence IDs; global diagnostics retain primary ownership. Capacity fallback and live-hub picker interpretation are explicit approval questions in the plan. No code or tests changed and no passing verification claimed.

**NEXT ACTION:** receive TL approval of the durable design, resolve the bounded dispatch/overflow and hub-picker details, then run change-code (plan-quality review before estimate/implementation). No code before that boundary.


### 2026-10-24 — detailed design approved

Ops:0 delivery 12d7cfb1-b715-4546-b159-69f6c8cde4bb authenticated as submitted from the bound hub: detailed plan approved, including one table, startup buffer/reducer, 128 x 4 KiB/four admissions/two-second contexts/two-minute expiry, no uncertain resend, primary global ownership, one aggregated capacity-loss bell, and connected-hub picker rule. Proceed to change-code. Initial invocation waited on #838's live close lock and was canceled before acquiring it to record the approval and select the full flow; no implementation/gate bypass occurred.

NEXT ACTION: run change-code --issue 844 --flow full; address plan-quality findings, derive the estimate only after that review passes, then execute the test-first tasks.
