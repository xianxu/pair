---
id: 000316
status: working
deps: []
github_issue:
created: 2026-09-23
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T21:53:28-07:00
flow: {kind: quick, provenance: inferred, spec: "ca6d9b71", done: "6d56dec0"}
---

# Couch Alt+n refuses a fresh thread for up to 60s after its first round

## Problem

A thread stays **provisional** (no ledger `binding` row) for up to 60 s after
its first round has already completed. During that gap Couch's Alt+n refuses
("its agent has not completed a turn yet…") although the turn is done, and
every other consumer of the binding (Couch's parked-row resume, the saved
`config-<tag>-<agent>.json`) lags the same way.

Observed on `couch-b929512be2acbf12` (2026-09-23, PDT): launch ≈21:47:03,
first send 21:48:01, round completed shortly after, binding + config written at
**21:49:04**, exactly one `SlowPoll` after the 60 s startup window closed at
≈21:48:03.

## Spec

Root cause: `sessionwatch.Run`'s cadence is anchored to **launch** time
(`Timeout` 60 s of 100 ms polls, then `SlowPoll` 60 s blind sleeps, from #143).
A proof can only appear after (a) a Pair-log send past the launch's
`PairLogOffset` and (b) the agent finishing that round. Both happen at arbitrary
times after launch, so anchoring to launch is the wrong clock.

Fix: anchor the fast cadence to the causal prerequisite, the **send**.

- The watcher treats Pair-log growth (its `ModTime` via the existing
  `Runtime.ModTime` seam, so no new IO surface) as the wake signal.
- Pure cadence policy (ARCH-PURE): given `now`, `watchStart`, `lastSend`:
  `Poll` within `Timeout` of `watchStart` (unchanged); `ActivePoll` (1 s) within
  `ActiveWindow` (30 min) of the last observed send; else `SlowPoll`
  (unchanged, #143's delayed-discovery backstop).
- Slow waits are sliced into `ActivePoll` steps that end early when the Pair
  log changes, so a send made during a slow period switches back to the fast
  cadence within ~1 s instead of up to 60 s later.
- Cost envelope (ARCH-CONSTRAINTS): one scan is a metadata walk of the agent
  store (measured ≈30 ms over 3.2k files) plus incremental parsing of changed
  transcripts. The 1 s rate applies only between a send and the first binding,
  and the Claude watcher exits once bound. That is 10× cheaper than the
  existing 100 ms startup rate.
- Proof rules are unchanged: same qualification, corroboration and ledger
  append. Only *when* the watcher looks changes (ARCH-DRY: no second resolver,
  no on-demand scan in Couch).

Out of scope, noted: plain-Pair `runRestart` picks `LatestLedgerEntryForAgent`
without regard to launch, so in the same window it may name the *previous*
conversation's session id. Verify separately.

## Done when

- A watcher whose startup window expired before the first send binds within
  ~`ActivePoll` of the round completing, not after a `SlowPoll`. A fake-clock
  test pins this and fails against the old cadence.
- With no send, the watcher still slow-polls (the #143 behavior test still
  passes), and a send during a slow wait wakes it early (tested).
- The cadence policy is a pure function with table tests.
- `make test` green; live check: new Couch thread, one round, Alt+n within a
  few seconds of the reply relaunches instead of refusing.

## Plan

- [x] Pure `scanDelay(now, watchStart, lastSend, opts)` + table tests
- [x] `ActivePoll`/`ActiveWindow` options + defaults in `applyWatcherDefaults`
- [x] Loop: track Pair-log ModTime → `lastSend`; sliced wait that wakes on change
- [x] Fake-clock regression test: send after startup window → binds within ActivePoll
- [x] `make test`; live smoke via operator

## Log

### 2026-09-23

- Proof = the Pair-log send text's normalized sha256 matches a transcript
  operator turn, followed by agent progress and corroborated by the live agent
  process (`sessioninventory.QualifyTurnSequence`, `sessionwatch.Run`). Couch
  only reads the resulting ledger `binding` row (`QuerySessionContext`).
- Timing evidence: ledger + config mtimes both 21:49:04, agent pid file born
  21:47:03 → matches the 60 s + 60 s cadence exactly.
- 855e65b3: `scanDelay` (pure) + `waitForScan` (sliced, wakes on a Pair-log
  ModTime change) in `sessionwatch/run.go`; baseline mtime taken at
  `watchStart`, so a send during the PID wait still counts. ARCH-PURE: policy is
  table-tested, and the glue reuses the existing `Runtime.ModTime`, so no new seam.
- Mutation check: against the old cadence the regression test fails with
  "bound 54.5s after the round completed", which matches the ~60 s seen live.
  The first draft injected the send from the sleep hook (on the poll grid) and
  PASSED against the old code; it was fixed by stamping the event at `sendAt`
  (lesson added).
- Verification: `go test ./... -count=1` green (74 pkgs); `make -k test` rc=0 with
  the five-var + COUCH_* scrub, sandbox off, TMPDIR=/private/tmp/p316. The
  scratchpad TMPDIR is too long for nvim sockets, and /tmp (a symlink) trips
  test-changelog. Earlier full runs flaked once each in
  workbench-route-nvim / submission-transaction under load avg ~7.6; both pass
  in isolation.


- Operator confirmed the #316 live smoke test passed on 2026-09-23. This completes the manual verification item alongside the automated verification recorded above.
