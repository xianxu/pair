---
gate: plan-quality
issue: 182
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-04T09:24:15-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: The failure model omits the branch where the park itself fails, which is the likeliest one
          detail: |-
            The plan enumerates refuse-before-park and park-ok-resume-failed. It never
            handles park failure. PairLifecycleController.Park is a multi-phase
            transaction with six failure exits (park.go:284 commit deadline, :504
            publish failed, :573 the 15s completion timeout budgeted at couch.go:119,
            :608 stale completion, :613 Pair cleanup failure, :616 child not gone,
            :628 revision conflict), each leaving record.Park != nil. In that state
            Pair has already been sent its quit intent (park.go:534), there is no
            VerifiedPark, DecideResume refuses with ResumeParking (resume.go:79), and
            the row is not a parked row Enter can resume -- recovery requires
            park --mode=retry/recover/abandon. That is worse than the outcome the
            whole design exists to prevent, and it has no task, no test, and no
            operator message. Name the outcome, decide whether Relaunch retries or
            surfaces the recovery verb, and test it.
          family: composition-failure-modes
          round: 1
        - id: PQ-2
          severity: Important
          title: The precondition-agreement test clears occupancy but not the park authority, so it asserts a false equivalence
          detail: |-
            Task 1's test uses withoutIncarnations(tc.record), but DecideResume also
            refuses on record.Park != nil (resume.go:79) and on
            record.VerifiedPark == nil && !input.Detached with its tombstone scan
            (resume.go:88-101). A healthy live never-parked fixture therefore gives
            precondition == nil and resumeErr == ResumeLegacyUnverified, failing for a
            reason unrelated to the extraction. Task 1 Step 3's prose already says
            DecideResume keeps those rules, so the prose and the test disagree. The
            transform must model the POST-PARK record -- occupancy cleared, Park nil,
            VerifiedPark stamped -- which is what "would this be resumable once
            parked?" means. Left as written, the fix under time pressure is to weaken
            the assertion, and the drift guard the plan is built around disappears.
          family: drift-guard-assertion
          round: 1
        - id: PQ-3
          severity: Important
          title: The plan promises to name the precondition that cannot be evaluated early and never names it
          detail: |-
            The Architecture section states "the one precondition that cannot be
            evaluated early is named rather than hoped over"; no later section
            discharges it. Two real candidates. First, the binding is established by
            validating an authorization proof against a scan of the agent's LIVE
            native-session artifacts (sessioninventory/query.go:66-159 ->
            ValidateBindingProof -> Observe(agent)), read while the agent is still
            writing that file -- nothing argues BindingEstablished pre-park implies
            BindingEstablished post-park. Second, park requires
            soleParkableIncarnation (park.go:268), a relaunch precondition absent from
            the plan's set, so a thread with a second incarnation passes the pre-check
            and fails at the park -- exactly what the ordering was meant to prevent.
          family: unstated-precondition
          round: 1
        - id: PQ-4
          severity: Important
          title: No test asserts a successful relaunch, which is Done-when bullet 5
          detail: |-
            Task 2 tests refusal, Task 3 tests park-then-failed-resume, Task 4 tests
            the addressing dialect. Nothing asserts that a healthy relaunch ends with
            one live incarnation at the same address, the row still in the switcher,
            and the same ledger identity -- the issue's own last Done-when bullet.
            Task 5's manual real-stack observation is the binary-freshness evidence,
            not a regression guard. ARCH-PURPOSE: the untested branch is the thing
            being built.
          family: done-when-untested
          round: 1
        - id: PQ-5
          severity: Important
          title: No operating envelope for the longest operation the switcher will ever run
          detail: |-
            ARCH-CONSTRAINTS at-plan gets no answer. Relaunch chains park's 15s
            completion wait (couch.go:119) and the resume's 10s StartBlocked ack
            (launch_existing.go:70) on operationQueue.Run's single worker loop
            (operation_queue.go:59-72), serializing every other switcher operation
            behind it. This path is already explicitly budgeted elsewhere
            (parkCommitSoftTarget 100ms, parkCommitDeadline 1s, couchtty/park_latency_test.go).
            State the workload class, the latency budget and its basis, what the
            switcher displays while it runs, and the bounded behavior on exceed.
          family: operating-envelope
          round: 1
        - id: PQ-6
          severity: Minor
          title: Open question 3 (--layout2) is answerable from the code and should be decided, not recorded
          detail: |-
            Park's teardown deletes the zellij session (launcher.QuiesceThreadSession
            -> zellij delete-session --force, session_quiescence.go:153), so
            post-park the cold resume's --layout2 (launch_existing.go:50) meets no
            live session and the conflict path warned about at
            launch_existing.go:47-52 cannot fire. Recording it as open invites the
            implementer to diverge the relaunch resume from the ordinary cold resume,
            which is the one change that would reintroduce the pair#181 M2 hazard.
          family: answerable-open-question
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-04T09:28:06-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Failure-state table plus Task 3b; asserts ParkIncomplete, no child spawn, open transaction, and park's recovery verbs in the message.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Test now transforms via asParked (occupancy cleared, Park nil, VerifiedPark stamped), matching DecideResume's actual check order at resume.go:78-101.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: 'Both late preconditions named: the binding window is declared irreducible and routed to the recoverable row; soleParkableIncarnation is checked explicitly pre-park.'
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Task 3c asserts same address, one live incarnation, changed PID, unchanged native session id, surviving live row.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Envelope states workload class, ~30s worst case from verified constants, the progress notice, and the bounded queue.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: The plan now states the decision (behave exactly as a normal cold resume), which session_quiescence.go:153 supports.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-09-04T09:59:18-07:00"
      agent: claude
      findings:
        - id: PQ-7
          severity: Minor
          title: Two of the envelope's three durations name budgets that do not exist as constants
          detail: |-
            Family repeat (2nd). Rule, not instance: every duration in the operating
            envelope must cite the constant producing it by file:line, and say so
            explicitly where the budget is derived rather than declared. Here the
            "5s exact-child-death wait (couch.go:119)" is not separate — child death is
            awaited inside the single 15s CompletionTimeout (park.go:549-555) — and the
            "10s blocked-start acknowledgement" is actually resumeRegistrationTimeout,
            5s (couch.go:107, launch_existing.go:110-111). Real worst case ~20s, not
            ~30s; the plan over-budgets, so no downstream decision changes.
          family: operating-envelope
          round: 3
      blocked: false
content_hash: 3daf5f639fed110f2617e0519636851256b93b35d9e64185dbbf9a4a219a5dac
---

# Gate ledger — pair#182 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-04T09:24:15-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `composition-failure-modes` The failure model omits the branch where the park itself fails, which is the likeliest one
  The plan enumerates refuse-before-park and park-ok-resume-failed. It never
  handles park failure. PairLifecycleController.Park is a multi-phase
  transaction with six failure exits (park.go:284 commit deadline, :504
  publish failed, :573 the 15s completion timeout budgeted at couch.go:119,
  :608 stale completion, :613 Pair cleanup failure, :616 child not gone,
  :628 revision conflict), each leaving record.Park != nil. In that state
  Pair has already been sent its quit intent (park.go:534), there is no
  VerifiedPark, DecideResume refuses with ResumeParking (resume.go:79), and
  the row is not a parked row Enter can resume -- recovery requires
  park --mode=retry/recover/abandon. That is worse than the outcome the
  whole design exists to prevent, and it has no task, no test, and no
  operator message. Name the outcome, decide whether Relaunch retries or
  surfaces the recovery verb, and test it.
- **PQ-2** [Important] `drift-guard-assertion` The precondition-agreement test clears occupancy but not the park authority, so it asserts a false equivalence
  Task 1's test uses withoutIncarnations(tc.record), but DecideResume also
  refuses on record.Park != nil (resume.go:79) and on
  record.VerifiedPark == nil && !input.Detached with its tombstone scan
  (resume.go:88-101). A healthy live never-parked fixture therefore gives
  precondition == nil and resumeErr == ResumeLegacyUnverified, failing for a
  reason unrelated to the extraction. Task 1 Step 3's prose already says
  DecideResume keeps those rules, so the prose and the test disagree. The
  transform must model the POST-PARK record -- occupancy cleared, Park nil,
  VerifiedPark stamped -- which is what "would this be resumable once
  parked?" means. Left as written, the fix under time pressure is to weaken
  the assertion, and the drift guard the plan is built around disappears.
- **PQ-3** [Important] `unstated-precondition` The plan promises to name the precondition that cannot be evaluated early and never names it
  The Architecture section states "the one precondition that cannot be
  evaluated early is named rather than hoped over"; no later section
  discharges it. Two real candidates. First, the binding is established by
  validating an authorization proof against a scan of the agent's LIVE
  native-session artifacts (sessioninventory/query.go:66-159 ->
  ValidateBindingProof -> Observe(agent)), read while the agent is still
  writing that file -- nothing argues BindingEstablished pre-park implies
  BindingEstablished post-park. Second, park requires
  soleParkableIncarnation (park.go:268), a relaunch precondition absent from
  the plan's set, so a thread with a second incarnation passes the pre-check
  and fails at the park -- exactly what the ordering was meant to prevent.
- **PQ-4** [Important] `done-when-untested` No test asserts a successful relaunch, which is Done-when bullet 5
  Task 2 tests refusal, Task 3 tests park-then-failed-resume, Task 4 tests
  the addressing dialect. Nothing asserts that a healthy relaunch ends with
  one live incarnation at the same address, the row still in the switcher,
  and the same ledger identity -- the issue's own last Done-when bullet.
  Task 5's manual real-stack observation is the binary-freshness evidence,
  not a regression guard. ARCH-PURPOSE: the untested branch is the thing
  being built.
- **PQ-5** [Important] `operating-envelope` No operating envelope for the longest operation the switcher will ever run
  ARCH-CONSTRAINTS at-plan gets no answer. Relaunch chains park's 15s
  completion wait (couch.go:119) and the resume's 10s StartBlocked ack
  (launch_existing.go:70) on operationQueue.Run's single worker loop
  (operation_queue.go:59-72), serializing every other switcher operation
  behind it. This path is already explicitly budgeted elsewhere
  (parkCommitSoftTarget 100ms, parkCommitDeadline 1s, couchtty/park_latency_test.go).
  State the workload class, the latency budget and its basis, what the
  switcher displays while it runs, and the bounded behavior on exceed.
- **PQ-6** [Minor] `answerable-open-question` Open question 3 (--layout2) is answerable from the code and should be decided, not recorded
  Park's teardown deletes the zellij session (launcher.QuiesceThreadSession
  -> zellij delete-session --force, session_quiescence.go:153), so
  post-park the cold resume's --layout2 (launch_existing.go:50) meets no
  live session and the conflict path warned about at
  launch_existing.go:47-52 cannot fire. Recording it as open invites the
  implementer to diverge the relaunch resume from the ordinary cold resume,
  which is the one change that would reintroduce the pair#181 M2 hazard.

## Round 2 — 2026-09-04T09:28:06-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Failure-state table plus Task 3b; asserts ParkIncomplete, no child spawn, open transaction, and park's recovery verbs in the message.
- PQ-2 — addressed — Test now transforms via asParked (occupancy cleared, Park nil, VerifiedPark stamped), matching DecideResume's actual check order at resume.go:78-101.
- PQ-3 — addressed — Both late preconditions named: the binding window is declared irreducible and routed to the recoverable row; soleParkableIncarnation is checked explicitly pre-park.
- PQ-4 — addressed — Task 3c asserts same address, one live incarnation, changed PID, unchanged native session id, surviving live row.
- PQ-5 — addressed — Envelope states workload class, ~30s worst case from verified constants, the progress notice, and the bounded queue.
- PQ-6 — addressed — The plan now states the decision (behave exactly as a normal cold resume), which session_quiescence.go:153 supports.

## Round 3 — 2026-09-04T09:59:18-07:00 (claude) — passed

### Raised

- **PQ-7** [Minor] `operating-envelope` Two of the envelope's three durations name budgets that do not exist as constants
  Family repeat (2nd). Rule, not instance: every duration in the operating
  envelope must cite the constant producing it by file:line, and say so
  explicitly where the budget is derived rather than declared. Here the
  "5s exact-child-death wait (couch.go:119)" is not separate — child death is
  awaited inside the single 15s CompletionTimeout (park.go:549-555) — and the
  "10s blocked-start acknowledgement" is actually resumeRegistrationTimeout,
  5s (couch.go:107, launch_existing.go:110-111). Real worst case ~20s, not
  ~30s; the plan over-budgets, so no downstream decision changes.

## Open findings

- **PQ-7** [Minor] `operating-envelope` Two of the envelope's three durations name budgets that do not exist as constants
