---
id: 000237
status: codecomplete
deps: []
github_issue:
created: 2026-09-12
updated: 2026-09-12
estimate_hours: 0.35
started: 2026-09-12T17:39:11-07:00
actual_hours: 0.04
---

# relaunch refuses once a thread's ledger passes 8 MiB: the session query caps the whole ledger file instead of each record

## Problem

Operator report, 2026-09-12, from couch's relaunch action on `couch-5003fd4f6c74f514`:

```
error: relaunch couch-5003fd4f6c74f514: could not resolve its native session binding: session inventory read exceeds limit
```

Measured: that thread's ledger, `ledger-couch-5003fd4f6c74f514.jsonl`, is
8,482,765 bytes — 39 rows, of which 14 `launch` rows carry 8.47 MB between
them (each ~600 KB; longest line 677,523 bytes). The 8 MiB it just crossed
is a **whole-file** cap:

- `sessioninventory.QuerySessionContext` (`query.go:99`) reads the owner
  ledger with `runtime.ReadFile(ledger, 8<<20)`. Over the cap `ReadFile`
  returns `ErrReadLimit`, the query returns it as an error, and every caller
  turns it into a refusal: couch relaunch (`couchcore/relaunch.go:120`), and
  also the launcher's own resume (`launcher/osruntime.go:705`), `reviewcmd`,
  `opener`, `slugcmd`, `titlepoller` (on a polling cadence), `contextcmd`, and
  the activity CLI — nine callers, one read.
- `RecoverPairBindings` (`pair_inventory.go:69`) reads the same ledger family
  with a 64 MiB whole-file cap; same class, later cliff.

The ledger is append-only JSONL, one record per line. The package already
has the right bound for that shape: transcripts are read through
`visitJSONLinesAt`, which caps **each record** (`jsonRecordLimit`, 8 MiB)
and never the file. The ledger reads predate that helper and kept a whole-
file cap, so a thread that has been relaunched enough times stops being
resumable — the file grows with every launch by construction, so this is a
cliff every long-lived thread reaches, not an anomaly.

Why the rows are 600 KB each is a separate defect — every launch record
embeds a boundary snapshot of the whole storage root — filed as #238. This
issue is the read: a per-file cap on a file that is defined to grow is the
wrong bound regardless of row size.

## Spec

- Read a ledger the way the package reads any append-only JSONL artifact:
  bounded **per record**, unbounded per file. One helper,
  `readJSONLArtifact(runtime, artifact, recordLimit)`, built on `ReadAt`
  chunks like `visitJSONLinesAt`, returning the raw bytes for
  `sessionledger.ParseLedger` (which keeps its own tolerance for an
  unterminated last line — a ledger being appended concurrently must stay
  readable, so the helper returns the partial tail rather than erroring).
- Both ledger reads (`QuerySessionContext`, `RecoverPairBindings`) use it
  with `jsonRecordLimit`. The `log-*.md` and `config-*.json` reads keep the
  whole-file cap: neither is a record stream.
- `ErrReadLimit` still fires for a single record over `jsonRecordLimit`.
- One chunked framer under both consumers (`frameJSONLArtifact`): the
  transcript visitor and the ledger reader differ only in what they do with
  a CR and with an unterminated tail, so those two decisions live in the
  consumers and the loop lives once (ARCH-DRY).
- Envelope (ARCH-CONSTRAINTS): the ledger is still parsed whole, in memory,
  on every query — and `titlepoller` queries on a cadence, so an 8.5 MB
  ledger costs that much parsing per poll. This issue removes the cliff; the
  bound on the cost is #238 (shrink the launch row), not a new cap here.

## Done when

- A ledger over 8 MiB made of ordinary records resolves through
  `QuerySession`; a ledger with one record over `jsonRecordLimit` still
  returns `ErrReadLimit`. Same pair for `RecoverPairBindings`.
- A ledger with an unterminated last line parses as before (the partial
  row is a malformed ordinal, not an error).
- The operator's thread relaunches.

## Plan

- [x] `readJSONLArtifact` in `scan_helpers.go` next to `visitJSONLinesAt`; tests for over-cap file / over-cap record / partial tail
- [x] `QuerySessionContext` + `RecoverPairBindings` read the ledger through it
- [x] Regression tests at both call sites with a >8 MiB ledger
- [x] Verify against the operator's real ledger with `pair session-inventory` and a relaunch — owner query resolves on the fixed build and fails on a main control build (Log); the relaunch itself is the operator's step

## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* Bugfix with the shape already in the package; design at ×0.2, impl at 40% of v2.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module  design=0.04 impl=0.16
item: milestone-review   design=0.00 impl=0.14
design-buffer: 0.15
total: 0.35
```

## Log

### 2026-09-12
- 2026-09-12: closed — Regressions red on the old read (confirmed by stashing the fix) then green: TestQuerySessionReadsALedgerLargerThanTheOldWholeFileCap, TestRecoverPairBindingsReadsALedgerLargerThanTheOldWholeFileCap; guards TestQuerySessionStillRefusesASingleOversizedLedgerRecord, TestQuerySessionToleratesALedgerMidAppend; FuzzReadJSONLArtifactAgreesWithTheLineFramer 30s / 1.9M execs clean. Full make test green outside the sandbox. LIVE on the operator ledger (8,482,765 bytes): PAIR_SCOPE_KEY=e108517d46ab4575 pair session-inventory --agent claude --scope current --owner couch-5003fd4f6c74f514 prints the bound native id on the fixed build and exits 2 (owner query failed) on a control build from main. The couch relaunch itself is the operator step after install.; review verdict: SHIP

- Filed from the operator's couch relaunch failure. Traced the message to
  `sessioninventory.ErrReadLimit` from `QuerySessionContext`'s 8 MiB
  `ReadFile` of the owner ledger; measured the ledger at 8,482,765 bytes.
  Checked the other read-limit sites first (transcript lines over 8 MiB:
  none in `~/.claude/projects`; the thread's own `wrap-events` file is 757
  KB) before landing on the ledger. Row profile: 14 launch rows at ~600 KB
  each → #238.
- Fix: one chunked framer, `frameJSONLArtifact`, now sits under both
  `visitJSONLinesAt` (transcripts; strips CR, treats an unterminated tail as
  a truncated record) and the new `readJSONLArtifact` (ledgers; returns the
  body byte for byte, tail included, for `ParseLedger`). Both ledger reads
  use it with `jsonRecordLimit`; the `log-*.md` / `config-*.json` reads keep
  the whole-file cap. Regressions: >8 MiB ledger resolves through
  `QuerySession` and >64 MiB through `RecoverPairBindings` (both red on the
  old read, confirmed by stashing the fix); one record over the cap still
  `ErrReadLimit`; a mid-append ledger with a partial last line resolves with
  the partial row diagnosed malformed. Fuzz
  `FuzzReadJSONLArtifactAgreesWithTheLineFramer` pins reader/framer
  agreement on the bound, the body, and the tail.
- Test oracle correction: three existing tests asserted "no body reads" as
  `ReadAt == 0` over every artifact, which the ledger's chunked read now
  trips. They assert `ReadAt == 0` over the native roots instead
  (`FakeRuntime.OperationCountForRoot`), which is the claim they were
  making; the ledger read is not a transcript body read.
- Live: `PAIR_SCOPE_KEY=e108517d46ab4575 pair session-inventory --agent
  claude --scope current --owner couch-5003fd4f6c74f514` on the operator's
  8,482,765-byte ledger prints the bound native id on the fixed build and
  `owner query failed` (exit 2) on a control build from main.
