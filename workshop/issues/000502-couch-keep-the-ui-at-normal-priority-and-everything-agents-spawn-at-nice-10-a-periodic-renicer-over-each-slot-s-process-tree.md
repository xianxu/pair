---
id: 000502
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'e73b5f44b2e697473944b3937323d8c1736ed354' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T15:10:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "6e2165db", done: "e89856dc"}
---

# couch: keep the UI at normal priority and everything agents spawn at nice 10 (a periodic renicer over each slot's process tree)

## Problem

pair#498 (PR #285) starts a slot's whole process tree at nice 10: wrapper, zellij and the agent included. The operator (10-12) wants the interactive layer at normal priority (Couch, the pair wrapper, zellij, the agent process: claude or codex) and only what the agent spawns (Bash tool commands, builds, tests, review subprocesses, MCP servers, stray loops) at lower priority, so the UI stays responsive while work runs low. A renice by hand showed the shape of the problem: slot servers and panes live in separate process groups, and running agents stayed at 0.

## Spec

Decided with the operator, 10-12:
- **A periodic renicer in Couch:** every few seconds (default 5 s; imprecision is fine: a new command at normal priority for its first seconds is acceptable), take one process-table snapshot (prefer the kernel's process table via sysctl over forking `ps`), and for each slot renice to `COUCH_SLOT_NICE` (default 10) every **strict descendant** of the slot's agent process that is below it.
- **Never touch the agent itself, the wrapper, zellij or Couch:** identify the agent by the process ID the wrapper reports, never by name. Without root a priority can't be raised again, so a wrong renice is permanent until restart.
- **Replaces #498's whole-tree nice:** slots launch at normal priority again; #498's setting becomes the renicer's target level (0 = off).
- **Optional, later:** Claude Code's shell-prefix hook (`CLAUDE_CODE_SHELL_PREFIX`, verify the name) to start commands at nice 10 from their first instant; the renicer stays as the agent-agnostic catch-all (Codex, MCP servers, anything).
- **Facts, not policy:** `couch --actors` shows each slot's agent niceness and how many descendants were reniced; one notice if an agent is ever found below normal.
- **Measure the cost:** CPU per scan and per minute at the chosen interval, recorded in the Log (target well under 0.1% of a core).

Operator rule: changes to the live machine (applying the renicer to running slots, changing the running Couch) are proposed to the operator first through the TL.

## Done when

- With the renicer on, a slot's agent, wrapper and zellij stay at 0 while a test suite it starts runs at 10 within a few seconds (verified live on one slot, with the operator's OK).
- The agent-pid-only rule is tested (a decoy process with the agent's name outside the tree is never touched); scan cost measured and recorded.
- #498's whole-tree launch nice is removed or turned into the renicer's level.

## Plan

Durable plan: `workshop/plans/000502-…-plan.md` (same slug). Single pass (no milestones):
- [x] procutil.Table reads Nice
- [x] couchcore.PlanRenice (pure) + table tests incl. the name decoy
- [x] couchcmd renicer loop behind a procSource seam (stateful fake, live conformance on own tree)
- [x] #498 launch nice removed; COUCH_SLOT_NICE is the renicer's level
- [x] --actors facts + one notice; atlas/README
- [x] scan cost measured, in the Log
- [ ] live verification on one slot after the operator's OK (via TL)

## Log

### 2026-10-12
- Claimed at pair:3 after the TL unclaimed pair:5's filing slip. Plan sent to the hub before building.
- Built.
  - `PlanRenice` is pure. The renicer loop sits behind `procSource`, with a stateful fake.
  - #498's launch nice is gone.
  - `--actors` shows `nice=N lowered=N [denied=N]`. A low agent gets one notice ("relaunch the slot to restore its
    priority").
- **Scan cost** (`BenchmarkReniceScan`, this machine, 1,425 processes):
  - 0.42–0.45 ms per pass (one `sysctl kern.proc.all` plus the plan);
  - at 5 s, 12 passes a minute ≈ 5.3 ms of CPU a minute ≈ 0.009% of one core;
  - each newly lowered process adds one identity sysctl plus one setpriority.
- **Sandbox:** the agent sandbox refuses `setpriority` (EPERM). The real-seam tests skip there and pass unsandboxed.
