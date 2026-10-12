---
id: 000489
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '354ee80bc31c1d55eebfc232a943c774b8b46abf' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T10:28:10-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "b5eb0429", done: "f4ca745b"}
---

# couch: relaunch/resume fails under load (pane not born in 15 s) and leaves the thread parked; make startup resilient

## Problem

10-12 08:3x, load average ~200: `couch --relaunch` failed for ariadne:0, brain:0, parley.nvim:1, parley.nvim:2 and pair:0; each is left parked (conversation kept). A later `couch --resume pair:0` failed the same way: `await Pair registration …: agent pane not born — context deadline exceeded [pane-birth wait consumed the budget: pane birth 15.002s …] (NO Pair session is live. Pair never started, or exited before registering)`. Relaunch parks first (the old agent is stopped on purpose) and then cold-starts a new wrapper on the current binary; with PAIR_DEV set it also rebuilds pair. On a loaded machine the new wrapper (and Claude Code) can take longer than a fixed 15 s to register, so the operation fails and the slot is left without a live agent. The operator: startup should be resilient; Couch's own share of the work should be small, so why a timeout even on a hot machine?

## Spec

Investigate, then fix at the root:
1. **Where the time goes** on a loaded machine: the pair rebuild (PAIR_DEV), process spawn, the agent's own startup, registration. Measure each step (the error already has a step breakdown; extend it).
2. **Whether waiting longer works:** does the wrapper eventually register after the 15 s budget? If yes, the budget is the bug: wait on the registration event with a generous hang ceiling and progress reporting, the same rule as the test sweeps (#480): no fixed short window for work whose duration depends on load.
3. **What a failure leaves behind:** if the new wrapper starts late, Couch must adopt it rather than leave the thread parked with a wrapper running unclaimed (or kill it cleanly). Never a state the operator has to puzzle out.
4. **The rebuild:** relaunching many slots shouldn't rebuild pair once per slot; build once per sweep (or not at all when the binary is current).
5. A relaunch sweep could check load and pace itself (one at a time, waiting for each), instead of the TL guessing.

## Done when

- The cause of each 08:3x failure is explained in the Log with measurements.
- Under a synthetic load (e.g. CPU burners matching ~200 load), relaunch and resume succeed, possibly slowly, with progress shown; a genuinely dead launch still fails clearly and leaves a clean state.
- Tests for the adopt-late and clean-failure paths.

## Plan

Root cause: one fixed 15 s window (`pairRegistrationTimeout`) covers ALL of Pair's startup, including PAIR_DEV's
`make build`. Both scale with load, and at load ~200 their sum (~20 s measured) passes 15 s. Fix (quick flow):

- [x] Registration waits on Pair, not a stopwatch: the wait ends at registration, at the helper's process group
  exiting (a dead launch fails at once, named "Pair exited before registering"), or at a 2 min hang ceiling. The
  existing post-ack cleanup stays the one failure path (kills the group, rolls back to parked): a clean state.
  Late starts inside the ceiling are adopted by the same wait.
- [x] Couch-launched Pair skips `dev_rebuild`; relaunch's binary-freshness gate applies under PAIR_DEV too
  (refuses the binary already running: "run make build in <checkout>"), so a sweep builds once, by hand.
- [x] CLI progress: `couch --resume/--relaunch` prints a "still running" line every 15 s while it polls.
- [x] Pacing: no load check (load average misreads, see memory); each CLI call waits for its launch, so a sequential
  sweep paces itself by real launch time.
- [x] Tests: late registration past the old window succeeds (injected delay); helper exits before registering →
  fast, named failure + clean rollback; DevRebuild skipped for Couch-launched Pair; freshness under PAIR_DEV; CLI
  progress line.

## Log

### 2026-10-12
- **08:3x measured (ops:0 transcript, pair:0 resume 08:40):** `claim 4.0s, checkout 61ms, prepare 1.4s, spawn 81ms,
  record+baseline 2.0s, ack 0s, registration 15.0s`, pane never born, no session live. Couch's own steps before the
  window took ~7.5 s (not in the budget); the remote job's prepare (inventory) took 9–16 s, also outside it.
- **zellij log 08:32–08:38:** only three new servers started (08:32:26 pair:2 OK, 08:36:53, 08:37:52); the other
  failed relaunches never reached zellij: Couch's deadline ended Pair while it was still before its zellij launch
  (dev_rebuild + startup).
- **Synthetic run (one, 90 s, 200 self-bounded spinners; TL then forbade real load — injected delay from here):**
  no-op `make build` 9.6 s (1.0 s at load ~12); `pair resume` → pane birth (probes/zellijbirthrace, no rebuild)
  9.3–11.1 s (0.9–1.1 s at load ~12). Sum ≈ 20 s > 15 s: the budget, not a launch bug. All three trials were born.
- 10:27, load ~10: TL's `couch --resume` of the same five slots all succeeded (corroborates).
- Residual risk noted, not changed: Pair's own birth watch probes zellij at 10 s and aborts only on proof of death
  (session not listed); birth at 11.1 s still passed in the loaded run.
- **Implemented:** `whileHelperAlive` (launch_existing.go) watches the helper's process group (both runners'
  `Alive` is `kill(-pgid,0)`) under the deadline, so a test's custom deadline context keeps synchronous
  cancellation (the first ordering broke TestColdResumeRegistrationTimeoutNamesTheOwnershipPhase: a custom parent
  makes WithCancelCause propagate on a goroutine). `pairRegistrationTimeout` 15 s → 2 min ceiling (ARCH-CONSTRAINTS:
  a hang ceiling, with the unbounded-work caveat kept). Couch launches skip `dev_rebuild` (`Env.CouchHosted`);
  `BinaryFacts.DevRebuild` removed, so relaunch's freshness gate applies under PAIR_DEV. CLI prints
  "still running after Ns" every 15 s. ARCH-FUNERAL: creates nothing durable (the watcher goroutine is joined by
  stop). Injected-delay tests: TestColdResumeAdoptsALateStart, TestColdResumeFailsAtOnceWhenPairExitsBeforeRegistering
  (also asserts the thread is left parked and unoccupied), TestCreateRebuildsOnlyOutsideCouch,
  TestSlotOperationCLIReportsProgressWhileRunning.
- **Full suite caught:** TestResumeRunsAsTheNewLiveOwner — its fake helper is already gone when the wait starts,
  with registration already written; the exit cancelled the wait before its first look. Rule: an exit ends the wait,
  then one bounded final look (`exitedFinalLook`, 1 s: nothing more can arrive) decides, so registration already
  written still counts and a dead launch still fails promptly.
- **Live evidence from TL (pair:1 timing parley.nvim:3, load 11–17):** relaunches succeeded at 15.1 s and 22.3 s;
  runs 3 and 4 ended *uncertain* at 12 s and 24 s (CLI context deadline), and run 4 left the slot parked; a resume
  then failed: `claim 3.6s, prepare 0.7s, record 1.0s, registration 15.0s`, pane not born, PAIR_DEV rebuilt every run.
  Second fixed window found: every status poll had a 2 s exchange timeout (`AdmissionTimeout`), so ONE late answer
  from a Couch busy launching ended the CLI uncertain while the operation went on. Folded in (same rule): a timed-out
  status poll is asked again; only the 5 min poll budget gives up. Dial errors, an unavailable caller, an unknown
  receipt stay immediately uncertain. Test: "per-call timeouts mid-poll are retried" / "timeouts until the budget
  are uncertain" (TestSlotOperationCLIPollLoop).
- parley.nvim:3 inspected: parked, no live actor, no Pair/zellij process with its cwd: the old cleanup left a clean
  state. Not resumed yet: the fix lives in the Couch process, so the resume is the live check after Couch restarts
  onto this build.

## Revisions

- 2026-10-12 10:35 (TL, ops:0): no real machine load (the one 200-spinner run froze the operator's Couch for a
  minute). The Done-when's "synthetic load matching ~200" line is satisfied by an injected-delay test plus the 10:27
  evidence (all five resumes succeeded at load ~10).
