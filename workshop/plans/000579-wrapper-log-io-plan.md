# Wrapper log IO implementation plan

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
