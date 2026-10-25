---
id: 000847
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-25
estimate_hours: 3.48
collaboration-mode: collaboration
card_mirror: '6ad9c85a18d9c26b3d7ab342d27faf60a01c9dd3' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-25T03:06:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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

- After the operator's one initial Couch start with the bench root as an extra hub (its `.hub` written by the bench tool), the bench hub brings up its claimed worker slot with an exact agent and argv, verifies it from the receipt's provenance, restarts it with new argv, and retires it with proof nothing it started survives: no Couch restart and no console steps (native test with fixture agents, claude and codex).
- Production slots (main-hub owned, unclaimed) and calls from the main hub or a worker are refused by these verbs (tests).

## Plan

Durable plan: [workshop/plans/000847-hub-scoped-cli-to-provision-and-retire-bench-slots-new-hub-root-agent-and-argv-plan.md](../plans/000847-hub-scoped-cli-to-provision-and-retire-bench-slots-new-hub-root-agent-and-argv-plan.md). TL approved the design 2026-10-25: distinct verbs sharing Reboot's implementation, retire keeps the worktree but proves no survivors, additive receipts, unchanged `.hub` files; the one remaining hand step is the initial Couch start with the bench root.

- [x] Task 1: Provisions predicate
- [x] Task 2: explicit launch profile through Reboot's Fresh path
- [x] Task 3: strict retirement
- [x] Task 4: operations, dispatch, guard, receipts
- [x] Task 5: CLI
- [x] Task 6: native acceptance (claude and codex)
- [x] Task 7: docs and close

## Estimate

Derived with estimate-logic-v3.1 against the v2 primitive table: the durable plan pre-resolves the design (spec-quality ×0.2, thorough-plan buffer 0.15); `impl=` at 40% of the table; familiarity ×1.5 for the broker guard and Fresh-recovery paths.

```estimate
model: estimate-logic-v3.1
familiarity: 1.5
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: greenfield-go-module design=0.4 impl=0.32
item: smaller-go-module design=0.06 impl=0.2
item: greenfield-go-module design=0.1 impl=0.32
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.0 impl=0.2
design-buffer: 0.15
total: 3.48
```

Items in plan order: Provisions predicate; explicit profile through Reboot's Fresh path; strict retirement; operations, guard and receipts; CLI; native acceptance with testcouch `.hub` claims; atlas; close review.

## Log

### 2026-10-23

### 2026-10-25
Revisions: Done-when reworded per TL (hub-start dropped in the 10-23 scope revision; the one hand step is named).
Implementation (commits 76ec41424, 54d669ef5, 6d89e0a79, 9c225aad3, 52fe353cb, adf28c138). Each new check was mutation-tested (reverting the code turns its test red): Provisions gate, vacant and recordful explicit-profile wiring, registry survivors and unknown-liveness rule. Native TestDisposableCouchBenchSlots PASS for claude (33.4 s) and codex (33.2 s). Found on the native runs and folded in (plan Revisions): bench-start parks a live agent before its Fresh reboot (Reboot otherwise asks a stop-set confirmation and the provenance would be lost); the harness lacked lsof for teardown. Caught in implementation: declaring bench ops PresentationInternal would have made `couch --internal bench-start` an ungated path; they are actor ops, and a CLI test pins the refusal.
