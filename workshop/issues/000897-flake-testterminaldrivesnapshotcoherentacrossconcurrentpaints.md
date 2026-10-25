---
id: 000897
status: open
deps: []
github_issue:
created: 2026-10-25
updated: 2026-10-25
estimate_hours:
card_mirror: '51c62688ba0c2c437b8a9843d4669aac59033dbb' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestTerminalDriveSnapshotCoherentAcrossConcurrentPaints

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/wrapcmd.TestTerminalDriveSnapshotCoherentAcrossConcurrentPaints: Exact-head fail/pass on228ce8f808ed7723331a5ec2559a156f6b45eb2e: integration attempt4e5a9fb7f3bcf56333a82921fe7b9dc2 failed exit2/12m25s, failed log .git/sdlc/test-stats/failed/4e5a9fb7f3bcf56333a82921fe7b9dc2.log SHA256 prefixf4d8f54eb95f; later targeted go test ./cmd/internal/wrapcmd -run ^TestTerminalDriveSnapshotCoherentAcrossConcurrentPaints$ -count=1 -v PASS0.03s/package0.425s exit0, no source change. Local evidence outside test-stats: /Volumes/TestScratch/tmp/pair-slot4/pair887-wrap-flake.U9Vjev/same-head-targeted.log. Deterministic two-feed overlay control0.405s proves exact gen2 completefalse cellB tail[A,B cursor] valid scrollback+screen result; Tail first row is historical A, cursor row2 is currentB. Fixture incorrectly assumes first tail row current; test/source byte-identical to main baseline (which passed diagnostically). Separate unclaimed fixture fix, not887 cache cleanup. No CI rerun or timeout change.

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-25
