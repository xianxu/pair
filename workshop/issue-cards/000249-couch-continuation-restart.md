---
id: '000249'
status: done
started: 2026-09-14T10:30:49-07:00
created: 2026-09-13
updated: 2026-09-14
estimate_hours: 6.216
actual_hours: 4.90
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
