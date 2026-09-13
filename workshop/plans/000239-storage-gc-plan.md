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

| Name | Lives in | Status |
|---|---|---|
| StorageOwner | cmd/internal/artifactpath/gc.go | new |
| ArtifactGroup | cmd/internal/artifactpath/gc.go | new |
| ActivityRecord | cmd/internal/storagegc/policy.go | new |
| RetentionDecision | cmd/internal/storagegc/policy.go | new |
| CollectionTransaction | cmd/internal/storagegc/transaction.go | new |

- **StorageOwner:** canonical Pair root, explicit legacy-or-scoped namespace, and validated tag. One owner has many artifacts. Reuse Paths/LegacyPaths and checked constructors; never parse display names or split tag/agent strings heuristically. Scope/tag address maps to the same owner for Couch and Pair. Additional artifact families widen the manifest, not independent globs (ARCH-DRY).
- **ArtifactGroup:** exact recognized files/directories for one owner, with identity evidence and exclusions. Raw capture and offset events are inseparable. Shared scope metadata, defaults, bindings and catalogs are not owned by a thread. Ambiguous or unknown ownership blocks the group; report unknown global entries separately.
- **ActivityRecord:** version, owner, incarnation ID, initialized-at, last-meaningful-use. One bounded JSON record per owner; archive grace is separate Couch-owned evidence. Decoding rejects unsupported versions, mismatched identities and impossible timestamps. Pure decision uses max(initialized-at, last-use, archive-grace) and an injected clock; future times retain.
- **RetentionDecision:** a tagged result: protected, live, grace, eligible, untracked, blocked. `Decide(now, evidence)` performs no IO. Incomplete inventory and ambiguous liveness dominate expiry; all references must be accounted for. Multiple archives use the newest grace and all visible references protect.
- **CollectionTransaction:** prepared, detaching, detached, cleaned; exact source/destination identities and generation, never broad deletion patterns. One transaction per owner incarnation; enables retry without deleting a newer same-tag incarnation. Pure transition tests enumerate interrupt/retry cases.

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| Coordinator | cmd/internal/storagegc/coordinator.go | new | stable root lock, activity and lifetime leases |
| StoreRegistry | cmd/internal/storagegc/stores.go | new | registered Couch namespaces for this Pair root |
| Collector | cmd/internal/storagegc/collector.go | new | metadata inventory and quarantine IO |
| ProcessProbe | cmd/internal/storagegc/process.go | new | process identity and existing session evidence |
| ThreadStore | cmd/internal/couchcore/threadstore.go | modified | archive grace and coordinated membership changes |
| ManagedUse | cmd/internal/storagegc/use.go | new | selected launch/view/write operations |
| GCCLI | cmd/internal/gccmd/run.go | new | preview/apply, bounded sweep scheduling |

Coordinator/Collector are thin shells around pure decisions, using portable temporary roots in tests. ProcessProbe has a stateful fake with alive/dead/unknown identities and explicit barriers; a local child-process conformance test checks real lease/process behavior. No network/service dependencies or credentials. Existing wrapper/session evidence supplements leases for pre-upgrade processes; a bare stale PID never proves death or life.

### Coordination and recovery contract

One stable root coordination lock, outside collectible groups, orders activity publication, live reservations, Couch membership changes and collection. Lock order is retention coordinator before Couch store.lock; never call back into the coordinator from a held Couch lock. Metadata reads for an apply occur under coordination and normal Couch recovery/locks. Preview takes read snapshots and reports pending recovery as blocked.

Managed launch reserves a lifetime lease under coordination before spawning; the supervised owner holds the lease until all owned writers exit. Attach, editor and viewer lifetimes are also leased, so a reader open for 61 days stays protected. Child-start failure releases reservation. Hand-offs transfer ownership before releasing the old lease; unknown or unsupported legacy runtime ownership retains. Meaningful use is recorded once per explicit action, without treating periodic lease checks as use. Foreground viewer closure records the end of that explicit viewing interval; background reader/writer lease release never refreshes the clock. A dormant live session is protected while alive but does not earn fresh use merely by exiting.

Short content writes acquire coordination before access, persist changed content and activity before releasing it. A failed activity commit after a successful write creates blocking recovery evidence, not an old valid timestamp. Long readers acquire their lease before opening paths. Locks/leases remain on stable inodes outside the deletable namespace.

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

- [ ] Add failing ownership tests spanning all Families, legacy/scoped collisions, overlapping tags, hyphenated agents, malformed suffixes, directories, unknown families and raw/event pairs. Add a mechanical coverage guard requiring every family to state collectable/shared/protected and its exact constructor/parser.
- [ ] Run `go test ./cmd/internal/artifactpath ./cmd/internal/storagegc -count=1`; expect missing GC symbols/tests to fail.
- [ ] Implement typed owner and exact inventory descriptors, reusing Paths/LegacyPaths. Do not reuse OwnsTagArtifact or RenameArtifacts as deletion authority.
- [ ] Implement Decide with direct unit cases for 60-day boundary ±1ns, missing/invalid/future clocks, archive later than use, use later than archive, visible parked/unreadable/multiple-store references, live/unknown owner and incomplete inventory. Fuzz monotonicity: adding protection must never turn retained into eligible.
- [ ] Run those packages to PASS and commit `#239: define exact storage ownership and retention policy`.

#### Task 2: Coordination, migration and Couch timestamps

Files: create `cmd/internal/storagegc/coordinator.go`, `stores.go`, `process.go` and corresponding `_test.go`; create `cmd/internal/couchcore/retention.go`, `retention_test.go`; modify `threadstore.go`, `couchcmd/run.go`.

- [ ] Write failing folder-backed tests for initialization grace, malformed metadata, registry incompleteness, unavailable custom stores, process death/PID reuse, lease handoff and lock ordering.
- [ ] Run `go test ./cmd/internal/storagegc ./cmd/internal/couchcore ./cmd/internal/couchcmd -count=1`; expect new contract assertions to fail.
- [ ] Implement Coordinator and registry/migration gating; move every membership-mutating ThreadStore entry through one coordinated seam. Enumerate callers mechanically with rg so archive, creation, registration and restore cannot bypass it.
- [ ] Journal archive grace alongside existing archival effects. Add a typed restore transaction coordinating manifest re-add and archive removal, with no new UI requirement; document that manual recovery requires stopping managed processes and GC.
- [ ] Run tests for both valid and undecodable archives, each injected journal failure, duplicate archive/retry, visible unreadable rows, and two registered stores sharing an owner. All PASS; commit `#239: coordinate retention with Couch membership and live owners`.

#### Task 3: Meaningful-use integration

Files: create `cmd/internal/storagegc/use.go`, `use_test.go`, `nvim/retention.lua`, `nvim/retention_test.lua`; modify `cmd/internal/launcher/createflow.go`, `lifecycle.go`, `cmd/internal/couchcore/resume.go`, `cmd/internal/pairlog/runcli.go`, `cmd/internal/opener/run.go`, `cmd/internal/scrollbackcmd/scrollbackcmd.go`, `cmd/internal/orientation/model.go`, `cmd/internal/wrapcmd/wrap.go`, `cmd/internal/sessionwatch/run.go`, `nvim/init.lua`, `nvim/scrollback.lua`; add integration tests alongside the modified Go entrypoints and `tests/retention-test.sh`.

- [ ] Write failing managed-use tests: selected create/attach/resume, explicit history navigation, live and parked scrollback, changelog view, changed draft/queue/history save, prompt prepare/commit; failed open and unchanged autosave do not refresh. A successful authored write followed by metadata failure must block deletion.
- [ ] Run the affected package tests and `bash tests/retention-test.sh`; expect missing integration assertions to fail.
- [ ] Wire resolved owner context into managed entrypoints and propagate explicit identity through orientation to parked readers. Use one lease/touch API; expose internal CLI operations for Lua through the existing dispatcher contract. Compare content before publishing a use event; do not read log payloads for GC.
- [ ] Cover all managed content readers/writers by a checked call-site inventory, including programmatic and explicit saves. Verify generic history scans, refresh/statusline reads, diagnostics and distiller writes stay non-use operations. Their file access still needs a lease when racing collection.
- [ ] Test a long-held viewer/session across expiry, release grace, failed launch, overlapping readers, and cancellation using barriers and a stateful process fake. Run affected Go/Lua tests to PASS; update `atlas/` and its index for ownership/use/lock contracts; commit.
- [ ] Close M1 through `sdlc milestone-close --issue 239 --milestone M1` with recorded verification; fix the binary's review findings. No live deletion is enabled by M1.

### M2 — Safe collection, CLI and scheduled sweeps

#### Task 4: Recoverable group deletion

Files: create `cmd/internal/storagegc/transaction.go`, `transaction_test.go`, `collector.go`, `collector_test.go`; extend `cmd/internal/couchcore/retention.go`, `cmd/internal/launcher/session_index.go` through their owning APIs.

- [ ] Write failing transition and filesystem tests for every table row above. Inject failure before/after every journal write, rename and cleanup; verify unrelated owners, shared metadata and all newer incarnations survive.
- [ ] Run `go test ./cmd/internal/storagegc ./cmd/internal/couchcore ./cmd/internal/launcher -count=1`; expect collection contract failures.
- [ ] Implement exact quarantine transactions and recovery, revalidation under coordination, coordinated archive removal, exact binding cleanup and cache invalidation. Preserve original evidence on error; never retry deletion against source names after detachment.
- [ ] Exercise race barriers for use/launch/restore versus final eligibility, symlink replacement, unknown file appearance, partial inventory, live legacy processes and custom namespace failures. Test that raw/events cannot become a visible half-pair after recovery.
- [ ] Run tests and `go test -race ./cmd/internal/storagegc ./cmd/internal/couchcore ./cmd/internal/launcher`; require PASS and commit `#239: collect expired owner groups with recoverable detachment`.

#### Task 5: Commands, automatic scheduling and end-to-end verification

Files: create `cmd/internal/gccmd/run.go`, `run_test.go`, `cmd/internal/storagegc/schedule.go`, `schedule_test.go`; modify `cmd/pair-go/main.go`, `cmd/internal/dispatcher/dispatcher.go`, dispatcher coverage tests, launcher and Couch lifecycle entrypoints, `Makefile`, `atlas/index.md`; create `atlas/storage-retention.md`.

- [ ] Write failing CLI tests for default preview, apply, migration/store registration, invalid args and busy/incomplete states. Snapshot a fixture tree before/after preview and require identical files, contents and mtimes; pending recovery stays blocked and unmodified.
- [ ] Implement public `pair gc` routing/help and explicit migration completion with registered-store summary. Apply works only after migration completeness is established. Report counts/logical bytes and reasons; no payload contents in diagnostics.
- [ ] Implement a context-bound, joined worker after UI/session readiness, daily completion clock and bounded resumable cursor. Never count sweep discovery as use; initialize legacy records only on mutating runs.
- [ ] Add scheduler tests for 100,000 fixture names, budget/cursor progress, repeated starts, busy lock, shutdown/cancellation and interrupted passes. Assert one worker and bounded group processing, not brittle elapsed-time thresholds.
- [ ] Run `go test ./... -count=1`, `go test -race ./cmd/internal/storagegc ./cmd/internal/gccmd ./cmd/internal/couchcore ./cmd/internal/launcher`, `make test-lua`, `bash tests/retention-test.sh`, and `git diff --check`; require PASS.
- [ ] Run real-store preview only and record measured eligible/protected/untracked totals without reading content or deleting live files. Run apply against an isolated representative fixture containing old standalone, visible parked, newly archived, expired archived and live sessions; assert the expected survivors and second-apply idempotence.
- [ ] Document 60-day clocks, migration, managed-use boundary, custom stores, commands, retained-error diagnostics and no global size ceiling; update atlas index and issue Log. Commit and close M2 with the binary review gate.
- [ ] Close #239 with measured actuals and verification, then publish through `sdlc pr` and `sdlc merge`. Do not run destructive real-store migration/apply as a test; automatic collection begins only after migration completion and the full legacy grace.

## Revisions

### 2026-09-13 — fresh plan review

The reviewer found two ordering gaps: custom Couch stores can be on another filesystem, and launcher lifetime alone cannot protect detached children. Added a cross-store journal/receipt ordering contract and named actual wrapper, watcher, editor and viewer process registrations, including parent-death and EXDEV tests. These refine the mechanism without changing the approved retention policy (ARCH-ORDER, ARCH-PURPOSE). Re-review approved this chunk with no remaining concrete blockers; operator technical-plan approval remains pending.

### Acceptance evidence

Function-level ownership tests guard exact membership and parser ambiguity; pure policy tests guard the entire decision precedence and time boundary. Stateful integration tests guard each IO seam, crash step and actor ordering. CLI fixture snapshots prove preview has no writes. The real preview validates inventory at observed scale without treating old timestamps as authorization to delete. Successful tests plus both SDLC milestone reviews and final close are required before claiming completion.
