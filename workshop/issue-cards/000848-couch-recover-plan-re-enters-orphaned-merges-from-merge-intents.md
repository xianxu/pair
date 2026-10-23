---
id: 000848
status: open
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
---

# couch recover plan re-enters orphaned merges from merge_intents

## Problem

After a full shutdown, merge runners that were in flight need re-entry by hand: nothing in the restart plan names them. ariadne#565 (PR 419, merge be2b8436) now records each in-flight `sdlc merge` as a per-branch intent file held by the runner; a killed or interrupted runner leaves an orphan. `sdlc state --json` carries the slot's own as `merge_intents[]`, each with `state` (live, orphaned, stale or unreadable), `branch`, `issue`, `head`, `root`, `workspace`, `pid`, `started_at`, `reentry` (orphaned only: the exact command, run in `root`) and `note` (set when the machine queue is paused). `sdlc landing status` prints the repository's orphans as lines.

`couch --recover-plan-from-sdlc` reads `sdlc fleet inventory --json` (cmd/internal/couchcore/recoverplan_fake.go models it), which does not carry merge intents yet.
