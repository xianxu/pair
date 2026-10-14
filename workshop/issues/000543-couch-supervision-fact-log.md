---
id: 000543
status: codecomplete
deps: [pair#530]
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours: 2.835
card_mirror: '4e349b95ca32f81e9321bc693d6369374382af63' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T20:50:26-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 3.28
---

# couch: log supervision facts for offline baseline evaluation

## Problem

The rules in pair#530 need factual evidence for a later offline comparison.
Keep recording simple; deriving labels inside Couch adds unnecessary state.

## Spec

Split from pair#530 by TL receipts ddcbf61f-f57a-4aa9-a740-f4614830f5c3
and addd69f2-20e6-4697-8404-8fa5396f0d52. Implement after #530 lands.

- Append one small JSONL row for each event Couch already sees: delivery to the
  hub with a bounded deterministic kind hint (landed, waiting-for-operator,
  progress, unknown), successful session-file write, genuine operator submission,
  and observed slot state change. Add one once-a-minute snapshot row.
- Session-file writes use one best-effort call from couch --session-sync write;
  unavailable telemetry never invalidates a successful file write. No second
  watcher, directory scan, retry queue or deferred write obligation.
- Log timestamps, supervisor generation, opaque entity identities and factual
  metadata, with unknowns explicit; no message bodies, transcripts or secrets.
- Reuse the existing diagnostic writer with a size cap. Logging failures cannot
  block Couch UI or delivery. Describe coverage gaps and retention accurately.
- Labels belong offline: a later script can derive positives/negatives from
  whether a write follows within ten minutes. No online pending candidates,
  negative labeling, censoring state machine, per-entity sampling capacity,
  learned weights or model training in this item.
- Keep parley.nvim#321 as the sole consumer owning the 15-minute file ping.

## Done when

- Event and snapshot rows round-trip with documented schema and bounded size;
  tests cover unavailable telemetry, write failures, rotation and unknown facts.
- File writes and message delivery still succeed when optional logging fails.
- Full make -k completes; atlas and operator-facing documentation describe the
  facts recorded and the limits of the evidence, without claiming trained scores.

## Plan

- [x] Define fact rows and offline ten-minute labels with focused tests.
- [x] Add bounded diagnostic recorder and wire existing observations.
- [x] Verify optional telemetry failures preserve writes and delivery.
- [x] Update docs and complete full-suite verification; close/landing use SDLC gates.

Durable plan: [supervision facts](../plans/000543-supervision-facts-plan.md).

## Log

### 2026-10-13
- 2026-10-13: closed — BR-1 reproduced with two failing tests, fixed strict horizon coverage, all 16 Python tests and real Go-to-Python storage integration pass (/tmp/pair543-br1-green.log, /tmp/pair543-br1-integration.log). Full make -k test completed: six unchanged couchcore creation-lock busy failures, all six passed isolated 26.819s; every other package/check passed (/tmp/pair543-full-suite.log, /tmp/pair543-core-rerun.log). Clean main merge; all five affected packages passed after merge (/tmp/pair543-merged-tests.log). Targeted race passed. Only offline comparison/docs/tests changed after full run. Headless checks; pair:0 untouched; README/atlas/inventory updated.; review verdict: SHIP

- Filed as the dependent fact-log half of #530. Design above supersedes the
  online sampling proposal in #530's original draft. Not started.

- Implementation checkpoint: bounded recorder and all five observation hooks are
  wired; optional authenticated session-write reporting preserves successful writes.
  Opt-in diagnostic hard cap leaves default writers unchanged. Offline reader uses
  closed schemas and bounded input; no online labels or candidate state.
- Focused Go adapter/storage tests passed (/tmp/pair543-integration.log), including
  actual Go JSONL through the Python CLI. Recorder/broker/diagnostic targeted race
  tests passed (/tmp/pair543-race.log); the sessionsync selection had no tests.
  Full protocol, sessionsync and diagnosticlog package runs passed; Python has
  14 passing tests. Snapshot maximum-size and generated-notice tests passed
  (/tmp/pair543-bounds.log). Full make -k test and close review remain.
- Estimate clarification: focused ship wall clock excludes idle waiting; adapter
  allowance includes private-socket/failure tests, and the review allowance covers
  suite supervision and landing. Offline design reuses stdlib JSON/CLI/test tools.

- Full `make -k test` completed (exit 2, /tmp/pair543-full-suite.log).
  All non-Go checks and all Go packages except couchcore passed. Exactly six
  unchanged couchcore tests failed with workspace host creation busy:
  TestManagedCreateReuseRefusesProfileDriftDuringSetup,
  TestReconcileResultListsWhatItSetAside, TestSavedWorkIsCollectedPastRetention,
  TestSetAsideHoldsMatchThePlan, TestSetAsideMovesTheWholeCheckoutAndCanBeRestored,
  TestSetAsideHostRecordsItsBranch. All six passed unchanged in an isolated
  rerun (26.819s, /tmp/pair543-core-rerun.log). The 100ms creation-lock settle
  is outside this patch; no clean-full-suite claim or timeout changes.
- Merged origin/main including #522 cleanly at
  a1f9c73defae97c1976c74e48cdd6f819c723891. No code conflict resolution.
  Post-merge affected packages all passed: couchcmd 72.359s, couchmessage
  7.212s, diagnosticlog 45.114s, sessionsync 12.829s, artifactpath 10.499s
  (/tmp/pair543-merged-tests.log). No project across the
  fleet references pair#543 (sdlc project find); pair:0 untouched.

## Revisions

### 2026-10-13 — dispatched after #530 landing

TL requests the tiny offline labeling script in this item (previous Spec said
later script). Record facts only online. Plan selects existing observation hooks,
private authenticated best-effort write reporting, and the diagnostic writer.
Inspection found MaxBytes is currently a soft rotation threshold; add an opt-in
hard generation bound so failed rotation cannot grow this log indefinitely.
Existing retained generations still follow diagnosticlog GC, not a new cleaner.
No changes to pair:0, currently staged for #522.

### 2026-10-13 — close BR-1: complete inclusive horizons

A negative label now requires contiguous evidence strictly beyond the inclusive
10-minute endpoint. Equal timestamps can still precede a qualifying write, so a
tail ending exactly at the endpoint, or a gap immediately after it, stays unknown.
Both cases reproduced false negatives before the fix; all 16 Python tests now
pass, including permutations and restoration of the missing boundary write.
The change affects only the offline comparison; online facts are unchanged.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against
baseline-v3.1.md, Method A only; source currently marked stale, provisional.
Familiar Go/Python territory. Thorough plan gives x0.2 design discount and 15%
design buffer; implementation is v2 hours x0.4. Recorder and offline reader
reuse diagnosticlog/standard libraries, halving their design base before the
spec discount. No external API discovery or cross-repo propagation.

Order: schema (typed prototype .5/.5 base); recorder (greenfield 1.5/.8);
offline reader (greenfield 1.5/.5); five event adapters (integration 2/1.5);
hard-cap extension (smaller module .3/.5); docs (.1/.2); close review (.1/.5).

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: typed-data-prototype design=0.10 impl=0.20
item: greenfield-go-module design=0.15 impl=0.32
item: greenfield-go-module design=0.15 impl=0.20
item: api-integration design=0.40 impl=0.60
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0.02 impl=0.20
design-buffer: 0.15
total: 2.835
```

Design subtotal 0.90 x1.15 + implementation 1.80 = 2.835 focused hours.
