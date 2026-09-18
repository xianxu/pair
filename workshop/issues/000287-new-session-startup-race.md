---
id: 000287
status: working
deps: []
github_issue:
created: 2026-09-18
updated: 2026-09-18
estimate_hours:
started: 2026-09-18T13:38:02-07:00
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

- [ ] Commit the repro driver (bare socket poke + standalone-launch counter) to
      `probes/`.
- [ ] Title poller: pane-evidence startup gate + unit test.
- [ ] Sweep other launch-time zellij callers against the rule.
- [ ] Live verification (repro driver 0/N; astro via Couch).
- [ ] File the upstream zellij issue.

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
