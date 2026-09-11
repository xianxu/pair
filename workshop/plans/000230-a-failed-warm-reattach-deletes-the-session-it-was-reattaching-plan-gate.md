---
gate: plan-quality
issue: 230
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-11T10:09:47-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Warm registration is satisfied by the pre-existing session, so rows 3-4's injections cannot reach routes 3-4
          detail: 'awaitResumeRegistration (launch_existing.go:238-256) returns once PairSession reports Present, and Present means listed-and-not-exited (artifactcollision.go:170-181), which a detached session already is. Row 3 ("Pair never registers") passes on the first poll. BeforeRegistration fires only from Artifacts.Registration (artifactcollision_fake.go:219-223), which the resume path never calls. Use BeforePairSession for both rows, and correct the Problem''s "slow pair resume times out" reachability claim: a warm route 3 needs PairSession to fail for the whole budget, or the session to vanish.'
          family: unverified-existing-behavior-claim
          round: 1
        - id: PQ-2
          severity: Important
          title: Spec and plan disagree on the routes 1-4 rollback rule, and no Revision records it
          detail: The Spec rolls back only once the session is proven present, and marks the start unknown when presence cannot be observed. The plan rolls back once the helper is dead, whatever the session's presence. The plan's rule is defensible. Adopt one of the two and append a Revision amending the other, so the close review has a single contract to check.
          family: spec-plan-divergence
          round: 1
        - id: PQ-3
          severity: Important
          title: Session ownership travels on a caller-relayed StartResult whose zero value is the destructive branch
          detail: AbortStarted already treats StartResult as untrusted and re-validates it against c.reg (couch.go:527-552). Warm reaches it through couchtty's StartedChild relay, and RelaunchResult.Started (relaunch.go:52-57) already rebuilds StartResult field by field. Any relay that drops Warm deletes a warm session, which breaks the Spec's rule that deletion is never the answer to not knowing. The ARCH-SECURE note covers a spoofed true but not a dropped false. Either key ownership in couch by the ActorID that AbortStarted already looks up, or make the zero value non-destructive.
          family: destructive-zero-value-default
          round: 1
        - id: PQ-4
          severity: Important
          title: The fake's Quiesce only logs the call, so the Detached-again assertion cannot fail under the named mutation
          detail: FakeThreadArtifactCollisionChecker.Quiesce (artifactcollision_fake.go:236-247) appends to a log and leaves pairSessions and detachedSessions untouched. "The non-owning branch also quiesces" therefore still classifies ThreadDetached, and only the Quiesces() call log catches it. Make Quiesce clear the address's session state, in the fake or at least via a per-test QuiesceHook, so the behavioural assertion is a real oracle (ARCH-MOCK).
          family: stateless-fake-for-stateful-effect
          round: 1
        - id: PQ-5
          severity: Minor
          title: '"quiescePostAckStart is the only caller of Artifacts.Quiesce" is false: ArchiveThread calls it too (detach.go:219)'
          detail: That call is the operator's deliberate delete, gated by archivableRecord, so the class is still six routes. Correct the sentence in both the plan and the issue Revision, and say why ArchiveThread is out of the class.
          family: unverified-existing-behavior-claim
          round: 1
        - id: PQ-6
          severity: Minor
          title: No non-goals section; three adjacent behaviours are left unstated
          detail: (1) If the detached session dies between confirmStillDetached and exec, `pair resume <tag>` falls through to ActionCreate (decision.go:71-76), so a "warm" start can own a session it made. Non-owning cleanup then leaks that session rather than deleting it, which fails safe but should be written down. (2) The warm registration oracle proves nothing about the child attaching. (3) Couch dying between the helper kill and the durable write is left to reconcileInterruptedStarts.
          family: missing-non-goals
          round: 1
        - id: PQ-7
          severity: Minor
          title: Routes 5-6 re-implement Detach's post-signal half; extract it
          detail: Detach's second presence proof plus the bounded RetireIncarnation retry on revision conflicts (detach.go:96-147) is what routes 5-6 need, with detachedAt passed in. Extract it as a shared helper. Its per-attempt ctx.Err() check must not make route 5 (which can run under a cancelled ctx) fail when it should fall back to MarkIncarnationUnknown.
          family: arch-dry-reuse
          round: 1
        - id: PQ-8
          severity: Minor
          title: The cleanup decision is buried in IO functions; a pure decider like ReconcileStart would test it exhaustively
          detail: '(owns, helper liveness, presence observed/absent/unobservable, creating vs live) -> {rollback, retire, mark-unknown, quiesce} is a small pure table (precedent: ReconcileStart, starttransaction.go:178). Unit-test it exhaustively; the fake-seam table then only proves each route''s wiring.'
          family: arch-pure-decision
          round: 1
        - id: PQ-9
          severity: Minor
          title: 'Moving operationdispatch to ResumeStart changes the seam #206 M2 designs against'
          detail: '#206''s plan adds ResumeOptions{WarmOnly} to ResumeContext and has operationdispatch pass it. After this change those options belong on ResumeStart. Note it here so #206''s plan gets a Revision before M2 starts.'
          family: cross-issue-seam-change
          round: 1
        - id: PQ-10
          severity: Minor
          title: Task 1 enumerates per-row injections and assertions in prose; compress to one strategy line per risky function
          detail: 'Rows 3-4 have already gone stale (see the Important above). Keep the six-route table, which is the class enumeration. Replace the per-row bullets with one line per function: "launchTrackedThread/AbortStarted post-ack exits -> table over six routes x {warm, owning} at the fake seam; guard: per-route mutation."'
          family: test-prose-enumeration
          round: 1
        - id: PQ-11
          severity: Minor
          title: The Spec cites TestSessionDetachLive for a signal shape it does not pin
          detail: 'That test SIGTERMs a bare zellij client (live_zellij.go:63-75). The non-owning branch uses handleCleanup: SIGTERM, then an unconditional SIGKILL of the `pair resume` process group, possibly mid-attach (couch.go:612-641). State why the result transfers (KILL runs no handlers, and the server sits outside the client''s group), or extend the live test.'
          family: live-conformance-shape
          round: 1
      blocked: true
---

# Gate ledger — pair#230 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-11T10:09:47-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `unverified-existing-behavior-claim` Warm registration is satisfied by the pre-existing session, so rows 3-4's injections cannot reach routes 3-4
  awaitResumeRegistration (launch_existing.go:238-256) returns once PairSession reports Present, and Present means listed-and-not-exited (artifactcollision.go:170-181), which a detached session already is. Row 3 ("Pair never registers") passes on the first poll. BeforeRegistration fires only from Artifacts.Registration (artifactcollision_fake.go:219-223), which the resume path never calls. Use BeforePairSession for both rows, and correct the Problem's "slow pair resume times out" reachability claim: a warm route 3 needs PairSession to fail for the whole budget, or the session to vanish.
- **PQ-2** [Important] `spec-plan-divergence` Spec and plan disagree on the routes 1-4 rollback rule, and no Revision records it
  The Spec rolls back only once the session is proven present, and marks the start unknown when presence cannot be observed. The plan rolls back once the helper is dead, whatever the session's presence. The plan's rule is defensible. Adopt one of the two and append a Revision amending the other, so the close review has a single contract to check.
- **PQ-3** [Important] `destructive-zero-value-default` Session ownership travels on a caller-relayed StartResult whose zero value is the destructive branch
  AbortStarted already treats StartResult as untrusted and re-validates it against c.reg (couch.go:527-552). Warm reaches it through couchtty's StartedChild relay, and RelaunchResult.Started (relaunch.go:52-57) already rebuilds StartResult field by field. Any relay that drops Warm deletes a warm session, which breaks the Spec's rule that deletion is never the answer to not knowing. The ARCH-SECURE note covers a spoofed true but not a dropped false. Either key ownership in couch by the ActorID that AbortStarted already looks up, or make the zero value non-destructive.
- **PQ-4** [Important] `stateless-fake-for-stateful-effect` The fake's Quiesce only logs the call, so the Detached-again assertion cannot fail under the named mutation
  FakeThreadArtifactCollisionChecker.Quiesce (artifactcollision_fake.go:236-247) appends to a log and leaves pairSessions and detachedSessions untouched. "The non-owning branch also quiesces" therefore still classifies ThreadDetached, and only the Quiesces() call log catches it. Make Quiesce clear the address's session state, in the fake or at least via a per-test QuiesceHook, so the behavioural assertion is a real oracle (ARCH-MOCK).
- **PQ-5** [Minor] `unverified-existing-behavior-claim` "quiescePostAckStart is the only caller of Artifacts.Quiesce" is false: ArchiveThread calls it too (detach.go:219)
  That call is the operator's deliberate delete, gated by archivableRecord, so the class is still six routes. Correct the sentence in both the plan and the issue Revision, and say why ArchiveThread is out of the class.
- **PQ-6** [Minor] `missing-non-goals` No non-goals section; three adjacent behaviours are left unstated
  (1) If the detached session dies between confirmStillDetached and exec, `pair resume <tag>` falls through to ActionCreate (decision.go:71-76), so a "warm" start can own a session it made. Non-owning cleanup then leaks that session rather than deleting it, which fails safe but should be written down. (2) The warm registration oracle proves nothing about the child attaching. (3) Couch dying between the helper kill and the durable write is left to reconcileInterruptedStarts.
- **PQ-7** [Minor] `arch-dry-reuse` Routes 5-6 re-implement Detach's post-signal half; extract it
  Detach's second presence proof plus the bounded RetireIncarnation retry on revision conflicts (detach.go:96-147) is what routes 5-6 need, with detachedAt passed in. Extract it as a shared helper. Its per-attempt ctx.Err() check must not make route 5 (which can run under a cancelled ctx) fail when it should fall back to MarkIncarnationUnknown.
- **PQ-8** [Minor] `arch-pure-decision` The cleanup decision is buried in IO functions; a pure decider like ReconcileStart would test it exhaustively
  (owns, helper liveness, presence observed/absent/unobservable, creating vs live) -> {rollback, retire, mark-unknown, quiesce} is a small pure table (precedent: ReconcileStart, starttransaction.go:178). Unit-test it exhaustively; the fake-seam table then only proves each route's wiring.
- **PQ-9** [Minor] `cross-issue-seam-change` Moving operationdispatch to ResumeStart changes the seam #206 M2 designs against
  #206's plan adds ResumeOptions{WarmOnly} to ResumeContext and has operationdispatch pass it. After this change those options belong on ResumeStart. Note it here so #206's plan gets a Revision before M2 starts.
- **PQ-10** [Minor] `test-prose-enumeration` Task 1 enumerates per-row injections and assertions in prose; compress to one strategy line per risky function
  Rows 3-4 have already gone stale (see the Important above). Keep the six-route table, which is the class enumeration. Replace the per-row bullets with one line per function: "launchTrackedThread/AbortStarted post-ack exits -> table over six routes x {warm, owning} at the fake seam; guard: per-route mutation."
- **PQ-11** [Minor] `live-conformance-shape` The Spec cites TestSessionDetachLive for a signal shape it does not pin
  That test SIGTERMs a bare zellij client (live_zellij.go:63-75). The non-owning branch uses handleCleanup: SIGTERM, then an unconditional SIGKILL of the `pair resume` process group, possibly mid-attach (couch.go:612-641). State why the result transfers (KILL runs no handlers, and the server sits outside the client's group), or extend the live test.

## Open findings

- **PQ-1** [Important] `unverified-existing-behavior-claim` Warm registration is satisfied by the pre-existing session, so rows 3-4's injections cannot reach routes 3-4
- **PQ-2** [Important] `spec-plan-divergence` Spec and plan disagree on the routes 1-4 rollback rule, and no Revision records it
- **PQ-3** [Important] `destructive-zero-value-default` Session ownership travels on a caller-relayed StartResult whose zero value is the destructive branch
- **PQ-4** [Important] `stateless-fake-for-stateful-effect` The fake's Quiesce only logs the call, so the Detached-again assertion cannot fail under the named mutation
- **PQ-5** [Minor] `unverified-existing-behavior-claim` "quiescePostAckStart is the only caller of Artifacts.Quiesce" is false: ArchiveThread calls it too (detach.go:219)
- **PQ-6** [Minor] `missing-non-goals` No non-goals section; three adjacent behaviours are left unstated
- **PQ-7** [Minor] `arch-dry-reuse` Routes 5-6 re-implement Detach's post-signal half; extract it
- **PQ-8** [Minor] `arch-pure-decision` The cleanup decision is buried in IO functions; a pure decider like ReconcileStart would test it exhaustively
- **PQ-9** [Minor] `cross-issue-seam-change` Moving operationdispatch to ResumeStart changes the seam #206 M2 designs against
- **PQ-10** [Minor] `test-prose-enumeration` Task 1 enumerates per-row injections and assertions in prose; compress to one strategy line per risky function
- **PQ-11** [Minor] `live-conformance-shape` The Spec cites TestSessionDetachLive for a signal shape it does not pin
