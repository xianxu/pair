# Incremental Native Session Inventory Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make native-session launch, watcher, activity, and picker paths reuse durable classifications and inspect only metadata or newly appended bytes, while rendering confirmation immediately and preserving picker display metadata.

**Architecture:** Extend the existing `sessioninventory` boundary with a versioned artifact catalog whose reconciliation is pure and whose filesystem/persistence shell is injected. Launch records capture metadata-only exclusions, binding records carry durable authorization proofs, and watcher/query consumers request targeted deltas instead of reconstructing every native forest. Agent scanners remain the sole source of native facts; incremental adapters preserve their validated state and fail closed on identity, generation, schema, or unsupported mutation changes.

**Tech Stack:** Go 1.24, `golang.org/x/sys/unix`, JSON/JSONL, SQLite CLI behind the existing runtime seam, Neovim Lua, shell integration tests.

---

## Non-goals

- Pair will not prove append-only behavior by hashing or rereading categorized
  prefixes on interactive paths; explicit deep/live conformance owns provider
  corruption detection.
- Linux birth time is not promoted to file generation. Platforms without a
  real generation primitive report it unavailable and take the spec's
  fail-closed mutation path.
- This change does not add filesystem notifications, a background whole-corpus
  indexer, or a new rendered inventory format.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ArtifactFingerprint` | `cmd/internal/sessioninventory/catalog.go` | new |
| `CatalogEntry` / `Catalog` | `cmd/internal/sessioninventory/catalog.go` | new |
| `ArtifactObservation` / `CatalogDelta` | `cmd/internal/sessioninventory/reconcile.go` | new |
| `ReconcileCatalog` | `cmd/internal/sessioninventory/reconcile.go` | new |
| `ScannerState` / `IncrementalResult` | `cmd/internal/sessioninventory/scanner_state.go` | new |
| `ProviderContract` | `cmd/internal/sessioninventory/provider_contract.go` | new |
| `LaunchArtifactBoundary` | `cmd/internal/sessionledger/record.go` | new |
| `AuthorizationProof` / `ArtifactProof` | `cmd/internal/sessionledger/record.go` | new |
| `TargetRequest` / `TargetResult` | `cmd/internal/sessioninventory/target.go` | new |
| `MergeAuthorityMetadata` | `cmd/internal/launcher/ledger.go` | new |

- **`ArtifactFingerprint`** — content-free continuity tuple: stable file ID, non-reusable generation token, mutation token, size, and timestamps.
  - **Relationships:** one fingerprint per observed artifact generation; a catalog entry owns its latest fingerprint.
  - **DRY rationale:** launch, watcher, query, and durable proofs otherwise reproduce subtly different continuity checks.
  - **Future extensions:** platform-specific tokens can widen without changing reconciliation policy.
- **`CatalogEntry` / `Catalog`** — versioned durable scanner facts, raw/parser-complete offsets, authorization state, and generation number.
  - **Relationships:** one catalog per Pair data scope; one entry per `(agent, storage root, relative path)`.
  - **DRY rationale:** all consumers derive from one classified artifact source (`ARCH-DRY`, `ARCH-PURPOSE`).
  - **Future extensions:** diagnostic-only deep-validation timestamps or additional native agents.
- **`ArtifactObservation` / `CatalogDelta` / `ReconcileCatalog`** — pure comparison of filesystem metadata with prior state, producing unchanged, append, new, replace, truncate, delete, and schema-stale work.
  - **Relationships:** many observations reconcile against one catalog generation; the result owns no IO.
  - **DRY rationale:** one fail-closed mutation policy serves every scanner and consumer (`ARCH-PURE`).
  - **Future extensions:** batched directory notification inputs can feed the same observations.
- **`ScannerState` / `IncrementalResult`** — versioned agent-neutral envelope around the scanner-owned role, identity, chronology, disputed flag, first-record invariants, and accepted offsets.
  - **Relationships:** exactly one state per authorized catalog entry; agent validators own their payload.
  - **DRY rationale:** full, suffix, watcher, and proof migration validation share one transition contract.
  - **Future extensions:** new parser schemas migrate by invalidating or explicitly upgrading state.
- **`ProviderContract`** — closed, versioned allowlist of native stores whose producers promise append-only JSONL.
  - **Relationships:** every scanner schema names exactly one provider contract; reconciliation permits suffix work only when the persisted and current contract versions agree.
  - **DRY rationale:** append trust is one explicit policy rather than path-name conditionals in four scanners.
  - **Future extensions:** a provider upgrade adds a reviewed contract version and conformance fixture; unknown versions remain fail-closed.
- **`LaunchArtifactBoundary`** — metadata-only raw-size/generation exclusion captured before input.
  - **Relationships:** a launch record owns zero or more sorted boundaries; it grants no authorization.
  - **DRY rationale:** replaces event-derived baselines with the exact filesystem boundary every watcher needs.
  - **Future extensions:** database-specific opaque mutation cursors.
- **`AuthorizationProof` / `ArtifactProof`** — binding-owned durable scanner authorization and continuity tuples.
  - **Relationships:** one binding has one proof; a native root proof owns one or more artifact proofs (two for Agy).
  - **DRY rationale:** catalog loss does not force historical corpus reconstruction or create a second authority rule.
  - **Future extensions:** explicit proof upgrade/version migration.
- **`TargetRequest` / `TargetResult`** — a pure description/result for established, explicit-resume, new-launch, activity, or diagnostic access.
  - **Relationships:** each latency-sensitive consumer submits one bounded request; results contain only its authorized root and events.
  - **DRY rationale:** prevents consumers from reaching for whole-agent inventory scans.
  - **Future extensions:** bounded multi-root diagnostic requests.
- **`MergeAuthorityMetadata`** — overlays newest compatibility display fields onto typed authority without changing root selection.
  - **Relationships:** many compatibility rows may enrich one current typed owner row.
  - **DRY rationale:** picker/history consumers share one precedence rule.
  - **Future extensions:** additional non-authoritative presentation fields.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Runtime` metadata operations | `cmd/internal/sessioninventory/runtime.go` | modified | filesystem metadata, bounded reads, SQLite, process evidence |
| `OSRuntime` native metadata | `cmd/internal/sessioninventory/runtime_os.go`, `filemeta_*.go` | modified | platform filesystem syscalls without per-file subprocesses |
| `CatalogStore` | `cmd/internal/sessioninventory/catalog_store.go`, `catalog_store_unix.go` | new | locked, atomic, synced catalog generations |
| `CatalogCommitOutcome` | `cmd/internal/sessioninventory/catalog_store.go` | new | explicit non-authoritative, indeterminate, or committed publication result |
| `IncrementalInventory` | `cmd/internal/sessioninventory/incremental_inventory.go` | new | metadata enumeration plus targeted scanner reads |
| `ProofMigrator` | `cmd/internal/sessioninventory/proof_migration.go` | new | deduplicated background validation of one proofless named root |
| `FakeRuntime` | `cmd/internal/sessioninventorytest/fake_runtime.go` | modified | stateful files, generations, appends, replacements, SQLite and IO counts |
| launch/watch adapters | `cmd/internal/sessionwatch/lifecycle.go`, `run.go` | modified | ledger, catalog, Pair log, polling and process corroboration |
| Neovim confirmation | `nvim/init.lua` | modified | synchronous operator modal and optional cached text |

- **`Runtime` / `OSRuntime`** — expose one metadata operation returning the complete opaque fingerprint and preserve bounded `ReadAt`; Darwin/Linux implementations use syscalls rather than `stat(1)`.
  - **Injected into:** reconciliation shells, scanners, queries, launch, watcher, and conformance.
  - **Future extensions:** another OS adds one `filemeta_<os>.go` implementation.
- **`CatalogStore`** — locks the scope catalog, rereads generation, applies a pure mutation, writes a temporary file, syncs file/directory, renames, and unlocks.
  - **Injected into:** `IncrementalInventory`; tests use a portable temporary directory and injected failure runtime.
  - **Future extensions:** compaction remains an internal store detail.
- **`CatalogCommitOutcome`** — tells callers whether recovered bytes may authorize facts after any store failure.
  - **Injected into:** every catalog publication caller; recovery reparses the production file and agrees with the returned outcome at every fault boundary.
  - **Future extensions:** stable operation IDs can make post-publication durability retry idempotent.
- **`IncrementalInventory`** — thin orchestration façade that enumerates metadata, reconciles, dispatches only eligible reads/queries, and publishes authorized entries.
  - **Injected into:** launch, watcher, activity, recovery, and diagnostic CLI.
  - **Future extensions:** filesystem event hints may reduce enumeration without changing correctness.
- **`ProofMigrator`** — coalesces repeated requests for the same proofless binding, validates only that named root, and publishes a proof or leaves it unavailable.
  - **Injected into:** ordinary owner/activity lookups; explicit resume bypasses the queue but uses the same validator synchronously.
  - **Future extensions:** bounded retry/backoff policy without widening validation scope.
- **`FakeRuntime`** — shared stateful double used by production-boundary tests (`ARCH-MOCK`), including append-only provider state and operation counters.
  - **Injected into:** every incremental scanner/catalog integration test.
  - **Future extensions:** scripted concurrent store failures and provider-version drift.
- **Launch/watch adapters** — translate ledger owners and launch boundaries into targeted requests; they never choose a root from metadata alone.
  - **Injected into:** outer launcher and `pair-session-watch`.
  - **Future extensions:** another causal source can join at `TargetRequest`.
- **Neovim confirmation** — paints the modal before any optional enrichment; current implementation omits live activity from the critical path.
  - **Injected into:** `PairConfirmQuit` only.
  - **Future extensions:** cached/asynchronous enrichment may update a later UI, never the active modal.

### Risky function test strategies

| Function | Strategy |
|----------|----------|
| `readFileMetadata` | Feed supported/unavailable platform metadata; the guard never substitutes birth time for generation and returns opaque tokens without content IO. |
| `ProviderContractFor` | Fuzz store/schema values; only the closed version table authorizes suffix reuse. |
| `ReconcileCatalog` | Property-test arbitrary prior/observed catalogs; immutable sorted deltas and fail-closed zero states prevent accidental authority. |
| `CatalogStore.Update` | Fault-inject every publication boundary and stale writer interleaving; production-parser recovery must agree with the explicit commit outcome. |
| `ValidateAuthorizationProof` | Fuzz malformed/partial continuity tuples; strict version/root/artifact validation rejects every incomplete proof. |
| `FrameJSONLSuffix` | Fuzz arbitrary bytes and chunk boundaries, seeded with split/oversize records; bounded framing advances only complete-record offsets. |
| `ObserveStableArtifact` | Interleave growth, replacement, and truncation around metadata reads; resampling requires a stable EOF before proof publication. |
| `ValidateClaudeDelta`, `ValidateCodexDelta`, `ValidateMuseDelta`, `ValidateAgyDelta` | Replay sanitized native fixtures plus malformed schema/identity/role variants; scanner-owned state transitions fail disputed and never stop before observed EOF. |
| `SelectTargetWork` | Generate cold/proof/proofless/catalog-loss matrices for every agent; eligibility is bounded to the named or post-boundary artifact set. |
| `ProofMigrator.Request` | Interleave duplicate/failing requests; one keyed worker validates one named root and failures never widen scope. |
| `PrepareOSLaunch` / `Run` | Use the stateful runtime with operation counters and boundary mutation; launch stays metadata-only and watcher consumes only eligible deltas. |
| `MergeAuthorityMetadata` | Permute typed/compatibility rows; presentation fields merge while native authority fields remain typed-only. |

## Chunk 1: Durable incremental state

### Task 1: Platform metadata without subprocesses

**Files:**
- Modify: `cmd/internal/sessioninventory/runtime.go`
- Modify: `cmd/internal/sessioninventory/runtime_os.go`
- Create: `cmd/internal/sessioninventory/filemeta.go`
- Create: `cmd/internal/sessioninventory/filemeta_darwin.go`
- Create: `cmd/internal/sessioninventory/filemeta_linux.go`
- Create: `cmd/internal/sessioninventory/filemeta_other.go`
- Modify: `cmd/internal/sessioninventory/runtime_os_test.go`
- Modify: `cmd/internal/sessioninventorytest/fake_runtime.go`
- Modify: `cmd/internal/sessioninventorytest/fake_runtime_test.go`

- [x] **Step 1: Add failing tests for `readFileMetadata` and `FileEntry` using the risky-function strategy above.** Include the corpus-sized operation-count integration fixture without restating metadata policy in the test prose.
- [x] **Step 2: Run the red tests.** Run `go test -p 20 ./cmd/internal/sessioninventory ./cmd/internal/sessioninventorytest -run 'Test(OSRuntimeListFilesFingerprint|FakeRuntimeArtifactGeneration)' -count=1`; expect compile failures for the new fields/helpers.
- [x] **Step 3: Implement and verify the value seam.** Add value types `StableFileID`, `GenerationToken`, and `MutationToken`, extend `FileEntry`, and rerun the focused fake test; expect PASS for value preservation before touching OS code.
- [x] **Step 4: Implement Darwin metadata.** Populate stable ID from device/inode, generation from `Stat_t.Gen`, birth timestamp from `Btim`, and mutation from `Ctim` through `unix.Fstatat`; run the Darwin OSRuntime test and expect PASS with no subprocess.
- [x] **Step 5: Implement Linux/fallback metadata.** Use `unix.Statx` for ordinary metadata but always emit generation unavailable because `statx` has no inode-generation field. Other-platform implementations also remain unavailable until an equivalent primitive is proven.
- [x] **Step 6: Remove the command path and verify cross-compile.** Delete `fileBirthTime` and per-entry `exec.Command("stat", ...)`; run `gofmt -w cmd/internal/sessioninventory`, the Step 2 tests, and `GOOS=linux go test -c ./cmd/internal/sessioninventory`; expect PASS/compile success.
- [x] **Step 7: Extend and verify the stateful fake.** Model create, append, same-inode replacement, truncate, delete, metadata failure, and per-operation counts; run `go test -p 20 ./cmd/internal/sessioninventorytest -count=1`; expect PASS.
- [x] **Step 8: Commit.** Run `git add cmd/internal/sessioninventory cmd/internal/sessioninventorytest && git commit -m 'session inventory: #156 add syscall metadata fingerprints'`.

### Task 2: Pure catalog reconciliation

**Files:**
- Create: `cmd/internal/sessioninventory/catalog.go`
- Create: `cmd/internal/sessioninventory/catalog_test.go`
- Create: `cmd/internal/sessioninventory/reconcile.go`
- Create: `cmd/internal/sessioninventory/reconcile_test.go`
- Create: `cmd/internal/sessioninventory/provider_contract.go`
- Create: `cmd/internal/sessioninventory/provider_contract_test.go`
- Modify: `cmd/internal/sessioninventory/concept_contract_test.go`

- [x] **Step 1: Write failing `ProviderContractFor` tests using the risky-function strategy table.**
- [x] **Step 2: Run provider tests red.** Run `go test -p 20 ./cmd/internal/sessioninventory -run 'TestProviderContract' -count=1`; expect undefined provider-contract symbols.
- [x] **Step 3: Implement `ProviderContract`.** Add the closed store/schema version table and no path-name inference.
- [x] **Step 4: Run provider tests green.** Repeat Step 2; expect PASS.
- [x] **Step 5: Write failing `ReconcileCatalog` property/table tests using the risky-function strategy table.**
- [x] **Step 6: Run reconciliation red.** Run `go test -p 20 ./cmd/internal/sessioninventory -run 'TestReconcileCatalog' -count=1`; expect undefined catalog/reconcile symbols.
- [x] **Step 7: Implement immutable catalog values.** Reserve zero/unknown authorization values as fail-closed; add strict validation and clone/sort helpers; run `TestCatalog` and expect PASS.
- [x] **Step 8: Implement pure reconciliation.** Sort observations/deltas deterministically, never mutate caller slices/maps, and retract facts on disputed/replacement states; rerun `TestReconcileCatalog` and expect PASS.
- [x] **Step 9: Extend the concept contract.** Add `pair:156-concept` declarations for every new/modified pure entity and integration seam, and make the test derive the complete #156 declaration universe bidirectionally without weakening #155 historical pinning.
- [x] **Step 10: Run green.** Run `gofmt -w cmd/internal/sessioninventory` and `go test -p 20 ./cmd/internal/sessioninventory -run 'Test(ProviderContract|ReconcileCatalog|Catalog|Concept)' -count=1`; expect PASS.
- [x] **Step 11: Commit.** Run `git add cmd/internal/sessioninventory && git commit -m 'session inventory: #156 define incremental catalog reconciliation'`.

### Task 3: Locked catalog persistence

**Files:**
- Modify: `cmd/internal/artifactpath/paths.go`
- Modify: `cmd/internal/artifactpath/manifest.go`
- Modify: `cmd/internal/artifactpath/paths_test.go`
- Modify: `cmd/internal/artifactpath/coverage_test.go`
- Create: `cmd/internal/sessioninventory/catalog_store.go`
- Create: `cmd/internal/sessioninventory/catalog_store_unix.go`
- Create: `cmd/internal/sessioninventory/catalog_store_test.go`

- [x] **Step 1: Write the failing path test.** Pin `ScopePaths.SessionInventoryCatalog()` and exhaustive manifest classification.
- [x] **Step 2: Run the path test red.** Run `go test -p 20 ./cmd/internal/artifactpath -run 'TestSessionInventoryCatalog' -count=1`; expect the path member to be undefined.
- [x] **Step 3: Implement and verify the path.** Register the classified path, run Step 2 again, and expect PASS.
- [x] **Step 4: Write failing `CatalogStore.Update` transaction tests using the risky-function strategy table.**
- [x] **Step 5: Add its stateful fault runtime and production-parser recovery oracle.**
- [x] **Step 6: Run store tests red.** Run `go test -p 20 ./cmd/internal/sessioninventory -run 'TestCatalogStore' -count=1`; expect missing store/outcome symbols.
- [x] **Step 7: Implement strict read and transaction framing.** Reuse ledger flock/commit-outcome patterns: lock, strict-decode, compare generation, invoke a pure delta callback against the reread catalog, temp-write, sync, atomic rename, directory sync, unlock. Corruption returns a local error and no authority.
- [x] **Step 8: Implement stale-generation retry and recovery classification.** On mismatch recompute from the locked reread state; after publication errors reparse the production path and return the matching explicit outcome without duplicating a generation.
- [x] **Step 9: Run green and race coverage.** Run `gofmt -w cmd/internal/artifactpath cmd/internal/sessioninventory` then `go test -race -p 20 ./cmd/internal/artifactpath ./cmd/internal/sessioninventory -run 'Test(SessionInventoryCatalog|CatalogStore)' -count=1`; expect PASS.
- [x] **Step 10: Commit.** Run `git add cmd/internal/artifactpath cmd/internal/sessioninventory && git commit -m 'session inventory: #156 persist catalog generations atomically'`.

### Task 4: Versioned launch boundaries and binding proofs

**Files:**
- Modify: `cmd/internal/sessionledger/record.go`
- Modify: `cmd/internal/sessionledger/record_test.go`
- Modify: `cmd/internal/sessionledger/store_test.go`

- [x] **Step 1: Write v2 `ParseLedger`, `EncodeRecord`, and `ValidateAuthorizationProof` tests using the risky-function strategy table.**
- [x] **Step 2: Run red.** Run `go test -p 20 ./cmd/internal/sessionledger -run 'Test(RecordV2|AuthorizationProof|LaunchArtifactBoundary)' -count=1`; expect undefined v2 fields.
- [x] **Step 3: Implement and verify v2 launch records.** Keep v1 readable; encode metadata-only sorted artifact boundaries in new v2 launches. Run launch wire tests to PASS before adding proofs.
- [x] **Step 4: Implement and verify v2 binding proofs.** Add strict proof/state/artifact decoding and root matching; run proof tests to PASS.
- [x] **Step 5: Preserve append authority semantics.** Extend existing byte-boundary/failure tests so v2 proof publication is committed exactly when the ledger row is authoritative.
- [x] **Step 6: Run green.** Run `gofmt -w cmd/internal/sessionledger` and `go test -race -p 20 ./cmd/internal/sessionledger -count=1`; expect PASS.
- [x] **Step 7: Commit.** Run `git add cmd/internal/sessionledger && git commit -m 'session ledger: #156 persist launch boundaries and proofs'`.

## Chunk 2: Incremental authorization and consumers

### Task 5: Agent-specific full and suffix validators

**Files:**
- Create: `cmd/internal/sessioninventory/jsonl_incremental.go`
- Create: `cmd/internal/sessioninventory/jsonl_incremental_test.go`
- Create: `cmd/internal/sessioninventory/scanner_state.go`
- Create: `cmd/internal/sessioninventory/scanner_state_test.go`
- Create: `cmd/internal/sessioninventory/incremental_inventory.go`
- Create: `cmd/internal/sessioninventory/incremental_inventory_test.go`
- Modify: `cmd/internal/sessioninventory/scan_helpers.go`
- Modify: `cmd/internal/sessioninventory/scan_claude.go`
- Modify: `cmd/internal/sessioninventory/scan_claude_test.go`
- Modify: `cmd/internal/sessioninventory/scan_codex.go`
- Modify: `cmd/internal/sessioninventory/scan_codex_test.go`
- Modify: `cmd/internal/sessioninventory/scan_muse.go`
- Modify: `cmd/internal/sessioninventory/scan_muse_test.go`
- Modify: `cmd/internal/sessioninventory/scan_agy.go`
- Modify: `cmd/internal/sessioninventory/scan_agy_test.go`
- Modify: `cmd/internal/sessioninventory/events.go`
- Modify: `cmd/internal/sessioninventory/events_test.go`

- [x] **Step 1: Write failing `FrameJSONLSuffix` fuzz/property tests using the risky-function strategy table.**
- [x] **Step 2: Run framing red.** Run `go test -p 20 ./cmd/internal/sessioninventory -run 'TestIncrementalJSONL' -count=1`; expect missing framer/state APIs.
- [x] **Step 3: Implement and verify pure framing.** Add one bounded JSONL suffix framer that retains incomplete tails; rerun Step 2 and expect PASS.
- [x] **Step 4: Write and run `ObserveStableArtifact` tests red using the risky-function strategy table.** Run `go test -p 20 ./cmd/internal/sessioninventory -run 'TestObserveStableArtifact' -count=1`; expect missing orchestration.
- [x] **Step 5: Implement and verify IO resampling.** Resample fingerprint/EOF after each scan; growth loops through the new suffix, replacement/truncation disputes, and proof publication waits for one stable resample. Repeat Step 4 and expect PASS.
- [x] **Step 6: Write `ValidateClaudeDelta` tests using the risky-function strategy table.**
- [x] **Step 7: Run Claude red.** Run `go test -p 20 ./cmd/internal/sessioninventory -run 'TestIncrementalClaude' -count=1`; expect FAIL on missing state transitions.
- [x] **Step 8: Implement Claude transitions.** Adapt existing Claude parsing into scanner state without a second schema parser.
- [x] **Step 9: Run Claude green.** Repeat Step 7; expect PASS.
- [x] **Step 10: Write `ValidateCodexDelta` tests using the risky-function strategy table.**
- [x] **Step 11: Run Codex red.** Run `go test -p 20 ./cmd/internal/sessioninventory -run 'TestIncrementalCodex' -count=1`; expect FAIL.
- [x] **Step 12: Implement Codex transitions.** Adapt the existing parser into scanner state.
- [x] **Step 13: Run Codex green.** Repeat Step 11; expect PASS.
- [x] **Step 14: Write `ValidateMuseDelta` tests using the risky-function strategy table.**
- [x] **Step 15: Run Muse red.** Run `go test -p 20 ./cmd/internal/sessioninventory -run 'TestIncrementalMuse' -count=1`; expect FAIL.
- [x] **Step 16: Implement Muse transitions.** Adapt the existing parser into scanner state.
- [x] **Step 17: Run Muse green.** Repeat Step 15; expect PASS.
- [x] **Step 18: Write `ValidateAgyDelta` tests using the risky-function strategy table.**
- [x] **Step 19: Run Agy red.** Run `go test -p 20 ./cmd/internal/sessioninventory -run 'TestIncrementalAgy' -count=1`; expect FAIL.
- [x] **Step 20: Implement Agy transitions.** Adapt the existing database validator and transcript join into scanner state.
- [x] **Step 21: Run Agy green.** Repeat Step 19; expect PASS.
- [x] **Step 22: Write portable `TestAppendOnlyProviderConformance` comparisons.** Compare normalized scanner-state observations from the stateful fake and sanitized versioned fixtures; mutation tests operate only on temporary copies.
- [x] **Step 23: Run portable conformance red.** Run `go test -p 20 ./cmd/internal/sessioninventory -run '^TestAppendOnlyProviderConformance$' -count=1`; expect FAIL.
- [x] **Step 24: Implement portable conformance fixtures/checks.** Add the versioned provider fixtures and deep-validation comparison.
- [x] **Step 25: Run portable conformance green.** Repeat Step 23; expect PASS.
- [x] **Step 26: Write `TestLiveProviderContractConformance`.** Read installed Claude/Codex/Muse/Agy stores only for redacted normalized observations; copy any selected Agy SQLite database/transcript into `t.TempDir()` before mutation-based fake comparison, never mutate installed authority.
- [x] **Step 27: Run the live comparison.** Run `PAIR_LIVE_NATIVE_SESSIONS=1 go test -p 20 ./cmd/internal/sessioninventory -run '^TestLiveProviderContractConformance$' -count=1 -v`; expect PASS or an explicit provider-contract drift report without paths, IDs, or content.
- [x] **Step 28: Run green and fuzz framing.** Run `gofmt -w cmd/internal/sessioninventory` then `go test -p 20 ./cmd/internal/sessioninventory -run 'TestIncremental|TestAppendOnly|Fuzz' -count=1`; expect PASS.
- [x] **Step 29: Commit.** Run `git add cmd/internal/sessioninventory && git commit -m 'session inventory: #156 validate only targeted native deltas'`.

### Task 6: Metadata-only launch and incremental watcher

**Files:**
- Create: `cmd/internal/sessioninventory/target.go`
- Create: `cmd/internal/sessioninventory/target_test.go`
- Modify: `cmd/internal/sessionwatch/lifecycle.go`
- Modify: `cmd/internal/sessionwatch/lifecycle_test.go`
- Modify: `cmd/internal/sessionwatch/lifecycle_store_test.go`
- Modify: `cmd/internal/sessionwatch/run.go`
- Modify: `cmd/internal/sessionwatch/run_test.go`
- Modify: `cmd/internal/sessionwatch/runtime.go`

- [x] **Step 1: Write failing `PrepareOSLaunch`, `SelectTargetWork`, and `Run` tests using the risky-function strategy table.** The stateful fixture separately measures cold, warm, corrupt-catalog, and every-agent authorization paths.
- [x] **Step 2: Add the pure four-agent cold-selection generator and stateful boundary mutation seam.**
- [x] **Step 3: Add the exact-correlation and stale-launch integration oracle.**
- [x] **Step 4: Run red.** Run `go test -p 20 ./cmd/internal/sessionwatch ./cmd/internal/sessioninventory -run 'Test(PrepareOSLaunchIncremental|ColdAuthorizationMatrix|CorruptCatalogColdLaunch|WatcherIncremental|WatcherLaunchBoundary)' -count=1`; expect whole-scan operation counts or missing target APIs.
- [x] **Step 5: Implement cold launch orchestration.** Record metadata-only boundaries and no facts. Corrupt catalog state uses the same untrusted snapshot, never a corpus rebuild; verify the cold 1,573-entry operation counters independently.
- [x] **Step 6: Implement agent-specific new eligibility.** Claude/Codex/Muse accept only absent-at-baseline artifacts; Agy joins only an absent-at-baseline DB with an absent-at-baseline same-ID transcript and queries only that DB.
- [x] **Step 7: Implement watcher eligibility.** Eligibility is exactly new-after-boundary or established/explicit target with a valid proof; catalog/proof loss never expands scope.
- [x] **Step 8: Implement stable proof publication.** Resample through a stable EOF, then publish binding and complete proof in the same authoritative row; concurrent catalog writers cannot lose a cursor or disputed transition.
- [x] **Step 9: Replace the poll loop's whole scans.** Reconcile metadata, read only eligible new/suffix work, project only post-launch events, and persist catalog state after validation. Keep process corroboration and Pair-log exact-correlation semantics unchanged.
- [x] **Step 10: Run green/race.** Run `gofmt -w cmd/internal/sessioninventory cmd/internal/sessionwatch` and `go test -race -p 20 ./cmd/internal/sessioninventory ./cmd/internal/sessionwatch -count=1`; expect PASS.
- [x] **Step 11: Commit.** Run `git add cmd/internal/sessioninventory cmd/internal/sessionwatch && git commit -m 'session watch: #156 observe post-launch catalog deltas'`.

### Task 7: Target every latency-sensitive inventory consumer

**Files:**
- Modify: `cmd/internal/sessioninventory/query.go`
- Modify: `cmd/internal/sessioninventory/query_test.go`
- Modify: `cmd/internal/sessioninventory/activity.go`
- Modify: `cmd/internal/sessioninventory/activity_test.go`
- Modify: `cmd/internal/sessioninventory/pair_inventory.go`
- Modify: `cmd/internal/sessioninventory/pair_inventory_test.go`
- Modify: `cmd/internal/sessioninventory/runcli.go`
- Modify: `cmd/internal/sessioninventory/shadow_test.go`
- Create: `cmd/internal/sessioninventory/proof_migration.go`
- Create: `cmd/internal/sessioninventory/proof_migration_test.go`
- Modify: `cmd/internal/launcher/osruntime.go`
- Modify: `cmd/internal/launcher/osruntime_test.go`
- Modify: `cmd/internal/contextcmd/contextcmd.go`
- Modify: `cmd/internal/contextcmd/contextcmd_test.go`
- Modify: `cmd/internal/reviewcmd/runtime.go`
- Modify: `cmd/internal/reviewcmd/run_test.go`
- Modify: `cmd/internal/slugcmd/slugcmd.go`
- Modify: `cmd/internal/slugcmd/slugcmd_test.go`
- Modify: `cmd/internal/opener/runtime.go`
- Modify: `cmd/internal/opener/run_test.go`
- Modify: `cmd/internal/titlepoller/runtime.go`
- Modify: `cmd/internal/titlepoller/run_test.go`

- [x] **Step 1: Enumerate the consumer class in a failing cross-language shadow sweep.** Parse Go, Lua, and shell entry points; reject whole-agent scans and direct inventory subprocesses from launch, existence, owner, activity, recovery, context, review, slug, opener, title-poller, and confirmation paths. The fixed allowlist contains only named diagnostic/conformance entry points and cannot be extended by an unreviewed call-site discovery.
- [x] **Step 2: Write `ValidateAuthorizationProof` catalog-loss tests over the generated four-agent artifact class.**
- [x] **Step 3: Write `ProofMigrator.Request` lifecycle tests using the risky-function strategy table.**
- [x] **Step 4: Write `SelectTargetWork` operation-count tests for every interactive request kind.**
- [x] **Step 5: Run red.** Run `go test -p 20 ./cmd/internal/sessioninventory ./cmd/internal/launcher ./cmd/internal/contextcmd ./cmd/internal/reviewcmd ./cmd/internal/slugcmd ./cmd/internal/opener ./cmd/internal/titlepoller -run 'Test(Target|Proofless|CatalogLoss|NoWholeInventory|AgentSessionExists|EstablishedSession)' -count=1`; expect shadow violations and excess reads.
- [x] **Step 6: Implement and verify `ProofMigrator`.** Add the coalescing worker in `proof_migration.go`; run `go test -p 20 ./cmd/internal/sessioninventory -run 'TestProofMigrator' -count=1`; expect PASS.
- [x] **Step 7: Migrate launcher existence/owner/activity.** Route `launcher/osruntime.go`, `query.go`, and `activity.go` through one target request; run `go test -p 20 ./cmd/internal/launcher ./cmd/internal/sessioninventory -run 'Test(AgentSessionExists|EstablishedSession|Target|Activity)' -count=1`; expect PASS.
- [x] **Step 8: Migrate context and review.** Route their runtime seams through the same façade; run `go test -p 20 ./cmd/internal/contextcmd ./cmd/internal/reviewcmd -count=1`; expect PASS.
- [x] **Step 9: Migrate slug, opener, and title-poller.** Route their runtime seams through the same façade; run `go test -p 20 ./cmd/internal/slugcmd ./cmd/internal/opener ./cmd/internal/titlepoller -count=1`; expect PASS.
- [x] **Step 10: Run green and the exhaustive sweep.** Repeat Step 5, then run `rg -n 'InventoryWithRuntime|NativeEventsWithRuntime|session-inventory.*--activity' cmd nvim tests --glob '*.go' --glob '*.lua' --glob '*.sh'`; every remaining occurrence must already be in the fixed diagnostic/conformance allowlist. Expect no latency-sensitive direct call.
- [x] **Step 11: Commit.** Run `git add cmd/internal/sessioninventory cmd/internal/launcher cmd/internal/contextcmd cmd/internal/reviewcmd cmd/internal/slugcmd cmd/internal/opener cmd/internal/titlepoller && git commit -m 'session inventory: #156 target every interactive query'`.

## Chunk 3: Immediate UI, picker metadata, and verification

### Task 8: Paint Alt+X confirmation before enrichment

**Files:**
- Modify: `Makefile`
- Modify: `nvim/init.lua`
- Create: `nvim/confirm_quit_test.lua`
- Modify: `tests/term-pane-shortcuts-test.sh`

- [x] **Step 1: Write deterministic failing Lua cases.** Separately model indefinitely blocking, missing, and failed inventory/activity seams; invoke `PairConfirmQuit` and assert `vim.fn.confirm` is observed and actionable first with zero inventory subprocess starts. Also pin saved config prompt data and Yes/No actions.
- [x] **Step 2: Run red.** Run `nvim -l nvim/confirm_quit_test.lua`; expect the current synchronous established/activity lookup to occur before confirmation.
- [x] **Step 3: Make the modal critical path local-only.** Build prompt from env/config sidecars without calling `_G.PairEstablishedSessionID` or `pair session-inventory`; omit age/idle enrichment. Call `vim.fn.confirm` immediately after the existing visibility defer and only execute `pair quit` after Yes.
- [x] **Step 4: Register and run green.** Add `nvim -l nvim/confirm_quit_test.lua` to `make test-lua`, then run `nvim -l nvim/confirm_quit_test.lua && make test-lua && bash tests/term-pane-shortcuts-test.sh`; expect PASS.
- [x] **Step 5: Commit.** Run `git add Makefile nvim/init.lua nvim/confirm_quit_test.lua tests/term-pane-shortcuts-test.sh && git commit -m 'nvim: #156 show quit confirmation before inventory work'`.

### Task 9: Preserve picker display metadata under typed authority

**Files:**
- Modify: `cmd/internal/launcher/ledger.go`
- Modify: `cmd/internal/launcher/ledger_test.go`
- Modify: `cmd/internal/launcher/history.go`
- Modify: `cmd/internal/launcher/history_test.go`
- Modify: `cmd/internal/launcher/pick_test.go`

- [x] **Step 1: Write the observed picker regression first.** Given a compatibility row for tag `1` with repo `pair`, followed by typed launch/binding authority, assert one row displays `pair/1 claude`, keeps args/timestamps/root for presentation, and uses only the typed binding's native session ID.
- [x] **Step 2: Add precedence negatives.** Newer compatibility metadata may enrich display but may not replace typed root authority; metadata from another owner/agent/tag cannot leak; a typed unbound launch remains unbound even if compatibility has a session ID.
- [x] **Step 3: Run red.** Run `go test -p 20 ./cmd/internal/launcher -run 'Test.*Typed.*Metadata|Test.*PairSlashOne' -count=1`; expect `RepoName == ""` / `?/1`.
- [x] **Step 4: Implement `MergeAuthorityMetadata`.** Group compatibility rows by the same history owner/tag context and choose presentation fields by compatibility chronology. Compatibility may supply only `RepoName`, `RepoRoot`, `Started`, `LastActive`, and saved `Args`; it never supplies native `SessionID`, root-node identity, proof, authorization status, or typed source ordinal.
- [x] **Step 5: Run green.** Run `gofmt -w cmd/internal/launcher` and `go test -p 20 ./cmd/internal/launcher -run 'Test.*Ledger|Test.*History|Test.*Pick' -count=1`; expect PASS.
- [x] **Step 6: Commit.** Run `git add cmd/internal/launcher && git commit -m 'launcher: #156 preserve typed picker display metadata'`.

### Task 10: Performance, conformance, documentation, and full verification

**Files:**
- Modify: `Makefile`
- Create: `cmd/internal/sessioninventory/performance_test.go`
- Modify: `cmd/internal/sessioninventory/conformance_live_test.go`
- Modify: `cmd/internal/sessioninventory/runcli_test.go`
- Modify: `atlas/session-identity.md`
- Modify: `atlas/architecture.md`
- Modify: `atlas/index.md`
- Modify: `workshop/issues/000156-incremental-session-inventory.md`
- Modify: `workshop/plans/000156-incremental-session-inventory-plan.md`

- [x] **Step 1: Add `TestIncrementalPrelaunchOperationBudget`.** The generated 1,573-entry cold/warm stateful fixture mechanically guards linear metadata, zero body/SQLite/process work, and one-time categorization.
- [x] **Step 2: Add opt-in real-data and provider-contract conformance.** `TestLiveIncrementalInventoryPrelaunch` measures the one-second contract. `TestLiveProviderContractConformance` compares redacted normalized facts from installed stores with the fake; mutation-based Agy checks use temporary copies only (`ARCH-MOCK`).
- [x] **Step 3: Run focused verification.** Run `go test -race -p 20 ./cmd/internal/sessioninventory ./cmd/internal/sessioninventorytest ./cmd/internal/sessionledger ./cmd/internal/sessionwatch ./cmd/internal/launcher -count=1`; expect PASS.
- [x] **Step 4: Run product integration verification.** Run `make test-lua && bash tests/term-pane-shortcuts-test.sh && bash tests/pair-session-watch-test.sh`; expect PASS.
- [x] **Step 5: Run both operator live checks.** Run `PAIR_LIVE_SESSION_INVENTORY=1 go test -p 20 ./cmd/internal/sessioninventory -run '^TestLiveIncrementalInventoryPrelaunch$' -count=1 -v` for the performance contract, then `PAIR_LIVE_NATIVE_SESSIONS=1 go test -p 20 ./cmd/internal/sessioninventory -run '^TestLiveProviderContractConformance$' -count=1 -v` for fake-versus-installed behavior. Expect no printed native IDs/content and no mutation of installed stores.
- [x] **Step 6: Add the durable conformance entry point.** Add `make test-session-inventory-conformance` to run both exact live commands with `-p 20`; document it in `atlas/session-identity.md` as required during #156 verification, before every scanner/provider contract version change, and in the monthly operator maintenance run.
- [x] **Step 7: Update the atlas and issue log.** Map the catalog/proof/query flow, corruption recovery, append-only trust boundary, immediate Alt+X ordering, and conformance cadence; link any new atlas page from `atlas/index.md`. Record measured results and `ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`, and `ARCH-MOCK` consequences in the issue log.
- [x] **Step 8: Run complete verification with bounded concurrency.** Run `go test -p 20 ./... -count=1`, `go vet -p 20 ./...`, `make test-lua`, the relevant shell suites from Step 4, `zellij --config-dir zellij setup --check`, and `git diff --check`; expect every command PASS and no more than 20 Go package workers. Inspect `make test-lua` and shell wrappers before running; if they invoke Go, pass or add the repository's `GOFLAGS=-p=20` seam.
- [x] **Step 9: Commit the final evidence.** Run `git add Makefile cmd/internal/sessioninventory atlas workshop/issues/000156-incremental-session-inventory.md workshop/plans/000156-incremental-session-inventory-plan.md && git commit -m 'session inventory: #156 verify incremental native indexing'`.

## Revisions

### 2026-08-29 — close review production-authority convergence

The first whole-issue review found that several planned authorities existed
only as primitives or tests. The corrective round routes watcher/query work
through a production `IncrementalInventory` façade backed by
`ReconcileCatalog`, wires keyed proofless migration and durable publication into
the watcher lifecycle without making ordinary lookups perform validation,
implements raw `B-1` launch-boundary framing, makes same-key catalog publication
monotonic under stale writers, removes the v1 whole-corpus watcher adapter, and
compares installed provider transitions against the stateful fake. It also
corrects the Task 2 checklist and the effective `Makefile.local` file list.
The B-1 framer remains downstream of target authority: it can retain wholly
post-launch records for an established or explicit target, but it never makes
an unbound preexisting artifact eligible merely because that artifact grew.
