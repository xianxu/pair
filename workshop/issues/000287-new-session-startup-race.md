---
id: 000287
status: working
deps: []
github_issue:
created: 2026-09-18
updated: 2026-09-18
estimate_hours:
started: 2026-09-18T13:38:02-07:00
flow: {kind: quick, provenance: inferred, spec: "ec74e4b5", done: "78beb9e3"}
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
- A live repro driver (committed under `probes/`) runs N standalone cold
  launches in a scratch repo and counts zellij panics: before the fix most
  launches panic; after it, 0/N.
- A Couch cold create of a thread in `~/workspace/astro` starts normally.
- Upstream zellij issue filed; link recorded in the Log.

## Plan

- [ ] Repro probe `probes/zellijbirthrace` (`poke`, `launch`, `-hammer`),
      committed; baseline counts measured against the pre-fix `pair`.
- [ ] Title poller: pane-evidence startup gate + fake-runtime tests.
- [ ] Launcher create path clears the agent pane sidecar before spawning
      anything + test (attach doesn't clear).
- [ ] Couch cold resume: `PaneMarks` baseline before ack, no `PairSession`
      before birth + tests (stateful fake models the pane).
- [ ] Sweep other launch-time zellij callers against the rule (done in
      design; residuals recorded in the plan and atlas).
- [ ] Docs: atlas (poller, pane sidecar, Couch registration), probe
      SKILL.md, lessons.
- [ ] Live verification: full suite; probe after fix 0/N; operator smoke of
      Couch cold create in astro + a cold resume under the zellij tracer;
      casualty cleanup.
- [ ] Upstream zellij issue drafted, filed on operator go-ahead, link logged.

## Log

### 2026-09-18

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

## Revisions

- 2026-09-18 (pair session): The `## Plan` rows were rewritten to match the
  durable plan. The "sweep" row became concrete tasks: a launcher clear, and a
  Couch cold-resume birth wait, added after the sweep. The probe gained a
  `-hammer` mode to measure the Couch-style poll. The Done-when is unchanged.
