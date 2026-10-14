---
id: 000543
status: working
deps: [pair#530]
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours: 2.835
card_mirror: 'b7665addbea28e28f03c2bb746f55459d3613490' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T20:50:26-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
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

- [ ] Define fact rows and offline ten-minute labels with focused tests.
- [ ] Add bounded diagnostic recorder and wire existing observations.
- [ ] Verify optional telemetry failures preserve writes and delivery.
- [ ] Update docs, full suite, close review and land.

Durable plan: [supervision facts](../plans/000543-supervision-facts-plan.md).

## Log

### 2026-10-13

- Filed as the dependent fact-log half of #530. Design above supersedes the
  online sampling proposal in #530's original draft. Not started.

## Revisions

### 2026-10-13 — dispatched after #530 landing

TL requests the tiny offline labeling script in this item (previous Spec said
later script). Record facts only online. Plan selects existing observation hooks,
private authenticated best-effort write reporting, and the diagnostic writer.
Inspection found MaxBytes is currently a soft rotation threshold; add an opt-in
hard generation bound so failed rotation cannot grow this log indefinitely.
Existing retained generations still follow diagnosticlog GC, not a new cleaner.
No changes to pair:0, currently staged for #522.

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
