---
id: 000399
status: working
deps: []
github_issue:
created: 2026-10-06
updated: 2026-10-06
estimate_hours: 3.65
card_mirror: '59006bed5b8af141123a805e7d07caf90e7fd79b' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-06T19:29:06-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
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

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module        design=0.1 impl=0.2
item: smaller-go-module        design=0.1 impl=0.2
item: smaller-go-module        design=0.1 impl=0.2
item: milestone-review         design=0.0 impl=0.2
item: greenfield-go-module     design=0.2 impl=0.32
item: smaller-go-module        design=0.1 impl=0.2
item: greenfield-go-module     design=0.2 impl=0.32
item: smaller-go-module        design=0.1 impl=0.2
item: skill-or-dispatcher      design=0.1 impl=0.12
item: milestone-review         design=0.0 impl=0.2
item: atlas-docs               design=0.05 impl=0.08
item: milestone-review         design=0.0 impl=0.2
design-buffer: 0.15
total: 3.65
```

Design hours are discounted because the durable plan is reviewed and approved;
`impl=` values are 40% of the v2 primitive ranges (v3.1).

- `smaller-go-module` — M1 server parsing with socket path, SocketState, Probe's orphaned state
- `smaller-go-module` — M1 SessionOrphaned in presence + ReasonOrphanedServer through classification/actions
- `smaller-go-module` — M1 resume diagnostic code, startup/switcher text, report agent `orphaned`
- `milestone-review` — M1
- `greenfield-go-module` — M2 PlanReap + identity-gated Reaper over a fake process table
- `smaller-go-module` — M2 `reap` operation surface (ops, dispatch, CLI, socket)
- `greenfield-go-module` — M2 `recover` action: ConfirmByPlan, PrepareRecover, menu frame, CLI
- `smaller-go-module` — M2 report steps `reap` → `resume`, totality
- `skill-or-dispatcher` — M2 couch --skill recovery line
- `milestone-review` — M2
- `atlas-docs` — M3 atlas session-state table, lessons
- `milestone-review` — M3 close (live acceptance run by the operator)

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* (Calibration doc flagged stale; numbers provisional.)

## Plan

Durable plan: `workshop/plans/000399-couch-model-orphaned-zellij-servers-alive-socket-gone-and-reap-before-resume-plan.md`.

- [x] M1 — Observe and name: the server argv carries its socket path, so an
      orphan is a `zellij --server <socket>` process whose `<socket>` is gone (one
      bulk `ps` per refresh plus an `Lstat` per server). `Probe` returns
      `SessionOwnerOrphaned`; session presence gains `orphaned`; the thread reads
      `unusable/orphaned-server` (never `parked`, never archivable); resume,
      startup and the switcher show "<name>: server PID N lost its socket — reap
      to resume"; the report says agent `orphaned`.
- [x] M2 — Reap: a pure `PlanReap` over one process-tree snapshot plus an
      identity-gated `Reaper` (TERM, then KILL after a bound, never a recycled
      pid); `couch --reap repo:N --confirm` through the socket; the report's
      steps become `reap` then `resume`. A switcher Tab action **recover** runs
      that row's report steps (resume / reap→resume / reboot, or refuses on a
      hold); its confirmation follows the steps (operator decision 2026-10-06).
- [ ] M3 — Live acceptance (unlink one socket, report, reap, resume), atlas,
      lessons.

## Log

### 2026-10-06
- 2026-10-06: closed M1 — Round 3. BR-2 fixed as a rule (lessons.md): the startup refusal lists the tree while the server parents it, kills descendants, then the server; TestOrphanRefusalStepsWouldHaveWorkedOnTheIncident checks the order and forbids the 2026-10-06-failing commands. Round-2 minors: ServerVerdict tagged enum (contradictory flags unrepresentable), TestManyThreadsWithAnOrphanReadOrphaned; the starting-server provisional rule recorded as an M2 requirement in the plan revision. couchcore, couchtty, couchcmd, launcher all pass in a full scrubbed run (all PAIR_/COUCH_/ZELLIJ unset, short isolated TMPDIR).; review verdict: SHIP

- Filed from the brain session that did the manual recovery. A related
  lesson (tests must only `RemoveAll` paths from their own `t.TempDir()`;
  unsandboxed `go test` must redirect `TMPDIR`) belongs in
  `workshop/lessons.md` with this fix.
- During the recovery, the Console would not switch to newly resumed slots
  while the four remote `--resume` jobs (~12 s each, back to back) ran. That
  is likely the capacity-one lifecycle queue refusing ("overloads") the
  switch. Investigate separately: a switch to a live thread shouldn't wait on
  another slot's lifecycle job, or at least needs visible "busy" feedback.

### 2026-10-07 — M1 implementation notes
- 2026-10-07: closed M2 — Round 2. BR-9: a confirmation never outlives its op success (default rule; TestReapSuccessClosesItsConfirmation failed before). BR-10: title poller is Setsid-spawned outside the server tree, so reap also runs the tag pidfile helper reapers (TestReapTagHelpersEndsTheHelpersOutsideTheTree; refuses without a data dir); lesson. BR-11: startup refusal names Tab → recover, no hand-written kill recipe (test forbids kill/pkill). BR-12: README documents --reap/--recover/orphaned/Tab → recover. Artifact inventory lists no #399 file. couchcore, couchtty, couchcmd, launcher, couchmessage pass in a full scrubbed run. actual = measured 2.99 - M1 1.35.; review verdict: SHIP

- e7c0f173: one server type with its socket path (PQ-1; the lowercase copy
  deleted), one argv parser (fuzzed), `SocketState` that trusts only ENOENT (PQ-2).
- dfec2c1a: `Probe` asks the socket before `list-panes`. A gone socket gives
  `SessionOwnerOrphaned` plus one sentence; an unknown socket gives Unknown.
- 5c610a37: session presence takes one host server snapshot per refresh (an
  injectable `Servers` seam; tests use a fake table) and projects
  `SessionOrphaned` for names `list-sessions` doesn't report live.
- 955ed445: `unusable/orphaned-server` outranks every no-session reading and the
  record's own faults. It is not archivable or rebootable, and nothing is offered.
- 343a1655: resume refuses with `resume-orphaned-server` (never reboot advice).
  Rows carry the orphan so the switcher names the pid.
  **Found during design:** startup ignored unusable rows as debris, so a repo
  whose primary was orphaned would have started a SECOND agent beside the
  running one. Startup now refuses, naming the server. Test-first: the new test
  failed with "a second primary started beside an orphaned one".
- Recovery report: agent `orphaned` with `{pid, session}`, class and hold
  `orphaned-server`, no steps (M2 adds reap → resume). The orphan rule outranks the
  workspace-handoff rule.
- Test runs inside this live slot need EVERY `PAIR_*`/`COUCH_*`/`ZELLIJ*` var
  unset, plus a scratchpad `TMPDIR` (the #399 lesson); the five-var scrub leaks.
  couchcore takes 505–532 s, close to `go test`'s default 10-minute limit; one
  run timed out under load, so runs use `-timeout 30m`.

### 2026-10-07 — M2 progress

- 33b3f4a6: `launcher.PlanReap` and `Reaper`. One snapshot while the server still
  parents its tree; SIGTERM deepest-first with the server last; SIGKILL survivors
  by pid after a bound; identity re-read before every signal; a changed server is
  refused with no signal; a SIGKILL survivor is named rather than waited on. The
  fake table models 2026-10-06 (a TERM-ignoring wrap that reparents). Mutation
  check: server-first ordering fails two tests.
- ac85310f: `Couch.Reap`. Admits only an orphaned row, and only if the verdict
  holds again, for the same server identity, after 1 s (M1 review: one snapshot is
  provisional). `launcher.OSOrphanReaper` runs the tree kill, then the existing
  quiescence loop for zellij's EXITED record. `ActorActions` offers `reap` (only)
  on an orphan, so the switcher shows it beside resume/reboot (one admission
  table; the menu is deliberately unfiltered).
- Next: the operation surface (ops, dispatch, slotOperations, CLI `--reap`,
  socket, switcher confirmation/progress text), the report steps reap → resume,
  then Task 9b `recover`.
