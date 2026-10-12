---
gate: boundary-review
issue: 489
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-12T11:40:46-07:00"
      dispatched: "2026-10-12T11:29:12-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: Exit watch tests process-group liveness, but the Couch-launched title poller shares the group for up to 30 s, so a dead launch does not "fail at once"
          detail: 'whileHelperAlive (launch_existing.go:461) uses ownedProcessGroupAlive. Couch-hosted sidecars stay in the helper''s group (osruntime.go:458-462); SpawnTitlePoller runs before launchWatched (createflow.go:859) and waits StartupGrace=30s (titlepoller/run.go:77,127). A Pair that dies at birth (the #288 case, about 10 s) therefore holds the wait until about 30 s, longer than the old 15 s. The fake''s SetExited kills the whole group, so TestColdResumeFailsAtOnceWhenPairExitsBeforeRegistering can''t see it. Fix: watch the direct child''s exit (execHandle.done or ptyHandle child wait, exposed on Handle) and keep group liveness for cleanup; teach the fake a surviving group member and add that test.'
          family: fake-diverges-from-real-seam
          round: 1
        - id: BR-2
          severity: Important
          title: atlas/couch.md:2101 still says a cold resume runs to its 15 s deadline after the helper exits
          detail: The registration wait is now a 2-minute ceiling ended by the helper exiting. Rewrite the pair#288 cold-resume bullet to describe what really happens (it depends on I1).
          family: docs-restate-replaced-behavior
          round: 1
        - id: BR-3
          severity: Minor
          title: TestRegistrationCeilingClearsAStartSlowedByLoad only checks that a constant is at least 1 minute
          family: test-pins-constant-not-behavior
          round: 1
        - id: BR-4
          severity: Minor
          title: TestColdResumeAdoptsALateStart decides timing with time.Since on the real clock, so it only ever runs one interleaving
          detail: Pass a controllable clock into the sidecar hook so the late start becomes a reproducible ordering (ARCH-ORDER).
          family: wall-clock-single-interleaving-test
          round: 1
        - id: BR-5
          severity: Minor
          title: After timed-out status polls, the "still <status>" line reports the last status received without saying the polls timed out
          family: progress-reports-stale-status
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-12T13:18:26-07:00"
      dispatched: "2026-10-12T13:00:24-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: whileHelperAlive keys on Handle.Exited (launch_existing.go:461); I swapped back a group-Alive poll in a scratch copy and TestColdResumeFailsAtOnceWhenPairExitsBeforeRegistering hung to the 30 s timeout (red), and it passes at head.
          round: 2
        - id: BR-2
          disposition: addressed
          note: atlas/couch.md:2111 now says the wait ends when the helper's direct child exits; stale group wording elsewhere is raised as a new family instance.
          round: 2
        - id: BR-3
          disposition: addressed
          note: The constant-only test is gone; TestColdResumeAdoptsALateStart checks the deadline the real launch path arms (default from couch.go:200).
          round: 2
        - id: BR-4
          disposition: addressed
          note: The late start is now driven by poll count through PaneSidecarsHook (50th look), not time.Since on the real clock.
          round: 2
        - id: BR-5
          disposition: addressed
          note: The progress line names timed-out polls (messages.go:276) and TestSlotOperationCLIProgressNamesTimedOutPolls pins it; a leftover case is raised separately.
          round: 2
      findings:
        - id: BR-6
          severity: Important
          title: couch.go:396, birthwatch.go:25, atlas/architecture.md:351 and Plan line 47 still say the wait ends on the process group exiting
          detail: '2nd instance of this family. Rule: describe the exit signal once (whileHelperAlive doc + atlas/couch.md pair-489 section) and make every other site name it rather than restate it. Sweep with rg for process-group-exit and old 15 s registration wording across cmd/, atlas/, README.md and the issue file in the same round.'
          family: docs-restate-replaced-behavior
          round: 2
        - id: BR-7
          severity: Important
          title: The final registration check after the helper exits has no test that fails without it
          detail: 'With the check disabled (if false && await(final)) the full couchcore suite still passes. Add a fake test: AfterAcknowledge writes the registration and calls SetChildExited, and PaneSidecarsHook holds the first poll until Exited() has closed; assert the resume succeeds (ARCH-ORDER).'
          family: behavior-fix-without-red-test
          round: 2
        - id: BR-8
          severity: Minor
          title: The timed-out-polls note persists after fresh answers, and the budget-exhausted message hides timeouts
          detail: '2nd instance. timeouts resets only when a line prints (messages.go:279), not on a successful poll (messages.go:297). Rule: every status line derives staleness from the time of the last answer received (lastAnswer), not from a counter of events since the last print; apply it to both the progress line and the budget-exhausted message.'
          family: progress-reports-stale-status
          round: 2
        - id: BR-9
          severity: Minor
          title: No live conformance check that the real Exited() closes while a group member survives, as the fake now models
          detail: '2nd instance. Rule: every state transition added to FakeRunner gets a matching live runner-conformance case in the same change; here, sh -c ''sleep 5 & exit 0'' must give Exited closed while Alive stays true.'
          family: fake-diverges-from-real-seam
          round: 2
        - id: BR-10
          severity: Minor
          title: stubHandle.Exited returns a nil channel, which blocks forever if ever awaited
          family: test-double-unsafe-default
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-12T14:29:41-07:00"
      dispatched: "2026-10-12T14:21:21-07:00"
      agent: claude
      dispose:
        - id: BR-6
          disposition: addressed
          note: couch.go:396, birthwatch.go:25 and architecture.md:351 now point at whileHelperAlive or the atlas section; the Plan line and Revisions now say direct child. An rg sweep found no remaining process-group-exit or 15 s registration wording.
          round: 3
        - id: BR-7
          disposition: addressed
          note: With the final look disabled (if false && ...), TestColdResumeRegistrationWrittenBeforeTheExitStillCounts FAILS ("Pair exited before registering"); it passes once restored.
          round: 3
        - id: BR-8
          disposition: addressed
          note: stale() derives from lastAnswer, which only a successful poll updates; it covers the progress and budget messages, and TestSlotOperationCLIProgressNamesTimedOutPolls asserts the note clears.
          round: 3
        - id: BR-9
          disposition: addressed
          note: TestRealRunnersExitedPrecedesTheGroup runs ungated on the exec and pty runners and passes.
          round: 3
        - id: BR-10
          disposition: addressed
          note: The couchtty stubHandle.Exited returns a closed channel.
          round: 3
      findings:
        - id: BR-11
          severity: Minor
          title: Dead-launch error still says "ownership poll consumed the budget" and "context canceled"
          detail: launch_existing.go:404-424 keeps the first await's registrationWaitError, so the text blames a budget the exit ended. When exited, wrap context.Cause (the exit error) and leave out the phase bracket.
          family: failure-text-names-wrong-cause
          round: 3
        - id: BR-12
          severity: Minor
          title: atlas/couch.md pair#489 sentence reads as if registration happens when the child exits
          detail: '"the wait ends at registration, when the helper''s DIRECT child exits ..., or at ..." should list three separate end conditions.'
          family: doc-ambiguous-wording
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: 646773eb662bd25738bbc0f143cce4ef02d3aa18
      blocked: false
---

# Gate ledger — pair#489 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-12T11:40:46-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `fake-diverges-from-real-seam` Exit watch tests process-group liveness, but the Couch-launched title poller shares the group for up to 30 s, so a dead launch does not "fail at once"
  whileHelperAlive (launch_existing.go:461) uses ownedProcessGroupAlive. Couch-hosted sidecars stay in the helper's group (osruntime.go:458-462); SpawnTitlePoller runs before launchWatched (createflow.go:859) and waits StartupGrace=30s (titlepoller/run.go:77,127). A Pair that dies at birth (the #288 case, about 10 s) therefore holds the wait until about 30 s, longer than the old 15 s. The fake's SetExited kills the whole group, so TestColdResumeFailsAtOnceWhenPairExitsBeforeRegistering can't see it. Fix: watch the direct child's exit (execHandle.done or ptyHandle child wait, exposed on Handle) and keep group liveness for cleanup; teach the fake a surviving group member and add that test.
- **BR-2** [Important] `docs-restate-replaced-behavior` atlas/couch.md:2101 still says a cold resume runs to its 15 s deadline after the helper exits
  The registration wait is now a 2-minute ceiling ended by the helper exiting. Rewrite the pair#288 cold-resume bullet to describe what really happens (it depends on I1).
- **BR-3** [Minor] `test-pins-constant-not-behavior` TestRegistrationCeilingClearsAStartSlowedByLoad only checks that a constant is at least 1 minute
- **BR-4** [Minor] `wall-clock-single-interleaving-test` TestColdResumeAdoptsALateStart decides timing with time.Since on the real clock, so it only ever runs one interleaving
  Pass a controllable clock into the sidecar hook so the late start becomes a reproducible ordering (ARCH-ORDER).
- **BR-5** [Minor] `progress-reports-stale-status` After timed-out status polls, the "still <status>" line reports the last status received without saying the polls timed out

## Round 2 — 2026-10-12T13:18:26-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — whileHelperAlive keys on Handle.Exited (launch_existing.go:461); I swapped back a group-Alive poll in a scratch copy and TestColdResumeFailsAtOnceWhenPairExitsBeforeRegistering hung to the 30 s timeout (red), and it passes at head.
- BR-2 — addressed — atlas/couch.md:2111 now says the wait ends when the helper's direct child exits; stale group wording elsewhere is raised as a new family instance.
- BR-3 — addressed — The constant-only test is gone; TestColdResumeAdoptsALateStart checks the deadline the real launch path arms (default from couch.go:200).
- BR-4 — addressed — The late start is now driven by poll count through PaneSidecarsHook (50th look), not time.Since on the real clock.
- BR-5 — addressed — The progress line names timed-out polls (messages.go:276) and TestSlotOperationCLIProgressNamesTimedOutPolls pins it; a leftover case is raised separately.

### Raised

- **BR-6** [Important] `docs-restate-replaced-behavior` couch.go:396, birthwatch.go:25, atlas/architecture.md:351 and Plan line 47 still say the wait ends on the process group exiting
  2nd instance of this family. Rule: describe the exit signal once (whileHelperAlive doc + atlas/couch.md pair-489 section) and make every other site name it rather than restate it. Sweep with rg for process-group-exit and old 15 s registration wording across cmd/, atlas/, README.md and the issue file in the same round.
- **BR-7** [Important] `behavior-fix-without-red-test` The final registration check after the helper exits has no test that fails without it
  With the check disabled (if false && await(final)) the full couchcore suite still passes. Add a fake test: AfterAcknowledge writes the registration and calls SetChildExited, and PaneSidecarsHook holds the first poll until Exited() has closed; assert the resume succeeds (ARCH-ORDER).
- **BR-8** [Minor] `progress-reports-stale-status` The timed-out-polls note persists after fresh answers, and the budget-exhausted message hides timeouts
  2nd instance. timeouts resets only when a line prints (messages.go:279), not on a successful poll (messages.go:297). Rule: every status line derives staleness from the time of the last answer received (lastAnswer), not from a counter of events since the last print; apply it to both the progress line and the budget-exhausted message.
- **BR-9** [Minor] `fake-diverges-from-real-seam` No live conformance check that the real Exited() closes while a group member survives, as the fake now models
  2nd instance. Rule: every state transition added to FakeRunner gets a matching live runner-conformance case in the same change; here, sh -c 'sleep 5 & exit 0' must give Exited closed while Alive stays true.
- **BR-10** [Minor] `test-double-unsafe-default` stubHandle.Exited returns a nil channel, which blocks forever if ever awaited

## Round 3 — 2026-10-12T14:29:41-07:00 (claude) — passed

### Disposed

- BR-6 — addressed — couch.go:396, birthwatch.go:25 and architecture.md:351 now point at whileHelperAlive or the atlas section; the Plan line and Revisions now say direct child. An rg sweep found no remaining process-group-exit or 15 s registration wording.
- BR-7 — addressed — With the final look disabled (if false && ...), TestColdResumeRegistrationWrittenBeforeTheExitStillCounts FAILS ("Pair exited before registering"); it passes once restored.
- BR-8 — addressed — stale() derives from lastAnswer, which only a successful poll updates; it covers the progress and budget messages, and TestSlotOperationCLIProgressNamesTimedOutPolls asserts the note clears.
- BR-9 — addressed — TestRealRunnersExitedPrecedesTheGroup runs ungated on the exec and pty runners and passes.
- BR-10 — addressed — The couchtty stubHandle.Exited returns a closed channel.

### Raised

- **BR-11** [Minor] `failure-text-names-wrong-cause` Dead-launch error still says "ownership poll consumed the budget" and "context canceled"
  launch_existing.go:404-424 keeps the first await's registrationWaitError, so the text blames a budget the exit ended. When exited, wrap context.Cause (the exit error) and leave out the phase bracket.
- **BR-12** [Minor] `doc-ambiguous-wording` atlas/couch.md pair#489 sentence reads as if registration happens when the child exits
  "the wait ends at registration, when the helper's DIRECT child exits ..., or at ..." should list three separate end conditions.

## Open findings

- **BR-11** [Minor] `failure-text-names-wrong-cause` Dead-launch error still says "ownership poll consumed the budget" and "context canceled"
- **BR-12** [Minor] `doc-ambiguous-wording` atlas/couch.md pair#489 sentence reads as if registration happens when the child exits
