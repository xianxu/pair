# couch — the session supervisor

`couch` is a second binary in this repo (`cmd/couch`) that supervises agent
sessions: it registers them, spawns them, and knows what is running where. It
is **not** an extension of `pair`. pair is what the operator sits inside, so a
supervisor bug must not break the ability to fix it; the fallback is always to
launch pair the old way.

Project: `workshop/projects/couch.md`. Built in `pair#145`.

## What exists today

`couch start|list|show|stop|name|describe`, over a registry persisted to
`~/.local/share/pair/couch/registry.json`.

**couch hosts `pair` whole.** The stack is couch → pair → zellij → claude+nvim.
couch spawns `pair --layout2` and hands it couch's own stdio, so `couch start`
blocks for the child's lifetime. Verified by operator smoke on 2026-08-21; the
alternative (couch absorbing zellij's role) was considered and rejected because
the agent child is never spawned by Go — zellij spawns it from a KDL layout, and
`entrypoint.ValidRootMarkers` *defines* a valid pair install as having those
layouts.

**There is no pty yet.** Attaching, switching and detaching are `pair#146`.

## Identity: the working tree

An actor is keyed on the **canonical absolute path of a worktree root** — not on
an issue, not on a subdirectory. `kbench/competition/arc-agi-3/` starts there
and registers under `/Users/xianxu/workspace/kbench`, because the collision
hazards (one index lock, one branch, one `git status`) are tree-scoped.

- `Worktree` holds the path in **original case**; `Key()` folds it for lookup.
  The split matters: pair feeds `launcher.ResolveRepoScope` a raw path, so a
  folded path would derive a different scope key for the same tree.
- **Name registration IS the collision guard.** Refusing a second agent on an
  occupied tree is `Register` failing because the key is taken, not a separate
  check. `--same-tree` overrides it and is recorded.
- `ActorID` (`couch-ah8d`) identifies an *incarnation*, not an address —
  Erlang's pid to `Worktree`'s registered name.
- Operator names and agent-supplied descriptions attach to the **tree**, so they
  survive an actor being replaced.

## Seams

Everything touching the world is injected, so the domain tests without
processes, disk, wall-clock or randomness: `Runner`/`Handle` (spawn), `PathOps`
(symlinks), `GitRunner`, `ProcOps` (out-of-process liveness), `Store`, `Clock`,
`IDGen`. Each has a fake; `FakeRunner` is stateful and is compared against real
processes by `conformance_live_test.go` (gated on `PAIR_LIVE_COUCH=1`).

`Runner` was genuinely new — pair has no async process-exec seam.
`launcher.ProcOps` is named for pair's own sidecars, and `wrapcmd` spawns its
child inline and unseamed.

## Liveness is recomputed, never stored

Because `couch start` blocks, every read command runs in a **second process**
with no `Handle`. So `ActorRecord` persists `{PID, Identity}` where `Identity`
is `procutil`'s kernel start token, and a reader recompares it: a recycled PID
reports not-live because the token differs.

Within one process, `ExecRunner` reaps its children in a background goroutine
and liveness is a closed channel — **not** `kill -0`, which succeeds for a
zombie and would report an exited-but-unreaped child as running.

## Actor loop

One goroutine per actor, holding a bounded mailbox. `Enqueue` is a pure function
(collapse by kind, drop the oldest non-control entry over capacity, never drop
control) so the policy is testable without goroutines. The loop drains
control-first.

It is a mutex-guarded queue rather than two channels with a priority select: the
bounded/collapse policy must apply at insertion, and a buffered channel cannot
collapse a duplicate already in it. Queries are direct calls behind the same
mutex — Go shares memory, so message passing here buys ordering and decoupling,
not fidelity to Erlang.

## Terminology

- **tree** — a worktree root; the unit of identity and of the collision guard.
- **actor** — one agent session on one tree; a goroutine plus a child process.
- **incarnation** — one run of an actor; a new `ActorID` after a revival.
- **parked thread** — a tree with a name but no running agent. Still listed
  (dimmed), because it is exactly the thread an operator loses track of.

## Planned, not built

`pair#146` tty switching · `pair#147` cluster transport and queries ·
`pair#148` brain as advisor. Cross-repo enablers: `ariadne#199` (exposed query
API), `ariadne#200` (fleet inventory).
