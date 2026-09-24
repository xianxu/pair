---
id: 000318
status: done
deps: [ariadne#248]
github_issue:
created: 2026-09-23
updated: 2026-09-24
estimate_hours:
started: 2026-09-23T23:42:37-07:00
flow: {kind: quick, provenance: inferred, spec: "00b0737e", done: "607f2845"}
actual_hours: N/A
---

# Declare Pair post-move build

## Problem

The shared Ariadne procedure for moving an issue branch between slots asks
each repository to declare the build needed after the move. Pair's `:0` runs
from `bin/pair`, but its local agent instructions do not name that step. An
agent can switch branches and then test the old binary.

## Spec

In Pair's authored `AGENTS.local.md`, declare that after an issue branch moves
to `pair:0`, the agent runs `make build` there and checks the selected branch
HEAD remains the one built. Already-running sessions retain the old binary;
relaunch or start a fresh thread to test it. Point to the shared slot-move
procedure from ariadne#248. This is a Pair-only instruction; it does not change
the existing build target or generic bootstrap (ARCH-DRY).

## Done when

- [x] `AGENTS.local.md` names the post-move `make build` in `:0`, the HEAD check,
  and the fresh-session requirement.
- [x] The declaration is present in Pair's composed agent instructions after
  Weave compilation, with the operator's active `:0` work left untouched.

## Plan

- [x] Edit the Pair-local fragment in this isolated worktree.
- [x] Verify the authored fragment and composed output in a disposable fixture;
  review and publish through Pair's issue workflow.

## Log


- 2026-09-24: closed — Remote Pair tracker history joined without changing reviewed AGENTS.local declaration; weave-composed instruction and unchanged primary Pair checkout remain verified; isolated worktree lacks focused-hours telemetry.; review verdict: SHIP
### 2026-09-23
- 2026-09-23: closed — Isolated-worktree telemetry unavailable for focused-hours measurement; weave compile composed Pair AGENTS.md with make build, HEAD check and fresh-session instruction alongside ariadne#248 move guide; real Pair primary remained untouched.; review verdict: FIX-THEN-SHIP

Created as the Pair-owned portion of ariadne#248. The primary Pair checkout is
on an unrelated issue branch with operator scratch files; this worktree starts
from remote main and leaves that checkout unchanged.

The authored fragment was composed through the real `weave compile` pipeline in
this disposable worktree with a peer clone of ariadne#248. Generated `AGENTS.md`
contains both the exported phrase-to-procedure link and Pair's post-move build,
HEAD check and fresh-session instruction. `git status` in Pair's primary
checkout remained on its unrelated issue branch with its original scratch files.

The close review noted #317 and project commits in its historical window
`c3acba71..defbbac5`. They were already published on `origin/main` before this
branch's PR: `git diff --stat origin/main...HEAD` contains only this issue's
`AGENTS.local.md` and issue record. Those commits belong to their own work;
pair#318 does not claim them.

## Revisions

### 2026-09-23 — separate the reviewed Pair delta from inherited history

**Reason:** concurrent #317/project commits landed between the close review's
historical base and this issue branch, making the pinned window wider than the
actual PR delta.

**Delta:** retain the authored Pair declaration and document its two-file PR
delta against current `origin/main`; no unrelated files are changed or claimed
by pair#318.
