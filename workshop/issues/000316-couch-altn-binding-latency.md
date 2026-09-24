---
id: 000316
status: working
deps: []
github_issue:
created: 2026-09-23
updated: 2026-09-23
estimate_hours:
started: 2026-09-23T21:53:28-07:00
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

- [ ] Pure `scanDelay(now, watchStart, lastSend, opts)` + table tests
- [ ] `ActivePoll`/`ActiveWindow` options + defaults in `applyWatcherDefaults`
- [ ] Loop: track Pair-log ModTime → `lastSend`; sliced wait that wakes on change
- [ ] Fake-clock regression test: send after startup window → binds within ActivePoll
- [ ] `make test`; live smoke via operator

## Log

### 2026-09-23

- Proof = the Pair-log send text's normalized sha256 matches a transcript
  operator turn, followed by agent progress and corroborated by the live agent
  process (`sessioninventory.QualifyTurnSequence`, `sessionwatch.Run`). Couch
  only reads the resulting ledger `binding` row (`QuerySessionContext`).
- Timing evidence: ledger + config mtimes both 21:49:04, agent pid file born
  21:47:03 → matches the 60 s + 60 s cadence exactly.
