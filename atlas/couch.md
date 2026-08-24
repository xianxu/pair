# couch — the session supervisor

`couch` is a second binary in this repo (`cmd/couch`) that supervises agent
sessions: it registers them, spawns them, and knows what is running where. It
is **not** an extension of `pair`. pair is what the operator sits inside, so a
supervisor bug must not break the ability to fix it; the fallback is always to
launch pair the old way.

Project: `workshop/projects/couch.md`. Registry/spawn shipped in `pair#145`;
the console and switcher through the actor panel shipped in `pair#146` M1-M3.

## What exists today

A registry persisted to `~/.local/share/pair/couch/registry.json`, and a set of
operations over it.

**The operation set is deliberately not listed here.** `couchcore.Operations()`
is the single source, the CLI dispatches through it, and a test asserts the two
are identical -- so any list in prose is a second copy that drifts. It already
did: this file named six operations while seven shipped. Run `couch --help`,
which renders the declared set.

**couch hosts `pair` whole.** The stack is couch → pair → zellij → agent+nvim.
couch starts `pair resume <tag> --layout2` inside a child pty and owns the
operator tty until the console exits. Verified by operator smoke; the
alternative (couch absorbing zellij's role) was considered and rejected because
the agent child is never spawned by Go — zellij spawns it from a KDL layout, and
`entrypoint.ValidRootMarkers` *defines* a valid pair install as having those
layouts.

**`couch start` IS the console (`pair#146` M2).** It allocates a pty per child,
puts the operator's terminal in raw mode, and routes bytes -- so it no longer
hands the child its own stdio and block. The mechanism is shared with `pair term`
rather than written twice: `cmd/internal/ptychild` (a child on a pty, its
bounded replay ring, the #127 query deny-list, one scanner over its output) and
`cmd/internal/hostty` (the operator's terminal: size, raw mode, coalesced
resizes, the control constants). See `atlas/architecture.md`, "The terminal
plumbing is shared with couch".

`--no-console` keeps the stdio-inheriting path, and announces itself loudly. It
is not dead code kept for symmetry: it is the fallback if the tty layer
misbehaves, which is why `ExecRunner` stays a live production path with a live
conformance check behind it.

**The pty is a CAPABILITY on a handle, not a second Runner signature.**
`Runner.Start` is unchanged; a handle from `PtyRunner` additionally satisfies
`TerminalHandle`. `ExecRunner`'s does not, and a test asserts that -- a
capability check no runner can fail is vacuous. `Terminal()` returns the
concrete `*ptychild.Child` rather than an interface, because `FakeRunner`'s
double IS one, so a test takes the branch production takes.

## The reserved row: a reservation, not compositing

The child is given a terminal one row shorter and the host's scrolling region is
pinned above the last row (DECSTBM). The child is never told, so this is a
resize rather than compositing.

Three things that design has to survive, each learned the expensive way:

- **Scrolling.** What DECSTBM is for. A child scrolling at the bottom of its own
  screen scrolls inside the region.
- **Erasing.** DECSTBM does *not* cover it. Every full-screen app clears the
  display on startup and that takes the row with it while the region stays
  intact -- which is why the signal is `Screen.TakeRowDirty` (erase, margin
  reset, RIS, alt-screen transition) rather than anything named for the region.
  The console repaints on it.
- **Not corrupting the child.** A pty read boundary falls wherever the kernel
  puts it, so a paint written between two chunks can land inside one of the
  child's escape sequences. Two rules keep that impossible rather than unlikely:
  **`Console.Run` is the only goroutine that writes to the host** (resizes and
  hotkeys are events it drains, not writers), and every console-originated write
  goes through a gate that defers while the CHILD's stream is mid-sequence,
  paying the debt on the next chunk that ends on a boundary.

  Both halves were learned by getting them wrong. Asking the *child* whether it
  was mid-sequence answered about a later chunk, because ptychild's pump feeds
  its scanner before the console has drained the earlier one -- so the tracking
  belongs to the stream the console WRITES. And feeding the console's own
  escapes into that scanner let it frame our bytes together with the child's
  partial and report "safe" precisely when it was not, so the scanner is fed
  child bytes only.

Verified against a real terminal emulator (`vtscreen_test.go`) and against a
real pty child (`console_live_test.go`, `PAIR_LIVE_COUCH=1`), and confirmed by
operator smoke on the full Ghostty -> couch -> pair -> zellij -> claude stack
2026-08-23.

## Navigation

`ctrl-space` is intercepted before the child sees it. It arrives in TWO
encodings and both are recognised: the legacy `0x00`, and CSI-u
`\x1b[32;5u` under the Kitty keyboard protocol, which zellij enables -- so the
legacy byte is the one a real session almost never sends. The interceptor
returns a SPLIT (bytes for the focus being left, bytes for the focus landed on),
because a concatenated buffer cannot say which child the tail belongs to. It
suspends inside a bracketed paste: a pasted NUL that switched actors and ate a
byte would be untraceable data loss.

The focus ladder is deliberately small: a non-root child goes to the root
actor, the root actor goes to couch's panel, and the panel stays put. Liveness
is consulted before going home so a dead root cannot become a frozen landing.
The stdin pump does not treat a `Read` boundary as an event boundary: after it
finds a hotkey, it waits for the Run loop to acknowledge the focus transition
before routing the suffix. The same stream rule holds for legacy Escape in the
panel — a bare ESC is held briefly because it may be the first byte of a split
arrow sequence; the Run loop's ambiguity timer turns it into an Escape key only
when no continuation arrives.

The panel is couch's own screen. It owns input while visible, suppresses
background-child painting, and supports arrows + Enter, Escape, direct
typeahead, and a `:` command namespace (`:1`–`:9`, `:s`, `:x`, `:n`, `:d`).
Every action
dispatches through `couchcore.Operations()`; `start`'s returned `StartResult` is
load-bearing because the console consumes it to attach the new terminal child.
The printable namespace is intentionally collision-free: ordinary letters and
digits always begin a filter, rather than sometimes becoming a command because
the query happened to be empty.

A panel row carries two identities that must not be conflated: the canonical
worktree feeds the shared human resolver, while the console-local child id is
the deterministic switch and bell target. `Couch.LookupTrees` is the one match
rule for the panel, CLI and future advisor; it searches the displayed repo-name
fallback, operator name/description, and agent-published description. This is
why a row displayed as `pair` is findable by typing `pair`.

Rows themselves always start at `Couch.Summarize(nil)`, the same durable source
as `couch list`; a pure join adds only hosted-child routing IDs and bell state.
That direction matters: building rows from hosted panes would silently omit a
parked tree and leave successful name/description changes stale in the running
panel.

## Spawning: `pair resume <tag> --layout2`

The tag derives from the worktree root, so re-entry is deterministic and a
console restart reattaches the same session rather than landing on pair's
session picker. Layout is pinned to layout2 for now: couch owns terminal
switching, so layout3's third pane is the layer couch replaces.

On a COLD create, couch asks Pair to use the repo's saved agent-argument default
without opening `runConfigPicker`; no default means no user-configured args.
Pair consumes the temporary `PAIR_USE_REPO_DEFAULT=1` handoff at process entry
and carries only typed launch policy downstream, so sidecars, zellij, and panes
cannot inherit it. Existing live sessions still take the attach path unchanged.
Direct `pair resume <tag>` still owns the saved-config choice, and direct
`pair -- <agent-arguments>` is the current way to replace the repo default.

## The agent-facing operation

`couch publish-description` is run BY a session, inside its own tree, not by the
operator -- which is why it is the one operation the README does not carry. A
spawned child is told `$COUCH_TREE`, so the agent can name what it is working on
in one line, and `Describe` prefers that sidecar over anything the operator
typed. It is a LABEL, not state: a stale one still finds the right tree, which
is why it is allowed to go stale where a published status document would not be
(see the cold-revival experiment in the project file).

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
processes, disk, wall-clock or randomness. The seam set itself lives in
`Couch`'s struct fields in `cmd/internal/couchcore/couch.go` -- read it there
rather than from a list here, for the same reason the operations are not
enumerated.

The property that matters: each seam has a fake, and the fakes that model
*behaviour* rather than data are compared against the real thing by
`conformance_live_test.go` (gated on `PAIR_LIVE_COUCH=1`). That check found a
real bug -- `Alive()` reporting a zombie as running -- which no test against the
fake could have.

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

## Actor loop — built, unit-tested, not yet instantiated

`Actor` exists and is tested, but **no command starts one**: `Couch.Spawn`
launches a child and returns. It is groundwork for `pair#147`, where messages
between actors begin to exist. Described here because the design constraints are
the interesting part, not because a running couch has one.

The intended shape is one goroutine per actor, holding a bounded mailbox. `Enqueue` is a pure function
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

`pair#146` M4 exits/detach/notices · `pair#147` cluster transport and queries ·
`pair#148` brain as advisor. Cross-repo enablers: `ariadne#199` (exposed query
API), `ariadne#200` (fleet inventory).
