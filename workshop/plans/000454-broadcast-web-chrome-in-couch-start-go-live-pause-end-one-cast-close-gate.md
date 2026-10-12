---
gate: boundary-review
issue: 454
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-11T20:38:54-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: Tasks 1/2/3/7/8 list test cases in prose; cut each to the function name plus one strategy line per risky function
          detail: |-
            e.g. splitFrames: fuzz seeded with spike captures and truncated or oversized frames; Tracker: generated input sequences against a stated active-tab invariant.
            (carried from plan-quality PQ-3, deferred to the boundary review)
          family: test-plan-enumerates-cases
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-10-11T20:38:54-07:00"
      dispatched: "2026-10-11T20:36:10-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: Chrome.Err() reports a cdp Conn failure (ErrBacklog, ErrFrameTooLarge, malformed) as ErrClosedByOperator
          detail: chrome.go:405-416 default branch ignores c.conn.Err(); after a conn failure Chrome exits 0 on pipe close, so the cause is lost, contrary to the plan's never-silently-dropped promise. Prefer a non-EOF, non-ErrClosed conn error; add a regression test.
          family: failure-cause-misattributed
          round: 2
        - id: BR-3
          severity: Important
          title: Chrome run-loop transitions (end cause precedence, setLive while ending, noPages cancel, Err mapping) have no default-run tests (ARCH-ORDER/ARCH-PURE)
          detail: Only the env-gated live test reaches chrome.go's loop. Extract the transitions to return effects, or build Chrome over an injected Conn plus process handle, then sequence-test against a scripted pipe peer.
          family: state-machine-untested-sequences
          round: 2
        - id: BR-4
          severity: Important
          title: cmd.Wait without WaitDelay can block forever on inherited stderr, hanging Close via finish's Kill then wait on exited
          detail: chrome.go:139-164, 397-402. Stdout/Stderr are a non-file writer, so Wait waits on helper processes holding the pipe. Set cmd.WaitDelay.
          family: unbounded-process-reap
          round: 2
        - id: BR-5
          severity: Important
          title: Tracker keeps the previous document's pageTitle after navigating to an uninjectable page (chrome://, chrome-error://)
          detail: tracker.go:86-89 updates URL but not pageTitle, so Active() pairs the new URL with the old title and viewers would see the wrong title. Clear pageTitle when the URL changes; add a tracker case.
          family: stale-derived-state-on-navigation
          round: 2
        - id: BR-6
          severity: Important
          title: launchError maps every exit 21 to ErrProfileInUse and drops Chrome's output
          detail: 'Reproduced: with a fresh empty profile, Chrome exits 21 when it cannot create its ProcessSingleton socket dir, and every live test reported profile in use. Quote c.output in the error and match the singleton text, not the code alone.'
          family: failure-cause-misattributed
          round: 2
        - id: BR-7
          severity: Minor
          title: Tracker.Active picks among equal activated values by map iteration order, so the published tab can flip
          family: nondeterministic-tie-break
          round: 2
        - id: BR-8
          severity: Minor
          title: Chrome.SetLive while ending returns nil with no effect; Browser contract says ErrEnded
          family: contract-drift
          round: 2
        - id: BR-9
          severity: Minor
          title: FakeBrowser does not model the PageTitle path or noPagesGrace, so title and close-grace bugs cannot show up against the fake
          family: fake-fidelity
          round: 2
        - id: BR-10
          severity: Minor
          title: Plan needs a Revisions entry for the M1 divergences
          detail: publisher[T] replaces latest[T]; PageTitle input added; setAutoAttach replaces per-target attach; LaunchOptions changed; live harness headless; node test path moved.
          family: plan-drift
          round: 2
      boundary: M1
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 3
      timestamp: "2026-10-11T20:47:01-07:00"
      dispatched: "2026-10-11T20:43:34-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: The Revisions Test strategy entry adds the per-function strategy lines (splitFrames fuzz, Tracker property test), and FuzzFrameReader and the tracker rand test exist.
          round: 3
        - id: BR-2
          disposition: addressed
          note: endError reads the conn error; dropping that branch fails TestEndError (backlog row). A related race is raised separately.
          round: 3
        - id: BR-3
          disposition: addressed
          note: process seam plus scripted peer in chrome_loop_test.go; the SetLive-while-ending mutation goes red. One of its tests is flaky (raised separately).
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: WaitDelay set at chrome.go:155 but removing it fails no test; add an sh 'sleep 30 &' osProcess test.
          round: 3
        - id: BR-5
          disposition: addressed
          note: DocumentChanged clears pageTitle; removing the clear fails TestTrackerURLTitleFollowChanges.
          round: 3
        - id: BR-6
          disposition: addressed
          note: profileInUse matches Chrome's output, TestProfileInUse covers a look-alike exit-21 failure, and the output is quoted via endError.
          round: 3
        - id: BR-7
          disposition: not-addressed
          note: Tie-break by id implemented at tracker.go:121-125, but no test covers equal activated values.
          round: 3
        - id: BR-8
          disposition: addressed
          note: setLive returns ErrEnded; the mutation fails TestChromeEndCausePrecedenceAndKill.
          round: 3
        - id: BR-9
          disposition: not-addressed
          note: 'fake.go unchanged: no PageTitle/DocumentChanged and no no-pages grace.'
          round: 3
        - id: BR-10
          disposition: addressed
          note: The "M1 as built" Revisions entry lists publisher, PageTitle, setAutoAttach, LaunchOptions, headless harness, and node test path.
          round: 3
      findings:
        - id: BR-11
          severity: Important
          title: cdp reader stops draining after a malformed or oversized frame, despite its comment; lost-connection loop test flakes 4 of 10 runs
          detail: 'conn.go:197-199 returns from read, so the peer (and real Chrome) blocks writing, never sees its command pipe close, and is SIGKILLed after closeWait (5s), which races waitDone''s 5s. Rule: every Conn doc-comment promise is pinned by a test. Keep draining after every failure kind, test that the writer never blocks, and set test deadlines from closeWait.'
          family: contract-drift
          round: 3
        - id: BR-12
          severity: Important
          title: 'A clean operator quit can be reported as "lost Chrome": the Wait goroutine''s outR.Close yields os.ErrClosed, which endError treats as a conn failure'
          detail: '3rd in family. Rule: an end cause is only what was observed before our own teardown began; errors teardown can produce (outR.Close, conn.Close, EPIPE after exit) are never causes. Have the loop record whether the event stream closed while the process still ran, and pass that to endError. Reproduced: endError(notEnding, "read |0: file already closed", 0) returns "lost Chrome". Add a peer exit mode that closes the reader side.'
          family: failure-cause-misattributed
          round: 3
        - id: BR-13
          severity: Minor
          title: Chrome.exited field (chrome.go:71) is never set or read after the process seam refactor
          family: dead-code
          round: 3
      boundary: M1
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 4
      timestamp: "2026-10-11T20:51:55-07:00"
      dispatched: "2026-10-11T20:49:39-07:00"
      agent: claude
      dispose:
        - id: BR-4
          disposition: addressed
          note: startProcess sets WaitDelay (chrome.go:205); with it reverted, TestStartProcessReapsDespiteInheritedOutput times out at 5s.
          round: 4
        - id: BR-7
          disposition: not-addressed
          note: Tie-break by id is in place (tracker.go:121-125), but reverting it leaves Tracker/Fake/Conformance green over 20 runs. Add a tracker test with 3+ background-only hidden pages and call Active repeatedly.
          round: 4
        - id: BR-9
          disposition: withdrawn
          note: Gap accepted and documented (fake.go CloseTab doc, issue line 91); the scripted-peer loop tests cover the real PageTitle/grace paths through production code.
          round: 4
        - id: BR-11
          disposition: addressed
          note: Drain after oversized/malformed (conn.go:195,204); with it reverted, TestReaderDrainsAfterProtocolFailure fails. Exit-reason tests pass 40/40 under -race.
          round: 4
        - id: BR-12
          disposition: addressed
          note: endError accepts only cdp.ErrProtocol; with the deny-list restored, TestEndError rows fail every time and the closeReader loop subtest fails on some runs.
          round: 4
        - id: BR-13
          disposition: addressed
          note: Chrome.exited field removed (chrome.go:69-73); package builds and vets clean.
          round: 4
      boundary: M1
      recipe: milestone-review
      isolation: checkout
      reviewed: 6c86576ebc04593f77579108ec0e4aa33dc9dfd7
      blocked: false
    - "n": 5
      timestamp: "2026-10-11T21:46:48-07:00"
      dispatched: "2026-10-11T21:38:05-07:00"
      agent: claude
      findings:
        - id: BR-14
          severity: Important
          title: Host.Mount cancel racing open completion leaves the Host open with no mounts or waiters (tunnel leaked)
          detail: 'host.go:113-130 — open() wins mu with waiting==1 and sets attempt=nil, then the cancelled waiter decrements waiting to 0 with last=false and returns; reproduced 1/300 with OpenDelay≈ctx timeout. Rule: every loss of interest (open done, waiter cancel, Unmount) runs one "nobody wants it" helper that closes the Host; add the planned gate seam that holds open() after start() so the test can cancel at that point and assert Opens−Closes==0. 2nd finding in this family.'
          family: state-machine-untested-sequences
          round: 5
        - id: BR-15
          severity: Minor
          title: webcast.Server.Owns compares the token with == instead of the shared constant-time ownsToken
          detail: webcast/server.go:47; plan PQ-1 specifies a constant-time compare per mount and broadcast.Server already uses ownsToken. Export it and use it in both.
          family: capability-token-compare
          round: 5
      boundary: M2
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 6
      timestamp: "2026-10-11T21:54:26-07:00"
      dispatched: "2026-10-11T21:49:21-07:00"
      agent: claude
      dispose:
        - id: BR-14
          disposition: addressed
          note: releaseIfUnwantedLocked is the single release path; TestHostLastWaiterGivingUpAfterTheOpenCloses fails 3/3 against pre-fix host.go (Opens 1 Closes 0) and passes at head.
          round: 6
        - id: BR-15
          disposition: addressed
          note: OwnsToken exported (host.go:355) and used by broadcast.Server (server.go:87,115) and webcast.Server.Owns (webcast/server.go:49).
          round: 6
      findings:
        - id: BR-16
          severity: Important
          title: A Mount arriving during an abandoned open joins it and returns another caller's context.Canceled
          detail: 'host.go:102 counts a new arrival on any hostOpening attempt, including one releaseIfUnwantedLocked already cancelled (:160). Reproduced: a tunnel that takes 100ms to wind down after cancel (as cloudflared''s SIGTERM grace does) makes a background-context Mount fail with "context canceled". 4th in family. Rule: a Mount''s error must come from its own request, never from an attempt another caller abandoned. Mark abandoned attempts (flag or hostAbandoning state), have Mount wait uncounted and reopen, add FakeTunnel.CancelDelay plus a regression test.'
          family: failure-cause-misattributed
          round: 6
        - id: BR-17
          severity: Minor
          title: BR-14 regression test drives abandonWait directly instead of through Mount via a gate seam
          detail: 'host_test.go:255 bumps h.waiting by hand and calls abandonWait, so a Mount ctx.Done branch that stopped calling it would be caught only by the probabilistic stress loop. 3rd in family. Rule: Host sequence tests reach a state through public events with an injected ordering seam (hold open() after start()), not by mutating fields.'
          family: state-machine-untested-sequences
          round: 6
      boundary: M2
      recipe: milestone-review
      isolation: checkout
      blocked: true
    - "n": 7
      timestamp: "2026-10-11T22:05:53-07:00"
      dispatched: "2026-10-11T21:57:16-07:00"
      agent: claude
      dispose:
        - id: BR-16
          disposition: addressed
          note: attempt.abandoned makes the attempt unjoinable (host.go:107,165); reverting the guard turns TestHostMountAfterAnAbandonedOpenSucceeds red with "context canceled".
          round: 7
        - id: BR-17
          disposition: not-addressed
          note: host_test.go:259-266 still mutates h.waiting and calls abandonWait directly; no gate seam drives it through Mount.
          round: 7
      findings:
        - id: BR-18
          severity: Minor
          title: Late-cancel and failing-open tunnel behaviours are test-local wrappers, not settings on the shared FakeTunnel
          detail: 'lateCancelTunnel (host_test.go:294) and failingTunnel (:123) model cloudflared behaviour BR-16 depends on, but only in broadcast tests; webcast/couch tests on FakeTunnel cannot exercise it. Rule: every dependency behaviour a test relies on is a FakeTunnel setting (CancelDelay, FailOpens), covered by the live conformance check.'
          family: fake-fidelity
          round: 7
      boundary: M2
      recipe: milestone-review
      isolation: checkout
      reviewed: c0bccd94e72b8c68bc77a9895326ee77ac9772e4
      blocked: false
---

# Gate ledger — pair#454 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-11T20:38:54-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `test-plan-enumerates-cases` Tasks 1/2/3/7/8 list test cases in prose; cut each to the function name plus one strategy line per risky function
  e.g. splitFrames: fuzz seeded with spike captures and truncated or oversized frames; Tracker: generated input sequences against a stated active-tab invariant.
  (carried from plan-quality PQ-3, deferred to the boundary review)

## Round 2 — 2026-10-11T20:38:54-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `failure-cause-misattributed` Chrome.Err() reports a cdp Conn failure (ErrBacklog, ErrFrameTooLarge, malformed) as ErrClosedByOperator
  chrome.go:405-416 default branch ignores c.conn.Err(); after a conn failure Chrome exits 0 on pipe close, so the cause is lost, contrary to the plan's never-silently-dropped promise. Prefer a non-EOF, non-ErrClosed conn error; add a regression test.
- **BR-3** [Important] `state-machine-untested-sequences` Chrome run-loop transitions (end cause precedence, setLive while ending, noPages cancel, Err mapping) have no default-run tests (ARCH-ORDER/ARCH-PURE)
  Only the env-gated live test reaches chrome.go's loop. Extract the transitions to return effects, or build Chrome over an injected Conn plus process handle, then sequence-test against a scripted pipe peer.
- **BR-4** [Important] `unbounded-process-reap` cmd.Wait without WaitDelay can block forever on inherited stderr, hanging Close via finish's Kill then wait on exited
  chrome.go:139-164, 397-402. Stdout/Stderr are a non-file writer, so Wait waits on helper processes holding the pipe. Set cmd.WaitDelay.
- **BR-5** [Important] `stale-derived-state-on-navigation` Tracker keeps the previous document's pageTitle after navigating to an uninjectable page (chrome://, chrome-error://)
  tracker.go:86-89 updates URL but not pageTitle, so Active() pairs the new URL with the old title and viewers would see the wrong title. Clear pageTitle when the URL changes; add a tracker case.
- **BR-6** [Important] `failure-cause-misattributed` launchError maps every exit 21 to ErrProfileInUse and drops Chrome's output
  Reproduced: with a fresh empty profile, Chrome exits 21 when it cannot create its ProcessSingleton socket dir, and every live test reported profile in use. Quote c.output in the error and match the singleton text, not the code alone.
- **BR-7** [Minor] `nondeterministic-tie-break` Tracker.Active picks among equal activated values by map iteration order, so the published tab can flip
- **BR-8** [Minor] `contract-drift` Chrome.SetLive while ending returns nil with no effect; Browser contract says ErrEnded
- **BR-9** [Minor] `fake-fidelity` FakeBrowser does not model the PageTitle path or noPagesGrace, so title and close-grace bugs cannot show up against the fake
- **BR-10** [Minor] `plan-drift` Plan needs a Revisions entry for the M1 divergences
  publisher[T] replaces latest[T]; PageTitle input added; setAutoAttach replaces per-target attach; LaunchOptions changed; live harness headless; node test path moved.

## Round 3 — 2026-10-11T20:47:01-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — The Revisions Test strategy entry adds the per-function strategy lines (splitFrames fuzz, Tracker property test), and FuzzFrameReader and the tracker rand test exist.
- BR-2 — addressed — endError reads the conn error; dropping that branch fails TestEndError (backlog row). A related race is raised separately.
- BR-3 — addressed — process seam plus scripted peer in chrome_loop_test.go; the SetLive-while-ending mutation goes red. One of its tests is flaky (raised separately).
- BR-4 — not-addressed — WaitDelay set at chrome.go:155 but removing it fails no test; add an sh 'sleep 30 &' osProcess test.
- BR-5 — addressed — DocumentChanged clears pageTitle; removing the clear fails TestTrackerURLTitleFollowChanges.
- BR-6 — addressed — profileInUse matches Chrome's output, TestProfileInUse covers a look-alike exit-21 failure, and the output is quoted via endError.
- BR-7 — not-addressed — Tie-break by id implemented at tracker.go:121-125, but no test covers equal activated values.
- BR-8 — addressed — setLive returns ErrEnded; the mutation fails TestChromeEndCausePrecedenceAndKill.
- BR-9 — not-addressed — fake.go unchanged: no PageTitle/DocumentChanged and no no-pages grace.
- BR-10 — addressed — The "M1 as built" Revisions entry lists publisher, PageTitle, setAutoAttach, LaunchOptions, headless harness, and node test path.

### Raised

- **BR-11** [Important] `contract-drift` cdp reader stops draining after a malformed or oversized frame, despite its comment; lost-connection loop test flakes 4 of 10 runs
  conn.go:197-199 returns from read, so the peer (and real Chrome) blocks writing, never sees its command pipe close, and is SIGKILLed after closeWait (5s), which races waitDone's 5s. Rule: every Conn doc-comment promise is pinned by a test. Keep draining after every failure kind, test that the writer never blocks, and set test deadlines from closeWait.
- **BR-12** [Important] `failure-cause-misattributed` A clean operator quit can be reported as "lost Chrome": the Wait goroutine's outR.Close yields os.ErrClosed, which endError treats as a conn failure
  3rd in family. Rule: an end cause is only what was observed before our own teardown began; errors teardown can produce (outR.Close, conn.Close, EPIPE after exit) are never causes. Have the loop record whether the event stream closed while the process still ran, and pass that to endError. Reproduced: endError(notEnding, "read |0: file already closed", 0) returns "lost Chrome". Add a peer exit mode that closes the reader side.
- **BR-13** [Minor] `dead-code` Chrome.exited field (chrome.go:71) is never set or read after the process seam refactor

## Round 4 — 2026-10-11T20:51:55-07:00 (claude) — passed

### Disposed

- BR-4 — addressed — startProcess sets WaitDelay (chrome.go:205); with it reverted, TestStartProcessReapsDespiteInheritedOutput times out at 5s.
- BR-7 — not-addressed — Tie-break by id is in place (tracker.go:121-125), but reverting it leaves Tracker/Fake/Conformance green over 20 runs. Add a tracker test with 3+ background-only hidden pages and call Active repeatedly.
- BR-9 — withdrawn — Gap accepted and documented (fake.go CloseTab doc, issue line 91); the scripted-peer loop tests cover the real PageTitle/grace paths through production code.
- BR-11 — addressed — Drain after oversized/malformed (conn.go:195,204); with it reverted, TestReaderDrainsAfterProtocolFailure fails. Exit-reason tests pass 40/40 under -race.
- BR-12 — addressed — endError accepts only cdp.ErrProtocol; with the deny-list restored, TestEndError rows fail every time and the closeReader loop subtest fails on some runs.
- BR-13 — addressed — Chrome.exited field removed (chrome.go:69-73); package builds and vets clean.

## Round 5 — 2026-10-11T21:46:48-07:00 (claude) — BLOCKED

### Raised

- **BR-14** [Important] `state-machine-untested-sequences` Host.Mount cancel racing open completion leaves the Host open with no mounts or waiters (tunnel leaked)
  host.go:113-130 — open() wins mu with waiting==1 and sets attempt=nil, then the cancelled waiter decrements waiting to 0 with last=false and returns; reproduced 1/300 with OpenDelay≈ctx timeout. Rule: every loss of interest (open done, waiter cancel, Unmount) runs one "nobody wants it" helper that closes the Host; add the planned gate seam that holds open() after start() so the test can cancel at that point and assert Opens−Closes==0. 2nd finding in this family.
- **BR-15** [Minor] `capability-token-compare` webcast.Server.Owns compares the token with == instead of the shared constant-time ownsToken
  webcast/server.go:47; plan PQ-1 specifies a constant-time compare per mount and broadcast.Server already uses ownsToken. Export it and use it in both.

## Round 6 — 2026-10-11T21:54:26-07:00 (claude) — BLOCKED

### Disposed

- BR-14 — addressed — releaseIfUnwantedLocked is the single release path; TestHostLastWaiterGivingUpAfterTheOpenCloses fails 3/3 against pre-fix host.go (Opens 1 Closes 0) and passes at head.
- BR-15 — addressed — OwnsToken exported (host.go:355) and used by broadcast.Server (server.go:87,115) and webcast.Server.Owns (webcast/server.go:49).

### Raised

- **BR-16** [Important] `failure-cause-misattributed` A Mount arriving during an abandoned open joins it and returns another caller's context.Canceled
  host.go:102 counts a new arrival on any hostOpening attempt, including one releaseIfUnwantedLocked already cancelled (:160). Reproduced: a tunnel that takes 100ms to wind down after cancel (as cloudflared's SIGTERM grace does) makes a background-context Mount fail with "context canceled". 4th in family. Rule: a Mount's error must come from its own request, never from an attempt another caller abandoned. Mark abandoned attempts (flag or hostAbandoning state), have Mount wait uncounted and reopen, add FakeTunnel.CancelDelay plus a regression test.
- **BR-17** [Minor] `state-machine-untested-sequences` BR-14 regression test drives abandonWait directly instead of through Mount via a gate seam
  host_test.go:255 bumps h.waiting by hand and calls abandonWait, so a Mount ctx.Done branch that stopped calling it would be caught only by the probabilistic stress loop. 3rd in family. Rule: Host sequence tests reach a state through public events with an injected ordering seam (hold open() after start()), not by mutating fields.

## Round 7 — 2026-10-11T22:05:53-07:00 (claude) — passed

### Disposed

- BR-16 — addressed — attempt.abandoned makes the attempt unjoinable (host.go:107,165); reverting the guard turns TestHostMountAfterAnAbandonedOpenSucceeds red with "context canceled".
- BR-17 — not-addressed — host_test.go:259-266 still mutates h.waiting and calls abandonWait directly; no gate seam drives it through Mount.

### Raised

- **BR-18** [Minor] `fake-fidelity` Late-cancel and failing-open tunnel behaviours are test-local wrappers, not settings on the shared FakeTunnel
  lateCancelTunnel (host_test.go:294) and failingTunnel (:123) model cloudflared behaviour BR-16 depends on, but only in broadcast tests; webcast/couch tests on FakeTunnel cannot exercise it. Rule: every dependency behaviour a test relies on is a FakeTunnel setting (CancelDelay, FailOpens), covered by the live conformance check.

## Open findings

- **BR-7** [Minor] `nondeterministic-tie-break` Tracker.Active picks among equal activated values by map iteration order, so the published tab can flip
- **BR-17** [Minor] `state-machine-untested-sequences` BR-14 regression test drives abandonWait directly instead of through Mount via a gate seam
- **BR-18** [Minor] `fake-fidelity` Late-cancel and failing-open tunnel behaviours are test-local wrappers, not settings on the shared FakeTunnel
