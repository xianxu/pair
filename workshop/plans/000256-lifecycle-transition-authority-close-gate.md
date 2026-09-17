---
gate: boundary-review
issue: 256
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-17T08:50:28-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Critical
          title: An open park makes the new re-adoption hard-error and wedges couch startup
          detail: A record with Park != nil, a dead incarnation and a surviving session now classifies detached (actionableinventory.go:316), passes DecideResume, then hits RetireIncarnation's park precondition (threadstore.go:551) via retireDeadIncarnationBeforeStart (resume.go:555-568), which returns the raw store error. startup.go:203-232 has no fallback and only decorates errors carrying a ResumeDiagnosticCode, so couch refuses to start in that tree with an internal message. Reproduced end to end against the real store. This is the fourth site of the class the Log enumerated as three.
          family: classification-not-authority
          round: 1
        - id: BR-2
          severity: Important
          title: SessionUnresolved is checked above VerifiedPark, so one failed list-sessions hides every parked row
          detail: 'actionableinventory.go:321-324 returns unusable/unknown before the cold-resume branch at :328. Verified: a fully-proved parked record classifies parked under SessionAbsent and unusable/unknown under SessionUnresolved. A parked record''s resume authority is durable and needs no session answer, and the only two verdicts the session could produce are parked and detached, both actionable.'
          family: unknown-must-not-mask-durable-authority
          round: 1
        - id: BR-3
          severity: Important
          title: Nothing pins that an Unknown liveness probe must not retire an incarnation
          detail: Mutating resume.go:564 from `!= Dead` to `== Live` (retire on Unknown) produces no test failure across couchcore, couchtty and couchcmd. This is the guard preventing a resume from abandoning a running agent, and it is the issue's own Done-when about Unknown never authorizing destructive recovery.
          family: fail-closed-guard-untested
          round: 1
        - id: BR-4
          severity: Important
          title: ScopedThreadArtifactCollisionChecker.SessionPresence has no test
          detail: Every presence test runs against the fake. The production implementation (artifactcollision.go:370-410), including the readable-scope-no-binding then SessionAbsent versus unreadable-scope then SessionUnresolved branch that decides archive-eligibility, is unexercised. The sandboxedChecker harness (artifactcollision_zellij_test.go:29) already stubs zellij for exactly this.
          family: production-seam-only-tested-through-fake
          round: 1
        - id: BR-5
          severity: Important
          title: Three atlas passages still document the retired reason vocabulary as current
          detail: atlas/couch.md:584-585 lists stale-incarnation and unrecorded-child in the closed vocabulary and says ThreadEvidence carries a ProofStatus per question; :963-965 says a stale IncarnationLive shows as unusable/stale-incarnation, the exact sentence this issue disproves; :251 references the shape in passing. The new section at :1277 says the opposite.
          family: atlas-contradicts-code
          round: 1
        - id: BR-6
          severity: Important
          title: README still lists a deleted reason label and claims parks leave a diagnostic
          detail: README.md:527 names `stale — helper ownership unresolved` among the labels an operator sees; that label was deleted here. README.md:517-518 says open start or park transactions leave a diagnostic instead of guessing a process died, which is no longer true of parks since nothing in the classification path reads record.Park.
          family: readme-documents-removed-surface
          round: 1
        - id: BR-7
          severity: Important
          title: ThreadBusy changed meaning but five wording sites still describe a park
          detail: 'Task 2 owned two of these and neither was made: actionableinventory.go:20-22 (the constant''s own doc still says "a park transaction in flight ... it resolves on its own") and layout.go:79-80 (holdsSession''s rationale). Three more are M2/Task 4''s: menu.go:1234-1239, menu_render.go:432 ("parking…"), couchcmd/run.go:770 ("parking in progress"). Task 4''s premise that the busy row is now unreachable is also wrong.'
          family: stale-wording-after-referent-change
          round: 1
        - id: BR-8
          severity: Important
          title: startClaimed reads durable state where the plan specified an in-memory observation
          detail: 'actionableinventory.go:288,341-355 reads Incarnation.Start off the record; the plan says ephemeral state stays ephemeral and the branch table row 3 says "(in-memory observation)". No ## Revisions entry records the change. Residual risk: ReconcileStart keeps a claim occupied on Unknown evidence (starttransaction.go:186-210), so such a record reads busy indefinitely and menu.go:1234 offers it only name/describe.'
          family: plan-code-divergence
          round: 1
        - id: BR-9
          severity: Minor
          title: Two tests assert something other than what their names say
          detail: threadreason_test.go:67-70 (TestStaleLabelDoesNotClaimSupervisorDied now checks ReasonSessionGone's label; its subject is gone) and sessionevidence_test.go:85-93 (TestAbsentBindingIsAbsentNotUnresolved asserts Unresolved).
          family: test-name-contradicts-assertion
          round: 1
        - id: BR-10
          severity: Minor
          title: The session name-index and uniqueness predicate are copy-pasted across two projectors
          detail: sessionevidence.go:95-118 and detachedsessions.go:62-82 duplicate the ambiguity loop and `name == "" || claims[name] != 1 || ambiguous[name]`. The diff extracted the shared READ (resolveScopedBindings) but not the shared RULE; divergence in it would be silent.
          family: duplicated-fail-closed-rule
          round: 1
      boundary: M1
      blocked: true
    - "n": 2
      timestamp: "2026-09-17T09:33:58-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Mutation-verified: disabling the park-abandon block reds TestOrphanedParkDoesNotWedgeTheResumeChain; un-wrapping the AbandonPark error reds TestResumeFailuresCarryADiagnosticCode. See new finding for the fifth site of the same class.'
          round: 2
        - id: BR-2
          disposition: addressed
          note: 'Mutation-verified: moving the SessionUnresolved arm back above the VerifiedPark branch reds TestParkedRowSurvivesAnUnresolvedSessionQuestion/session_question_failed. Checked the third outcome (ReasonBindingLost) is not archive-eligible.'
          round: 2
        - id: BR-3
          disposition: addressed
          note: 'Mutation-verified: `!= Dead` -> `== Live` reds TestUnknownLivenessNeverRetiresAnIncarnation/unknown_must_not_retire.'
          round: 2
        - id: BR-4
          disposition: addressed
          note: 'Mutation-verified: dropping the readable[scope] check reds TestSessionPresenceAnswersThroughTheProductionChecker on the unreadable-scope case.'
          round: 2
        - id: BR-5
          disposition: addressed
          note: atlas/couch.md:251, :579-592 and :961-968 corrected; the vocabulary list now matches AllThreadReasons() element for element. Remaining mentions are the retirement narrative itself.
          round: 2
        - id: BR-6
          disposition: addressed
          note: README.md:517-531 corrected; every label it now names exists in ThreadReason.Label(), and the park claim matches ClassifyThread no longer reading record.Park.
          round: 2
        - id: BR-7
          disposition: addressed
          note: Four of five made (actionableinventory.go:21-30, layout.go:79-82, menu_render.go:432-434, couchcmd/run.go:771). menu.go:1234-1239 remains, which the finding itself scoped to M2/Task 4; the plan's Revisions re-scopes Task 4 off its false premise.
          round: 2
        - id: BR-8
          disposition: addressed
          note: 'Plan ## Revisions now adopts the durable read explicitly, states the bound (reconcileInterruptedStarts at couch.New) and the Unknown-evidence residual risk, and re-scopes Task 4.'
          round: 2
        - id: BR-9
          disposition: addressed
          note: TestStaleLabelDoesNotClaimSupervisorDied deleted; the projector test renamed to TestProjectionAnswersOnlyForBindingsItWasGiven. A third instance of the same rule is raised below.
          round: 2
        - id: BR-10
          disposition: addressed
          note: indexSessionsByName + uniquelyClaimed extracted and used by both projectors; I diffed both call sites by hand and the predicate is exactly equivalent to the two copies.
          round: 2
      findings:
        - id: BR-11
          severity: Critical
          title: An Unknown-liveness incarnation with a surviving session wedges couch startup in the whole tree
          detail: '2nd in family — fix the RULE, not this site. Reproduced end to end through StartInteractive: a record whose launcher is unobservable plus a live session classifies detached, is ranked highest by SelectResumableRoot, passes DecideResume, is correctly declined by the Dead-only retire gate, and then CommitStartClaim (threadstore.go:506) refuses with a bare fmt.Errorf. startupResumeRefusal only decorates coded errors, so couch refuses to start with "thread {...} already has 1 incarnation(s)". The identical fixture at base b6a0766 spawns normally, so this is a regression from this milestone. Two one-place rules — every error leaving ResumeContextWith must carry a ResumeDiagnosticCode (CommitStartClaim, resolveRepoIdentity, Proc.Current, allocateStartNonce, the DetachedSessions observe error and the missing-resolver error all escape uncoded today), and the class enumeration must be written down as "every guard refusing on record.Incarnations or record.Park", which includes CommitStartClaim''s own precondition on the non-Dead branch.'
          family: classification-not-authority
          round: 2
        - id: BR-12
          severity: Critical
          title: The park abandon runs before RetireIncarnation's preconditions, so a failure destroys the park permanently
          detail: 'retireDeadIncarnationBeforeStart (resume.go:580) screens Start/PID/Identity but not incarnation.State, while RetireIncarnation (threadstore.go:557) refuses anything other than IncarnationLive. Reproduced against the real store with an IncarnationUnknown record carrying a matching open park and a dead process: the abandon lands (park nil, one tombstone appended), the retire then fails with "retire needs a live incarnation, found unknown", and every retry repeats it. The thread can no longer resume and cannot archive either (thread.go:360 refuses an unknown helper) — a new instance of the wedge this issue exists to remove, plus loss of the #275 audit trail. Reachable: soleParkableIncarnation (park.go:797) explicitly permits parking an unknown incarnation, and markLiveRecordUnknown (couch.go:648) produces that state. Fix: hoist the retirement''s preconditions ahead of the irreversible write. Related ARCH-ORDER note: this AbandonPark bypasses PairLifecycleController''s per-thread park worker, unlike every other abandon.'
          family: irreversible-step-before-precondition
          round: 2
        - id: BR-13
          severity: Important
          title: Two of the three guards in retireDeadIncarnationBeforeStart are unpinned
          detail: '2nd in family — fix the RULE, not these sites. Mutation-checked: reverting resume.go:601 from refuseResume(ResumeNotRunning, ...) to `return nil, err` produces no failure anywhere in couchcore, because TestResumeFailuresCarryADiagnosticCode only reaches the AbandonPark arm. The park-identity-mismatch refusal (resume.go:581-586) has no fixture at all — both park tests give the park the same identity as the incarnation. Measured prevalence: three fail-closed exits in one function, one pinned. The rule is one table test over {no park, matching park, foreign park} x {Dead, Unknown, Live} x {Live, Unknown incarnation} asserting durable state plus a non-empty ResumeDiagnosticOf for every refusal — which also gives the two Critical findings their regressions and fails whenever a new exit is added.'
          family: fail-closed-guard-untested
          round: 2
        - id: BR-14
          severity: Minor
          title: run_test.go failure message still says stale-incarnation while the assertion checks session-gone
          detail: '2nd in family — state the rule. cmd/internal/couchcmd/run_test.go:1238 reads "want unusable/stale-incarnation after the child exited" under an assertion on ReasonSessionGone. Rule: a test''s prose — name, comment and failure message — must name what it asserts. Sweep by grepping the window''s touched test files for "stale"/"unrecorded" inside string literals; that finds this one and nothing else.'
          family: test-name-contradicts-assertion
          round: 2
        - id: BR-15
          severity: Minor
          title: DetachedSessions' doc comment was orphaned by the resolveScopedBindings extraction
          detail: '2nd in family — state the rule. artifactcollision.go:262-278 is DetachedSessions'' doc (cost model, "Pinned by TestDetachedSessionsBindsNothingForAnUnreadableScope") but now runs straight into resolveScopedBindings'' own doc with no blank line, so godoc attaches the whole block to the private helper, the fail-closed paragraph appears twice verbatim, and DetachedSessions at :408 has no doc at all. Confirmed against the base version, where the comment sat on DetachedSessions. Rule: when a function moves or is split, its doc comment moves with it.'
          family: stale-wording-after-referent-change
          round: 2
        - id: BR-16
          severity: Minor
          title: The plan's Integration and Pure tables still name entities the code does not ship
          detail: '2nd in family — state the rule. The Integration table names `observeSessions` in sessionevidence.go; the code ships SessionPresenceResolver.SessionPresence (production impl in artifactcollision.go:370) plus ProjectSessionPresence. startClaimed, indexSessionsByName, uniquelyClaimed and retireDeadIncarnationBeforeStart appear in no table. Round 1 raised this as a recommendation and it survived. Rule: at each milestone close, re-derive the Core-concepts tables by grepping every row''s name and path, and record any divergence in ## Revisions rather than leaving the plan claiming what the code does not deliver.'
          family: plan-code-divergence
          round: 2
      boundary: M1
      blocked: true
---

# Gate ledger — pair#256 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-17T08:50:28-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Critical] `classification-not-authority` An open park makes the new re-adoption hard-error and wedges couch startup
  A record with Park != nil, a dead incarnation and a surviving session now classifies detached (actionableinventory.go:316), passes DecideResume, then hits RetireIncarnation's park precondition (threadstore.go:551) via retireDeadIncarnationBeforeStart (resume.go:555-568), which returns the raw store error. startup.go:203-232 has no fallback and only decorates errors carrying a ResumeDiagnosticCode, so couch refuses to start in that tree with an internal message. Reproduced end to end against the real store. This is the fourth site of the class the Log enumerated as three.
- **BR-2** [Important] `unknown-must-not-mask-durable-authority` SessionUnresolved is checked above VerifiedPark, so one failed list-sessions hides every parked row
  actionableinventory.go:321-324 returns unusable/unknown before the cold-resume branch at :328. Verified: a fully-proved parked record classifies parked under SessionAbsent and unusable/unknown under SessionUnresolved. A parked record's resume authority is durable and needs no session answer, and the only two verdicts the session could produce are parked and detached, both actionable.
- **BR-3** [Important] `fail-closed-guard-untested` Nothing pins that an Unknown liveness probe must not retire an incarnation
  Mutating resume.go:564 from `!= Dead` to `== Live` (retire on Unknown) produces no test failure across couchcore, couchtty and couchcmd. This is the guard preventing a resume from abandoning a running agent, and it is the issue's own Done-when about Unknown never authorizing destructive recovery.
- **BR-4** [Important] `production-seam-only-tested-through-fake` ScopedThreadArtifactCollisionChecker.SessionPresence has no test
  Every presence test runs against the fake. The production implementation (artifactcollision.go:370-410), including the readable-scope-no-binding then SessionAbsent versus unreadable-scope then SessionUnresolved branch that decides archive-eligibility, is unexercised. The sandboxedChecker harness (artifactcollision_zellij_test.go:29) already stubs zellij for exactly this.
- **BR-5** [Important] `atlas-contradicts-code` Three atlas passages still document the retired reason vocabulary as current
  atlas/couch.md:584-585 lists stale-incarnation and unrecorded-child in the closed vocabulary and says ThreadEvidence carries a ProofStatus per question; :963-965 says a stale IncarnationLive shows as unusable/stale-incarnation, the exact sentence this issue disproves; :251 references the shape in passing. The new section at :1277 says the opposite.
- **BR-6** [Important] `readme-documents-removed-surface` README still lists a deleted reason label and claims parks leave a diagnostic
  README.md:527 names `stale — helper ownership unresolved` among the labels an operator sees; that label was deleted here. README.md:517-518 says open start or park transactions leave a diagnostic instead of guessing a process died, which is no longer true of parks since nothing in the classification path reads record.Park.
- **BR-7** [Important] `stale-wording-after-referent-change` ThreadBusy changed meaning but five wording sites still describe a park
  Task 2 owned two of these and neither was made: actionableinventory.go:20-22 (the constant's own doc still says "a park transaction in flight ... it resolves on its own") and layout.go:79-80 (holdsSession's rationale). Three more are M2/Task 4's: menu.go:1234-1239, menu_render.go:432 ("parking…"), couchcmd/run.go:770 ("parking in progress"). Task 4's premise that the busy row is now unreachable is also wrong.
- **BR-8** [Important] `plan-code-divergence` startClaimed reads durable state where the plan specified an in-memory observation
  actionableinventory.go:288,341-355 reads Incarnation.Start off the record; the plan says ephemeral state stays ephemeral and the branch table row 3 says "(in-memory observation)". No ## Revisions entry records the change. Residual risk: ReconcileStart keeps a claim occupied on Unknown evidence (starttransaction.go:186-210), so such a record reads busy indefinitely and menu.go:1234 offers it only name/describe.
- **BR-9** [Minor] `test-name-contradicts-assertion` Two tests assert something other than what their names say
  threadreason_test.go:67-70 (TestStaleLabelDoesNotClaimSupervisorDied now checks ReasonSessionGone's label; its subject is gone) and sessionevidence_test.go:85-93 (TestAbsentBindingIsAbsentNotUnresolved asserts Unresolved).
- **BR-10** [Minor] `duplicated-fail-closed-rule` The session name-index and uniqueness predicate are copy-pasted across two projectors
  sessionevidence.go:95-118 and detachedsessions.go:62-82 duplicate the ambiguity loop and `name == "" || claims[name] != 1 || ambiguous[name]`. The diff extracted the shared READ (resolveScopedBindings) but not the shared RULE; divergence in it would be silent.

## Round 2 — 2026-09-17T09:33:58-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Mutation-verified: disabling the park-abandon block reds TestOrphanedParkDoesNotWedgeTheResumeChain; un-wrapping the AbandonPark error reds TestResumeFailuresCarryADiagnosticCode. See new finding for the fifth site of the same class.
- BR-2 — addressed — Mutation-verified: moving the SessionUnresolved arm back above the VerifiedPark branch reds TestParkedRowSurvivesAnUnresolvedSessionQuestion/session_question_failed. Checked the third outcome (ReasonBindingLost) is not archive-eligible.
- BR-3 — addressed — Mutation-verified: `!= Dead` -> `== Live` reds TestUnknownLivenessNeverRetiresAnIncarnation/unknown_must_not_retire.
- BR-4 — addressed — Mutation-verified: dropping the readable[scope] check reds TestSessionPresenceAnswersThroughTheProductionChecker on the unreadable-scope case.
- BR-5 — addressed — atlas/couch.md:251, :579-592 and :961-968 corrected; the vocabulary list now matches AllThreadReasons() element for element. Remaining mentions are the retirement narrative itself.
- BR-6 — addressed — README.md:517-531 corrected; every label it now names exists in ThreadReason.Label(), and the park claim matches ClassifyThread no longer reading record.Park.
- BR-7 — addressed — Four of five made (actionableinventory.go:21-30, layout.go:79-82, menu_render.go:432-434, couchcmd/run.go:771). menu.go:1234-1239 remains, which the finding itself scoped to M2/Task 4; the plan's Revisions re-scopes Task 4 off its false premise.
- BR-8 — addressed — Plan ## Revisions now adopts the durable read explicitly, states the bound (reconcileInterruptedStarts at couch.New) and the Unknown-evidence residual risk, and re-scopes Task 4.
- BR-9 — addressed — TestStaleLabelDoesNotClaimSupervisorDied deleted; the projector test renamed to TestProjectionAnswersOnlyForBindingsItWasGiven. A third instance of the same rule is raised below.
- BR-10 — addressed — indexSessionsByName + uniquelyClaimed extracted and used by both projectors; I diffed both call sites by hand and the predicate is exactly equivalent to the two copies.

### Raised

- **BR-11** [Critical] `classification-not-authority` An Unknown-liveness incarnation with a surviving session wedges couch startup in the whole tree
  2nd in family — fix the RULE, not this site. Reproduced end to end through StartInteractive: a record whose launcher is unobservable plus a live session classifies detached, is ranked highest by SelectResumableRoot, passes DecideResume, is correctly declined by the Dead-only retire gate, and then CommitStartClaim (threadstore.go:506) refuses with a bare fmt.Errorf. startupResumeRefusal only decorates coded errors, so couch refuses to start with "thread {...} already has 1 incarnation(s)". The identical fixture at base b6a0766 spawns normally, so this is a regression from this milestone. Two one-place rules — every error leaving ResumeContextWith must carry a ResumeDiagnosticCode (CommitStartClaim, resolveRepoIdentity, Proc.Current, allocateStartNonce, the DetachedSessions observe error and the missing-resolver error all escape uncoded today), and the class enumeration must be written down as "every guard refusing on record.Incarnations or record.Park", which includes CommitStartClaim's own precondition on the non-Dead branch.
- **BR-12** [Critical] `irreversible-step-before-precondition` The park abandon runs before RetireIncarnation's preconditions, so a failure destroys the park permanently
  retireDeadIncarnationBeforeStart (resume.go:580) screens Start/PID/Identity but not incarnation.State, while RetireIncarnation (threadstore.go:557) refuses anything other than IncarnationLive. Reproduced against the real store with an IncarnationUnknown record carrying a matching open park and a dead process: the abandon lands (park nil, one tombstone appended), the retire then fails with "retire needs a live incarnation, found unknown", and every retry repeats it. The thread can no longer resume and cannot archive either (thread.go:360 refuses an unknown helper) — a new instance of the wedge this issue exists to remove, plus loss of the #275 audit trail. Reachable: soleParkableIncarnation (park.go:797) explicitly permits parking an unknown incarnation, and markLiveRecordUnknown (couch.go:648) produces that state. Fix: hoist the retirement's preconditions ahead of the irreversible write. Related ARCH-ORDER note: this AbandonPark bypasses PairLifecycleController's per-thread park worker, unlike every other abandon.
- **BR-13** [Important] `fail-closed-guard-untested` Two of the three guards in retireDeadIncarnationBeforeStart are unpinned
  2nd in family — fix the RULE, not these sites. Mutation-checked: reverting resume.go:601 from refuseResume(ResumeNotRunning, ...) to `return nil, err` produces no failure anywhere in couchcore, because TestResumeFailuresCarryADiagnosticCode only reaches the AbandonPark arm. The park-identity-mismatch refusal (resume.go:581-586) has no fixture at all — both park tests give the park the same identity as the incarnation. Measured prevalence: three fail-closed exits in one function, one pinned. The rule is one table test over {no park, matching park, foreign park} x {Dead, Unknown, Live} x {Live, Unknown incarnation} asserting durable state plus a non-empty ResumeDiagnosticOf for every refusal — which also gives the two Critical findings their regressions and fails whenever a new exit is added.
- **BR-14** [Minor] `test-name-contradicts-assertion` run_test.go failure message still says stale-incarnation while the assertion checks session-gone
  2nd in family — state the rule. cmd/internal/couchcmd/run_test.go:1238 reads "want unusable/stale-incarnation after the child exited" under an assertion on ReasonSessionGone. Rule: a test's prose — name, comment and failure message — must name what it asserts. Sweep by grepping the window's touched test files for "stale"/"unrecorded" inside string literals; that finds this one and nothing else.
- **BR-15** [Minor] `stale-wording-after-referent-change` DetachedSessions' doc comment was orphaned by the resolveScopedBindings extraction
  2nd in family — state the rule. artifactcollision.go:262-278 is DetachedSessions' doc (cost model, "Pinned by TestDetachedSessionsBindsNothingForAnUnreadableScope") but now runs straight into resolveScopedBindings' own doc with no blank line, so godoc attaches the whole block to the private helper, the fail-closed paragraph appears twice verbatim, and DetachedSessions at :408 has no doc at all. Confirmed against the base version, where the comment sat on DetachedSessions. Rule: when a function moves or is split, its doc comment moves with it.
- **BR-16** [Minor] `plan-code-divergence` The plan's Integration and Pure tables still name entities the code does not ship
  2nd in family — state the rule. The Integration table names `observeSessions` in sessionevidence.go; the code ships SessionPresenceResolver.SessionPresence (production impl in artifactcollision.go:370) plus ProjectSessionPresence. startClaimed, indexSessionsByName, uniquelyClaimed and retireDeadIncarnationBeforeStart appear in no table. Round 1 raised this as a recommendation and it survived. Rule: at each milestone close, re-derive the Core-concepts tables by grepping every row's name and path, and record any divergence in ## Revisions rather than leaving the plan claiming what the code does not deliver.

## Open findings

- **BR-11** [Critical] `classification-not-authority` An Unknown-liveness incarnation with a surviving session wedges couch startup in the whole tree
- **BR-12** [Critical] `irreversible-step-before-precondition` The park abandon runs before RetireIncarnation's preconditions, so a failure destroys the park permanently
- **BR-13** [Important] `fail-closed-guard-untested` Two of the three guards in retireDeadIncarnationBeforeStart are unpinned
- **BR-14** [Minor] `test-name-contradicts-assertion` run_test.go failure message still says stale-incarnation while the assertion checks session-gone
- **BR-15** [Minor] `stale-wording-after-referent-change` DetachedSessions' doc comment was orphaned by the resolveScopedBindings extraction
- **BR-16** [Minor] `plan-code-divergence` The plan's Integration and Pure tables still name entities the code does not ship
