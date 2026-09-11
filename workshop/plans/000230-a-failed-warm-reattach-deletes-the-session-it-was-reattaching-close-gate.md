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

## Open findings

- **BR-1** [Minor] `unverified-existing-behavior-claim` "quiescePostAckStart is the only caller of Artifacts.Quiesce" is false: ArchiveThread calls it too (detach.go:219)
- **BR-2** [Minor] `missing-non-goals` No non-goals section; three adjacent behaviours are left unstated
- **BR-3** [Minor] `arch-dry-reuse` Routes 5-6 re-implement Detach's post-signal half; extract it
- **BR-4** [Minor] `test-prose-enumeration` Task 1 enumerates per-row injections and assertions in prose; compress to one strategy line per risky function
- **BR-5** [Important] `fail-closed-fallback-missing` A failed retire on the warm live-record routes returns with no durable fallback, leaving a live incarnation behind a dead helper
- **BR-6** [Important] `unconsumed-single-source` StartCleanup.Quiesce is never read by production; the quiesce half is decided at three call sites
- **BR-7** [Important] `plan-table-vs-code-drift` Core concepts tables name applyStartCleanup, which was never built, and classify an IO helper as PURE
- **BR-8** [Important] `stale-artifact-claim` The issue's Revisions still assert StartResult.Warm, a sole Quiesce caller, and a 24-row table
- **BR-9** [Minor] `dead-branch` The DurableRetire arm of failTrackedPostAckStart is unreachable and duplicates couch.go's retire block
- **BR-10** [Minor] `weak-assertion` Route 3's warm assertion is weaker than the plan promised
- **BR-11** [Minor] `shape-label-mismatch` AbortStarted labels a spawn StartColdResume, and rescans the registry it just searched
