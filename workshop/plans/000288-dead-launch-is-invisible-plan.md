# Dead launch is invisible: implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A Pair create whose zellij server dies at birth ends the launcher
within a bounded time, with a clear error and no zellij client left behind,
instead of hanging on a blank pane while Couch keeps the thread `live`.

**Architecture:** While the blocking `LaunchSession` runs, a watch goroutine
waits for #287's birth evidence (the agent pane sidecar, cleared by the create
path) for a measured bound. If the evidence doesn't appear, it asks zellij
once whether the session is live. Only proof of death (no pane AND no live
session) cancels the launch's context. The OS runtime maps that cancel to
SIGTERM, then SIGKILL, on the zellij client, and reaps it. The launcher then
fails through the existing zellij-launch-failure path. The three pane-birth
waiters (title poller, Couch cold resume, and this new one) share one wait
loop, and the evidence path gets one declaration, both in a new `panebirth`
package.

**Tech Stack:** Go (`cmd/internal/panebirth` new, `launcher`, `titlepoller`,
`couchcore`), the `probes/zellijbirthrace` instrument, zellij 0.45.1.

---

## Measurements the design rests on (2026-09-18, macOS, zellij 0.45.1)

- **Birth time** (`probes/zellijbirthrace launch`, end to end from `pair
  resume`, so an upper bound on the time from `LaunchSession`):
  - idle, n=20: 0.61–1.27 s;
  - all 12 cores saturated (`yes` ×12, load 33), n=15: 1.49–2.92 s;
  - cold zellij cache (a fresh `HOME` per trial; `zellij setup --check`
    confirms its cache, data and plugin dirs follow `HOME`, as after an
    upgrade), n=4: 0.44–0.52 s. Plugin compilation doesn't delay the agent
    pane's sidecar.
- **How a birth dies** (bare zellij, socket connect-and-close from a tight
  thread loop):
  - **A (most):** the client gets "Lost connection", restores the terminal
    and exits **1** by itself. Pair already fails there, with `LaunchSession`
    returning 1, so it is not this issue.
  - **B (about 1 in 5 of the deaths):** the client stays alive and writes
    **0 bytes**, so the terminal is untouched and still cooked. zellij lists
    no session. It exits 50 ms after SIGTERM (status 15). This is the hang:
    the astro thread sat in it for 2 h.
- **Couch** (Explore digest, cited in the issue Log):
  - A helper that exits is noticed as soon as its pty closes, and the thread
    then classifies as `parked` or `session-gone`. `ArchiveThread` accepts
    both and retires the stale `live` incarnation. No new Couch state is
    needed.
  - A cold create is `established` before zellij starts, so Couch's 15 s
    registration deadline never covers it. That is why the hang was
    unbounded.
  - Couch shows only `<label> [<actor>] exited (N)`, not the launcher's
    stderr.
  - The launcher must reap its zellij child before it exits. Otherwise the pty
    stays open and the helper can linger as a zombie that reads as alive.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `panebirth.Await` | `cmd/internal/panebirth/panebirth.go` | new |
| `panebirth.Evidence` (was `titlepoller.BirthEvidence`) | `cmd/internal/panebirth/panebirth.go` | new (moved) |
| `birthVerdict` + `judgeUnborn` | `cmd/internal/launcher/birthwatch.go` | new |
| `titlepoller.awaitPaneBirth` | `cmd/internal/titlepoller/run.go` | deleted |
| `couchcore.(*Couch).awaitPaneBirth` loop | `cmd/internal/couchcore/launch_existing.go` | modified |

- **`panebirth.Await(ctx, clock, poll, grace, born) error`.** It polls
  `born()` every `poll` until birth (returns nil), until `grace` has passed on
  `clock`, or until `ctx` ends. It returns `ErrUnborn` joined with the cause
  and the last observation error. A failed observation is "not yet", never
  birth, and it doesn't end the wait (that is #287's rule, now stated once).
  - **The cause:** `ctx.Err()` when the context ended. When the grace
    passed, it is `context.DeadlineExceeded`, because the grace *is* a
    deadline. So a caller whose context and grace race, as Couch's do, always
    sees `DeadlineExceeded`, and `diagnoseRegistrationFailure` keeps its
    #215 diagnosis.
  - **`Clock`:** `Now()` plus `Sleep(ctx, d)`. The sleep returns early when
    `ctx` ends, so a stopped wait returns at once and not up to one poll
    later. The title poller passes `pollerClock{rt}`, which adapts its
    fake-able `Runtime` and ignores `ctx`. Couch and the launcher pass
    `panebirth.WallClock{}`.
  - **Call order:** observe, then ctx, then deadline, then sleep, with one
    `Now()` at entry and one per round. That is the title poller's existing
    order, so its fake-clock tests keep their meaning.
  - **DRY rationale:** It is the third waiter on pane birth. #287's close
    review (ARCH-DRY) named this as the moment to extract.
  - **Relationships:** Evidence is 1:1 with (dataDir, tag, agent). The create
    path clears the file, and the title poller and the launcher await it.
    Couch awaits a different observation (a glob against `PaneMarks`) through
    the same loop.
- **`birthVerdict` / `judgeUnborn(sessions, err, session)`.** This is what the
  launcher concludes once the bound has passed with no pane:
  - `err != nil` → `birthUnknown`. A failed probe is not proof of death.
  - the session is absent, or `SessionExited` → `birthDead`;
  - otherwise (live, attached, or detached) → `birthAlive`.

  zellij's own empty inventory ("No active zellij sessions found.") already
  maps to `(nil, nil)` in `ZellijSource.runContext`, which makes it absent.
  The other verdicts are `birthBorn`, and `birthStopped`, meaning the client
  exited before any verdict.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `ZellijOps.LaunchSession(ctx, …)` | `cmd/internal/launcher/runtime.go`, `osruntime.go` | modified | `exec.CommandContext` + SIGTERM/SIGKILL |
| `watchBirth` | `cmd/internal/launcher/birthwatch.go` | new | `Runtime.FileSize`, `Runtime.SessionLiveness` |
| create-path watch wiring | `cmd/internal/launcher/createflow.go` | modified | goroutine around `LaunchSession` |
| `fakeRuntime.LaunchSession` | `cmd/internal/launcher/createflow_test.go` | modified | stateful fake (pane command + hung client) |
| `probes/zellijbirthrace launch` exit watch | `probes/zellijbirthrace/main.go` | modified | live conformance |

- **`LaunchSession(ctx, session, configDir, layout)`.** Cancelling `ctx`
  ends the client. `cmd.Cancel` sends SIGTERM, and `cmd.WaitDelay`
  (`clientKillGrace` = 2 s) escalates to SIGKILL. `Run` returns only once
  the child has been reaped, which prevents the zombie and pty-hold from the
  Couch digest. Attach is untouched.
- **`watchBirth(ctx, rt, evidence, session, bound, abort)`.** This is the IO
  shell. It loops over `panebirth.Await(bound)`:
  - born → `birthBorn`;
  - `ctx` done (the client exited first) → `birthStopped`;
  - otherwise `judgeUnborn(rt.SessionLiveness())`:
    - dead → call `abort()` and return;
    - alive → stand down;
    - unknown → another round: wait up to `bound` again, then ask again.
- **Fake.** `LaunchSession` models zellij:
  - `launchWrites` are files the pane command writes as the session starts;
    setting the sidecar there is a birth.
  - `launchBlock` makes a hung client that returns only when `ctx` is
    cancelled (recorded in `launchCancelled`) or `launchRelease` is closed.
    A 5 s safety return fails the test instead of hanging it.
  - `livenessScript` scripts successive probe answers.
  - `livenessAfterLaunch` signals each probe made during a launch. The send
    is non-blocking, so a full buffer never stalls the watch.

  Everything the watch goroutine reads goes through `f.mu`.

## Ordering (ARCH-ORDER)

**Extent.** The watch starts immediately before `LaunchSession`. It is
cancelled and joined right after `LaunchSession` returns, on every path, so
nothing outlives `runCreate`. `abort` cancels the launch context. It is
idempotent, and once the child has been reaped it does nothing.

**Events, and what governs each:**
- **Pane written before the bound** → born, and the watch ends (ignore).
- **The client exits on its own first** (death A, a zellij error, a normal
  quit) → the watch is cancelled (`birthStopped`) and the existing result
  path runs unchanged.
- **The bound passes and the probe says live** → stand down. This covers the
  case where the pane sidecar failed to write but the session is healthy.
  Tearing that session down later would skip `runCleanup`'s quit-marker
  handling, so once zellij has listed the session live, the watch never
  acts.
- **The probe fails** → unknown, then another bound, then ask again. No
  teardown without an answer.
- **Dead verdict** → preempt: abort, then fail through the existing
  launch-failure path (`restoreLayoutRecord`, message, `launchStep{code: 1}`,
  `handedOff: false`), so there is no cleanup and no restart re-entry.
- **Dead verdict racing the client's own exit.** The failure path runs only
  when the cause matches the verdict. Two guards do this:
  - `watchBirth` rechecks `ctx` after the probe and returns `birthStopped` if
    the client has already exited, so it never aborts a finished launch.
  - `runCreate` takes the dead path only on `birthDead && code != 0`. A
    killed client returns -1. A session that came up without its sidecar and
    was quit cleanly while the probe found nothing returns 0, so it takes the
    normal handoff path and `runCleanup` still runs.

**The event most likely to be mishandled:** a slow but healthy birth whose
server hasn't bound its socket by the bound. zellij then lists nothing, and
the watch would kill a launch that was about to succeed. The bound is 10 s.
That is 3.4× the worst birth measured under full CPU saturation, and still
under Couch's 15 s registration deadline. The Go monotonic clock doesn't
advance during system sleep on macOS, so a lid closed mid-launch doesn't
count against the bound. This is a recorded residual.

**The other half of that residual.** The watch's own probe is a
`list-sessions`, which is itself a #287 birth-window connection. If a birth
is still in its window at 10 s, the probe can kill it. Because zellij
accepted the probe's connection, it may also list the session live, and the
watch then stands down while the client hangs. Both halves need a birth
slower than 10 s, 3.4× the loaded worst case. Recorded, not engineered
around.

**Residual: other threads' births.** `SessionLiveness` runs two
machine-wide `list-sessions`, so the probe can also land in *another*
thread's concurrent birth window. That is #287's external-prober residual,
and this watch adds to it only rarely: at most one probe per launch that has
gone 10 s with no pane, which never happens on a healthy launch.

**Residual: the join.** If the client exits while a probe is in flight, the
join waits for that probe to finish (at most two 5 s `zellijQueryTimeout`
calls). It is rare and bounded. A `SessionLiveness(ctx)` would remove it,
but that is a new seam method for a rare case.

**Residual: the terminal.** If the 2 s SIGKILL escalation is ever needed,
the tty is not restored. Death B was measured untouched (0 bytes written),
and it exits on SIGTERM.

**Nondeterminism and reproduction.** Nondeterminism enters through the clock
(the bound), the order in which the client exit and the verdict complete, and
the probe's answers. The fake makes each one explicit:
- `launchBlock`, `launchRelease` and `launchCancelled` control the client;
- `livenessScript` controls the probe;
- `BirthBound` shrinks the bound.

A test that needs the probe to have happened before the client exits waits on
`livenessAfterLaunch`, a channel the fake signals, instead of sleeping.

## Lifecycle (ARCH-FUNERAL)

Nothing durable is created.
- The watch goroutine is bounded by `runCreate`.
- The killed client is reaped by `exec.Cmd.Wait`.
- The dead server's stale socket is removed by zellij's own `list-sessions`,
  and our probe runs one.
- The title poller exits after its 30 s `StartupGrace`, and the session
  watcher exits at its own deadline.
- The layout record is restored.

## Out of scope

- **Couch showing the launcher's reason.** The exit notice shows only
  `exited (1)`. The zellij log hint is visible only to a standalone launch.
  A follow-up could surface the child's final stderr line.
- **Couch's cold-resume wait not watching the helper.** It waits out its 15 s
  deadline even after the launcher has exited at about 10 s, then rolls back
  to `parked` as it does today.
- **Death A's message.** Pair already exits 1 there. Adding a "never came
  up" message would need care, because a missing pane sidecar alone must
  not turn a normal quit into a failure.
- **A server that dies *after* its pane was born.** The watch has already
  returned `birthBorn`, so it doesn't act, and the operator would see the
  same hung pane. #287's mechanism kills births, not born sessions, so this
  isn't the observed failure. The probe counts it separately
  (`hung-after-birth`). If the live run shows a count above 0, file a
  follow-up issue with the trial notes.
- **Conformance cadence.** The fake's model of death B is zellij 0.45.1's
  behaviour: the client hangs, writes 0 bytes, isn't listed, and exits on
  SIGTERM. Re-run `zellijbirthrace launch -hammer 10ms` on every zellij
  version change. The probe's SKILL.md and the atlas paragraph both say so.

## Chunk 1: shared pane-birth wait

### Task 1: `panebirth` package

**Files:**
- Create: `cmd/internal/panebirth/panebirth.go`, `cmd/internal/panebirth/panebirth_test.go`

- [x] **Step 1: failing tests** (`panebirth_test.go`, a fake clock whose
  `Sleep(ctx, d)` advances `now` by `d` and counts calls, and whose `Now()`
  does not advance on its own):
  - born on the first observation → nil, zero sleeps;
  - born on round 3 → nil, 2 sleeps;
  - never born, grace passes on the clock (grace = 5 × poll) →
    `errors.Is(err, ErrUnborn)` and `errors.Is(err,
    context.DeadlineExceeded)`, 5 sleeps;
  - cancelled ctx → `ErrUnborn` and `context.Canceled`, with no further
    observation after the cancel is seen;
  - `WallClock.Sleep(ctx, time.Hour)` with a cancelled ctx returns in under
    100 ms;
  - an observation error followed by birth → nil, so a failed stat never ends
    the wait. Errors only, then grace passes → the error is in the chain.
  - `Evidence` returns exactly `artifactpath.ResolveScoped(d, t).PaneChecked(a)`,
    and an invalid tag errors.
- [x] **Step 2:** `go test ./cmd/internal/panebirth/` → FAIL (undefined).
- [x] **Step 3: implement.**

```go
// Package panebirth is Pair's evidence that a zellij session has been born
// (#287), and the one wait on it (#288). The layout's pane command writes the
// agent pane sidecar as its first act, and a pane exists only once the first
// client has initialized the session -- the window in which zellij 0.45.1
// panics when a connection it accepted closes. A waiter that asks zellij
// nothing until this evidence appears can't open that window.
package panebirth

// Evidence is the file a create's birth is judged by: this agent's pane
// sidecar. It is the ONE declaration of that path. The launcher's create path
// clears exactly this file before it spawns anything; if a waiter named a
// different file, the clear would miss and a stale sidecar would pass at once.
func Evidence(dataDir, tag, agent string) (string, error) {
	paths, err := artifactpath.ResolveScoped(dataDir, tag)
	if err != nil {
		return "", err
	}
	return paths.PaneChecked(agent)
}

// ErrUnborn is a wait that ended without birth.
var ErrUnborn = errors.New("agent pane not born")

// Clock is a wait's time: Now bounds it, Sleep paces it. Sleep returns early
// when ctx ends; a clock that cannot (a test's) may ignore ctx.
type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration)
}

// WallClock is the real clock, for waiters without a clock seam of their own.
type WallClock struct{}

func (WallClock) Now() time.Time { return time.Now() }
func (WallClock) Sleep(ctx context.Context, d time.Duration) {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}

// Await polls born every poll until it reports birth (nil), grace has passed
// on clock, or ctx is done. A failed observation is "not yet", never birth,
// and does not end the wait: a transient stat error after a good create must
// not read as a dead one. The last such error rides the returned one, for the
// diagnosis. A grace that passes is reported as context.DeadlineExceeded --
// it is a deadline -- so a caller whose context and grace race sees the same
// cause whichever fires first.
func Await(ctx context.Context, clock Clock, poll, grace time.Duration, born func() (bool, error)) error {
	deadline := clock.Now().Add(grace)
	var lastErr error
	for {
		ok, err := born()
		if err == nil && ok {
			return nil
		}
		if err != nil {
			lastErr = err
		}
		if err := ctx.Err(); err != nil {
			return errors.Join(ErrUnborn, err, lastErr)
		}
		if !clock.Now().Before(deadline) {
			return errors.Join(ErrUnborn, context.DeadlineExceeded, lastErr)
		}
		clock.Sleep(ctx, poll)
	}
}
```

- [x] **Step 4:** `go test ./cmd/internal/panebirth/` → PASS.

### Task 2: move the title poller and Couch onto it

**Files:**
- Modify: `cmd/internal/titlepoller/run.go:105-200` (gate call; delete
  `BirthEvidence` and `awaitPaneBirth`, keep `paneBirthPoll`)
- Modify: `cmd/internal/couchcore/launch_existing.go:376-401` (`awaitPaneBirth` body)
- Modify: `cmd/internal/launcher/createflow.go:776`, `cmd/internal/launcher/pane_birth_test.go:28` (`titlepoller.BirthEvidence` → `panebirth.Evidence`)

- [x] **Step 1:** Title poller gate:

```go
	panePath, err := panebirth.Evidence(opts.DataDir, opts.Tag, opts.Agent)
	if err != nil || panebirth.Await(context.Background(), pollerClock{rt}, paneBirthPoll, opts.StartupGrace, func() (bool, error) {
		_, ok := rt.ModTime(panePath)
		return ok, nil
	}) != nil {
		return 0
	}
```

```go
// pollerClock is the poller's Runtime as a panebirth.Clock. Its Sleep is the
// fake-able one; nothing cancels the poller's wait, so ctx is not needed.
type pollerClock struct{ rt Runtime }

func (c pollerClock) Now() time.Time                         { return c.rt.Now() }
func (c pollerClock) Sleep(_ context.Context, d time.Duration) { c.rt.Sleep(d) }
```

  Keep the #287 comment block and repoint "the create path clears it" at
  `panebirth.Evidence`.
- [x] **Step 2:** Couch: the loop body becomes one `Await` over
  `observePaneSidecars` + `baseline.BornIn`, `panebirth.WallClock{}`, poll
  10 ms, and a grace of `time.Until(deadline)` from `ctx.Deadline()`. The
  registration context always has one; without one it returns an error. It
  returns the `Await` error directly. That error carries
  `context.DeadlineExceeded` whether the context or the grace fired first, so
  `diagnoseRegistrationFailure` (`launch_existing.go:274`) keeps its #215
  diagnosis.
  - Keep the doc comment's "a failed observation is not yet" paragraph, and
    cite `panebirth.Await` as the owner of that rule.
  - Strengthen `TestColdResumeTimesOutWhenThePaneIsNeverBorn` to assert the
    diagnosis suffix (`waited`), which it doesn't check today, so that losing
    `DeadlineExceeded` fails a test.
- [x] **Step 3:** `go test ./cmd/internal/titlepoller/ ./cmd/internal/couchcore/ ./cmd/internal/launcher/ ./cmd/internal/artifactpath/`
  → PASS. Expect the artifactpath manifest and the couchcore plan-contract
  inventory gates to ask for the new files to be classified. Follow their
  messages, as #287 did for `couchcore/panebirth.go`.
- [x] **Step 4:** Commit `#288: panebirth: one pane-birth wait for the poller, Couch and (next) the launcher`.

## Chunk 2: the launcher's birth watch

### Task 3: cancellable `LaunchSession` + stateful fake

**Files:**
- Modify: `cmd/internal/launcher/runtime.go:40-44` (signature and doc)
- Modify: `cmd/internal/launcher/osruntime.go:140-147`
- Modify: `cmd/internal/launcher/createflow.go:791` (pass `context.Background()` for now; Task 4 wires the watch)
- Modify: `cmd/internal/launcher/createflow_test.go:213-223` (fake), `retention_test.go:49-52`
- Test: `cmd/internal/launcher/osruntime_launch_test.go` (new)

- [x] **Step 1: failing test (OS side).** Factor the command construction as
  `cancellableHandoff(ctx, name string, args ...string) *exec.Cmd` so it can
  be driven with `sh`. Both scripts must `exec` the sleeper. macOS `/bin/sh`
  otherwise forks it, and the orphan keeps the test binary's stdout open,
  so `go test` waits out the whole sleep.
  - `sh -c 'exec sleep 60'` with a cancel after 50 ms → `runBlockingHandoff`
    returns `(-1, nil)` within 1 s, and the pid is gone (`syscall.Kill(pid,
    0)` is `ESRCH`).
  - `sh -c 'trap "" TERM; exec sleep 60'` (an ignored signal stays ignored
    across `exec`), with `cmd.WaitDelay` set to 200 ms in the test → returns
    within 1.5 s, having been killed.
- [x] **Step 2:** Run it: FAIL (undefined).
- [x] **Step 3: implement.**

```go
// clientKillGrace is how long a cancelled client gets to exit on SIGTERM
// before SIGKILL. A zellij client hung on a dead-at-birth server exits in
// about 50 ms on SIGTERM (measured, #288).
const clientKillGrace = 2 * time.Second

// cancellableHandoff is a blocking handoff that ctx can end: SIGTERM, then
// SIGKILL after clientKillGrace. Run returns only once the child is reaped,
// so a cancelled launch leaves no process holding the terminal.
func cancellableHandoff(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = clientKillGrace
	return cmd
}

func (OSRuntime) LaunchSession(ctx context.Context, session, configDir, layout string) (int, error) {
	return runBlockingHandoff(cancellableHandoff(ctx, "zellij",
		"--config-dir", configDir,
		"--new-session-with-layout", layout,
		"--session", session))
}
```

  Runtime doc: "Cancelling ctx ends the client (SIGTERM, then SIGKILL) and
  still returns only after it is reaped. Only the create path's birth watch
  cancels it (#288)."
- [x] **Step 4: fake.** Add these fields, all guarded by `f.mu` where the
  watch goroutine reads them:
  - `launchWrites map[string]string`, applied under `f.mu` at the start of
    `LaunchSession`: the pane command's writes, i.e. birth;
  - `launchBlock bool` and `launchRelease chan struct{}`;
  - `launchCancelled bool`;
  - `livenessScript []livenessAnswer` (`{sessions []Session; err error}`), one
    answer per call, with the last repeating. It overrides `f.sessions` only
    while `LaunchSession` is running;
  - `livenessAfterLaunch chan struct{}` (buffered), signalled per probe made
    during a launch.

  When blocking, `LaunchSession` selects on `ctx.Done()` (sets
  `launchCancelled` and returns `-1, nil`, like a SIGTERMed client),
  `launchRelease` (returns `launchCode`), and `time.After(5 * time.Second)`
  (returns `99, errors.New("fake: launch never released")` so a broken watch
  fails loudly).
- [x] **Step 5:** `go test -race ./cmd/internal/launcher/` → PASS, with no
  behaviour change. There is no watch yet, and existing creates don't block.
- [x] **Step 6:** Commit `#288: launcher: LaunchSession takes a context that ends a hung client`.

### Task 4: the watch

**Files:**
- Create: `cmd/internal/launcher/birthwatch.go`, `cmd/internal/launcher/birthwatch_test.go`
- Modify: `cmd/internal/launcher/createflow.go:788-799`, `runtime.go` (`LaunchOptions.BirthBound`)
- Test: `cmd/internal/launcher/pane_birth_test.go` (end-to-end cases)

- [x] **Step 1: failing pure test** (`birthwatch_test.go`, a table):

| sessions / err | want |
|---|---|
| `err` | `birthUnknown` |
| `nil, nil` (zellij's empty inventory) | `birthDead` |
| other sessions only | `birthDead` |
| ours `SessionExited` | `birthDead` |
| ours `SessionLive` / `Attached` / `Detached` | `birthAlive` |

- [x] **Step 2: failing end-to-end tests** (`pane_birth_test.go`, fake
  runtime, `opts.BirthBound = 30 * time.Millisecond`):
  1. **Dead birth, the hang:** `launchBlock`, no pane, and liveness `[]` →
     code 1; stderr contains `never came up` and the session name;
     `launchCancelled`; the layout record is restored (compare it with a
     prior record the test seeds); no quit/restart marker is read, since the
     path has `handedOff: false`; and the retention finishes as a failure,
     the way `retention_test` asserts the LaunchSession-error path.
  2. **Born:** `launchWrites` holds the evidence, and `launchBlock` is
     released after 10 bounds → code 0, not cancelled, and no liveness probe
     during the launch.
  3. **Alive without a pane:** liveness lists ours live. The test waits on
     `livenessAfterLaunch`, waits 5 bounds more, then releases → code 0, not
     cancelled, exactly 1 probe. Once zellij has listed the session live, the
     watch never acts.
  4. **Unknown, then dead:** the liveness script is `[err, err, []]` →
     cancelled, and the fake records the probe count at the moment of the
     cancel. It must be 3, so the cancel came after the answer and not
     before.
  5. **The client exits first:** no block, `BirthBound` left at the 10 s
     default → `RunLaunch` returns in under 1 s, with no probe during the
     launch. The watch ended with the client.
  6. **Clean quit while the probe finds nothing:** `launchBlock`, no pane.
     The liveness hook releases the launch with `launchCode` 0, waits until
     `LaunchSession` has returned, then answers `[]` → code 0, no "never came
     up", and the normal handoff path (`handedOff`, cleanup reached). The
     outcome holds under every interleaving of `stopWatch` and the verdict.
- [x] **Step 3:** Run them → FAIL.
- [x] **Step 4: implement `birthwatch.go`.**

```go
// birthBound is how long a create waits for its agent pane before asking
// zellij whether the session is alive (#288). Measured with
// probes/zellijbirthrace, launch to pane, end to end from `pair resume`:
// 0.6-1.3 s idle, 1.5-2.9 s with every core saturated. 10 s is 3.4x the
// loaded worst case and still under Couch's 15 s registration deadline.
const birthBound = 10 * time.Second

// birthPoll is the watch's stat cadence: one stat per tick. A client that
// exits first ends the wait at once (the sleep watches ctx), not a tick late.
const birthPoll = 100 * time.Millisecond

type birthVerdict int

const (
	birthStopped birthVerdict = iota // the client exited before a verdict
	birthBorn                        // this launch's pane wrote its sidecar
	birthAlive                       // no pane, but zellij lists the session live: not ours to end
	birthUnknown                     // no pane, and zellij could not be asked: no proof of death
	birthDead                        // no pane, and zellij lists no live session
)

// judgeUnborn is the verdict on a create whose pane did not appear within
// the bound, from one liveness snapshot.
func judgeUnborn(sessions []Session, err error, session string) birthVerdict {
	if err != nil {
		return birthUnknown // a failed observation is not an absence
	}
	for _, s := range sessions {
		if s.Name == session {
			if s.State == SessionExited {
				return birthDead
			}
			return birthAlive
		}
	}
	return birthDead
}

// watchBirth waits for the create's birth evidence while LaunchSession blocks.
// Only proof of death -- no pane AND no live session -- calls abort. It never
// acts on a session zellij has listed live: that session may be healthy with
// a missing sidecar, and ending it would skip the quit cleanup. An unanswered
// probe earns another bound and another question, never a teardown.
func watchBirth(ctx context.Context, rt Runtime, evidence, session string, bound time.Duration, abort func()) birthVerdict {
	for {
		err := panebirth.Await(ctx, panebirth.WallClock{}, birthPoll, bound, func() (bool, error) {
			_, ok := rt.FileSize(evidence)
			return ok, nil
		})
		switch {
		case err == nil:
			return birthBorn
		case ctx.Err() != nil:
			return birthStopped
		}
		sessions, probeErr := rt.SessionLiveness()
		if ctx.Err() != nil {
			return birthStopped // the client ended while we asked: not ours to judge
		}
		switch v := judgeUnborn(sessions, probeErr, session); v {
		case birthDead:
			abort()
			return v
		case birthAlive:
			return v
		}
	}
}
```

- [x] **Step 5: wire into `runCreate`** (it replaces the bare `LaunchSession`
  call):

```go
	launchCtx, abort := context.WithCancel(context.Background())
	defer abort()
	watchCtx, stopWatch := context.WithCancel(context.Background())
	bound := opts.birthBound()
	verdict := make(chan birthVerdict, 1)
	go func() { verdict <- watchBirth(watchCtx, rt, birthEvidence, session, bound, abort) }()
	code, err := rt.LaunchSession(launchCtx, session, configDir, layout)
	stopWatch()
	// Dead only when the cause matches: a killed client returns -1, while a
	// clean quit that raced the probe returns 0 and keeps the normal path.
	if <-verdict == birthDead && code != 0 {
		restoreLayoutRecord(rt, dataDir, chosenTag, priorLayout)
		fmt.Fprintf(stderr, "pair: zellij session '%s' never came up: no agent pane after %s, and zellij lists no live session.\n", session, bound)
		fmt.Fprintf(stderr, "      Its server died while starting; zellij's log is %s\n", zellijLogPath())
		return launchStep{code: 1}, nil
	}
```

  `zellijLogPath()` is `filepath.Join(os.TempDir(), fmt.Sprintf("zellij-%d",
  os.Getuid()), "zellij-log", "zellij.log")`, zellij's default log location.
  It is a pointer in a message, not something Pair reads. Add
  `LaunchOptions.BirthBound time.Duration` (0 means `birthBound`) with a
  `birthBound()` accessor.
- [x] **Step 6:** `go test -race -count=1 ./cmd/internal/launcher/ ./cmd/internal/panebirth/` → PASS.
- [x] **Step 7: mutation checks** (revert each after it runs):
  - `abort()` removed → test 1 fails, via the fake's 5 s safety return and
    `launchCancelled` false;
  - `birthAlive` treated as `birthDead` → test 3 fails;
  - `birthUnknown` treated as `birthDead` → test 4 fails;
  - `stopWatch()` removed → test 5 fails its 1 s limit, because the join then
    waits out the 10 s bound;
  - both cause guards removed (the post-probe `ctx` check and `code != 0`) →
    test 6 fails.
- [x] **Step 8:** Commit `#288: launcher: a create whose pane never comes and whose session is gone fails instead of hanging`.

## Chunk 3: live conformance and docs

### Task 5: probe reports whether Pair notices a death

**Files:** Modify `probes/zellijbirthrace/main.go` (`launchTrial`, `launchFlags`, the summary line), `probes/zellijbirthrace/SKILL.md`

- [x] **Step 1:** Add the flag `-exit-wait DUR` (default 20 s). On a `died`
  verdict, before teardown, wait up to that long for the launcher to exit:
  - it exits → note ` launcher exited after X`, then check the process table
    (`ps -axo command=`) for a line containing both
    `--new-session-with-layout` and the trial's ASCII tag. Match on the tag,
    not the 📁 session name, because `ps` may escape non-ASCII. A match
    means ` zellij client left behind`.
  - otherwise → note ` launcher still running after DUR (hung)`.

  Count hangs by when the death came:
  - `hung=` for deaths *before* the pane was born, which #288 targets;
  - `hung-after-birth=` for deaths after it. There the watch has already
    returned `birthBorn` and doesn't act; that is out of scope, and the
    count is only reported.

  `PROBE-RESULT` gains both. Keep the `born`, `died` and `inconclusive`
  counts as they are.
- [x] **Step 2:** `go vet ./probes/zellijbirthrace/`, and `make test-smoke`
  (the bare run still only describes itself).
- [x] **Step 3: live, sandbox off.** Build the pre-fix `pair` (main) and the
  fixed one into `<scratch>/{base,fix}/pair`, then run `launch -n 20
  -hammer 10ms` against each.
  - Expect base: `hung` ≥ 1 (death B).
  - Expect fix: `hung=0` and no client left behind. Each before-birth death
    ends one of two ways:
    - death A: the client exits by itself, so the watch stops and the
      launcher exits in about 1–2 s;
    - death B: the launcher exits in about 10–11 s.

  If base shows no hang in 20 trials, rerun with a larger `-n`. The verdict
  needs a window that could have seen a hang (lessons.md, #262).
- [x] **Step 4:** Record both runs in SKILL.md's Results table and in the
  issue Log.

### Task 6: docs + operator smoke

- [x] `atlas/architecture.md`:
  - retarget the birth-gate paragraph to `panebirth`;
  - add a "Launcher birth watch (`pair#288`)" paragraph: the bound, the
    verdicts, teardown only on proof, and the Couch consequence.
- [x] `atlas/couch.md`:
  - note that a dead-at-birth cold create now ends its helper at about 10 s
    and the thread reads `session-gone`, which is archivable;
  - and that the cold-resume wait still runs to its 15 s deadline.
- [x] Check `atlas/index.md` for new files: `panebirth` needs no new atlas
  file.
- [x] `workshop/lessons.md`: add a rule if the work surfaced one.
- [x] Full suite: `TMPDIR=<scratch> make test` (memory: the test-changelog
  TMPDIR quirk), then `go test ./... -count=1`.
- [ ] `make install`, then ask the operator to smoke-test Couch:
  1. start `while :; do zellij list-sessions --short >/dev/null 2>&1; sleep 0.01; done`
     in a spare terminal;
  2. cold-create a scratch thread in Couch (retry until one launch dies;
     death B is about 1 in 5 of the deaths);
  3. check that its pane exits within about 10 s with `exited (1)`, and that
     the thread archives without restarting Couch;
  4. stop the hammer.

## Revisions

- **2026-09-18, plan review round 1 (fresh-context reviewer).** The code
  claims were verified. These changes were folded in:
  - `Clock.Sleep` takes `ctx`, so a stopped wait returns at once and not a
    poll late. The title poller adapts through `pollerClock`.
  - A grace that passes is reported as `context.DeadlineExceeded`. Couch's
    context and grace race, and this way Couch keeps its #215 diagnosis;
    its timeout test now asserts that diagnosis.
  - Task 3 passes `context.Background()` at the call site, so it compiles
    on its own.
  - The OS kill test `exec`s its sleeper. A forked orphan would keep
    `go test`'s stdout open.
  - The fake's probe signal is non-blocking. Test 4 records the probe count
    at the cancel. A `stopWatch()` mutation check was added.
  - The probe splits `hung` from `hung-after-birth`, sets its expectations
    per death mode, and matches the client on the ASCII tag.
  - Residuals recorded: the watch's own probe inside a birth still in its
    window after 10 s; the join waiting on an in-flight probe; the tty after
    a SIGKILL escalation.
- **2026-09-18, change-code plan-quality (6 Minor findings, advisory).**
  These were folded in:
  - the verdict must match its cause: a post-probe `ctx` recheck plus
    `code != 0`, with test 6 and a mutation check;
  - the residual of other threads' births;
  - the cold-cache measurement: 0.44–0.52 s, so the bound stands;
  - the conformance cadence: re-run on each zellij version change;
  - `hung-after-birth` stated in Out of scope, with its follow-up rule.

  Not folded: "the plan restates tests and code". The code blocks stay,
  because they are what the reviewed design is.
