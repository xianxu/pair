# Wrapper log IO implementation plan

> Current governing design: the scope/protocol revision at the end of this document supersedes the initial proposal below.

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Use superpowers-executing-plans for the session-warm implementation; SDLC owns fresh reviews at each boundary.

**Goal:** Remove per-event metadata churn from wrapper diagnostics, bound their storage, and batch raw capture writes while preserving existing capture authority.

**Architecture:** Keep diagnosticlog's coordinated append and durable rotation protocol; coalesce records before entering it. Deduplicate immutable registry declarations, retain exact generation checks, and add an explicit optional count policy for wrapper diagnostics. Keep the raw/event capture format and its witness identity intact.

**Tech Stack:** Go, existing diagnosticlog/procutil/storagegc integration, filesystem fixtures, counted IO observation, manual clock events, recorded SDLC suite.

**Approval status:** PROPOSED. The original issue also requires raw size rotation. The hub must explicitly approve separating that portion into a faithful checkpoint/segment follow-up; until then this plan is not the whole issue contract and must not enter change-code. No original Done-when criterion has been removed.

## Evidence and alternatives

The issue Log records physical86x amplification, at least2187 state inode replacements/minute, two actual metadata publications per append, and the exact live rotation blocker. Hot optional appends already omit fsync: adding only a conditional fsync would not fix this.

Recommended: coalesce above the existing coordinated writer; preserve its partial-write/replacement/crash recovery behavior. Alternative: replace it with a persistent single-writer log owner. The latter could remove more metadata but changes cross-process authority for shared adaptation/debug writers; it is a larger protocol migration without a demonstrated need. Raw suffix eviction is a separate alternative requiring full semantic VT checkpoints; screen-only ANSI seeds are incorrect for arbitrary future input.

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| BatchPolicy / batch transition model | cmd/internal/diagnosticlog/batch_model.go | new |
| Generation retention selection | cmd/internal/diagnosticlog/generation_limit.go | new |

The batch model owns pending records/bytes, active-flush progress, closing and closed states. It accepts explicit enqueue, flush-trigger, IO-result and close events. No caller mutates its authoritative counters/state. A partial write consumes only confirmed bytes; an error after n=len consumes all bytes and records the error, never duplicates payload on retry. Full records stay together at rotation boundaries. Future policy changes belong to this owner, not wrapper callers.

The generation selector consumes a complete bounded list of certified immutable generations, orders by last-write/start/name and chooses the oldest excess entries. It never selects current. Incomplete traversal, malformed metadata and changed identities retain data and refuse growth rather than guessing.

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| Buffered diagnostic owner | cmd/internal/diagnosticlog/buffered.go | new | existing Writer; manual tick source in tests |
| Counted filesystem writer seam | cmd/internal/diagnosticlog/writer.go, append.go | modified | actual payload and metadata writes, renames and syncs |
| Stable registry publication | cmd/internal/diagnosticlog/registry.go | modified | existing registry lock and exact declaration |
| Runtime writer proof | cmd/internal/diagnosticlog/proof.go | modified | existing stateful processInspection seam |
| Wrapper capture batching | cmd/internal/wrapcmd/capture_buffer.go, wrap.go | new / modified | existing witnessed raw/event file pair and masterPump tick |
| Wrapper diagnostic wiring | cmd/internal/wrapcmd/wrap.go; cmd/internal/adapt/adapt.go | modified | buffered diagnostic owner for wrap-events, debug and wrapper adaptation |

Filesystem tests use portable temporary directories and existing fault checkpoints. The buffered owner also uses a stateful sink modeling partial progress, errors and close; its manual timer models idle, dirty, flushing and shutdown ordering. Existing processInspection doubles model protocol/legacy participants across repeated probes. Real OS conformance and before/after live disk counters remain required; the counter seam is not claimed to measure APFS journal bytes.

## Operating envelope and lifecycle

- Diagnostics: initial target64KiB flush threshold, maximum256KiB pending plus at most256KiB in flight, at most1024 queued records. Flush dirty data within5s while IO/scheduling succeeds; idle ticks publish nothing. Size-triggered flushes shorten that window. On overload reject whole new records with ErrBusy and count loss; never wait for process inspection or unbounded IO from a terminal callback. Validate the threshold/interval against the actual physical-after measurement; do not claim <3x without it.
- A process crash can lose pending/in-flight diagnostic memory (bounded512KiB; normally the last5s). Persistent IO failure invalidates the time bound and is reported by writer status/Close. Host power loss has no interval guarantee for optional diagnostics: rotation/collection remain the durable boundaries.
- Raw/events: buffer at most64KiB each, flush every250ms from the owning pump and at orderly shutdown. Flush raw before publishing sidecar offsets; logical offsets count accepted raw bytes. Publish the capture witness immediately before buffering begins. The healthy-process crash-loss target is the last250ms plus at most one bounded pump chunk; byte bounds are deterministic, scheduler/IO latency is not. Do not promise power-loss durability that the current raw capture does not have.
- Diagnostic storage proposal:64MiB current +3 immutable generations (256MiB payload bound), trimming certified oldest diagnostic generations on rotation. Unknown writer proof or incomplete enumeration stops further optional append growth at the hard cap; it never authorizes deletion. Pre-existing oversized/uncertified data stays unchanged and is reported as blocked, not silently truncated. Confirm this explicit shorter-than365-day diagnostic policy with the hub.
- Each buffer worker belongs to its writer; Close excludes new admissions, flushes accepted bytes, stops the timer and joins the worker before returning. Wrapper startup failures and fresh exec use the same teardown path. No background task survives Close. Registry and stable lock inode lifecycles remain unchanged.
- Capture rotation is excluded ONLY if the hub approves the split. Follow-up must own checkpoint export/restore, certified segment manifest, bounded eviction, reader migration, archive preservation and explicit retained-history completeness semantics. Current witness/park/recovery/previous readers remain exact and get compatibility regressions here.

## Architecture decisions

- ARCH-DRY: reuse coordinated diagnostic append, existing durable rotation/deletion and stateful process inspection; one batch owner across wrapper diagnostic emitters.
- ARCH-PURE / ARCH-ORDER: queue transitions and excess-generation choice are pure; clock, file progress and failures enter as events. Close during flush, admission during flush, partial progress and failure after all bytes are written have explicit tests. IO never rolls back confirmed bytes.
- ARCH-PURPOSE: both metadata churn and blocked rotation are root causes. Original raw-rotation requirement cannot disappear under a claim of completeness; scope change requires explicit hub approval and a separately filed contract.
- ARCH-MOCK: tests reuse real folder-backed owned storage and processInspection doubles; buffered IO uses a stateful sink behind the same writer interface. No new external binary/service dependency.
- ARCH-CONSTRAINTS: budgets above are per wrapper, with fleet co-tenancy; no per-event process scans, no timer-only writes, no timing-based correctness assertions. Physical counter measurements compare actual live output rates.
- ARCH-SECURE: preserve generation/identity checks and strict process-generation fences. Protocol-aware rows need no root canonicalization before legacy comparison; unknown legacy roots still refuse. Registry no-op compares a validated exact declaration under the existing lock; symlinks/replacements/malformed files never become equality. An unchanged registration is not a reason to remove necessary sync after an uncertain publication: retain/reconfirm the durability boundary.
- ARCH-FUNERAL: pending memory dies at joined Close; immutable diagnostic generations have an explicit count/age end; unknown evidence blocks deletion and growth. All new source files join artifactpath's exhaustive source inventory.

## M1 — Coalesced IO and authority-preserving batching

Files: new batch_model.go/_test.go and buffered.go/_test.go; writer.go, append.go, registry.go, publication_test.go, registry_test.go; wrap.go and new capture_buffer.go/_test.go; adapt.go and tests; artifactpath/manifest.go; atlas/storage-retention.md and storage-retention-io.md, README diagnostic guidance.

- [ ] Add a counted real-writer regression using1000 distinct JSONL events. Count all metadata and payload bytes/publications at the real write boundary, with a separate initial-open baseline. Baseline exceeds3x. The fixed synthetic batched stream must stay below3x and prove all records retained in order. Add idle/no-change publication assertions and both size/manual-interval triggers; no elapsed-time assertions.
- [ ] Write pure state sequence tests and stateful sink tests for busy admission, byte+record bounds, close during flush, partial progress, full progress plus error, repeated Close, write-after-close and worker/timer joining. Capture pre-flush process-loss behavior and closed-worker absence deterministically.
- [ ] Implement bounded record batching above existing diagnostic Writer. Empty flush triggers have no effects. Keep ordinary direct Writer behavior for short-lived CLI writers and existing durability tests. Use the common buffered owner for the wrapper's wrap-events, late-open debug and adaptation writers.
- [ ] Add registry no-change tests that count writes/renames and stable inode; test changed declaration, malformed/replaced entries and failure after rename before directory sync. Implement exact no-op publication without suppressing unresolved durability work.
- [ ] Add raw/event integration tests against the real witnessed pair: before flush, interval flush, size flush, raw-before-sidecar ordering, flush error and shutdown; preserve exact original byte output, resize/time offsets and witness identity. Add replay parity covering split UTF8/CSI/OSC and main/alt transitions; no screen-seed shortcut.
- [ ] Implement raw/event buffering in the pump lifecycle, preserving witness publication before child output and existing syncs at ParkScrollback/recovery/delivery authority boundaries. No diagnostic flush is needed to acknowledge a delivery: its authoritative receipt has its own owner.
- [ ] Run focused package tests and race tests, make -k build, artifactpath inventory guard and a bounded live scratch-wrapper before/after experiment. Record process disk bytes, appended bytes and inode replacements using the same measurement method as baseline. If physical amplification remains>=3x, stop and revise based on measured costs.
- [ ] Update atlas/README with measured outcome, exact process-crash/power-loss limits and visibility/overload policy; commit evidence and close M1 through SDLC's fresh review.

## M2 — Safe bounded diagnostic generations and qualification

Files: proof.go, process_inventory_test.go; new generation_limit.go/_test.go; writer.go, collect.go and existing deletion/rotation fault tests; wrap.go options; artifactpath/manifest.go; atlas/storage-retention.md and relevant README paragraph.

- [ ] Red regression: protocol-aware runtime with a removed data root must not globally block rotation; identical legacy runtime remains unknown/refused. Preserve PID-generation fencing before/after environment reads, denial and cancellation cases. Fix classification at the owning process-inspection boundary.
- [ ] Red integration: repeated small-threshold rotations stay within current+3 certified immutable generations, newest records remain readable, and current does not exceed hard cap when proof is unknown. Pure selection tests cover tied clocks, malformed/future metadata and incomplete enumeration. Bound traversal (initial maximum64 certified generations); incomplete/oversized inventory blocks growth and requires explicit GC rather than unbounded scanning in callbacks.
- [ ] Implement optional wrapper diagnostic count retention using existing deletion intent/replay and sync ordering. No raw or parked-capture file is eligible for diagnostic pruning. Reuse asynchronous maintenance outside terminal callbacks and keep unknown/replaced generations retained. Report blocked legacy excess separately from guaranteed new-write growth bounds.
- [ ] Test every interruption around count-prune intent, payload removal and metadata/state finalization; replay must never unlink a replacement. Prove rotation can resume after proof becomes available and retained generation count does not grow under repeated errors.
- [ ] Add unchanged-format compatibility tests: raw/event replay and exact previous-capture selection remain valid while diagnostic generations rotate; exercise real park/retained-reader seams, not only a filename parser. If raw rotation stays in scope, this plan must be revised before implementation to cover the full checkpoint representation instead.
- [ ] Run focused packages/race, make -k build; push the final code head; run required sdlc local-check run --name suite. Complete operator/hub-approved live scratch smoke, with before/after counters and recovery/previous-peek evidence. No unrelated live slot relaunch without hub authorization.
- [ ] Update issue/atlas/README and evidence, close M2 then issue via mandatory fresh SHIP reviews, open PR and land via SDLC. Report one final outcome/PR/full real git hashes to the hub.

## Validation commands

Targeted: go test ./cmd/internal/diagnosticlog ./cmd/internal/wrapcmd ./cmd/internal/adapt ./cmd/internal/artifactpath -count=1 (run focused selections during iteration); go test -race on modified integration packages; relevant launcher/scrollback/couch retained-recording selections. Use make wrappers for environment-dependent integration fixtures. Full required suite is exclusively sdlc local-check run --name suite on a pushed head. Builds use make -k build. Exact regression names and red/green evidence enter the issue Log as each test is introduced.


## Revisions

### 2026-10-15T22:18:28Z — post-landing live acceptance

The operator explicitly requests a live measurement after landing (verified hub receipt e774d4c0-5be6-4c77-8e0a-8e25ceeb3cd4). In addition to pre-merge qualification, after landing measure the selected live slot on the landed binary using process disk-write counters, raw+wrap-event file growth, and diagnostic state inode-change rate. Record exact commit/build, process generation, paths, durations, activity context, before/after values and ratio in the issue Log. Existing baselines are85.8x over10.006s and2187 observed state replacements/minute. Sampling gives a lower bound on inode replacements. Send the result to hub, which separately owns the whole-disk sample. If binary rollout needs a live-slot relaunch, obtain the hub's specific authorization rather than restarting another slot. This addition is acceptance evidence, not approval of the pending scope split or retention policy.


### 2026-10-15T22:23:51Z — hub scope decision; governing implementation revision

Authority: submitted hub receipt8cb950d6-7819-48dd-9813-e10321f20ad8. Raw+events rotation is now unassigned #582, with the full consumer/checkpoint findings published on main. #579 changes diagnostic IO and diagnostic bounds only; raw/events capture production, format and readers remain unchanged. This revision supersedes the original M1/M2 steps wherever they conflict. Post-landing acceptance above still applies.

**Concrete protocol:** Buffer optional wrapper diagnostic records into at most64KiB record-aligned flush batches (threshold64KiB, dirty-only5s timer,256KiB pending and1024 records). Replace append-intent's duplicated base64 Data and two atomic state publications with a compact, bounded, checksummed append reservation in a single stable .append-reservation file under the existing per-log lock. The reservation stores before-generation, proposed length, SHA256 of the batch and time, never payload. Write the reservation once in place, then append the batch, verify observed full length/hash and publish state.json once. Thus a healthy flush has one small reservation write, one payload append and one state publication/rename, with no hot-path fsync. Existing publish/rotate/delete sync boundaries remain.

**Why a reservation remains:** It preserves exact tail provenance before further writes/destructive collection without storing the payload twice. Bare checkpoint-and-adopt would accept foreign tails. A hash validates a complete append; it cannot certify an arbitrary partially written prefix after a process is killed mid-syscall. Such an unverified partial tail is retained and visibly blocks that optional log until reconciled; never guessed, discarded or collected. In a live Write that returns partial progress, the writer still has the original data and can checkpoint the actual prefix using a shortened reservation+hash before returning n+error. Test this distinction explicitly. The compact-reservation/partial-crash limitation is a material protocol tradeoff for hub approval before change-code.

**Reservation ordering and compatibility:** New code first recovers any existing legacy s.Appending payload intent using the existing exact-prefix recovery, then uses compact reservations for subsequent writes. Old binaries either see an exactly checkpointed v1-compatible state or refuse an ahead-of-checkpoint payload; they cannot certify/delete the new uncertain tail. A stale/torn reservation may be replaced only when actual raw identity+size+mtime exactly match the committed Current checkpoint. Otherwise strict bounded parsing, reservation identity/Before equality, length and SHA must all validate before recovering. No conflicting legacy/compact operation may be adopted. A reservation write failure precedes any new payload effect; a checkpoint failure after full append preserves n and permits hash-verified recovery without duplication. Process death after the reservation but before payload leaves the exact old checkpoint; after complete payload permits verified recovery. Same-size edits, foreign/oversized tails, replaced/shrunk files and unknown evidence continue to refuse. The per-log lock fences cooperating writers; no new process scan enters append callbacks.

**New artifact lifecycle:** .append-reservation belongs to one managed diagnostic path, is a bounded regular no-follow/CLOEXEC file (cap4KiB), reused instead of creating per-append files. It is consumed/replaced under the stable log lock and removed during verified empty-log retirement after all pending recovery; unknown files remain untouched. Malformed reservation plus a changed payload is retained, never cleanup authority. Add artifactpath classification/inventory and retirement/interruption tests.

**No-op registration:** Under the registry lock, read/validate the existing exact declaration. Equality skips create/write/rename. A needed directory durability reconfirmation after an uncertain publication is still permitted; do not claim an unchanged inode proves a prior directory sync succeeded. Missing declarations republish before append; altered/malformed declarations refuse rather than silently overwriting unrelated authority.

**Governing concept additions:** CompactReservation and its pure validation/observed-tail classification live in diagnosticlog/append_reservation.go (new) with pure table tests; its bounded filesystem read/write shell lives alongside it or a focused reservation_io.go (new). Existing diskState/legacy recoverAppend remain only for old persisted intent migration. The primary appender in append.go is modified, not a parallel wrapper-only shortcut. All diagnostic writers therefore stop duplicating payload; wrapper buffering amortizes the remaining checkpoint over batches. Other direct writers preserve synchronous n/error semantics. The buffered owner integrates wrap-events, late-open debug and wrapper adaptation only; it does not add workers to short-lived CLI writers.

**Governing M1 steps:**
- [ ] Freeze counted baseline red on a real1000-record stream. Add actual write-boundary counters for reservation/metadata/payload bytes, state renames and syncs (initial Open counted separately), and assert buffered ratio<3, one state rename per flush, no base64 payload in persisted reservation/state, zero idle/no-change writes and no hot syncs. Verify original records in order; the counter is not a physical-APFS estimate.
- [ ] Red reservation/recovery cases: old intent migration; death before reservation, after reservation, after complete payload and after checkpoint; torn/foreign/oversized/same-size/shrink/replacement evidence; live partial-write checkpoint and unverified partial process-crash refusal; cancellation and failure after confirmed n bytes. Keep real killed-publisher regressions, adapting old protocol-specific assertions rather than deleting their safety purpose.
- [ ] Implement compact reservations plus a single checkpoint publication per append batch, preserving exact old-state verification, n semantics and existing rotation/deletion durability. Reservation cleanup participates in empty-log retirement. Prove old persisted intents migrate without duplication and stale reservation cannot certify a newer foreign tail.
- [ ] Red+green pure batch transitions and stateful writer/timer tests: record/byte bounds, dirty interval and size triggers, concurrent admission during flush, partial/full-with-error progress, overload counting, Close during flush, joined worker and no late-open leak. Wire all three wrapper diagnostic emitters through this owner. No raw/event change.
- [ ] Red+green unchanged-registry stable-inode/no-write test, missing/altered/malformed declaration and uncertain durability retry. Update artifactpath inventory immediately, then focused tests/race and make -k build. Run the same live counter/inode method on an authorized scratch wrapper; revise from evidence if>=3x. Update atlas/README with the actual result and crash limitations; close M1 through SDLC fresh review.

**Governing M2 steps:**
- [ ] Preserve the original diagnostic size/count proposal:64MiB current+3 certified immutable generations, reuse exact deletion-intent replay and bounded complete enumeration (maximum64 candidates per pass; incomplete evidence blocks growth). Hard cap prevents further optional growth while proof is unknown; preexisting oversized/uncertified data is retained and reported separately, not claimed bounded retroactively. Diagnostics only; #582 owns raw/archive policy.
- [ ] Red+green the demonstrated protocol-aware deleted-root rotation blocker, preserving all generation/environment fences and legacy unknown-root refusals. Pure retention selection tests plus repeated rotation, proof refusal/recovery, partial traversal and every delete interruption/replacement test.
- [ ] Focused packages/race, make -k build and required pushed-head sdlc local-check run --name suite. Update atlas/README/issue and close M2+issue on SHIP, land via SDLC, then perform the specifically requested post-landing live before/after process-byte/inode measurement and record/report it. Coordinate any live-slot relaunch with the hub. Report PR and full real hashes.

**Crash-loss statement:** Normal diagnostic process death can lose at most the512KiB bounded in-memory queue+in-flight batches; with healthy scheduled IO this is normally the last5s. A complete appended batch can recover by its reservation hash even if checkpoint publication was interrupted. A killed partial syscall is retained as uncertain evidence and blocks that optional log rather than inventing bytes. Ongoing IO failure/overload is counted separately and voids the healthy time claim. No bounded host-power-loss guarantee is introduced without hot fsync. Delivery authority/park/capture sync behavior is unchanged.
