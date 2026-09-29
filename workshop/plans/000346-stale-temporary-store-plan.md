# Couch recovery, observed bindings and preserved TTY history Implementation Plan

> **For agentic workers:** Follow AGENTS.md §3 for bounded delegation; SDLC owns milestone reviews. Execute with TDD in pair:0 (`/Users/xianxu/workspace/pair`), branch `000346-stale-temporary-store`.

**Goal:** Keep Couch usable after reboot, resume recorded conversations under nonblocking probation, observe actual current-launch identity, and preserve the TTY history used by text features.

**Architecture:** Separate registry structure from GC availability; separate durable requested/observed UUID from optional transcript parsing; share fresh/resume post-launch correlation. Preserve captures before path reuse under exclusive ownership, using existing archive and retention conventions.

**Tech Stack:** Go, Lua editor integration, JSONL ledgers, native transcript adapters, filesystem locks, existing stateful runtime fixtures.

**Status:** Operator agreed the model and explicitly requested implementation in pair:0 on 2026-09-29. This reconciliation implements that direction; `change-code` owns the plan-quality and estimate gates before code changes.

## Core concepts

### Pure entities

| Name | Lives in | Status |
| --- | --- | --- |
| RegistryEntry availability diagnostic | `cmd/internal/storagegc/stores.go` | new |
| StoreRegistry structural validity | `cmd/internal/storagegc/stores.go` | modified |
| RequestedNativeID and current-launch binding projection | `cmd/internal/sessionledger/record.go` | modified |
| Resume target projection | `cmd/internal/sessioninventory/query.go` | new |
| Launch candidate eligibility and event suffix | `cmd/internal/sessioninventory/target.go`, `round.go` | modified |
| Codex root source classification | `cmd/internal/sessioninventory/scan_codex.go` | modified |

One owner has many launch records; one launch has a requested UUID and at most one unambiguous confirmed root. Requested A and observed D are distinct facts, so their difference is not conflicting evidence. CurrentLaunch remains the single fold/transition authority. Legacy conflicting binding rows remain ambiguous. Byte boundaries identify current-launch evidence; filesystem identity only controls parser-cache reuse. No parallel identity registry or parser.

### Integration points

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| InspectRegistry | `cmd/internal/storagegc/stores.go` | new | read-only structural inventory plus availability diagnostics |
| RegisterStore / explicit missing-store abandonment | `cmd/internal/storagegc/stores.go`, `cmd/internal/gccmd/run.go` | modified | retention lock and atomic registry writes |
| Ledger requested/confirmed writes | `cmd/internal/sessionledger/store.go` | modified | versioned append, current-launch compare under lock |
| Launch preparation and observation | `cmd/internal/sessionwatch/lifecycle.go`, `run.go` | modified | metadata baseline, correlation, native adapters, ledger/config publication |
| Resume admission | `cmd/internal/couchcore`, `cmd/internal/launcher/osruntime.go`, `launch_args_policy.go` | modified | persisted target, launch argv and UI classification |
| Capture preservation | `cmd/internal/launcher/osruntime.go`, `lifecycle.go`, `cmd/internal/wrapcmd/wrap.go`, shared capture helper as needed | modified | raw/events archive, owner lock and retention coordinator |
| TTY naming input | `cmd/internal/slugcmd/slugcmd.go` | modified | existing scrollback replay, bounded plain text and existing model runner |

Existing native runtime fixtures model transcript appends/metadata changes, watcher fixtures model process identity and clock, and launcher/Couch stateful fakes model lifecycle. Use real temporary file roots for ledger/archives/retention. No external agent service is added; naming reuses its existing runner and fake.

## Decisions, constraints and ordering

- ARCH-PURPOSE: implement the active issue Spec/Done when, including original registry defect, early Alt+n, A-to-D observation, automatic capture preservation and TTY naming. Prior full-body admission and closed source allowlists are superseded.
- ARCH-DRY: reuse CurrentLaunch, existing round thresholds, byte-offset event positions, artifactpath addressing, scrollback renderer and archive/retention conventions. Reuse ParkScrollback copy/publication conventions for startup, leaving compaction semantics intact.
- ARCH-PURE: fold requested/confirmed identity and select launch suffixes as pure functions; IO gathers observations and commits under existing locks.
- ARCH-ORDER: unknown fresh launch permits interaction but has no invented resume UUID; requested A -> probation(A); unique new-launch evidence A/D -> confirmed(A/D); silence/ambiguity -> unchanged; old-launch observation -> rejected. Alt+n during probation creates a new launch requesting A. Never infer failure from silence or timeout. Retain requested A, observed D and evidence reason in the ledger. Confirmation and config updates must not let a stale observer overwrite a newer launch.
- ARCH-CONSTRAINTS: startup/resume target projection reads the owner ledger only, no native body scan. Observation runs off the UI path at existing bounded poll cadence; admit changed existing transcripts and new candidates, reuse incremental catalog and record-size limits, skip unchanged old corpus bodies. A single raw capture is uncapped by row count; plain rendering remains bounded at the existing default except explicit unlimited callers. Stream bounded snapshots to durable archives before source retirement; do not buffer whole captures. Bounded exclusive-lock acquisition refuses a competing same-tag writer without truncation. No network dependency on startup; CPU/network budgets otherwise unchanged.
- ARCH-SECURE: Pair-owned ledger schema remains versioned/strict; agent-owned extensible metadata remains open-world. Preserve safe path, regular-file and bounded record readers. Unknown source is not automatic root rejection; known child/internal evidence still excludes non-root conversations. No transcript text in diagnostic history. Malformed optional native data cannot revoke durable UUID target. No credentials introduced.
- ARCH-MOCK: isolate HOME/XDG/PAIR_DATA_DIR/COUCH_STORE_DIR and inherited artifact overrides in process tests; stateful fixtures exercise actual disk/ledger writes, partial failure and restart. Metadata-only incident inventory supplies live conformance without restarting brain:0.
- ARCH-FUNERAL: append-only ledger owns requested/observed history. Catalog remains disposable. Capture families use existing owner/archive descriptors and GC retention; exclusive capture locks are released on close/process death and registered with artifact ownership if durable paths are introduced. No unlimited new diagnostic sidecar family. Missing registry entries require explicit abandonment, never automatic GC permission.
- Compatibility: retain reads of v1/v2 ledger records and their UUIDs without native proof revalidation for resume. New launch fields use a new explicit ledger version, not silent changes to old strict schemas. Do not rewrite old rows. A proof remains optional parsed-content evidence, not UUID authority. Exact identity-only consumers use the target projection; telemetry consumers keep parsed query results and degrade locally.

## Observation and capture transition contracts

### Incomplete native baseline (PQ-2)

V3 launch stores `BaselineComplete` separately from its artifact list. `prepareRuntimeLaunch` records requested identity and starts the agent even when native metadata enumeration fails; incomplete is never encoded as an empty complete snapshot. The watcher cannot use absent-at-launch handshake or current-launch correlation while coverage is unknown. On the first complete later scan it establishes an in-memory observation epoch with that snapshot's raw offsets and the current Pair prompt-log offset; only subsequent exchanges can confirm. Watcher restart repeats this conservative epoch acquisition. Offline repair cannot use an unknown launch baseline to infer a missing historical binding. This sacrifices early evidence, not interaction or the resume target. The observer reports probation without erasing A. Silence with a live owned process keeps the existing slow polling cadence; disappearance of that exact process ends observation, not durable identity.

### Startup capture preservation (operator simplification)

Preservation happens before the wrapper reuses a capture pathname, including Alt+n wrapper exec. Quit leaves raw/events intact and removes the question/discard behavior. Keep existing compaction named-copy behavior; #347 separately designs permanent capture identities. No durable pending transaction, exactly-once archive guarantee or live-copy protocol redesign is required here.

Acquire a stable per-capture writer lock nonblocking before startup preservation and retain it through capture writing. A competing writer causes an immediate diagnostic without mutation. Close/sync and release before exec; the next wrapper reacquires before modifying files. Source-file locks are insufficient because truncation/rotation changes their lifetime. Register any new lock artifact with artifactpath/retention, and use close-on-exec descriptors.

Use a narrow shared archive helper based on existing ParkScrollback copy behavior: reserve a unique destination, stream and sync all existing source members, publish existing capture metadata, then permit active file reuse. An existing unreadable sidecar is failure, not optional absence. Preserve partial leftovers without attaching unrelated metadata. Failure leaves original sources untouched; incomplete archives are retained or cleaned only when demonstrably owned by this attempt. A crash may produce a redundant archive on retry, which is acceptable. No source is truncated until a complete archive exists. A crash between subsequent active-file opens cannot lose old data because that archive is already durable.

Compaction continues to reference its exact named copy; it does not take the writer lifetime lock or participate in startup archival. Existing producer retention protects archive publication. Ledger history follows existing owner/session cleanup, and completed captures retain existing retention duration. Idempotent observation does not append duplicate ledger rows.

## Chunk 1: Registry startup and explicit abandonment (M1)

**Files:** `cmd/internal/storagegc/stores.go`, `stores_test.go`, `collector_test.go`; `cmd/internal/gccmd/run.go`, `run_test.go`; `cmd/internal/couchcore/retention_test.go`; `cmd/internal/couchcmd` production-boundary test; `tests/couch-recovery-smoke.sh`, `tests/with-isolated-pair.sh`; `atlas/couch.md`, `README.md`.

- [x] Add failing tests at the coordinated-store/CLI boundary and pure structural validator using the strategies below; verify the expected failure before implementation.
- [x] Split `StoreRegistry.validateStructure` from full availability validation. `RegisterStore` checks structure and its selected canonical directory; `ReadRegistry`, `CompleteMigration` and GC keep complete availability checks. Other registrations remain intact.
- [x] Add coordinator `ForgetMissingStore` and exclusive `pair gc --forget-missing-store PATH`. Under the coordinator lock require an exact clean registered path and confirmed missing directory, preserve other registrations and atomically reset migration acknowledgment. Missing is explicit permanent abandonment, never empty-store proof.
- [x] Isolate smoke HOME/XDG/Pair/Couch roots and inherited artifact overrides; preserve the unknown provenance of the original scratchpad entry instead of inventing an originating command.
- [x] Run focused red/green tests, then `go test ./cmd/internal/storagegc ./cmd/internal/gccmd ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/couch -count=1` and relevant `-race` suites. Update atlas/Log, commit and `sdlc milestone-close --issue 346 --milestone M1 --verified '<evidence>'`.

| Risky function | Test strategy |
| --- | --- |
| StoreRegistry.validateStructure | Table/property tests over malformed schemas and path ordering; availability never changes structural validity. |
| RegisterStore / NewCoordinatedThreadStore | Real temporary namespace disappearance followed by production list/load/save; assert only the selected store's availability gates use. |
| ForgetMissingStore | Stateful filesystem failure/alias permutations under coordinator; assert exact removal and migration reset or byte-identical registry on rejection. |
| Collector.Preview / Apply | Eligible payload fixture with unavailable registered namespace; assert no deletion until explicit valid migration acknowledgment. |
| InspectRegistry / gccmd.Run | Real unavailable namespace; structurally valid paths remain visible with restore/remount or explicit abandonment guidance, without mutation. |
| gccmd.Run / smoke process environment | Rejected operation combinations have no writes; sentinel operator-shaped root stays byte-identical through isolated subprocess. |

## Chunk 2: Durable targets and current-launch observation (M2)

**Files:** `cmd/internal/sessionledger/record.go`, `store.go`; `cmd/internal/sessioninventory/query.go`, `target.go`, `offline.go`, `events.go`, `scan_codex.go`; `cmd/internal/sessionwatch/lifecycle.go`, `run.go`, new `recover.go`/CLI; `cmd/internal/launcher/osruntime.go`, `createflow.go`, `launch_args_policy.go`; `cmd/internal/couchcore/resume.go`, `actionableinventory.go`; identity consumers in `cmd/internal/reviewcmd`, `opener`; colocated tests and dispatcher; `atlas/couch.md`.

- [x] Introduce red ledger/target tests using the strategy table; add v3 requested UUID, request origin, baseline completeness and confirmation reason. Requested A and observed D coexist as different facts. Legacy v1/v2 remain readable; confirmed conflicts stay ambiguous.
- [x] Add ledger-only `QueryResumeTarget` and migrate Couch/launcher admission and identity-only callers. Return confirmed UUID else current requested UUID, never arbitrary earlier fresh launches. Optional parsed queries stay separate. Device changes or missing body/cache do not gate target use.
- [x] Prepare launches metadata-only with the incomplete-baseline contract above, no prebinding or resume-body read. Pass Pair-generated fresh UUID origin from createflow.
- [x] Share post-launch observation across fresh/resume, admitting new and changed existing paths. Decode the existing event position to record byte offset and filter with the matching artifact baseline in `RoundsAfterLaunch`. Rebuild parsing independently, but retain probation when suffix provenance is unknown. Use current matching thresholds and ambiguity rules.
- [x] Make Codex metadata open-world; retain understood positive root/child evidence and safe file reading. New Pair-chosen root filename supplies the explicit handshake only when baseline absence is known. Rebuild obsolete rejected cache states. Keep requested-target authority independent of internal field interpretation.
- [x] Commit unique confirmation using the current-launch lock; same-root duplicate is idempotent, competing confirmation is refused, and requested-to-observed transition remains diagnosable. Ensure config publication cannot erase saved argv or affect a newer launch. Lifecycle telemetry follows independently.
- [x] Factor one observation pass into scoped `session-repair` preview/apply for audited v2 unbound launches. Preview is read-only; apply rechecks owner/launch and writes ledger/catalog only. No PID liveness prerequisite for historical recorded evidence.
- [ ] Run `go test ./cmd/internal/sessionledger ./cmd/internal/sessioninventory ./cmd/internal/sessionwatch ./cmd/internal/couchcore ./cmd/internal/launcher ./cmd/internal/reviewcmd ./cmd/internal/opener -count=1` and relevant `-race` suites. Update atlas/Log, commit and close M2 via SDLC.

| Risky function | Test strategy |
| --- | --- |
| CurrentLaunch / ledger codec | Generated event sequences and version round trips; assert requested/observed separation, owner isolation and legacy ambiguity. |
| QueryResumeTarget / RequireNativeResumeBinding | Production Couch/launcher admission with optional transcript IO failing; assert exact UUID/agent retained and early Alt+n works. |
| prepareRuntimeLaunch | Stateful incomplete metadata enumeration; assert startup records coverage uncertainty and no transcript-body read. |
| SelectTargetWork / RoundsAfterLaunch | Append/replace/partial-record fixture sequences; mechanically assert every confirming record starts after the appropriate observed byte boundary. |
| codexRole / applyCodexRecord | Fuzz valid extensible fields and malformed input, seeded with root/child distinctions; unknown values do not act as a closed allowlist. |
| Run / confirmation append | Stateful clock/process/ledger interleavings; assert silence preserves target, stale observers cannot mutate, and confirmed UUID follows unique current-epoch evidence. |
| session-repair / config publication | Real temporary ledger/config retries and injected uncertain writes; assert preview no writes, apply idempotence and saved argv/new launch preservation. |

## Chunk 3: Automatic capture preservation and TTY text consumers (M3)

**Files:** shared archive helper in `cmd/internal/launcher` and tests; `cmd/internal/wrapcmd/wrap.go`; `cmd/internal/launcher/osruntime.go`, `lifecycle.go`, runtime/quit/compaction tests; `cmd/internal/artifactpath`, `storagegc` artifact/retention integration; `cmd/internal/slugcmd/slugcmd.go`, `slug.go`, tests; reusable plain-render API in `cmd/internal/scrollbackcmd`; quit UI/help references in `nvim/init.lua`, `pairlifecycle`, README/atlas.

- [ ] Implement red production archive/create tests, then the startup preservation contract above. Reuse artifactpath/retention publication and ordinary raw/events formats; no pending journal.
- [ ] Replace independent destructive wrapper opens with writer ownership and preservation before creation. Close/sync/release on restart. Quit leaves raw/events intact, removes prompt/discard paths, and keeps unrelated cleanup. Keep current compaction copy behavior and exact continuation references.
- [ ] Expose a bounded plain replay API and use it for naming without invented role labels; preserve existing model budgets and proposal validation. Leave exact Pair prompt history intact and optional telemetry locally degradable.
- [ ] Run `go test ./cmd/internal/wrapcmd ./cmd/internal/launcher ./cmd/internal/slugcmd ./cmd/internal/scrollbackcmd ./cmd/internal/storagegc -count=1`, relevant race suites and applicable Lua tests. Update help/atlas with the behavior change.
- [ ] Run `go test ./cmd/... -count=1` and `make build`. Inspect rebuilt inventory and supported repair for all incident rows without restarting brain:0. Close M3 and issue through SDLC, then publish through PR/merge.

| Risky function | Test strategy |
| --- | --- |
| Startup archive helper | Failure injection around member copy/sync/publication and fresh-process retry; assert old data remains in sources or completed archive before any reuse, allowing redundant archives. |
| Capture writer acquisition / wrapper startup | Competing real processes and exec restart with existing capture; assert one writer and byte preservation on contention/failure. |
| ParkScrollback / compaction | Existing production compaction tests retain specifically named copy and exact continuation reference without writer-lock acquisition. |
| PreserveScrollback / CleanupSidecars | Production quit cleanup asserts no prompt/archive requirement, retained raw/events and independent sidecar cleanup. |
| Plain replay / slug input | Large terminal stream with control sequences; assert bounded model input, unlimited retained raw output and naming without native binding. |

## Native conformance and scope limits

At M2 completion compare sanitized structural observations from the four incident Codex files with adapter/round behavior in a read-only audit; the already-established send matches are evidence, not new test payloads. On future native adapter upgrades rerun this conformance and controlled disposable CLI sessions when available. Never start paid agents merely for a smoke without need, and never interrupt brain:0. The capture work does not change retention duration or concatenate all historical archives into the default view. Root association does not attempt to infer arbitrary undocumented native lineage. Empty-slot label rendering remains outside this issue.

## Verification notes

Tests precede implementation and must fail for the missing behavior. Use stateful disk/process fixtures at the final production decision boundary. Keep raw conversation contents out of fixtures/reports. Reconcile entity names and paths above with final implementation before each review; no test-only bypasses. No live OS reboot or interruption of brain:0 is needed to reproduce device renumbering or archive leftovers.

## Revisions

### 2026-09-29 — UUID is conversation identity

Operator pointed out that native transcript UUIDs already provide conversation identity; volume information is unnecessary for scoping/deduplication. Replaced the proposed device-only/same-inode exception with exact-conversation revalidation whenever filesystem continuity is lost. Device/inode/generation remain cache hints. Added positive replacement/copy cases with the same validated UUID and negative cases for contradictory internal identity. No implementation has started.

### 2026-09-29 — Fresh review: preserve saved arguments

Reviewer found that reusing `ObserveAndPersist` config refresh with no CLI argv could erase saved options or write stale config after a concurrent launch. Repair is now explicitly ledger/catalog-only; the live watcher retains its config behavior. Added byte-preservation and production cold-resume tests, plus a rejected-vscode old-catalog regression.

### 2026-09-29T09:59:21-07:00 — Superseded by agreed launch observation and TTY ownership

The operator approved the conceptual model now recorded in issue #346: retain a durable UUID resume target independently of file/parser proofs, launch under probation, and observe current-launch exchanges on both existing and new root transcripts. Strong evidence may replace requested A with actual D while retaining history; silence and early Alt+n remain supported. Unknown native source values are open-world, not blanket rejection. Automatically preserve raw TTY and its sidecar on quit and before same-tag reuse; use printable TTY for text features and isolate native telemetry. This supersedes the exact-body-revalidation, conflicting-internal-UUID refusal, closed source allowlist, new-file-only recovery and immutable-existing-binding assumptions in this historical plan. The issue's active Spec/Done when/Plan are the current contract. Detailed implementation planning and review remain pending; no code has been changed.

### 2026-09-29T10:07:21-07:00 — Implementation reconciliation after operator go-ahead

Replaced the superseded executable body with the agreed target/observation separation and TTY ownership design. Retained original registry work and prior revision history. User requested starting #346 in pair:0; plan-quality and estimate gates will run before implementation. Three real review boundaries remain: registry, binding, and TTY/text consumers.

### 2026-09-29T10:10:47-07:00 — Plan-quality recovery contracts

Addressed PQ-1 with explicit capture phases, two-lock ordering, exec release/reacquire and live-copy semantics; PQ-2 with explicit incomplete coverage and conservative later observation epochs; PQ-3 by replacing test inventories with named risky functions and mechanical strategies. Also quantified contention behavior, tied ledger/pending history to owner retention, and specified native adapter conformance.

### 2026-09-29T10:22:03-07:00 — Operator simplifies archival ownership

Replace archive-on-quit and exactly-once recovery machinery with startup-only copy-before-truncate, writer exclusion and retained originals on failure. Redundant archives after crash are acceptable. Existing compaction named copies remain; permanent capture IDs are filed separately as #347. This explicitly supersedes the stronger PQ-1 transaction solution because its extra guarantee is not required by the operator.

### 2026-09-29T10:41:21-07:00 — M1 review surfaces reconciled

Add RegistryEntry/InspectRegistry to the concept tables and name the shared smoke environment wrapper and README as delivered surfaces. BR-1–3 corrections are tested at production CLI/subprocess boundaries; conservative unregister availability remains documented.

### 2026-09-29 — M2 review corrections

The live watcher and offline repair must share `ResolveBindings`, including its
per-send candidate intersection, rather than merely share round extraction.
Restart-marker construction must use the same pure target projection as direct
queries. The requested UUID is still a probation target, not confirmation; the
chosen-filename handshake confirms the latter. Review BR-5 proposes changing
this existing policy for a chosen UUID whose file does not yet exist; operator
preference has been requested, and no additional filename admission gate is
introduced while that policy question remains unresolved.
