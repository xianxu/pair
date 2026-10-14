# Supervision facts implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3; use bounded subagents where appropriate and execute with superpowers-executing-plans. One atomic issue, one close review.

**Goal:** Record bounded factual observations for offline evaluation of the ten-minute write baseline, without changing Couch decisions.

**Architecture:** The message service owns one best-effort JSONL recorder using diagnosticlog. Existing event hooks offer rows to a bounded writer buffer; the existing wait tick offers minute snapshots. A small Python standard-library script derives labels offline.

**Tech stack:** Go, existing Unix-socket protocol and diagnosticlog; Python 3 unittest.

## Core concepts

| Pure entity | Lives in | Status |
|---|---|---|
| supervisionFact and bounded kind hint | cmd/internal/couchcmd/supervision_facts.go | new |
| supervisionFactSlot, factLogLifecycle.advance | cmd/internal/couchcmd/supervision_facts.go | new |
| factSlotState, factStates | cmd/internal/couchcmd/supervision_fact_events.go | new |
| parse_fact, WriteIndex, derive_labels | scripts/supervision-labels/labels.py | new |

Facts carry schema version, random supervisor lifetime, sequence, UTC time and
monotonic elapsed milliseconds. Subject and sender are opaque lifetime-salted
hashes of slot identity; incarnation hashes distinguish replacement wrappers.
No paths, bodies, transcript fragments, nonces or repository names are logged.
Delivery hints use a bounded leading-line classifier: landed, waiting-for-operator,
progress or unknown. Hints are observations of wording, never intent judgments.

| Integration | Lives in | Status | Wraps |
|---|---|---|---|
| supervisionFactLog | cmd/internal/couchcmd/supervision_factlog.go | new | diagnosticlog, bounded in-memory writer channel |
| event adapters | cmd/internal/couchcmd/supervision_fact_events.go (new); message_service.go, supervision.go (modified) | new/modified | existing registry, delivery callback and wait tick |
| accepted submission observer | cmd/internal/couchmessage/broker.go | modified | accepted monotonic submission evidence |
| write report | cmd/internal/couchcmd/supervision_write.go (new); session_sync.go, couchmessage/protocol.go, sessionsync/sync.go (modified) | new/modified | successful guarded write, one private-socket call |
| read_facts, main | scripts/supervision-labels/labels.py | new | bounded binary file input and stdout JSONL |
| optional hard generation bound | cmd/internal/diagnosticlog/writer.go | modified | existing rotation/retention owner |

There is one recorder per supervisor and no online candidate or label state.
Only observed slot projections and minute cadence are retained, bounded by the
existing 128-actor limit. The broker keeps ownership of accepted submissions.

## Event contract

| Event | Evidence and fields |
|---|---|
| hub_delivery | Existing endpoint callback to current hub; sender/recipient, message-ID hash, kind hint, generated flag, confirmed/uncertain/cancelled outcome. No claim about queued messages that never reach delivery. |
| session_write | Successful Sync.Write result carries file owner; CLI reports it once with current caller credentials. Accept only current caller matching owner. Owner slot is the label subject; no file path is sent. |
| operator_submit | Broker accepts a larger submission counter. Record counter delta, including handshake/activity reconciliation; a jump is one observation of that delta, not invented individual events. |
| slot_state | Registry's observed connected/settled/picker projection changes, including removal and replacement. Null means unknown. No attach/resume behavior changes. |
| snapshot | Once per minute from existing tick: bounded current slot facts (including hub), known/unknown, resting, settled, pending, quiet age, submission count. Empty inventory is explicit. |

The new session-write operation uses the existing request envelope, closed
validation and authenticated caller resolution. The successful file write is
already committed: missing/older supervisor, timeout, refusal or log failure
cannot turn it into a failed write. One call, at most 200 ms, no retry. A caller
may report its own successful write; this is diagnostic evidence, not authority.
Cross-owner writes and shell calls without a current identity are unobserved.

## Bounds, failure, lifecycle

- Default stable path: supervisor namespace/supervision.jsonl, registered with
  the resolved Pair diagnostic root. Private files, one random lifetime per start.
- Producers only construct bounded values and try to enqueue; they never do disk
  IO or wait for writer capacity. One writer, 256 rows buffered, max 32 KiB/row
  (ordinary event rows much smaller; snapshots cover at most 128 actors).
- Sequence increments before enqueue so dropped or failed writes leave visible
  gaps. Snapshot facts report cumulative drops. No retry/replay of dropped rows.
- Add an opt-in diagnosticlog hard-MaxBytes policy: rotate by existing rules;
  if rotation cannot proceed, refuse further bytes at the generation limit.
  Use the existing 64 MiB generation size and daily rotation. This is not a
  total-store cap: historical generations retain the existing 365-day GC policy.
- Writer opens and writes off the UI/delivery path; an unavailable sink drops
  facts. Shutdown closes admission, drops buffered tail on cancellation and joins
  the one writer before closing its diagnostic handle. No per-event goroutines.
  Filesystem syscall latency is not claimed to be cancellable.

## Offline labels

`python3 scripts/supervision-labels/labels.py LOG...` reads JSONL segments and
emits one JSON row per non-generated hub-delivery observation. Label is true
when a matching subject write is observed within (delivery, delivery+10 min],
false only when an intact same-lifetime sequence covers the whole horizon, and
null for insufficient evidence (truncation, sequence gap, restart or malformed
rows). False means no write observed in that window, not proof of neglect.
Use monotonic elapsed time; UTC is for humans. Never join different lifetimes.
Uncertain delivery remains explicitly uncertain in output. Sort/deduplicate
rows by lifetime/sequence for rotated segment input; conflicting duplicates are
gaps. The script records no online state and writes output only to stdout.

## Implementation and verification

- [x] TDD fact schema/hints and offline labels: threshold edges, unknown hints,
  no secret-bearing fields, same/different subject, missing horizon, restart,
  gaps, duplicates, out-of-order segments and malformed rows. Add script SKILL.md.
- [x] TDD optional writer hard cap with failed rotation; recorder serialization,
  bounded queue, dropped rows, write/open failure, shutdown and private temporary
  directory rotation. Reuse diagnosticlog tests/portable backend, no production paths.
- [x] Wire all adapters; private-socket integration tests cover successful and
  rejected CLI write reports, older/unavailable supervisor and successful file
  writes despite telemetry failure. Controlled endpoints prove delivery remains
  successful when logging fails; submission reconciliation and slot transitions
  produce exactly the observed facts, including unknowns/replacements.
- [ ] Update README, atlas/couch.md and source inventory for new artifact writers.
  Run focused Go/Python tests, race tests, and complete `make -k test`; record
  evidence, close review, merge main and land via sdlc. No pair:0 stage.

## Architectural checks

ARCH-DRY: one watcher and existing diagnostic retention; ARCH-PURE: schema/hints
and offline labels separate from IO; ARCH-PURPOSE: all five fact classes plus
usable offline labels. ARCH-MOCK: private directories/socket and stateful
blocked/failing writers through the real adapters. ARCH-CONSTRAINTS: bounded
row/channel/actor sizes and nonblocking producers. ARCH-SECURE: authenticated
telemetry, lifetime-salted identities, no content. ARCH-ORDER: accepted evidence
only, monotonic time/sequence, no retries; shutdown owns the writer. ARCH-FUNERAL:
recorder dies with supervisor, buffers discarded, existing registered GC owns
rotated segments. No new authoritative state machine.

## Revisions

### 2026-10-13 — plan review PQ-1/2/3

**Recorder order (PQ-1).** `factLogLifecycle.advance(event)` is the small pure
resource-lifecycle transition, not a supervision candidate model. States are
opening, active, unavailable, closed. Opened changes opening to active; open
failure changes it to unavailable; close changes any state to closed. Offers
in opening/active attempt one enqueue; unavailable/closed drop. Write failure
records a drop but stays active (a later independent event may work); no retry.
Late open completion after close closes its returned handle and cannot reopen.

`supervisionFactLog.offer` holds one short mutex while assigning monotonically
nondecreasing elapsed time, increasing sequence, copying the cumulative drop
count and trying the bounded channel. It performs no IO. Writer completion and
close take the same mutex for lifecycle/drop bookkeeping. The channel is never
closed: cancellation ends the worker, avoiding send-on-closed races. Close marks
closed, cancels, joins worker; the worker discards buffered rows and closes its
owned sink. Offers after close do nothing. IO executes outside that mutex; a
controlled blocked sink/open seam proves producer responsiveness and late-close
ordering. Sink open failure is permanent for that lifetime, not a retry loop.

**Offline envelope (PQ-2).** `read_facts(paths)` uses bounded binary readline
(max 32 KiB), limits all input to 64 MiB and 100,000 rows, and refuses oversized
batches with an actionable nonzero error before labeling. Users select a shorter
time window/segment set. `parse_fact` validates schema version, closed event and
hint enums, integer nonnegative sequence/time fields (booleans are not integers),
opaque identity shapes, known fields and bounded snapshot arrays. Invalid or
unsupported records are skipped with a stderr count and make all negative
labels in that input batch unknown; valid positive observations remain usable.
Conflicting duplicates and backwards elapsed time similarly invalidate negative
coverage for that lifetime. Contiguous ordered sequence through the horizon is
required for false; incomplete tails remain null. Sorting is bounded by the
batch envelope. Parsing/IO is separate from pure `derive_labels(records, gaps)`.

**Function-level test strategy (PQ-3).**

| Production function | Adversarial strategy | Guard proved |
|---|---|---|
| factKindHint, encodeFact | control-bearing/oversized Unicode and unknown phrases | closed hints and content-free bounded rows |
| parse_fact, read_facts | malformed types/schema/oversized line and total input | validation before inference, bounded batch |
| derive_labels | permuted duplicate/gapped records around exactly ten minutes | no unsupported negative, positive subject/time match |
| factLogLifecycle.advance, supervisionFactLog.offer | generated offer/open/fail/close sequences plus controlled blocked IO | terminal close, ordered sequence, nonblocking admission |
| diagnosticlog.Writer.Write hard-limit option | threshold-sized writes with failed rotation/proof | no bytes beyond selected cap; unchanged default policy |

Integration guarantees remain successful delivery/write despite failed telemetry,
one authenticated write-report attempt, and accepted observation facts only.

### 2026-10-13 — implementation reconciliation

The operator-submit row records the accepted cumulative submission counter,
rather than its delta: this preserves the broker observation directly, leaving
any delta calculation offline. Snapshot empty inventories serialize as an
explicit empty array. Input integers are bounded to the Go wire ranges (uint64
counters, int64 elapsed times), and unknown snapshot values remain null. Docs,
source inventory and the full-suite Python target are updated. Focused tests
pass; full suite and the single close review remain outstanding.

The Core concepts inventory now names the implemented lifecycle, slot projection,
validated parser and write index; adapter and reader paths reflect their final
files. This is inventory reconciliation, not a scope expansion.
