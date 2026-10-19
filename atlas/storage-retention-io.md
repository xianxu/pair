# Managed storage I/O inventory

This is the checked map of the managed retention paths integrated in #239,
including the merged continuation/recovery paths. It complements
[Storage retention](storage-retention.md) and the artifact-family registry.
Editor guards below apply when the managed launcher advertises
`PAIR_RETENTION_PROTOCOL=1`; compatibility with an older unmanaged launcher does
not gain these guarantees by loading the Lua files alone.

An artifact classification says what a file is; this map says which entrypoint
protects access, what renews meaningful use, and which behavioral test exercises
that contract.

This is a bounded source audit, not proof that every filesystem call in Pair is
managed. It covers the entrypoint families below. The anchor test checks that
listed functions/tests still exist and that selected direct guard calls remain
in their function bodies. It does not prove arbitrary transitive call graphs,
execute the named integration tests, or detect a newly introduced unlisted
entrypoint. A new managed consumer must add a row and behavioral evidence.

| Entrypoint / checked source | Artifact accessed | Existing protection chain | Meaningful-use rule | Checked behavioral evidence |
| --- | --- | --- | --- | --- |
| Create Pair: `cmd/internal/launcher/createflow.go#runCreate` | Selected tag's draft, config, ledger, session binding and launch sidecars | `cmd/internal/launcher/retention.go#beginRetention` → `cmd/internal/launcher/retention.go#BeginRetention` → process registration and start reservation before selected effects; `BeforeSpawn` marks uncertain child effects; child roles acknowledge reservation after registration | Successful foreground launch refreshes use; pre-spawn failure cancels reservation, uncertain post-spawn failure retains it | `cmd/internal/launcher/retention_test.go#TestRetentionCreateReservesBeforeEffectsAndFinishesSuccess`; `cmd/internal/launcher/retention_test.go#TestRetentionCreateFailureStopsEffectsOrRetainsFailedStart` |
| Attach and restart loop: `cmd/internal/launcher/createflow.go#runOnce`, `cmd/internal/launcher/createflow.go#RunLaunch` | Existing tag's selected state and restart source/destination namespaces | `cmd/internal/launcher/retention.go#retainRestartOwners` protects different destination tags; `cmd/internal/launcher/retention.go#finishLaunchRetention` releases after cleanup, marker reads and acknowledgment | Explicit successful attach counts; `PAIR_RETENTION_BACKGROUND` suppresses renewal; process lifetime alone does not renew use | `cmd/internal/launcher/retention_test.go#TestRetentionHeldThroughPostHandoffReads`; `cmd/internal/launcher/retention_test.go#TestRetentionRestartProtectsNewTagBeforeReadingIt`; `cmd/internal/launcher/retention_test.go#TestRetentionBackgroundAttachDoesNotRefreshUse` |
| Standalone continuation seed/retry: `cmd/internal/launcher/createflow.go#runCreate`, `cmd/internal/launcher/createflow.go#RunLaunch` | Exact embedded checkpoint or selected source's digest-bearing seed written to managed draft | `cmd/internal/launcher/retention.go#writeRetainedDraft` → `cmd/internal/launcher/retention.go#WriteChanged` → durable use intent before actual write; restart intent acknowledged only after replacement handoff | Changed seed is authored use even if later launch fails; unchanged seed does not renew; failed acknowledgment retains restart intent and releases process lifetime | `cmd/internal/launcher/retention_test.go#TestRetentionCheckpointHandoffAndAcknowledgment`; `cmd/internal/launcher/retention_test.go#TestRetentionContinuationWriteIsUseEvenWhenLaunchFails`; `cmd/internal/launcher/retention_test.go#TestRetentionPostHandoffMarkerFailureReleasesLifetime` |
| Wrapper entry: `cmd/internal/wrapcmd/wrap.go#Run` | Managed raw scrollback, event/adaptation sidecars and wrapper-owned session state | `cmd/internal/storagegc/lease.go#AcquireSelectedProcess` → `cmd/internal/storagegc/lease.go#AcquireRoleProcessTarget`; actual wrapper registers before run and closes its lease afterward | Raw output and lifetime registration do not themselves renew authored use; startup reservation is acknowledged only after role registration | `cmd/internal/wrapcmd/retention_test.go#TestRunProtectsActualWrapperLifetime`; `cmd/internal/storagegc/lease_test.go#TestRoleAcquisitionAcknowledgesStartAfterRegistration` |
| Prompt/history publication: `cmd/internal/pairlog/retention.go#managedLogWrite` | Exact selected owner's prompt log | Owner/path validation → process lease → `cmd/internal/storagegc/coordinator.go#BeginUse` → existing log-store publication; `logEffects.Rename` observes actual changed publication | Changed committed bytes complete use; no-op retry cancels intent without renewal; indeterminate publication keeps intent | `cmd/internal/pairlog/retention_test.go#TestManagedLogChangesPublishUseButRetriesDoNot`; `cmd/internal/pairlog/retention_test.go#TestManagedLogRejectsWrongOwnerBeforeWriting` |
| Draft/history/queue programmatic saves: `nvim/init.lua#local function write_file`, `nvim/init.lua#retention:change` | Managed draft/history/queue bytes, including Lua autosave, queue deletion and append-style changes | Editor `nvim/retention.lua#function M.setup` registers actual PID; `nvim/retention.lua#function Guard:write` and `nvim/retention.lua#function Guard:change` invoke `pair retention begin` before native IO, then complete | Byte-identical write returns without renewal; changed authored bytes renew; partial or failed effects retain intent | `nvim/retention_test.lua#guard:write`; `cmd/internal/retentioncmd/run_test.go#TestUseCompleteAndUnchanged` |
| Native editor writes: `nvim/init.lua#retention:watch_writes` | Managed files owned by draft editor's write predicate | `nvim/retention.lua#function Guard:watch_writes` creates `BufWritePre` intent before editor write and completes in `BufWritePost`; native buffer writes and Lua explicit saves are separate paths | Compare disk bytes first; unchanged save does not renew; failed metadata precondition aborts write; uncertain completion retains protection | `nvim/retention_test.lua#native:watch_writes` |
| Foreground history navigation and viewer lifetime: `nvim/init.lua#retention:view`, `nvim/scrollback.lua#setup('scrollback-viewer')`, `nvim/changelog.lua#setup('changelog-viewer')` | History slot, raw-derived viewer or changelog document | `nvim/retention.lua#function Guard:view` wraps explicit read; setup registers viewer target and initial read; `nvim/retention.lua#function Guard:close` completes foreground-view use before release | Explicit successful read counts; foreground viewer closes with renewed use; simple background registration does not count | `nvim/retention_test.lua#guard:close`; `cmd/internal/storagegc/coordinator_test.go#TestProcessRegistrationDoesNotRefreshExistingUse` |
| Internal editor retention commands: `cmd/internal/retentioncmd/run.go#Run` | Owner metadata, registrations, start acknowledgments and use intents | Validated selected owner and actual PID → coordinator methods under stable root lock; commands receive editor PID rather than claiming the short-lived CLI is the writer | `complete` records actual use; `unchanged` removes only the intent; register/release is lifetime protection | `cmd/internal/retentioncmd/run_test.go#TestRegisterActualWriterLifecycle`; `cmd/internal/retentioncmd/run_test.go#TestEditorRegisterAcknowledgesLaunchReservation` |
| Background title/adaptation readers: `cmd/internal/titlepoller/runcli.go#RunCLI` / `cmd/internal/titlepoller/runcli.go#runCLI`, `cmd/internal/sessionwatch/runcli.go#RunCLI` | Selected owner title/config/adaptation/session-watch sidecars | `cmd/internal/storagegc/lease.go#AcquireSelectedProcess` before polling/adaptation open; selected-tag mismatch refuses; close releases actual process | Polling and process registration do not renew use; they protect files while the process is alive | `cmd/internal/titlepoller/runcli_test.go#TestRunCLIRefusesMissingWrapperLeaseBeforeStorageEffects`; `cmd/internal/sessionwatch/runcli_test.go#TestRunCLIRefusesUnleasedStorageBeforeAdaptOpen`; `cmd/internal/storagegc/coordinator_test.go#TestProcessRegistrationDoesNotRefreshExistingUse` |
| Couch workspace-owner publication: `cmd/internal/couchcmd/titles.go#WriteWorkspaceOwner` | Existing scoped cmux owner metadata | Exact current title target and selected owner → `PublishMetadata` under root coordination lock | Derived metadata does not renew authored use | `cmd/internal/couchcmd/titles_test.go#TestCouchTitlesWorkspaceRespectsExistingListing`; existing PublishMetadata recovery tests |
| Title metadata publication: `cmd/internal/titlepoller/runtime.go#WriteFile`, `cmd/internal/storagegc/stateio.go#PublishMetadata` | Exact selected title PID/configured workspace-owner plaintext; private pending staging | Captured selected owner/path checks → root coordination lock → bounded synced staging and atomic rename; existing pending recovery waits for publisher death/released lock | Derived metadata does not renew authored use | `cmd/internal/titlepoller/runcli_process_test.go#TestTitleCLILeaseEOFRecoversInterruptedPublications`; `cmd/internal/storagegc/stateio_test.go#TestPlaintextMetadataInterruptedPublicationRecovery` |
| Background changelog distillation: `cmd/internal/changelogcmd/changelogcmd.go#runWithEnv` | Raw/event input, cleaned derivative, changelog/anchor/ready outputs | Exact owner validation → `cmd/internal/storagegc/lease.go#AcquireSelectedProcess` before reading; lease closes after render | Background distillation is derived activity, not foreground meaningful use | `cmd/internal/changelogcmd/retention_test.go#TestManagedDistillerLeasesBeforeReadingWithoutTouchingUse` |
| Foreground viewer openers: `cmd/internal/opener/runcli.go#RunScrollbackCLI`, `cmd/internal/opener/runcli.go#RunChangelogCLI` | Selected scrollback/changelog source and detached viewer handoff | Opener lease before access; existing protected-child handoff carries owner/start reservation into actual viewer registration | Opener lifetime is protection; foreground viewer use is recorded by the editor guard, not inferred from successful spawn alone | `cmd/internal/opener/retention_test.go#TestViewerHandoffReservesAndPassesOwnerBeforeSpawn`; `cmd/internal/opener/retention_test.go#TestDetachedChildRetainsProtectionAfterLauncherExits` |
| Park capture production: `cmd/internal/launcher/osruntime.go#ParkScrollback` | Timestamped raw/event pair and capture creation metadata | Capture-producer process lease before reading/copying/moving; coordinator publishes exact capture metadata after payload creation | Capture publication establishes its independent age; capturing raw output does not renew authored owner use | `cmd/internal/launcher/osruntime_test.go#TestParkScrollbackPublishesProducerClock`; `cmd/internal/launcher/osruntime_test.go#TestParkScrollbackEventsFailureRetainsRawWithoutEvents` |
| Raw capture rendering: `cmd/internal/scrollbackcmd/scrollbackcmd.go#RunWithEnv` | Exact canonical raw capture/event pair and rendered derivative | `cmd/internal/scrollbackcmd/retention.go#acquireRenderLease` registers selected exact target before read; `cmd/internal/scrollbackcmd/retention.go#acknowledgeRenderHandoff` clears matching predecessor intent only after registration | Renderer itself does not renew use; precise target registration protects the capture independently of other captures | `cmd/internal/scrollbackcmd/retention_test.go#TestManagedCaptureReaderRegistersExactTargetBeforeRead`; `cmd/internal/scrollbackcmd/retention_test.go#TestExplicitOwnerRejectsOtherCaptureBeforeOutput` |
| Explicit Couch resume and automatic warm reattachment: `cmd/internal/couchcore/retention.go#beginResumeRetention` | Pair owner storage addressed by exact Couch thread | Coordinator actual-process registration spans resume; success callback publishes explicit-use intent when meaningful, then releases | Explicit successful resume renews; background reattachment does not | `cmd/internal/couchcore/retention_test.go#TestRetentionExplicitResumeTouchesButBackgroundReattachDoesNot` |
| Couch continuation publication/recovery: `cmd/internal/couchcore/continuation_store.go#PublishContinuation`, `cmd/internal/couchcore/recovery_execute.go#RecoverThread` | Embedded checkpoint/request in ThreadRecord and derived materialization in Couch namespace | Publication uses revision-checked store mutation via `cmd/internal/couchcore/threadstore.go#withLock`; production coordinated store takes Pair root before Couch store. Active store references protect Pair owners. Materialization is derived in Couch namespace, outside Pair-owner collection; recovery launches reuse guarded resume/create | Request publication/reconciliation is lifecycle bookkeeping, not independent authored Pair use; launch/seed rules above own renewal. External selected checkpoint documents are not managed Pair-owned files | `cmd/internal/couchcore/retention_test.go#TestRetentionMembershipWaitsForRootCoordination`; `cmd/internal/couchcore/recovery_execute_test.go#TestRecoverThreadAbsentSourceRetainsExactSnapshot` |
| Couch archive/restore and collector references: `cmd/internal/couchcore/retention.go#RestoreThread`, `cmd/internal/couchcore/retention.go#RetentionSnapshot`, `cmd/internal/couchcore/archive_gc.go#DetachArchive` | Archived ThreadRecord, grace clock, cross-store detach receipt and active membership | Coordinated membership journal; collector holds root lock and exact reference/receipt proof before detach | Archive establishes independent grace; snapshots and archive bookkeeping do not renew Pair meaningful use; unknown/unreadable references protect | `cmd/internal/couchcore/retention_test.go#TestRetentionArchiveAndRestoreRecoverEachJournalStep`; `cmd/internal/couchcore/retention_test.go#TestRetentionSnapshotProtectsUnreadableAndBlocksPendingRecovery`; `cmd/internal/couchcore/archive_gc_test.go#TestArchiveDetachReceiptSurvivesEveryCrashStep` |
| Preview/inventory: `cmd/internal/storagegc/snapshot.go#WithReadLock` | Existing owner/start/use/process metadata and collection candidates | Read-only snapshot path; no metadata initialization or cleanup masquerading as preview | Inspection does not renew use or create retention state | `cmd/internal/storagegc/snapshot_test.go#TestReadSnapshotDoesNotInitializeRoot`; `cmd/internal/storagegc/collector_test.go#TestCollectorPreviewIsReadOnlyAndSeparatesCaptureAge` |

The named Lua tests are top-level executable fixtures rather than Go-style test
functions; their anchors identify the exercised guard calls. Some rows combine
an entrypoint-specific integration test with a coordinator contract test. This
is stated coverage, not a claim that the entrypoint test independently proves
every coordinator invariant.

<!-- retention-call: cmd/internal/launcher/osruntime.go#ParkScrollbackContext -> AcquireSelectedProcess -->
<!-- retention-call: cmd/internal/launcher/osruntime.go#ParkScrollbackContext -> PublishCaptureMetadata -->
<!-- retention-call: cmd/internal/launcher/createflow.go#runCreate -> beginRetention -->
<!-- retention-call: cmd/internal/launcher/createflow.go#runCreate -> writeRetainedDraft -->
<!-- retention-call: cmd/internal/launcher/createflow.go#RunLaunch -> finishLaunchRetention -->
<!-- retention-call: cmd/internal/launcher/retention.go#BeginRetention -> RegisterProcess -->
<!-- retention-call: cmd/internal/launcher/retention.go#BeginRetention -> ReserveStart -->
<!-- retention-call: cmd/internal/wrapcmd/wrap.go#Run -> AcquireSelectedProcess -->
<!-- retention-call: cmd/internal/pairlog/retention.go#managedLogWrite -> BeginUse -->
<!-- retention-call: cmd/internal/titlepoller/runcli.go#runCLI -> AcquireSelectedProcess -->
<!-- retention-call: cmd/internal/titlepoller/runtime.go#WriteFile -> PublishMetadata -->
<!-- retention-call: cmd/internal/couchcmd/titles.go#WriteWorkspaceOwner -> PublishMetadata -->
<!-- retention-call: cmd/internal/storagegc/stateio.go#PublishMetadata -> WithLock -->
<!-- retention-call: cmd/internal/sessionwatch/runcli.go#RunCLI -> AcquireSelectedProcess -->
<!-- retention-call: cmd/internal/changelogcmd/changelogcmd.go#runWithEnv -> AcquireSelectedProcess -->
<!-- retention-call: cmd/internal/scrollbackcmd/scrollbackcmd.go#RunWithEnv -> acquireRenderLease -->
<!-- retention-call: cmd/internal/couchcore/retention.go#beginResumeRetention -> RegisterProcess -->
<!-- retention-call: cmd/internal/couchcore/threadstore.go#withLock -> WithLock -->

## Boundaries and follow-through

Diagnostic logs use their separate generation/writer registry and independent
age policy described in [Storage retention](storage-retention.md); they are not
renewed through authored-content use intents. Ordinary repository files, native
agent stores and operator-selected external checkpoints are outside the managed
Pair owner file set. The artifact registry remains authoritative for membership;
this table does not add file families by implication.

A managed registration can conservatively block collection without advancing a
clock. Uncertain effects retain intent. These are deliberate distinctions:
background reads, derived writes and namespace observation must not keep an idle
owner young merely because a process wakes up.

When adding or moving a consumer, update its row, the source/test anchors and any
direct-call guard below it. Run
`go test ./cmd/internal/storagegc -run '^TestManagedIOInventoryAnchors$' -count=1`
plus the behavioral test for the changed row. Removing a guard or changing its
clock semantics requires updating and reviewing the implementation and test,
not only repointing this document to a surviving function.

Diagnostic payload effects are coordinated separately from meaningful session
use: `cmd/internal/diagnosticlog/creation.go#ensureCurrent` records an empty
reserved inode before no-replace publication; `cmd/internal/diagnosticlog/append.go#appendChunk`
writes a compact reservation in place, appends at most 64 KiB, verifies the exact
observed batch, then renames one checkpoint. The reservation stores length and
hash rather than another copy of the payload. `recoverAppend` first reconciles
legacy payload intents, then complete matching reserved batches. An uncertifiable
partial batch moves intact to an explicitly unverified generation after writer
and identity proof; logging continues in a fresh current file. Foreign complete
tails refuse. Both creation and append use the stable per-log lock and do not
renew session activity. `compact_append_test.go` and `quarantine_test.go` cover
interrupted recovery and proof fences; `amplification_test.go` counts actual
payload, reservation and metadata writes, checkpoint renames and hot syncs.

Long-lived wrapper diagnostics use `buffered.go`: whole records enter a bounded
256 KiB owner, flush at 64 KiB or on a dirty-only five-second tick, and drain and
join on close. Empty ticks perform no IO. A healthy process can lose up to five
seconds of pending diagnostics on a crash, subject to scheduling/IO latency;
the absolute accepted in-memory bound is 256 KiB per writer. Raw scrollback and
its offset sidecar use separate 64 KiB buffers, flushed raw first by the capture
pump every 250 ms, on every pump exit before child wait, and on orderly teardown. Pump exit also
closes admission, so an in-flight resize cannot leave sidecar events behind after
its clock stops. These intervals describe process
crash exposure, not power-loss durability. Receipt, park, recovery, generation
rotation keeps its durable synchronization. Managed diagnostic deletion uses certified seven-day age or oversized legacy
size, independent of backup and session migration. Whole-generation eviction
reuses the durable deletion intent, without copying payloads. Live open holders
and unknown process evidence remain blocked. There is no global byte ceiling
on seven days of busy diagnostics. Raw
capture size rotation remains separate work (#582): no VT state is discarded.

Exact registry declarations reuse unchanged content and inode, but reconfirm the
registry directory's durability on Open. Failed sync after rename remains an
error on identical retry until synchronization succeeds; no declaration rewrite
is needed (`registry_test.go`).


## History backup storage

The explicit `pair archive create` batch path is owned by
[`historyarchive.Archiver`](../cmd/internal/historyarchive/archive.go) and
[`archivecmd`](../cmd/internal/archivecmd/run.go). Its sources are selected stable
regular files. It checks identity, size, mtime and hash across description and tar
writing, preserves those sources, and does not renew managed-session use or
change a source's retention policy. Inventory and help do not create backup
storage. This is not yet the archive-before-eviction integration: consumers must
wait for successful verified catalog evidence before any future deletion step.

The archive invocation owns temporary ciphertext; ordinary returns remove it.
Serialized recovery removes only owned staging remnants from interrupted writes:
the catalog lock protects catalog-adjacent staging, and the sink serializes its
own publication staging recovery. Published bundle/label objects are never
cleaned up by this recovery. A partial published pair requires inspection; a
complete matching uncataloged pair can be verified and adopted on explicit retry.

The sink owns immutable encrypted bundles and plaintext labels. The private local
catalog owns full source manifests and verified object references under one
cross-process lock. Records stream across 64 MiB shards, with automatic rollover
and synced repair of an incomplete trailing JSONL line; complete records are
retained and validated. Every success, including reuse, reconfirms durability.
These artifacts remain until an explicit retention decision; no background
schedule or automatic deletion is introduced. Disk usage is encrypted payloads
plus labels, private catalog shards and in-flight staging. A bundle targets
256 MiB raw payload, admits at most 1 GiB raw payload and 2 GiB ciphertext, and
uses one zstd compression thread; compression working memory is additional.

[History backup](history-backup.md) maps the complete request, bounds, privacy
and recovery contract. Behavioral coverage is in `historyarchive/archive_test.go`,
`sink_test.go`, `catalog_test.go`, `files_test.go` and `codec_test.go`; the isolated
real transform test is `codec_live_test.go`. These archive paths do not imply a
new managed-owner file family or change retention of authored session or history data.
