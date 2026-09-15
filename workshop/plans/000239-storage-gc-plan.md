# Pair and Couch storage GC implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expire inactive standalone Pair data and archived Couch threads after 60 days, while preserving switcher-visible threads and data currently in use.

**Architecture:** Put exact artifact ownership in artifactpath, pure retention decisions in a small storagegc package, and filesystem coordination in its IO adapters. Managed users and GC share coordination; Couch continues to own its manifest and archive transactions. Preview and automatic collection use the same decision engine.

**Tech Stack:** Go, existing file-backed stores and advisory locks, Lua managed editor entry points, deterministic filesystem/process integration tests.

**Approval:** Feature policy approved on 2026-09-13; this technical plan awaits review and operator approval. The issue's approved revision supersedes its original speculative policies.

## Chunk 1: Policy, ownership, and implementation

### Product contract

- Eligible means 60 complete days since meaningful use, collected on the next successful sweep. It is not deletion at an exact wall-clock instant.
- Every address in any registered Couch manifest protects its associated Pair data, including parked and unreadable entries. Archiving starts a fresh 60-day grace. Later meaningful use extends that grace.
- Standalone Pair drafts, queues, prompt history, ledger, captures and recovery artifacts expire together. No change to agent-native stores or repository files.
- A successful managed open/resume/view and a changed authored-content write count. Inventory, passive redraw, diagnostic appends, unchanged saves, and background distillation do not. External editors and arbitrary file reads cannot be observed.
- Existing data without explicit metadata gets a new 60-day grace on initialization; never infer inactivity from atime or old mtime. Preview neither initializes nor recovers storage by writing.
- `pair gc` previews with retained/eligible/untracked/blocked reasons and logical bytes; `pair gc --apply` initializes clocks and collects eligible groups. Automatic sweeps are opportunistic, at most daily, while Pair or Couch is running.
- Size-based truncation is excluded: this does not promise a global disk ceiling or immediate recovery of the measured 14.8 GiB. Visible Couch data and frequently used sessions may continue growing.

### Core concepts

| Name | Kind | Lives in | Status |
|---|---|---|---|
| StorageOwner | PURE | cmd/internal/artifactpath/gc.go | new |
| ArtifactGroup | PURE | cmd/internal/artifactpath/gc.go | new |
| ActivityRecord / RetentionDecision | PURE | cmd/internal/storagegc/policy.go | new |
| CollectionTransaction / CollectionEvent / ReduceTransaction | PURE | cmd/internal/storagegc/transaction_model.go | new |
| StoreRegistry | PURE | cmd/internal/storagegc/stores.go | new persisted value |

- **StorageOwner:** canonical Pair root, explicit legacy-or-scoped namespace, and validated tag. One owner has many artifacts. Reuse Paths/LegacyPaths and checked constructors; never parse display names or split tag/agent strings heuristically. Scope/tag address maps to the same owner for Couch and Pair. Additional artifact families widen the manifest, not independent globs (ARCH-DRY).
- **ArtifactGroup:** exact recognized files/directories for one owner, with identity evidence and exclusions. Raw capture and offset events are inseparable. Shared scope metadata, defaults, bindings and catalogs are not owned by a thread. Ambiguous or unknown ownership blocks the group; report unknown global entries separately.
- **ActivityRecord:** version, owner, incarnation ID, initialized-at, last-meaningful-use. One bounded JSON record per owner; archive grace is separate Couch-owned evidence. Decoding rejects unsupported versions, mismatched identities and impossible timestamps. Pure decision uses max(initialized-at, last-use, archive-grace) and an injected clock; future times retain.
- **RetentionDecision:** a tagged result: protected, live, grace, eligible, untracked, blocked. `Decide(now, evidence)` performs no IO. Incomplete inventory and ambiguous liveness dominate expiry; all references must be accounted for. Multiple archives use the newest grace and all visible references protect.
- **CollectionTransaction:** persisted phases are `prepared`, `detached`, and `finalized`. `ReduceTransaction` accepts proved-detachment and owner-retirement events without changing frozen source/destination identities, owner, incarnation, bucket or archive receipts. One pending transaction per owner prevents overlapping incarnations; source names are never revisited after detachment. Pure transition tests cover rejected events, idempotent completed steps and non-regressing sequences.

| Name | Kind | Lives in | Status | Wraps |
|---|---|---|---|---|
| Coordinator / Locked | INTEGRATION | cmd/internal/storagegc/coordinator.go | new | stable root lock, activity/intents, registry publication and process leases |
| Collector | INTEGRATION | cmd/internal/storagegc/collector.go, transaction.go | new | metadata inventory, reducer-backed quarantine effects and recovery |
| ProcessProbe / OSProcessProbe | INTEGRATION | cmd/internal/storagegc/process.go | new | process-birth identity and liveness evidence |
| ThreadStore | INTEGRATION | cmd/internal/couchcore/threadstore.go, retention.go | modified | coordinated references, archive grace and restore |
| ProcessLease / AcquireSelectedProcess | INTEGRATION | cmd/internal/storagegc/lease.go | new | managed reader/writer process lifetimes |
| Coordinator.BeginUse / CompleteUse / WriteChanged / RecoverUse | INTEGRATION | cmd/internal/storagegc/coordinator.go, use.go | new | durable content intents and meaningful-use clocks |
| gccmd.Run | INTEGRATION | cmd/internal/gccmd/run.go | new | public flags, migration, preview and apply |
| Scheduler / ScheduleWorker | INTEGRATION | cmd/internal/storagegc/schedule.go | new | bounded scheduled batches and worker lifetime |

Coordinator/Collector are thin shells around pure decisions, using portable temporary roots in tests. ProcessProbe has a stateful fake with alive/dead/unknown identities and explicit barriers; a local child-process conformance test checks real lease/process behavior. No network/service dependencies or credentials. Existing wrapper/session evidence supplements leases for pre-upgrade processes; a bare stale PID never proves death or life.

### Coordination and recovery contract

One stable root coordination lock, outside collectible groups, orders activity publication, live reservations, Couch membership changes and collection. Lock order is retention coordinator before Couch store.lock; never call back into the coordinator from a held Couch lock. Metadata reads for an apply occur under coordination and normal Couch recovery/locks. Preview takes read snapshots and reports pending recovery as blocked.

Managed launch reserves a lifetime lease under coordination before spawning; the supervised owner holds the lease until all owned writers exit. Attach, editor and viewer lifetimes are also leased, so a reader open for 61 days stays protected. Child-start failure releases reservation. Hand-offs transfer ownership before releasing the old lease; unknown or unsupported legacy runtime ownership retains. Meaningful use is recorded once per explicit action, without treating periodic lease checks as use. Foreground viewer closure records the end of that explicit viewing interval; background reader/writer lease release never refreshes the clock. A dormant live session is protected while alive but does not earn fresh use merely by exiting.

Short content writes acquire coordination before access. BeginUse durably journals an intent (owner incarnation, operation ID, start time and exact target) before permitting changed content to be written; failure to persist/fsync intent refuses the content effect. CompleteUse publishes activity durably before retiring intent. Outstanding intent always blocks GC, including process death between content persistence and activity publication. RecoverUse conservatively publishes use at the intent time, retaining a live operation until its owner exits; then retires the intent. Recovery does not guess whether content was unchanged after a crash. All managed write entrypoints use this protocol; a failed cleanup cannot erase protection. Long readers acquire their lease before opening paths. Locks/leases remain on stable inodes outside the deletable namespace.

Concrete lease ownership: the launcher holds a durable pending-start reservation, not sole lifetime protection. `wrapcmd.Run` and `sessionwatch.Run` register their own PID plus OS process-birth identity before artifact access and retain that record for their entire process lifetime. The Lua retention helper registers Neovim's own process identity through the internal CLI before managed storage is opened; the record survives the short CLI process. Explicit opener/viewer commands register the actual viewer child before access, or keep their own lease while waiting for a foreground child. Protection is based on these durable per-process registrations plus ProcessProbe, not parent survival or inherited advisory locks. Registration writes take the root lock; GC rechecks process identities under that lock. Clean exit removes its registration; death is reaped only on verified identity mismatch/death. Unsupported process-birth inspection reports unknown and retains.

Each pane acknowledges registration before the launcher clears its pending-start reservation. If the launcher dies before acknowledgement, recovery consults the existing Zellij/session evidence and registered child identities; unresolved starts remain blocking, with a reason in preview. Detached children remain protected by their own records even if Couch, launcher or supervisor dies. No extra supervisor/heartbeat daemon is introduced. Test the actual child-survives-parent ordering with a local subprocess, in addition to the stateful process fake.

Every Couch constructor that associates a store with a Pair root registers its canonical store path under coordination before publishing references. Default and configured COUCH_STORE_DIR are registered during migration. Registry read failure or an unavailable registered store blocks collection for the root. Pre-upgrade custom stores cannot be discovered from default paths: migration must present the registered store list and require explicit configuration of any additional stores before destructive GC is enabled. This is a one-time migration completeness requirement, not an inference from file age. A root-level migration-complete marker gates automatic/apply deletion; initialization and preview remain available before completion.

ArchiveThread journals its archive timestamp sidecar with the archive copy, manifest removal and record removal, including undecodable records. Use the same transaction machinery for GC removal of archived records and sidecars. A supported restore transaction coordinates manifest re-add and retains archive grace until the restored reference is durable; manual file moves are outside concurrent-GC guarantees.

Collection under the coordinator: recover pending transactions; reread all protection/use/live evidence; enumerate exact members with lstat; reject symlinks, unexpected types and identity changes; journal a unique quarantine destination; rename members on the same filesystem; durably detach the namespace; then remove quarantine contents. Large bytes are never copied into metadata journals. Launch waits for recovery and creates a new incarnation only after old detachment. Shared binding entries are filtered via a locked exact-owner rewrite; derived inventory caches are invalidated under their owning API, never interpreted as deletion authority.

Custom stores can be on different filesystems. Pair quarantine lives under the Pair root; Couch archives remain small metadata handled through their own store journal, never renamed into Pair quarantine. The durable root transaction first records all registered store paths, expected archive identities and child operation IDs; then detaches Pair members; then removes each matching archive and sidecar via that store's journal, leaving an operation receipt until root finalization; finally marks the root transaction detached. Recovery finishes these steps before permitting launch, restore, unregister or a new incarnation. If a store is unavailable, retain the pending transaction and block that owner's reuse; never discard evidence or replay against a replacement archive. Store receipts are removed after root finalization. Pair rename returning EXDEV leaves a blocked transaction without a copy-and-delete fallback. Tests use injected EXDEV and separate portable roots to exercise interruption at each cross-store step.

| State/event | Result/effect |
|---|---|
| eligible + use before final lock | retain; activity/lease wins |
| eligible + archive/restore/membership update | serialized fresh decision, newest evidence wins |
| prepared/detaching + crash | recover exact transaction before owner can reopen; keep incomplete members inaccessible |
| detaching + IO error/cancel | leave journal blocking use/GC until recovery; report error |
| detached + new same-tag owner | new incarnation allowed; cleanup touches quarantine only |
| detached + cleanup error | retry quarantine removal, never source paths |
| duplicate collector | nonblocking lock failure skips optional sweep; explicit apply reports busy |
| unknown process/store/metadata | retain with actionable reason |

### Operating envelope and lifecycle

ARCH-CONSTRAINTS: metadata-only inventory, no scans of multi-GiB log contents. Initial measured workload is roughly 1,000 files/14.8 GiB; test 100,000 filenames. Automatic work uses one worker, a 2-second scheduling budget and at most 100 owner groups per run, with a persisted continuation cursor. No optional sweep on a keystroke or before initial UI readiness. Lock attempts are nonblocking for automatic GC; a group's detachment checks cancellation between filesystem steps and leaves recoverable state. Filesystem calls can exceed the scheduling budget; do not claim a hard realtime deadline. Resume partial sweeps while running; mark a daily pass complete only at the end. Worker is tied to the launcher/Couch context and joined at shutdown.

ARCH-FUNERAL: activity records and leases end with the collected owner; dead lease entries are reaped after verified process death. Lease lock files are removed only with the root lock held after no holder exists. Archive timestamp sidecars end with archive removal/restore. Transactions and quarantines end after verified cleanup, with at most one pending transaction per owner. Registry is one entry per store, explicit unregister only after proving no remaining references; cursor and migration state are fixed-size replace-in-place records. The root lock inode persists (one per root). No per-poll history or unbounded sweep log is added.

ARCH-SECURE: persisted manifests/registry/retention/journals are untrusted input; validate at decoding and fail closed. Resolve only configured owned roots; use exact validated relative paths and no symlink traversal. Test roots must be explicitly injected and must never fall back to real HOME. No credentials are read or minted.

ARCH-PURE: policy and transaction state transitions remain deterministic; filesystem and clock values enter as evidence. ARCH-MOCK: folder-backed tests and stateful process fake cover the real seams. ARCH-ORDER: the transition table and scheduler barriers exercise competing actors and death. ARCH-PURPOSE: all known artifact producers/readers and both Couch/standalone policies are covered; diagnostic caps are an explicit scope revision, not claimed delivered.

### M1 — Activity, ownership and protection (no deletion)

#### Task 1: Exact ownership and pure retention policy

Files: create `cmd/internal/artifactpath/gc.go`, `gc_test.go`, `cmd/internal/storagegc/policy.go`, `policy_test.go`; modify `cmd/internal/artifactpath/manifest.go` and `coverage_test.go`.

- [x] Add failing MatchArtifact and Decide tests using the function strategies below. Add a mechanical coverage guard requiring every family to state collectable/shared/protected and its exact constructor/parser.
- [x] Run `go test ./cmd/internal/artifactpath ./cmd/internal/storagegc -count=1`; expect missing GC symbols/tests to fail.
- [x] Implement typed owner and exact inventory descriptors, reusing Paths/LegacyPaths. Do not reuse OwnsTagArtifact or RenameArtifacts as deletion authority.
- [x] Implement Decide and its time/protection strategy below; keep the policy pure.
- [x] Run those packages to PASS and commit `#239: define exact storage ownership and retention policy`.

#### Task 2: Coordination, migration and Couch timestamps

Files: create `cmd/internal/storagegc/coordinator.go`, `stores.go`, `process.go` and corresponding `_test.go`; create `cmd/internal/couchcore/retention.go`, `retention_test.go`; modify `threadstore.go`, `couchcmd/run.go`.

- [x] Write failing Coordinator/StoreRegistry/ProcessProbe tests using the function strategies below.
- [x] Run `go test ./cmd/internal/storagegc ./cmd/internal/couchcore ./cmd/internal/couchcmd -count=1`; expect new contract assertions to fail.
- [x] Implement Coordinator and registry/migration gating; move every membership-mutating ThreadStore entry through one coordinated seam. Enumerate callers mechanically with rg so archive, creation, registration and restore cannot bypass it.
- [x] Journal archive grace alongside existing archival effects. Add a typed restore transaction coordinating manifest re-add and archive removal, with no new UI requirement; document that manual recovery requires stopping managed processes and GC.
- [x] Run the ArchiveThread and RestoreThread strategy below; require PASS and commit `#239: coordinate retention with Couch membership and live owners`.

#### Task 3: Meaningful-use integration

Files: create `cmd/internal/storagegc/use.go`, `use_test.go`, `nvim/retention.lua`, `nvim/retention_test.lua`; modify `cmd/internal/launcher/createflow.go`, `lifecycle.go`, `cmd/internal/couchcore/resume.go`, `cmd/internal/pairlog/runcli.go`, `cmd/internal/opener/run.go`, `cmd/internal/scrollbackcmd/scrollbackcmd.go`, `cmd/internal/orientation/model.go`, `cmd/internal/wrapcmd/wrap.go`, `cmd/internal/sessionwatch/run.go`, `nvim/init.lua`, `nvim/scrollback.lua`; add integration tests alongside the modified Go entrypoints and `tests/retention-test.sh`.

- [x] Write failing BeginUse/CompleteUse/RecoverUse and managed-entrypoint integration tests using the function strategies below.
- [x] Run the affected package tests and `bash tests/retention-test.sh`; expect missing integration assertions to fail.
- [x] Wire resolved owner context into managed entrypoints and propagate explicit identity through orientation to parked readers. Use one lease/touch API; expose internal CLI operations for Lua through the existing dispatcher contract. Compare content before publishing a use event; do not read log payloads for GC.
- [x] Cover all managed content readers/writers by a checked call-site inventory, including programmatic and explicit saves. Verify generic history scans, refresh/statusline reads, diagnostics and distiller writes stay non-use operations. Their file access still needs a lease when racing collection.
- [x] Run managed-use and lifetime-registration strategies below; require affected Go/Lua tests to PASS; update `atlas/` and its index for ownership/use/lock contracts; commit.
- [ ] Close M1 through `sdlc milestone-close --issue 239 --milestone M1` with recorded verification; fix the binary's review findings. No live deletion is enabled by M1.

### M2 — Safe collection, CLI and scheduled sweeps

#### Task 4: Recoverable group deletion

Files: create `cmd/internal/storagegc/transaction.go`, `transaction_test.go`, `collector.go`, `collector_test.go`; extend `cmd/internal/couchcore/retention.go`, `cmd/internal/launcher/session_index.go` through their owning APIs.

- [x] Write failing ReduceTransaction and Collector.Apply tests using the function strategies below.
- [x] Run `go test ./cmd/internal/storagegc ./cmd/internal/couchcore ./cmd/internal/launcher -count=1`; expect collection contract failures.
- [x] Implement exact quarantine transactions and recovery, revalidation under coordination, coordinated archive removal, exact binding cleanup and cache invalidation. Preserve original evidence on error; never retry deletion against source names after detachment.
- [x] Exercise the Collector.Apply race/fault strategy below, including raw/event group visibility after recovery.
- [x] Run tests and `go test -race ./cmd/internal/storagegc ./cmd/internal/couchcore ./cmd/internal/launcher`; require PASS and commit `#239: collect expired owner groups with recoverable detachment`.

#### Task 5: Commands, automatic scheduling and end-to-end verification

Files: create `cmd/internal/gccmd/run.go`, `run_test.go`, `cmd/internal/storagegc/schedule.go`, `schedule_test.go`; modify `cmd/pair-go/main.go`, `cmd/internal/dispatcher/dispatcher.go`, dispatcher coverage tests, launcher and Couch lifecycle entrypoints, `Makefile`, `atlas/index.md`; create `atlas/storage-retention.md`.

- [x] Write failing gccmd.Run tests using the function strategies below.
- [x] Implement public `pair gc` routing/help and explicit migration completion with registered-store summary. Apply works only after migration completeness is established. Report counts/logical bytes and reasons; no payload contents in diagnostics.
- [x] Implement a context-bound, joined worker after UI/session readiness, daily completion clock and bounded resumable cursor. Never count sweep discovery as use; initialize legacy records only on mutating runs.
- [x] Implement the Scheduler.Run bounded-work strategy below; assert one worker and bounded group processing, not brittle elapsed-time thresholds.
- [x] Run `go test ./... -count=1`, `go test -race ./cmd/internal/storagegc ./cmd/internal/gccmd ./cmd/internal/couchcore ./cmd/internal/launcher`, `make test-lua`, `bash tests/retention-test.sh`, and `git diff --check`; require PASS.
- [x] Run real-store preview only and record measured eligible/protected/untracked totals without reading content or deleting live files. Run apply against an isolated representative fixture containing old standalone, visible parked, newly archived, expired archived and live sessions; assert the expected survivors and second-apply idempotence.
- [ ] Document 60-day clocks, migration, managed-use boundary, custom stores, commands, retained-error diagnostics and no global size ceiling; update atlas index and issue Log. Commit and close M2 with the binary review gate.
- [ ] Close #239 with measured actuals and verification, then publish through `sdlc pr` and `sdlc merge`. Do not run destructive real-store migration/apply as a test; automatic collection begins only after migration completion and the full legacy grace.

## Revisions

### 2026-09-13 — fresh plan review

The reviewer found two ordering gaps: custom Couch stores can be on another filesystem, and launcher lifetime alone cannot protect detached children. Added a cross-store journal/receipt ordering contract and named actual wrapper, watcher, editor and viewer process registrations, including parent-death and EXDEV tests. These refine the mechanism without changing the approved retention policy (ARCH-ORDER, ARCH-PURPOSE). Re-review approved this chunk with no remaining concrete blockers; operator technical-plan approval remains pending.

### Acceptance evidence

Function-level ownership tests guard exact membership and parser ambiguity; pure policy tests guard the entire decision precedence and time boundary. Stateful integration tests guard each IO seam, crash step and actor ordering. CLI fixture snapshots prove preview has no writes. The real preview validates inventory at observed scale without treating old timestamps as authorization to delete. Successful tests plus both SDLC milestone reviews and final close are required before claiming completion.

### 2026-09-13 — operator approval

Operator approved implementation with “go ahead”; earlier pending-approval
notes are historical. Proceed through change-code and both review boundaries.

### 2026-09-13 — plan gate PQ-1 and PQ-2

PQ-1 addressed by durable BeginUse intent before every managed content effect,
with fail-closed intent creation, recovery, and retirement after activity is
published. PQ-2 addressed by replacing procedural test inventories with the
function strategies below. Acceptance commands remain in their tasks.

### Function-level test strategies

| Production function | Adversarial input class and mechanical guard |
|---|---|
| artifactpath.MatchArtifact | Arbitrary filenames and overlapping owner/agent identities: fuzz constructor round-trips and reject ambiguous matches; exhaustive Families coverage guard. |
| artifactpath.NewStorageOwner | Untrusted root/scope/tag components: fuzz path confinement and namespace separation. |
| storagegc.DecodeActivity | Truncated, hand-edited and future-version bytes: fuzz decoder; no invalid record becomes expiry evidence. |
| storagegc.Decide | Arbitrary clocks and conflicting protection evidence: property-test time boundary and monotonic protection without IO. |
| storagegc.ReduceTransaction | Interrupted/duplicate/out-of-order transaction events: enumerate state/event product; prohibit source deletion after detachment. |
| Coordinator.BeginUse / CompleteUse / RecoverUse | Process death and IO failure at each durable boundary: stateful filesystem fault injection proves intent precedes content and survives until activity is durable. |
| Coordinator.RegisterProcess / ReleaseProcess | Parent-child lifetime interleavings and PID reuse: stateful process fake plus child-survives-parent subprocess conformance proves protection follows actual children. |
| ProcessProbe.Inspect | Alive/dead/unknown OS identities: injected backend and real-child conformance agree; unavailable identity never means dead. |
| StoreRegistry.Register / CompleteMigration | Missing, aliased, malformed and unavailable roots: folder-backed state machine tests keep deletion disabled unless inventory is complete. |
| ThreadStore.ArchiveThread / RestoreThread | Invalid records and interruption at each journal effect: existing store fault hooks prove reference/grace conservation. |
| AcquireSelectedProcess / Coordinator.WriteChanged | Explicit use versus background access and unchanged writes: stateful entrypoint tests verify activity deltas; checked producer/consumer inventory guards missing integrations. |
| Collector.Preview | Arbitrary portable trees and pending transactions: before/after filesystem snapshot equality proves read-only behavior. |
| Collector.Apply / Recover | Concurrent use, namespace replacement and failure at every detach/cross-store step: barrier schedules and fault injection prove unrelated/new incarnations survive; EXDEV fails closed. |
| gccmd.Run | Invalid/public/internal command forms and migration states: dispatcher route coverage plus isolated end-to-end fixture assertions. |
| Scheduler.Run | Oversized inventories, cancellation and repeated starts: injected clock/budget with 100,000 names proves bounded groups, one worker and cursor progress. |

### 2026-09-13 — operator separates diagnostic retention

The operator requests a separate **7-day debugging-log bucket**. This
supersedes exclusion of writer rotation and blanket 60-day diagnostic
retention. Session/recovery data keeps its approved 60-day meaningful-use
policy and Couch visibility protection. Diagnostic retention is generation
age: viewing a trace or keeping its thread visible does not extend it.

Debugging families: wrap-events, adapt flight recorder, optional PAIR_WRAP_LOG
and PAIR_SLUG_LOG, Couch COUCH_TRACE, COUCH_INPUT_TRACE and COUCH_MOUSE_TRACE.
Raw scrollback plus offset events, parked captures, prompt logs, ledger,
changelog and drafts remain session/recovery data. The record/state distinction
matters more than a filename containing "log".

Add a shared `diagnosticlog.Writer` in `cmd/internal/diagnosticlog/writer.go`
and pure `DiagnosticSegment`/`DecideSegment` in `policy.go`. Keep the current
trace pathname for compatibility. Rotate into unique segments in an adjacent
managed directory after 24 hours or 64 MiB; the byte threshold only segments,
never discards young data. Closed segments expire seven days after their last
write (at most one extra day of record-age slack). An idle current file can
expire after seven days under writer coordination; upgraded writers reopen
on their next append. Old processes without this protocol retain their open
files until they stop, rather than writing into unlinked storage.

Legacy diagnostic mtime measures generation age, so stopped diagnostic files
over seven days old do not need the session onboarding grace. Still require
exact ownership and inactive-writer evidence. Explicit external trace paths
are registered only by their writer, including their exact segment directory;
never discover arbitrary /tmp logs by glob. Invalid/unavailable registered
paths retain. Registration ends after writer, current file and segments are
gone; unknown pre-upgrade external paths cannot be collected automatically.

All writers take a stable per-log lock nonblocking before append/rotate;
optional logging skips on contention/error instead of blocking terminal IO.
Verify current inode after acquiring the lock before using a cached fd, so
another process's rotation cannot strand writes in old segments. Collector
takes the same lock. Root coordinator precedes diagnostic lock; no inverse
acquisition. Segment cleanup uses the bounded scheduler, never unbounded work
in the terminal path. Persist rotation intent before rename; recovery under
the same lock completes/recognizes the exact rename before any next append.
No extra daemon. Metadata and lock lifetimes end with the managed log after
all writers have gone. Reuse artifactpath/procutil (ARCH-DRY), pure age policy
(ARCH-PURE), explicit rotation states (ARCH-ORDER), bounded writer work
(ARCH-CONSTRAINTS) and one retention constant (ARCH-PURPOSE/ARCH-FUNERAL).

This adds diagnostic collection to **M2**; M1 still enables no deletion.
Before M2 final verification:

- [x] `DecideSegment`: arbitrary age/size/future clocks → boundary/property tests ensuring younger data survives and Couch visibility does not override debug expiry.
- [x] `Writer.Append`/`Rotate`: concurrent writers and crashes at each durable boundary → portable filesystem barriers/fault injection prove young-segment preservation and current-inode reopen.
- [x] `CollectSegments`: malformed metadata, old writers and substituted external paths → exact identity/no-symlink guards and preview snapshot equality.
- [x] Implement policy/writer/registry, then wire wrapcmd/wrap.go, adapt/adapt.go, slugcmd/slugcmd.go, couchtty/trace.go, nvim/adapt.lua, bin/lib/adapt-log.sh and launcher adaptation initialization. Shell/Lua use one internal append command through dispatcher rather than direct file appends.
- [x] Verify all emitters with their existing schema/trace tests, diagnosticlog race tests and counted hot-path work; exercise live rotation and stopped legacy eligibility.
- [x] Extend preview with 7-day diagnostic and 60-day session buckets; document segment-age slack and exact external-path scope. Run original acceptance commands plus `go test -race ./cmd/internal/diagnosticlog` and `bash tests/adapt-schema-test.sh`.

Metadata-only sizing on 2026-09-13: wrapper diagnostics 13.017 GiB total,
11.538 GiB last written over seven days ago, versus 0.321 GiB over sixty.
These are candidate sizes, not verified reclaimable bytes.

### 2026-09-13 — diagnostic appendix review disposition

The review raised three ordering gaps, addressed as follows. Rotation and
expiry both require absence of legacy/unknown writers. For Pair-owned logs,
use complete owner/session process evidence plus upgraded writer registrations;
for configured external logs, registration records each contributing Pair
root and process, and an injected open-file inspection seam (stateful fake,
local subprocess conformance) must agree that no unregistered process holds
the file. Any older relevant Pair/Couch runtime without registration blocks
management of its trace path even if it opens that path only transiently.
Unverifiable external legacy traces are retained, not adopted by assumption.
Concurrent manual or third-party writes are outside the managed protocol and
must cause detected identity/mtime inconsistency to retain rather than prune.

Persist generation start independently of last-write time, in rotation
metadata shared across reopen/restart. Continuous low-volume appends do not
move that clock. The current generation rotates on elapsed start time; its
closed generation's last-write time determines seven-day expiry. Missing
legacy generation metadata is initialized conservatively while preserving
the old file; never claim its mtime proves start time.

Per-log coordination inodes are deliberately stable and never unlinked by
GC, including after payload collection. Each is a zero-length file, one inode
per distinct canonical trace path, named next to that exact path so multiple
Pair roots use the same lock. This small persistent cost prevents a paused
opener acquiring a stale lock after cleanup; registry and segment metadata
still disappear after final collection. Every upgraded writer resolves the
canonical path before opening that shared lock. Explicitly test a paused
opener across cleanup and two roots sharing one external trace path.

Add Writer.Rotate tests for mixed old/new writers and continuous writes across
restart; add lock/open-file conformance tests alongside the existing barrier
suite. These are mechanism corrections, not retention-policy changes.

### 2026-09-13 — operator expires parked captures independently

Each parked capture is a separate immutable retention unit, expiring 60 days
after capture even while its parent tag remains active or Couch-visible.
This supersedes parked captures' inclusion in the tag-wide protection rule.
Delete the raw file and companion offset events together; do not truncate
either in place. An active reader postpones deletion until it finishes but
does not renew capture age. Parent tag use/visibility is not capture use.

Add `CaptureRetention` to artifactpath and pure `DecideCapture` to storagegc.
New ParkScrollback writes explicit captured-at metadata with its exact raw/
events identity; producer completes publication before releasing reservation.
Existing timestamped captures can use conservative legacy capture evidence:
the latest of both file mtimes and the filename timestamp interpreted with
the latest possible timezone offset (+14h to a UTC interpretation). Refuse
invalid names, future timestamps, mismatched pairs and uncertain ownership.
This avoids guessing last meaningful use or delaying every known old capture
by a fresh session-onboarding grace. Recognized single raw captures are valid
when no events were produced; orphan events without raw remain blocked.

Implement independent capture-group preview/collection in M2, reusing exact
quarantine transactions and actual-reader registrations. Managed parked-read
entrypoints register the exact capture in addition to owner identity; owner
process liveness alone need not retain all old captures. Older processes with
uncertain capture access conservatively block affected captures until stopped.
`DecideCapture` tests arbitrary capture clocks/liveness with boundary and
monotonic-protection properties; capture collector tests paired detachment,
legacy timezone bounds, and a reader racing expiry through scheduler barriers.

Reporting now separates **7-day diagnostics**, **60-day parked captures**,
and **60-day inactive-tag session data**. The third bucket retains old content
while its tag keeps receiving meaningful use; it does not prune individual
prompt or ledger rows. No other retention-policy change is implied.

### 2026-09-13 — agreed final parked-capture policy

Operator confirmed 7 days for old parked captures, superseding the preceding
60-day capture revision. Expiry follows each capture’s age, independent of
tag activity and Couch visibility; raw and event sidecars are removed together.
Active readers or incomplete handoffs defer deletion. Session data remains
60 days since meaningful use; Couch-visible threads remain protected until
archival, which starts a fresh 60-day grace.


### 2026-09-14 — Resume and boundary reconciliation

The checkpoint implemented M1 and M2 together before either review. This was a
workflow deviation: there was no earlier no-deletion implementation boundary.
The first milestone review will therefore see the combined implementation;
M2 still requires its acceptance fixture, real preview, and final integration
review. Neither review is retroactively claimed. Runtime collection remains
migration-gated, and no real-store apply/migration is authorized as a test.

Merged newer checkpoint/recovery/shortcut behavior while retaining lease and
acknowledgment ordering. Full Go and race checks pass after integration; native
Neovim caught a top-level-local limit, resolved by narrowing existing helper
scope. Public apply acceptance and a checked managed-I/O inventory complete the
outstanding technical verification rather than relying on package tests alone.


### 2026-09-14 22:34 PDT — BR-1/BR-5: retirement and enforced transaction model

Reason: the M1 boundary review found that an eligible owner with only activity
metadata aborted collection, and that the documented pure reducer did not exist.
The Core concepts tables above now name actual production symbols and their
PURE/INTEGRATION kinds. They replace the earlier planned four-phase enumeration
and conceptual managed-use/CLI type labels; earlier revision records remain intact.

`CollectionTransaction` remains schema version 1. The existing persisted phase
values are retained; no journal migration or extra intermediate phase is needed.

| Current phase | Completion event | Next phase | Effects authorized by the adapter |
|---|---|---|---|
| prepared | detachment-proved | detached | Before the event: detach exact archive receipts and rename exact source identities into quarantine; verify the resulting tree |
| detached | detachment-proved | detached | Idempotent event only; do not repeat source-name effects |
| detached | owner-retired | finalized | Before the event: clean bindings and retire only matching session incarnation metadata; captures do not retire their owner |
| finalized | owner-retired | finalized | Idempotent event; remaining work only forgets receipts and removes exact quarantine entries/journal |

All other phase/event combinations fail without mutation. The filesystem adapter
publishes the reduced state before replacing its in-memory value. An AST guard
rejects direct production phase assignments outside the reducer. Cancellation
checks before effects preserve the journal for a later coordinated recovery.

Empty session items may enter this same transaction path only when the collector
has established `Eligible` under coordination. Empty capture transactions remain
invalid. Visible owners, live/unknown readers and unfinished handoffs retain
metadata. Independent capture expiry can leave newly initialized activity behind;
a later successful sweep beyond its sixty-day grace now retires it normally.

Verification: `/tmp/pair239-br1-red.log` reproduced the original post-capture
empty-collection failure. `/tmp/pair239-br1-br5-focused.log` passed the pure
phase/event matrix, depth-eight event sequences, bypass guard, metadata protection
matrix, six interrupted-retirement boundaries and five cancellation/resume
boundaries. A regressive reducer mutation failed in
`/tmp/pair239-br5-mutation.log`. Full storagegc race tests passed 14.875s in
`/tmp/pair239-br1-br5-race.log`; the final transaction-focused run passed 5.203s
in `/tmp/pair239-br1-br5-final-focused.log`. These address BR-1/BR-5; they do not
claim the other M1 findings or the boundary gate are closed.


### 2026-09-14 — M1 review recovery and maintenance corrections

BR-2/BR-3: unpublished JSON now stages centrally under `.retention/pending`
without becoming authoritative. Exact legacy pending names are skipped by reads
and removable under coordination; killed-publisher tests cover complete and
partial writes before rename. Confirmed dead pre-spawn reservations retire during
recovery and before admission limits. Spawned uncertainty requires explicit
parent/child death evidence plus the reachable `resolve-start` acknowledgment;
live or unknown identities still retain. Unchanged recovery no longer rewrites
owner state (ARCH-FUNERAL, ARCH-ORDER).

BR-4: apply recovers Couch journals before its discovery snapshot and onboards
missing archive grace only for selected owners. Each journal witnesses the exact
archive identity and grants a full sixty days; malformed existing sidecars remain
blocked. Preview remains read-only (ARCH-PURPOSE).

BR-6: automatic `ApplyPage` limits visited owners rather than deletions and
persists the last owner key, including pre-migration and retained owners. It
shares one immutable payload inventory only inside an uninterrupted root lock
while refreshing clocks/references. Explicit `Apply` still scans past retained
owners and limits collections, avoiding repeated-command starvation. Discovery
checks cancellation between entries and caps payload and owner metadata inventory;
no partial scan becomes deletion authority. The two-second cooperative scheduler
budget, nonblocking root acquisition, context checks between journal/delete
steps, and bounded recovery yield preserve retryable state. Diagnostic options
carry the same maintenance context. The scheduler retries contention and budget
expiry with the old durable cursor; it does not claim a hard syscall deadline or
guarantee that an oversized inventory completes within two seconds.

Production tests use seven owners and counted probes/persists to prove every
pre-migration page advances within its owner budget; 100,000 real filenames prove
bounded owner effects and incomplete-inventory retention. Separate deadline,
busy-lock and canceled-effect tests prove yielding and recovery. This replaces
the earlier deletion-count-only claim; review remains pending.


### 2026-09-14 23:07 PDT — M1 round2 class completion

The second review addressed BR-1/BR-2/BR-3/BR-5, retained BR-4/BR-6, and raised
BR-7 (the interrupted-publication family). Remaining requirements:

- Owner discovery derives namespaces from validated durable references and owner
  metadata as well as physical directories. Legacy/tracked archives and
  metadata-only scoped owners remain discoverable with no payload directory;
  unsafe existing directories and invalid owner identities still retain/refuse.
- Diagnostic discovery shares exact path classification but does not evaluate
  session clocks or process liveness. Each diagnostic page proves its own writer
  and generation safety. All nested maintenance store locks must yield rather
  than block behind a foreground writer; journal effects must check cancellation
  as well as lock acquisition.
- Publish the prepared transaction journal before creating its unique quarantine
  directory. Replay creates a missing directory from validated journal authority.
  Repeated failed publication, cancellation, real process death, durable sync
  failure and unsafe replacement tests cover the ordering. Unknown preexisting
  unjournaled directories have no deletion authority and remain untouched.

Pre-publication artifact enumeration (ARCH-ORDER, ARCH-FUNERAL): fixed root
infrastructure (`.retention`, owners/transactions/pending/quarantine parent
folders, coordinator/scheduler lock inodes) has bounded per-root cardinality.
All Pair JSON publications stage under the coordinated pending directory and
receive bounded recovery after death. Transaction IDs remain memory-only until
their journal is published; only that durable journal authorizes the subsequent
unique quarantine. Couch coordinated journal/record/archive/grace/receipt
publications also require a bounded recoverable stage, separately from unlocked
continuation materialization; implementation and killed-publisher tests are in
progress. Do not sweep generic `.thread-store-*` files used by unlocked writers.

Hosted conformance uncovered two bootstrap assumptions: a sibling Makefile and
preinstalled Go. The workflow now invokes Makefile.local and provisions the
module-declared toolchain. Isolated-workflow regressions reproduce both failures.
A subsequent hosted Zellij detach fixture timed out; its exact live group passes
locally on the same version, so one hosted retry is in progress rather than an
unsubstantiated timeout increase.


### 2026-09-14 23:12 PDT — Publication class and maintenance envelope implemented

Coordinated Couch publications now use one reserved `.thread-store-publication`
file per store under its existing exclusive lock. This covers journal creation,
journal entry targets (records/manifests/preferences/archive/grace/receipts), and
the existing successful-start record update. Recovery removes only that exact
regular staging file, then replays any durable journal. Generic unlocked
continuation publication remains separate; its temporary names are never swept.
The helper rejects unsafe staging types and cross-store publication targets;
rename failure retains journal authority rather than copying across filesystems.

Maintenance passes its cancellation check through journal commit, replay, each
entry effect and staging publication; cancellation leaves any durable journal
for retry. Tests kill actual publishers after stage fsync for journal, archive,
grace and receipt targets, proving no target publication occurred early and
that replay completes precisely. Cancellation after staged fsync and between
journal entries is covered independently. Targeted Couch publication/retention/
archive race passed11.295s. Pair transaction full race passed14.745s.

Diagnostic production-page race passed1.973s with zero session probes/writes,
while expired debug data is collected and cursor advances. Namespace tests cover
both absent and unsafe directories, plus all legacy/tracked/metadata-only owner
variants. Hosted conformance passed on its retry, with no timeout or test change.
Full-tree and focused integration race runs are in progress on the combined
round3 tree; the third milestone review has not run yet.

### 2026-09-14 23:30 PDT — Complete diagnostic recovery and cancellation classes

M1 round3 addressed BR-4 but kept BR-6/BR-7 and added BR-8. Diagnostic
registry, state and segment metadata must have explicit publication ownership,
not just the Pair and Couch journals audited in the previous revision:

| Publication | Synchronization and recovery authority |
| --- | --- |
| Pair activity, leases, starts, stores, runtimes, captures, transactions and scheduler | Root coordinator; central `.retention/pending` recovery |
| Couch journal and its record/manifest/preferences/archive/grace/receipt targets | Couch store lock; exact reserved publication stage and durable journal replay |
| Diagnostic registry entry | Registry lock shared by publishers/retirement; exact `.metadata-publication` stage, explicit unfiltered pagination cursor/completion |
| Diagnostic current state and segment generation metadata | Per-log lock; one central `.metadata-publication` stage per log, recovered before subsequent log operations |
| Unlocked continuation materialization | Independent publisher; not part of retention's coordinated staging authority |

The registry lock follows the per-log lock where both are needed. Registry-only
recovery must not acquire per-log locks, and automatic maintenance must yield on
contention. Preview stays read-only. Tests must distinguish an unpublished
reserved stage from another publisher's live work and from unknown legacy names.
Pagination completion comes from traversal progress rather than the number of
valid entries returned, so skipped temporary entries cannot truncate discovery.

Every nested diagnostic process inspection inherits the worker context,
including subprocess deadlines and per-process traversal. Session recovery's
source/quarantine walks and pending-owner lookup must also check that context.
The budget is cooperative between filesystem calls, not a hard syscall timeout.

Diagnostic deletion's durable intent remains authoritative through payload
removal, exact generation metadata removal, each ancestor removal and intent
retirement. Replay must accept already-completed removals while refusing unsafe
replacement ancestors or mismatched metadata. Fault/cancellation regressions
cover those boundaries, including repeat replay after partial cleanup.

The next review will use `WF_BOUNDARY_ROUND_CAP=10` to keep Important findings
blocking beyond the default third round; this tightens acceptance rather than
silently demoting remaining findings. No review or ledger gate is waived.


### 2026-09-14 23:44 PDT — Diagnostic class fixes verified

Registry pages now return entries, next offset and explicit completion, checking
cancellation during offset skipping and each row. Registry mutations use a
permanent lock; automatic contention yields with the existing retry semantics.
All diagnostic publications stage centrally per owning log/registry under
`.metadata-publication`, with cancellation before effects and rename. Registry
and retirement callbacks carry the operation context. Legacy numeric per-log
stages retain their existing locked cleanup authority; old registry temporary
names remain untouched and cannot truncate the inventory. Radix directories
are fixed-depth bounded infrastructure, not per-generation orphan artifacts.

Killed-publisher tests cover registry entries, current state and segment metadata,
including a concurrent collector refusing the live lock. A staged publication
cancelled before rename leaves the prior target intact. Deletion replay tests
cover every effect boundary and reject replacement payload/metadata/ancestors.
Process-proof tests enforce worker deadlines within both lsof and ps, and no
subsequent inspection after cancellation. Storage recovery tree walks now check
the same held context. Combined diagnostic/runtime/CLI race passes in
`/tmp/pair239-round4-integration-race.log`; storage full race passes in
`/tmp/pair239-round4-storage-race.log`. Final full Go run remains in progress.


Final combined verification passed: full Go suite
`/tmp/pair239-round4-full-go-final.log` (diagnostic26.920s,
runtime132.773s including100kfixture, Couch160.748s), diagnostic/runtime/CLI
race and storage race above. Registry cancellation/lock cursor tests also passed
race after that snapshot. No real operator store was modified.

Conservative boundary retained: death during the first-ever current-log state
publication can leave a current file without authoritative metadata. Collection
refuses that payload; the next managed Open initializes its identity under the
log lock. This is intentionally not authority to infer and delete unknown data.
Reserved staging still has a recovery owner; no new per-attempt stage survives
recovery. The missing-metadata refusal remains visible in previews/apply errors.

Hosted conformance investigation located an external Zellij0.45.1 server panic
before FirstClientConnected, at zellij-server/src/lib.rs:1462. Diagnostic excerpt
is `/tmp/pair239-zellij-server-panic-evidence.log`. The fixture probes list-sessions
while startup is still underway; investigating whether this probe triggers the
premature RemoveClient event before changing readiness behavior.

The refreshed `sdlc actual` measurement reports0.38h with historical mention
fallback warnings (`/tmp/pair239-round4-actual.log`); it does not cover this
long implementation/review session reliably or isolate milestone increments.
Subsequent closes will use the precise `--no-actual` exception, recording N/A
and excluding velocity calibration instead of presenting this partial number
as total effort or inventing increments. Verification and review gates remain.

### 2026-09-14 — Hosted conformance startup race resolved

Zellij's list-sessions connection probe can trigger RemoveClient before session
initialization in0.45.1. The disposable fixture now waits for server-rendered
cursor-position/reset output before probing and accepts only an exact live row.
Client setup bytes alone are insufficient. Fake subprocess regressions cover
that ordering, early exit and deadline failure; helper full race passes2.249s.
Final fresh-config live group passes3 repetitions16.426s
(`/tmp/pair239-ci-render-live-green.log`). No production terminal behavior or
operator session changed. This test-only delta follows the full Go pass above.
