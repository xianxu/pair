---
id: 000847
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'fd1b41a413aaaec0120b8065838a1c1e81e63adf' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-23
