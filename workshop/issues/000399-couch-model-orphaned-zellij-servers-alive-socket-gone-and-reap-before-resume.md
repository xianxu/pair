---
id: 000399
status: working
deps: []
github_issue:
created: 2026-10-06
updated: 2026-10-06
estimate_hours:
card_mirror: 'fc2df348c93f3a6de69c5528ba50be8975a3c7eb' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-06T19:29:06-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# couch: model orphaned zellij servers (alive, socket gone) and reap before resume

## Problem

A zellij server can lose its socket while the process, and the agent inside
it, keeps running. On 2026-10-06 at 16:47:31 an unsandboxed `go test` of an
uncommitted #397 test edit ran `os.RemoveAll` on the real `$TMPDIR`
(`defer os.RemoveAll(filepath.Dir(filepath.Dir(store)))` walked up from
`$TMPDIR/TestX…/001`). That deleted `zellij-501/contract_version_1/` and
orphaned all 20 session servers: PPID 1, alive, unreachable. `zellij
list-sessions` reported them EXITED.

Couch has no state for this:
- **Startup / resume:** `SessionOwnerProbe.Probe`
  (`launcher/session_owner_os.go:65`) finds the server process, calls
  `action list-panes`, and returns the failure as a hard error. Brain startup
  failed, and `couch --resume pair:2` failed the same way (`observe live session
  panes: zellij --session 📁1-37 … exit status 1`).
- **Recovery report:** `--recover-plan-from-sdlc` classified these threads'
  agents as `parked` and offered `resume`, a step guaranteed to fail. If the
  probe were ever bypassed, it would start a second agent on a conversation
  the orphan was still writing to.

Manual recovery that worked: kill each socketless server (two needed their
`pair wrap` child stopped and then SIGKILL; stray `pair title` helpers had to
be reaped), then re-run the report and its steps. After that, all four
`resume` steps and `reconcile brain:1` succeeded.

## Spec

- Model the session resource state explicitly: **orphaned** = server process
  alive (exact PID + start identity, as the probe already gathers) and its
  socket path absent. It is distinct from live (reachable), parked
  (no process) and unknown.
- `Probe` returns that state instead of an error. Callers decide what to do.
- `--recover-plan-from-sdlc`: the agent dimension reports `orphaned` with the
  PID. The row's steps put a **reap** step first (terminate the server's
  process tree: wrap, term, nvim, title helpers; escalate after a bound;
  reauthorize identity before signalling, like existing server escalation),
  followed by `resume`. Reap is not automatic when the agent may still be
  producing output; the operator confirms.
- Startup and the switcher show a named diagnostic ("📁1-37: server PID N lost
  its socket — reap to resume") instead of the raw zellij exit status.

## Done when

- [ ] Fake-backed test: process present and socket absent → `orphaned`,
      never an error and never `parked`.
- [ ] Recover-plan test: an orphaned thread yields `reap` then `resume`.
      After reap, the next report shows `parked` with `resume` only.
- [ ] Reap kills the whole tree, including the children that ignored SIGTERM
      on 2026-10-06, and leaves no PPID-1 `pair title`/`pair term`/nvim
      residue.
- [ ] Live acceptance: start a session, unlink its socket, run the report,
      reap, resume → succeeds.
- [ ] Atlas: the session-state table gains `orphaned`.

## Plan

Durable plan: `workshop/plans/000399-couch-model-orphaned-zellij-servers-alive-socket-gone-and-reap-before-resume-plan.md`.

- [ ] M1 — Observe and name: the server argv carries its socket path, so an
      orphan is a `zellij --server <socket>` process whose `<socket>` is gone (one
      bulk `ps` per refresh plus an `Lstat` per server). `Probe` returns
      `SessionOwnerOrphaned`; session presence gains `orphaned`; the thread reads
      `unusable/orphaned-server` (never `parked`, never archivable); resume,
      startup and the switcher show "<name>: server PID N lost its socket — reap
      to resume"; the report says agent `orphaned`.
- [ ] M2 — Reap: a pure `PlanReap` over one process-tree snapshot plus an
      identity-gated `Reaper` (TERM, then KILL after a bound, never a recycled
      pid); `couch --reap repo:N --confirm` through the socket; the report's
      steps become `reap` then `resume`.
- [ ] M3 — Live acceptance (unlink one socket, report, reap, resume), atlas,
      lessons.

## Log

### 2026-10-06

- Filed from the brain session that did the manual recovery. A related
  lesson (tests must only `RemoveAll` paths from their own `t.TempDir()`;
  unsandboxed `go test` must redirect `TMPDIR`) belongs in
  `workshop/lessons.md` with this fix.
- During the recovery, the Console would not switch to newly resumed slots
  while the four remote `--resume` jobs (~12 s each, back to back) ran. That
  is likely the capacity-one lifecycle queue refusing ("overloads") the
  switch. Investigate separately: a switch to a live thread shouldn't wait on
  another slot's lifecycle job, or at least needs visible "busy" feedback.
