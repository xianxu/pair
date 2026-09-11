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
    - "n": 2
      timestamp: "2026-09-11T10:25:48-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Row 3 now kills the session from AfterAcknowledge so awaitResumeRegistration (launch_existing.go:245-255) times out and asserts ReasonSessionGone; row 4 bumps the revision concurrently; the route-6 explanation matches artifactcollision.go:168-182, and Task 5 logs it (the Problem section's slow-resume bullet is still stale).
          round: 2
        - id: PQ-2
          disposition: addressed
          note: 'Spec rewritten and a "rollback rule, stated once" Revision appended: rollback needs a dead helper, retire also needs observed presence, and DecideStartCleanup is the single home.'
          round: 2
        - id: PQ-3
          disposition: addressed
          note: AbortStarted reads the registry's ActorRecord.Warm, found by ActorID (couch.go:538-549); c.reg is never reloaded mid-life (writes only at launch_existing.go:148, couch.go:555/821/943), so a relayed zero value cannot select quiesce.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Task 2 makes the fake's Quiesce clear the detached session and mark PairSession absent; this will flip TestResumeAmbiguousAckKeepsUnknownOccupied (resume_launch_test.go:135) from unknown to rollback, which matches production and Task 2 Step 3 anticipates.
          round: 2
        - id: PQ-5
          disposition: not-addressed
          note: 'The "only caller" sentence remains in the plan and the issue Revision (detach.go:219 is a second caller); same family: "Unknown is recoverable" is false inside couch, where the only exit from IncarnationUnknown is park (park.go:228/651/772), which quits the agent. Rule for the family: every universally-quantified claim about existing code (only/never/pins/recoverable) cites the function, test or grep that proves it.'
          round: 2
        - id: PQ-6
          disposition: not-addressed
          note: Still no non-goals section; the Open question covers only the SIGKILL policy, not the create fall-through, the attach-proof gap, or couch dying mid-cleanup.
          round: 2
        - id: PQ-7
          disposition: not-addressed
          note: CleanupRetire "retries a revision conflict the bounded way Detach does", a second copy of detach.go:118-147 rather than an extracted helper; the ctx.Err() caveat for route 5 still applies.
          round: 2
        - id: PQ-8
          disposition: addressed
          note: 'DecideStartCleanup plus a 24-row table; caveat for close: CleanupQuiesce omits the owning paths'' durable tails (cold rollback-or-unknown at launch_existing.go:177-185; spawn and owning routes 5-6 reconcile plus MarkIncarnationUnknown at couch.go:505-518), so keep them verbatim or extend the action set; existing tests would catch dropping them.'
          round: 2
        - id: PQ-9
          disposition: addressed
          note: '#206''s plan records the ResumeStart seam (lines 31-33, 181, 365-367); observation for close: after PQ-3 nothing reads StartResult.Warm (not the console, not #206), so ResumeStart and the startup/operationdispatch edits carry an unused field; dropping them would let #206 stay on ResumeContext.'
          round: 2
        - id: PQ-10
          disposition: not-addressed
          note: The per-route injections are now correct and arguably design content given PQ-1; the per-row assertion bullets in Task 3 Step 2 are what should compress.
          round: 2
        - id: PQ-11
          disposition: not-addressed
          note: The plan concedes TestSessionDetachLive pins only SIGTERM, but the Spec still cites it; the one-line reason is that the zellij server and agent predate the fresh pair-resume process group, so the group SIGKILL (couch.go:619-628) cannot reach them.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-09-11T10:40:32-07:00"
      agent: claude
      dispose:
        - id: PQ-5
          disposition: not-addressed
          note: Plan fixed (couch.go:580, detach.go:219). The issue's Revision still says "the only caller" and "StartResult carries Warm", and the plan's ARCH-SECURE bullet repeats StartResult.Warm; append a correcting Revision.
          round: 3
        - id: PQ-6
          disposition: not-addressed
          note: The Non-goals section lists other items; the ActionCreate fallthrough (decision.go:71-76), the registration oracle, and couch dying mid-cleanup are still unstated as non-goals.
          round: 3
        - id: PQ-7
          disposition: not-addressed
          note: Extraction planned, but stripping ctx.Err() from the shared loop removes Detach's interrupt (detach.go:119-129), and no test pins it; keep the check and pass context.WithoutCancel at routes 5-6.
          round: 3
        - id: PQ-10
          disposition: not-addressed
          note: 'Task 3 is still per-row prose and row 6 is stale (ResumeStart; see resume.go:328); Task 5 has three rows that cannot be killed as named: start.Warm, the decider''s unused Quiesce output, and route 5''s uncancelled ctx.'
          round: 3
        - id: PQ-11
          disposition: addressed
          note: The Open question says why SIGKILL carries over (helper group only), and Detach already SIGTERMs a pair client's group in production (detach.go:89).
          round: 3
      findings:
        - id: PQ-12
          severity: Important
          title: The decider's owning+claim quadrant is only cold resume's tail; spawn at routes 1-4 takes failPostAckStart's reconcile (launch_existing.go:169-171)
          detail: 'This is the 3rd finding in this family, after PQ-5 and PQ-11. All three are claims about existing code made without reading the whole function or call graph they describe. Rule: a claim that the plan preserves or restates existing behaviour must be derived from the branch table of every function it replaces, each if/switch arm with its file:line, and every condition that code branches on must either become a decider input or be named as deliberately dropped. Applied to failTrackedPostAckStart, failPostAckStart and ReconcileStart, the rule surfaces resume (launch_existing.go:169), which is not a decider input: a spawn''s routes 1-4 tail is reconcileInterruptedStarts on registration evidence, not rollback-or-unknown on presence. As written, an unset fake binding reads PresenceUnobserved, gives mark-unknown, and fails couch_test.go:799-805, :873 and :890. Fix: give the decider the start''s shape (spawn, cold resume or warm; 36 rows), with spawn going to DurableReconcile in both phases. Prevalence: 3 findings in 3 rounds; 1 of roughly 15 existing-behaviour claims checked this round was wrong.'
          family: unverified-existing-behavior-claim
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-11T10:47:40-07:00"
      agent: claude
      dispose:
        - id: PQ-5
          disposition: not-addressed
          note: Plan correct (couch.go:580, detach.go:219); the issue Revision still says "only caller" and "StartResult carries Warm", ARCH-SECURE still names StartResult.Warm (the real relayed copy is StartResult.Record.Warm), and the Estimate still says 24 rows; append a correcting Revision.
          round: 4
        - id: PQ-6
          disposition: not-addressed
          note: Non-goals now exists but lists other items; the ActionCreate fall-through, the attach-proof gap and couch dying mid-cleanup are still unstated as non-goals.
          round: 4
        - id: PQ-7
          disposition: not-addressed
          note: 'Unchanged since round 3: stripping ctx.Err() from the shared loop removes Detach''s interrupt (detach.go:126-129), unpinned by any test; keep it and pass context.WithoutCancel at route 5.'
          round: 4
        - id: PQ-10
          disposition: not-addressed
          note: Task 3 is still per-row prose; row 6 still names ResumeStart, which the plan drops; Task 5's "start.Warm" mutation names a nonexistent field (should be start.Record.Warm).
          round: 4
        - id: PQ-12
          disposition: addressed
          note: Three-valued StartShape, spawn goes to DurableReconcile in both phases; every arm of launch_existing.go:168-186 and couch.go:502-520 maps to an input or output; 36-row table passes.
          round: 4
      blocked: false
content_hash: 04be5dfa09fff332ccfdd034a9837136d69a4d1a91087ee5f3334501fc7e68cf
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

## Round 2 — 2026-09-11T10:25:48-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Row 3 now kills the session from AfterAcknowledge so awaitResumeRegistration (launch_existing.go:245-255) times out and asserts ReasonSessionGone; row 4 bumps the revision concurrently; the route-6 explanation matches artifactcollision.go:168-182, and Task 5 logs it (the Problem section's slow-resume bullet is still stale).
- PQ-2 — addressed — Spec rewritten and a "rollback rule, stated once" Revision appended: rollback needs a dead helper, retire also needs observed presence, and DecideStartCleanup is the single home.
- PQ-3 — addressed — AbortStarted reads the registry's ActorRecord.Warm, found by ActorID (couch.go:538-549); c.reg is never reloaded mid-life (writes only at launch_existing.go:148, couch.go:555/821/943), so a relayed zero value cannot select quiesce.
- PQ-4 — addressed — Task 2 makes the fake's Quiesce clear the detached session and mark PairSession absent; this will flip TestResumeAmbiguousAckKeepsUnknownOccupied (resume_launch_test.go:135) from unknown to rollback, which matches production and Task 2 Step 3 anticipates.
- PQ-5 — not-addressed — The "only caller" sentence remains in the plan and the issue Revision (detach.go:219 is a second caller); same family: "Unknown is recoverable" is false inside couch, where the only exit from IncarnationUnknown is park (park.go:228/651/772), which quits the agent. Rule for the family: every universally-quantified claim about existing code (only/never/pins/recoverable) cites the function, test or grep that proves it.
- PQ-6 — not-addressed — Still no non-goals section; the Open question covers only the SIGKILL policy, not the create fall-through, the attach-proof gap, or couch dying mid-cleanup.
- PQ-7 — not-addressed — CleanupRetire "retries a revision conflict the bounded way Detach does", a second copy of detach.go:118-147 rather than an extracted helper; the ctx.Err() caveat for route 5 still applies.
- PQ-8 — addressed — DecideStartCleanup plus a 24-row table; caveat for close: CleanupQuiesce omits the owning paths' durable tails (cold rollback-or-unknown at launch_existing.go:177-185; spawn and owning routes 5-6 reconcile plus MarkIncarnationUnknown at couch.go:505-518), so keep them verbatim or extend the action set; existing tests would catch dropping them.
- PQ-9 — addressed — #206's plan records the ResumeStart seam (lines 31-33, 181, 365-367); observation for close: after PQ-3 nothing reads StartResult.Warm (not the console, not #206), so ResumeStart and the startup/operationdispatch edits carry an unused field; dropping them would let #206 stay on ResumeContext.
- PQ-10 — not-addressed — The per-route injections are now correct and arguably design content given PQ-1; the per-row assertion bullets in Task 3 Step 2 are what should compress.
- PQ-11 — not-addressed — The plan concedes TestSessionDetachLive pins only SIGTERM, but the Spec still cites it; the one-line reason is that the zellij server and agent predate the fresh pair-resume process group, so the group SIGKILL (couch.go:619-628) cannot reach them.

## Round 3 — 2026-09-11T10:40:32-07:00 (claude) — BLOCKED

### Disposed

- PQ-5 — not-addressed — Plan fixed (couch.go:580, detach.go:219). The issue's Revision still says "the only caller" and "StartResult carries Warm", and the plan's ARCH-SECURE bullet repeats StartResult.Warm; append a correcting Revision.
- PQ-6 — not-addressed — The Non-goals section lists other items; the ActionCreate fallthrough (decision.go:71-76), the registration oracle, and couch dying mid-cleanup are still unstated as non-goals.
- PQ-7 — not-addressed — Extraction planned, but stripping ctx.Err() from the shared loop removes Detach's interrupt (detach.go:119-129), and no test pins it; keep the check and pass context.WithoutCancel at routes 5-6.
- PQ-10 — not-addressed — Task 3 is still per-row prose and row 6 is stale (ResumeStart; see resume.go:328); Task 5 has three rows that cannot be killed as named: start.Warm, the decider's unused Quiesce output, and route 5's uncancelled ctx.
- PQ-11 — addressed — The Open question says why SIGKILL carries over (helper group only), and Detach already SIGTERMs a pair client's group in production (detach.go:89).

### Raised

- **PQ-12** [Important] `unverified-existing-behavior-claim` The decider's owning+claim quadrant is only cold resume's tail; spawn at routes 1-4 takes failPostAckStart's reconcile (launch_existing.go:169-171)
  This is the 3rd finding in this family, after PQ-5 and PQ-11. All three are claims about existing code made without reading the whole function or call graph they describe. Rule: a claim that the plan preserves or restates existing behaviour must be derived from the branch table of every function it replaces, each if/switch arm with its file:line, and every condition that code branches on must either become a decider input or be named as deliberately dropped. Applied to failTrackedPostAckStart, failPostAckStart and ReconcileStart, the rule surfaces resume (launch_existing.go:169), which is not a decider input: a spawn's routes 1-4 tail is reconcileInterruptedStarts on registration evidence, not rollback-or-unknown on presence. As written, an unset fake binding reads PresenceUnobserved, gives mark-unknown, and fails couch_test.go:799-805, :873 and :890. Fix: give the decider the start's shape (spawn, cold resume or warm; 36 rows), with spawn going to DurableReconcile in both phases. Prevalence: 3 findings in 3 rounds; 1 of roughly 15 existing-behaviour claims checked this round was wrong.

## Round 4 — 2026-09-11T10:47:40-07:00 (claude) — passed

### Disposed

- PQ-5 — not-addressed — Plan correct (couch.go:580, detach.go:219); the issue Revision still says "only caller" and "StartResult carries Warm", ARCH-SECURE still names StartResult.Warm (the real relayed copy is StartResult.Record.Warm), and the Estimate still says 24 rows; append a correcting Revision.
- PQ-6 — not-addressed — Non-goals now exists but lists other items; the ActionCreate fall-through, the attach-proof gap and couch dying mid-cleanup are still unstated as non-goals.
- PQ-7 — not-addressed — Unchanged since round 3: stripping ctx.Err() from the shared loop removes Detach's interrupt (detach.go:126-129), unpinned by any test; keep it and pass context.WithoutCancel at route 5.
- PQ-10 — not-addressed — Task 3 is still per-row prose; row 6 still names ResumeStart, which the plan drops; Task 5's "start.Warm" mutation names a nonexistent field (should be start.Record.Warm).
- PQ-12 — addressed — Three-valued StartShape, spawn goes to DurableReconcile in both phases; every arm of launch_existing.go:168-186 and couch.go:502-520 maps to an input or output; 36-row table passes.

## Open findings

- **PQ-5** [Minor] `unverified-existing-behavior-claim` "quiescePostAckStart is the only caller of Artifacts.Quiesce" is false: ArchiveThread calls it too (detach.go:219)
- **PQ-6** [Minor] `missing-non-goals` No non-goals section; three adjacent behaviours are left unstated
- **PQ-7** [Minor] `arch-dry-reuse` Routes 5-6 re-implement Detach's post-signal half; extract it
- **PQ-10** [Minor] `test-prose-enumeration` Task 1 enumerates per-row injections and assertions in prose; compress to one strategy line per risky function
