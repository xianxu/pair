---
id: 000420
status: working
deps: []
github_issue:
created: 2026-10-09
updated: 2026-10-10
estimate_hours:
card_mirror: '72a01b48577f865a7b6570770aa8d6096f9269ae' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-10T17:48:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "00404941", done: "e14f1eab"}
---

# idle sessions burn CPU: pair title / session-watch / couch poll forever

## Problem

Idle sessions burn CPU. Reported by ariadne:1 (TL) via Couch, 2026-10-09:

- 27 `pair title` and 5 `session-watch` processes poll forever.
- The top title process used 104 CPU minutes over 8 idle days.
- Each `session-watch` uses about 30 CPU minutes per day.
- The couch server sits at 20–40% CPU and peaks above 100%.
- Under that load, couch disconnected when Ghostty stalled for 5s.

Operator idea: start polling only after real user input, when a transcript is
expected, and back off when idle.

Measurements from pair:4 for ops:0, 2026-10-10 (ops learnings kink 54):

- `session-watch` pids 7939 (astro) and 9608 (brain-slot1) each burn about 2s
  of CPU in bursts, about 200 CPU-min in 44h. That is about 110 per day, not
  30.
- The astro watcher's transcript has not changed since Oct 8, so it never
  binds.
- Each scan re-observes the whole native inventory: 3355 transcripts, 1.4G in
  `~/.claude/projects` and 4.1G in `~/.codex/sessions`.
- `pair title` processes 38919 and 38763 have 115 and 70 CPU-min. There are
  about 87 pair processes.

Root causes (pair:3, 2026-10-10, measured on main 4c147e5d):

1. **Orphaned title pollers.** `titlepoller` `SessionAlive` reads
   `zellij list-sessions --short`. That listing still names EXITED
   (resurrectable) sessions, so a poller never sees its session end.
   📁1-43, 1-44, 1-51, 1-67 and 1-69 have been EXITED for 5–9 days, and
   their pollers (38919, 38763, 97082, 96389, 96206) still tick.
2. **Unbound watchers validate the whole machine's new transcripts.** The
   astro watcher's launch baseline is 3150 artifacts. 452 transcripts written
   since then by *other* sessions are "new launch" candidates. Each one is
   validated (a cold pass costs 64s) and tracked forever. The cost grows with
   fleet activity, at one burst of about 1.8s CPU per 60s SlowPoll. Yet
   correlation needs a Pair-log turn past the launch's `PairLogOffset`
   (`RoundsAfterLaunch` → `QualifyTurnSequence`), and the astro Pair log has
   not changed since Sep 18. None of that work could ever bind.
3. **Bound-session reads walk the whole native root.** Each title tick calls
   `SessionActivity`. That is `QuerySession` → `ValidateBindingProof` →
   `Observe`, plus `ActivityForSession` → `ListFiles`: two full walks, about
   1.9s wall, to read one transcript's mtime. `ContextCount` adds about 2s
   while the session is active. Measured: 0.3–0.5s CPU per 60s tick per
   session. Couch's idle fading (`threadactivity`) shares the same path.
4. **The couch server's 20–40% CPU is not idle polling.** A sample shows pty
   ingest, VT emulation (`Endpoint.capture`/`Feed`) and presenting for agents
   that are actively writing. It is out of scope here; see the Revisions entry.

## Spec

Make each poller's cost track what can change. Idle threads should cost a
stat, not a walk.

- **Title-poller funeral (root cause 1).** Session liveness excludes EXITED
  rows. The parser for `zellij list-sessions --no-formatting` rows moves from
  `launcher` (private `sessionRowState`) to a new pure package
  `cmd/internal/zellijsession`. `launcher` and `titlepoller` both use it
  (ARCH-DRY; `launcher` imports `titlepoller`, so the poller cannot import
  `launcher`). The existing `MissThreshold` debounce still governs the exit.
- **Named-artifact observation (root cause 3).** Add
  `Runtime.StatFile(Artifact) (FileEntry, error)`. `OSRuntime` builds the entry
  with the same per-file code `ListFiles` uses, which is extracted
  (ARCH-DRY). `ValidateBindingProof` observes only the proof's artifacts, and
  `ActivityForSession` stats only the root transcript, instead of walking the
  root. `TargetEstablished` selection already discards every other artifact,
  so the results are unchanged; only the cost moves from O(store) to
  O(proof).
- **Send-gated unbound watcher (root cause 2).** Before a Pair-log turn exists
  past the effective launch's `PairLogOffset` (the log size is at most that
  offset), the watcher skips native validation and round matching, because
  nothing can correlate. It keeps the metadata paths: epoch acquisition and
  the chosen-id filename handshake. The handshake walks once, then again only
  after the agent pane's recording (`ScrollbackRaw`) moves (review round 1).
  `waitForScan` already wakes within `ActivePoll` of a send.
- **mtime floor on candidates.** Once a turn is pending, an untracked
  candidate is validated only if its mtime is at or after the last moment the
  watcher saw no turn, minus one minute of slack. A round's native events are
  written after its turn, so an older artifact cannot hold one. Its next write
  makes it a candidate again. This keeps the first send after a long idle
  from cold-validating every transcript on the machine.
- **Idle expiry (ARCH-FUNERAL).** An unbound watcher exits after
  `IdleExpiry` (24h) with no observed send. The window starts at
  `max(watchStart, lastSend)`. Trade-off, accepted per ops:0's direction: a
  thread idle that long and then used gets no binding until its next relaunch
  (Alt+N), which spawns a fresh watcher.

ARCH-FUNERAL lines:

- The title poller ends with its session, now including an EXITED session.
- The session watcher ends at binding, agent exit, the startup deadline (no
  PID) or idle expiry.
- The tracked-candidate map dies with the watcher. While the watcher waits
  for a turn it stays empty.
- Nothing durable is added.

## Done when

- An idle session (no operator input, no agent output) costs near-zero CPU in
  `pair title` and `session-watch`:
  - An unbound watcher with no pending Pair turn does no validation.
  - Its only native walks are a chosen-id launch's first handshake walk and
    one after each burst of agent output.
  - A title tick reads its bound session by stat, not a store walk.
  - Both are measured by operation-counting tests and a live probe.
  - The couch server's CPU is endpoint emulation of live output, not idle
    polling (see Revisions).
- Polling resumes promptly after real user input: a send wakes the watcher
  within `ActivePoll`, and a turn pending after a long idle still binds
  (tested).
- Process count is bounded (ARCH-FUNERAL):
  - A title poller exits when its session is EXITED or absent.
  - An unbound watcher exits after `IdleExpiry` without a send.
- ~~Couch stays connected through a multi-second terminal stall.~~ Covered by
  pair#430 (see Revisions).

## Plan

- [x] `zellijsession.RowState`/`Live`, a pure parser with table tests,
  including an EXITED row; `launcher` delegates to it; `titlepoller`
  `SessionAlive` is a thin exec wrapper over `Live`. The exit after
  `MissThreshold` misses is the poller's existing, unchanged loop, so no new
  poller test was needed.
- [x] `Runtime.StatFile` on `OSRuntime`, sharing per-file entry construction
  with `ListFiles`, plus the fake runtime and the test runtimes.
- [x] `ValidateBindingProof` and `ActivityForSession` use named-artifact
  observation. Test: a counting fake shows no `ListFiles`, and the results
  match the walk.
- [x] `sessionwatch`: send gate, mtime floor and `IdleExpiry`, each pinned by
  a fake-clock test. With no send: no validation, and an exit after expiry.
  A send past the offset still binds. An artifact older than the floor is
  never validated.
- [x] Live probe of the new code against the real stores (see Log).
  Post-land rollout is handed to ops:0: `make build` on `pair:0`, then
  relaunch. Running pollers keep the old binary; orphaned title pollers exit
  once they are restarted under the new binary.

## Log

### 2026-10-09

- Filed only (not implemented) at ariadne:1's request.

### 2026-10-10

- Claimed by pair:3 on ops:0's dispatch. Diagnosis is in Problem (root causes
  1–4). Probes: `sample` on 7939 (lstat, fstatat and open dominate), a
  70-second `ps` CPU-delta trace, and throwaway timing tests replaying one
  astro scan and one title tick against the real stores.
- Implemented. Tests pin each mechanism, and each was mutation-checked
  (reverted or disabled, the test fails):
  - `TestRunUnsentWatcherSkipsNativeStoreAndExpires`: with the gate forced
    open, the store is walked 262 times.
  - `TestRunPendingTurnSkipsCandidatesOlderThanQuietFloor`: with no floor,
    the foreign transcript is validated.
  - `query_test` and `activity_test`: with the old code, `ListFiles` is
    called once.
  - `zellijsession` table tests.
- An existing harness gap surfaced: `OperationCountForRoot` matches a `root:`
  prefix, but `ListFiles` records the bare root name, so a root-scoped
  `ListFiles` count is always 0. The tests use `OperationCount(op, root)`.
- Live probe with the new code against the real stores:
  - `SessionAlive(📁1-43)` is false; that session has been EXITED for 9 days.
  - `SessionActivity` fell from 1.94s to 0.84s wall.
  - `ContextCount` fell from 2.04s to 0.59s.
  - pprof: the remainder is 74% `sessionledger.ParseLedger`. Launch records
    embed a whole-store baseline (3150 entries per launch), filed as pair#431.
- The diff (269 added code lines) leaves the quick-flow shell, so close runs
  the full review.
- Close review round 1 (FIX-THEN-SHIP):
  - **Important (fixed).** A chosen-id launch (every fresh Claude) still
    walked the store each SlowPoll while idle. It now walks once, then only
    after `ScrollbackRaw` moves. `TestRunIdleChosenWatcherWalksOnlyAfterAgentOutput`
    counts one walk in 3h; with the gate disabled it counts 190.
  - **Minor: Plan item 1 claimed a poller test (fixed).** The item now names
    what shipped: the pure `Live` table test, and the existing loop does the
    exiting.
  - **Minor: idle pass re-read the whole Pair log (fixed).** An unchanged log
    mtime since the last quiet read skips the read.
  - **Minor: loop-state flags, ARCH-ORDER (deferred).** Naming the phases
    (epoch, quiet, pending, bound) as an enum restructures the whole loop,
    which is a refactor beyond this load fix. Recorded here for the next
    change to the watcher.
  - **Note: the on-disk catalog is not cleared on a quiet pass (no change).**
    `clear` empties only the in-memory tracking. The catalog merges published
    validations and never authorizes a binding by itself, so stale entries
    are harmless.
- Close review round 2 (FIX-THEN-SHIP):
  - **BR-1 Important: no conformance test for the real `StatFile` (fixed).**
    `TestOSRuntimeStatFileMatchesListFiles` (temp dir) checks five cases:
    - A regular file yields the same entry as `ListFiles`.
    - A missing file is `os.ErrNotExist`.
    - A directory is `ErrNotRegularArtifact`.
    - A symlink is refused, and is not mistaken for missing.
    - An unknown root is `ErrRootUnknown`.
  - **Minor: idle expiry counted only sends (fixed).** Agent output
    (`ScrollbackRaw` mtime movement after launch) now counts as activity.
    `TestRunAgentOutputDefersIdleExpiry` exits at about 44h with output at
    20h, and at 24h with output ignored.
  - **Minor: symlink parity of walk versus stat (no change; parity holds).**
    The old walk refused a symlinked transcript with a `ListingIssuesError`.
    When the root transcript was not among the regular files, it returned
    that error, and `StatFile` returns an error too. The new conformance test
    pins both refusals.
  - **Minor: `OperationCountForRoot` stays vacuous (fixed in the helper).**
    The count now includes root-level keys.
    `TestOperationCountForRootIncludesRootLevelOperations` fails without the
    fix.
  - **Minor: ARCH-ORDER phase enum (deferred, as in round 1).**
- Not changed: the bound Codex lifecycle follower still observes the store
  each pass. Measured at about 5 CPU-min per 44h (pid 26613), so not a
  hotspot.

## Revisions

### 2026-10-10 — live Plan item split (pair:3)

- The fleet rollout cannot happen before landing: running pollers hold the old
  binary. The Plan item now covers the pre-land live probe. The rollout is a
  handoff to ops:0.

### 2026-10-10 — scope (pair:3, settled by ops:0 with pair:4)

- The last Done-when line, "Couch stays connected through a multi-second
  terminal stall", is **covered by pair#430**. That is the presenter
  write-path resilience fix in `cmd/internal/terminal` and `couchtty`. This
  issue keeps to idle polling and process bounds.
- In the first Done-when line, the couch server is not an idle poller. Its
  measured CPU is endpoint emulation of live agent output (root cause 4). This
  issue reduces the load couch shares with the pollers through
  `threadactivity`'s named stat, and leaves the presenter alone. Couch-side
  ticker cost on a fully idle fleet is unmeasured here and goes to a follow-up
  issue if it shows up after this lands.
