---
id: 000847
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-25
estimate_hours:
collaboration-mode: collaboration
card_mirror: '23b26ef734fa02e75908bbd6008f87e2146ec155' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-25T03:06:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# Hub-scoped CLI to provision and retire bench slots (new hub root, agent and argv, fresh) without a Couch restart or console steps

## Problem

## Spec

**Let a hub provision bench slots from a run descriptor, so model-bench needs no hand setup (operator, 10-23).**
For model-bench (ops `teams/devinfra/projects/model-bench/project.md`), the couch-bench slot should prepare the
bench TL hub and worker slots with the exact configuration the run descriptor names (agent, model, effort, argv),
instead of the operator restarting Couch with an extra root and setting agents and parameters by hand in the
switcher. ariadne#577's first real run found the gaps, verified against Couch's docs and source: adding a hub root
needs a restart with a new positional path; a hub can't CLI-drive another self-owned hub (switch agent, set
parameters, reboot fresh); and the global Start form for opening an existing primary is console-only.

Add supported, hub-scoped CLI verbs (agent-first, per the existing direction):
- start a new disposable hub at a given root, without restarting Couch;
- open or switch a slot owned by that hub to a given agent with explicit argv (replacing saved parameters), fresh;
- retire those slots and the root afterwards;
all authorised only for slots the calling hub provisioned, with receipts the observer can verify (root, session,
generation, launch profile).


**Scope revision (operator, 10-23):** starting the bench hub isn't the constraint (the operator can start Couch with
it as an extra root). Drop the hub-start verb. The need is a hub managing *its own* slots by command: open or start
a slot it owns with an exact agent and argv, restart it fresh, switch its model, and retire it, with verifiable
receipts. Lessons from metis (`../metis`): borrow its provenance-record and append-only-ledger patterns in the bench
tool, not its run engine (metis has no agent management).

## Done when

- A hub can bring up a bench hub and worker with exact agent and argv, verify them, and retire them, with no Couch
  restart and no console steps (native test with fixture agents).
- Production slots are untouchable by these verbs (test).

## Plan

Draft for TL review (2026-10-25); not yet approved.

What exists (mapped): `--switch-agent repo:N --agent A --agent-args … --confirm` (hub-gated by authorizeHubMutation, works on a vacant slot, explicit argv replaces saved repo parameters); `--reboot repo:N --confirm` (fresh; a vacant slot fresh-starts; no agent/argv); `--set-model` (live drive op). Missing: a "slot this hub provisioned" notion (the main hub owns every unclaimed slot by default), an exact-profile fresh start/restart, receipt provenance, and any slot retirement.

Design:
1. Bench ownership predicate `HubOwnership.Provisions(caller, target)`: true only when the caller is a non-main hub and the target resolves to it through an explicit `.hub` claim (exact `repo:N` or whole `repo`). Default main-hub ownership never qualifies, so production slots are structurally out of reach. Existing `.hub` claim files and slot layout are unchanged (transitional compatibility).
2. Two new hub slot operations through the existing broker path (slotOperations, handleHubSlotOperation, admission check plus queued effect guard re-proving caller and Provisions):
   - `couch --bench-start repo:N --agent A --agent-args "…" --confirm`: fresh start of a vacant slot, or fresh restart of a live one, with exactly that profile (argv source explicit; replaces saved parameters, as switch-agent does). Switching model = another bench-start with new argv (fresh, new generation). Reuses Reboot with RebootTarget gaining the launch profile.
   - `couch --bench-retire repo:N --confirm`: stop the agent and archive its thread so the slot is vacant and the next bench-start is clean. Worktree kept (the bench runner owns checkouts).
   Existing verbs keep their current gates and behavior.
3. Receipt provenance: OperationReceipt gains optional Root, Session (pair session name), Generation (launch nonce) and LaunchProfile {agent, argv, argv source} on a succeeded bench-start; additive, omitempty.
4. Tests: unit for Provisions and the guard; testcouch gains `.hub` claim support; native (claude and codex): main hub fixture:0, bench hub fixture:1 claiming fixture:2; bench hub bench-starts fixture:2 with exact argv (receipt and fixture-observed argv match), bench-starts again with new argv (new generation), retires it; bench-start/retire on an unclaimed production slot and from the main hub refuse. Atlas + couch SKILL.

Full flow expected (code beyond the quick-flow shell); durable plan at workshop/plans/ after TL approval.

## Log

### 2026-10-23
