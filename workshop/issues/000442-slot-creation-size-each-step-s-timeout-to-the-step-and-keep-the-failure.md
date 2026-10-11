---
id: 000442
status: open
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '060051f6eec7e990987f865162817293509e49d7' # card fields mirrored from issue-cards; edit via sdlc
---

# Slot creation: size each step's timeout to the step, and keep the failure

## Problem

Adding parley.nvim:3 from Couch failed on 10-11 around 09:56 with a timeout (operator report). It left an empty `worktree/parley.nvim-slot3/` and no record of the error anywhere under `~/.local/share/pair/couch`.

Likely cause, from the code: slot creation runs `git worktree add` through `WorkspaceProvisioner.git`, which uses `ProbeTimeout` (5 s, `cmd/internal/couchcore/provision.go:22,41`; `slotconverge.go:176`). A probe budget is meant for read-only checks, but `worktree add` writes a full checkout. parley.nvim (1687 files) took 2.2 s with the machine quiet, so with the fleet running suites it can pass 5 s. The `sdlc workspace --json` identity probe also uses 5 s, and on a new slot `sdlc` may build itself first. Unconfirmed until the error text is captured; the operator's screen is the only record.

## Spec

- **Budgets by step class:** read-only probes keep a short budget; steps that write a checkout (`worktree add`, branch creation) get a write budget (e.g. 2 min, like `FetchTimeout`); `weave compile` keeps `SetupTimeout`. The first `sdlc workspace` probe in a new slot gets a budget that covers sdlc building itself, or the probe runs after compile.
- **A failure is kept:** a failed creation records the step, the command, its timeout and stderr in Couch's state (and the UI shows the step name), so a report like "some timeout" can be diagnosed afterward.
- **Retry is clean:** a failed creation leaves nothing that blocks re-running it (the empty slot directory is reused or removed).

## Done when

- A `worktree add` slower than 5 s completes (test with a fake slow command).
- A failed creation leaves a readable record naming the step, timeout and stderr (test).
- Re-running creation after a failure succeeds without manual cleanup (test).
- parley.nvim:3 is created live.

## Plan

- [ ] Confirm the failing step (the operator's error text, or reproduce under load); classify each creation step's budget.
- [ ] Write budget for checkout-writing steps; identity probe after compile or with a build-sized budget.
- [ ] Persist creation failures; show the step in the UI.
- [ ] Tests for the Done-when rows; create parley.nvim:3 live.

## Log

### 2026-10-11
