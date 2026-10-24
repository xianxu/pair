---
id: 000844
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-24
estimate_hours: 3.24
collaboration-mode: collaboration
card_mirror: 'c446c605ef046f6d5aedf081f511de06c1e5b06b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T10:30:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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

- [x] Implement the pure typed routing table and bounded delivery reducer after TL approval.
- [x] Integrate Console startup/shutdown and existing guarded system-message delivery.
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


### 2026-10-24 — plan gate round 1

Full-flow plan review returned one Important finding PQ-1 (executable-test-strategy): name the production functions under test and compress prose scenario inventories. Revised the entire test contract and task section to name RouteNotice, AdvanceNoticeDelivery, resolveNoticeOwner, runNoticeDelivery, publishWarningLocked, applyNoticeRoute and the existing condition/notification functions, with one adversarial strategy each. State sequences call the production reducer and check independent no-duplicate, epoch-fencing and resource-bound invariants; stateful shell fixtures control effect completion ordering. No policy change, implementation or estimate yet. Existing Attention unit baseline passed (go test ./cmd/internal/couchcore -run TestAttention -count=1, 0.350s).



### 2026-10-24 — implementation checkpoint, integration verification running

change-code full flow passed: PQ-1 addressed, round 3 clean, estimate-quality INFO (producer/integration budgets optimistic). Implemented typed routing, bounded pure delivery state and four-worker shell, Console feed-once projection, startup/close binding, exact owner/epoch broker backend and the complete producer/condition/archive/stall sweep. Existing picker/question wait watcher retains canonical sends, thresholds, mute and deduplication; HubHandled denotes that effect ownership rather than proof of successful submission. This avoids introducing immediate duplicate picker traffic. Startup waits on surviving/explicitly attached wrappers, never treating a nil boot service as absence.

Test-first failures covered missing APIs, typed producer facts, UTF-8 clipping, admitted receipt tracking, full-capacity condition coalescing, shutdown cancellation, and OFF/ON before async dispatch. Publication now captures cached owner readiness/epoch under the documented memory-only lock order; broker IO runs only outside Console/publication locks. Whole-package AST guards cover Warn, locked warnings and direct attention additions outside audited projections.

Verified focused race aggregate: go test -race ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'TestNotice|TestConsoleNotice|TestCondition|TestPicker|TestWaitNotice|TestWaitAutomation|TestArchive|TestAutomaticInputStall' -count=1 -timeout=120s PASS (core 20.671s, tty 3.021s, cmd 13.406s). Adapter race x3 PASS 4.092s; startup real-broker integration race PASS 1.920s; unknown-class fuzz PASS 10,811 executions. Full tests for those three packages are currently running; no full-package or recorded check pass claimed yet.

Reader audit: wait_notice_test.go still reads the same Claude/Codex wrapper screen fixtures and retains its registered fixed-scanner guard; only fixture cleanup/expected routing changed. New notice_class_inventory_test.go files and tty notice_producer_inventory_test.go scan only their own Go package's tracked source or literal AST fixtures, so normal package ownership covers their inputs; each is explicitly local_or_literal_fixture. Individually recorded those four named certificates; make test-affected-inventory and git diff --check PASS. No selector or local-gates configuration changes.

NEXT ACTION: read the retained three-package test result, fix/classify failures, complete atlas/plan verification, push and run recorded --pre-close, inventory then close. The full recorded check has not started.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only. Calibration is flagged stale by estimate-source, so these are provisional ship-wall-clock hours. Read v2/v2.1 procedure and table: retain issue/spec design cost; use x0.2 design discount for approved module/integration/refactor decisions, 40% implementation conversion, familiarity 1.0 and 15% design buffer.

Reuse check: existing AttentionList/feed, cached authority, guarded sendAutomaticSystem, canonical receipts and stateful fixtures cover the stack; no novel external library/API discovery is needed. Policy is one small extension, delivery reducer one single-concern greenfield module, dispatcher an integration over the existing broker, producer migrations two bounded cross-cutting surfaces (TTY and service), and condition projection one existing TUI seam. No double-counted new screen or transport.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.04
item: smaller-go-module design=0.04 impl=0.16
item: greenfield-go-module design=0.2 impl=0.32
item: api-integration design=0.2 impl=0.4
item: cross-cutting-refactor design=0.1 impl=0.2
item: cross-cutting-refactor design=0.1 impl=0.2
item: tui-screen design=0.1 impl=0.2
item: atlas-docs design=0.05 impl=0.04
item: milestone-review design=0.0 impl=0.2
design-buffer: 0.15
total: 3.24
```

Design subtotal 1.29 x 1.15 + implementation subtotal 1.76 = 3.2435, rounded 3.24 h. One final boundary review is included; no extra milestone review. Implementation includes focused and recorded verification, exact-count effect tests and inventory audit.
