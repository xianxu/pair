# New-session startup race implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** No process in a Pair launch path opens a zellij connection to a session
it is waiting on until that session has initialized, so zellij 0.45.1's
`RemoveClient` unwrap can no longer kill a new session at birth.

**Architecture:** The agent pane's sidecar (`pane-<tag>-<agent>.json`, written by
the layout's pane command as its first act) is the birth evidence. A pane exists
only after the first client has initialized the session. The launcher's create
path clears this agent's sidecar before it spawns anything, so for the title
poller, "the file exists" means "this launch's pane ran". Couch starts the
launcher and can't see that clear, so it snapshots the tag's sidecars before
releasing the blocked helper. It then waits for the snapshot to change.

**Tech Stack:** Go (`cmd/internal/titlepoller`, `launcher`, `couchcore`), a Go
probe under `probes/`, zellij 0.45.1.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `titlepoller.Run` startup gate | `cmd/internal/titlepoller/run.go` | modified |
| `PaneMarks` / `PaneMarks.BornIn` | `cmd/internal/couchcore/panebirth.go` | new |

- **Title poller startup gate.** It replaces the `SessionAlive` startup loop.
  The poller polls `ModTime(paths.Pane(opts.Agent))` every 250 ms for up to
  `StartupGrace`. It makes no zellij call until the file exists, and if the
  grace runs out it exits 0 without having made one. After the gate it enters
  the unchanged steady loop (60 s `SessionAlive`, frame and cmux titles). The
  same rule covers both spawn sites. On attach the live pane wrote the file
  long ago, so the gate passes on the first stat.
  - **Relationships:** The gate runs once per poller process, on one file.
  - **DRY rationale:** It reuses `artifactpath.Paths.PaneChecked` and the
    existing `Runtime.ModTime` seam. No new runtime method.
- **PaneMarks.** A snapshot of the tag's pane sidecars, `map[path]mtime`.
  `before.BornIn(now)` is true iff some path in `now` is missing from `before`
  or has a different mtime. The comparison uses mtime *equality* only, never
  ordering, so it doesn't depend on the clock. A sidecar the launcher removed
  reads as absent, which isn't birth, so Couch keeps waiting through that
  clear. A stale twin from another agent is unchanged, so it can't fake a
  birth.
  - **Relationships:** Couch holds one per cold resume, and only for the
    length of that start.
  - **Future extensions:** The out-of-scope follow-up ("a dead launch is
    invisible") can use the same evidence to fail a launch whose pane never
    appears.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| launcher create-path clear | `cmd/internal/launcher/createflow.go` | modified | `Runtime.Remove` |
| `PaneBirthIO.PaneSidecars` | `cmd/internal/couchcore/artifactcollision.go` | new | glob + stat of `PaneGlob()` |
| `FakeThreadArtifactCollisionChecker` pane model | `cmd/internal/couchcore/artifactcollision_fake.go` | modified | stateful fake |
| `awaitResumeRegistration` birth wait | `cmd/internal/couchcore/launch_existing.go` | modified | — |
| `zellijbirthrace` probe | `probes/zellijbirthrace/main.go` | new | zellij, `pair`, unix socket |

- **Launcher clear.** `rt.Remove(artifactPaths.Pane(agent))` goes immediately
  before `SpawnSessionWatcher`, `SpawnTitlePoller` and `LaunchSession`. It
  follows the pattern `prepareLaunchReadiness` already uses: remove the old
  evidence, then launch. Only the create path clears. Attach leaves the live
  pane's file alone.
- **PaneSidecars.** Implemented on `ScopedThreadArtifactCollisionChecker`. It
  resolves the thread's scoped dir the way `PairSessionContext` does, globs
  `PaneGlob()` and stats each match. A glob with no matches returns an empty
  map, not an error. A file that vanishes between the glob and the stat is
  skipped.
  - **Injected into:** `Couch` through `c.Artifacts`, type-asserted like
    `PairSessionIO`.
- **Fake pane model (ARCH-MOCK).** `SetPairSession(addr, name, true)` moves
  that address's sidecar to a new mtime when it goes from not-present to
  present. The rule it models is "a session that has become live has had its
  pane write the sidecar". So every existing cold-resume test, which calls
  `SetPairSession` from `AfterAcknowledge`, keeps passing unchanged. An
  explicit `SetPaneSidecar(addr, agent, mtime)` lets new tests separate birth
  from presence.
- **Probe.** It has two modes. `poke` starts a raw zellij session, then
  connects to and closes its socket the moment the socket appears; this proves
  the zellij bug without Pair. `launch` runs N cold `pair resume <tag>
  --layout3` launches in a scratch repo and counts sessions that died at
  birth. An optional `-hammer 10ms` flag runs a concurrent `list-sessions`
  loop that imitates Couch's registration poll.

## Architecture principles

- **ARCH-ORDER.** The create-path order is: clear the sidecar, spawn the
  sidecars, `LaunchSession`, the pane writes the sidecar (after the session
  has initialized), then the first zellij call from the poller or Couch.
  Tests pin each edge:
  - A launcher test asserts the stale sidecar is gone when
    `SpawnTitlePoller` runs.
  - The poller test's fake event log asserts no `SessionAlive` or
    `RenamePane` before the pane is seen.
  - A Couch test asserts no `PairSession` before `BornIn`.

  A failed stat or glob counts as "not yet", never as birth (the lesson
  "query failure cannot prove an empty external state"). Couch baselines
  before `Acknowledge`, and the helper is blocked until then, so the snapshot
  always comes before the launcher's clear.
- **ARCH-PURPOSE.** The purpose is "new sessions start". The poller is the
  measured trigger. Couch's cold-resume registration poll (`list-sessions`
  every 10 ms from ack to birth) is the densest prober in any launch path, so
  it is fixed here too. The probe's `-hammer` mode measures what that poll
  does. The residual callers (below) are recorded, not gated.
- **ARCH-DRY.** One piece of evidence, the pane sidecar, serves both
  consumers. The clear-then-await pattern copies `RemoveReadyRecord`. The
  probe reuses `probes/zellijprobe` (`Start`, `Scrub`, `SyncBuffer`).
- **ARCH-PURE.** The gate decision in `Run` sits behind the `Runtime` seam.
  `PaneMarks.BornIn` is a pure function with a table test.
- **ARCH-MOCK.** The poller's `fakeRuntime` and Couch's stateful fake are
  extended, not replaced. Live evidence comes from the probe against real
  zellij.
- **ARCH-CONSTRAINTS.** The gate costs one `stat` per 250 ms for at most 30 s
  per launch (usually a few ticks), and titles start ≤250 ms after birth.
  Couch's birth wait costs one glob plus stat per 10 ms. That is cheaper than
  the `list-sessions` process it replaces, and it stays inside the existing
  registration deadline.
- **ARCH-SECURE.** No new trust boundary. The sidecar sits in the operator's
  own scoped data dir and is written by Pair's own layout. The probe runs
  only against sessions it named, in a scratch repo, with `XDG_DATA_HOME`
  isolated.
- **ARCH-FUNERAL.** Production creates nothing durable: the clear *removes*
  one file, which the pane rewrites exactly as before. The probe creates a
  scratch repo, an isolated data dir, a fake-agent bin dir and its own zellij
  sessions and sidecar processes. On exit, and on every failure path, it
  deletes the sessions it named, kills processes whose argv carries its
  per-run tag, and removes all three temp dirs.
  `TestNoProbeExitsPastItsOwnCleanup` enforces the `os.Exit(run())` shape.

## Sweep result (launch-path zellij callers)

| Caller | Disposition |
|---|---|
| title poller startup loop | **fixed**: pane gate |
| Couch cold-resume registration (`PairSession` → `list-sessions` every 10 ms) | **fixed**: `BornIn` before the first `PairSession` |
| Couch cold create registration (claim file) | safe: no zellij call while waiting |
| Couch fresh start (ready file, then `list-sessions`) | safe: the ready file is written in-session |
| session-watch, `RecordOuterTTY`, `CmuxRename`, terminal title | safe: none of them calls zellij |
| launcher pre-launch `list-sessions` / `list-clients` / `delete-session` | safe for its own session (the server doesn't exist yet); residual for other sessions' births |
| Couch menu refresh after a start completes | residual: machine-wide `list-sessions`, fired when the start completes. On a cold create that is claim time, before `LaunchSession`. Measured in the live check with `probes/zellijcalls` |
| other threads' pollers (60 s), `pair list`, picker | residual: machine-wide, low probability |
| `zellijprobe.WaitUntilListed` | residual, dev-only: lists immediately after `Start`, so it can kill the probe's own session. Recorded; the new probe does not call it before birth |

Upstream zellij is the only cure for the residual rows.

## Tasks

### Task 1: Repro probe (commit before the fix, so "before" is measured)

**Files:** Create `probes/zellijbirthrace/main.go`, `probes/zellijbirthrace/SKILL.md`.

- [ ] `poke -n N`: for each trial, `zellijprobe.Start` a one-pane layout named
      `birthrace<i>-<pid>`. Poll `<sockdir>/<name>` every 1 ms, where sockdir
      defaults to `$TMPDIR/zellij-<uid>/contract_version_1` and can be set with
      `-sockdir`. `net.Dial("unix")` and `Close` it the moment it exists.
      Then wait 2 s and classify the trial:
      - `died`: `zellij.log` gained a `Panic occurred` line (counted before
        and after, `-zellij-log` override), or the pty client stopped.
      - `survived`: the session renders.
      - `inconclusive`: the socket never appeared.
      `Close()` afterward.
- [ ] `launch -n N [-pair PATH] [-hammer DUR] [-timeout 15s]`: the parent
      re-execs itself as a detached child
      (`sh -c '"$0" __launch … &'`, so it reparents to launchd). The child
      escapes the launcher's `InZellijPane` ancestry refusal, writes results
      to a file and exits; the parent prints the file. The child:
      1. Makes a scratch `git init` repo, a temp `XDG_DATA_HOME`, and a fake
         `claude` (`exec sleep 3600`) on PATH.
      2. Per trial, starts `pair resume br<childpid>t<i> --layout3` under a
         pty. The env is `zellijprobe.Scrub` of `ZELLIJ`, `PAIR_` and
         `COUCH_`, plus those overrides.
      3. Waits for the pane sidecar (`$XDG_DATA_HOME/pair/repos/*/pane-<tag>-claude.json`)
         → `born`; for a `zellij.log` panic delta → `died`; or for the timeout
         → `inconclusive`.
      4. Only after `born`, confirms the session is listed and not EXITED,
         using its name from `session-names.jsonl`.
      5. Tears down with `kill-session` and `delete-session --force` on that
         name, `pkill -KILL -f <tag>` (the tag appears only in argv of
         processes this trial started), and closes the pty.
      6. `-hammer` runs `zellij list-sessions --short` every DUR from trial
         start until the verdict.
      7. Removes all temp dirs at exit.
- [ ] Output one line per trial plus `PROBE-RESULT mode=… n=… born|survived=… died=… inconclusive=…`.
      A conclusive run exits 0, whatever the verdict. Errors exit 1.
- [ ] `go vet ./probes/...` and `go test ./probes/...` (probe-shape test).
- [ ] With the sandbox off, run the baseline against current `bin/pair`, built
      from main before the fix:
      - `poke -n 5`
      - `launch -n 10`
      - `launch -n 10 -hammer 10ms`

      Record the counts in `## Log`. Expected: poke and plain launch die most of
      the time.
- [ ] Commit: `#287: probes: zellijbirthrace — count sessions that die at birth`.

### Task 2: Title poller pane gate (TDD)

**Files:** Modify `cmd/internal/titlepoller/run.go` and `run.go`'s `Options`
docs. Test in `cmd/internal/titlepoller/run_test.go`.

- [ ] Fake: add an ordered `events []string` log (`stat-pane`,
      `session-alive`, `rename`). Add `paneAfterSleeps int`: the pane path's
      mtime becomes present after that many `Sleep` calls (-1 means never).
- [ ] Failing tests:
  - `TestRunMakesNoZellijCallBeforeThePaneAppears`: the pane appears after 3
    sleeps. No `session-alive` or `rename` event comes before the first
    successful `stat-pane`. After it, the loop probes and renders as before.
  - `TestRunExitsWithoutTouchingZellijWhenThePaneNeverAppears`: the grace
    expires, `sessionAliveCalls == 0` and the exit code is 0.
  - `TestRunPassesTheGateAtOnceForALivePane` (the attach shape): the pane is
    present at start, so the gate costs one stat.
  - Update the existing tests: their `sessionAliveSeq` loses the grace probe,
    they seed the pane mtime, and `TestRunExitsOnSessionMissThreshold` expects
    3 calls, not 4.
- [ ] Implement: resolve `panePath, err := paths.PaneChecked(opts.Agent)`
      (on error, return 0, since the gate can't be satisfied safely). Replace
      the `SessionAlive` grace loop with `rt.ModTime(panePath)` polling at
      250 ms bounded by `StartupGrace`. Update the `StartupGrace` comment.
- [ ] `go test ./cmd/internal/titlepoller/ -count=1` → PASS.
- [ ] Commit.

### Task 3: Launcher clears the agent pane sidecar on create (TDD)

**Files:** Modify `cmd/internal/launcher/createflow.go` (before
`SpawnSessionWatcher`, ~l.767). Test in `cmd/internal/launcher/createflow_test.go`.

- [ ] Fake: `SpawnTitlePoller` records whether `artifactpath.ResolveScoped(dataDir, tag).Pane(agent)`
      is in `f.files` when it is called. `LaunchSession` records the same.
- [ ] Failing test `TestCreateClearsTheAgentPaneSidecarBeforeSpawningAnything`:
      seed a stale sidecar and run a create. The file is absent at both
      `SpawnTitlePoller` and `LaunchSession`. A stale twin for another agent
      is left alone. An attach-path test asserts the attach path does *not*
      remove the live sidecar.
- [ ] Implement with one `rt.Remove(artifactPaths.Pane(agent))` and a comment
      that points at the poller gate and #287.
- [ ] `go test ./cmd/internal/launcher/ -count=1` → PASS.
- [ ] Commit.

### Task 4: Couch cold resume waits for birth before `list-sessions` (TDD)

**Files:** Create `cmd/internal/couchcore/panebirth.go` and
`panebirth_test.go`. Modify `artifactcollision.go` (the `PaneBirthIO`
interface plus the scoped implementation), `artifactcollision_fake.go` and
`launch_existing.go`. Add tests in a new `cmd/internal/couchcore/coldresume_birth_test.go`.

- [ ] `PaneMarks.BornIn` table test. Rows: absent→present, same→same,
      changed mtime, removed, a stale twin that is unchanged while ours is
      absent, and the clear-then-rewrite sequence.
- [ ] Fake: add the pane model (the present-edge bump in `SetPairSession`,
      `SetPaneSidecar`, and `PaneSidecars` with an optional error).
- [ ] Failing tests:
  - `TestColdResumeMakesNoSessionProbeBeforeThePaneIsBorn`: a
    `BeforePairSession` hook fails the test if `PaneSidecars` has not yet
    reported birth. `AfterAcknowledge` sets the sidecar and then the session.
  - `TestWarmReattachSkipsTheBirthWait`: a warm reattach calls no
    `PaneSidecars`.
  - `TestColdResumeTimesOutWhenThePaneNeverAppears`: this goes through the
    existing registration error and diagnosis, not a new error path.
  - A scoped-implementation test over a temp dir: glob and stat, an empty
    dir, a missing dir.
- [ ] Implement:
  - Before `h.Acknowledge()`, and only when `shape == StartColdResume`,
    take `baseline, err := PaneSidecars(addr)`. On error, fail the start
    through `failTrackedPreAckStart`.
  - `awaitResumeRegistration(ctx, address, baseline *PaneMarks)`: when the
    baseline is non-nil, poll `PaneSidecars` every 10 ms until
    `baseline.BornIn(now)`, then run the existing `PairSession` loop. The
    registration deadline stays the same.
- [ ] `go test ./cmd/internal/couchcore/ -count=1` → PASS.
- [ ] Commit.

### Task 5: Docs

- [ ] `atlas/architecture.md`:
  - In the title-poller paragraph, the startup gate replaces "waits for the
    session to appear".
  - In the l.336 `pane-<tag>-<agent>.json` sentence, the sidecar is now also
    birth evidence, and the create path clears it.
- [ ] `atlas/couch.md`: in the registration section, cold resume waits for
      birth, then checks liveness.
- [ ] `probes/zellijbirthrace/SKILL.md`: usage, the sandbox-off requirement,
      and how to read the result. Link it from `atlas/index.md` if probes are
      indexed there.
- [ ] `workshop/lessons.md`: the rule.
- [ ] Commit.

### Task 6: Verification

- [ ] `make build`. Then run the full suite with the sandbox off and the
      retention-owner scrub:
      `env -u PAIR_DATA_DIR -u PAIR_TAG -u PAIR_RETENTION_PROTOCOL -u PAIR_RETENTION_BACKGROUND -u PAIR_RETENTION_START_ID make -k test`,
      then `go test ./... -count=1`.
      `make test-changelog` is a known pre-existing failure. Note it, don't
      chase it.
- [ ] Probe after the fix: `launch -n 20` should give 0 died, and
      `launch -n 10 -hammer 10ms` should be recorded. `poke` is unchanged
      because it is the zellij bug itself. Record the counts in `## Log`.
- [ ] Operator smoke test: a Couch cold create in `~/workspace/astro`, plus a
      cold resume of a parked thread, under `probes/zellijcalls/trace.sh arm`.
      This gives timings for the residual Couch menu refresh.
- [ ] Clean up the casualties listed in the Log:
  - Archive `couch-3a64268b355e8183` in Couch.
  - Scope `3efc24a8a43ffcd9` and tag `probe-standalone` in `fcff31946c0ac9d2`.
    Delete them only after the operator confirms, and never delete a Couch
    record file.

### Task 7: Upstream report

- [ ] Draft the zellij issue: title, 0.45.1 version, the `lib.rs:1462`
      location, a minimal repro (`probes/zellijbirthrace poke`, or its
      15-line python equivalent), and the proposed fix (`RemoveClient` should
      tolerate `session_data == None`). Show it to the operator. File it with
      `gh issue create -R zellij-org/zellij` only on their go-ahead, and
      record the link in `## Log`.

## Revisions

- **2026-09-18, close review (FIX-THEN-SHIP).** Deltas against the plan
  above:
  - **The already-live cold resume.** This row belongs in the Sweep table. A
    session that is live but *attached* fails the detached proof, reaches the
    cold path, and Pair refuses the resume, so no pane is born. The birth wait
    would run out the deadline, and the cold-resume cleanup, which owns the
    session, would delete it. `coldResumeBirthBaseline` now asks once, before
    release, whether the session is live. If it is, there's no wait
    (pre-#287 behavior); if liveness can't be observed, the start fails
    before release.
  - **Failed-observation policy.** The code now matches the ARCH-ORDER line
    above. A failed stat or glob during Couch's wait is "not yet", and the
    last error rides the deadline's error. (It used to end the wait, and the
    cleanup would then quiesce a session this launch had just made.)
  - **Fake surface.** `SetPaneSidecar(addr, agent)` has no mtime argument
    (the fake's clock is a counter). `ClearPaneSidecar`, `PaneQueries` and
    `PaneSidecarsHook` were added. A `FakeRunner.OnRelease` →
    `PairLaunchReleased` model of Pair was added and then **removed**: it
    birthed the pane of any released launch whose session was up, which the
    host never does for a refused resume, and so it hid the deletion path. A
    pane is born only on the session's live edge or explicitly.
  - **Interface location.** `PaneBirthIO` is declared in `panebirth.go`;
    only the scoped implementation lives in `artifactcollision.go`.
  - **One path declaration.** `titlepoller.BirthEvidence(dataDir, tag,
    agent)` is the file the gate waits for, and the launcher clears exactly
    that path. The two can't name different files and quietly bring back the
    stale-sidecar race.
