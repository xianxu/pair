---
id: 000792
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '7c0f09869eb75a96b9a1ac6bbc05104ee46751fe' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T00:59:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Couch: no verb reaps proven orphans of a live slot's previous incarnation; extend --reap-orphans

## Problem

The resource monitor has positive prior-incarnation provenance, but the confirmed orphan command does not consume that group. Reusing its expanding tree reaper for an old root could also signal a current live descendant. The fix must preserve exact process scope and put the current authority/ownership veto beside each signal.

## Spec

10-21 ~20:15: the operator restarted echo:2 (hub ops-echo). `couch --resources` then listed **proven orphans** of its
previous incarnation: zellij 66561, pair 66575 and 66576 (`pair wrap`, `pair term`), nvim 66577 and 66583, and Claude
66695 (~350 MB RSS). No verb ends them:
- `couch --reap echo:2 --confirm` refuses `not-offered: echo:2 is live` (it would target the live slot);
- `couch --reap-orphans --confirm` covers only unattributed embedded nvim;
- from a different hub, `--reap` refuses with "slot belongs to hub …" (correct).

Fix: `--reap-orphans --confirm` (run by the owning hub, or the machine steward) also ends **proven orphans of a
previous incarnation** of a live slot, the set `--resources` already proves, each signal fenced on the process
identity the sweep recorded, never touching the live incarnation's processes. Report what was ended.

### Proposed implementation contract

Keep the existing argv and preview/confirmation distinction. Reuse the monitor's `SelectResourceProcesses` proof; freeze the admitted exact process keys and never expand their descendants. A CLI exact-set coordinator uses the existing TERM/3 s/KILL/2 s cadence. Each short supervisor-side effect request rechecks the frozen target against current process/inventory proof, authenticates an owning hub when caller credentials are supplied, and rechecks a nonempty start identity before signalling one PID. Unknown, stale or current ownership refuses; lost replies remain uncertain and do not authorize automatic resend/escalation.

Hub pin 941a9381-a841-44ca-98b0-7e10a90936b3 approves authenticated owning hubs only (workers and other hubs refused), with entirely unbound local CLI requests using the existing owner-only socket as the operator boundary. There is no new machine-steward role. The [durable plan](../plans/000792-proven-orphan-reap-plan.md) describes the protocol, state outcomes, counted scan cost, overlap handling, tests and live acceptance; the complete design awaits approval before change-code.

## Done when

- With a live slot plus a proven orphan set from its prior incarnation (fake process table), `--reap-orphans --confirm`
  ends exactly the orphans and none of the live processes (test).
- A pid reused since the sweep is not signalled (identity fence test).
- A current descendant beneath an old root and newly discovered/unproven children are not added to the confirmed exact set.
- Caller replacement, hub reassignment, live ownership transfer and unknown evidence before either TERM or KILL veto that effect; canceled/lost-reply/failed signals retain explicit partial outcomes.
- Preview and confirmed text/JSON report each admitted process once, including legacy overlap, and preserve the existing families. Public disposable-Couch conformance keeps the current Claude/Codex fixture slot alive and responsive.

## Plan

- [ ] Approve the durable plan and enter change-code through the full-flow gate.
- [ ] Add pure exact-target proof and strict protocol shapes; verify the negative authority/identity cases first.
- [ ] Implement the guarded exact executor and supervisor effect owner, with stateful transition and counted-cost tests.
- [ ] Wire preview/confirmed CLI reporting, prove disposable public-command behavior, document, verify and land through SDLC.

## Log

### 2026-10-21

### 2026-10-22 — claim and plan

Dispatched by ops:0 after #777 and the #768 manual archive repair landed. PR #540 is merged; both retained repair runners exited successfully and the workspace returned cleanly to its resting branch. Claimed #792 before design. start-plan's automatic dependency refresh generated an unrelated merge-check workflow change in an initially clean tree; restored only that generated diff and reran start-plan successfully, keeping protected gate configuration outside this issue.

Code inspection: ResourceView.Orphans already contains exact positive provenance from SelectResourceProcesses. The existing CLI consumes only OrphanNvim and LeftRunning and performs tree-expanded signals locally. The protocol has confirmation but no caller credentials. A bounded read-only helper audit confirmed TreeReaper expands every descendant, while teardown stopExact is unexported and lacks an effect guard; neither is safe to reuse unchanged for this live-slot case. Thread/lifetime leases cannot freeze an old process set while keeping the current incarnation live.

Asked the hub to pin the authority boundary in receipt eda16b29-ce04-4710-a5b2-4df4e17c03c8. Its submitted reply 941a9381-a841-44ca-98b0-7e10a90936b3 confirms owning authenticated hubs, no worker/other-hub permission, and the owner-only unbound operator CLI. The proposed full-flow plan keeps waits in the CLI and sends each guarded signal in the supervisor, with no new durable operation state. ARCH-DRY and ARCH-PURE preserve the existing provenance source and cadence; ARCH-SECURE and ARCH-STATE require current per-effect vetoes and explicit uncertainty. No implementation code changed and no estimate derived.

## Revisions

- 2026-10-22 — Design inspection found descendant expansion and a missing caller authority boundary. Preserve the original incident and two acceptance requirements; add the approved hub/operator contract and adversarial acceptance cases, and link the full implementation plan. No relaxation of identity or live-incarnation protection.
