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
    - "n": 2
      timestamp: "2026-08-29T13:57:42-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: The worker is called by session-watch, but no production lifecycle restarts that watcher for an already-bound v1 owner after upgrade; the regression invokes Run directly.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: ObserveLaunchBoundarySuffix has only test callers, so production still never reads B-1.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: A higher raw offset with a lower parser-complete offset is accepted and can regress the parser cursor.
          round: 2
        - id: BR-4
          disposition: addressed
          note: IncrementalInventory now exists and production launch, watcher, query, and launcher paths call the shared reconciliation/selection seam.
          round: 2
        - id: BR-5
          disposition: addressed
          note: The v1 unbound watcher fails closed without inventory or event corpus scans, with a production-boundary regression.
          round: 2
        - id: BR-6
          disposition: not-addressed
          note: Stateful append comparison covers Claude, Codex, and Muse, but Agy still validates one copied whole-file sample without a fake transition comparison.
          round: 2
        - id: BR-7
          disposition: addressed
          note: Task 2 is fully checked and the corrective work is recorded in a dated Revisions entry.
          round: 2
      findings:
        - id: BR-8
          severity: Critical
          title: Catalog publication failure still writes a proofless v2 binding
          detail: Run clears the proof map after catalog failure, but ObserveAndPersist falls back to AppendBindingIfCurrent and terminates with proofless authority. This is the 2nd finding in family proofless-binding-migration; enforce the class rule that v2 never publishes a binding without its proof.
          family: proofless-binding-migration
          round: 2
        - id: BR-9
          severity: Critical
          title: Concurrent catalog publication can hide a genuinely post-launch artifact
          detail: TargetNewLaunch first retains only CatalogWorkNew entries, so an artifact absent from this launch baseline becomes ineligible if another writer catalogs it before the watcher observes it. This is the 2nd finding in family catalog-authority-single-source; define launch newness solely from the durable launch boundary.
          family: catalog-authority-single-source
          round: 2
        - id: BR-10
          severity: Important
          title: README documentation is missing for the conformance target
          detail: The range adds the operator-runnable make test-session-inventory-conformance surface, but README.md is unchanged.
          family: user-facing-surface-documentation
          round: 2
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

## Round 2 — 2026-08-29T13:57:42-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — not-addressed — The worker is called by session-watch, but no production lifecycle restarts that watcher for an already-bound v1 owner after upgrade; the regression invokes Run directly.
- BR-2 — not-addressed — ObserveLaunchBoundarySuffix has only test callers, so production still never reads B-1.
- BR-3 — not-addressed — A higher raw offset with a lower parser-complete offset is accepted and can regress the parser cursor.
- BR-4 — addressed — IncrementalInventory now exists and production launch, watcher, query, and launcher paths call the shared reconciliation/selection seam.
- BR-5 — addressed — The v1 unbound watcher fails closed without inventory or event corpus scans, with a production-boundary regression.
- BR-6 — not-addressed — Stateful append comparison covers Claude, Codex, and Muse, but Agy still validates one copied whole-file sample without a fake transition comparison.
- BR-7 — addressed — Task 2 is fully checked and the corrective work is recorded in a dated Revisions entry.

### Raised

- **BR-8** [Critical] `proofless-binding-migration` Catalog publication failure still writes a proofless v2 binding
  Run clears the proof map after catalog failure, but ObserveAndPersist falls back to AppendBindingIfCurrent and terminates with proofless authority. This is the 2nd finding in family proofless-binding-migration; enforce the class rule that v2 never publishes a binding without its proof.
- **BR-9** [Critical] `catalog-authority-single-source` Concurrent catalog publication can hide a genuinely post-launch artifact
  TargetNewLaunch first retains only CatalogWorkNew entries, so an artifact absent from this launch baseline becomes ineligible if another writer catalogs it before the watcher observes it. This is the 2nd finding in family catalog-authority-single-source; define launch newness solely from the durable launch boundary.
- **BR-10** [Important] `user-facing-surface-documentation` README documentation is missing for the conformance target
  The range adds the operator-runnable make test-session-inventory-conformance surface, but README.md is unchanged.

## Open findings

- **BR-1** [Critical] `proofless-binding-migration` Proofless binding migration is unreachable in production
- **BR-2** [Critical] `launch-boundary-framing` Persisted raw launch boundaries are ignored by watcher selection
- **BR-3** [Critical] `catalog-monotonic-publication` A stale same-key catalog writer can overwrite newer cursor or disputed state
- **BR-6** [Important] `external-conformance-cadence` Live provider conformance does not compare installed behavior with the stateful fake
- **BR-8** [Critical] `proofless-binding-migration` Catalog publication failure still writes a proofless v2 binding
- **BR-9** [Critical] `catalog-authority-single-source` Concurrent catalog publication can hide a genuinely post-launch artifact
- **BR-10** [Important] `user-facing-surface-documentation` README documentation is missing for the conformance target
