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
| StoreRegistry structural validity | `cmd/internal/storagegc/stores.go` | modified |
| RequestedNativeID and current-launch binding projection | `cmd/internal/sessionledger/record.go` | modified |
| Resume target projection | `cmd/internal/sessioninventory/query.go` | new |
| Launch candidate eligibility and event suffix | `cmd/internal/sessioninventory/target.go`, `round.go` | modified |
| Codex root source classification | `cmd/internal/sessioninventory/scan_codex.go` | modified |

One owner has many launch records; one launch has a requested UUID and at most one unambiguous confirmed root. Requested A and observed D are distinct facts, so their difference is not conflicting evidence. CurrentLaunch remains the single fold/transition authority. Legacy conflicting binding rows remain ambiguous. Byte boundaries identify current-launch evidence; filesystem identity only controls parser-cache reuse. No parallel identity registry or parser.

### Integration points

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| RegisterStore / explicit missing-store abandonment | `cmd/internal/storagegc/stores.go`, `cmd/internal/gccmd/run.go` | modified | retention lock and atomic registry writes |
| Ledger requested/confirmed writes | `cmd/internal/sessionledger/store.go` | modified | versioned append, current-launch compare under lock |
| Launch preparation and observation | `cmd/internal/sessionwatch/lifecycle.go`, `run.go` | modified | metadata baseline, correlation, native adapters, ledger/config publication |
| Resume admission | `cmd/internal/couchcore`, `cmd/internal/launcher/osruntime.go`, `launch_args_policy.go` | modified | persisted target, launch argv and UI classification |
| Capture preservation | `cmd/internal/launcher/osruntime.go`, `lifecycle.go`, `cmd/internal/wrapcmd/wrap.go`, shared capture helper as needed | modified | raw/events archive, owner lock and retention coordinator |
| TTY naming input | `cmd/internal/slugcmd/slugcmd.go` | modified | existing scrollback replay, bounded plain text and existing model runner |

Existing native runtime fixtures model transcript appends/metadata changes, watcher fixtures model process identity and clock, and launcher/Couch stateful fakes model lifecycle. Use real temporary file roots for ledger/archives/retention. No external agent service is added; naming reuses its existing runner and fake.

## Decisions, constraints and ordering

- ARCH-PURPOSE: implement the active issue Spec/Done when, including original registry defect, early Alt+n, A-to-D observation, automatic capture preservation and TTY naming. Prior full-body admission and closed source allowlists are superseded.
- ARCH-DRY: reuse CurrentLaunch, existing round thresholds, byte-offset event positions, artifactpath addressing, scrollback renderer and archive/retention conventions. Extract shared preservation from ParkScrollback rather than maintaining startup and quit variants.
- ARCH-PURE: fold requested/confirmed identity and select launch suffixes as pure functions; IO gathers observations and commits under existing locks.
- ARCH-ORDER: unknown fresh launch permits interaction but has no invented resume UUID; requested A -> probation(A); unique new-launch evidence A/D -> confirmed(A/D); silence/ambiguity -> unchanged; old-launch observation -> rejected. Alt+n during probation creates a new launch requesting A. Never infer failure from silence or timeout. Retain requested A, observed D and evidence reason in the ledger. Confirmation and config updates must not let a stale observer overwrite a newer launch.
- ARCH-CONSTRAINTS: startup/resume target projection reads the owner ledger only, no native body scan. Observation runs off the UI path at existing bounded poll cadence; admit changed existing transcripts and new candidates, reuse incremental catalog and record-size limits, skip unchanged old corpus bodies. A single raw capture is uncapped by row count; plain rendering remains bounded at the existing default except explicit unlimited callers. Archive within the same filesystem by rename where possible; do not buffer whole captures. Bounded exclusive-lock acquisition refuses a competing same-tag writer without truncation. No network dependency on startup; CPU/network budgets otherwise unchanged.
- ARCH-SECURE: Pair-owned ledger schema remains versioned/strict; agent-owned extensible metadata remains open-world. Preserve safe path, regular-file and bounded record readers. Unknown source is not automatic root rejection; known child/internal evidence still excludes non-root conversations. No transcript text in diagnostic history. Malformed optional native data cannot revoke durable UUID target. No credentials introduced.
- ARCH-MOCK: isolate HOME/XDG/PAIR_DATA_DIR/COUCH_STORE_DIR and inherited artifact overrides in process tests; stateful fixtures exercise actual disk/ledger writes, partial failure and restart. Metadata-only incident inventory supplies live conformance without restarting brain:0.
- ARCH-FUNERAL: append-only ledger owns requested/observed history. Catalog remains disposable. Capture families use existing owner/archive descriptors and GC retention; exclusive capture locks are released on close/process death and registered with artifact ownership if durable paths are introduced. No unlimited new diagnostic sidecar family. Missing registry entries require explicit abandonment, never automatic GC permission.
- Compatibility: retain reads of v1/v2 ledger records and their UUIDs without native proof revalidation for resume. New launch fields use a new explicit ledger version, not silent changes to old strict schemas. Do not rewrite old rows. A proof remains optional parsed-content evidence, not UUID authority. Exact identity-only consumers use the target projection; telemetry consumers keep parsed query results and degrade locally.

## Chunk 1: Registry startup and explicit abandonment (M1)

**Files:** `cmd/internal/storagegc/stores.go`, `stores_test.go`; `cmd/internal/couchcore/retention_test.go`; `cmd/internal/couchcmd` process/acceptance tests; `cmd/internal/storagegc/collector_test.go`; `cmd/internal/gccmd/run.go` and tests; `atlas/couch.md`, `atlas/index.md` as needed.

- [ ] Add failing regressions: register normal and auxiliary stores, delete auxiliary, construct/load/save the normal coordinated thread store and list it through Couch. Existing/new normal registration preserves the missing entry; unreadable/aliased selected stores and malformed registries still refuse.
- [ ] Add collector assertions that preview and apply cannot collect eligible payloads when any registered namespace is unavailable. Cover permission failures separately from absence with an injectable availability seam where platform privileges make chmod unreliable.
- [ ] Split structural validation from full availability validation. `RegisterStore` checks structure plus the selected existing canonical store; `ReadRegistry` and `CompleteMigration` retain full inventory validation. Audit all `loadRegistry` callers so availability requirements are explicit.
- [ ] Add `pair gc --forget-missing-store <exact-path>` and coordinator method. It is exclusive with register/complete/apply operations, requires an exact clean absolute registered path and confirmed ENOENT, rejects existing paths/symlinks/unknown paths/permission errors, preserves other entries, atomically sets `MigrationComplete=false`. Retry of an already removed entry reports no such registration rather than silently acknowledging a new inventory.
- [ ] Test GC remains blocked after forgetting until an independent `--complete-migration --store ...` acknowledges the remaining inventory. Document remount/restore as the alternative to permanent abandonment; never infer safe deletion from `/tmp` spelling.
- [ ] Add an isolated smoke recipe/helper setting HOME, XDG_DATA_HOME, PAIR_DATA_DIR and COUCH_STORE_DIR consistently and removing inherited explicit artifact-path overrides. Run a sentinel test proving the operator-shaped registry remains byte-identical. Record that the historical scratchpad invocation has not been identified.
- [ ] Run `go test ./cmd/internal/storagegc ./cmd/internal/gccmd ./cmd/internal/couchcore ./cmd/internal/couchcmd ./cmd/couch -count=1`, then the relevant packages with `-race`.
- [ ] Update atlas and issue Log, commit with `#346 M1`, then `sdlc milestone-close --issue 346 --milestone M1 --verified '<actual evidence>'`.

## Chunk 2: Durable targets and current-launch observation (M2)

**Files:** `cmd/internal/sessionledger/record.go`, `store.go` and colocated tests; `cmd/internal/sessioninventory/query.go`, `target.go`, `round.go`, `events.go`, `scan_codex.go` and tests; `cmd/internal/sessionwatch/lifecycle.go`, `run.go` and tests; `cmd/internal/launcher/osruntime.go`, `createflow.go`, `launch_args_policy.go`; `cmd/internal/couchcore` binding resolver, inventory classification and resume tests; identity-only consumers in `cmd/internal/reviewcmd`, `opener`; `atlas/couch.md`.

- [ ] Write red ledger tests for requested A without confirmation, A confirmed, requested A observed D, stale observer, duplicate append, and v1/v2 reads; legacy conflicting confirmed rows remain ambiguous. Add versioned requested ID and observation reason fields with strict encode/decode parity. Use the current-launch fold and lock, not last-row-wins across owners or launches.
- [ ] Write production Couch cold-resume and launcher admission tests with changed device/inode, deleted/corrupt catalog, unknown native fields, missing body and no park receipt. Add a ledger-only target query: current confirmed UUID else current requested UUID, plus confidence. Preserve agent/owner scope and UUID validation. Migrate identity-only consumers; do not label optional parser success as resume authority.
- [ ] Change launch preparation to record requested A plus pre-input Pair-log/native metadata offsets, without parsing/prebinding A. Pass Pair-generated fresh Claude/Qoder UUID as an explicit requested-ID origin distinct from a resume request. Tests prove args/session-id actually reach the launched child and no native body IO blocks startup.
- [ ] Write red fresh/resume observation tests: existing A appended, fresh D after requested A, existing D appended, historical matching messages only, partial record spanning baseline, same prompt in two candidates, silence, early Alt+n. Existing event positions encode byte offsets; filter records against the matching root transcript's launch RawSize. Never compare byte sizes directly with encoded event positions. Fresh and resumed flows use the same causal matcher.
- [ ] Select changed existing artifacts as well as new ones; skip unchanged baseline bodies. Parser state can rebuild when continuity changes, but only provably post-launch records count for confirmation. If shrink/replacement prevents a reliable suffix boundary, retain probation rather than count historical bytes. Remove the watcher's prebound early exit during probation. Keep lifecycle following separate from identity confirmation.
- [ ] Add root-source tests for vscode and an unfamiliar valid string, explicit parent/subagent/internal, unfamiliar object shape, malformed JSON and filename/internal metadata disagreement. Stop treating unknown extensible source values as a closed enum; use understood root/child evidence plus correlation. Scanner diagnostics must not erase a persisted target. Invalidate/rebuild obsolete rejected catalog entries as needed.
- [ ] Add Pair-supplied UUID handshake: new matching root filename absent at the launch baseline can confirm chosen X without prompt matching; existing filename alone cannot assert a fresh handshake. Test parent/subagent artifact paths are not selected. Preserve safe path access without requiring a whole body parse.
- [ ] Persist launch-specific unique confirmation under the ledger lock. Requested A plus observed D records the transition without treating them as conflicting confirmed roots. Preserve old rows for debugging. Existing ambiguous root evidence does not silently choose a winner. Test concurrent new launch, duplicate confirmation, indeterminate append and reconciliation; config publication must preserve saved argv and newer-launch ownership.
- [ ] Provide one-shot preview/apply recovery for audited unbound v2 launches by factoring the same observation pass. Keep preview read-only and content-free; apply rechecks launch and evidence. Repair ledger/catalog only, never config argv. Test scoped addressing, ambiguity, insufficient evidence (pair:4), repeated apply and competing observer. A stopped process is not required for evidence-based recovery.
- [ ] Run focused red/green tests as above, then `go test ./cmd/internal/sessionledger ./cmd/internal/sessioninventory ./cmd/internal/sessionwatch ./cmd/internal/couchcore ./cmd/internal/launcher ./cmd/internal/reviewcmd ./cmd/internal/opener -count=1` and relevant packages with `-race`. Mutation-check the current-launch byte boundary and early-resume admission.
- [ ] Update atlas and issue Log; commit and close M2 through SDLC with evidence.

## Chunk 3: Automatic capture preservation and TTY text consumers (M3)

**Files:** `cmd/internal/wrapcmd/wrap.go` and capture tests; `cmd/internal/launcher/osruntime.go`, `lifecycle.go`, quit/compaction tests; shared capture helper under `cmd/internal` if needed; `cmd/internal/artifactpath`, `storagegc` for any new lock artifact; `cmd/internal/slugcmd/slugcmd.go` and tests; `nvim/init.lua` quit UI and its tests; `atlas/couch.md`, relevant scrollback docs, `README.md`.

- [ ] Write red wrapper startup tests with pre-existing raw/events; require recoverable archives before same-tag reuse, including crash leftovers, raw-only/events-only partial families, repeated starts, archive collision/failure, and two active owners. Test real capture creation, not only the archive helper.
- [ ] Extract existing archive conventions into shared preservation with an exclusive per-capture owner lock held throughout writing. Preserve before every destructive create, including Alt+n wrapper exec. Reuse lifetime ownership correctly across exec or reacquire before mutation; an active competing writer is not an archival candidate. Safely register any persistent lock path with GC. Recover partial archive operations without losing either member or silently pairing unrelated captures; do not truncate on failure.
- [ ] Write red quit cleanup tests proving preservation always runs without an operator question and failed preservation retains originals even though unrelated cleanup continues. Remove prompt and default-discard behavior; never delete unarchived raw/events. Preserve timing/resize sidecar transfer errors. Normal quit followed by startup must not duplicate a fully archived capture. Keep compaction's archive/checkpoint descriptors working.
- [ ] Write capture beyond 2,000 rows through the actual wrapper and assert full raw retention plus unlimited-render tail; default view can still be bounded. Existing archives remain available through current retention ownership; do not change retention duration.
- [ ] Write naming tests with no native binding/transcript but meaningful TTY conversation. Use bounded printable replay (existing renderer) as model input while retaining existing budgets/fallback, not an independent ANSI stripper. Keep exact draft history untouched. Test missing/failed optional native telemetry does not disable text features or resume.
- [ ] Run `go test ./cmd/internal/wrapcmd ./cmd/internal/launcher ./cmd/internal/slugcmd ./cmd/internal/scrollbackcmd ./cmd/internal/storagegc -count=1` and relevant race tests; run applicable Lua tests. Update help/docs in the same change.
- [ ] Run `go test ./cmd/... -count=1` in isolated test roots and `make build`. Use rebuilt `couch --list` and supported recovery preview/apply to account for all 23 incident rows; leave brain:0 live and pair:4 explicitly unconfirmed where evidence is absent. Record actual outcomes rather than assuming UI labels prove confirmation.
- [ ] Close M3 with SDLC review, then `sdlc close --issue 346 --verified '<actual evidence>'`; publish through SDLC PR/merge. Do not claim completion with outstanding acceptance cases.

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
