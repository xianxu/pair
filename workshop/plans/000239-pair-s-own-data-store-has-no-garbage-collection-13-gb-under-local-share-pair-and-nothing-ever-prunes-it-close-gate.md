---
gate: boundary-review
issue: 239
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-14T22:22:59-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Expired metadata-only owners abort collection
          detail: collector.go:228 emits eligible empty session groups, while transaction.go:193 rejects them. Add coordinated metadata-only retirement and a regression covering capture cleanup followed by sixty-day expiry (ARCH-FUNERAL, ARCH-PURPOSE).
          family: metadata-only-retirement
          round: 1
        - id: BR-2
          severity: Critical
          title: Crashed metadata writes leave temporary files that block recovery
          detail: storagegc/stateio.go:41 relies on deferred temporary-file removal, but collector.go:103 and transaction recovery interpret leftovers as authoritative records. Recover unpublished residue under coordination and test process death before rename (ARCH-ORDER, ARCH-FUNERAL).
          family: interrupted-publication-recovery
          round: 1
        - id: BR-3
          severity: Critical
          title: Abandoned startup reservations never retire
          detail: storagegc/use.go:60 never reconciles Starts; even confirmed pre-spawn parent death permanently blocks collection and eventually exhausts start.go:50's reservation cap. Add evidence-based recovery with live and unknown child protection (ARCH-ORDER, ARCH-FUNERAL).
          family: startup-reservation-reconciliation
          round: 1
        - id: BR-4
          severity: Critical
          title: Pre-upgrade Couch archives cannot enter retention grace
          detail: couchcore/retention.go:78 turns absent legacy grace into permanent ClockError evidence, and apply never initializes archive clocks. Journal a full onboarding grace for missing legacy clocks while retaining malformed evidence (ARCH-PURPOSE, ARCH-FUNERAL).
          family: legacy-retention-onboarding
          round: 1
        - id: BR-5
          severity: Critical
          title: The completed plan claims a transaction reducer that does not exist
          detail: The plan at lines 39 and 133 promises pure transition coverage, but transaction.go:403 directly mutates phases inside I/O. Implement the enforced pure state/event model and reconcile the Core concepts table and phase enumeration (ARCH-PURE, ARCH-ORDER).
          family: enforced-pure-transitions
          round: 1
        - id: BR-6
          severity: Important
          title: Scheduled collection limits deletions but processes every owner under the shared lock
          detail: collector.go:342 performs full snapshots and owner rewrites regardless of batch limit; gcruntime/schedule.go:58 has no owner continuation cursor. Enforce the declared scheduling budget, nonblocking acquisition, and cancellation between effects with production-batch tests (ARCH-CONSTRAINTS).
          family: bounded-maintenance-work
          round: 1
      boundary: M1
      blocked: true
    - "n": 2
      timestamp: "2026-09-14T22:53:48-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: TestSessionRetirementLeavesYoungCaptureDiscoverableUntilSevenDays covers capture cleanup followed by metadata expiry. Restoring empty-session rejection makes its metadata-retirement assertion fail.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Central staging and coordinated cleanup handle unpublished JSON. TestInterruptedMetadataPublisherProcess exercises killed publishers with complete and partial writes; restoring destination-local staging makes the regression fail.
          round: 2
        - id: BR-3
          disposition: addressed
          note: Recovery and admission reclaim confirmed-dead pre-spawn reservations while preserving live, unknown and spawned evidence. Disabling recovery makes TestRecoverDeadUnspawnedStartsPreservesUnknownAndSpawned fail.
          round: 2
        - id: BR-4
          disposition: not-addressed
          note: storagegc/inventory.go:175 only merges known owners into physically discovered namespaces. A legacy Couch archive without its Pair repos/<scope> directory is omitted from Apply, so collector.go:475 never onboards it. TestReviewLegacyArchiveWithoutPairScope reproduces this on the pinned head.
          round: 2
        - id: BR-5
          disposition: addressed
          note: Production phase advancement calls ReduceTransaction through the persistence adapter. The phase/event matrix, sequence tests and bypass guard pass; a regressive retirement transition makes the matrix fail. The revised plan names the implemented phases and symbols.
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: 'gcruntime/schedule.go:64 still invokes a full owner preview for every diagnostic page: a limit-2 regression probes all 8 owners. Recovery also reaches blocking Couch flock through retention.go:392, ignoring an expired maintenance context while holding the root lock. Both scratch regressions fail.'
          round: 2
      findings:
        - id: BR-7
          severity: Important
          title: Unpublished quarantine directories have no recovery path
          detail: storagegc/transaction.go:208 creates the unique quarantine directory before publishing its journal at line 222. Publication failure or cancellation leaves it unreachable by journal-only recovery at line 586; three injected publication failures leave three directories. This is the 2nd finding in family interrupted-publication-recovery. Define and enforce recovery for every artifact created before authoritative publication, rather than fixing this instance alone.
          family: interrupted-publication-recovery
          round: 2
      boundary: M1
      blocked: true
---

# Gate ledger — 000239-pair-s-own-data-store-has-no-garbage-collection-13-gb-under-local-share-pair-and-nothing-ever-prunes-it#239 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-14T22:22:59-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `metadata-only-retirement` Expired metadata-only owners abort collection
  collector.go:228 emits eligible empty session groups, while transaction.go:193 rejects them. Add coordinated metadata-only retirement and a regression covering capture cleanup followed by sixty-day expiry (ARCH-FUNERAL, ARCH-PURPOSE).
- **BR-2** [Critical] `interrupted-publication-recovery` Crashed metadata writes leave temporary files that block recovery
  storagegc/stateio.go:41 relies on deferred temporary-file removal, but collector.go:103 and transaction recovery interpret leftovers as authoritative records. Recover unpublished residue under coordination and test process death before rename (ARCH-ORDER, ARCH-FUNERAL).
- **BR-3** [Critical] `startup-reservation-reconciliation` Abandoned startup reservations never retire
  storagegc/use.go:60 never reconciles Starts; even confirmed pre-spawn parent death permanently blocks collection and eventually exhausts start.go:50's reservation cap. Add evidence-based recovery with live and unknown child protection (ARCH-ORDER, ARCH-FUNERAL).
- **BR-4** [Critical] `legacy-retention-onboarding` Pre-upgrade Couch archives cannot enter retention grace
  couchcore/retention.go:78 turns absent legacy grace into permanent ClockError evidence, and apply never initializes archive clocks. Journal a full onboarding grace for missing legacy clocks while retaining malformed evidence (ARCH-PURPOSE, ARCH-FUNERAL).
- **BR-5** [Critical] `enforced-pure-transitions` The completed plan claims a transaction reducer that does not exist
  The plan at lines 39 and 133 promises pure transition coverage, but transaction.go:403 directly mutates phases inside I/O. Implement the enforced pure state/event model and reconcile the Core concepts table and phase enumeration (ARCH-PURE, ARCH-ORDER).
- **BR-6** [Important] `bounded-maintenance-work` Scheduled collection limits deletions but processes every owner under the shared lock
  collector.go:342 performs full snapshots and owner rewrites regardless of batch limit; gcruntime/schedule.go:58 has no owner continuation cursor. Enforce the declared scheduling budget, nonblocking acquisition, and cancellation between effects with production-batch tests (ARCH-CONSTRAINTS).

## Round 2 — 2026-09-14T22:53:48-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — TestSessionRetirementLeavesYoungCaptureDiscoverableUntilSevenDays covers capture cleanup followed by metadata expiry. Restoring empty-session rejection makes its metadata-retirement assertion fail.
- BR-2 — addressed — Central staging and coordinated cleanup handle unpublished JSON. TestInterruptedMetadataPublisherProcess exercises killed publishers with complete and partial writes; restoring destination-local staging makes the regression fail.
- BR-3 — addressed — Recovery and admission reclaim confirmed-dead pre-spawn reservations while preserving live, unknown and spawned evidence. Disabling recovery makes TestRecoverDeadUnspawnedStartsPreservesUnknownAndSpawned fail.
- BR-4 — not-addressed — storagegc/inventory.go:175 only merges known owners into physically discovered namespaces. A legacy Couch archive without its Pair repos/<scope> directory is omitted from Apply, so collector.go:475 never onboards it. TestReviewLegacyArchiveWithoutPairScope reproduces this on the pinned head.
- BR-5 — addressed — Production phase advancement calls ReduceTransaction through the persistence adapter. The phase/event matrix, sequence tests and bypass guard pass; a regressive retirement transition makes the matrix fail. The revised plan names the implemented phases and symbols.
- BR-6 — not-addressed — gcruntime/schedule.go:64 still invokes a full owner preview for every diagnostic page: a limit-2 regression probes all 8 owners. Recovery also reaches blocking Couch flock through retention.go:392, ignoring an expired maintenance context while holding the root lock. Both scratch regressions fail.

### Raised

- **BR-7** [Important] `interrupted-publication-recovery` Unpublished quarantine directories have no recovery path
  storagegc/transaction.go:208 creates the unique quarantine directory before publishing its journal at line 222. Publication failure or cancellation leaves it unreachable by journal-only recovery at line 586; three injected publication failures leave three directories. This is the 2nd finding in family interrupted-publication-recovery. Define and enforce recovery for every artifact created before authoritative publication, rather than fixing this instance alone.

## Open findings

- **BR-4** [Critical] `legacy-retention-onboarding` Pre-upgrade Couch archives cannot enter retention grace
- **BR-6** [Important] `bounded-maintenance-work` Scheduled collection limits deletions but processes every owner under the shared lock
- **BR-7** [Important] `interrupted-publication-recovery` Unpublished quarantine directories have no recovery path
