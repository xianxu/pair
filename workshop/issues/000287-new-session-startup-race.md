---
id: 000287
status: codecomplete
deps: []
github_issue:
created: 2026-09-18
updated: 2026-09-18
estimate_hours:
started: 2026-09-18T13:38:02-07:00
flow: {kind: full, provenance: inferred}
actual_hours: 1.54
---

# New sessions die at birth: title poller probes zellij during server startup

## Problem

Creating a new Pair session fails most of the time: the zellij server panics
~20 ms after starting, before the session exists. The operator sees a blank
pane that echoes typed characters (the zellij client never reached raw mode);
Couch marks the thread `live` while nothing runs. Reattaching existing sessions
is unaffected, which is why this surfaced as "can't start a thread for astro"
— astro was the only cold create being attempted. **It is not astro-specific:**
a standalone `pair resume <new-tag> --layout3` in a fresh scratch repo reproduces
it.

Zellij log (`$TMPDIR/zellij-501/zellij-log/zellij.log`), every failed launch:

```
Starting Zellij server!
Panic occurred: At zellij-server/src/lib.rs:1462:26
  called `Option::unwrap()` on a `None` value
```

Failures on 09-15 21:07, 22:10, 22:54 ×3; 09-16 ×6; 09-17 17:17; 09-18 12:24,
12:33, 12:41, 12:42.

## Root cause

- **zellij 0.45.1 bug (still on upstream `main`):** `ServerInstruction::RemoveClient`
  does `session_data…as_ref().unwrap()`. A connection that is accepted *first*
  and then closes before any real client has initialized the session crashes
  the server. Reproduced with a bare connect-and-close on a new session's
  socket (1 ms poll → panic, same location).
- **Pair's trigger:** the launcher spawns `pair title` (`SpawnTitlePoller`)
  *before* `LaunchSession`. The poller's startup loop (`titlepoller/run.go`,
  "wait for the zellij session to appear") calls `SessionAlive` immediately,
  and that runs `zellij list-sessions --short`. `list-sessions` connects to
  **every** socket (`assert_socket`). Because the poller is already running when
  the new socket is bound, its probe can be accepted before the real client,
  and the probe's hang-up then panics the server.
- Evidence: a `zellij` PATH shim logging every call during a reproducing launch
  showed `list-sessions --short` from `pair title` 4 ms after `Starting Zellij
  server!`, and the panic 17 ms later. Firing `list-sessions` *after* the
  socket appears (a fresh process, ~15 ms startup) never crashed (0/3), because
  the real client wins the accept. So the race is decided by whoever connects
  first, and the pre-spawned poller is positioned to win it.

## Spec

- **The title poller must not open a zellij connection until the session has
  initialized.** Replace the startup wait's `list-sessions` probe with
  Pair-owned evidence produced from *inside* the session. Candidate: the agent
  pane's sidecar (`pane-<tag>-<agent>.json`, written by the layout's pane
  command), required to be fresh (newer than the poller's start, so a stale
  file from a previous launch doesn't count). Once seen, the steady-state loop
  (60 s `SessionAlive`) is safe.
  - Rejected: a fixed sleep before the first probe (a timing guess, not a fix);
    `stat` on zellij's socket file (couples Pair to zellij's socket layout, which
    just moved to `contract_version_1/`, and "socket exists" isn't "session
    initialized").
- **Sweep the other startup-window probers** in the launch path the same way
  (anything spawned before or alongside `LaunchSession` that shells out to
  `zellij`): the shim showed only `pair title`, but `session-watch` and the
  pane helpers should be checked against the same rule.
- **Report upstream** (zellij): `RemoveClient` before the first client initializes
  the session panics. That's the only cure for the residual risk: any
  `list-sessions` on the machine (other threads' pollers at 60 s) can still hit
  a new server's ~20 ms window, at low probability.

### Design (pair session, 2026-09-18)

Durable plan: `workshop/plans/000287-new-session-startup-race-plan.md`.

- **Evidence.** The agent pane's sidecar. The layout's pane command writes it
  as its first act, and a pane exists only once the first client has
  initialized the session, so any connection after it can't reach the
  `RemoveClient` unwrap.
- **Freshness: clear, then await.** The create path removes this agent's
  sidecar before it spawns the session watcher, the title poller or zellij.
  That is the same pattern as `RemoveReadyRecord` before readiness. For the
  poller, "the file exists" therefore means "this launch's pane ran". Attach
  doesn't clear, so the live pane's file satisfies the gate on the first stat.
  One rule covers both spawn sites. Rejected alternatives:
  - *mtime newer than the poller's own start.* Under load the poller's first
    clock read can come after the pane write, and then a fresh file reads as
    stale.
  - *A launcher-supplied timestamp.* This adds a contract field and a
    wall-clock comparison against file mtimes, which on some filesystems come
    from a coarser clock. The ordering already guarantees what the timestamp
    would.
  - *A launch ordinal or nonce written into the file.* This needs a layout
    change. Worse, it breaks the single-instance handoff. A poller left over
    from a launch that died at birth still holds the pidfile and waits for
    *its* ordinal. The next launch's poller defers to it, so no poller serves
    the new session.
- **Couch cold resume uses the same evidence.** The sweep found Couch's
  cold-resume registration polling `list-sessions` every 10 ms from the
  helper ack onward, which covers the birth window. That makes it the densest
  prober in any launch path. Couch starts the launcher, so it can't see the
  clear. Instead it snapshots the tag's sidecars (`map[path]mtime`) before
  releasing the blocked helper, and makes no `PairSession` call until the
  snapshot changes. The comparison is by equality, so it needs no clock. A
  removal doesn't count as birth, and neither does an unchanged stale twin.
- **Sweep result.** The full table is in the plan. Fixed: the title poller
  and Couch's cold-resume poll. Safe: session-watch, cmux/tty/title, Couch's
  cold create (claim file) and fresh start (ready file). Residual, curable
  only upstream: any machine-wide `list-sessions`. That covers other threads'
  pollers, `pair list`, the picker, the launcher's pre-launch checks against
  *other* births, and Couch's menu refresh after a start completes. The live
  trace times that last one. Also residual, dev-only:
  `zellijprobe.WaitUntilListed` lists immediately after `Start`.

### Out of scope (follow-up)

- **A dead launch is invisible.** When the server dies at birth, the zellij
  client hangs on a cooked tty (blank, echoing pane), the launcher waits
  forever, and Couch keeps the record `live`. The launcher should notice that
  the session never came up and fail loudly. File separately.

## Done when

- A unit test (fake runtime) proves the title poller makes no `SessionAlive` /
  zellij call before fresh pane evidence exists, and resumes normal polling
  after.
- The launcher's create path clears the agent pane sidecar before it spawns
  anything, and attach leaves the live pane's sidecar alone (tests pin both).
- A Couch cold resume makes no session probe until the thread's pane is born
  against a baseline taken before the helper is released. A warm reattach
  takes no baseline. Stateful-fake tests cover the order, a pane that is
  never born, and a baseline that can't be read.
- A live repro driver (committed under `probes/`) runs N standalone cold
  launches in a scratch repo and counts zellij panics. Before the fix they
  panic (4/10 plain; 10/10 under a 10 ms `list-sessions` loop); after it,
  0/N.
- A Couch cold create of a thread in `~/workspace/astro` starts normally, and
  a park + cold resume of a thread works.
- An upstream zellij report is filed with a repro verified against 0.45.1:
  zellij-org/zellij#5632 (https://github.com/zellij-org/zellij/issues/5632).

## Plan

- [x] Repro probe `probes/zellijbirthrace` (`poke`, `launch`, `-hammer`),
      committed; baseline counts measured against the pre-fix `pair`.
- [x] Title poller: pane-evidence startup gate + fake-runtime tests.
- [x] Launcher create path clears the agent pane sidecar before spawning
      anything + test (attach doesn't clear).
- [x] Couch cold resume: `PaneMarks` baseline before ack, no `PairSession`
      before birth + tests (stateful fake models the pane).
- [x] Sweep other launch-time zellij callers against the rule (done in
      design; residuals recorded in the plan and atlas).
- [x] Docs: atlas (poller, pane sidecar, Couch registration), probe
      SKILL.md, lessons.
- [x] Live verification: full suite; probe after fix 0/20; operator smoke
      of Couch cold create (astro, 42shots) and a park + cold resume;
      casualty cleanup. The zellij-tracer timing of Couch's menu refresh was
      not taken, so that refresh stays a recorded residual (see Design).
- [x] Upstream zellij issue drafted, verified (its python repro panicked
      0.45.1 3/3) and filed on the operator's go-ahead:
      zellij-org/zellij#5632.

## Log

### 2026-09-18
- 2026-09-18: closed — probes/zellijbirthrace before fix: poke 3/5 died, launch 4/10, launch -hammer 10ms 10/10; after fix: launch 0/20 died (hammer 9/10 = external-prober residual). Unit: titlepoller gate tests (no zellij call before pane), launcher clear-before-spawn + attach-keeps tests, couchcore PaneMarks table + cold-resume birth-wait tests; mutation (wait disabled) turns both cold-resume tests red. go test ./... 72 pkgs ok; make -k test all ok except pre-existing test-changelog (same on main). Operator smoke: Couch cold create in astro and 42shots start normally; park + cold resume ok; astro casualty archived. Upstream zellij report drafted, repro verified 3/3 on 0.45.1; filing tracked by #288 (operator: land now).; review verdict: FIX-THEN-SHIP
- 2026-09-18: flow upgraded quick → full — 811 added lines in code files (limit 100)

Diagnosed from a brain session (operator: "can't start a thread for
../astro"). Current casualty: astro thread `couch-3a64268b355e8183`
(`📁astro-couch-6`), with a hung zellij client on a dead server and a `live`
record; archive it in Couch after the fix. Probe leftovers to clean: tag
`probe-standalone` in astro's scope (`fcff31946c0ac9d2`) and the scratch-repo
scope `3efc24a8a43ffcd9` under `~/.local/share/pair/repos/`.

Unrelated leak found on the way: an orphaned `zellij action rename-pane … [rename:
long-tab-name…]` (pid 72206, ~45 h old) targeting test session
`couchnestedrows-71075`. A pair test leaves a hung zellij CLI behind.

#### Handoff to the pair session

- **Tree state at filing:** `main` had another session's uncommitted edits
  (`.github/workflows/merge-check.yml`, `Makefile` type change, `bootstrap.sh`,
  deleted `scripts/issue-sync.sh`, new
  `scripts/merge-checks.d/40-duplicate-issue-id.sh`). They were left untouched.
  Find out whether they're still live before branching in place. This issue's
  commit (`#287: issue-sync: spec/plan`) is local, not pushed.
- **How the diagnosis was confirmed** (reuse these for the repro driver):
  - *Raw zellij works.* `zellij --config-dir pair/zellij
    --new-session-with-layout …/main-3.kdl --session <s>` from astro, under
    `script -q`, starts normally, even with the stuck client's exact env
    (`ps eww`) and an emoji session name.
  - *Pair fails, not only in astro.* A standalone `pair resume <new-tag>
    --layout3` in astro and in a fresh `git init` scratch repo both panic.
  - *The crash is reproducible without pair.* A bare connect-and-close on the
    new socket (`$TMPDIR/zellij-501/contract_version_1/<session>`, python,
    1 ms poll) gives the same panic on the first try.
  - *Ruled out:* hammering `zellij list-sessions` or `zellij --session <s>
    action list-panes` from a sequential loop, and a fresh `list-sessions`
    fired once the socket appears (0/3; process startup makes it lose the
    accept race). Also ruled out: `--config-dir` changes, session-name
    length, astro's dot-dirs, a second zellij binary, and the zellij version
    (0.45.1 installed 09-10; failures begin 09-15).
  - *Caller identified:* a `zellij` PATH shim logging `$PPID` + argv named
    `pair title <tag> claude <session>` as the prober. It's a Heisenbug: the
    shim's ~20 ms per call sometimes lets the real client win, so a
    shimmed run can pass.
- **Running the repro from inside a Pair pane:** the launcher refuses nested
  launches by walking process ancestry (`InZellijPane`), so env scrubbing
  alone doesn't help. Double-fork with `setsid` so the child reparents to
  launchd, `env -i` a minimal env, and wrap in `script -q` for a pty. Under
  the harness sandbox, zellij/ps/pty calls need the sandbox disabled, and
  `$TMPDIR` differs between sandboxed and unsandboxed shells.
- **Open question for the fix:** confirm the agent pane sidecar is written
  early enough and reliably enough to gate on, and pick the freshness rule
  (mtime vs. launch ordinal/nonce). The layout's pane command writes
  `{"pane_id","cwd"}` to `$PAIR_AGENT_PANE_…` at start.

#### Pair session: design

- Claimed. The open question above is answered in `### Design`: the evidence
  is the pane sidecar, with freshness by clear-then-await. It is written first
  thing by `sh -c "printf … > $PAIR_AGENT_PANE_PATH; …"` in both layouts, and
  that path is bound on every create (`EnvironmentBindings`). The durable plan
  is `workshop/plans/000287-new-session-startup-race-plan.md`.
- Sweep (Explore subagent over `launcher`, `couchcore`, `couchtty`,
  `sessionwatch` and the layouts). The one new launch-path prober is Couch
  cold resume: `awaitResumeRegistration` → `PairSession` → `LivenessContext`,
  i.e. `list-sessions` every 10 ms from the helper ack onward. It is added to
  scope. The bundled layout copies under `runtimebundle/assets` are generated
  from `zellij/layouts` on every `make build`, and the design changes no
  layout. Nothing in Pair attaches to an EXITED session by design. There is
  one narrow snapshot race (live→EXITED between the decision and
  `AttachSession`), which is not addressed here.
- Tree state: the other session's uncommitted edits (merge-check workflow,
  Makefile, bootstrap.sh, `scripts/issue-sync.sh`,
  `scripts/merge-checks.d/40-duplicate-issue-id.sh`) are still in the working
  tree. They look like a base-layer propagation from ariadne and are left
  untouched.

#### Pair session: implementation

- **Baseline.** `probes/zellijbirthrace`, macOS, zellij 0.45.1, pre-fix `pair`
  built from main:
  - `poke -n 5`: 3/5 died. That is the zellij bug with no Pair involved.
  - `launch -n 10`: 4/10 died.
  - `launch -n 10 -hammer 10ms`: 10/10 died. That is Couch's cold-resume
    poll cadence, so under it every cold resume dies.
- **After the fix** (`bin/pair` at `57b303e7`'s code):
  - `launch -n 20`: **0/20 died**. Births took 0.65–1.41 s.
  - `launch -n 10 -hammer 10ms`: 9/10 still died. That hammer is an
    *external* prober, which is the residual risk only upstream can cure.
    Couch's own poll no longer probes during a birth.
- **Probe details.**
  - With no arguments it only describes itself and exits 0. `make
    test-smoke` runs every probe bare, and this one's modes kill zellij
    servers and start sessions, so it is an instrument like `zellijcalls`,
    not a smoke test.
  - Its `-pair` binary must be named `pair`: a `pair-before` copy dispatches
    as `pair-go`.
- **Couch fake model.** Seven existing Couch tests hung at the 15 s
  registration deadline. They had set a cold-resumed thread's session live
  before the start, which with birth-at-live-edge put the birth before the
  baseline. Rather than editing seven unrelated tests, the test env gained a
  model of Pair (`FakeRunner.OnRelease` → `PairLaunchReleased`): a released
  launch whose session is up has had its pane born. `couchcore` went from
  258 s to 118 s. Mutation check: with the birth wait disabled, both new
  cold-resume tests fail.
- **Observed, not chased.** Birth time rose monotonically across one probe
  run (0.65 s → 1.41 s over 20 trials in one scratch scope). It looks like
  per-scope state growing across trials, e.g. `session-names.jsonl` and the
  ledger. That is possibly launcher pre-launch work linear in a scope's
  history. It is not in #287's window; worth a look under #218 (couch thread
  startup time).
- **Upstream.** Confirmed in the source: in v0.45.1 `lib.rs:1462:26` is
  `session_data…as_ref().unwrap()` in `RemoveClient`'s regular-client branch
  (the watcher branch just above uses `if let Some`), and `main` has the same
  code. The report is drafted, pending the operator's go-ahead.

#### Pair session: verification

- `go test ./...`: 72 packages ok after `d1cdf69f`. The first run caught two
  `artifactpath` inventory gates, now fixed: `panebirth.go` is classified,
  artifactcollision declares the `composite-pane-glob` binding, the fake no
  longer spells artifact paths, and the probe's root `.gitignore` entry is
  in. `make -k test`: everything passes except `test-changelog`, the known
  pre-existing failure (`viewer: process target is outside selected owner
  directory`, same as on main).
- `make install`. The operator's Couch (pid 34889, run from `bin/couch`) was
  restarted onto the fix.
- **Operator smoke, 2026-09-18.**
  - A Couch cold create of a thread in `~/workspace/42shots` started
    normally.
  - Park, then cold resume of that thread, worked. This is the path the old
    10 ms poll killed.
  - The astro casualty `couch-3a64268b355e8183` was archived in Couch. Its
    first archive attempt was refused as "live" because its hung launcher
    (pid 23676) was still alive. Restarting Couch killed that launcher's pty,
    so the helper read dead and the session absent, and archive went through.
  - A Couch cold create of a thread in `~/workspace/astro` started normally.
  - Diagnosis leftovers removed on the operator's go-ahead:
    - the scratch-repo scope `3efc24a8a43ffcd9`;
    - the 11 `probe-standalone` files in astro's scope;
    - the ~2-day-old orphan `zellij action rename-pane` (pid 72206) that a
      test had left behind. Its source is not identified here.
- Follow-up filed: #288, a launch whose server dies at birth hangs the
  launcher and leaves the thread live. That is the Spec's out-of-scope row,
  demonstrated live by the astro casualty.

#### Upstream report (filed as zellij-org/zellij#5632)

Filed 2026-09-18 on the operator's go-ahead:
https://github.com/zellij-org/zellij/issues/5632. The duplicate search turned
up no report of this panic. #2476 is the same class through another handler
(`lib.rs:416`, 2023, a `--server` session attached before any client), and
the filed text cites it under "Related". The draft below was otherwise filed
as-is, except that the OS line reads macOS 26.6.2 (arm64).

Verified against zellij's source: `lib.rs:1462:26` in v0.45.1 is the
`session_data…as_ref().unwrap()` in `RemoveClient`, and `main` still has the
same code. The repro panicked the server 3/3 here.

````markdown
Title: Server panics in RemoveClient when a connection closes before the first client initializes the session

**zellij version:** 0.45.1 (the same code is still on `main`)
**OS:** macOS 15 (Darwin 25.6)

### What happens

If a connection to a brand-new session's socket is accepted and then closes
before the first real client has initialized the session, the server panics:

```
Starting Zellij server!
Panic occurred: At zellij-server/src/lib.rs:1462:26
  called `Option::unwrap()` on a `None` value
```

The session never comes up. The client that started it hangs on a cooked tty,
so the terminal shows a blank screen that echoes keystrokes.

In practice any `zellij list-sessions` running on the machine while a new
server is starting can trigger this, because `list-sessions` connects to every
session socket to check whether it is alive. Tools that poll `list-sessions`,
such as a status bar or a launcher waiting for its session to appear, kill new
sessions at a high rate:

- A process connecting within ~1 ms of the socket appearing: 3/5 sessions died.
- A launcher whose helper ran `list-sessions` right as it started the session:
  4/10 died.
- A `list-sessions` loop at 10 ms cadence during startup: 10/10 died.

### Cause

`ServerInstruction::RemoveClient`, in the regular-client branch, unwraps
`session_data` unconditionally:

```rust
remove_client!(client_id, os_input, session_state, session_data);
session_data
    .write()
    .unwrap()
    .as_ref()
    .unwrap()          // lib.rs:1462 — None until the first client initializes the session
    .senders
    .send_to_screen(ScreenInstruction::RemoveClient(client_id))
    .unwrap();
```

`session_data` is `None` until the first client's `NewClient` builds the
session. A connection that is accepted first (such as the liveness check in
`list-sessions`) and then hangs up produces a `RemoveClient` while it is still
`None`. The watcher branch just above already handles this with
`if let Some(session_data) = …`.

### Minimal reproduction

Connect to and close the new session's socket as soon as it appears, before
the starting client connects:

```python
# repro.py: run with the zellij binary on PATH
import os, pathlib, socket, subprocess, time
name = f"birth-race-{os.getpid()}"
sockdir = pathlib.Path(os.environ.get("TMPDIR", "/tmp")) / f"zellij-{os.getuid()}" / "contract_version_1"
sock = sockdir / name
client = subprocess.Popen(["script", "-q", "/dev/null", "zellij", "--session", name])
while not sock.exists():
    time.sleep(0.001)
s = socket.socket(socket.AF_UNIX); s.connect(str(sock)); s.close()
time.sleep(2)
print(subprocess.run(["zellij", "list-sessions", "--no-formatting"], capture_output=True, text=True).stdout)
client.kill()
subprocess.run(["zellij", "delete-session", "--force", name])
```

The server's log shows the panic, and the session is gone or EXITED. It
doesn't fire on every run because the starting client sometimes wins the
accept; three runs are usually enough. (The socket directory above is the
macOS default. Adjust it for `ZELLIJ_SOCKET_DIR` or `XDG_RUNTIME_DIR`
setups.)

### Suggested fix

Treat a missing `session_data` in `RemoveClient` as "nothing to notify": guard
the `send_to_screen` and `send_to_plugin` calls with
`if let Some(session_data) = session_data.write().unwrap().as_ref()`, as the
watcher branch does. The same `.as_ref().unwrap()` pattern appears in other
handlers in `lib.rs`, and any of them reachable before `NewClient` has the same
exposure.
````

#### Pair session: close review (FIX-THEN-SHIP) and dispositions

The review is in `workshop/plans/000287-new-session-startup-race-close-review.md`.
Its findings block didn't parse, so the gate ledger holds none, and the
dispositions are recorded here.

- **Important 1, fixed: a cold resume against a live-but-attached session
  timed out, and the cleanup deleted that session.** Such a session fails the
  detached proof, reaches the cold path, and Pair refuses the resume, so no
  pane is ever born. `coldResumeBirthBaseline` now asks once, before release,
  whether the session is live: live means no wait (the pre-#287 outcome),
  unobservable means the start fails before release. Tests:
  `TestColdResumeAgainstALiveSessionNeitherWaitsNorDeletesIt` and
  `TestColdResumeRefusesToReleasePairWhenLivenessIsUnknown`. Mutation:
  dropping the live skip reproduces the timeout on a live session.
- **Important 2, fixed: the fake invented a birth.** `FakeRunner.OnRelease` /
  `PairLaunchReleased` are removed. A pane is born only on the session's live
  edge or through an explicit `SetPaneSidecar`. The seven older tests pass
  without it: their session is live before release, so they take the
  live-session branch, which is the honest reading of that state.
  `TestResumeUnobservableSessionKeepsUnknownOccupied` now makes zellij
  unreachable only after the ack, since that cleanup is what it tests. The
  lesson written earlier this session ("fix the fake, not the tests") is
  corrected in `workshop/lessons.md`.
- **Minor, fixed:**
  - A failed observation mid-wait is now "not yet", with the last error
    joined into the deadline's error
    (`TestColdResumeWaitsThroughAFailedPaneObservation`; mutation-checked).
  - The prefix-tag comment on `PaneSidecars` now states the limit instead of
    overstating the filter.
  - The Done-when's "most launches panic" now carries the measured figures.
  - The probe's detached child stops, and removes the report dir, when its
    parent is gone. Verified live: `kill -9` of the parent mid-run left no
    dir, process or session.
- **Minor, explained in the code:** the poller's silent exit on an
  unresolvable pane path. Its stdio is /dev/null, and the launcher resolves
  the same path first and refuses the launch on failure.
- **Coverage gap (c), closed:** `titlepoller.BirthEvidence` is the single
  declaration of the awaited file. Both the gate and the launcher's clear use
  it, and the launcher test asserts against it.
- **ARCH-DRY note:** there are two `awaitPaneBirth` helpers (poller, Couch).
  Extract them when #288 adds the launcher's waiter. Noted there.
- **Plan:** a `## Revisions` entry in the durable plan covers the live-session
  row, the observation policy, the fake-surface drift and the interface
  location.

## Revisions

- 2026-09-18 (pair session): The `## Plan` rows were rewritten to match the
  durable plan. The "sweep" row became concrete tasks: a launcher clear, and a
  Couch cold-resume birth wait, added after the sweep. The probe gained a
  `-hammer` mode to measure the Couch-style poll. The Done-when is unchanged.
- 2026-09-18 (pair session): The Done-when row "Upstream zellij issue filed;
  link recorded in the Log" is **not met at close**. The operator asked to
  land before reviewing the draft, and filing would post under their GitHub
  account, so it was not filed without their go-ahead. The draft, verified
  against zellij's source and with a working repro, is in the Log. Filing it
  is a Plan row on #288. Nothing else in the Done-when changed.
- 2026-09-18 (pair session): `## Done when` restated for what was built. It
  adds the launcher-clear and Couch cold-resume rows the design added, and
  extends the Couch row to cover park + cold resume. The upstream row becomes
  "drafted and verified; filing tracked by #288", per the revision above.
  Triggered by `sdlc close`'s done-when-freshness gate.
- 2026-09-18 (pair session, after close): the upstream deferral is undone.
  The operator gave the go-ahead, the report was filed as
  zellij-org/zellij#5632, and the Done-when upstream row is back to "filed,
  link recorded". #288's filing row is ticked with the link.
