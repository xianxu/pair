---
id: 000329
status: working
deps: []
github_issue:
created: 2026-09-25
updated: 2026-09-28
estimate_hours:
card_mirror: '216f5292ae852f73324b0544ff18d2cc097e6c2e' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-28T11:21:37-07:00
flow: {kind: quick, provenance: inferred, spec: "b257de4d", done: "4661ada9"}
---

# Detach kills an unbound session watcher

## Problem

A thread that starts a new conversation needs a turn before it can be
relaunched. The `pair session-watch` sidecar waits for the agent's first
completed turn and writes the ledger binding. For a Couch-launched pair, that
sidecar stays in the actor's process group on purpose
(`cmd/internal/launcher/osruntime.go:461`), so Couch can clean up a failed
start.

Detach ends that group (`cmd/internal/couchcore/detach.go:27`: "the
session-watcher and title-poller sidecars sharing its process group go with
them"). Reattach is a `pair resume <tag>` attach: it restarts the title poller
(`cmd/internal/launcher/lifecycle.go:108`) but never the watcher. Only the
create path starts one (`cmd/internal/launcher/createflow.go:784`).

So a thread detached (or through a Couch restart) before its first completed
turn stays unbound for good. Alt+n refuses it with "its agent has not
completed a turn yet", however many turns it has taken since.

Observed 2026-09-25 on three live threads: `ariadne` (codex), `ariadne-slot1`
and `ariadne-slot2`. In each, the ledger's current launch has no binding, and
no session watcher is running for it. `ariadne-slot1` (`couch-6ab16bffb21c607a`)
launched claude at 2026-09-24 15:00 with `--session-id 30512fb8…`. Its first
message came at 19:48, followed by hours of turns, and it's still unbound.

## Spec

Options (to decide at design time):

- **Resolve on demand (preferred).** When relaunch finds the current launch
  unbound, run the watcher's own check right then: a completed round after the
  launch boundary, matched to a Pair send and corroborated by the live agent
  process. For a live thread, all of that exists when the operator presses
  alt+n. The watcher becomes a way to have the answer ready sooner, not the
  only thing that can produce it.
- **Run the watcher inside the zellij session**, as `pair wrap` already does
  for a fresh conversation (`cmd/internal/wrapcmd/wrap.go:2443`). It then lives
  as long as the agent, not the client. Couch's failed-start cleanup needs a
  different mechanism.
- **Restart the watcher on attach when the current launch is unbound.** This
  must pass the ORIGINAL launch's start time as `--pid-not-before`. Otherwise
  the old agent-pid file reads as stale, the watcher finds no agent and quits
  after its 60 s startup deadline.

### Decision (2026-09-28): run the watcher from `pair wrap`, for every launch

`pair wrap` starts the agent, so it is the one process that holds everything the
watcher needs, exactly: the launch ordinal (`PAIR_LAUNCH_ORDINAL`), the scope
(`PAIR_SCOPE_KEY`), the tag, the agent's final argv, and the instant just before
the agent was spawned — a precise `--pid-not-before`. It already spawns the
watcher this way, in its own session, for an in-pane fresh conversation. So:

- wrap spawns `pair session-watch` for the current launch right after it writes
  the agent-pid file, in its own session (`Setsid`). This is the ONE spawn site.
- The launcher's `SpawnSessionWatcher` is removed, and so is the fresh path's
  separate `watcherArgv`: the exec'd replacement wrap spawns its own watcher
  from the new `PAIR_LAUNCH_ORDINAL`. Keeping either would spawn a duplicate.
- The watcher now lives with the agent, not with the pair client. Detach, a Couch
  restart, or reattach doesn't touch it.

Why not the other two options:
- *Resolve on demand* needs the agent argv and a per-launch pid boundary at
  relaunch time. The ledger's launch row records neither, and Couch would run
  native inventory + `lsof` synchronously inside alt+n. That makes a second
  binding writer whose only job is to repair the first one's lifetime.
- *Restart on attach* has the same gap: attach has no agent argv and no launch
  start time, so it would have to guess both.

Couch's failed-start cleanup needs no new mechanism. A wrap-spawned watcher is
outside the actor's process group, like the agent itself, and its end is
already named (ARCH-FUNERAL): after binding, Codex watchers continue observing
lifecycle events while other agents' watchers exit. A watcher also exits when
the agent's process identity changes, or at its 60 s startup deadline if no agent pid appears. A
failed start kills the zellij session, and with it the agent, so the watcher
exits on its next scan.

Limit: threads already stuck are running an old `wrap` binary. Nothing short
of restarting their agent repairs them. Threads launched after this change
are the "equivalents" in Done-when.


## Done when

- Composed regression for both Codex and Claude: launch, detach before the first
  turn, reattach, complete a turn, and authorize relaunch from the resulting
  binding. The regression fails under the original watcher ownership.
- The three stuck threads above (or their equivalents) become relaunchable
  after one turn, without manual repair.

## Plan

- [x] wrap: after writing the agent-pid file, spawn `session-watch` for the
      current launch (ordinal/scope/tag from env, agent argv, pid bound taken
      just before `pty.Start`), via `startWatcherProcess` (own session).
- [x] Drop `freshExecRequest.watcherArgv`; the exec'd wrap spawns its own.
- [x] Remove launcher `SpawnSessionWatcher` (interface, OSRuntime, fakes, create call).
- [x] Sweep comments that say the watcher shares the Couch actor's group
      (`couchcore/detach.go`, `couchcore/procops.go`, `launcher/osruntime.go`).
- [x] Tests: a wrap run with a launch ordinal spawns exactly one watcher with
      the right ordinal/scope/args, and a bound no later than the pid file's
      mtime. The real `startWatcherProcess` child gets its own session, so a
      group kill of the spawner misses it: this is the detach regression.
      The fresh path spawns no watcher of its own.
- [x] `TMPDIR=<scratchpad> make test`; ask the operator for a live smoke
      (detach before the first turn → reattach → one turn → alt+n).

## Revisions

- 2026-09-28: close review BR-1 requires a composed Codex/Claude regression
  through detach before the first turn, reattach, first completed turn, and
  relaunch authorization. Existing spawn/group tests prove mechanism only.
  BR-2 corrects watcher-lifetime prose: Codex keeps observing after binding.
  The acceptance criteria remain unchanged.

## Log

### 2026-09-25

- Found while diagnosing #328. The tools thread's refusal was #328, a
  different cause that leads to the same message.

### 2026-09-28

- Chose "run the watcher from `pair wrap`" over resolve-on-demand/restart-on-attach
  (see Spec › Decision). wrap is the only process holding the ordinal, the agent
  argv, and an exact pid bound; the ledger's launch row records neither argv nor
  a launch time (ARCH-DRY: one spawn site, fresh path included).
- Mutation-checked (cp/cmp revert): no spawn at agent start →
  `TestWrapSpawnsOneWatcherForItsLaunch` fails (0 spawns); dropping `Setsid` →
  `TestStartWatcherProcessEscapesTheSpawnersProcessGroup` fails (shares the
  spawner's group); bound taken after the pid write → the bound/mtime assertion fails.
- Found while testing: this agent shell inherits a live `PAIR_LAUNCH_ORDINAL`.
  Under `go test`, `os.Executable()` is the test binary, so an unstubbed wrap
  run would re-exec it as a "watcher". wrapcmd's `TestMain` now stubs
  `startWatcherProcess` package-wide.
- Suite (scrubbed env, scratchpad TMPDIR): `go test ./...` is green except
  `artifactpath TestProductionArtifactReferencesAreExactlyClassified`
  (`couchcmd/shortcut_focus.go` unclassified). `make test` stops at `test-lua`
  (`nvim/workbench_route_test.lua:40 <M-n>`). Both reproduce on a clean
  `git archive main`, introduced by #333 (`91863e53`), not by this branch.
  `wrapcmd` `TestNotificationBrokerBeforeExecAndCleanup` failed once under
  full-suite load, then passed 8/8 alone and in two full-package runs.
- Operator smoke on pair:3: started a new thread, completed several rounds,
  detached, reattached, and successfully relaunched it. This confirms the live
  detach/reattach/relaunch sequence after turns; detach before the first turn
  was not part of this operator run. The process-group and spawn-boundary
  regression tests cover the watcher ownership mechanism. The final plan row
  records the suite attempt (limitations above) and completed smoke, not a
  claim that the broad suite passed.
- Fresh watcher spawn, process-group isolation, and replacement-wrap handoff
  tests passed under `go test -race ./cmd/internal/wrapcmd -run
  'Test(WrapSpawnsOneWatcherForItsLaunch|StartWatcherProcessEscapesTheSpawnersProcessGroup|FreshAgentInvocationHandsTheWatcherToTheReplacementWrap)'
  -count=1`. Full wrapcmd and launcher package tests are running.
- Fresh full launcher package tests passed. Full wrapcmd tests hit the recorded
  `TestNotificationBrokerBeforeExecAndCleanup` startup-hook failure; eight
  isolated repeats passed. Close reviewer independently passed wrapcmd,
  launcher, and sessionwatch package suites. Close round 1 requested the
  composed acceptance regression (BR-1) and lifetime prose correction (BR-2).
- BR-1 addressed with `TestDetachBeforeFirstTurnAuthorizesRelaunch`: real wrap
  startup and watcher subprocess, isolated Codex/Claude native transcript
  fixtures, real Couch detach signaling a client process group, warm reattach
  eligibility and replacement client, then the first completed native turn.
  Before the turn, relaunch is unbound; afterward the real native resolver
  reads the watcher-written authorization proof and relaunch preconditions
  pass. The terminal server/attachment is represented by a stateful fake;
  this is composed authorization evidence, not a full interactive Zellij test.
  Three race-enabled repetitions passed. Mutating watcher ownership into the
  detached client's process group made both agents fail at post-reattach
  binding, and production was restored byte-for-byte.
- BR-2 addressed across wrap's spawn comment, atlas, and this issue: Codex
  continues lifecycle observation after binding; other agents' watchers exit.
