---
id: 000442
status: working
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '66d0f0eeb76f84fabefd19e6b0e40d2375b46c60' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T10:00:28-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "3fb43775", done: "4043411e"}
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

Durable plan: `workshop/plans/000442-slot-creation-size-each-step-s-timeout-to-the-step-and-keep-the-failure-plan.md`.

- [x] Confirm the failing step (forensics on the live repository, see Log); classify each creation step's budget.
- [x] Write budget (`WriteTimeout`, 2 min) for `worktree add` / `worktree repair`; identity probe unchanged (see Revisions).
- [x] `ProvisionCommandError` names command + budget + timed-out; the advice names the step; a blocking outcome is recorded at `<common>/couch-workspaces/N/last-failure.json`, cleared on success, shown by `couch --show`.
- [x] Tests for the Done-when rows (`provision_budget_test.go`).
- [x] Full `make test`; create parley.nvim:3 live.

## Revisions

### 2026-10-11 — design landed (pair:2)
- **Identity probe:** no change. Couch execs the PATH `sdlc` (`~/.local/bin/sdlc` → a prebuilt binary); the self-building `sdlc` is the agent shell's function, which `exec` never runs. And the new slot's identity probe (`verifyHost`) already runs after compile.
- **Branch creation:** its only heavy command (the fetch) already has `FetchTimeout`; `update-ref` / `config` stay probes.
- **Record location:** the slot's directory in the Git common dir (beside the creation lease), not `~/.local/share/pair/couch`. The provisioner owns slot records and that directory survives a failure that never made the environment. It holds one record per slot (bounded), written under the creation lease.
- **Retry:** already clean by design (pair#387). An environment without a host is `ErrSlotNeedsReconcile`, a number to reuse, and the next reconcile re-runs `worktree add`. Now proved by a test.

## Log

### 2026-10-11 — failing step confirmed (forensics, parley.nvim)
- `main-slot3` reflog: `couch-slot-create` at 09:56:29; `branch.main-slot3.{remote,merge}` set → create-branch and set-upstream completed.
- `worktree/parley.nvim-slot3/` empty (mtime 09:56); no `.git/worktrees` entry for slot 3 → `git worktree add` was killed. On SIGTERM (OSProvisionIO's group stop at the 5 s probe budget), git's junk handler removes the half-made checkout and its registration, which matches this state exactly.
- No other step between set-upstream and worktree-add runs a command that could take 5 s.

### 2026-10-11

### 2026-10-11 — verification
- New tests pass; mutation (worktree add back on the probe budget) fails `TestWorktreeAddUsesWriteBudget`.
- Full `go test ./...` unsandboxed: couchcore green after moving the record path off `SlotLayout` (`TestSlotLayoutMethodsBelongToOneResource` caught a method that isn't a resource location). Remaining failures are pre-existing or environmental: launcher `TestLaunchNative*`/`TestCheckpoint*` and sessionsync `TestTwoProcessWritersExactlyOneWins` fail on origin/main too; couchcmd `TestContinuationWriterPublishesExactCheckpointAcrossWorktrees` fails on its fixture's missing `origin` and the inherited `PAIR_DATA_DIR` scope.
- `make -k test`: only `test-review` (Lua review-window) fails, and it fails identically on origin/main.
- Live: `couch --show parley.nvim:3` (this branch's binary) showed env/branch/upstream present with the plan `worktree-add host on main-slot3`, matching the forensics. `couch --reconcile parley.nvim:3` → `parley.nvim:3 created` in 23 s (worktree add + weave compile), with no manual cleanup.

### 2026-10-11 — close review round 1 (FIX-THEN-SHIP, 2 Important + 4 Minor), all fixed
- BR-1: the retry in `TestFailedCreationIsRecordedAndRetryIsClean` ran real git under the 200 ms budget. Fix: restore the budget before the retry.
- BR-2: a cause with a 64 KiB stderr tail exceeded the 16 KiB record limit, so the record was silently dropped. Fix: `boundCause` keeps head + tail within 8 KiB (`TestSlotFailureRecordFitsTheStore`, mutation-checked).
- Minors: the record is now cleared only after `verifyHost` and the marker read; every advice branch names the step; an unreadable record shows as `last failure: unreadable: …`; render test `TestRenderSlotReportLastFailure`.

