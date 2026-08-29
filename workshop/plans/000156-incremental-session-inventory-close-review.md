# Boundary Review — pair#156 (whole-issue close)

| field | value |
|-------|-------|
| issue | 156 — incremental native session inventory |
| repo | pair |
| issue file | workshop/issues/000156-incremental-session-inventory.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6cd38d381c9e0c4fc146284ca7b4c2e7707e7218..a13897ec17b07d52043f2b03181bd0ea9a948c5e |
| command | sdlc close --issue 156 |
| reviewer | codex |
| timestamp | 2026-08-29T13:33:12-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation has strong low-level primitives and all executed suites pass, but several central Spec requirements are unreachable or test-only. Most importantly, proofless migration is never invoked, launch-boundary offsets are recorded but ignored, stale catalog writers can overwrite newer state, and the promised catalog-backed production façade does not exist. These are blocking ARCH-PURPOSE correctness gaps.

### 1. Strengths

- `CatalogStore` has a clean locked/atomic publication boundary and tests write, sync, rename, directory-sync, and unlock outcomes ([catalog_store.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/catalog_store.go:61)).
- Platform metadata uses filesystem APIs rather than a per-file `stat` subprocess.
- Scanner validation consumes records through a stable observed EOF and rejects replacement/truncation ([incremental_inventory.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/incremental_inventory.go:295)).
- Alt+X confirmation is isolated into a small local-only module and tested before subprocess execution ([confirm_quit.lua](/Users/xianxu/workspace/pair/nvim/confirm_quit.lua:18)).
- Authorization-proof parsing is strict, rejects duplicate artifacts, and validates complete offsets ([record.go](/Users/xianxu/workspace/pair/cmd/internal/sessionledger/record.go:400)).

### 2. Critical findings

- [proof_migration.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/proof_migration.go:39): `ProofMigrator.Request` has no production caller. [query.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/query.go:96) only reports that migration is pending, permanently leaving legacy bindings unavailable. Wire targeted validation and proof publication into the owner lookup lifecycle, with an integration test that fails if no durable proof appears.

- [target.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/target.go:15): `TargetArtifactBoundary` discards stable ID, generation, mutation, and `RawSize`; [run.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/run.go:204) copies only path fields. No production code uses `RawSize` or reads `B-1`, so the specified newline/incomplete-record launch-boundary framing is absent. Add the boundary algorithm and positive newline/incomplete/truncation/replacement watcher tests.

- [run.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/run.go:181): catalog merging unconditionally overwrites an existing same-key entry with the caller’s possibly stale state. Two serialized writers can therefore publish a lower cursor or erase a newer disputed transition. Recompute/merge monotonically against the locked current entry and test same-key writers completing newest-first and oldest-last.

- [incremental_inventory.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/incremental_inventory.go:29): the plan’s `IncrementalInventory` façade does not exist, and `ReconcileCatalog` has no production caller. Launch, watcher, and query each implement parallel metadata/proof logic; the performance test invokes reconciliation manually. Route production consumers through the catalog/reconciliation seam, eliminating the duplicated continuity policy. This violates ARCH-DRY and ARCH-PURPOSE and contradicts the Core concepts table.

- [run.go](/Users/xianxu/workspace/pair/cmd/internal/sessionwatch/run.go:101): an upgraded watcher handling a v1 launch still performs whole-corpus inventory and event scans every poll. [shadow_test.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/shadow_test.go:22) explicitly allowlists these interactive calls despite the plan limiting exceptions to diagnostic/conformance entry points. Replace this compatibility path with bounded migration or fail it closed without scanning.

### 3. Important findings

- [provider_live_test.go](/Users/xianxu/workspace/pair/cmd/internal/sessioninventory/provider_live_test.go:36): the live “conformance” check accepts one recognizable installed sample; it neither compares normalized behavior with `FakeRuntime` nor observes append behavior. Connect live observations to the stateful fake’s modeled transitions and fail on behavioral divergence. ARCH-MOCK remains incomplete.

- [000156 plan](/Users/xianxu/workspace/pair/workshop/plans/000156-incremental-session-inventory-plan.md:169): all eleven Task 2 steps remain unchecked, although the issue claims full completion. Resolve the checklist honestly after the implementation gaps are fixed.

### 4. Minor findings

None.

### 5. Test coverage notes

Passed at pinned HEAD:

- `go test -p 20 ./... -count=1`
- Focused `go test -race -p 20 ...`
- `go vet -p 20 ./...`
- `make test-lua`
- Relevant terminal and watcher shell suites
- `make test-session-inventory-conformance`
- `zellij --config-dir zellij setup --check`
- `git diff --check`

The missing regressions correspond directly to the critical findings: reachable proof migration, positive launch-boundary framing, stale same-key catalog writers, production warm-catalog reuse, and removal of the v1 watcher scan allowlist.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — flag:** continuity and selection policy is repeated instead of flowing through `ReconcileCatalog`.
- **ARCH-PURE — pass:** reconciliation, selection, scanner transitions, and metadata merge remain pure; IO is behind injected runtimes.
- **ARCH-PURPOSE — flag:** the persistent catalog, proofless migration, and launch-boundary contracts are not fully delivered.
- **ARCH-MOCK — flag:** the stateful fake is useful, but the promised live behavioral comparison is missing.

### 7. Plan revision recommendations

Append a dated `## Revisions` entry recording:

- the missing production `IncrementalInventory` façade and corrective consumer routing;
- proofless migration wiring and durable publication;
- implementation of raw launch-boundary framing;
- monotonic same-key catalog publication;
- removal or explicit redesign of the v1 whole-scan adapter;
- corrected file lists, including `Makefile.local`, and completion state for Task 2.

```findings
findings:
  - id: new
    severity: Critical
    family: proofless-binding-migration
    title: |
      Proofless binding migration is unreachable in production
    detail: |
      ProofMigrator.Request has only test callers, while QuerySession merely reports migration pending and returns provisional forever. Wire one-root validation and durable proof publication into the production owner lifecycle, with an integration regression that fails without the wiring.
  - id: new
    severity: Critical
    family: launch-boundary-framing
    title: |
      Persisted raw launch boundaries are ignored by watcher selection
    detail: |
      TargetArtifactBoundary retains only path fields, and no production code consumes RawSize or reads the B-1 boundary byte. The newline/incomplete-record contract and its positive watcher regressions are therefore absent.
  - id: new
    severity: Critical
    family: catalog-monotonic-publication
    title: |
      A stale same-key catalog writer can overwrite newer cursor or disputed state
    detail: |
      persistTrackedCatalog unconditionally assigns each incoming entry over the locked current entry. Serialization prevents byte corruption but does not prevent an older validator from publishing after and replacing a newer cursor, scanner state, or disputed transition.
  - id: new
    severity: Critical
    family: catalog-authority-single-source
    title: |
      The planned catalog-backed IncrementalInventory production seam does not exist
    detail: |
      ReconcileCatalog is called only by tests, there is no IncrementalInventory entity, and launch, watcher, and query use parallel policies. This contradicts the Core concepts table and violates ARCH-DRY and ARCH-PURPOSE.
  - id: new
    severity: Critical
    family: latency-path-whole-scan
    title: |
      The v1 watcher compatibility path still scans the complete corpus
    detail: |
      sessionwatch.Run invokes InventoryWithRuntime and NativeEventsWithRuntime on every v1 poll, and the shadow test explicitly allowlists both calls. An in-place upgrade therefore retains the latency behavior the Spec prohibits on interactive paths.
  - id: new
    severity: Important
    family: external-conformance-cadence
    title: |
      Live provider conformance does not compare installed behavior with the stateful fake
    detail: |
      The live test succeeds after finding one whole-file sample accepted by the validator. It does not compare normalized transitions with FakeRuntime or exercise the append-only behavior Pair relies on, leaving ARCH-MOCK drift undetected.
  - id: new
    severity: Important
    family: plan-checklist-integrity
    title: |
      Task 2 remains wholly unchecked at the claimed whole-issue boundary
    detail: |
      Eleven ProviderContract, reconciliation, concept-contract, verification, and commit steps remain unchecked in the durable plan. Reconcile the checklist after the blocking implementation gaps are addressed.
```
