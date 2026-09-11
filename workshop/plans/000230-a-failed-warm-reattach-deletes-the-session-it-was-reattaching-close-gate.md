---
gate: boundary-review
issue: 230
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-11T11:15:44-07:00"
      agent: sdlc
      findings:
        - id: BR-1
          severity: Minor
          title: '"quiescePostAckStart is the only caller of Artifacts.Quiesce" is false: ArchiveThread calls it too (detach.go:219)'
          detail: |-
            That call is the operator's deliberate delete, gated by archivableRecord, so the class is still six routes. Correct the sentence in both the plan and the issue Revision, and say why ArchiveThread is out of the class.
            (carried from plan-quality PQ-5, deferred to the boundary review)
          family: unverified-existing-behavior-claim
          round: 1
        - id: BR-2
          severity: Minor
          title: No non-goals section; three adjacent behaviours are left unstated
          detail: |-
            (1) If the detached session dies between confirmStillDetached and exec, `pair resume <tag>` falls through to ActionCreate (decision.go:71-76), so a "warm" start can own a session it made. Non-owning cleanup then leaks that session rather than deleting it, which fails safe but should be written down. (2) The warm registration oracle proves nothing about the child attaching. (3) Couch dying between the helper kill and the durable write is left to reconcileInterruptedStarts.
            (carried from plan-quality PQ-6, deferred to the boundary review)
          family: missing-non-goals
          round: 1
        - id: BR-3
          severity: Minor
          title: Routes 5-6 re-implement Detach's post-signal half; extract it
          detail: |-
            Detach's second presence proof plus the bounded RetireIncarnation retry on revision conflicts (detach.go:96-147) is what routes 5-6 need, with detachedAt passed in. Extract it as a shared helper. Its per-attempt ctx.Err() check must not make route 5 (which can run under a cancelled ctx) fail when it should fall back to MarkIncarnationUnknown.
            (carried from plan-quality PQ-7, deferred to the boundary review)
          family: arch-dry-reuse
          round: 1
        - id: BR-4
          severity: Minor
          title: Task 1 enumerates per-row injections and assertions in prose; compress to one strategy line per risky function
          detail: |-
            Rows 3-4 have already gone stale (see the Important above). Keep the six-route table, which is the class enumeration. Replace the per-row bullets with one line per function: "launchTrackedThread/AbortStarted post-ack exits -> table over six routes x {warm, owning} at the fake seam; guard: per-route mutation."
            (carried from plan-quality PQ-10, deferred to the boundary review)
          family: test-prose-enumeration
          round: 1
      boundary: '*'
      no_cap: true
      blocked: false
    - "n": 2
      timestamp: "2026-09-11T11:15:44-07:00"
      agent: claude
      findings:
        - id: BR-5
          severity: Important
          title: A failed retire on the warm live-record routes returns with no durable fallback, leaving a live incarnation behind a dead helper
          detail: |-
            couch.go:514 returns errors.Join(cause, cleanupErr, retireErr) directly, so every failure of
            retireDetachedIncarnation (session lost between observeSessionPresence and the retire's re-proof,
            exhausted revision conflicts, a RetireIncarnation precondition refusal) skips the mark-unknown tail
            the Spec and the decider prescribe. Pre-pair#230 this path always reached MarkIncarnationUnknown.
            Probe-confirmed via a read-only overlay test: warm route 5 leaves incarnation state "live" and the
            switcher renders unusable/stale-incarnation. Fix: fall through to the mark-unknown tail on retireErr,
            and pin it with a PairSession hook that reports present once then absent.
          family: fail-closed-fallback-missing
          round: 2
        - id: BR-6
          severity: Important
          title: StartCleanup.Quiesce is never read by production; the quiesce half is decided at three call sites
          detail: |-
            Measured: forcing Quiesce true for every input (startcleanup.go:98) leaves the entire seam suite green;
            only the decider's own table and property tests fail. Production reads shape.OwnsSession() at couch.go:503,
            couch.go:507 and launch_existing.go:191. The atlas paragraph and the plan claim the pure function decides
            whether to quiesce. Fix: drop the field and name OwnsSession() as the single source the call sites consume,
            or route the quiesce through a shell that consumes the decision; update atlas/couch.md to match.
          family: unconsumed-single-source
          round: 2
        - id: BR-7
          severity: Important
          title: Core concepts tables name applyStartCleanup, which was never built, and classify an IO helper as PURE
          detail: |-
            plan.md:172 (and :325, :333) lists applyStartCleanup as a new integration entity in couch.go; it does not
            exist, and the two ad-hoc tails it was to replace remain, each consulting the decider differently.
            plan.md:72 lists retireDetachedIncarnation under Pure entities though it observes PairSession and writes
            the thread store. The review contract rates a table/code contradiction Critical; rated Important here
            because the code is correct and tested and the divergence is in the plan's description. Needs a
            "## Revisions" entry or the refactor before archiving.
          family: plan-table-vs-code-drift
          round: 2
        - id: BR-8
          severity: Important
          title: The issue's Revisions still assert StartResult.Warm, a sole Quiesce caller, and a 24-row table
          detail: |-
            issue:169 says StartResult carries Warm (ownership actually comes from ActorRecord.Warm); issue:154 says
            quiescePostAckStart is the only caller of Artifacts.Quiesce (ArchiveThread is the second, detach.go:246);
            issue:113 says 24-row exhaustive table (36 rows shipped). The plan gate raised exactly these as PQ-5 and
            disposed them not-addressed in rounds 2, 3 and 4, and the close Log's carry-forward list drops PQ-5
            silently. Fix: one correcting "## Revisions" entry plus a Log line disposing PQ-5.
          family: stale-artifact-claim
          round: 2
        - id: BR-9
          severity: Minor
          title: The DurableRetire arm of failTrackedPostAckStart is unreachable and duplicates couch.go's retire block
          detail: |-
            At claim phase no shape yields DurableRetire (spawn returns early; cold and warm claims yield rollback or
            mark-unknown), so launch_existing.go:205 can never execute, and the ctx built at :190 exists only to feed
            it. Additionally context.WithoutCancel(context.Background()) is a no-op at both :190 and couch.go:522 --
            neither function receives the caller's context, so "uncancellable" comes from Background, not the wrapper.
          family: dead-branch
          round: 2
        - id: BR-10
          severity: Minor
          title: Route 3's warm assertion is weaker than the plan promised
          detail: |-
            warm_failure_test.go:167 asserts only that the row is not ThreadDetached; the plan promised
            ReasonSessionGone. Any wrong-but-not-detached state passes.
          family: weak-assertion
          round: 2
        - id: BR-11
          severity: Minor
          title: AbortStarted labels a spawn StartColdResume, and rescans the registry it just searched
          detail: |-
            couch.go:583 defaults to StartColdResume for any non-warm start, including a spawn (harmless today since
            both own their session, but the name is false); couch.go:584 runs a second full c.reg.Records() scan
            immediately after the identity loop found the same record, which the plan said would be reused.
          family: shape-label-mismatch
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-11T11:39:13-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: Plan "The class, enumerated" names both Quiesce callers and why ArchiveThread (detach.go:246) is out of class; the issue's PQ-5/BR-8 Revision corrects the sole-caller claim.
          round: 3
        - id: BR-2
          disposition: not-addressed
          note: The new Non-goals lists three different items; the pair resume create fall-through and couch dying between helper kill and durable write appear nowhere in plan or issue.
          round: 3
        - id: BR-3
          disposition: addressed
          note: retireDetachedIncarnation is shared by Detach (detach.go:97) and applyStartCleanup (couch.go:540); ctx.Err() kept, and cleanup passes Background so cancellation cannot interrupt it.
          round: 3
        - id: BR-4
          disposition: not-addressed
          note: Task 3 still enumerates rows in prose and still promises Spawn rows for routes 5-6 that the table does not have; archival, never blocks.
          round: 3
        - id: BR-5
          disposition: addressed
          note: couch.go:549 falls through to markLiveRecordUnknown; reverting it reddens TestAFailedRetireStillLeavesTheRecordRecoverable by name (record keeps state=live).
          round: 3
        - id: BR-6
          disposition: addressed
          note: Quiesce field gone, OwnsSession is the single source, and the ownership-ignoring mutation reddens all warm rows; the stale atlas sentence is carried in the new stale-artifact-claim finding.
          round: 3
        - id: BR-7
          disposition: addressed
          note: applyStartCleanup exists (couch.go:515) as the single shell, retireDetachedIncarnation sits under Integration points, and the plan Revisions records StartCleanup removed and Shape as a string.
          round: 3
        - id: BR-8
          disposition: addressed
          note: The issue Revision titled three-claims-are-wrong (PQ-5, BR-8) corrects StartResult.Warm, the sole-caller claim and the row count, and names PQ-5.
          round: 3
        - id: BR-9
          disposition: addressed
          note: failTrackedPostAckStart is one line; the retire arm lives only in the shared shell at live-record phase; the WithoutCancel no-op is gone (its detach.go comment is in the new finding).
          round: 3
        - id: BR-10
          disposition: addressed
          note: warm_failure_test.go:168 now requires ThreadUnusable with ReasonSessionGone, and passes.
          round: 3
        - id: BR-11
          disposition: addressed
          note: ActorRecord.Shape records spawn/cold/warm and AbortStarted reads it in the identity loop with no second scan; labelling a spawn cold reddens three spawn tests.
          round: 3
      findings:
        - id: BR-12
          severity: Important
          title: The shared cleanup shell drops the session-observation error that the old cold-resume tail returned
          detail: |-
            observeSessionPresence (launch_existing.go:191-204) collapses a PairSession error, or a missing PairSessionIO,
            into PresenceUnobserved and discards it; applyStartCleanup (couch.go:515-561) never reports it. The base tail
            joined bindingErr and "exact Pair session observer is unavailable" into the returned error. Overlay probe
            (cold resume, ack fails, BeforePairSession returns "zellij unreachable"): base returns the ack cause plus
            "zellij unreachable" and passes; head returns only the ack cause and fails. The record still lands Unknown,
            hence Important not Critical, but the operator loses why it was not rolled back, and the new warm live-record
            arm inherits the silence (ARCH-SECURE, degrade visibly). Fix: return (SessionPresence, error), join it in
            applyStartCleanup, and assert "zellij unreachable" in TestResumeUnobservableSessionKeepsUnknownOccupied.
          family: swallowed-error-context
          round: 3
        - id: BR-13
          severity: Important
          title: Eight comment and doc passages still restate design decisions this issue reversed, including the atlas
          detail: |-
            2nd finding in family stale-artifact-claim, so this states the rule rather than one site. Rule: a commit that
            reverses a design decision sweeps every restatement of the old one in the same commit; grep the retired names
            and claims (Warm, zero value, WithoutCancel, whether to quiesce, StartCleanup) across code comments, test
            comments, atlas and plan, and fix each hit or record a Revisions line. Measured prevalence, 8 sites in 5 files:
            atlas/couch.md:679 says the decider returns whether to quiesce (BR-6 residue); atlas/couch.md:686,
            couch.go:613-616 and couch.go:623-626 (duplicated) say the relayed struct's zero value is destructive, but an
            empty Shape answers non-owning, the safe side; warm_failure_test.go:271-276 and :298 keep the Warm/zero-value
            framing though the test relays StartColdResume; detach.go:149-151 says cleanup passes context.WithoutCancel
            but it passes context.Background() (couch.go:540); plan.md:194-200 and its Task 5 row still describe
            WithoutCancel with no Revisions entry covering it.
          family: stale-artifact-claim
          round: 3
      blocked: true
    - "n": 4
      timestamp: "2026-09-11T11:57:29-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: not-addressed
          note: The new Non-goals (plan.md:223-234) lists other items; (1) the ActionCreate fallthrough after confirmStillDetached (resume.go:425, launcher/decision.go:74,103) and (3) couch dying between helper kill and durable write are still unstated anywhere.
          round: 4
        - id: BR-4
          disposition: not-addressed
          note: 'Task 3 (plan.md:294-316) is still per-row prose and stale again: it promises Spawn owning rows for routes 5-6 and an owning route 3; the shipped owning table is cold-resume only and skips route 3, with no Revisions line.'
          round: 4
        - id: BR-12
          disposition: addressed
          note: 'Verified by reverting: dropping the error in observeSessionPresence, or the join at couch.go:522, turns TestCleanupSurfacesWhyItCouldNotObserveTheSession red.'
          round: 4
        - id: BR-13
          disposition: not-addressed
          note: 'The 8 named sites are fixed but 7 remain in 4 files: plan.md:216-217 still names StartResult.Warm (on the rule''s own list), and startcleanup.go:20-21,102-103,106-107, startcleanup_test.go:27-28,43-44 and couch.go:498-501 describe the deleted if-!resume arm and failPostAckStart as the spawn/owning tail. Rule, 3rd time in this family: derive the grep list from what the diff deleted or re-scoped, not from the finding''s examples.'
          round: 4
      findings:
        - id: BR-14
          severity: Minor
          title: applyStartCleanup joins the session-observation error for shapes whose decision never reads presence
          detail: 'couch.go:521-522 observes and joins for every shape; probe: a spawn whose acknowledge fails now also returns "exact Pair session binding is absent", which base never produced. The comments at couch.go:519-520, launch_existing.go:192-195 and warm_failure_test.go:362-364 say it explains why the thread was left occupied, yet the pinning test''s warm claim rolls back. Observe only where the decider reads presence (cold claim, warm live record), and move the assertion to TestResumeUnobservableSessionKeepsUnknownOccupied.'
          family: diagnostic-scoped-to-decision
          round: 4
        - id: BR-15
          severity: Minor
          title: The DurableRetire arm returns on a GetThread error before any disposition, so the atlas "no path" claim overclaims
          detail: '2nd finding in this family. Rule: at the live-record phase, every applyStartCleanup exit that has not retired the incarnation falls through to markLiveRecordUnknown, as one structural fallback after the switch rather than per arm. Measured prevalence: 1 remaining site, couch.go:537-540, which contradicts atlas/couch.md:685-686.'
          family: fail-closed-fallback-missing
          round: 4
      blocked: true
---

# Gate ledger — pair#230 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-11T11:15:44-07:00 (sdlc) — passed

### Raised

- **BR-1** [Minor] `unverified-existing-behavior-claim` "quiescePostAckStart is the only caller of Artifacts.Quiesce" is false: ArchiveThread calls it too (detach.go:219)
  That call is the operator's deliberate delete, gated by archivableRecord, so the class is still six routes. Correct the sentence in both the plan and the issue Revision, and say why ArchiveThread is out of the class.
  (carried from plan-quality PQ-5, deferred to the boundary review)
- **BR-2** [Minor] `missing-non-goals` No non-goals section; three adjacent behaviours are left unstated
  (1) If the detached session dies between confirmStillDetached and exec, `pair resume <tag>` falls through to ActionCreate (decision.go:71-76), so a "warm" start can own a session it made. Non-owning cleanup then leaks that session rather than deleting it, which fails safe but should be written down. (2) The warm registration oracle proves nothing about the child attaching. (3) Couch dying between the helper kill and the durable write is left to reconcileInterruptedStarts.
  (carried from plan-quality PQ-6, deferred to the boundary review)
- **BR-3** [Minor] `arch-dry-reuse` Routes 5-6 re-implement Detach's post-signal half; extract it
  Detach's second presence proof plus the bounded RetireIncarnation retry on revision conflicts (detach.go:96-147) is what routes 5-6 need, with detachedAt passed in. Extract it as a shared helper. Its per-attempt ctx.Err() check must not make route 5 (which can run under a cancelled ctx) fail when it should fall back to MarkIncarnationUnknown.
  (carried from plan-quality PQ-7, deferred to the boundary review)
- **BR-4** [Minor] `test-prose-enumeration` Task 1 enumerates per-row injections and assertions in prose; compress to one strategy line per risky function
  Rows 3-4 have already gone stale (see the Important above). Keep the six-route table, which is the class enumeration. Replace the per-row bullets with one line per function: "launchTrackedThread/AbortStarted post-ack exits -> table over six routes x {warm, owning} at the fake seam; guard: per-route mutation."
  (carried from plan-quality PQ-10, deferred to the boundary review)

## Round 2 — 2026-09-11T11:15:44-07:00 (claude) — BLOCKED

### Raised

- **BR-5** [Important] `fail-closed-fallback-missing` A failed retire on the warm live-record routes returns with no durable fallback, leaving a live incarnation behind a dead helper
  couch.go:514 returns errors.Join(cause, cleanupErr, retireErr) directly, so every failure of
  retireDetachedIncarnation (session lost between observeSessionPresence and the retire's re-proof,
  exhausted revision conflicts, a RetireIncarnation precondition refusal) skips the mark-unknown tail
  the Spec and the decider prescribe. Pre-pair#230 this path always reached MarkIncarnationUnknown.
  Probe-confirmed via a read-only overlay test: warm route 5 leaves incarnation state "live" and the
  switcher renders unusable/stale-incarnation. Fix: fall through to the mark-unknown tail on retireErr,
  and pin it with a PairSession hook that reports present once then absent.
- **BR-6** [Important] `unconsumed-single-source` StartCleanup.Quiesce is never read by production; the quiesce half is decided at three call sites
  Measured: forcing Quiesce true for every input (startcleanup.go:98) leaves the entire seam suite green;
  only the decider's own table and property tests fail. Production reads shape.OwnsSession() at couch.go:503,
  couch.go:507 and launch_existing.go:191. The atlas paragraph and the plan claim the pure function decides
  whether to quiesce. Fix: drop the field and name OwnsSession() as the single source the call sites consume,
  or route the quiesce through a shell that consumes the decision; update atlas/couch.md to match.
- **BR-7** [Important] `plan-table-vs-code-drift` Core concepts tables name applyStartCleanup, which was never built, and classify an IO helper as PURE
  plan.md:172 (and :325, :333) lists applyStartCleanup as a new integration entity in couch.go; it does not
  exist, and the two ad-hoc tails it was to replace remain, each consulting the decider differently.
  plan.md:72 lists retireDetachedIncarnation under Pure entities though it observes PairSession and writes
  the thread store. The review contract rates a table/code contradiction Critical; rated Important here
  because the code is correct and tested and the divergence is in the plan's description. Needs a
  "## Revisions" entry or the refactor before archiving.
- **BR-8** [Important] `stale-artifact-claim` The issue's Revisions still assert StartResult.Warm, a sole Quiesce caller, and a 24-row table
  issue:169 says StartResult carries Warm (ownership actually comes from ActorRecord.Warm); issue:154 says
  quiescePostAckStart is the only caller of Artifacts.Quiesce (ArchiveThread is the second, detach.go:246);
  issue:113 says 24-row exhaustive table (36 rows shipped). The plan gate raised exactly these as PQ-5 and
  disposed them not-addressed in rounds 2, 3 and 4, and the close Log's carry-forward list drops PQ-5
  silently. Fix: one correcting "## Revisions" entry plus a Log line disposing PQ-5.
- **BR-9** [Minor] `dead-branch` The DurableRetire arm of failTrackedPostAckStart is unreachable and duplicates couch.go's retire block
  At claim phase no shape yields DurableRetire (spawn returns early; cold and warm claims yield rollback or
  mark-unknown), so launch_existing.go:205 can never execute, and the ctx built at :190 exists only to feed
  it. Additionally context.WithoutCancel(context.Background()) is a no-op at both :190 and couch.go:522 --
  neither function receives the caller's context, so "uncancellable" comes from Background, not the wrapper.
- **BR-10** [Minor] `weak-assertion` Route 3's warm assertion is weaker than the plan promised
  warm_failure_test.go:167 asserts only that the row is not ThreadDetached; the plan promised
  ReasonSessionGone. Any wrong-but-not-detached state passes.
- **BR-11** [Minor] `shape-label-mismatch` AbortStarted labels a spawn StartColdResume, and rescans the registry it just searched
  couch.go:583 defaults to StartColdResume for any non-warm start, including a spawn (harmless today since
  both own their session, but the name is false); couch.go:584 runs a second full c.reg.Records() scan
  immediately after the identity loop found the same record, which the plan said would be reused.

## Round 3 — 2026-09-11T11:39:13-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — Plan "The class, enumerated" names both Quiesce callers and why ArchiveThread (detach.go:246) is out of class; the issue's PQ-5/BR-8 Revision corrects the sole-caller claim.
- BR-2 — not-addressed — The new Non-goals lists three different items; the pair resume create fall-through and couch dying between helper kill and durable write appear nowhere in plan or issue.
- BR-3 — addressed — retireDetachedIncarnation is shared by Detach (detach.go:97) and applyStartCleanup (couch.go:540); ctx.Err() kept, and cleanup passes Background so cancellation cannot interrupt it.
- BR-4 — not-addressed — Task 3 still enumerates rows in prose and still promises Spawn rows for routes 5-6 that the table does not have; archival, never blocks.
- BR-5 — addressed — couch.go:549 falls through to markLiveRecordUnknown; reverting it reddens TestAFailedRetireStillLeavesTheRecordRecoverable by name (record keeps state=live).
- BR-6 — addressed — Quiesce field gone, OwnsSession is the single source, and the ownership-ignoring mutation reddens all warm rows; the stale atlas sentence is carried in the new stale-artifact-claim finding.
- BR-7 — addressed — applyStartCleanup exists (couch.go:515) as the single shell, retireDetachedIncarnation sits under Integration points, and the plan Revisions records StartCleanup removed and Shape as a string.
- BR-8 — addressed — The issue Revision titled three-claims-are-wrong (PQ-5, BR-8) corrects StartResult.Warm, the sole-caller claim and the row count, and names PQ-5.
- BR-9 — addressed — failTrackedPostAckStart is one line; the retire arm lives only in the shared shell at live-record phase; the WithoutCancel no-op is gone (its detach.go comment is in the new finding).
- BR-10 — addressed — warm_failure_test.go:168 now requires ThreadUnusable with ReasonSessionGone, and passes.
- BR-11 — addressed — ActorRecord.Shape records spawn/cold/warm and AbortStarted reads it in the identity loop with no second scan; labelling a spawn cold reddens three spawn tests.

### Raised

- **BR-12** [Important] `swallowed-error-context` The shared cleanup shell drops the session-observation error that the old cold-resume tail returned
  observeSessionPresence (launch_existing.go:191-204) collapses a PairSession error, or a missing PairSessionIO,
  into PresenceUnobserved and discards it; applyStartCleanup (couch.go:515-561) never reports it. The base tail
  joined bindingErr and "exact Pair session observer is unavailable" into the returned error. Overlay probe
  (cold resume, ack fails, BeforePairSession returns "zellij unreachable"): base returns the ack cause plus
  "zellij unreachable" and passes; head returns only the ack cause and fails. The record still lands Unknown,
  hence Important not Critical, but the operator loses why it was not rolled back, and the new warm live-record
  arm inherits the silence (ARCH-SECURE, degrade visibly). Fix: return (SessionPresence, error), join it in
  applyStartCleanup, and assert "zellij unreachable" in TestResumeUnobservableSessionKeepsUnknownOccupied.
- **BR-13** [Important] `stale-artifact-claim` Eight comment and doc passages still restate design decisions this issue reversed, including the atlas
  2nd finding in family stale-artifact-claim, so this states the rule rather than one site. Rule: a commit that
  reverses a design decision sweeps every restatement of the old one in the same commit; grep the retired names
  and claims (Warm, zero value, WithoutCancel, whether to quiesce, StartCleanup) across code comments, test
  comments, atlas and plan, and fix each hit or record a Revisions line. Measured prevalence, 8 sites in 5 files:
  atlas/couch.md:679 says the decider returns whether to quiesce (BR-6 residue); atlas/couch.md:686,
  couch.go:613-616 and couch.go:623-626 (duplicated) say the relayed struct's zero value is destructive, but an
  empty Shape answers non-owning, the safe side; warm_failure_test.go:271-276 and :298 keep the Warm/zero-value
  framing though the test relays StartColdResume; detach.go:149-151 says cleanup passes context.WithoutCancel
  but it passes context.Background() (couch.go:540); plan.md:194-200 and its Task 5 row still describe
  WithoutCancel with no Revisions entry covering it.

## Round 4 — 2026-09-11T11:57:29-07:00 (claude) — BLOCKED

### Disposed

- BR-2 — not-addressed — The new Non-goals (plan.md:223-234) lists other items; (1) the ActionCreate fallthrough after confirmStillDetached (resume.go:425, launcher/decision.go:74,103) and (3) couch dying between helper kill and durable write are still unstated anywhere.
- BR-4 — not-addressed — Task 3 (plan.md:294-316) is still per-row prose and stale again: it promises Spawn owning rows for routes 5-6 and an owning route 3; the shipped owning table is cold-resume only and skips route 3, with no Revisions line.
- BR-12 — addressed — Verified by reverting: dropping the error in observeSessionPresence, or the join at couch.go:522, turns TestCleanupSurfacesWhyItCouldNotObserveTheSession red.
- BR-13 — not-addressed — The 8 named sites are fixed but 7 remain in 4 files: plan.md:216-217 still names StartResult.Warm (on the rule's own list), and startcleanup.go:20-21,102-103,106-107, startcleanup_test.go:27-28,43-44 and couch.go:498-501 describe the deleted if-!resume arm and failPostAckStart as the spawn/owning tail. Rule, 3rd time in this family: derive the grep list from what the diff deleted or re-scoped, not from the finding's examples.

### Raised

- **BR-14** [Minor] `diagnostic-scoped-to-decision` applyStartCleanup joins the session-observation error for shapes whose decision never reads presence
  couch.go:521-522 observes and joins for every shape; probe: a spawn whose acknowledge fails now also returns "exact Pair session binding is absent", which base never produced. The comments at couch.go:519-520, launch_existing.go:192-195 and warm_failure_test.go:362-364 say it explains why the thread was left occupied, yet the pinning test's warm claim rolls back. Observe only where the decider reads presence (cold claim, warm live record), and move the assertion to TestResumeUnobservableSessionKeepsUnknownOccupied.
- **BR-15** [Minor] `fail-closed-fallback-missing` The DurableRetire arm returns on a GetThread error before any disposition, so the atlas "no path" claim overclaims
  2nd finding in this family. Rule: at the live-record phase, every applyStartCleanup exit that has not retired the incarnation falls through to markLiveRecordUnknown, as one structural fallback after the switch rather than per arm. Measured prevalence: 1 remaining site, couch.go:537-540, which contradicts atlas/couch.md:685-686.

## Open findings

- **BR-2** [Minor] `missing-non-goals` No non-goals section; three adjacent behaviours are left unstated
- **BR-4** [Minor] `test-prose-enumeration` Task 1 enumerates per-row injections and assertions in prose; compress to one strategy line per risky function
- **BR-13** [Important] `stale-artifact-claim` Eight comment and doc passages still restate design decisions this issue reversed, including the atlas
- **BR-14** [Minor] `diagnostic-scoped-to-decision` applyStartCleanup joins the session-observation error for shapes whose decision never reads presence
- **BR-15** [Minor] `fail-closed-fallback-missing` The DurableRetire arm returns on a GetThread error before any disposition, so the atlas "no path" claim overclaims
