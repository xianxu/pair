---
id: 000274
status: open
deps: []
github_issue:
created: 2026-09-16
updated: 2026-09-16
estimate_hours:
---

# `pair term` ignores SIGHUP, so a dead session leaks its whole process tree forever

## Problem

When a zellij session ends abnormally, its panes do not die. `pair term` and
everything under it — including the embedded Neovim — survive indefinitely,
reparented to init, still holding their pty.

Measured on the operator's machine, 2026-09-16:

- **106 orphaned `pair term` trees** (`ppid=1`), plus **29 orphaned `nvim`**
  holding **16 GB each at the top end**.
- Machine state before cleanup: **95 GB PhysMem used, 54 GB in the compressor,
  69 MB unused**. After killing the orphans: **18 GB used, 3 GB compressor,
  77 GB unused**. The leak was ~77 GB.
- Oldest survivor started **Sep 6** — ten days of accumulation.
- **Measure with `top`, not `ps -o rss`.** These processes keep nearly all
  their pages in the compressor, so RSS reports a few hundred MB for a process
  whose real footprint is 16 GB. An RSS-based estimate under-reported this leak
  by more than an order of magnitude.

This was found while diagnosing a machine-wide memory shortage: a build step
in a sibling repo was killed by the OS for lack of memory, with ~73 MB of free
pages and 53 GB in the compressor.

## Why it happens

Each pane is spawned by `zellij/layouts/main-3.kdl:86` as

```
sh -c "zellij action rename-pane --pane-id \"$ZELLIJ_PANE_ID\" terminal 2>/dev/null; exec pair term"
```

The `exec` makes `pair term` the pane process itself, and zellij gives each
pane its own pty — so every `pair term` is a **session leader with its own
process group**, holding a controlling terminal (`ps` shows `Ss+`: `s` =
session leader, `+` = foreground group of its tty).

That is deliberate and fine. The consequence is that the pane is *outside*
zellij's process group, so the only mechanism that reaps it when the server
dies is **carrier loss**: the server closes the pty master, and the kernel
sends `SIGHUP` to the foreground process group.

`pair term` does not answer SIGHUP. Verified directly:

- `kill -HUP <pid>` on orphan 56264 → still `Ss+`, unaffected.
- `lsof /dev/ttys001` shows **only `pair` itself** on fds 0/1/2 — the master
  end is long gone, so the hangup had already fired with nothing to act on.
- `cmd/internal/termcmd/` contains no signal handling at all, so `pair term`
  is not installing the ignore itself.

**Leading hypothesis for the origin, not yet proven:** the disposition is
inherited as `SIG_IGN`. Go preserves an inherited `SIG_IGN` rather than
installing its own terminating handler, so if the daemonized zellij server
(`ppid=1`, no tty) has SIGHUP ignored, every pane process inherits it. The
repo already documents this hazard in the same idiom —
`cmd/internal/titlepoller/runcli.go:36-40` explicitly calls
`signal.Ignore(syscall.SIGHUP)` with the comment *"bin/pair spawns us with `&
disown` … a terminal teardown would SIGHUP us and freeze the pane"*, and
`cmd/internal/launcher/osruntime.go:413` refers to "the historical disown
behavior". Confirming the inheritance path is the first task below; macOS `ps`
will not show signal masks, so it needs a deliberate probe.

**`SIGTERM` is NOT sufficient**, corrected after measuring the cleanup: sending
SIGTERM to all 362 pids in the orphaned trees killed the `pair term` leaders
but the embedded `nvim` children **survived it and were reparented to init**,
converting "orphaned trees" into "orphaned nvim" and freeing comparatively
little. They needed `SIGKILL`. Any sweep this issue adds must escalate, and
must verify descendants are gone rather than assuming the leader's death
cascades.

## Spec

A pane process must not outlive the session that owns it. Concretely:

- When a zellij server exits — cleanly, by crash, or by `kill -9` — every
  `pair term` it spawned, and every descendant of those, must terminate.
- The mechanism must not depend on the operator being attached, on a clean
  shutdown path, or on the session being ended through pair's own UI. The
  common case in the measured data is a session that died *without* a graceful
  teardown.
- Whatever makes a pane survivable must be scoped to the processes that
  genuinely need it. If `& disown` / `SIG_IGN` is load-bearing for the
  titlepoller or the opener, those keep it; a pane's shell does not inherit it.
- Termination must be observable: after a session ends, `ps -Ao ppid=,args= |
  awk '$1==1' | grep "pair term"` is empty.

Pick one of two shapes and say why:

1. **Do not inherit the ignore.** Reset the SIGHUP disposition to `SIG_DFL` in
   the pane's process before `exec pair term`, so carrier loss reaps it the way
   the kernel intends. Smallest change, and it restores the standard mechanism
   rather than adding a second one.
2. **Make the server own its panes.** Have the session record its pane pids and
   sweep them on exit. More machinery, and it cannot cover `kill -9` of the
   server, which is exactly the case that leaks today.

Option 1 looks right; option 2 alone would not have prevented the measured
leak. They are not exclusive.

**The orphan is not dormant — it is a runaway, and that is why it is so
expensive.** Measured against the healthy population on the same machine
minutes later:

| | count | size each | %CPU | accumulated CPU |
|---|---|---|---|---|
| healthy embedded nvim (attached sessions) | 30 | **25–92 MB** | **0.0%** | seconds |
| orphaned nvim | 29 | **6–16 GB** | 5–47% | **28–49 min** |

An embedded nvim is ~50 MB when its session is alive. It reaches 16 GB only
after being orphaned, while burning CPU the whole time — the machine's load
average was **43.38** with nothing actually running. So the memory is a
*symptom* of the orphaning, not an independent leak or a configuration
problem: something in the orphan retries forever (most plausibly writing to a
pty whose master is gone and getting an error back) and accumulates as it
spins.

That makes the fix more urgent than "stale processes linger": every abnormally
ended session leaves behind a process that actively consumes CPU and grows
without bound until the machine runs out of memory.

Worth confirming during the fix: identify the loop. If the orphan's growth is
the retry path, then making it die on carrier loss removes both problems at
once; if it can also spin while *attached*, that is a second bug and needs its
own issue. No evidence of the latter — the healthy population is flat at 0.0%
CPU and under 100 MB.

## Done when

- A pane process started through the zellij layout dies when its server is
  killed, including `kill -9` of the server, verified by an automated test
  rather than by hand.
- The SIGHUP disposition of a live `pair term` is `SIG_DFL`, and there is a
  check that fails if a future launch path re-introduces an inherited
  `SIG_IGN` — the finding here is a *class*: any process pair spawns into a
  pane inherits whatever disposition the launcher leaves.
- Whatever processes legitimately need to survive teardown (titlepoller,
  opener, clipcmd's detached helpers) still do, with a test naming each.
- Cleanup guidance exists for machines already carrying orphans, since the
  fix does not retroactively reap the 106 trees measured above.
- `atlas/` documents which pair processes are session-scoped and which are
  deliberately detached, and what reaps each.

## Plan

- [ ] Prove the disposition and its origin. *(Disposition PROVED 2026-09-17 from
  #256 M3 — see Log; what remains is the ORIGIN: walk the launch chain (zellij
  server → `sh -c` → `pair term`) to find where `SIG_IGN` enters.)*
- [ ] Enumerate every process pair spawns and classify each as session-scoped
  or deliberately detached — the fix must not reap the second group.
- [ ] Implement the chosen shape; restore `SIG_DFL` for pane processes.
- [ ] Add a test that starts a session, `kill -9`s the server, and asserts no
  surviving descendants.
- [ ] Document the process taxonomy in `atlas/`, and record a one-liner for
  reaping pre-existing orphans.

## Log

### 2026-09-17 — the inheritance hypothesis is PROVED, from #256 M3

#256 M3 Task 10 needed to know what `zellij delete-session --force` reaps before
it could word archive's confirmation, so it ran the probe this issue's first Plan
item asks for. Recorded here because the answer is this issue's, not #256's.

Two runs, `zellij 0.45.1`, same fixture, throwaway session
(`pair256-quiesce-probe` / `pair256-inherit-probe`), one variable — the SIGHUP
disposition of the shell that launched the session:

| Launching shell | Pane child after `zellij delete-session --force` |
|---|---|
| default SIGHUP | **gone** |
| `trap '' HUP` (SIG_IGN) | **alive**, reparented to PID 1 |

Within the first run, a child with `trap '' HUP` survived while its sibling with
the default disposition died, so the reaping mechanism is confirmed to be
**SIGHUP to the pane's foreground process group** — not SIGKILL, and not a
zellij-side process sweep.

So the hypothesis in **Why it happens** is right: the disposition is inherited,
`SIG_IGN` anywhere up the launch chain reaches every pane process, and Go
preserves it. The first Plan item narrows from "prove the disposition and its
origin" to just the origin — *which* link in couch's chain (bin/pair's historical
`& disown`, the zellij server's own launch, or `pair wrap`) introduces it.

**Consequence already shipped in #256 M3:** archive's confirmation on a detached
row says the session stops *and its running agent may survive*, rather than
promising a stop couch cannot deliver. When this issue lands, that wording gets
to become a promise.

### 2026-09-16

Filed from parley.nvim#262's session, where a `sdlc milestone-close` was killed
by the OS for lack of memory. Diagnosis found 80 orphaned parley test processes
(parley.nvim#220, reclaimed) and then this, an order of magnitude larger.

Evidence is in the Problem section: 106 trees / 16.93 GB / oldest Sep 6;
`kill -HUP` verified ineffective on a live orphan; `lsof` showing the pty held
only by the orphan itself; `termcmd` free of signal handling.

The operator's three attached sessions at the time (`📁pair`, `📁tools`,
`📁parley-nvim`) were identified by their zellij client processes and are
unaffected by this bug — the leak is entirely in sessions whose server is gone.
