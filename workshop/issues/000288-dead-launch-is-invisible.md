---
id: 000288
status: working
deps: []
github_issue:
created: 2026-09-18
updated: 2026-09-18
estimate_hours: 2.66
started: 2026-09-18T18:33:40-07:00
flow: {kind: full, provenance: inferred}
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

### Design (2026-09-18)

Durable plan: `workshop/plans/000288-dead-launch-is-invisible-plan.md`.

- **The bound is 10 s**, measured from `LaunchSession`. Birth measured end to
  end was 0.6–1.3 s idle and 1.5–2.9 s with all 12 cores saturated. 10 s is
  3.4× the loaded worst case and still under Couch's 15 s registration
  deadline.
- **Teardown needs proof.** After the bound, the watch asks zellij once:
  - no live session → dead. Cancel the launch.
  - listed live → stand down for good. The session may be healthy with a
    sidecar that failed to write, and ending it would skip the quit cleanup.
  - the probe fails → wait another bound and ask again.
  - The verdict must match its cause. After a dead verdict the launcher fails
    only if the client didn't exit cleanly (`code != 0`), and the watch never
    aborts a launch that has already returned. A clean quit that races the
    probe keeps the normal path and its cleanup.
- **How the client is ended.** `LaunchSession` gains a `ctx`. Cancelling it
  sends SIGTERM, then SIGKILL after 2 s, and returns only once the client
  has been reaped. The hung client (death B) has written nothing to the tty
  and exits about 50 ms after SIGTERM, so the terminal needs no repair.
  After a dead verdict the launcher fails through the existing
  zellij-launch-failure path: layout record restored, message naming
  zellij's log, `code: 1`, no cleanup and no restart re-entry.
- **Death A is left alone.** In the other death mode the client notices the
  lost server and exits 1 itself. Pair already fails there, so this issue
  doesn't touch it.
- **One shared wait (ARCH-DRY).** The new `cmd/internal/panebirth` holds
  `Evidence` (moved from `titlepoller.BirthEvidence`) and `Await`, the one
  poll loop. It serves the title poller, Couch's cold resume and the new
  launcher watch.
- **Couch needs no new state.** A helper that exits is noticed when its pty
  closes. The thread then reads `parked` or `session-gone`, and archive
  accepts both. Out of scope, noted in the plan: Couch shows only
  `exited (1)`, not the launcher's reason.

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*
`sdlc estimate-source` reports the calibration as stale, so this is
provisional. The window runs from the 18:33 claim. Design already in it:

- the birth-time measurements: idle, loaded and cold cache;
- the death-mode repro;
- the Couch map, the plan and two review rounds.

Impl is 40% of the v2 ranges. The design buffer is +15%, because a thorough
plan doc exists. Familiarity is 1.0, because #287 worked in these same three
packages.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec              design=1.0  impl=0.05
item: smaller-go-module       design=0.06 impl=0.14
item: smaller-go-module       design=0.06 impl=0.14
item: smaller-go-module       design=0.06 impl=0.2
item: smaller-go-module       design=0.06 impl=0.14
item: real-api-discovery      design=0.0  impl=0.18
item: atlas-docs              design=0.05 impl=0.05
item: milestone-review        design=0.0  impl=0.14
item: milestone-review        design=0.0  impl=0.14
design-buffer: 0.15
total: 2.66
```

The items, in order:
- `issue-spec` covers the in-window design.
- The four `smaller-go-module`s are:
  - `panebirth` plus moving the poller and Couch onto it;
  - the cancellable `LaunchSession` and the stateful fake;
  - the birth watch, its wiring and the six sequence tests with mutation
    checks (upper impl);
  - the probe's `-exit-wait`/`hung=`.

  Design on each is ×0.2, because the plan pre-resolves it.
- `real-api-discovery` is the live zellij runs before and after the fix
  under `-hammer`.
- `atlas-docs` is the architecture and Couch paragraphs.
- The two `milestone-review`s are the close review plus one expected fix
  round (the close gate runs in rounds).

## Done when

- `probes/zellijbirthrace poke`-style kill of a Pair launch (or a
  fake-runtime test that withholds the pane) ends the launcher with a clear
  error within the bound instead of hanging, and the zellij client is gone.
- A Couch thread whose launch died at birth can be archived without
  restarting Couch.

## Plan

Tasks and steps are in the durable plan.

- [x] Measure the birth-time distribution under load to pick the bound.
- [ ] `panebirth`: shared `Evidence` + `Await`; the title poller and Couch
      move onto it (plan Tasks 1–2).
- [ ] Launcher: a cancellable `LaunchSession` and the birth watch, with
      fake-runtime tests and mutation checks (plan Tasks 3–4).
- [ ] Probe `-exit-wait`/`hung=`; live before/after under `-hammer 10ms`
      (plan Task 5).
- [ ] Atlas + suite + operator Couch smoke: a dead-at-birth thread archives
      without restarting Couch (plan Task 6).
- [x] File the upstream zellij `RemoveClient` panic report (from #287), on
      the operator's go-ahead: zellij-org/zellij#5632.

## Log

### 2026-09-18

Filed from #287. Casualty evidence and the birth-time measurements are in
#287's Log.

#### Design session

- **Birth time** (`probes/zellijbirthrace launch`, base `pair` built from
  main at `6c80d40b`):
  - idle, n=20: 0.61–1.27 s, climbing monotonically across trials as #287
    saw;
  - `yes` ×12 on 12 cores (load avg 33), n=15: 1.49–2.92 s, all born.
- **Death modes** (bare zellij, a Python pty harness in the scratchpad, with
  the socket poked from a tight thread loop):
  - **A:** the client logs "Lost connection to the Zellij server", restores
    the tty and exits 1 by itself.
  - **B, the hang:** the client stays alive and writes 0 bytes, and
    `list-sessions` doesn't list the session. It exits 50 ms after SIGTERM
    (status 15). In the second batch 1 of 5 deaths was B.
  - A healthy client's SIGTERM path prints "Bye from Zellij!" and restores
    the tty.
- **Couch** (Explore digest):
  - When a helper exits, Couch notices when its pty closes
    (`couchtty/console.go` `onExit`), shows `ExitNotice` `"<label> [<actor>]
    exited (N)"`, and drops the pane. The launcher's own stderr is not shown.
  - After the exit, `ClassifyThread` reads `parked` or `session-gone`.
    `ArchiveThread` accepts both, and `clearLifecycleDebris` retires the stale
    `live` incarnation, so no new state is needed.
  - A cold create's claim is `established` before zellij launches
    (`createflow.go` claim vs `LaunchSession`), so Couch's 15 s registration
    deadline doesn't cover it. That is why astro hung without limit.
  - A cold resume waits for birth under that deadline, and the cleanup after
    it rolls back to `parked`.
  - Couch never relaunches on its own.
  - The launcher must reap its zellij child, or the pty stays open.
- **Cold cache** (the probe built with an overlay that gives each trial a
  fresh `HOME`; zellij's cache, data and plugin dirs follow `HOME`), n=4:
  0.44–0.52 s. Plugin compilation doesn't delay the pane sidecar, so the
  10 s bound stands (plan-quality finding 4).
