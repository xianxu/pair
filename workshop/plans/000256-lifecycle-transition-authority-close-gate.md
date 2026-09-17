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
    - "n": 4
      timestamp: "2026-09-17T10:47:04-07:00"
      agent: claude
      dispose:
        - id: BR-17
          disposition: addressed
          note: Blanket deferred coder gone from resume.go (only the pre-existing retention join at :359 remains, present at base); menu_reattach.go:244-252 reads an empty code correctly again; TestResumeCodeStillMeansAStructuredRefusal and TestEveryStartupResumeFailureIsActionable pin both halves including errors.Is through the decoration. The second instance (actionableinventory.go "must not start to") is rewritten.
          round: 4
        - id: BR-18
          disposition: addressed
          note: 'artifactcollision.go:320-331 binds SessionPresenceResolver, DetachedSessionResolver, NativeBindingResolver and PairSessionIO on the production type plus two on the fake, with the silent-failure rationale in the comment. Residual not re-raised: contextPairSessionObserver (recovery_execute.go:16) and couchcmd/run.go:125''s anonymous interface stay unpinned, but both have an explicit non-context fallback rather than degrading to universal unknown.'
          round: 4
        - id: BR-19
          disposition: not-addressed
          note: atlas/couch.md:1330 now says "One class, four sites" — correct. The issue Log, named in the finding as the fourth home and again in round 3's own notes (m1-review.md:476), still says three at issue lines 123, 133 and 141, with an enumeration that omits the site carrying the irreversible-ordering rule. See I1.
          round: 4
        - id: BR-20
          disposition: not-addressed
          note: 'classify_test.go untouched in 54d36c37; name, doc (still citing a #248 case that no longer exists) and failure message all still disagree with a body admitting three newlyActionable #256 shapes. The guard added this round, TestEveryResumeDiagnosticCodeIsProducedBySomeSite, is a fresh instance of the same rule — it counts substring mentions, so a code with comment mentions and no producer passes.'
          round: 4
        - id: BR-21
          disposition: not-addressed
          note: resume.go:608 still calls c.Threads.AbandonPark directly, bypassing PairLifecycleController.Abandon's per-thread worker; no change in park.go or run.go. Minor, CAS-protected.
          round: 4
        - id: BR-22
          disposition: addressed
          note: ResumeCreating deleted and TestEveryResumeDiagnosticCodeIsProducedBySomeSite derives its identifiers from the declaration, so re-adding an unproduced code reddens it. Soundness gap in the guard itself recorded under BR-20 rather than re-raised here.
          round: 4
      findings:
        - id: BR-23
          severity: Critical
          title: The orphaned-park abandon probes one process and writes a permanent tombstone about another
          detail: '2nd in family — fix the RULE, not this site. resume.go:600-612 abandons thread.Park after probing thread.Incarnations[0], omitting the identity check on the claim that a park owned by another process is unrepresentable. threadrecord/lifecycle.go:91 permits matches==0 when Phase is "unknown" and the transaction carries a replacement_incarnation failure; park.go:653-662 produces exactly that, and threadrecord/record_test.go:303 pins it as valid. Confirmed by execution in a scratch copy: a record created through the production ThreadStore with a live incarnation {99,"replacement"} and a park owned by {42,"original-owner"}, with pid 42 ALIVE, retires cleanly and abandons pid 42''s park — one permanent tombstone, retireErr nil. The live owner''s later FinalizePark then fails with "park abandon identity does not match active transaction", so the park silently never completes and #275''s audit trail for it is gone. The rule: an irreversible step''s precondition must be proved about the exact entity the step acts on, and a guard omitted as "unrepresentable" must cite the validator clause that makes it so, read including its exceptions, and be pinned by a test that tries to build the fixture through the real store. Enumerable sibling: lifecycle.go:97-105''s resumeOccupied escape. Also correct sessionevidence_test.go:372''s exclusion note and plan:842.'
          family: irreversible-step-before-precondition
          round: 4
        - id: BR-24
          severity: Important
          title: A changed referent left six unre-derived sites, one of which renders a false diagnostic to the operator
          detail: '4th in family — fix the RULE, not these sites. Behavioural member first: ResumeNotRunning is declared (resume.go:31-34) as "not running at all … the OPPOSITE of ResumeLive" and is emitted at :569 for "could not be proved dead" and at :624 for a store error; menu_reattach.go:239 skips only ResumeNotDetached/ResumeSessionGone, so a background reattach of a #272 row with an unobservable launcher renders "resume-not-running" on a row whose agent is running, and no test pins which code that exit emits. Doc-only members: startup.go:137-139 ("a candidate outside both sets keeps ProofUnresolved and classifies unknown" — presence is now gathered after the ask gate, so it classifies detached); actionableinventory.go:419-421 restating it despite startup.go:124-126 declaring itself the one home; actionableinventory.go:376 ("shared by inventory" — the inventory no longer calls it); artifactcollision_fake.go:59-60 (says unset reads unresolved, code returns SessionAbsent since b5fce898); startup.go:69 (occupiedIncarnation "shared by archive and resume" — resume no longer reads it). Three rounds of hand-enumeration have each missed sites, so the rule needs a mechanism: a comment asserting what a path classifies names the test that pins it, and a comment enumerating callers derives that list — both idiomatic here.'
          family: stale-wording-after-referent-change
          round: 4
        - id: BR-25
          severity: Important
          title: The plan's task bodies still direct three things the code deliberately did not do
          detail: 3rd in family — fix the RULE, not these sites. Round 2's stated rule covered the Core-concepts tables only, and those are clean (I grepped every row). The class is every normative statement in the plan. plan:358-359 prescribes SessionUnresolved above VerifiedPark, which is the bug BR-2 fixed by inverting them. plan:424-426 says "Keep ReasonUnrecordedChild" where the code deleted it, with no Revision recording the reversal. plan:605-612's Task 8a describes work pulled forward into M1 and names the deleted ResumeCreating as its red state. Widen the re-derivation rule to task bodies and sweep them this round, so the next milestone does not open against instructions that undo boundary fixes.
          family: plan-code-divergence
          round: 4
        - id: BR-26
          severity: Minor
          title: The ARCH-CONSTRAINTS budget figures the plan requires were never recorded in the Log
          detail: plan:294-300 requires both figures in the issue Log — the startup evidence round and one steady-state refresh, on the 7-record 4-scope store. Neither is there. The envelope itself is enforced well and structurally (SessionPresenceQueries()==1, DetachedQueries()==0 at three sites, TestSessionPresenceCountsNoClients), which is stronger evidence than a timing number, so this is bookkeeping rather than risk.
          family: declared-measurement-not-recorded
          round: 4
      boundary: M1
      blocked: true
    - "n": 5
      timestamp: "2026-09-17T11:15:39-07:00"
      agent: claude
      dispose:
        - id: BR-19
          disposition: not-addressed
          note: 'Round 4 prepended a new Log section mentioning a fourth site but left all three named homes unchanged: issue lines 123, 176 and 184 still say "three sites" with an enumeration of 1-3. The issue file now contradicts itself.'
          round: 5
        - id: BR-20
          disposition: not-addressed
          note: classify_test.go was untouched in 4b23a713 (last change b5fce898); name, doc and failure message still disagree with the body. The round's own new test resume_test.go:333 adds a fourth instance — its failure message prints ResumeNotRunning as the value it must not be.
          round: 5
        - id: BR-21
          disposition: not-addressed
          note: resume.go:630 still calls c.Threads.AbandonPark directly; no change in park.go or couchcmd/run.go, and no comment saying why the per-thread worker is not needed here. Minor, CAS-protected.
          round: 5
        - id: BR-23
          disposition: addressed
          note: 'Mutation-verified: deleting the owner probe at resume.go:625-629 reddens TestForeignOwnedParkIsRepresentableAndRefused, which builds the fixture through the production ThreadStore and the store accepts it. Checked the guard''s edge: validateParkIdentity requires a non-empty ProcessIdentity with no exception, so observeExactProcess''s empty-identity path is unreachable here. The residual stale premise in two other homes is raised separately.'
          round: 5
        - id: BR-24
          disposition: addressed
          note: All six re-derived. Behavioural member mutation-verified (ResumeUnknown -> ResumeNotRunning reddens resume_test.go:319). ResumeNotRunning keeps its producer at relaunch.go:103. Two sites gained the requested mechanism by naming the test that pins the claim.
          round: 5
        - id: BR-25
          disposition: addressed
          note: plan:355-364 branch order inverted, :423-427 records the ReasonUnrecordedChild reversal with its reason, :594-598 marks Task 8a LANDED IN M1 and flags its deleted red state; the widened rule is in the round-4 Revisions entry. Siblings it did not reach are raised as a Minor rather than re-raised here.
          round: 5
        - id: BR-26
          disposition: addressed
          note: The issue Log now carries the ARCH-CONSTRAINTS accounting — 0 client queries where the old path made up to 6, Physical on 4 records rather than 3, four sites asserting SessionPresenceQueries()==1 — and names M2's operator verification as the owner of the wall-clock figure.
          round: 5
      findings:
        - id: BR-27
          severity: Critical
          title: An open park with zero incarnations makes couch refuse to start in the whole tree
          detail: '3rd in family — fix the RULE, not this site. retireDeadIncarnationBeforeStart returns (nil, nil) whenever len(Incarnations) != 1 (resume.go:555), so an open-park record with ZERO incarnations is never cleared. Confirmed by execution against the production store — the same replacementUnknown exception round 4 found (threadrecord/lifecycle.go:91), read at a different incarnation count: CreateThread ACCEPTS the record, ClassifyThread returns detached, the re-adoption declines silently, and CommitStartClaim refuses with an uncoded "has an open park transaction". Through StartInteractive on a startupFixture, head refuses couch in the tree while the identical fixture at base b6a0766a returns err = nil and spawns normally. The record is also unarchivable (ArchiveThread — "a park transaction is still open"), and RecoverActiveParks runs after the dispatch returns (couchcmd/run.go:342) so a failed start exits before it. I did not trace a single reproducible production sequence that writes the shape; the store accepting it is the standard round 4 installed, and ARCH-SECURE treats a record from another version as untrusted input. The rule round 2 stated — "every guard refusing on record.Incarnations or record.Park" — was written into resume.go:604-608 as four SITES rather than as that predicate, which is why its other half went unchecked. Make the re-adoption total over the shapes validateLifecycle accepts — hoist the park abandon above the count gate, since its own owner probe authorizes it independently, and refuse with a code rather than declining silently for counts it cannot clear. Mechanism: add an incarnation-count dimension to TestReAdoptionExitsAreTotalAndCoded.'
          family: classification-not-authority
          round: 5
        - id: BR-28
          severity: Important
          title: The premise round 4 disproved is still current in the atlas and in the fixed function's own comment
          detail: '3rd in family — fix the RULE, not these sites. atlas/couch.md:1343-1345 still reads "one probe answers both, since the park identity is copied from the incarnation", the exact claim TestForeignOwnedParkIsRepresentableAndRefused now falsifies, and :1349-1354''s "two rules fell out of the sweep" predates round 4''s two new rules. resume.go:597-600 states the same retracted reasoning ten lines above its own correction at :606-624, so a reader who stops at the outer comment gets the disproved version. This is one rule with BR-19, still open on the site-count homes: when a boundary round changes a claim, every home of it is re-derived in the same round — code comment, plan Revisions, atlas, issue Log. Three rounds of hand-sweeping have each left homes behind, so apply BR-24''s mechanism here: a claim that justifies a guard''s shape cites the test that pins it, so the claim and its evidence move together. Measured prevalence in this window: 2 stale homes for the park-identity claim, 3 for the site count, 6 for BR-24''s referent change, 5 for BR-7''s.'
          family: atlas-contradicts-code
          round: 5
        - id: BR-29
          severity: Minor
          title: SessionPresence's error is discarded, so a host-wide failure renders every row checking… with no cause
          detail: 'actionableinventory.go:567 does `if presence, presenceErr := presenceResolver.SessionPresence(ctx, addresses); presenceErr == nil` and drops the error entirely — no trace, no carried field. A zellij failure or an unreadable scope path turns every row in every tree into unusable/unknown with the reason recorded nowhere, while PathError on the same ThreadEvidence struct is carried per record. Fail-closed is correct and tested; anonymous is the shape #181 removed. Carry it the way PathError is carried, or surface it once in the banner.'
          family: degradation-without-diagnostic
          round: 5
        - id: BR-30
          severity: Minor
          title: Round 4's own table edit left the sentence below it false, and two Core-concepts statements still direct the reversed design
          detail: '4th in family — the rule has been stated twice and hand-applied twice, so state the mechanism instead. plan:363-364 says "Rows 7-9 read only resume authority, which is genuinely durable" while row 7 is now SessionUnresolved after round 4''s swap. Row 8''s binding-lost is reachable only INSIDE the VerifiedPark branch in the code, above row 7, and the VerifiedPark + ProofUnresolved -> unknown sub-case is missing from the table entirely. plan:355 still says "(in-memory observation)" and plan:213-215 still says "Ephemeral state stays ephemeral", both describing the design the Revisions entry adopted the opposite of. The mechanism: stop restating the branch table in the plan and point at ClassifyThread plus classify_test.go''s everyThreadShape, which is derived and tested — a hand-maintained restatement of the model is a deferred consumer (ARCH-PURPOSE).'
          family: plan-code-divergence
          round: 5
      boundary: M1
      blocked: true
    - "n": 6
      timestamp: "2026-09-17T11:37:33-07:00"
      agent: claude
      dispose:
        - id: BR-19
          disposition: not-addressed
          note: Round 5 did not touch the issue file. Lines 123, 176 and 184 still say "three sites" with an enumeration of 1-3, while atlas:1328 says four and plan:935 says FOUR — and the issue's own round-table Log section says a fourth site was found, so the file contradicts itself.
          round: 6
        - id: BR-20
          disposition: not-addressed
          note: 'classify_test.go untouched since b5fce898. :270 name says "exactly what the old projector accepted", :268-269 doc cites only the #248 exception, :275 failure message prints tc.wasActionableBefore alone, while :274 asserts (wasActionableBefore || newlyActionable) over three #256 shapes.'
          round: 6
        - id: BR-21
          disposition: not-addressed
          note: 'resume.go:624 still calls c.Threads.AbandonPark directly; no change in park.go, no comment saying why the per-thread worker is unnecessary. Correcting the finding''s premise: RecoverActiveParks is launched after the dispatch returns (run.go:342), so it does not run concurrently with startup resume. CAS-protected, Minor.'
          round: 6
        - id: BR-27
          disposition: addressed
          note: 'Mutation-verified — restoring `if len(thread.Incarnations) != 1 { return nil, nil }` reddens TestReAdoptionExitsAreTotalAndCoded/matching-park/0-incarnation/dead/live with "no refusal and no result ... park=true". The fixture builds through the production store and is not skipped. Residual: the table''s dimensions are hand-written, raised separately.'
          round: 6
        - id: BR-28
          disposition: addressed
          note: atlas:1343-1345's "one probe answers both" is gone; :1349-1360 now carries four rules, each citing the test that pins it. resume.go:450-458 no longer restates the park rules and points at retireDeadIncarnationBeforeStart instead. The surviving copy at plan:943 is inside an append-only round-1 Revisions entry, which is a dated record rather than current guidance.
          round: 6
        - id: BR-29
          disposition: not-addressed
          note: actionableinventory.go:567 unchanged — `if presence, presenceErr := ...; presenceErr == nil` still drops the error with no trace and no carried field.
          round: 6
        - id: BR-30
          disposition: not-addressed
          note: 'Two of three fixed — plan:213-215 now adopts the durable start claim, and the "(in-memory observation)" table is gone. The third survives verbatim: plan:367-368 still reads "Rows 7-9 read only resume authority, which is genuinely durable", now a dangling reference to a deleted table and still the false claim the finding named.'
          round: 6
      findings:
        - id: BR-31
          severity: Important
          title: The totality table's dimensions are hand-written, so the totality claim is unproven for shapes the store accepts
          detail: 'This is the 3rd finding in family `fail-closed-guard-untested`, so the rule, not the site. resume.go:545 and atlas/couch.md:1354-1356 claim retireDeadIncarnationBeforeStart is total over what validateLifecycle accepts and that TestReAdoptionExitsAreTotalAndCoded enumerates it, but sessionevidence_test.go:376-377 loops park in {none,matching} and count in {0,1} — constants, not the domain. Verified against the production store: CreateThread ACCEPTS a two-incarnation record, ClassifyThread returns detached so startup ranks it highest, and the function refuses resume-unknown "thread carries more than one recorded incarnation" with no cell covering it. Measured: 4 of ~10 exits uncovered — Start != nil (silent decline), default count >= 2, PID <= 0 or empty Identity, and a foreign park (only a separately-named test). The rule is round 5''s own, applied one layer down: a table asserting totality takes its dimensions from the domain''s oracle, not from the author''s enumeration. The oracle is already there — t.Skipf on CreateThread refusal. Widen count to {0,1,2}, add park "foreign" and a startClaimed variant, and let the skip exclude what is out of domain, so a new exit without a cell fails instead of passing.'
          family: fail-closed-guard-untested
          round: 6
        - id: BR-32
          severity: Minor
          title: SessionObservation.Name is written in four places and read nowhere
          detail: 2nd in family — the rule is BR-22's dual. sessionevidence.go:48-52 documents Name as carried "so a consumer that acts on the observation does not re-derive the name from a second index read", but no production path and no test reads it; only evidence.Session.State is consumed (actionableinventory.go:323,351). BR-22's guard proves every ResumeDiagnosticCode has a producer; the same surface needs the other end — a declared element of a closed surface is guarded at BOTH ends, or the unguarded end rots. Either give it the consumer its doc describes or drop the field.
          family: vocabulary-entry-without-producer
          round: 6
      boundary: M1
      blocked: false
    - "n": 7
      timestamp: "2026-09-17T13:57:04-07:00"
      agent: claude
      boundary: M2
      blocked: false
      protocol_error: no valid findings block
    - "n": 8
      timestamp: "2026-09-17T14:33:47-07:00"
      agent: claude
      findings:
        - id: BR-33
          severity: Critical
          title: 'switch-agent re-derives "nothing runs" instead of consuming the classification: 2 of 4 parked producers refuse, and a hosted `live` row now fails OPEN'
          detail: |-
            This is the 5th finding in family classification-not-authority; fix the rule, not
            these sites. Measured through the production gather path in a git archive scratch
            tree: a driverless start claim with a resolvable ledger classifies `parked` and
            PrepareAgentSwitch refuses permanently ("occupied thread: park requires exactly one
            identified live or unknown incarnation"); the receipt+SessionUnresolved producer the
            classifier keeps deliberately is refused where the pre-M2 guard admitted it; and a
            record couch HOSTS with no incarnation and no session binding classifies `live` and is
            now ACCEPTED, while SwitchAgent parks the source only if hasOccupiedIncarnation
            (switchagent.go:267) -- two agents on one tree. Reverting the else-if to
            record.VerifiedPark == nil refuses that row, so it is a regression in this window.
            The rule: an action guard consumes the classification (state, reason, live evidence)
            as M3 Task 8 plans for archive; where a strict re-observation is needed the policy is
            a pure predicate beside RecoverySessionRefusal and must read the same world fact the
            same way (switchagent.go:71 treats ErrPairSessionBindingAbsent as proof of absence
            against its own doc at artifactcollision.go:15). And the offered-implies-permitted
            enumeration must be DERIVED from ClassifyThread/everyThreadShape, not hand-listed:
            parkedproducers_test.go:111 compares a map filled from its own two-cell literal, so
            its totality assertion cannot fail, and everyThreadShape is itself missing the
            driverless-claim-with-ledger row.
          family: classification-not-authority
          round: 8
        - id: BR-34
          severity: Important
          title: I2's rule was written down but the git grep it prescribes was never run -- four sites still state the retired `parked` referent
          detail: |-
            This is the 6th finding in family stale-wording-after-referent-change; the rule is the
            deliverable, not the sites. The Revisions entry names the homes (atlas + plan +
            function comment + every exported doc comment + README.md) and "git grep of the old
            referent string as the mechanical check"; the check was not executed. Remaining:
            cmd/internal/couchcore/ops.go:381, the operator/advisor-facing `resume` summary
            ("resume a verified-parked one"), the same class of surface as README.md:470 which was
            swept; atlas/couch.md:35, the atlas's own statement of the classification rule
            ("parked when verified park exists ... or incarnation"); atlas/couch.md:498; and
            atlas/couch.md:863-871 ("Resume accepts verified park or proved detachment ... The
            occupied-incarnation refusal is unchanged"), which DecideResume contradicts at
            resume.go:118. Make the grep a checked step of the boundary close, the way C2 just did
            for the Core-concepts tables.
          family: stale-wording-after-referent-change
          round: 8
        - id: BR-35
          severity: Important
          title: Two fixtures were retuned to keep their old verdict and the new startup behaviour they used to cover has no test
          detail: |-
            startup_test.go:328 and couchcmd/run_test.go:406 both changed BindingEstablished ->
            BindingUnbound so "startup creates a NEW thread" still passes. The behaviour the issue
            Log advertises as what the operator will notice -- startup adopts a session-gone row
            whose ledger resolves rather than starting a second thread in the same tree -- has no
            test at the startup level in either direction. Restoring the established binding in a
            scratch tree shows the adoption path IS taken: startup selected the cold row and
            reported "couch could not resume the thread in this tree ... and will not start a
            second one" after a 15 s registration wait, extending
            TestStartInteractiveResumeRefusalDoesNotCreateFallbackRoot's deliberate no-fallback
            policy to a new class of rows. The rule: when a fixture is retuned so an existing test
            keeps its old verdict under new behaviour, the new behaviour gets its own test in the
            same commit -- the retune is the signal that a branch changed owner.
          family: fixture-retuned-to-preserve-old-verdict
          round: 8
        - id: BR-36
          severity: Minor
          title: ClassifyThread's late unresolved-session guard is unreachable after the M2 reordering
          detail: |-
            actionableinventory.go:412 is subsumed: :395 returns for Unresolved with no receipt and
            :405 returns for every remaining receipt-holder, so the branch is dead by construction.
            Replacing its body with a panic and running the whole couchcore suite produced zero
            hits. 4th in this family, as the dual of "a guard nothing pins is not a guard": a
            branch subsumed by an earlier predicate is a guard no test can reach, and the comment
            defending its load-bearing distinction is a claim nothing checks. Delete it, or move
            the receipt exception so the distinction is decided there.
          family: fail-closed-guard-untested
          round: 8
        - id: BR-37
          severity: Minor
          title: TestEveryResumeDiagnosticCodeIsReachableFromProduction asserts exactly one code
          detail: |-
            resume_test.go:373 pins ResumeTombstoned only; its own comment admits it. Its sibling
            TestEveryResumeDiagnosticCodeIsProducedBySomeSite:295 derives the identifier set from
            the declaration so it cannot be satisfied by forgetting a row -- do the same here, or
            name the test for the one code. It also t.Skipf's if the store rejects its fixture
            (:396); it passes today, but a silent skip is how a reachability guard stops guarding.
          family: test-name-contradicts-assertion
          round: 8
      boundary: M2
      blocked: true
    - "n": 9
      timestamp: "2026-09-17T15:05:41-07:00"
      agent: claude
      boundary: M2
      blocked: true
      protocol_error: no valid findings block
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

## Round 4 — 2026-09-17T10:47:04-07:00 (claude) — BLOCKED

### Disposed

- BR-17 — addressed — Blanket deferred coder gone from resume.go (only the pre-existing retention join at :359 remains, present at base); menu_reattach.go:244-252 reads an empty code correctly again; TestResumeCodeStillMeansAStructuredRefusal and TestEveryStartupResumeFailureIsActionable pin both halves including errors.Is through the decoration. The second instance (actionableinventory.go "must not start to") is rewritten.
- BR-18 — addressed — artifactcollision.go:320-331 binds SessionPresenceResolver, DetachedSessionResolver, NativeBindingResolver and PairSessionIO on the production type plus two on the fake, with the silent-failure rationale in the comment. Residual not re-raised: contextPairSessionObserver (recovery_execute.go:16) and couchcmd/run.go:125's anonymous interface stay unpinned, but both have an explicit non-context fallback rather than degrading to universal unknown.
- BR-19 — not-addressed — atlas/couch.md:1330 now says "One class, four sites" — correct. The issue Log, named in the finding as the fourth home and again in round 3's own notes (m1-review.md:476), still says three at issue lines 123, 133 and 141, with an enumeration that omits the site carrying the irreversible-ordering rule. See I1.
- BR-20 — not-addressed — classify_test.go untouched in 54d36c37; name, doc (still citing a #248 case that no longer exists) and failure message all still disagree with a body admitting three newlyActionable #256 shapes. The guard added this round, TestEveryResumeDiagnosticCodeIsProducedBySomeSite, is a fresh instance of the same rule — it counts substring mentions, so a code with comment mentions and no producer passes.
- BR-21 — not-addressed — resume.go:608 still calls c.Threads.AbandonPark directly, bypassing PairLifecycleController.Abandon's per-thread worker; no change in park.go or run.go. Minor, CAS-protected.
- BR-22 — addressed — ResumeCreating deleted and TestEveryResumeDiagnosticCodeIsProducedBySomeSite derives its identifiers from the declaration, so re-adding an unproduced code reddens it. Soundness gap in the guard itself recorded under BR-20 rather than re-raised here.

### Raised

- **BR-23** [Critical] `irreversible-step-before-precondition` The orphaned-park abandon probes one process and writes a permanent tombstone about another
  2nd in family — fix the RULE, not this site. resume.go:600-612 abandons thread.Park after probing thread.Incarnations[0], omitting the identity check on the claim that a park owned by another process is unrepresentable. threadrecord/lifecycle.go:91 permits matches==0 when Phase is "unknown" and the transaction carries a replacement_incarnation failure; park.go:653-662 produces exactly that, and threadrecord/record_test.go:303 pins it as valid. Confirmed by execution in a scratch copy: a record created through the production ThreadStore with a live incarnation {99,"replacement"} and a park owned by {42,"original-owner"}, with pid 42 ALIVE, retires cleanly and abandons pid 42's park — one permanent tombstone, retireErr nil. The live owner's later FinalizePark then fails with "park abandon identity does not match active transaction", so the park silently never completes and #275's audit trail for it is gone. The rule: an irreversible step's precondition must be proved about the exact entity the step acts on, and a guard omitted as "unrepresentable" must cite the validator clause that makes it so, read including its exceptions, and be pinned by a test that tries to build the fixture through the real store. Enumerable sibling: lifecycle.go:97-105's resumeOccupied escape. Also correct sessionevidence_test.go:372's exclusion note and plan:842.
- **BR-24** [Important] `stale-wording-after-referent-change` A changed referent left six unre-derived sites, one of which renders a false diagnostic to the operator
  4th in family — fix the RULE, not these sites. Behavioural member first: ResumeNotRunning is declared (resume.go:31-34) as "not running at all … the OPPOSITE of ResumeLive" and is emitted at :569 for "could not be proved dead" and at :624 for a store error; menu_reattach.go:239 skips only ResumeNotDetached/ResumeSessionGone, so a background reattach of a #272 row with an unobservable launcher renders "resume-not-running" on a row whose agent is running, and no test pins which code that exit emits. Doc-only members: startup.go:137-139 ("a candidate outside both sets keeps ProofUnresolved and classifies unknown" — presence is now gathered after the ask gate, so it classifies detached); actionableinventory.go:419-421 restating it despite startup.go:124-126 declaring itself the one home; actionableinventory.go:376 ("shared by inventory" — the inventory no longer calls it); artifactcollision_fake.go:59-60 (says unset reads unresolved, code returns SessionAbsent since b5fce898); startup.go:69 (occupiedIncarnation "shared by archive and resume" — resume no longer reads it). Three rounds of hand-enumeration have each missed sites, so the rule needs a mechanism: a comment asserting what a path classifies names the test that pins it, and a comment enumerating callers derives that list — both idiomatic here.
- **BR-25** [Important] `plan-code-divergence` The plan's task bodies still direct three things the code deliberately did not do
  3rd in family — fix the RULE, not these sites. Round 2's stated rule covered the Core-concepts tables only, and those are clean (I grepped every row). The class is every normative statement in the plan. plan:358-359 prescribes SessionUnresolved above VerifiedPark, which is the bug BR-2 fixed by inverting them. plan:424-426 says "Keep ReasonUnrecordedChild" where the code deleted it, with no Revision recording the reversal. plan:605-612's Task 8a describes work pulled forward into M1 and names the deleted ResumeCreating as its red state. Widen the re-derivation rule to task bodies and sweep them this round, so the next milestone does not open against instructions that undo boundary fixes.
- **BR-26** [Minor] `declared-measurement-not-recorded` The ARCH-CONSTRAINTS budget figures the plan requires were never recorded in the Log
  plan:294-300 requires both figures in the issue Log — the startup evidence round and one steady-state refresh, on the 7-record 4-scope store. Neither is there. The envelope itself is enforced well and structurally (SessionPresenceQueries()==1, DetachedQueries()==0 at three sites, TestSessionPresenceCountsNoClients), which is stronger evidence than a timing number, so this is bookkeeping rather than risk.

## Round 5 — 2026-09-17T11:15:39-07:00 (claude) — BLOCKED

### Disposed

- BR-19 — not-addressed — Round 4 prepended a new Log section mentioning a fourth site but left all three named homes unchanged: issue lines 123, 176 and 184 still say "three sites" with an enumeration of 1-3. The issue file now contradicts itself.
- BR-20 — not-addressed — classify_test.go was untouched in 4b23a713 (last change b5fce898); name, doc and failure message still disagree with the body. The round's own new test resume_test.go:333 adds a fourth instance — its failure message prints ResumeNotRunning as the value it must not be.
- BR-21 — not-addressed — resume.go:630 still calls c.Threads.AbandonPark directly; no change in park.go or couchcmd/run.go, and no comment saying why the per-thread worker is not needed here. Minor, CAS-protected.
- BR-23 — addressed — Mutation-verified: deleting the owner probe at resume.go:625-629 reddens TestForeignOwnedParkIsRepresentableAndRefused, which builds the fixture through the production ThreadStore and the store accepts it. Checked the guard's edge: validateParkIdentity requires a non-empty ProcessIdentity with no exception, so observeExactProcess's empty-identity path is unreachable here. The residual stale premise in two other homes is raised separately.
- BR-24 — addressed — All six re-derived. Behavioural member mutation-verified (ResumeUnknown -> ResumeNotRunning reddens resume_test.go:319). ResumeNotRunning keeps its producer at relaunch.go:103. Two sites gained the requested mechanism by naming the test that pins the claim.
- BR-25 — addressed — plan:355-364 branch order inverted, :423-427 records the ReasonUnrecordedChild reversal with its reason, :594-598 marks Task 8a LANDED IN M1 and flags its deleted red state; the widened rule is in the round-4 Revisions entry. Siblings it did not reach are raised as a Minor rather than re-raised here.
- BR-26 — addressed — The issue Log now carries the ARCH-CONSTRAINTS accounting — 0 client queries where the old path made up to 6, Physical on 4 records rather than 3, four sites asserting SessionPresenceQueries()==1 — and names M2's operator verification as the owner of the wall-clock figure.

### Raised

- **BR-27** [Critical] `classification-not-authority` An open park with zero incarnations makes couch refuse to start in the whole tree
  3rd in family — fix the RULE, not this site. retireDeadIncarnationBeforeStart returns (nil, nil) whenever len(Incarnations) != 1 (resume.go:555), so an open-park record with ZERO incarnations is never cleared. Confirmed by execution against the production store — the same replacementUnknown exception round 4 found (threadrecord/lifecycle.go:91), read at a different incarnation count: CreateThread ACCEPTS the record, ClassifyThread returns detached, the re-adoption declines silently, and CommitStartClaim refuses with an uncoded "has an open park transaction". Through StartInteractive on a startupFixture, head refuses couch in the tree while the identical fixture at base b6a0766a returns err = nil and spawns normally. The record is also unarchivable (ArchiveThread — "a park transaction is still open"), and RecoverActiveParks runs after the dispatch returns (couchcmd/run.go:342) so a failed start exits before it. I did not trace a single reproducible production sequence that writes the shape; the store accepting it is the standard round 4 installed, and ARCH-SECURE treats a record from another version as untrusted input. The rule round 2 stated — "every guard refusing on record.Incarnations or record.Park" — was written into resume.go:604-608 as four SITES rather than as that predicate, which is why its other half went unchecked. Make the re-adoption total over the shapes validateLifecycle accepts — hoist the park abandon above the count gate, since its own owner probe authorizes it independently, and refuse with a code rather than declining silently for counts it cannot clear. Mechanism: add an incarnation-count dimension to TestReAdoptionExitsAreTotalAndCoded.
- **BR-28** [Important] `atlas-contradicts-code` The premise round 4 disproved is still current in the atlas and in the fixed function's own comment
  3rd in family — fix the RULE, not these sites. atlas/couch.md:1343-1345 still reads "one probe answers both, since the park identity is copied from the incarnation", the exact claim TestForeignOwnedParkIsRepresentableAndRefused now falsifies, and :1349-1354's "two rules fell out of the sweep" predates round 4's two new rules. resume.go:597-600 states the same retracted reasoning ten lines above its own correction at :606-624, so a reader who stops at the outer comment gets the disproved version. This is one rule with BR-19, still open on the site-count homes: when a boundary round changes a claim, every home of it is re-derived in the same round — code comment, plan Revisions, atlas, issue Log. Three rounds of hand-sweeping have each left homes behind, so apply BR-24's mechanism here: a claim that justifies a guard's shape cites the test that pins it, so the claim and its evidence move together. Measured prevalence in this window: 2 stale homes for the park-identity claim, 3 for the site count, 6 for BR-24's referent change, 5 for BR-7's.
- **BR-29** [Minor] `degradation-without-diagnostic` SessionPresence's error is discarded, so a host-wide failure renders every row checking… with no cause
  actionableinventory.go:567 does `if presence, presenceErr := presenceResolver.SessionPresence(ctx, addresses); presenceErr == nil` and drops the error entirely — no trace, no carried field. A zellij failure or an unreadable scope path turns every row in every tree into unusable/unknown with the reason recorded nowhere, while PathError on the same ThreadEvidence struct is carried per record. Fail-closed is correct and tested; anonymous is the shape #181 removed. Carry it the way PathError is carried, or surface it once in the banner.
- **BR-30** [Minor] `plan-code-divergence` Round 4's own table edit left the sentence below it false, and two Core-concepts statements still direct the reversed design
  4th in family — the rule has been stated twice and hand-applied twice, so state the mechanism instead. plan:363-364 says "Rows 7-9 read only resume authority, which is genuinely durable" while row 7 is now SessionUnresolved after round 4's swap. Row 8's binding-lost is reachable only INSIDE the VerifiedPark branch in the code, above row 7, and the VerifiedPark + ProofUnresolved -> unknown sub-case is missing from the table entirely. plan:355 still says "(in-memory observation)" and plan:213-215 still says "Ephemeral state stays ephemeral", both describing the design the Revisions entry adopted the opposite of. The mechanism: stop restating the branch table in the plan and point at ClassifyThread plus classify_test.go's everyThreadShape, which is derived and tested — a hand-maintained restatement of the model is a deferred consumer (ARCH-PURPOSE).

## Round 6 — 2026-09-17T11:37:33-07:00 (claude) — passed

### Disposed

- BR-19 — not-addressed — Round 5 did not touch the issue file. Lines 123, 176 and 184 still say "three sites" with an enumeration of 1-3, while atlas:1328 says four and plan:935 says FOUR — and the issue's own round-table Log section says a fourth site was found, so the file contradicts itself.
- BR-20 — not-addressed — classify_test.go untouched since b5fce898. :270 name says "exactly what the old projector accepted", :268-269 doc cites only the #248 exception, :275 failure message prints tc.wasActionableBefore alone, while :274 asserts (wasActionableBefore || newlyActionable) over three #256 shapes.
- BR-21 — not-addressed — resume.go:624 still calls c.Threads.AbandonPark directly; no change in park.go, no comment saying why the per-thread worker is unnecessary. Correcting the finding's premise: RecoverActiveParks is launched after the dispatch returns (run.go:342), so it does not run concurrently with startup resume. CAS-protected, Minor.
- BR-27 — addressed — Mutation-verified — restoring `if len(thread.Incarnations) != 1 { return nil, nil }` reddens TestReAdoptionExitsAreTotalAndCoded/matching-park/0-incarnation/dead/live with "no refusal and no result ... park=true". The fixture builds through the production store and is not skipped. Residual: the table's dimensions are hand-written, raised separately.
- BR-28 — addressed — atlas:1343-1345's "one probe answers both" is gone; :1349-1360 now carries four rules, each citing the test that pins it. resume.go:450-458 no longer restates the park rules and points at retireDeadIncarnationBeforeStart instead. The surviving copy at plan:943 is inside an append-only round-1 Revisions entry, which is a dated record rather than current guidance.
- BR-29 — not-addressed — actionableinventory.go:567 unchanged — `if presence, presenceErr := ...; presenceErr == nil` still drops the error with no trace and no carried field.
- BR-30 — not-addressed — Two of three fixed — plan:213-215 now adopts the durable start claim, and the "(in-memory observation)" table is gone. The third survives verbatim: plan:367-368 still reads "Rows 7-9 read only resume authority, which is genuinely durable", now a dangling reference to a deleted table and still the false claim the finding named.

### Raised

- **BR-31** [Important] `fail-closed-guard-untested` The totality table's dimensions are hand-written, so the totality claim is unproven for shapes the store accepts
  This is the 3rd finding in family `fail-closed-guard-untested`, so the rule, not the site. resume.go:545 and atlas/couch.md:1354-1356 claim retireDeadIncarnationBeforeStart is total over what validateLifecycle accepts and that TestReAdoptionExitsAreTotalAndCoded enumerates it, but sessionevidence_test.go:376-377 loops park in {none,matching} and count in {0,1} — constants, not the domain. Verified against the production store: CreateThread ACCEPTS a two-incarnation record, ClassifyThread returns detached so startup ranks it highest, and the function refuses resume-unknown "thread carries more than one recorded incarnation" with no cell covering it. Measured: 4 of ~10 exits uncovered — Start != nil (silent decline), default count >= 2, PID <= 0 or empty Identity, and a foreign park (only a separately-named test). The rule is round 5's own, applied one layer down: a table asserting totality takes its dimensions from the domain's oracle, not from the author's enumeration. The oracle is already there — t.Skipf on CreateThread refusal. Widen count to {0,1,2}, add park "foreign" and a startClaimed variant, and let the skip exclude what is out of domain, so a new exit without a cell fails instead of passing.
- **BR-32** [Minor] `vocabulary-entry-without-producer` SessionObservation.Name is written in four places and read nowhere
  2nd in family — the rule is BR-22's dual. sessionevidence.go:48-52 documents Name as carried "so a consumer that acts on the observation does not re-derive the name from a second index read", but no production path and no test reads it; only evidence.Session.State is consumed (actionableinventory.go:323,351). BR-22's guard proves every ResumeDiagnosticCode has a producer; the same surface needs the other end — a declared element of a closed surface is guarded at BOTH ends, or the unguarded end rots. Either give it the consumer its doc describes or drop the field.

## Round 7 — 2026-09-17T13:57:04-07:00 (claude) — passed

**Protocol error:** no valid findings block — this round contributed no findings.

## Round 8 — 2026-09-17T14:33:47-07:00 (claude) — BLOCKED

### Raised

- **BR-33** [Critical] `classification-not-authority` switch-agent re-derives "nothing runs" instead of consuming the classification: 2 of 4 parked producers refuse, and a hosted `live` row now fails OPEN
  This is the 5th finding in family classification-not-authority; fix the rule, not
  these sites. Measured through the production gather path in a git archive scratch
  tree: a driverless start claim with a resolvable ledger classifies `parked` and
  PrepareAgentSwitch refuses permanently ("occupied thread: park requires exactly one
  identified live or unknown incarnation"); the receipt+SessionUnresolved producer the
  classifier keeps deliberately is refused where the pre-M2 guard admitted it; and a
  record couch HOSTS with no incarnation and no session binding classifies `live` and is
  now ACCEPTED, while SwitchAgent parks the source only if hasOccupiedIncarnation
  (switchagent.go:267) -- two agents on one tree. Reverting the else-if to
  record.VerifiedPark == nil refuses that row, so it is a regression in this window.
  The rule: an action guard consumes the classification (state, reason, live evidence)
  as M3 Task 8 plans for archive; where a strict re-observation is needed the policy is
  a pure predicate beside RecoverySessionRefusal and must read the same world fact the
  same way (switchagent.go:71 treats ErrPairSessionBindingAbsent as proof of absence
  against its own doc at artifactcollision.go:15). And the offered-implies-permitted
  enumeration must be DERIVED from ClassifyThread/everyThreadShape, not hand-listed:
  parkedproducers_test.go:111 compares a map filled from its own two-cell literal, so
  its totality assertion cannot fail, and everyThreadShape is itself missing the
  driverless-claim-with-ledger row.
- **BR-34** [Important] `stale-wording-after-referent-change` I2's rule was written down but the git grep it prescribes was never run -- four sites still state the retired `parked` referent
  This is the 6th finding in family stale-wording-after-referent-change; the rule is the
  deliverable, not the sites. The Revisions entry names the homes (atlas + plan +
  function comment + every exported doc comment + README.md) and "git grep of the old
  referent string as the mechanical check"; the check was not executed. Remaining:
  cmd/internal/couchcore/ops.go:381, the operator/advisor-facing `resume` summary
  ("resume a verified-parked one"), the same class of surface as README.md:470 which was
  swept; atlas/couch.md:35, the atlas's own statement of the classification rule
  ("parked when verified park exists ... or incarnation"); atlas/couch.md:498; and
  atlas/couch.md:863-871 ("Resume accepts verified park or proved detachment ... The
  occupied-incarnation refusal is unchanged"), which DecideResume contradicts at
  resume.go:118. Make the grep a checked step of the boundary close, the way C2 just did
  for the Core-concepts tables.
- **BR-35** [Important] `fixture-retuned-to-preserve-old-verdict` Two fixtures were retuned to keep their old verdict and the new startup behaviour they used to cover has no test
  startup_test.go:328 and couchcmd/run_test.go:406 both changed BindingEstablished ->
  BindingUnbound so "startup creates a NEW thread" still passes. The behaviour the issue
  Log advertises as what the operator will notice -- startup adopts a session-gone row
  whose ledger resolves rather than starting a second thread in the same tree -- has no
  test at the startup level in either direction. Restoring the established binding in a
  scratch tree shows the adoption path IS taken: startup selected the cold row and
  reported "couch could not resume the thread in this tree ... and will not start a
  second one" after a 15 s registration wait, extending
  TestStartInteractiveResumeRefusalDoesNotCreateFallbackRoot's deliberate no-fallback
  policy to a new class of rows. The rule: when a fixture is retuned so an existing test
  keeps its old verdict under new behaviour, the new behaviour gets its own test in the
  same commit -- the retune is the signal that a branch changed owner.
- **BR-36** [Minor] `fail-closed-guard-untested` ClassifyThread's late unresolved-session guard is unreachable after the M2 reordering
  actionableinventory.go:412 is subsumed: :395 returns for Unresolved with no receipt and
  :405 returns for every remaining receipt-holder, so the branch is dead by construction.
  Replacing its body with a panic and running the whole couchcore suite produced zero
  hits. 4th in this family, as the dual of "a guard nothing pins is not a guard": a
  branch subsumed by an earlier predicate is a guard no test can reach, and the comment
  defending its load-bearing distinction is a claim nothing checks. Delete it, or move
  the receipt exception so the distinction is decided there.
- **BR-37** [Minor] `test-name-contradicts-assertion` TestEveryResumeDiagnosticCodeIsReachableFromProduction asserts exactly one code
  resume_test.go:373 pins ResumeTombstoned only; its own comment admits it. Its sibling
  TestEveryResumeDiagnosticCodeIsProducedBySomeSite:295 derives the identifier set from
  the declaration so it cannot be satisfied by forgetting a row -- do the same here, or
  name the test for the one code. It also t.Skipf's if the store rejects its fixture
  (:396); it passes today, but a silent skip is how a reachability guard stops guarding.

## Round 9 — 2026-09-17T15:05:41-07:00 (claude) — BLOCKED

**Protocol error:** no valid findings block — this round contributed no findings.

## Open findings

- **BR-19** [Important] `atlas-contradicts-code` The atlas records "One class, three sites" while the code and the plan record four
- **BR-20** [Minor] `test-name-contradicts-assertion` TestClassifyThreadAcceptsExactlyWhatTheOldProjectorAccepted now admits four deliberately new shapes
- **BR-21** [Minor] `transition-bypasses-its-owner` The re-adoption's AbandonPark bypasses the per-thread park worker every other abandon goes through
- **BR-29** [Minor] `degradation-without-diagnostic` SessionPresence's error is discarded, so a host-wide failure renders every row checking… with no cause
- **BR-30** [Minor] `plan-code-divergence` Round 4's own table edit left the sentence below it false, and two Core-concepts statements still direct the reversed design
- **BR-31** [Important] `fail-closed-guard-untested` The totality table's dimensions are hand-written, so the totality claim is unproven for shapes the store accepts
- **BR-32** [Minor] `vocabulary-entry-without-producer` SessionObservation.Name is written in four places and read nowhere
- **BR-33** [Critical] `classification-not-authority` switch-agent re-derives "nothing runs" instead of consuming the classification: 2 of 4 parked producers refuse, and a hosted `live` row now fails OPEN
- **BR-34** [Important] `stale-wording-after-referent-change` I2's rule was written down but the git grep it prescribes was never run -- four sites still state the retired `parked` referent
- **BR-35** [Important] `fixture-retuned-to-preserve-old-verdict` Two fixtures were retuned to keep their old verdict and the new startup behaviour they used to cover has no test
- **BR-36** [Minor] `fail-closed-guard-untested` ClassifyThread's late unresolved-session guard is unreachable after the M2 reordering
- **BR-37** [Minor] `test-name-contradicts-assertion` TestEveryResumeDiagnosticCodeIsReachableFromProduction asserts exactly one code
