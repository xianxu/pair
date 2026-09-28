---
id: '000288'
status: done
started: 2026-09-18T18:33:40-07:00
created: 2026-09-18
updated: 2026-09-18
estimate_hours: 2.66
actual_hours: 1.92
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
