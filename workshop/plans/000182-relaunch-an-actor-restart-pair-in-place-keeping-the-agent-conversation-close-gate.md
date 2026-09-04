---
gate: boundary-review
issue: 182
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-04T10:47:03-07:00"
      agent: sdlc
      findings:
        - id: BR-1
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
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: operating-envelope
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-04T10:47:03-07:00"
      agent: claude
      findings:
        - id: BR-2
          severity: Important
          title: A resolver IO failure is reported as "binding is not established", and the branch meant to catch it is dead code
          detail: |-
            relaunch.go:96-108 — ResolveEstablished returns a ZERO resolution on a real
            IO error (artifactpath.Resolve / os.UserHomeDir / QuerySessionContext), so
            bindingResumeDiagnostic's default arm yields ResumeBindingUnbound and
            CheckResumePreconditions returns at line 103; `if bindingErr != nil` at 106
            is unreachable. Proved by deleting lines 106-108 in a scratch worktree and
            running -run 'Relaunch|Resume|Park': still green. ResumeContext
            (resume.go:279-283) returns the resolver's error directly, so the two
            callers of the same seam disagree. Resolve the binding only after the
            non-binding preconditions pass, return bindingErr when it is not a
            *ResumeRefusal, and pin it with an erroring-resolver case. ARCH-SECURE.
          family: swallowed-cause-fabricated-diagnostic
          round: 2
        - id: BR-3
          severity: Important
          title: Relaunch on a thread that is not running refuses with `resume-live` and names no next action
          detail: |-
            relaunch.go:81-83 maps every soleParkableIncarnation failure to
            refuseResume(ResumeLive, err.Error()), but park.go:768-783 fails for NO
            live/unknown incarnation as well as for two. On a parked row the operator
            gets "resume-live: park requires exactly one identified live or unknown
            incarnation" — a code contradicting the state and a message about park's
            internals. Reachable: the Spec's panel form relaunches the HIGHLIGHTED row,
            which is often parked. Split the cases; for a thread with no incarnation
            refuse with its own code naming the working gesture ("Enter resumes it").
          family: refusal-names-no-next-action
          round: 2
        - id: BR-4
          severity: Important
          title: Resume's evidence-gathering preamble is re-derived in Relaunch, and the first copy already diverges
          detail: |-
            relaunch.go:88-101 vs resume.go:262-283 both derive `agent` from the saved
            profile, type-assert c.Artifacts to NativeBindingResolver, and compute
            pathExists from c.Path.Physical. They already differ on nil Path: relaunch
            defaults pathExists to true and skips the call, so the precondition PASSES
            and ResumeContext then nil-derefs one step later. The plan extracted the
            RULES for exactly this reason; the evidence needs the same treatment —
            one resumeEvidence(ctx, thread, address) helper with one nil-Path policy.
            ARCH-DRY.
          family: parallel-derivation-drift
          round: 2
        - id: BR-5
          severity: Important
          title: The declared operation reaches no operator surface, and the plan asserts it does
          detail: |-
            menuActionItems (menu.go:1008) returns hardcoded slices and consumes
            Operations() nowhere; ParseCLI is a closed flag set. So the plan's Chunk 2
            premise ("reachable from the switcher's action list the moment it is
            declared") is false, and the Done-when bullet "an actor action relaunch
            appears alongside detach and park ... reachable from the same
            declared-operation surface" has no owning task in M1 or M2. The class is
            six hand-maintained per-operation sites, not the two Task 10 enumerates:
            menu.go:1008, confirmationMenuItems, menu.go:1306, menu.go:1320,
            console.go:1375, console.go:1425. Fix the plan now and sweep the
            enumeration in one M2 round. ARCH-PURPOSE, ARCH-DRY.
          family: declared-source-hand-maintained-consumers
          round: 2
        - id: BR-6
          severity: Minor
          title: No Relaunch-level test for the agent-unsupported and profile-missing refusals
          detail: |-
            The plan's Task 2 table lists four cases; three shipped. The rules are
            covered purely in precondition_test.go and the diagnostic PRECEDENCE is
            correct (profile checks run before the binding check, so an unsupported
            agent still yields ResumeAgentUnsupported even though ResolveEstablished
            was already called with it), but nothing pins that precedence through
            Relaunch.
          family: done-when-untested
          round: 2
        - id: BR-7
          severity: Minor
          title: Task 1 Step 5 is unticked though committed, and the issue's stale-estimate warning contradicts the re-derived Estimate block
          detail: |-
            plan line 159 ("Step 5: Commit") is unchecked although e7c6c6e8 landed it —
            milestone-close's plan-unchecked guard will refuse on it. Separately, the
            issue's ## Revisions still says "⚠ The estimate is now stale" while ##
            Estimate has since been re-derived for the grown scope (3.58 → 6.20).
          family: plan-record-lags-code
          round: 2
        - id: BR-8
          severity: Minor
          title: RelaunchResult does not compose with finishOperation's existing ParkResult/StartResult arms
          detail: |-
            relaunch.go:43 — console.go:1324 and :1328 already know how to land a
            ParkResult address and force-switch onto a StartResult handle.
            RelaunchResult{Outcome, Record, Handle} matches neither, so M2 adds a third
            arm. RelaunchResult{Outcome, Start StartResult} would reuse the resume arm
            unchanged.
          family: result-shape-forces-new-consumer-arm
          round: 2
        - id: BR-9
          severity: Minor
          title: relaunch.go inserted out of alphabetical order in NonArtifactSources
          detail: |-
            artifactpath/manifest.go:524 places it between pathops.go and procops.go.
            Nothing enforces the order; noted so the next insert does not compound it.
          family: manifest-ordering
          round: 2
      boundary: M1
      blocked: true
---

# Gate ledger — pair#182 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-04T10:47:03-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `operating-envelope` Two of the envelope's three durations name budgets that do not exist as constants
  Family repeat (2nd). Rule, not instance: every duration in the operating
  envelope must cite the constant producing it by file:line, and say so
  explicitly where the budget is derived rather than declared. Here the
  "5s exact-child-death wait (couch.go:119)" is not separate — child death is
  awaited inside the single 15s CompletionTimeout (park.go:549-555) — and the
  "10s blocked-start acknowledgement" is actually resumeRegistrationTimeout,
  5s (couch.go:107, launch_existing.go:110-111). Real worst case ~20s, not
  ~30s; the plan over-budgets, so no downstream decision changes.
  (carried from plan-quality PQ-7, deferred to the boundary review)

## Round 2 — 2026-09-04T10:47:03-07:00 (claude) — BLOCKED

### Raised

- **BR-2** [Important] `swallowed-cause-fabricated-diagnostic` A resolver IO failure is reported as "binding is not established", and the branch meant to catch it is dead code
  relaunch.go:96-108 — ResolveEstablished returns a ZERO resolution on a real
  IO error (artifactpath.Resolve / os.UserHomeDir / QuerySessionContext), so
  bindingResumeDiagnostic's default arm yields ResumeBindingUnbound and
  CheckResumePreconditions returns at line 103; `if bindingErr != nil` at 106
  is unreachable. Proved by deleting lines 106-108 in a scratch worktree and
  running -run 'Relaunch|Resume|Park': still green. ResumeContext
  (resume.go:279-283) returns the resolver's error directly, so the two
  callers of the same seam disagree. Resolve the binding only after the
  non-binding preconditions pass, return bindingErr when it is not a
  *ResumeRefusal, and pin it with an erroring-resolver case. ARCH-SECURE.
- **BR-3** [Important] `refusal-names-no-next-action` Relaunch on a thread that is not running refuses with `resume-live` and names no next action
  relaunch.go:81-83 maps every soleParkableIncarnation failure to
  refuseResume(ResumeLive, err.Error()), but park.go:768-783 fails for NO
  live/unknown incarnation as well as for two. On a parked row the operator
  gets "resume-live: park requires exactly one identified live or unknown
  incarnation" — a code contradicting the state and a message about park's
  internals. Reachable: the Spec's panel form relaunches the HIGHLIGHTED row,
  which is often parked. Split the cases; for a thread with no incarnation
  refuse with its own code naming the working gesture ("Enter resumes it").
- **BR-4** [Important] `parallel-derivation-drift` Resume's evidence-gathering preamble is re-derived in Relaunch, and the first copy already diverges
  relaunch.go:88-101 vs resume.go:262-283 both derive `agent` from the saved
  profile, type-assert c.Artifacts to NativeBindingResolver, and compute
  pathExists from c.Path.Physical. They already differ on nil Path: relaunch
  defaults pathExists to true and skips the call, so the precondition PASSES
  and ResumeContext then nil-derefs one step later. The plan extracted the
  RULES for exactly this reason; the evidence needs the same treatment —
  one resumeEvidence(ctx, thread, address) helper with one nil-Path policy.
  ARCH-DRY.
- **BR-5** [Important] `declared-source-hand-maintained-consumers` The declared operation reaches no operator surface, and the plan asserts it does
  menuActionItems (menu.go:1008) returns hardcoded slices and consumes
  Operations() nowhere; ParseCLI is a closed flag set. So the plan's Chunk 2
  premise ("reachable from the switcher's action list the moment it is
  declared") is false, and the Done-when bullet "an actor action relaunch
  appears alongside detach and park ... reachable from the same
  declared-operation surface" has no owning task in M1 or M2. The class is
  six hand-maintained per-operation sites, not the two Task 10 enumerates:
  menu.go:1008, confirmationMenuItems, menu.go:1306, menu.go:1320,
  console.go:1375, console.go:1425. Fix the plan now and sweep the
  enumeration in one M2 round. ARCH-PURPOSE, ARCH-DRY.
- **BR-6** [Minor] `done-when-untested` No Relaunch-level test for the agent-unsupported and profile-missing refusals
  The plan's Task 2 table lists four cases; three shipped. The rules are
  covered purely in precondition_test.go and the diagnostic PRECEDENCE is
  correct (profile checks run before the binding check, so an unsupported
  agent still yields ResumeAgentUnsupported even though ResolveEstablished
  was already called with it), but nothing pins that precedence through
  Relaunch.
- **BR-7** [Minor] `plan-record-lags-code` Task 1 Step 5 is unticked though committed, and the issue's stale-estimate warning contradicts the re-derived Estimate block
  plan line 159 ("Step 5: Commit") is unchecked although e7c6c6e8 landed it —
  milestone-close's plan-unchecked guard will refuse on it. Separately, the
  issue's ## Revisions still says "⚠ The estimate is now stale" while ##
  Estimate has since been re-derived for the grown scope (3.58 → 6.20).
- **BR-8** [Minor] `result-shape-forces-new-consumer-arm` RelaunchResult does not compose with finishOperation's existing ParkResult/StartResult arms
  relaunch.go:43 — console.go:1324 and :1328 already know how to land a
  ParkResult address and force-switch onto a StartResult handle.
  RelaunchResult{Outcome, Record, Handle} matches neither, so M2 adds a third
  arm. RelaunchResult{Outcome, Start StartResult} would reuse the resume arm
  unchanged.
- **BR-9** [Minor] `manifest-ordering` relaunch.go inserted out of alphabetical order in NonArtifactSources
  artifactpath/manifest.go:524 places it between pathops.go and procops.go.
  Nothing enforces the order; noted so the next insert does not compound it.

## Open findings

- **BR-1** [Minor] `operating-envelope` Two of the envelope's three durations name budgets that do not exist as constants
- **BR-2** [Important] `swallowed-cause-fabricated-diagnostic` A resolver IO failure is reported as "binding is not established", and the branch meant to catch it is dead code
- **BR-3** [Important] `refusal-names-no-next-action` Relaunch on a thread that is not running refuses with `resume-live` and names no next action
- **BR-4** [Important] `parallel-derivation-drift` Resume's evidence-gathering preamble is re-derived in Relaunch, and the first copy already diverges
- **BR-5** [Important] `declared-source-hand-maintained-consumers` The declared operation reaches no operator surface, and the plan asserts it does
- **BR-6** [Minor] `done-when-untested` No Relaunch-level test for the agent-unsupported and profile-missing refusals
- **BR-7** [Minor] `plan-record-lags-code` Task 1 Step 5 is unticked though committed, and the issue's stale-estimate warning contradicts the re-derived Estimate block
- **BR-8** [Minor] `result-shape-forces-new-consumer-arm` RelaunchResult does not compose with finishOperation's existing ParkResult/StartResult arms
- **BR-9** [Minor] `manifest-ordering` relaunch.go inserted out of alphabetical order in NonArtifactSources
