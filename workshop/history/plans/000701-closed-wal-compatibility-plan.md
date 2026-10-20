---
collaboration-mode: discussion
---

# Closed WAL compatibility implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 for execution strategy. Execute the warm snapshot-safety work in the main session; bounded independent test work can be delegated. SDLC owns the boundary review.

**Goal:** Back up an observed quiescent WAL database without native sidecars using the installed system SQLite, preserving native bytes, opaque SQL schema and rowids.

**Architecture:** Before any SQLite acquisition, select a cold route only for the WAL header and absence of all coordination files. Make a private copy through the held source descriptor, prove a second matching full read under unchanged source and parent metadata, then join the existing private SQLite validation and atomic staging path. Every other source retains the existing pinned backup route. No tool-error-driven fallback or retry.

**Tech stack:** Go, existing x/sys/unix descriptor operations, SHA256/fileDigest, SQLite CLI and its existing exec-alarm wrapper; disposable Python SQLite fixtures.

## Decision and evidence

1. **Recommended: verified private copy for observed cold WAL input.** No extra runtime dependency and no SQLite connection to the cold native path. Costs two bounded full reads; namespace churn may conservatively refuse an otherwise safe copy. This is the issue's option (a), strengthened with ctime, held descriptors, parent mutation and matching read digests.
2. **Pinned known SQLite implementation.** Ordinary SQLite handles concurrent writers correctly and Homebrew 3.53.4 passes the disposable closed-WAL probe. However, making that implementation mandatory adds deployment/version ownership and a runtime dependency. It is unnecessary if the conservative cold route is accepted; do not silently pick a Homebrew path from this machine.
3. **Immutable native open or blind copy.** Rejected: neither participates in live writer coordination, and private integrity alone cannot establish a coherent copy.

Measured before implementation, on canonical disposable paths: system 3.54.0 readonly schema access exits 1 without source changes; Homebrew 3.53.4 succeeds with coordination sidecars. System SQLite validates a private writable copy and preserves an opaque table, rowid 73 and BLOB 0001FF. The initial /var spelling tripped -nofollow independently; all conformance fixture paths must resolve symlinks.

SQLite references: [read-only WAL contract](https://www.sqlite.org/wal.html#read_only_databases), [unsafe active file copies](https://www.sqlite.org/howtocorrupt.html#_backup_or_restore_while_a_transaction_is_active). The backup API remains the active-writer route.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| coldWALHeader | cmd/internal/historyarchive/sqlite_cold_model.go | new |
| coldFileStamp | cmd/internal/historyarchive/sqlite_cold_model.go | new |
| coldSourceStamp | cmd/internal/historyarchive/sqlite_cold_model.go | new |
| sameColdSource | cmd/internal/historyarchive/sqlite_cold_model.go | new |

- `coldWALHeader` recognizes the SQLite signature and WAL read/write header bytes (2/2), requiring the full inspected header. It does not interpret SQL schemas or general database contents. Noneligible headers use the established SQLite route; a short/errored inspection cannot authorize a copy.
- `coldSourceStamp` is a value projection of source and parent device/inode, size, mode, mtime and ctime, plus explicit absence of WAL, SHM and rollback journal. Pure comparison never calls Stat. One capture owns all stamps; none are cached between passes.
- `sameColdSource` compares those projections and both complete-read sizes/digests. Reuse `sameSource` where its identity/size/mtime part applies rather than duplicating it; ctime and directory mutation are additional proof. Pure value tests include each changed field independently. These entities add no durable format or user state.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| observeColdSource | cmd/internal/historyarchive/sqlite_cold.go | new | held DB/parent descriptors, nofollow name checks, unix.Stat metadata |
| copyColdWAL | cmd/internal/historyarchive/sqlite_cold.go | new | bounded context-aware descriptor reads and private file creation |
| SQLiteSnapshotter.Snapshot | cmd/internal/historyarchive/sqlite_snapshot.go | modified | route choice and existing lease, validation and atomic staging |
| SQLiteRunner / ExecSQLiteRunner | cmd/internal/historyarchive/sqlite_snapshot_exec.go | existing, reused | private validation; current active acquisition unchanged |
| coldSQLiteFile | cmd/internal/historyarchive/sqlite_cold.go | new | held descriptor adapter for coldSQLiteInput |
| coldSnapshotOutput | cmd/internal/historyarchive/sqlite_cold.go | new | private writer preserving local IO failure provenance |
| coldInputFixture | cmd/internal/historyarchive/sqlite_cold_test.go | new | stateful descriptor/reader adapter with explicit mutation gates |

Use existing `openSafeDir`, `safeParent`, `openRegularAt`, `fileDigest`, `validateRoots`, source size cap, scratch lifetime lease and `snapshotFailure`. Do not promote the unbounded `copyRegularFile` test helper into production. Register every new production file in artifactpath's existing exhaustive manifest.

The test source adapter holds real disposable files and advances mutations at a specified read/observation boundary; its state includes bytes, identities, directory generation, sidecar presence, read counts and cancellation. It shares the production copy/observation seam, rather than mocking a final success boolean. Real SQLite conformance separately qualifies the operating-system observations and CLI behavior.

## Consistency contract and sequencing

Trusted: ordinary SQLite writers; local filesystem file identity and high-resolution mtime/ctime observations; private work directory inaccessible to other ordinary users. SQL schema is opaque. A hostile process able to forge source metadata or an unreliable/coarse metadata filesystem is outside this observed-quiescence proof; do not call it exclusive ownership or prove no process has the DB open.

The single-resource question: our scratch lease owns work cleanup, but cannot exclude source writers. SQLite does not honor flock, ordinary read transactions coexist with WAL writers, and manually reproducing SQLite's internal lock protocol would add a second SQLite implementation. An exclusive native writer lock would need authority this read-only collector does not have. Hence the conservative observation route, only for absent coordination files; active writers stay with native SQLite's locks.

1. Preserve all current canonical root-overlap, relative-path, regular-file, descriptor and DB+WAL size admission **before effects**.
2. Hold the original DB descriptor and its parent descriptor for the whole capture. Inspect the fixed SQLite header without consuming the subsequent full read. Observe -wal, -shm and -journal with nofollow semantics; presence selects the normal route (unsafe special files still refuse). An observation error never means absence.
3. If cold eligible, capture source and parent stamps; confirm source pathname and parent pathname still identify held descriptors. Use one acquisition-start deadline, `SnapshotReadSeconds(DBSize)`, never renewed. No SQLite process opens this native path.
4. Create only fixed private `snapshot.sqlite` with exclusive creation and 0600. Stream the original descriptor into it with `fileDigest`, up to the admitted size/max bound; require exact observed length. No unbounded reopen by native pathname.
5. Reobserve source and directory, including all sidecar names and path identities. Rewind that same descriptor and hash the entire source a second time, bounded by the same deadline. Require exact length, equal digest and unchanged stamps before/between/after the reads. Parent mtime/ctime detects a sidecar's create/delete even when final names are absent. DB ctime catches an ordinary write that restores mtime. Refuse on any error or uncertainty; do not switch to active acquisition in this attempt.
6. The final successful fence is the capture point. Later ordinary writes are allowed: the owned private candidate represents the earlier stable state. No read transaction remains pinned and the cold route creates no native sidecars.
7. Join existing size/regular-file check, `SQLiteValidate` (DELETE journal and integrity), chmod, unchanged-content comparison and atomic stage rename. Validation acts only on the private candidate. Unknown tables/columns, BLOBs and rowids survive; no VACUUM or SQL dump/rebuild.
8. Existing deferred cleanup and lease-based orphan sweep remove the private candidate/any private sidecars on every path, including owner death. Existing continuous pass publishes only SnapshotReady and records other outcomes. Prior staged files and verified bundles remain intact on refusal.

## Operator projection

Actions: allow the next scheduled pass; ask the hub to repair the reported configuration/source.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Eligible stable cold input; private integrity succeeds | Existing ready/healthy database report | Allow the next scheduled pass | None; new snapshot is at the capture point |
| Source quiescence cannot be established (changed/unreadable/unknown capture evidence or deadline) | Existing deferred database report with precise uncertain-input reason | Allow the next scheduled pass | This pass has no new bundle; earlier verified bundle retained |
| Native active source | Existing pinned-backup report | Allow the next scheduled pass | Existing outcomes unchanged |
| Corrupt private candidate, proven unsafe path/configuration or tool failure | Existing rejected/failed report with reason | Ask the hub to repair the reported configuration/source | This pass has no new bundle; earlier verified bundle retained |
| Cancellation or collector death | Existing interrupted pass; leased work removed by current cleanup/sweep | Allow the next scheduled pass | Only unpublished private work |

No new UI, warning timer, schema, status enum or online retry. #700's freshness warning remains the operator escalation when deferred/failed sources age. Namespace churn from unrelated files may defer the cold route; say so in the diagnostic instead of claiming the DB is corrupt.

## Operating envelope and residue

- Batch operation, no UI critical path. Same serial per-database scheduling, 8 GiB raw source cap and 5..30 s acquisition budget as current code; cold reading has no native lock pin. Once private, existing 600 s validation bound applies.
- Let B be DB bytes, N source databases: cold acquisition reads at most fixed header + 2B and writes B, uses fixed 64 KiB buffers; metadata/name checks have a fixed per-source count. No sessions-tree walk or directory listing, no growth in catalog scans, no retry. N/2N and B/2B tests count actual reader and observation operations, not elapsed time.
- One private candidate of at most 8 GiB under existing sqlite-* work/lease; no extra copy file is needed. Producer is copyColdWAL, last consumer is private validation/staging; defer or existing Sweep removes residue. One pass result uses existing runtime reporting, no durable new records.
- Cancellation checked by contextReader on each block and before validation/publication. Owner death cannot leave a native reader from the cold acquisition; any validation child retains the existing alarm and inherited lease.
- No live Codex data, real archive publication, clipboard, Drive, new install or pair:0 use during development. Headless conformance uses disposable files. Ask the hub separately if acceptance requires a real memories_1 pass.

## Function-level verification strategy

| Production function | Adversarial input / independent oracle | Named test / mechanical guard |
|---|---|---|
| coldWALHeader | Malformed/truncated header class through generated-prefix/property checks | TestColdWALHeader |
| sameColdSource | Mutated evidence values through independent field-variation properties | TestSameColdSourceRequiresEveryStampAndNoSidecars |
| observeColdSource | Unavailable or changed filesystem authority through a stateful filesystem fixture; unknown never grants copy authority | TestObserveColdSourceNeverTreatsUnknownAsAbsent |
| copyColdWAL | Controlled capture event sequences with independent exact-output/source-preservation/refusal invariants; counted B/2B and N/2N work | TestColdWALCopyRefusesMutations, TestColdWALCopyBoundsAndCancellation, TestColdWALCopyCountedCost |
| coldSnapshotOutput.Write | Real local write failure through a read-only descriptor; preserve the typed output category independently of source uncertainty | TestColdSnapshotOutputPreservesLocalWriteFailure |
| SQLiteSnapshotter.Snapshot | Stateful lane selection and failed-candidate sequences; exact acquisition counts and prior-stage/cleanup invariants | TestSQLiteSnapshotColdAndActiveRoutes, TestSQLiteColdRefusalPreservesPreviousStage |
| ExecSQLiteRunner | Disposable native CLI conformance against the fake contract; independent SQL and full-source-image oracles | TestSQLiteClosedWALNativeConformance, TestSQLiteClosedWALRealWriterDuringCapture |
| ContinuousPass | Mixed-result production pass through the portable store; verified-catalog and isolation invariants | TestContinuousColdWALPublicationAndIsolation |
| All new sources | exhaustive source inventory and test-only symbol checks | full artifactpath package |

The real conformance fixture qualifies the configured SQLite (ordinary portable success) and /usr/bin/sqlite3 on macOS (record version and reproduce its specific refusal). Do not assert every SQLite build must refuse. Installed Homebrew conformance is optional when available, never an installation requirement. Existing active-writer conformance must keep advancing while backup succeeds.

## Implementation tasks (one close boundary)

### 1. Red characterization and controlled races

Files: new sqlite_cold_model_test.go, sqlite_cold_test.go and sqlite_cold_conformance_test.go beside existing snapshot tests. Extend continuous_snapshot_integration_test.go only for the production publication path.

- [x] Retain the measured closed fixture, opaque schema/rowids and canonical path qualification. Write the public Snapshot regression first; run it before code and require the system CLI failure on macOS.
- [x] Add the stateful source adapter and explicit read-boundary mutation gates, with a stable positive control and each changed source field/sidecar as negative controls. No sleeps or external-tool wall-time assertions.
- [x] Add exact counted B/2B and N/2N cost assertions and cancellation/cleanup/previous-stage tests; capture red evidence.

### 2. Minimal cold route, shared publication

Files: new sqlite_cold_model.go, sqlite_cold.go; modify sqlite_snapshot.go and artifactpath/manifest.go.

- [x] Implement pure header/stamp rules, descriptor-based observation and bounded copy/re-read. Reuse existing hashing and context machinery; keep metadata IO out of pure entities.
- [x] Wire one pre-acquisition route choice. Cold refusal never falls through to pinned backup; native active acquisition, private validation and all publication logic keep their current owners.
- [x] Run the red subset to green, then the whole historyarchive package and source inventory. Commit code and actual evidence with the session Agent trailer.

### 3. Qualification and delivery

Files: atlas/history-backup.md, README.md (correct the existing all-transaction summary); issue and this plan.

- [x] Run existing SQLite active-writer, owner-death, source-overlap and continuous-publication tests alongside all new cold tests, then full artifactpath guards. Explain any changed expectation from its contract; do not merely update it.
- [x] Document supported observed-cold semantics, conservative namespace refusal and the single recovery action. Keep no format/dependency claim honest; no manifest/catalog codec changes (ARCH-EVOLVE).
- [x] Run the recorded branch suite on the pushed implementation head.

Delivery after qualification: close review; address findings with evidence. Enter sdlc merge promptly on SHIP, let it own main integration and its authoritative suite, then report Git-verified merge and acceptance to the hub. Stop before #710 until dispatched onward.

## Architecture disposition

- ARCH-DRY: both paths join one validator, staging publisher, catalog and cleanup owner; existing hashing/safety helpers reused.
- ARCH-PURE: header and captured-stamp comparison are values; descriptor/stat/copy/CLI operations are integration.
- ARCH-PURPOSE: reproduces the actual closed-WAL failure and fixes it without restoring active-writer starvation.
- ARCH-MOCK: mutation sequence fake shares the actual read seam; disposable SQLite conformance validates the real CLI and metadata contract.
- ARCH-CONSTRAINTS: two bounded source reads, one fixed candidate, existing caps/deadlines; counted growth tests.
- ARCH-SECURE: no writable source open, immutable native open, schema interpretation, credentials or source/output overlap; trusted FS/writer assumptions explicit.
- ARCH-STATE: no new durable phases; capture uncertainty refuses before current ready/publication transition. Explicit mutation gates cover the risky interleavings.
- ARCH-FUNERAL: existing lease/defer/sweep own the private candidate and sidecars; no new persistent family.
- ARCH-EVOLVE: format-neutral; opaque SQL objects preserved as bytes, current catalog/manifest codecs unchanged.

## Revisions

### 2026-10-19 12:16 PDT — hub approval and PQ-1 strategy compression

Hub approved the mechanism (receipt a29363a1-7c62-43e8-92bd-2bc7cc45e531) and explicitly requires uncertain cold captures to defer without retry. The projection now makes that distinction from a proven unsafe configuration or corrupt candidate. PQ-1 found the strategy table too close to a prose case inventory: compressed the whole table to one adversarial class/mechanical guard per risky function, retaining exact test names, controlled sequencing and independent oracles. No mechanism or scope change.

### 2026-10-19 12:23 PDT — implementation map and executable qualification

The value stamp has an explicit coldFileStamp member type and the OS adapter is coldSQLiteFile; mapped both alongside the stateful fixture and actual test declarations. Real writer conformance adds rollback to isolate transient namespace mutation from DB changes. Native readonly characterization runs on a separate fixture because conforming CLIs can create sidecars: only production Snapshot is required to preserve its untouched source image. Added a dedicated continuous_cold_test.go for catalog/publication qualification instead of growing the older test file. README's existing all-transaction summary is corrected without a new invocation or review-marker resolution. Separated future close/landing instructions from implementation checkboxes so qualification can honestly finish before the close gate. No scope or mechanism change.

### 2026-10-19 12:35 PDT — BR-1 preserves failure evidence

The first boundary found that routing every cold error to deferred contradicted the approved projection for proven unsafe paths/local failures. The complete rule is now: source mutation or unknown source evidence/deadline defers; a proven invalid source type/size or local output create/write/close failure rejects. Typed errors carry that distinction through the shared snapshotFailure formatter, without matching error text. The private writer adapter preserves write provenance while hashing/copying remains shared. Updated controls for the whole class went red before the fix; normal mutation/deadline controls remain deferred. Prior-stage and catalog retention, no fallback and all cleanup owners are unchanged. The queued old-code suite was cancelled through local-check cancel before starting; a fresh recorded run is required on the correction.
