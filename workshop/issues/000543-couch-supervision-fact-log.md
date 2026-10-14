---
id: 000543
status: open
deps: [pair#530]
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '1dce3aa30824e1c21cc84ad8edf6ae730cbc5ec7' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-13

- Filed as the dependent fact-log half of #530. Design above supersedes the
  online sampling proposal in #530's original draft. Not started.
