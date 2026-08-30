# Couch Verified Park and Resume Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Couch park a promptly acknowledged, durably verified invocation of Pair's existing Alt+x full quit, then resume the same composite thread with its exact saved launch profile and established native session.

**Architecture:** A new leaf package owns the versioned Pair lifecycle request/result vocabulary and crash-safe local file protocol; both `launcher` and `couchcore` consume it without importing each other. Couch persists a pure park transaction before publishing or triggering anything, while Pair remains the sole cleanup executor and publishes the only accepted completion proof. Resume extracts Couch's existing blocked-launch tail but starts the already allocated address and requires the exact #155 native binding at both preflight and Pair launch time (`ARCH-DRY`, `ARCH-PURE`, `ARCH-PURPOSE`, `ARCH-MOCK`).

**Tech Stack:** Go, strict JSON, local atomic files and `golang.org/x/sys/unix` advisory locks, existing ThreadStore journal/CAS, Pair launcher and Zellij seams, Couch PTY console, stateful fakes, opt-in macOS/Linux live conformance.

**Execution constraint:** Work directly on `/Users/xianxu/workspace/pair` as requested by the operator. Run only one Go test command at a time and always cap package parallelism with `-p 20` (ARCH-CONSTRAINTS).

---

## Core Concepts

### Pure Entities

| Name | Lives in | Status |
|------|----------|--------|
| `LifecycleIdentity` | `cmd/internal/pairlifecycle/model.go` | new |
| `QuitRequest` | `cmd/internal/pairlifecycle/model.go` | new |
| `QuitCompletion` | `cmd/internal/pairlifecycle/model.go` | new |
| `CleanupResult` | `cmd/internal/pairlifecycle/cleanup.go` | new |
| `ParkTransaction` | `cmd/internal/couchcore/parktransaction.go` | new |
| `ParkAttempt` | `cmd/internal/couchcore/parktransaction.go` | new |
| `ParkDecision` | `cmd/internal/couchcore/parktransaction.go` | new |
| `VerifiedPark` | `cmd/internal/couchcore/thread.go` | new |
| `ResumeEligibility` | `cmd/internal/couchcore/resume.go` | new |
| `ThreadRecord` | `cmd/internal/couchcore/thread.go`, `cmd/internal/threadrecord/record.go`, and `cmd/internal/threadrecord/lifecycle.go` | modified |

**`LifecycleIdentity`** — stable `(transaction nonce, address, incarnation PID, incarnation process identity)` authority shared by all attempts.
- **Relationships:** 1:1 with a `ParkTransaction`; 1:N with numbered `QuitRequest`/`QuitCompletion` attempts.
- **DRY rationale:** Pair and Couch must compare one exact authority shape rather than reconstructing identity from filenames, revisions, or process exit.
- **Future extensions:** #135 consumes the same successful cleanup identity and proof without introducing another quiescence vocabulary.

**`QuitRequest` / `QuitCompletion`** — immutable, versioned attempt records with strict validation and exact matching.
- **Relationships:** 1:1 per numbered attempt; both belong to one `LifecycleIdentity`.
- **DRY rationale:** Direct Pair Alt+x and Couch Park share one cleanup result vocabulary; only the Couch entry carries a transaction consumer.
- **Future extensions:** Additional Pair-owned cleanup modes widen the enum and validator, never ad hoc marker fields.

**`CleanupResult`** — pure typed result of the ordered Pair full-quit lifecycle.
- **Relationships:** 1:1 with a direct quit or Couch attempt; converted to `QuitCompletion` only for Couch.
- **DRY rationale:** Existing void/best-effort cleanup cannot distinguish completed teardown from client exit; one result classifier serves direct, Couch, tests, and #135.
- **Future extensions:** Progress reporting may expose completed stages without changing success authority.

**`ParkTransaction` / `ParkAttempt` / `ParkDecision`** — append-only durable state and pure transitions for request, awaiting completion, timeout/unknown, retry/recovery, success, and tombstoned abandonment.
- **Relationships:** A thread owns at most one active transaction; a transaction owns 1:N attempts and zero or one terminal success/tombstone.
- **DRY rationale:** Every caller and restart uses one transition authority instead of duplicating retry and late-result rules in handlers.
- **Future extensions:** Remote lifecycle clients can observe the same phases without gaining teardown authority.

**`VerifiedPark`** — terminal thread metadata proving that Pair completed full quit for the removed exact incarnation.
- **Relationships:** 1:1 with a resumable thread; derived only from an active, non-tombstoned transaction success.
- **DRY rationale:** “No incarnation” alone remains legacy/unverified and cannot silently become resumable.
- **Future extensions:** May later carry managed-worktree state from #153.

**`ResumeEligibility`** — pure allow/refuse decision over durable thread state, path evidence, supported agent, and established native binding.
- **Relationships:** 1:1 with a resume request; consumes `VerifiedPark` and the thread-level latest successful launch profile.
- **DRY rationale:** CLI, menu, restart recovery, and tests share every fail-closed refusal class.
- **Future extensions:** #153 may turn missing managed paths into reprovision-before-this-decision, without weakening it.

**`ThreadRecord`** — gains optional latest launch profile, monotonic recency, active park state, verified park, and durable lifecycle/tombstone history.
- **Relationships:** Owns incarnations and lifecycle state for one composite address.
- **DRY rationale:** The launch profile must survive successful incarnation removal; transaction history must survive process restart and late results.
- **Future extensions:** The v2 decoder continues to migrate v1 records in memory; later versions add another explicit decoder branch rather than changing an old version's meaning.

### Integration Points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `LifecycleStore` | `cmd/internal/pairlifecycle/store.go` | new | atomic local request/completion files |
| `LifecycleLock` | `cmd/internal/pairlifecycle/store_unix.go` | new | crash-released OS advisory lock |
| `QuitLifecycleOps` | `cmd/internal/launcher/lifecycle.go` | new | Pair/Zellij/editor/sidecar cleanup effects |
| `ThreadStore` park methods | `cmd/internal/couchcore/threadstore.go` | modified | durable journal and revision CAS |
| `PairLifecycleController` | `cmd/internal/couchcore/park.go` | new | request publication, exact-session trigger, completion observation |
| `NativeBindingResolver` | `cmd/internal/couchcore/artifactcollision.go` | modified | #155 `sessioninventory.QuerySession` authority |
| `ExistingThreadLauncher` | `cmd/internal/couchcore/couch.go` | new | blocked Pair child start/registration on an existing address |
| `ConsoleOperationQueue` | `cmd/internal/couchtty/console.go` | modified | bounded async park/resume UI effects |
| `FakePairLifecycle` | `cmd/internal/pairlifecycletest/fake.go` | new | stateful Pair/session/file lifecycle model |

**`LifecycleStore` / `LifecycleLock`** — publishes immutable request/completion attempts only after file sync, rename, and directory sync under a stable lock inode; the next lock holder can finish a rename-before-sync crash.
- **Injected into:** Pair cleanup publisher and Couch coordinator.
- **Future extensions:** Remote transport may mirror committed records, but local authority remains this store.

**`QuitLifecycleOps`** — narrow context-aware adapter for exact Zellij quiescence, editor reap, scrollback preservation, sidecar removal, poller cleanup, and cmux cleanup.
- **Injected into:** the shared Pair cleanup orchestrator; direct and Couch entries differ only in confirmation/completion policy.
- **Future extensions:** New cleanup stages join this ordered seam and the stateful fake together.

**ThreadStore park methods** — CAS each phase at the current record revision and finalize success in one durable incarnation-removal/recency/verified-park record commit.
- **Injected into:** Couch coordinator and restart reconciliation.
- **Future extensions:** #135 reads the same successful proof; it cannot write a parallel lifecycle file.

**`PairLifecycleController`** — resolves the exact Pair session, commits/publishes attempts, delivers at-least-once triggers, reconciles completions, and coalesces one worker per thread.
- **Injected into:** Couch `park`, Retry, Recover, and Abandon operations.
- **Future extensions:** Progress subscriptions can observe the coordinator without changing authority.

**`NativeBindingResolver`** — returns only an established scanner-authorized native root for the exact `{scope, tag, agent}`.
- **Injected into:** resume preflight; Pair performs the same required-ID check again at launch to close the TOCTOU gap.
- **Future extensions:** Additional harness binding shapes widen #155, not Couch fallback rules.

**`ExistingThreadLauncher`** — shared tail of today's `Spawn` after allocation/profile selection, parameterized by an already-existing thread and exact profile.
- **Injected into:** new-thread start and verified resume.
- **Future extensions:** #153 calls the same path after deterministic worktree reprovisioning.

**`ConsoleOperationQueue`** — renders local feedback first, then schedules bounded owner operations and returns results to the single host-writer loop.
- **Injected into:** actor-focused Alt+x and panel/menu actions.
- **Future extensions:** Other slow owner operations can use the queue without blocking terminal input.

**`FakePairLifecycle`** — persisted state machine for request authority, exact-session presence, ordered cleanup effects, completion publication, crashes, and restarts.
- **Injected into:** Pair direct/Couch equivalence tests and Couch coordinator integration tests.
- **Future extensions:** #135 reuses it for transfer quiescence scenarios (ARCH-MOCK).

## Chunk 1: Durable Lifecycle Foundation

### Task 1: Define and test the shared lifecycle protocol

**Files:**
- Create: `cmd/internal/pairlifecycle/model.go`
- Create: `cmd/internal/pairlifecycle/model_test.go`
- Create: `cmd/internal/pairlifecycle/leaf_contract_test.go`
- Modify: `cmd/internal/artifactpath/paths.go`
- Modify: `cmd/internal/artifactpath/manifest.go`
- Test: `cmd/internal/artifactpath/paths_test.go`
- Test: `cmd/internal/artifactpath/coverage_test.go`

- [ ] Write `TestValidateQuitRequest`, `TestValidateQuitCompletion`, and `TestMatchQuitCompletion` as failing tables for schema version, safe nonce, positive attempt, exact address, positive PID/non-empty process identity, exact Pair session, preserve-scrollback Couch mode, completion key, success/failure exclusivity, and one mismatch case for every repeated field.
- [ ] Run `go test -p 20 ./cmd/internal/pairlifecycle -run 'TestValidateQuit(Request|Completion)|TestMatchQuitCompletion' -count=1`; expect FAIL because the lifecycle types/functions do not exist.
- [ ] Add these exact public shapes and string enums (`preserve-scrollback`; `success|failure`; the spec's seven typed failure codes) with the shown JSON contract:

```go
type Identity struct {
    Nonce           string `json:"nonce"`
    RepoScope       string `json:"repo_scope"`
    Tag             string `json:"tag"`
    PID              int    `json:"pid"`
    ProcessIdentity string `json:"process_identity"`
}

type QuitRequest struct {
    SchemaVersion int         `json:"schema_version"`
    Identity      Identity    `json:"identity"`
    Attempt       uint64      `json:"attempt"`
    Session       string      `json:"session"`
    Mode          CleanupMode `json:"mode"`
    CompletionKey string      `json:"completion_key"`
}

type QuitCompletion struct {
    SchemaVersion int               `json:"schema_version"`
    Identity      Identity          `json:"identity"`
    Attempt       uint64            `json:"attempt"`
    Session       string            `json:"session"`
    Mode          CleanupMode       `json:"mode"`
    CompletionKey string            `json:"completion_key"`
    Outcome       CompletionOutcome `json:"outcome"`
    FailureCode   FailureCode       `json:"failure_code,omitempty"`
    CompletedAt   time.Time         `json:"completed_at"`
}
```

- [ ] Implement strict validation, defensive cloning, and `Match(request, completion)`; completion must repeat and match identity, attempt, session, mode, and completion key.
- [ ] Add `TestPairLifecyclePackageRemainsLeaf`, using Go import inspection like existing contract tests, to reject imports of `launcher`, `couchcore`, `threadrecord`, or `sessioninventory`.
- [ ] Run the focused pairlifecycle command again; expect PASS.
- [ ] Write failing `artifactpath` tests named `TestLifecyclePathsValidateComponents` and `TestLifecycleArtifactFamiliesAreClassified` for one scoped lifecycle directory, stable lock, immutable numbered request/completion files, and exact-session trigger reference.
- [ ] Run `go test -p 20 ./cmd/internal/artifactpath -run 'TestLifecycle' -count=1`; expect FAIL because the constructors/classifications do not exist.
- [ ] Implement the sole path constructors, validate every component before derivation, classify every new producer/consumer in the artifact manifest, and add a coverage assertion that launcher/Couch do not reconstruct lifecycle filenames.
- [ ] Run `go test -p 20 ./cmd/internal/pairlifecycle ./cmd/internal/artifactpath -run 'Lifecycle|QuitRequest|QuitCompletion|Manifest|Coverage' -count=1`; expect PASS.
- [ ] Commit with `git commit -m "#152: define Pair lifecycle protocol"`.

### Task 2: Implement the crash-safe lifecycle store and lock

**Files:**
- Create: `cmd/internal/pairlifecycle/store.go`
- Create: `cmd/internal/pairlifecycle/store_test.go`
- Create: `cmd/internal/pairlifecycle/store_unix.go`
- Create: `cmd/internal/pairlifecycle/store_subprocess_test.go`

- [ ] Write failing `TestLifecycleStorePublishFailureMatrix` cases for create/write, file sync, close, rename, directory sync, validation, and unlock for both request and completion; expect temporary files never to be readable authority.
- [ ] Write failing `TestLifecycleStoreImmutableFinal` cases: absent final publishes; valid byte-identical final is idempotently committed; valid different final returns typed conflict without rename; invalid final is quarantined/refused; prepared final is reconciled. Add two concurrent-publisher cases for identical and conflicting payloads.
- [ ] Run `go test -p 20 ./cmd/internal/pairlifecycle -run 'TestLifecycleStore(PublishFailureMatrix|ImmutableFinal)' -count=1`; expect FAIL because `LifecycleStore` does not exist.
- [ ] Implement one injected file runtime plus `Committed`, `Indeterminate`, and `Conflict` outcomes. Under the lock, inspect any existing final before rename and never use replacing rename as the immutability guard.
- [ ] Publish both record kinds as temp write-all → file sync → close → final-existence recheck → rename → directory sync. On failure before rename, remove temp; after rename, return indeterminate until reconciliation establishes directory durability.
- [ ] Run the store tests again; expect PASS.
- [ ] Write failing subprocess tests `TestLifecycleLockReleasedOnHolderDeath` and `TestLifecycleStoreReconcilesHolderDeathAfterRename` covering crash-before-rename and crash-after-rename for request and completion.
- [ ] Run `go test -p 20 ./cmd/internal/pairlifecycle -run 'TestLifecycle(LockReleased|StoreReconciles)' -count=1`; expect FAIL before the Unix lock/reconciliation implementation.
- [ ] Implement `unix.Flock(LOCK_EX)` on a stable close-on-exec inode that is never renamed/deleted. The next holder validates a final left after rename, directory-syncs it before authority, quarantines/refuses invalid data, and leaves unsyncable state indeterminate.
- [ ] Run the subprocess tests again; expect PASS and one immutable committed result after recovery.
- [ ] Run `go test -p 20 ./cmd/internal/pairlifecycle -run 'Store|Lock|Crash|Reconcile' -count=1`; expect PASS.
- [ ] Commit with `git commit -m "#152: add crash-safe lifecycle store"`.

### Task 3: Add the pure park transaction and persisted thread state

**Files:**
- Create: `cmd/internal/couchcore/parktransaction.go`
- Create: `cmd/internal/couchcore/parktransaction_test.go`
- Modify: `cmd/internal/threadrecord/record.go`
- Create: `cmd/internal/threadrecord/lifecycle.go`
- Test: `cmd/internal/threadrecord/record_test.go`
- Modify: `cmd/internal/couchcore/thread.go`
- Test: `cmd/internal/couchcore/thread_test.go`
- Modify: `cmd/internal/couchcore/starttransaction.go`
- Test: `cmd/internal/couchcore/starttransaction_test.go`

- [ ] Write failing `TestAdvanceParkTransactionMatrix` subtests for begin, request committed, awaiting, request-publish failure, timeout, missing/stale/failed completion, attempt append, conflict/replacement, late older-attempt success, transaction-wide closure, and Abandon tombstone; write `TestMonotonicLastActiveAt` for forward/equal/backward clocks.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore -run 'TestAdvanceParkTransactionMatrix|TestMonotonicLastActiveAt' -count=1`; expect FAIL because the pure model does not exist.
- [ ] Implement stable identity separate from `base_revision` and per-phase `record_revision`, strictly increasing attempts, active/success-eligible/non-tombstoned success validation, and monotonic recency.
- [ ] Run the focused pure-model command again; expect PASS.
- [ ] Write failing threadrecord golden tests: `TestDecodeV1MigratesToV2`, `TestV2RoundTrip`, `TestUnsupportedThreadRecordVersionRefuses`, and structural/defensive-copy/fuzz cases for the lifecycle fields.
- [ ] Run `go test -p 20 ./cmd/internal/threadrecord -run 'V1|V2|ThreadRecordVersion|Lifecycle' -count=1`; expect FAIL because schema v2 and its migration do not exist.
- [ ] Set `SchemaVersion = 2`. Decode an envelope version first, strictly decode v1 into the old shape and migrate it in memory, strictly decode v2 into the new shape, and refuse every other version. New writes use v2. Rollback to an old binary is intentionally fail-closed rather than supported: an old strict decoder refuses v2 instead of misreading it.
- [ ] Put persisted park/attempt/history shapes and validators in focused `threadrecord/lifecycle.go`; add thread-level `LatestLaunchProfile`, `LastActiveAt`, `Park`, `VerifiedPark`, and closed/tombstoned history to v2.
- [ ] Extend every couchcore converter/clone and threadrecord validator/fuzz seed to enforce active nonce uniqueness, identity completeness, append-only attempts, active/tombstone exclusion, verified park without incarnation, and defensive copies.
- [ ] Run the threadrecord golden/structural command again; expect PASS.
- [ ] Write `TestStartRegisteredCopiesLatestLaunchProfile` and `TestFailedStartDoesNotReplaceLatestLaunchProfile`.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore -run 'Test(StartRegisteredCopies|FailedStartDoesNotReplace)' -count=1`; expect FAIL because successful registration does not copy the thread-level profile.
- [ ] Update `StartRegistered` to copy the exact successful profile to both thread and incarnation.
- [ ] Run the focused start tests again; expect PASS.
- [ ] Run `go test -p 20 ./cmd/internal/threadrecord ./cmd/internal/couchcore -run 'ThreadRecord|AdvanceStart|ParkTransaction|Monotonic' -count=1`; expect PASS.
- [ ] Commit with `git commit -m "#152: persist park transaction state"`.

### Task 4: Add ThreadStore park CAS operations

**Files:**
- Modify: `cmd/internal/couchcore/threadstore.go`
- Test: `cmd/internal/couchcore/threadstore_test.go`
- Test: `cmd/internal/couchcore/admission_reconcile_test.go`
- Modify: `cmd/internal/couchcore/storejournal.go`
- Test: `cmd/internal/couchcore/storejournal_test.go`

- [ ] Write `TestThreadStoreParkCASLifecycle`, `TestBeginParkCapturesLegacyLiveProfile`, and `TestFinalizeParkPreservesCapturedProfile`.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore -run 'Test(ThreadStoreParkCASLifecycle|BeginParkCapturesLegacyLiveProfile|FinalizeParkPreservesCapturedProfile)' -count=1`; expect FAIL on missing methods.
- [ ] Implement `BeginPark` to validate one exact live incarnation and persist `requested` before any effect. If `LatestLaunchProfile` is absent on a migrated live v1 record, copy and validate the target incarnation's exact successful profile in this same CAS.
- [ ] Implement `AdvancePark`/`AppendParkAttempt` around the pure transition, preserving completion artifacts outside ThreadStore until final success commits.
- [ ] Implement `FinalizePark` as one `UpdateExistingThread`: revalidate active nonce and exact incarnation, remove only it, close lifecycle, set `VerifiedPark`, update monotonic `LastActiveAt`, and retain the captured exact profile. Implement Abandon as a tombstone-only update.
- [ ] Run the three lifecycle tests again; expect PASS, including old-v1-live → begin → finalize → profile-preserved.
- [ ] Write `TestParkStatesRemainOccupied` covering requested, awaiting, failed, timed-out, unknown, conflict, and tombstoned-with-live-incarnation; only verified success is free.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore -run 'TestParkStatesRemainOccupied' -count=1`; expect FAIL until admission consumes the new states.
- [ ] Update admission to count every non-finalized incarnation/park state as occupied.
- [ ] Run the occupancy test again; expect PASS.
- [ ] Write `TestThreadStoreParkJournalCrashRecovery` with explicit crash hooks before durable journal publication, after durable journal publication, after the park record target image, after journal removal/before root-directory sync, and after root-directory sync. For each boundary, restart must yield the exact prior or next revision, never a partial record, and repeated recovery must be idempotent.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore -run 'TestThreadStoreParkJournalCrashRecovery' -count=1`; expect FAIL because park updates do not yet use the journal and the latter hook boundaries do not exist.
- [ ] Route each park record CAS through a one-entry `storeJournal`; add explicit `BeforeJournalPublish`, after-journal-removal, and after-final-root-sync hooks while retaining the existing after-journal and after-target hooks. The before-publish hook must leave no authoritative journal and restart must observe the exact prior revision.
- [ ] Run the crash-recovery test again; expect PASS at every enumerated journal boundary.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore -run 'ThreadStore.*Park|BeginPark|FinalizePark|AbandonPark|Admission|JournalCrash' -count=1`; expect PASS.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore -run 'ThreadStore.*Park|FinalizePark|AbandonPark|Admission|Journal' -count=1`; expect PASS.
- [ ] Commit with `git commit -m "#152: add atomic park store transitions"`.

## Chunk 2: One Pair Full-Quit Path and Couch Coordinator

### Task 5: Build the shared stateful lifecycle fake and typed cleanup core

**Files:**
- Create: `cmd/internal/pairlifecycle/cleanup.go`
- Create: `cmd/internal/pairlifecycle/cleanup_test.go`
- Create: `cmd/internal/pairlifecycletest/fake.go`
- Create: `cmd/internal/pairlifecycletest/fake_test.go`
- Modify: `cmd/internal/launcher/lifecycle.go`
- Test: `cmd/internal/launcher/lifecycle_test.go`
- Modify: `cmd/internal/launcher/session_quiescence.go`
- Test: `cmd/internal/launcher/session_quiescence_test.go`
- Create: `cmd/internal/launcher/lifecycle_os.go`
- Test: `cmd/internal/launcher/lifecycle_os_test.go`
- Modify: `cmd/internal/launcher/osruntime.go` only for thin delegation

- [ ] Write `TestFakePairLifecycleStateTransitions` for this exact durable model: committed/prepared requests, exact session present/absent, ordered cleanup stage effects, per-stage injected failures, process-local in-progress ownership, immutable results, delivered-trigger count, and `Restart()` that drops only process-local state.
- [ ] Run `go test -p 20 ./cmd/internal/pairlifecycletest -run 'TestFakePairLifecycleStateTransitions' -count=1`; expect FAIL because the fake does not exist.
- [ ] Implement `FakePairLifecycle` behind the production `pairlifecycle` store/cleanup interfaces, not a test-only alternate API.
- [ ] Run the fake-state test again; expect PASS.
- [ ] Write `TestRunCleanupStageMatrix` with one failure at each mandatory stage: `session-quiescence`, `editor-reap`, `scrollback-preserve`, `sidecar-cleanup`, `poller-cleanup`, and `cmux-cleanup`. Assert this exact result shape:

```go
type StageFailure struct {
    Stage CleanupStage
    Code  FailureCode
    Err   error
}

type CleanupResult struct {
    Outcome     CompletionOutcome
    Failures    []StageFailure
    CompletedAt time.Time
}
```

- [ ] In those tests, require session-quiescence failure to stop destructive later stages; after quiescence succeeds, every later independent stage runs even if a sibling fails, failures accumulate in stage order, and success requires zero mandatory-stage failures.
- [ ] Write `TestRunCleanupIntentPolicy`: direct intent preserves exact current prompt/output/restart/nudge behavior; Couch intent performs no modal/nudge, forces scrollback preservation, and cannot succeed until all mandatory stages finish.
- [ ] Run `go test -p 20 ./cmd/internal/pairlifecycle -run 'TestRunCleanup(StageMatrix|IntentPolicy)' -count=1`; expect FAIL because `RunCleanup` does not exist.
- [ ] Implement pure orchestration `RunCleanup(ctx, intent, ops) CleanupResult` over a narrow stateful `QuitLifecycleOps`; use `cleanup_failed` for stage errors and `timeout` for context expiry while retaining stage details.
- [ ] Run the cleanup tests again; expect PASS.
- [ ] Write `TestCleanupDeadlinePropagation` with already-cancelled parent, captured outer deadline, captured Zellij child deadline, and barriers at the 5-second inner/10-second outer boundaries. Assert no operation creates `context.Background`, each operation sees the remaining parent budget, and no subprocess/observation deadline exceeds it.
- [ ] Run `go test -p 20 ./cmd/internal/launcher -run 'TestCleanupDeadlinePropagation' -count=1`; expect FAIL on the current background-based quiescence/runtime methods.
- [ ] Add focused context/error-returning OS effects in `lifecycle_os.go`; make quiescence derive `min(parent deadline, zjTimeout)` and make cleanup callers derive the one 10-second production outer deadline. Leave unrelated broad `Runtime` methods unchanged.
- [ ] Run the deadline test again; expect PASS without wall-clock sleeps.
- [ ] Write launcher regression tests that drive the same `FakePairLifecycle`: direct Alt+x output/prompt, Alt+d no-op, restart skips nudge, raw preservation/removal, exact-session cleanup, established-only resume hint, and Couch intent's no-modal forced-preserve policy.
- [ ] Run `go test -p 20 ./cmd/internal/launcher -run 'RunLaunchQuit|RunLaunchDetach|RunLaunchPark|DirectAltX|CouchQuitIntent' -count=1`; expect FAIL before launcher delegates to the typed core.
- [ ] Replace void `runCleanup` with a thin adapter to the typed core and focused OS operations; do not add cleanup logic to `createflow.go` or the existing 1,000-line `osruntime.go`.
- [ ] Run the launcher regressions again; expect PASS.
- [ ] Run `go test -p 20 ./cmd/internal/pairlifecycle ./cmd/internal/pairlifecycletest ./cmd/internal/launcher -run 'Cleanup|RunLaunchQuit|RunLaunchDetach|RunLaunchPark|SessionQuiescence' -count=1`; expect PASS.
- [ ] Commit with `git commit -m "#152: type Pair full-quit cleanup"`.

### Task 6: Version the quit intent and publish Pair completion

**Files:**
- Modify: `cmd/internal/launcher/restart.go`
- Test: `cmd/internal/launcher/restart_test.go`
- Modify: `cmd/internal/launcher/createflow.go`
- Test: `cmd/internal/launcher/createflow_test.go`
- Modify: `cmd/internal/launcher/markers.go`
- Test: `cmd/internal/launcher/markers_test.go`
- Modify: `cmd/internal/pairlifecycle/store.go`
- Test: `cmd/internal/pairlifecycle/store_test.go`

- [ ] Write `TestTypedQuitIntentCompatibility` for legacy/direct markers, versioned direct intent, exact Couch request reference, malformed reference, and unchanged compaction/restart marker behavior.
- [ ] Run `go test -p 20 ./cmd/internal/launcher -run 'TestTypedQuitIntentCompatibility' -count=1`; expect FAIL because markers are boolean/legacy-only.
- [ ] Upgrade the quit marker reader/writer to a typed direct-or-Couch intent while retaining legacy direct-marker reads; keep restart/compaction independent.
- [ ] Run the intent compatibility test again; expect PASS.
- [ ] Write `TestConsumeAttemptCriticalSection` covering uncommitted/malformed request refusal, one lock owner, duplicate triggers, existing committed result, concurrent same attempt, concurrent older/newer attempts, and crash after each cleanup stage followed by restart.
- [ ] Run `go test -p 20 ./cmd/internal/pairlifecycle -run 'TestConsumeAttemptCriticalSection' -count=1`; expect FAIL because the store has no cleanup transaction API.
- [ ] Add one store-owned API equivalent to `ConsumeAttempt(ctx, requestKey, func(context.Context, LockedAttempt) CleanupResult) (QuitCompletion, error)`. It acquires the transaction lock once, validates committed request authority, returns an existing committed result, runs/resumes one idempotent attempt, and publishes completion through non-relocking methods on `LockedAttempt` before releasing the lock.
- [ ] Require the lock to span dedupe → effective cleanup → immutable completion publication. Different attempts under one transaction serialize on the same stable inode; a crash releases it, and the next consumer resumes idempotent stage effects.
- [ ] Run the critical-section test again; expect PASS with at-least-once triggers and one effective outcome per attempt.
- [ ] Write `TestDirectAndCouchShareCleanupEffects` using `FakePairLifecycle`; assert identical ordered effective cleanup for direct and Couch paths, with only prompt policy and completion publication differing.
- [ ] Run `go test -p 20 ./cmd/internal/pairlifecycletest ./cmd/internal/launcher -run 'TestDirectAndCouchShareCleanupEffects' -count=1`; expect FAIL before Pair's outer cleanup consumes typed Couch intent.
- [ ] Wire Pair outer cleanup through `ConsumeAttempt` for Couch and direct typed cleanup without completion for direct Alt+x.
- [ ] Run the equivalence test again; expect PASS.
- [ ] Run `go test -p 20 ./cmd/internal/pairlifecycletest ./cmd/internal/launcher -run 'QuitIntent|QuitLifecycle|Direct.*Couch|Restart' -count=1`; expect PASS.
- [ ] Commit with `git commit -m "#152: share direct and Couch quit lifecycle"`.

### Task 7: Implement Couch's durable park coordinator

**Files:**
- Create: `cmd/internal/couchcore/park.go`
- Create: `cmd/internal/couchcore/park_test.go`
- Create: `cmd/internal/couchcore/parkworker.go`
- Create: `cmd/internal/couchcore/parkworker_test.go`
- Modify: `cmd/internal/couchcore/couch.go`
- Modify: `cmd/internal/couchcore/artifactcollision.go`
- Modify: `cmd/internal/couchcore/artifactcollision_fake.go`

- [ ] Write `TestParkCoordinatorOrdering` with barriers asserting requested CAS completes before request publication, request directory sync commits before exact-session trigger, and completion remains until final ThreadStore CAS.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore -run 'TestParkCoordinatorOrdering' -count=1`; expect FAIL because the coordinator does not exist.
- [ ] Add `PairLifecycleController` distinct from forced post-start `Artifacts.Quiesce`; resolve the exact indexed session, begin transaction, publish/reconcile request, trigger at least once only after authority, observe completion, finalize ThreadStore, then best-effort clean attempt files.
- [ ] Run the ordering test again; expect PASS and assert zero calls to `Artifacts.Quiesce`, direct session deletion, process scan, transcript scan, or detach.
- [ ] Write `TestParkCoordinatorTransitionMatrix` with named subtests: `request-publish-failed`, `awaiting-coalesces`, `cleanup-failed-retry-eligible`, `cleanup-failed-recover-required`, `timeout-late-success`, `session-absent-completion-missing`, `stale-completion`, `revision-conflict`, `replacement-incarnation`, `lock-sync-indeterminate`, `abandon-late-success-noop`, `older-success-closes-transaction`, and `newer-attempt-suppressed-or-obsolete`.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore -run 'TestParkCoordinatorTransitionMatrix' -count=1`; expect FAIL on the first unimplemented transition.
- [ ] Implement every subtest through the pure transition API and the shared `FakePairLifecycle`; Retry/Recover append immutable attempts, timeout keeps a late result eligible, unknown stays occupied, and tombstoned/closed results are no-ops.
- [ ] Run the transition-matrix test again; expect PASS.
- [ ] Write `TestParkCoordinatorRestartReconcilesActiveOnly`; persist each active phase, restart Couch/fake process-local state, and assert only ThreadStore-active transactions are reconciled with no corpus/process scan.
- [ ] Run the restart test; expect FAIL before constructor reconciliation.
- [ ] Reconcile only active durable park transactions during Couch construction.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore -run 'TestParkCoordinatorRestartReconcilesActiveOnly' -count=1`; expect PASS.
- [ ] Write `TestParkWorkerBoundsAndDeadlines` for one worker per address, duplicate nonce coalescing, queue capacity `<=` admission capacity, full-queue overload with zero publication/trigger effects, 999ms requested CAS success, and exactly-1s failure with occupied incarnation and zero request/trigger side effects.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore -run 'TestParkWorkerBoundsAndDeadlines' -count=1`; expect FAIL because the bounded worker/deadline shell does not exist.
- [ ] Implement the bounded queue with injected clock/barriers/deadlines; never spawn an unbounded goroutine or test fan-out.
- [ ] Run the worker/deadline test again; expect PASS without wall-clock sleeps.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore -run 'Park|Lifecycle|Restart|Admission|PostAck' -count=1`; expect PASS.
- [ ] Commit with `git commit -m "#152: coordinate verified Couch park"`.

## Chunk 3: Exact Resume and Non-Blocking Couch UI

### Task 8: Define resume eligibility and exact native-binding seam

**Files:**
- Create: `cmd/internal/couchcore/resume.go`
- Create: `cmd/internal/couchcore/resume_test.go`
- Modify: `cmd/internal/couchcore/artifactcollision.go`
- Modify: `cmd/internal/couchcore/artifactcollision_fake.go`
- Test: `cmd/internal/couchcore/artifactcollision_test.go`
- Modify: `cmd/internal/launcher/launch_args_policy.go`
- Test: `cmd/internal/launcher/launch_args_policy_test.go`

- [ ] Write `TestDecideResumeEligibilityMatrix` for verified parked success and exact typed refusals: live, creating, unknown, parking, tombstoned, legacy-unverified, missing path, missing/incomplete profile, unsupported agent, and provisional/ambiguous/unbound/established-without-root native binding. Ambiguous thread reference remains a resolver-boundary refusal.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore -run 'TestDecideResumeEligibilityMatrix' -count=1`; expect FAIL because `DecideResume` does not exist.
- [ ] Implement pure `DecideResume` returning the exact address, physical working path, saved profile, and required native ID or a stable diagnostic code.
- [ ] Add a narrow `NativeBindingResolver` whose production implementation calls #155's `sessioninventory.QuerySession` and returns only `BindingEstablished` with one exact root.
- [ ] Run the eligibility matrix again; expect PASS.
- [ ] Write `TestTrustedResumeProfileValidation` for this matrix: resume-required demands non-empty required ID, exact supported agent, non-null argv, and saved/saved source enums; it rejects explicit/path/root/repository-default sources and any ordinary/untrusted profile carrying resume-required fields.
- [ ] Run `go test -p 20 ./cmd/internal/launcher -run 'TestTrustedResumeProfileValidation' -count=1`; expect FAIL because the profile schema has no resume-required contract.
- [ ] Extend trusted Couch profile encode/decode with `resume_required`, `required_session_id`, and saved source enums; keep ordinary launch encoding unchanged.
- [ ] Run the profile validation test again; expect PASS.
- [ ] Write `TestRequiredNativeResumeBindingAtLaunch` with an established preflight ID whose point-of-launch query becomes provisional, ambiguous, missing, or a different established root. Each case must return diagnostic code `native-binding-changed`, call no child start, and never select fresh-start arguments.
- [ ] Run `go test -p 20 ./cmd/internal/launcher -run 'TestRequiredNativeResumeBindingAtLaunch' -count=1`; expect FAIL before the launch-time guard.
- [ ] Re-query exact established binding immediately before Pair creates/attaches the agent child; refuse every mismatch without fallback.
- [ ] Run the TOCTOU test again; expect PASS.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/launcher -run 'ResumeEligibility|NativeBinding|CouchLaunchProfile|RequiredSession' -count=1`; expect PASS.
- [ ] Commit with `git commit -m "#152: require exact native resume identity"`.

### Task 9: Extract same-address launch and implement Resume

**Files:**
- Create: `cmd/internal/couchcore/resume_launch.go`
- Create: `cmd/internal/couchcore/resume_launch_test.go`
- Modify: `cmd/internal/couchcore/couch.go` only for thin `Spawn` delegation
- Test: `cmd/internal/couchcore/couch_test.go` for existing-start regressions
- Modify: `cmd/internal/couchcore/admission.go`
- Test: `cmd/internal/couchcore/admission_test.go`
- Modify: `cmd/internal/couchcore/threadstore.go`
- Test: `cmd/internal/couchcore/threadstore_test.go`
- Modify: `cmd/internal/launcher/thread_claim.go`
- Test: `cmd/internal/launcher/thread_claim_test.go`

- [ ] Write `TestResumeLaunchExactProfileMatrix` with explicit Claude, Codex, Agy, and Muse rows; for each include empty argv and argv containing resume-looking tokens. Assert byte/order-exact argv, same scope/tag, exact `WorkingPath` subdirectory, saved native ID, and zero calls to tag allocation or path/root/repository defaults.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore -run 'TestResumeLaunchExactProfileMatrix' -count=1`; expect FAIL because only new-address `Spawn` exists.
- [ ] Extract the blocked-helper → acknowledgement → registration → live promotion → registry insertion tail into `resume_launch.go`, parameterized by an already chosen exact address/profile; leave `couch.go` with thin new-thread allocation/profile resolution followed by this helper.
- [ ] Add `Couch.Resume(address)` preflight/current-policy CAS and invoke the shared tail with `pair resume <same-tag> --layout2` plus the required native ID.
- [ ] Run the exact-profile matrix again; expect PASS.
- [ ] Write `TestExistingCouchResumeClaimMatrix` with exact established success and reserved, missing, malformed, wrong-scope, and wrong-tag refusal. Assert it never releases, recreates, or adopts a marker.
- [ ] Run `go test -p 20 ./cmd/internal/launcher -run 'TestExistingCouchResumeClaimMatrix' -count=1`; expect FAIL because claim registration has no exact re-entry mode.
- [ ] Implement the explicit existing-Couch-resume claim mode.
- [ ] Run the claim matrix again; expect PASS.
- [ ] Write `TestResumeAdmissionAndRollbackMatrix` with rows: policy refusal, fork failure, acknowledgement definitely undelivered, acknowledgement possibly delivered, registration missing, live-promotion CAS failure, and success. Assert exact durable phase/incarnation and whether Runner/cleanup ran.
- [ ] Require rollback to verified parked only after both the exact helper/process group and exact Pair session are proved absent. Possibly delivered acknowledgement, missing registration without absence proof, promotion failure after registration, and child exit alone remain occupied/unknown.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore -run 'TestResumeAdmissionAndRollbackMatrix' -count=1`; expect FAIL before resume-specific admission/rollback transitions.
- [ ] Implement verified-parked → creating admission on the same record, count all other occupied incarnations, preserve rollback evidence, clear verified park only on successful registration, and update latest profile with the same bytes.
- [ ] Run the rollback matrix again; expect PASS.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/launcher -run 'Resume|ExistingThread|ThreadAddress|Admission|LaunchProfile' -count=1`; expect PASS.
- [ ] Commit with `git commit -m "#152: resume exact Couch thread"`.

### Task 10: Declare park/resume operations and intercept Couch Alt+x

**Files:**
- Modify: `cmd/internal/couchcore/ops.go`
- Modify: `cmd/internal/couchcore/operationdispatch.go`
- Test: `cmd/internal/couchcore/ops_declarations_test.go`
- Test: `cmd/internal/couchcore/operationdispatch_test.go`
- Modify: `cmd/internal/couchcmd/run.go`
- Test: `cmd/internal/couchcmd/run_test.go`
- Modify: `cmd/internal/workbenchshortcut/shortcut.go`
- Test: `cmd/internal/workbenchshortcut/shortcut_test.go`
- Modify: `cmd/internal/couchtty/keys.go`
- Test: `cmd/internal/couchtty/keys_test.go`
- Create: `cmd/internal/couchtty/operation_queue.go`
- Create: `cmd/internal/couchtty/operation_queue_test.go`
- Modify: `cmd/internal/couchtty/console.go` only for thin event-loop wiring
- Test: `cmd/internal/couchtty/console_test.go`
- Modify: `cmd/internal/couchtty/panel.go`
- Test: `cmd/internal/couchtty/panel_test.go`

- [ ] Write `TestParkResumeAreOnlyNewOperations` and `TestNoCouchDetachSurface`: assert declarations/dispatch/panel controls add only park/resume, Park modes are exactly normal/retry/recover/abandon, and no detach declaration, dispatcher case, panel control, or Couch interception exists.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchtty -run 'Test(ParkResumeAreOnlyNewOperations|NoCouchDetachSurface)' -count=1`; expect FAIL before declarations/wiring.
- [ ] Declare/wire park and resume through the shared operation table. Park is live-owner/process/confirmation-required; Resume is live-owner/process/no-confirmation; recovery values are validated park modes.
- [ ] Run the operation-surface tests again; expect PASS.
- [ ] Write `TestResumeOperationResolutionBoundary`: CLI Resume combines `ref` with implicit current repo scope; zero/multiple matches refuse before eligibility, admission, defaults, tag allocation, or Runner; one match passes the exact composite address unchanged; panel/actor calls accept only their trusted implicit exact address and cannot override scope/tag; Park recovery modes do not change resolution.
- [ ] Run `go test -p 20 ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/couchtty -run 'TestResumeOperationResolutionBoundary' -count=1`; expect FAIL before dispatcher resolution is wired.
- [ ] Implement unique reference resolution at the operation executor boundary and keep `Couch.Resume` address-only.
- [ ] Run the resolution-boundary test again; expect PASS.
- [ ] Write `TestAltXCanonicalEncodingContract` and `TestInterceptorAltXFraming`: canonical legacy ESC-x and Kitty CSI-120;3u both hit Park; every split point preserves partial framing and exact before/rest; switch/Alt+x hits differ; exported encodings are defensive; Couch source contains no copied Alt+x literals.
- [ ] Add bracketed-paste subtests splitting paste markers and each Alt+x sequence at every boundary; all bytes must pass through unchanged inside paste.
- [ ] Run `go test -p 20 ./cmd/internal/workbenchshortcut ./cmd/internal/couchtty -run 'Test(AltXCanonicalEncodingContract|InterceptorAltXFraming)' -count=1`; expect FAIL before the encoding query/typed hit.
- [ ] Export a defensive canonical chord-encoding query and extend `Interceptor` with typed switch-vs-Park hits without duplicating byte literals.
- [ ] Run the encoding/framing tests again; expect PASS.
- [ ] Write `TestParkUIEventOrdering`: `parking…` is the first next-turn host write, no chord byte reaches the child, ThreadStore work begins afterward, ordinary input and child output continue across a blocked Park, duplicates coalesce, and overload returns one typed result with zero lifecycle side effects.
- [ ] Run `go test -p 20 ./cmd/internal/couchtty -run 'TestParkUIEventOrdering' -count=1`; expect FAIL because `runOp` blocks the host-writer loop.
- [ ] Implement bounded request/result state in `operation_queue.go`; wire only enqueue/result events into `console.go`. Route actor Alt+x and confirmed panel Park to the same declared operation and keep Alt+d pass-through.
- [ ] Run the UI ordering test again; expect PASS.
- [ ] Write `TestChildExitNeverProvesPark` with exit before request publication, while awaiting, after completion/before final CAS, and after finalization. Only matching completion plus final CAS releases capacity; restart reconciles the retained-completion case.
- [ ] Run `go test -p 20 ./cmd/internal/couchtty ./cmd/internal/couchcore -run 'TestChildExitNeverProvesPark' -count=1`; expect FAIL on current exit/forget behavior.
- [ ] Make child exit a routing event only and delegate durable reconciliation to the coordinator.
- [ ] Run the exit-authority test again; expect PASS.
- [ ] Run `go test -p 20 ./cmd/internal/workbenchshortcut ./cmd/internal/couchtty ./cmd/internal/couchcore ./cmd/internal/couchcmd -run 'AltX|Interceptor|Park|Resume|Operation|Panel' -count=1`; expect PASS.
- [ ] Commit with `git commit -m "#152: wire non-blocking park and resume UI"`.

## Chunk 4: Conformance, Performance, and Documentation

### Task 11: Complete stateful and live lifecycle conformance

**Files:**
- Test: `cmd/internal/pairlifecycletest/fake_test.go`
- Create: `cmd/internal/launcher/quit_lifecycle_live_test.go`
- Modify: `cmd/internal/launcher/session_quiescence_live_test.go` only to extract shared controlled-Zellij setup
- Modify: `.github/workflows/couch-zellij-conformance.yml`
- Modify: `Makefile`

- [ ] Write `TestQuitLifecycleConformanceScenarioFake` using one reusable scenario driver and redacted `EffectTrace`. Cover committed request, at-least-once duplicate delivery, ordered idempotent effects, immutable completion, restart after every durable boundary, late older-attempt success, and tombstoned-result no-op.
- [ ] Run `go test -p 20 ./cmd/internal/pairlifecycletest -run '^TestQuitLifecycleConformanceScenarioFake$' -count=1`; expect FAIL before the shared scenario/trace exists.
- [ ] Implement the reusable scenario driver and extend `FakePairLifecycle` until its one expected redacted trace passes.
- [ ] Run the fake conformance test again; expect PASS.
- [ ] Write opt-in `TestQuitLifecycleLive` that executes the same scenario once against `FakePairLifecycle` and once against a controlled real Pair/Zellij driver, redacts PIDs/paths/timestamps, and requires exact trace equality rather than independent assertions.
- [ ] Add a subprocess checkpoint after completion final-file rename and before directory sync: wait for readiness, kill the holder, then require the next process to acquire the stable inode lock, validate the exact payload, directory-sync it, and return one immutable committed result.
- [ ] Give every subprocess and controlled Zellij session a context deadline and `t.Cleanup` that kills/reaps the process, removes the exact session, and closes PTYs even on failure; reuse `session_quiescence_live_test.go` setup instead of adding another Zellij parser.
- [ ] Run `PAIR_LIVE_COUCH=1 go test -p 20 ./cmd/internal/launcher -run '^TestQuitLifecycleLive$' -count=1 -v`; expect FAIL before the real lifecycle driver is complete.
- [ ] Implement the real driver through the production request/store/trigger/cleanup/completion seams and holder-death recovery.
- [ ] Run the same live command again; expect PASS with equal fake/real traces and bounded teardown.
- [ ] Add the live test to `make test-couch-zellij-live`. Update workflow path filters explicitly for `cmd/internal/pairlifecycle/**`, `cmd/internal/pairlifecycletest/**`, `cmd/internal/artifactpath/**`, `cmd/internal/launcher/lifecycle*.go`, `session_quiescence*.go`, `markers*.go`, `restart*.go`, `cmd/internal/couchcore/park*.go`, this live test, `Makefile`, and the workflow itself.
- [ ] Run `go test -p 20 ./cmd/internal/pairlifecycletest -count=1`; expect PASS before the opt-in live probe.
- [ ] Run `PAIR_LIVE_COUCH=1 go test -p 20 ./cmd/internal/launcher -run '^TestQuitLifecycleLive$' -count=1 -v`; expect PASS.
- [ ] Commit with `git commit -m "#152: add park lifecycle conformance"`.

### Task 12: Measure the operating envelope and update architecture maps

**Files:**
- Create: `cmd/internal/couchtty/park_latency_test.go`
- Modify: `README.md`
- Modify: `atlas/couch.md`
- Modify: `atlas/session-identity.md`
- Modify: `atlas/architecture.md`
- Modify: `workshop/issues/000152-couch-verified-park-resume.md`
- Modify: every project returned by `sdlc project find --issue pair#152`

- [ ] Write opt-in `TestParkLatencySmoke`. With `PAIR_PARK_LATENCY_SMOKE=1`, verify/report the target as macOS Apple M2 Max, run 100 sequential samples under the ordinary-development-co-tenancy assumption, timestamp handler entry/first host write/requested-commit completion, and compute nearest-rank P95 after sorting each series.
- [ ] Use a real temporary ThreadStore/lifecycle store but fake the external Pair trigger. Require feedback P95 <100 ms, commit P95 <100 ms, commit max <1 second, and one prefilled-queue overload sample that refuses with no request/trigger. Print `feedback_p95=<duration> commit_p95=<duration> commit_max=<duration> overload=refused`; skip ordinary heterogeneous CI when the env flag is absent.
- [ ] Run `PAIR_PARK_LATENCY_SMOKE=1 go test -p 20 ./cmd/internal/couchtty -run '^TestParkLatencySmoke$' -count=1 -v`; expect FAIL before the measured UI/commit path is wired.
- [ ] Implement only the measurement harness needed around the already deterministic production path; correctness remains enforced by ordinary barrier/clock tests.
- [ ] Run the latency command again on the operator's M2 Max under normal development load; expect PASS and the four reported metrics within budget.
- [ ] Document that adversarial OS starvation is outside the claim, overload fails immediately/occupied, queue size is bounded by admission, cleanup has a 10-second default with the existing 5-second exact-Zellij inner wait, and resume adds no full inventory scan.
- [ ] Update README's generated/audited Couch operation and panel control surfaces with Park/Resume and no detach.
- [ ] Map lifecycle authority, prepared/committed files, advisory lock, transaction/attempt/recovery matrix, exact claim/native-binding resume, and #135's shared proof in `atlas/`; keep `atlas/index.md` unchanged unless a new page is added.
- [ ] Change all five original `## Plan` rows in #152 from `[ ]` to `[x]`, and all six rows under the authoritative acceptance/implementation revision from `[ ]` to `[x]`. Append a revision/log mapping the superseded “process inventory” wording to the delivered shared Pair lifecycle rather than rewriting that historical text.
- [ ] Run `sdlc project find --issue pair#152`; record every returned project path and update its #152 scope/detail notes now. Let `sdlc close` later tick/upsert the final actual/closed fields.
- [ ] Run `go test -p 20 ./cmd/internal/couchcmd ./cmd/internal/couchcore ./cmd/internal/couchtty -run 'README|Operation|Panel' -count=1`; expect PASS.
- [ ] Run `go test -p 20 ./cmd/internal/pairlifecycle ./cmd/internal/pairlifecycletest ./cmd/internal/artifactpath ./cmd/internal/threadrecord ./cmd/internal/launcher ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -count=1`; expect PASS.
- [ ] Run `go test -p 20 ./... -count=1`; expect PASS.
- [ ] Run `make test-lua`; expect PASS.
- [ ] Run `bash tests/term-pane-shortcuts-test.sh`; expect PASS.
- [ ] Run `bash tests/review-toggle-test.sh`; expect PASS.
- [ ] Run `zellij --config-dir zellij setup --check`; expect PASS.
- [ ] Run `zellij setup --dump-layout zellij/layouts/main-2.kdl >/dev/null`; expect exit 0.
- [ ] Run `zellij setup --dump-layout zellij/layouts/main-3.kdl >/dev/null`; expect exit 0.
- [ ] Run `git diff --check`; expect no output.
- [ ] Run the opt-in live lifecycle command from Task 11; expect PASS.
- [ ] Run the opt-in latency command from this task; expect PASS on the M2 Max.
- [ ] Smoke `pair-dev` under Couch: actor-focused Alt+x must paint `parking…` before teardown; Resume must return to the same tag and #155 native root. Record the tag/root before and after in #152's Log.
- [ ] Commit with `git commit -m "#152: document verified park and resume"`.

### Task 13: Close and ship #152 before beginning #151

**Files:**
- Modify: `workshop/issues/000152-couch-verified-park-resume.md`
- Modify: any referencing Couch project file found by `sdlc project find --issue pair#152`

- [ ] Run `sdlc actual --issue 152` to preview measured active time; do not type an estimate from memory.
- [ ] Run `sdlc close --issue 152 --verified 'focused lifecycle/operation suites and go test -p 20 ./... passed; Lua/terminal/Zellij config+layout checks passed; fake-vs-live Pair/Zellij lifecycle trace passed; M2 Max 100-sample park latency smoke met feedback/commit budgets; manual Couch Alt+x parked then resumed the same tag/native root'`.
- [ ] Let `sdlc close` dispatch the sole fresh-context boundary review and mutate #152 plus every discovered project to `codecomplete`; do not run a redundant review.
- [ ] Branch on the exact gate output. On `SHIP`, proceed. On `FIX-THEN-SHIP`, fix the full finding family, add a preventative rule to `workshop/lessons.md`, update tests/atlas/issue Log, and rerun the affected verification before committing; do not blindly rerun close. On a refusal, follow its emitted next action, including rerunning close only when it requires a new reviewed anchor.
- [ ] Commit all finalized codecomplete mutations, any finding fixes, lessons, issue/plan ticks, and project bookkeeping together with `git commit -m "#152: close verified park and resume"`; `sdlc close` does not commit.
- [ ] Run `sdlc state` and inspect the current branch. If it is `main`, run `sdlc push --yes`. Otherwise run `sdlc pr`, confirm the branch is pushed/clean, then run `sdlc merge --yes`.
- [ ] Verify the publish gate changed `codecomplete → done`, archived #152 and this plan under `workshop/history/`, updated every referencing project, returned the surviving checkout to clean `main`, and left no #152 feature worktree/branch.
- [ ] Only after that proof, claim and start-plan #151.

## Revisions

### 2026-08-29 — plan-quality compression (PQ-1)

**Reason:** the first `change-code` plan-quality pass accepted the architecture
but found that the task bodies above enumerate individual cases and restate the
diff procedure. The gate requires exact production functions plus one
adversarial or mechanical test strategy for each risky function family.

**Delta:** this compact execution map supersedes the procedural task bodies
above. The goal, architecture, core-concept tables, ordering invariants,
operating envelope, file ownership, and runnable verification commands above
remain authoritative. Implementation follows these function-level units in
order, using red-green-refactor and one commit per numbered unit; the old task
bodies remain only as append-only design history (`PQ-1`, `ARCH-PURE`,
`ARCH-PURPOSE`).

#### Chunk 1 — durable lifecycle foundation

1. Add `pairlifecycle.ValidateQuitRequest`,
   `pairlifecycle.ValidateQuitCompletion`, and
   `pairlifecycle.MatchQuitCompletion`, plus `artifactpath.LifecyclePaths`.
   Test strategy: strict/fuzzed malformed identities and path components must
   fail before derivation, while a mechanical import/manifest sweep prevents
   lifecycle filename reconstruction or a dependency from the leaf package
   back into launcher/Couch.
2. Add `pairlifecycle.Store.PublishRequest`, `PublishCompletion`, and
   `Reconcile`, backed by the stable advisory lock.
   Test strategy: one failure-injected atomic-publication harness exercises each
   IO boundary, concurrent identical/conflicting writers, and subprocess death;
   only an immutable validated final whose directory sync is established may
   become committed authority.
3. Add pure `couchcore.AdvanceParkTransaction` and
   `couchcore.MonotonicLastActiveAt`; add `threadrecord.DecodePersisted` and
   lifecycle validation/conversion for schema v2.
   Test strategy: generated event sequences and hostile v1/v2 payloads enforce
   increasing attempts, separate base/current revisions, active-nonce and
   tombstone invariants, late-success monotonicity, defensive copies, and
   fail-closed unknown versions without IO mocks.
4. Add `ThreadStore.BeginPark`, `AdvancePark`, `AppendParkAttempt`,
   `FinalizePark`, and `AbandonPark` on the existing journal/CAS boundary.
   Test strategy: replay and competing-writer barriers verify every mutation is
   one CAS, finalization alone removes the exact incarnation, and restart or a
   stale revision cannot release admission or lose tombstones.

Run after this chunk:

```bash
go test -p 20 ./cmd/internal/pairlifecycle ./cmd/internal/artifactpath ./cmd/internal/threadrecord ./cmd/internal/couchcore -run 'Lifecycle|Quit|Park|ThreadRecord|Journal|Admission' -count=1
```

#### Chunk 2 — one Pair full-quit path and Couch coordinator

5. Add `pairlifecycle.RunCleanup`, context-aware
   `launcher.quiesceZellijSession`, and a thin typed `launcher.runCleanup` over
   `QuitLifecycleOps`; provide the stateful `pairlifecycletest.Fake` through the
   same seam.
   Test strategy: fake effect traces and injected context barriers compare
   direct Alt+x with Couch cleanup and mechanically enforce effect ordering,
   idempotence, the 10-second outer/5-second inner deadline relationship, and
   Alt+d's unchanged detach behavior.
6. Add typed `launcher.ReadQuitIntent`/`WriteQuitIntent` compatibility and
   `pairlifecycle.Store.ConsumeAttempt`.
   Test strategy: malformed/legacy intents and crash-restart checkpoints around
   the single locked critical section prove at-least-once delivery yields one
   effective cleanup result and immutable completion per attempt.
7. Add `couchcore.PairLifecycleController.Park`, `Retry`, `Recover`, and
   `Abandon`, with `parkWorker.Submit` for bounded admission.
   Test strategy: a transition-sequence driver plus barriers mechanically
   asserts ThreadStore request commit precedes publication, directory authority
   precedes trigger, completion precedes final CAS, duplicates coalesce, queue
   overload has zero effects, and the 1-second pre-side-effect deadline fails
   occupied without wall-clock sleeps or corpus/process scans.

Run after this chunk:

```bash
go test -p 20 ./cmd/internal/pairlifecycle ./cmd/internal/pairlifecycletest ./cmd/internal/launcher ./cmd/internal/couchcore -run 'Cleanup|Quit|Park|Lifecycle|SessionQuiescence|Restart|Admission' -count=1
```

#### Chunk 3 — exact resume and non-blocking Couch UI

8. Add pure `couchcore.DecideResume` and the
   `NativeBindingResolver.ResolveEstablished` seam; extend
   `launcher.ValidateTrustedLaunchProfile` and
   `launcher.RequireNativeResumeBinding`.
   Test strategy: adversarial durable states/profiles and a binding changed
   between preflight and launch must all refuse with stable diagnostics before
   child start, default lookup, tag allocation, or fresh-session fallback.
9. Extract `couchcore.launchExistingThread`, implement `Couch.Resume`, and add
   `launcher.RegisterExistingCouchThread`.
   Test strategy: table-generated agents/argument bytes and failure barriers
   prove the same composite address, working path, saved profile, and required
   native root cross the launch boundary unchanged; rollback to parked is
   allowed only after exact helper and Pair-session absence are proved.
10. Declare/dispatch only `park` and `resume`; export
    `workbenchshortcut.ChordEncodings`; extend `couchtty.Interceptor.Feed`; and
    add bounded `operationQueue.Submit`/result handling.
    Test strategy: mechanical declaration/source sweeps forbid Couch detach and
    copied Alt+x bytes, every input split and bracketed-paste framing is tested,
    and event-loop barriers prove `parking...` is the first next-turn host write
    while input/output continue and child exit never acts as park proof.

Run after this chunk:

```bash
go test -p 20 ./cmd/internal/workbenchshortcut ./cmd/internal/couchtty ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/internal/launcher -run 'AltX|Interceptor|Park|Resume|Operation|Panel|NativeBinding|LaunchProfile' -count=1
```

#### Chunk 4 — conformance, operating envelope, and shipment

11. Add `pairlifecycletest.RunConformanceScenario` and live
    `launcher.TestQuitLifecycleLive` through the same store/cleanup seam.
    Test strategy: compare one redacted fake/real state trace, including
    duplicate delivery and holder death after rename, with deadline-bound
    subprocess/Zellij cleanup so the live probe cannot leak resources.
12. Add opt-in `couchtty.TestParkLatencySmoke` and update README, atlas, issue,
    and discovered project records.
    Test strategy: 100 sequential M2 Max samples under ordinary development
    co-tenancy require feedback P95 below 100 ms, requested-commit P95 below
    100 ms, every commit below 1 second, and overload refusal with no side
    effects; ordinary deterministic tests continue to enforce ordering.
13. Run the full verification and SDLC close/publish commands already specified
    above. Keep a single Go test command active at a time and `-p 20`; do not
    begin #151 until #152 is archived on clean `main` (`ARCH-CONSTRAINTS`).

Non-goals remain: Couch does not own PID inventory/teardown, expose detach,
scan transcript/process corpora on interaction paths, support rollback of v2
records into old binaries, provision missing worktrees (#153), or implement
#151's menu beyond the shared operation declarations required here.
