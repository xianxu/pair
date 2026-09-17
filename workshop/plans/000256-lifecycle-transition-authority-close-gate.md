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
    - "n": 3
      timestamp: "2026-09-17T10:07:45-07:00"
      agent: claude
      dispose:
        - id: BR-11
          disposition: addressed
          note: 'Mutation-proven: deleting the deferred coder at resume.go:360 reddens both exits of TestEveryResumeFailureCarriesADiagnosticCode. See I1 for the consumer the widened meaning broke.'
          round: 3
        - id: BR-12
          disposition: addressed
          note: 'Mutation-proven: moving the State guard below AbandonPark reddens matching-park/dead/unknown with the permanent-tombstone assertion. Bypass note carried forward as a Minor.'
          round: 3
        - id: BR-13
          disposition: addressed
          note: Twelve cells asserting wantRetire, a non-empty diagnostic on every refusal, and park survival on refusal; the start-in-flight exit remains outside the table (see test notes).
          round: 3
        - id: BR-14
          disposition: addressed
          note: run_test.go:1239 now reads session-gone; swept the window's touched test files for stale/unrecorded string literals and the remaining hits are ordinary prose.
          round: 3
        - id: BR-15
          disposition: addressed
          note: DetachedSessions has its own doc at artifactcollision.go:374 and the duplicated fail-closed paragraph is gone; residual noted as a Minor (resolveScopedBindings now has no doc at all).
          round: 3
        - id: BR-16
          disposition: addressed
          note: Tables re-derived; observeSessions removed, startClaimed / resolveScopedBindings / retireDeadIncarnationBeforeStart added, and every row's name and path greps to the stated file.
          round: 3
      findings:
        - id: BR-17
          severity: Important
          title: The blanket resume coder changed what "carries a code" means and the reattach pass still branches on the old meaning
          detail: '3rd in family — fix the RULE, not this site. resume.go:360-365 now gives every non-cancellation error a ResumeDiagnosticCode, so "carries a code" changed from "is a refusal" to "came out of resume". menu_reattach.go:244-252 still reads it the old way: its cell-7 branch falls back to firstErrorLine only when the code is empty, so a background reattach that fails on a spawn error or registration timeout now renders "reattach failed: resume-unknown" (menu_render.go:710) instead of the error''s first line, which decision 11 specified. The wrap also rebuilds the error from retErr.Error(), so errors.As, errors.Is(..., context.DeadlineExceeded) and Unwrap() []error no longer see through it (console_completion.go:133 walks joined errors). The rule: when a value''s MEANING changes, enumerate every reader and re-derive each in the same round. The enumeration is five sites — resume.go:138, relaunch.go:122, startup.go:230, console.go:1767, menu_reattach.go:239. A second live instance of the same rule: actionableinventory.go:472-474 still asserts that a record carrying an incarnation "must not start to" be physicalized, directly above the code that now does.'
          family: stale-wording-after-referent-change
          round: 3
        - id: BR-18
          severity: Important
          title: SessionPresenceResolver is reached only through a silently-failing type assertion with no compile-time binding
          detail: '2nd in family — state the rule. actionableinventory.go:555 does c.Artifacts.(SessionPresenceResolver) and drops the ok, there is no var _ SessionPresenceResolver = ScopedThreadArtifactCollisionChecker{}, and no test builds a Couch over the production checker (TestSessionPresenceAnswersThroughTheProductionChecker calls the method directly; the couchcmd acceptance uses the fake). Both types satisfy the interface today — verified by compiling the assertion in a scratch copy — so nothing is broken now. The exposure is that a receiver or signature drift compiles, passes every test, and turns every row in every tree into unusable/checking…, because the fake still satisfies it. The rule: every interface consumed via a runtime assertion on c.Artifacts carries a compile-time var _ against the production type, beside the existing fake assertion. Measured prevalence: five such interfaces, one pinned (PairSessionIO at park.go:812).'
          family: production-seam-only-tested-through-fake
          round: 3
        - id: BR-19
          severity: Important
          title: The atlas records "One class, three sites" while the code and the plan record four
          detail: '2nd in family — state the rule. atlas/couch.md:1328 heads the section "One class, three sites" and lists ClassifyThread, DecideResume and CommitStartClaim''s caller. resume.go:615 says "This is the FOURTH site of the class" and names the enumeration including the orphaned-park abandon, and the plan''s Revisions (2026-09-17 round 1) says "The class has FOUR sites, not three". The issue''s Log still says three as well. The omitted member is the one carrying the irreversible-ordering rule, so a reader taking the atlas as the map gets the enumeration minus its most dangerous entry. The rule: when a boundary round extends an enumeration, every home of it — code comment, plan Revisions, atlas, issue Log — is updated in that same round, not at a milestone-end sweep. Round 1 updated three of the four homes.'
          family: atlas-contradicts-code
          round: 3
        - id: BR-20
          severity: Minor
          title: TestClassifyThreadAcceptsExactlyWhatTheOldProjectorAccepted now admits four deliberately new shapes
          detail: '3rd in family. classify_test.go:268-276: the name says "exactly what the old projector accepted", the doc says "except #248 intentionally admits unbound warm sessions", and the failure message prints tc.wasActionableBefore alone — while the body asserts (wasActionableBefore || newlyActionable) over four #256 shapes. The rule is BR-14''s, widened: a test''s name, doc and failure message must all describe the assertion the body makes, and when the assertion''s scope changes all three are restated in the same edit.'
          family: test-name-contradicts-assertion
          round: 3
        - id: BR-21
          severity: Minor
          title: The re-adoption's AbandonPark bypasses the per-thread park worker every other abandon goes through
          detail: resume.go:627 calls c.Threads.AbandonPark directly; PairLifecycleController.Abandon (park.go:435-450) routes the same store call through submit, which serializes per thread. The revision CAS keeps this safe — a loser gets a coded refusal, not corruption — but park abandonment now has two authorities, and RecoverActiveParks (run.go:348) runs concurrently with startup resume over the same records (ARCH-ORDER).
          family: transition-bypasses-its-owner
          round: 3
        - id: BR-22
          severity: Minor
          title: ResumeCreating lost its only producer and the resume diagnostic vocabulary has no produced-by guard
          detail: Deleting occupiedResumeCode removed the only site emitting ResumeCreating (resume.go:16); ResumeLive survives via relaunch.go:107. ThreadReason has TestEveryReasonIsProducedBySomeShape for exactly this class — threadreason.go's own comment cites it as the reason unrecorded-child was deleted rather than kept as a placeholder — but ResumeDiagnosticCode has no equivalent guard, which is why the orphan went unnoticed in the same commit that deleted its producer.
          family: vocabulary-entry-without-producer
          round: 3
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

## Round 3 — 2026-09-17T10:07:45-07:00 (claude) — BLOCKED

### Disposed

- BR-11 — addressed — Mutation-proven: deleting the deferred coder at resume.go:360 reddens both exits of TestEveryResumeFailureCarriesADiagnosticCode. See I1 for the consumer the widened meaning broke.
- BR-12 — addressed — Mutation-proven: moving the State guard below AbandonPark reddens matching-park/dead/unknown with the permanent-tombstone assertion. Bypass note carried forward as a Minor.
- BR-13 — addressed — Twelve cells asserting wantRetire, a non-empty diagnostic on every refusal, and park survival on refusal; the start-in-flight exit remains outside the table (see test notes).
- BR-14 — addressed — run_test.go:1239 now reads session-gone; swept the window's touched test files for stale/unrecorded string literals and the remaining hits are ordinary prose.
- BR-15 — addressed — DetachedSessions has its own doc at artifactcollision.go:374 and the duplicated fail-closed paragraph is gone; residual noted as a Minor (resolveScopedBindings now has no doc at all).
- BR-16 — addressed — Tables re-derived; observeSessions removed, startClaimed / resolveScopedBindings / retireDeadIncarnationBeforeStart added, and every row's name and path greps to the stated file.

### Raised

- **BR-17** [Important] `stale-wording-after-referent-change` The blanket resume coder changed what "carries a code" means and the reattach pass still branches on the old meaning
  3rd in family — fix the RULE, not this site. resume.go:360-365 now gives every non-cancellation error a ResumeDiagnosticCode, so "carries a code" changed from "is a refusal" to "came out of resume". menu_reattach.go:244-252 still reads it the old way: its cell-7 branch falls back to firstErrorLine only when the code is empty, so a background reattach that fails on a spawn error or registration timeout now renders "reattach failed: resume-unknown" (menu_render.go:710) instead of the error's first line, which decision 11 specified. The wrap also rebuilds the error from retErr.Error(), so errors.As, errors.Is(..., context.DeadlineExceeded) and Unwrap() []error no longer see through it (console_completion.go:133 walks joined errors). The rule: when a value's MEANING changes, enumerate every reader and re-derive each in the same round. The enumeration is five sites — resume.go:138, relaunch.go:122, startup.go:230, console.go:1767, menu_reattach.go:239. A second live instance of the same rule: actionableinventory.go:472-474 still asserts that a record carrying an incarnation "must not start to" be physicalized, directly above the code that now does.
- **BR-18** [Important] `production-seam-only-tested-through-fake` SessionPresenceResolver is reached only through a silently-failing type assertion with no compile-time binding
  2nd in family — state the rule. actionableinventory.go:555 does c.Artifacts.(SessionPresenceResolver) and drops the ok, there is no var _ SessionPresenceResolver = ScopedThreadArtifactCollisionChecker{}, and no test builds a Couch over the production checker (TestSessionPresenceAnswersThroughTheProductionChecker calls the method directly; the couchcmd acceptance uses the fake). Both types satisfy the interface today — verified by compiling the assertion in a scratch copy — so nothing is broken now. The exposure is that a receiver or signature drift compiles, passes every test, and turns every row in every tree into unusable/checking…, because the fake still satisfies it. The rule: every interface consumed via a runtime assertion on c.Artifacts carries a compile-time var _ against the production type, beside the existing fake assertion. Measured prevalence: five such interfaces, one pinned (PairSessionIO at park.go:812).
- **BR-19** [Important] `atlas-contradicts-code` The atlas records "One class, three sites" while the code and the plan record four
  2nd in family — state the rule. atlas/couch.md:1328 heads the section "One class, three sites" and lists ClassifyThread, DecideResume and CommitStartClaim's caller. resume.go:615 says "This is the FOURTH site of the class" and names the enumeration including the orphaned-park abandon, and the plan's Revisions (2026-09-17 round 1) says "The class has FOUR sites, not three". The issue's Log still says three as well. The omitted member is the one carrying the irreversible-ordering rule, so a reader taking the atlas as the map gets the enumeration minus its most dangerous entry. The rule: when a boundary round extends an enumeration, every home of it — code comment, plan Revisions, atlas, issue Log — is updated in that same round, not at a milestone-end sweep. Round 1 updated three of the four homes.
- **BR-20** [Minor] `test-name-contradicts-assertion` TestClassifyThreadAcceptsExactlyWhatTheOldProjectorAccepted now admits four deliberately new shapes
  3rd in family. classify_test.go:268-276: the name says "exactly what the old projector accepted", the doc says "except #248 intentionally admits unbound warm sessions", and the failure message prints tc.wasActionableBefore alone — while the body asserts (wasActionableBefore || newlyActionable) over four #256 shapes. The rule is BR-14's, widened: a test's name, doc and failure message must all describe the assertion the body makes, and when the assertion's scope changes all three are restated in the same edit.
- **BR-21** [Minor] `transition-bypasses-its-owner` The re-adoption's AbandonPark bypasses the per-thread park worker every other abandon goes through
  resume.go:627 calls c.Threads.AbandonPark directly; PairLifecycleController.Abandon (park.go:435-450) routes the same store call through submit, which serializes per thread. The revision CAS keeps this safe — a loser gets a coded refusal, not corruption — but park abandonment now has two authorities, and RecoverActiveParks (run.go:348) runs concurrently with startup resume over the same records (ARCH-ORDER).
- **BR-22** [Minor] `vocabulary-entry-without-producer` ResumeCreating lost its only producer and the resume diagnostic vocabulary has no produced-by guard
  Deleting occupiedResumeCode removed the only site emitting ResumeCreating (resume.go:16); ResumeLive survives via relaunch.go:107. ThreadReason has TestEveryReasonIsProducedBySomeShape for exactly this class — threadreason.go's own comment cites it as the reason unrecorded-child was deleted rather than kept as a placeholder — but ResumeDiagnosticCode has no equivalent guard, which is why the orphan went unnoticed in the same commit that deleted its producer.

## Open findings

- **BR-17** [Important] `stale-wording-after-referent-change` The blanket resume coder changed what "carries a code" means and the reattach pass still branches on the old meaning
- **BR-18** [Important] `production-seam-only-tested-through-fake` SessionPresenceResolver is reached only through a silently-failing type assertion with no compile-time binding
- **BR-19** [Important] `atlas-contradicts-code` The atlas records "One class, three sites" while the code and the plan record four
- **BR-20** [Minor] `test-name-contradicts-assertion` TestClassifyThreadAcceptsExactlyWhatTheOldProjectorAccepted now admits four deliberately new shapes
- **BR-21** [Minor] `transition-bypasses-its-owner` The re-adoption's AbandonPark bypasses the per-thread park worker every other abandon goes through
- **BR-22** [Minor] `vocabulary-entry-without-producer` ResumeCreating lost its only producer and the resume diagnostic vocabulary has no produced-by guard
