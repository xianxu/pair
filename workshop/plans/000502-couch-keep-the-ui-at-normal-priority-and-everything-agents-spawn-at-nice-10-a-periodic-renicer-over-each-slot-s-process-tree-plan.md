# #502 plan: a periodic renicer, the UI at normal priority

Issue: `workshop/issues/000502-…md`.

## What's there (measured from the tree)

- **#498:** `couchcore/launchhelper.go` `startBlockedChild` runs
  `unix.Setpriority(PRIO_PGRP, child, COUCH_SLOT_NICE)` on the launched slot child, before exec, so the whole tree
  starts at 10. `couchcmd/nice.go` `slotNiceSettings` parses the level (default 10, 0 = off, 0–20).
- **Agent pid:** `pair wrap` writes it to `Paths.AgentPID()` as a bare decimal after the agent starts
  (`wrapcmd/wrap.go`), and removes it on exit. A crash leaves it stale. The agent is the wrapper's direct child.
- **Wrapper:** the message broker holds each connected slot's `Binding{Slot, Scope, Tag, PID, Start}`. `PID` and
  `Start` are the wrapper's pid and start identity.
- **Process table:** `procutil.Table()` (darwin) is one `sysctl kern.proc.all` call →
  `[]Process{PID, PPID, Identity}`; Linux reads `/proc`. Niceness isn't read yet (`P_nice` is there).
- **`couch --actors`:** answers from the broker's memory (`Broker.Actors` → `[]Candidate`), and the text is
  printed in `couchcmd/messages.go`.

## Design

### Pure core: `couchcore/renice.go` (ARCH-PURE)

```go
type ReniceRoot struct { // one slot, from the broker's binding and its agent-pid file
	Slot              string
	WrapperPID        int
	WrapperStart      string // the binding's Start (wrapper identity)
	AgentPID          int    // the agent-pid file; 0 when unread
}
type ReniceProc struct{ PID, PPID, Nice int; Identity string }
type ReniceSlot struct {
	Slot      string
	Agent     ReniceProc // zero when unconfirmed
	Confirmed bool
	Lower     []ReniceProc // strict descendants of the agent below level: nice < level
	AtLevel   int          // descendants already at or below level
}
func PlanRenice(table []ReniceProc, roots []ReniceRoot, level int, never []int) []ReniceSlot
```

- **The agent is trusted only when confirmed in the snapshot:**
  - the agent pid's `PPID` is the binding's wrapper pid;
  - the wrapper's identity in the snapshot is the binding's `Start`.
  - A stale or hand-edited agent-pid file, or a reused pid, never selects a tree. Names are never read.
- **Lower:** the agent's strict descendants (BFS over `PPID` in the same snapshot) whose nice is below `level`.
- **Never:** the agent itself, the wrapper, Couch's own pid and its ancestors, and any pid in `never` are
  excluded, even if they show up as descendants.
- **level 0:** nothing is lowered. Facts are still reported.

### IO shell: `couchcmd/renicer.go`

- **The loop:** a ticker (default 5 s) on the console's lifetime, as `slotGitTicker` does, one pass at a time.
  Each pass:
  1. reads the broker's connected bindings, the slots;
  2. reads each slot's agent-pid file with the hardened reader (`readMessageWrapperPID`'s rules: `O_NOFOLLOW`, a
     regular file, at most 32 bytes, a canonical integer);
  3. takes one `Table()` snapshot (`procutil.Table`, extended with `Nice`);
  4. calls `PlanRenice`;
  5. for each process to lower, re-checks its identity (`procutil.Identity(pid)`, one sysctl) and only then
     calls `unix.Setpriority(PRIO_PROCESS, pid, level)`.
- **Setpriority results:**
  - ESRCH (the process exited) is ignored.
  - EPERM/EACCES is counted and reported in `--actors`, never retried in a loop: a process at a different uid is
    not ours.
- **Seam (ARCH-MOCK):** `type procSource interface { Table() ([]ReniceProc, error); Identity(pid) (string, error); Renice(pid, nice int) error }`.
  - The real one is procutil plus unix.
  - The stateful fake is a process table whose `Renice` changes that process's nice, and whose processes can be
    spawned, exited or reused between calls.
  - Integration tests run the loop against it.
- **Live conformance:** a test starts a real `sh` → `sleep` tree of its own, runs the real `Table` and `Renice`
  on it, and checks that the grandchild went to the level and the parent didn't. It only touches the test's own
  processes.
- **Facts:**
  - The broker gets `SetNiceView(map[slot]NiceFacts{AgentNice, Lowered, Denied})`, read by `Actors`.
  - `couch --actors` adds `nice=<agent> lowered=<n>` to each line and the same fields to the JSON.
  - A slot whose confirmed agent is found above 0 (below normal priority, e.g. a #498 slot until it restarts)
    produces one notice per binding.

### Replace #498's launch nice

- `startBlockedChild` no longer lowers the process group, and slots launch at normal priority.
- `COUCH_SLOT_NICE` keeps its parsing (`slotNiceSettings`) and becomes the renicer's level: 0 turns the renicer
  off, unset is 10.
- #498's `couchcore/nice_test.go` grandchild-inherits test is replaced by "a launched child runs at Couch's own
  nice".
- Atlas `couch.md` (the #498 paragraph) is rewritten.

### Optional (not in this issue)

- Claude Code's shell-prefix hook, to start commands at nice from their first instant, is filed separately if
  wanted. The renicer stays the agent-agnostic catch-all.

## ARCH notes

- **CONSTRAINTS:**
  - **Workload:** background, a 5 s tick, one sysctl snapshot. That is about 1000 processes on this machine; the
    count is measured, not assumed.
  - **Target:** well under 0.1% of a core.
  - **Measure:** a benchmark of `Table()` plus `PlanRenice` on the real table, run once (light), giving cost per
    scan, per minute and as a % of a core in the Log.
  - **Over budget:** the interval is configurable (`COUCH_RENICE_INTERVAL`, minimum 1 s); a slow pass skips
    ticks rather than queueing them.
- **ORDER:** the renicer holds no state between ticks except facts (per binding) and the "noticed" set. The
  events that matter:
  - A process exits between the snapshot and the renice: ESRCH, ignored.
  - A pid is reused between the snapshot and the renice: the identity re-check skips it.
  - The agent-pid file is stale or a pid reused: unconfirmed, nothing lowered.
  - A new command at normal priority: lowered on the next tick (≤ 5 s, accepted by the operator).
  - A binding disconnects: its facts and noticed entry are dropped on the next pass.
  - The tick is single-flight, so passes never overlap.
- **SECURE:** the agent-pid file is untrusted input (another process's file under the scope dir): it is parsed
  strictly and confirmed against the wrapper's identity. The renicer can only lower priority, never raise it,
  and only within a confirmed slot's tree. It touches no credentials.
- **FUNERAL:** creates nothing durable: facts live in memory, keyed by the binding and pruned when it goes.
- **MOCK:** the procSource seam, its stateful fake, and the live conformance test above.

## Plan

- [ ] `procutil.Table` reads `Nice` (darwin `P_nice`, linux stat field 19). Tests.
- [ ] `couchcore.PlanRenice` and its table tests:
  - confirmation, including a stale file, a reused pid and a wrong wrapper;
  - the **decoy**: a process named like the agent, outside the tree, is never touched;
  - strict descendants only; never the agent, wrapper or Couch;
  - level 0; already low.
- [ ] `couchcmd/renicer.go`: the loop, the seam and the stateful fake, plus integration tests against the fake
  (exit, reuse, EPERM, a disconnected binding, single-flight). Live conformance on its own process tree.
- [ ] Remove #498's launch nice, and make `COUCH_SLOT_NICE` the renicer's level. Update the tests.
- [ ] `--actors` facts and the one notice. Atlas and README.
- [ ] Measure the scan cost and record it in the Log.
- [ ] Live verification on one slot, **after the operator's OK through the TL**: the agent, wrapper and zellij
  stay at 0, while a test suite the slot starts runs at 10 within a few seconds.
