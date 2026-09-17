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

## Open findings

- **BR-1** [Critical] `classification-not-authority` An open park makes the new re-adoption hard-error and wedges couch startup
- **BR-2** [Important] `unknown-must-not-mask-durable-authority` SessionUnresolved is checked above VerifiedPark, so one failed list-sessions hides every parked row
- **BR-3** [Important] `fail-closed-guard-untested` Nothing pins that an Unknown liveness probe must not retire an incarnation
- **BR-4** [Important] `production-seam-only-tested-through-fake` ScopedThreadArtifactCollisionChecker.SessionPresence has no test
- **BR-5** [Important] `atlas-contradicts-code` Three atlas passages still document the retired reason vocabulary as current
- **BR-6** [Important] `readme-documents-removed-surface` README still lists a deleted reason label and claims parks leave a diagnostic
- **BR-7** [Important] `stale-wording-after-referent-change` ThreadBusy changed meaning but five wording sites still describe a park
- **BR-8** [Important] `plan-code-divergence` startClaimed reads durable state where the plan specified an in-memory observation
- **BR-9** [Minor] `test-name-contradicts-assertion` Two tests assert something other than what their names say
- **BR-10** [Minor] `duplicated-fail-closed-rule` The session name-index and uniqueness predicate are copy-pasted across two projectors
