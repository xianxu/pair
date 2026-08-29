---
gate: boundary-review
issue: 156
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-08-29T13:33:12-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Proofless binding migration is unreachable in production
          detail: ProofMigrator.Request has only test callers, while QuerySession merely reports migration pending and returns provisional forever. Wire one-root validation and durable proof publication into the production owner lifecycle, with an integration regression that fails without the wiring.
          family: proofless-binding-migration
          round: 1
        - id: BR-2
          severity: Critical
          title: Persisted raw launch boundaries are ignored by watcher selection
          detail: TargetArtifactBoundary retains only path fields, and no production code consumes RawSize or reads the B-1 boundary byte. The newline/incomplete-record contract and its positive watcher regressions are therefore absent.
          family: launch-boundary-framing
          round: 1
        - id: BR-3
          severity: Critical
          title: A stale same-key catalog writer can overwrite newer cursor or disputed state
          detail: persistTrackedCatalog unconditionally assigns each incoming entry over the locked current entry. Serialization prevents byte corruption but does not prevent an older validator from publishing after and replacing a newer cursor, scanner state, or disputed transition.
          family: catalog-monotonic-publication
          round: 1
        - id: BR-4
          severity: Critical
          title: The planned catalog-backed IncrementalInventory production seam does not exist
          detail: ReconcileCatalog is called only by tests, there is no IncrementalInventory entity, and launch, watcher, and query use parallel policies. This contradicts the Core concepts table and violates ARCH-DRY and ARCH-PURPOSE.
          family: catalog-authority-single-source
          round: 1
        - id: BR-5
          severity: Critical
          title: The v1 watcher compatibility path still scans the complete corpus
          detail: sessionwatch.Run invokes InventoryWithRuntime and NativeEventsWithRuntime on every v1 poll, and the shadow test explicitly allowlists both calls. An in-place upgrade therefore retains the latency behavior the Spec prohibits on interactive paths.
          family: latency-path-whole-scan
          round: 1
        - id: BR-6
          severity: Important
          title: Live provider conformance does not compare installed behavior with the stateful fake
          detail: The live test succeeds after finding one whole-file sample accepted by the validator. It does not compare normalized transitions with FakeRuntime or exercise the append-only behavior Pair relies on, leaving ARCH-MOCK drift undetected.
          family: external-conformance-cadence
          round: 1
        - id: BR-7
          severity: Important
          title: Task 2 remains wholly unchecked at the claimed whole-issue boundary
          detail: Eleven ProviderContract, reconciliation, concept-contract, verification, and commit steps remain unchecked in the durable plan. Reconcile the checklist after the blocking implementation gaps are addressed.
          family: plan-checklist-integrity
          round: 1
      blocked: true
---

# Gate ledger — pair#156 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-29T13:33:12-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `proofless-binding-migration` Proofless binding migration is unreachable in production
  ProofMigrator.Request has only test callers, while QuerySession merely reports migration pending and returns provisional forever. Wire one-root validation and durable proof publication into the production owner lifecycle, with an integration regression that fails without the wiring.
- **BR-2** [Critical] `launch-boundary-framing` Persisted raw launch boundaries are ignored by watcher selection
  TargetArtifactBoundary retains only path fields, and no production code consumes RawSize or reads the B-1 boundary byte. The newline/incomplete-record contract and its positive watcher regressions are therefore absent.
- **BR-3** [Critical] `catalog-monotonic-publication` A stale same-key catalog writer can overwrite newer cursor or disputed state
  persistTrackedCatalog unconditionally assigns each incoming entry over the locked current entry. Serialization prevents byte corruption but does not prevent an older validator from publishing after and replacing a newer cursor, scanner state, or disputed transition.
- **BR-4** [Critical] `catalog-authority-single-source` The planned catalog-backed IncrementalInventory production seam does not exist
  ReconcileCatalog is called only by tests, there is no IncrementalInventory entity, and launch, watcher, and query use parallel policies. This contradicts the Core concepts table and violates ARCH-DRY and ARCH-PURPOSE.
- **BR-5** [Critical] `latency-path-whole-scan` The v1 watcher compatibility path still scans the complete corpus
  sessionwatch.Run invokes InventoryWithRuntime and NativeEventsWithRuntime on every v1 poll, and the shadow test explicitly allowlists both calls. An in-place upgrade therefore retains the latency behavior the Spec prohibits on interactive paths.
- **BR-6** [Important] `external-conformance-cadence` Live provider conformance does not compare installed behavior with the stateful fake
  The live test succeeds after finding one whole-file sample accepted by the validator. It does not compare normalized transitions with FakeRuntime or exercise the append-only behavior Pair relies on, leaving ARCH-MOCK drift undetected.
- **BR-7** [Important] `plan-checklist-integrity` Task 2 remains wholly unchecked at the claimed whole-issue boundary
  Eleven ProviderContract, reconciliation, concept-contract, verification, and commit steps remain unchecked in the durable plan. Reconcile the checklist after the blocking implementation gaps are addressed.

## Open findings

- **BR-1** [Critical] `proofless-binding-migration` Proofless binding migration is unreachable in production
- **BR-2** [Critical] `launch-boundary-framing` Persisted raw launch boundaries are ignored by watcher selection
- **BR-3** [Critical] `catalog-monotonic-publication` A stale same-key catalog writer can overwrite newer cursor or disputed state
- **BR-4** [Critical] `catalog-authority-single-source` The planned catalog-backed IncrementalInventory production seam does not exist
- **BR-5** [Critical] `latency-path-whole-scan` The v1 watcher compatibility path still scans the complete corpus
- **BR-6** [Important] `external-conformance-cadence` Live provider conformance does not compare installed behavior with the stateful fake
- **BR-7** [Important] `plan-checklist-integrity` Task 2 remains wholly unchecked at the claimed whole-issue boundary
