---
id: 000508
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: 'df8f85a3f6419cb9b35baa04a4d605653b49551f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T16:04:58-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# sessionledger: launch rows embed ~1 MB artifact-boundary snapshots; ledgers reach 15-33 MB with no writer cap

## Problem

Found while diagnosing #507 (2026-10-12, measured on the operator's store). Each Pair thread's session ledger (`<data>/repos/<scope>/ledger-<tag>.jsonl`, written append-only by `sessionledger`) has rows of about 1 MB: a launch row embeds a snapshot of the launch's artifact boundaries (`decodeLaunchArtifactBoundary` lists), and binding rows carry authorization proofs with scanner state. The largest ledgers were 15-33 MB (33 MB in 93 rows), 508 ledger files in all. Nothing caps the writer; the reader comment in `sessioninventory.readOwnerLaunch` says as much ("Both the ledger and its launch boundary snapshots grow without a writer size cap").

Cost: every reader that parses a ledger pays for all of it. #507 measured ~360 ms of CPU per thread per idle-fading probe, ~90% in `ParseLedger`'s strict JSON decode. #507 caches the parse in Couch, which removes the repeated cost there, but each first parse (and every title poller, resume and inventory reader) still decodes megabytes to learn the current launch and binding. The files also never shrink.

ARCH lifecycle: a row that embeds a growing snapshot is growth per event larger than the event. Who needs the boundary snapshot, for how long, and what removes it?

## Spec

## Done when

- A ledger row's size no longer scales with the launch's artifact set (or the snapshot lives where a reader that doesn't need it never decodes it), with a stated bound on ledger growth per launch and a removal path for old generations; existing large ledgers are handled (migrated, compacted or tolerated, stated which).

## Plan

- [ ]

## Log

### 2026-10-12
- 2026-10-12: unclaimed: filed from #507 diagnosis; unassigned per TL
