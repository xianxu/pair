---
id: 000249
status: working
deps: []
github_issue:
created: 2026-09-13
updated: 2026-09-14
estimate_hours: 6.216
started: 2026-09-14T10:30:49-07:00
---

# Fix continuation restart for Couch-hosted Pair threads

## Problem

Couch-hosted Pair can disappear when an agent saves a continuation: the writer
successfully commits the checkpoint and kills the current Zellij session, but
the replacement fails to launch. Couch retains a stale live incarnation.

Observed 2026-09-13 on Pair thread
`e108517d46ab4575/couch-36b623f6869ebaa2` (`📁pair-couch-27`):

- Native Codex binding `01a09ce1-0dae-7251-ae65-6964a7a2a92b` was established
  around 15:26; the session had substantial conversation. Missing transcript
  establishment does not explain this incident.
- 21:01:31 America/Los_Angeles: implementation checkpoint commit `4d18da7f`.
- 21:02:34: `pair continuation --slug pair-storage-retention ...` ran in the
  #239 worktree and committed `b96e1dc3`. The checkpoint is
  `workshop/continuation/20260913T210234-pair-storage-retention.md` there.
- 21:02:36–37: scrollback was preserved and the session ended. Later observation
  found no replacement Zellij session and Couch still recording helper PID 5330
  as live, displayed as `stale — couch exited unexpectedly`. The Couch
  supervisor and the other attached threads remained running.

### Reproduced registration mismatch

`continuationcmd` invokes `pair continue <slug>` after writing the checkpoint.
`launcher.runCompaction` preserves scrollback, writes a restart marker, then
kills the session. The outer `RunLaunch` loop consumes the marker and calls
`planRestart`, replacing `opts.Args` with arguments that have neither
ResumeRequired nor FreshRequired. Couch scope/tag remain in the environment.

Consequently `runCreate` calls `EnsureThreadAddress(..., couchOwned=true)` rather
than `RegisterExistingCouchThread`. That path accepts only a reserved marker;
the thread's existing marker is already established, so it returns
`Pair thread address already claimed` before launching the replacement.

A temporary Go overlay diagnostic exercised the production claim functions and
restart planner using a temporary store: reserve → establish → plan continuation
restart → attempt Couch claim. It confirmed the rejection and that read-only
existing-thread registration accepts the same marker. Command result:
`TestDiagnosticHostedRestartClaim` passed, launcher package 0.362s. This is a
component reproduction, not a full hosted restart reproduction; the original
outer-launcher error output was not recovered. No production code was changed.

The existing `TestRunLaunchContinueReentry` exercises a standalone fake whose
claim methods return canned errors; it does not enforce the Couch marker
lifecycle and therefore misses this interaction.

### Checkpoint lookup crosses worktrees incorrectly

The restart marker carries only a continuation slug. The outer launcher resolves
it under its own git root via `continuationDirPath`. Here it was started from
the main Pair checkout, but the writer saved the checkpoint in the #239 worktree.
The main checkout lacks that file. The current lookup silently leaves
ContinueDoc empty when resolution fails, so repairing registration alone can
still launch a fresh conversation without its handoff.

## Spec

Make continuation restart an explicit supported operation for Couch-hosted
threads. Preserve the same Pair address and correctly register the existing
hosted thread when launching the intended fresh agent conversation. Do not
weaken initial reservation checks or turn missing ownership evidence into
permission to create a session (ARCH-DRY, ARCH-SECURE).

Carry a durable, validated checkpoint reference across the writer/outer-launcher
boundary so a checkpoint authored in another worktree is resolved exactly.
Specify file lifetime and availability, and validate the handoff before stopping
the source where possible. Missing, unreadable, or mismatched checkpoints must
produce an actionable failure, never an unseeded fresh conversation presented
as successful continuation.

Define Couch's lifecycle across the restart, including helper survival or
replacement, registration, child exit, failure, and retry (ARCH-ORDER). After a
failure Couch must retain the checkpoint reference and truthful recoverable
state; it must not strand the thread behind a stale live incarnation or claim
the supervisor crashed when only one actor's restart failed.

The checkpoint remains the recovery source; native transcript binding of the
old conversation is not a substitute for the requested fresh continuation.
Audit adjacent restart-marker consumers for the same ownership/lookup class,
while keeping the work scoped to the continuation/restart contract.

## Done when

- An actual Couch-hosted continuation write restarts into the same managed
  thread with a fresh agent conversation seeded from the exact checkpoint.
- The flow works after warm reattachment and when the checkpoint is written
  in a different worktree from the outer launcher's working directory.
- Stateful integration coverage crosses writer → restart marker → teardown →
  real registration policy → replacement launch → Couch attachment, using
  portable fake processes/stores; no tests mutate real user sessions.
- Tests cover missing/unreadable checkpoints, established versus reserved
  claims, replacement failure, interruption and retry, without duplicate owners
  or silent unseeded launches. Existing standalone restart behavior stays covered.
- Failure leaves an actionable Couch state and durable checkpoint; docs explain
  supported continuation behavior and recovery.

## Plan

- [ ] Reproduce hosted restart with stateful ownership and cross-worktree checkpoint fixtures.
- [ ] Design restart ownership, checkpoint transport, and failure recovery in a durable plan.
- [ ] Implement, verify the full hosted flow and standalone regressions, and update docs.
- [ ] Close through the SDLC review gate.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against
`baseline-v3.1.md`. Method A only. Derived after plan-quality accepted round 3;
the calibration source is marked stale, so this remains provisional.

The primitives, in block order, cover issue/design authoring, checkpoint/request
model, persisted slot, writer/standalone transport, owner execution, interrupted
attempt reconciliation, Console state/polling, CLI bootstrap, cross-package
acceptance, live conformance, docs, and close review. Existing park, tracked
fresh start, orientation and queue implementations satisfy the library check;
there is no new service or third-party library to build. Implementation picks
use the upper table allowance where lifecycle coverage spans packages, scaled
to 40% by v3.1. Design uses the plan's 0.2 spec discount for implementation
primitives (not the design authoring itself); familiarity is 1.0 and the
thorough-plan design buffer is 15%.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.75 impl=0.12
item: greenfield-go-module design=0.20 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: greenfield-go-module design=0.40 impl=0.32
item: greenfield-go-module design=0.40 impl=0.32
item: tui-screen design=0.20 impl=0.40
item: smaller-go-module design=0.06 impl=0.20
item: api-integration design=0.20 impl=0.60
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 6.216
```

## Log

### 2026-09-13

Filed at operator request following investigation of the apparent Pair crash.
Native transcript and parked TTY capture place the exit immediately after the
continuation writer call. The capture is
`~/.local/share/pair/repos/e108517d46ab4575/parked-scrollback-couch-36b623f6869ebaa2-20260913T210236.raw`
with sibling events JSONL. Source inspected in launcher compaction, restart
planning, createflow, thread claims, continuation lookup, and Couch launch
registration. This differs from #248's detached-session inventory rejection:
this incident deliberately terminated the source but failed to replace it.
No implementation started; runtime identities above are historical evidence.

## Revisions

### 2026-09-14T10:40:00-07:00 — Shared recovery contract and execution order

Operator approved design and implementation of #248 → #249 → #250, pausing for
smoke after #248. Coordinated contract is recorded in
`workshop/plans/000248-couch-warm-reattach-gate-plan.md`; #249 receives its own
executable plan after that checkpoint. This issue owns durable continuation
request, checkpoint availability, replacement and retry, reused by #250.

Additional source findings: createflow seeds only the checkpoint basename under
the launcher's relative `workshop/continuation/`, so exact lookup must extend
through final prompt delivery. FreshRequired currently clears continuation
fields; setting that flag alone cannot fix registration. Existing tracked
fresh-existing launch and cleanup should be reused, with ownership independent
of conversation mode. Restart marker consumption currently precedes success;
the replacement must retain recoverable intent across failure.

### 2026-09-14T11:25:00-07:00 — Start implementation design after #248 shipment

Operator accepted #248 for shipment and requested the next issue. #248 merged
through PR #134; its coordinated plan is now archived under
`workshop/history/plans/000248-couch-warm-reattach-gate-plan.md`.

Source tracing confirmed that warm attachment retains the outer launcher loop,
but an in-place restart would leave Couch's helper ownership shape unchanged
and bypass its registration transaction. Design #249 around Couch-owned
replacement using existing verified park and fresh-existing start machinery
(ARCH-DRY, ARCH-ORDER). The writer transports an exact checkpoint snapshot;
the Couch owner picks up a durable request even while the panel is closed.
Failed requests retain that snapshot and require explicit retry. Account for
last-actor exit before replacement attachment so Couch itself stays open.
