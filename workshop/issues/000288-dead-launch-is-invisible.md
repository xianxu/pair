---
id: 000288
status: open
deps: []
github_issue:
created: 2026-09-18
updated: 2026-09-18
estimate_hours:
---

# A launch whose zellij server dies at birth hangs the launcher and leaves Couch's thread live

## Problem

Split out of #287 (its "out of scope" row). When a new zellij server dies
before its first client initializes the session:

- the zellij client hangs on a cooked tty, so the operator sees a blank pane
  that echoes typed keys;
- `LaunchSession` never returns, so the Pair launcher waits forever;
- Couch keeps the thread's incarnation `live`, because the recorded helper
  (the hung launcher) is alive. Archive and park both refuse, and the
  switcher shows "session gone".

Observed with astro thread `couch-3a64268b355e8183` (`📁astro-couch-6`),
2026-09-18. The launcher (pid 23676) hung for ~2 h. Nothing noticed. The
thread was only freed when restarting Couch killed the launcher's pty; after
that Couch saw the helper as dead and archive went through.

#287 removed Pair's own trigger, a birth-window probe (0/20 died after the
fix). But any machine-wide `list-sessions` can still kill a birth, and zellij
0.45.1's `RemoveClient` panic is not the only way a server can die young. A
dead birth should fail loudly, not look alive.

## Spec

- **The launcher notices a birth that never happens.** #287's evidence is
  already the right signal: the agent pane's sidecar, cleared by the create
  path, is written by the pane only once the session has initialized. While
  `LaunchSession` blocks, the launcher waits for that sidecar with a bound.
  If it doesn't appear, it checks whether the session exists (safe after
  the bound), and if not, tears down the hung client and exits non-zero
  with a message naming the zellij log.
- **The bound.** Couch's registration deadline is 15 s; measured birth is
  0.65–1.4 s. Pick the bound from measurements. Too short would kill a slow
  but healthy launch.
- **Couch.** A launcher that exits non-zero ends the helper, so the thread
  becomes recoverable or archivable through the existing paths. Check that
  no new state is needed there.

## Done when

- `probes/zellijbirthrace poke`-style kill of a Pair launch (or a
  fake-runtime test that withholds the pane) ends the launcher with a clear
  error within the bound instead of hanging, and the zellij client is gone.
- A Couch thread whose launch died at birth can be archived without
  restarting Couch.

## Plan

- [ ] Measure the birth-time distribution under load to pick the bound.
- [ ] Launcher: bounded pane-birth watch alongside `LaunchSession`, with
      teardown on timeout, plus fake-runtime tests. This is the third waiter
      on pane birth, after `titlepoller.awaitPaneBirth` and
      `couchcore.awaitPaneBirth`: extract one shared helper then (the ARCH-DRY
      note from #287's close review). Key it on
      `titlepoller.BirthEvidence`, the single declaration of the awaited file.
- [ ] Live check with the probe, and a Couch archive of a dead-at-birth
      thread.
- [x] File the upstream zellij `RemoveClient` panic report (from #287), on
      the operator's go-ahead: zellij-org/zellij#5632.

## Log

### 2026-09-18

Filed from #287. Casualty evidence and the birth-time measurements are in
#287's Log.
