---
id: 000715
status: open
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '8c2733ebacbfebae7c947972128ee0d1e676c02c' # card fields mirrored from issue-cards; edit via sdlc
---

# flake: TestContinuationWriterPublishesExactCheckpointAcrossWorktrees

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchcmd.TestContinuationWriterPublishesExactCheckpointAcrossWorktrees: Same exact head b1cc1870b39b9111f41a86c27ddd16afdb730735: recorded APFS suite attempt d7b39428f6f4d3de2182c233873dbb15 failed couchcmd.TestContinuationWriterPublishesExactCheckpointAcrossWorktrees/warm-reattached (3.25s parent, 1.01s subtest): launcher.TestCouchLaunchPromptTransportHelper TempDir RemoveAll cleanup failed unlinkat .retention: directory not empty. Full suite remains running to collect all failures. Unchanged exact suite checkout /Volumes/TestScratch/local-checks/sdlc-local-3359052142 targeted go test ./cmd/internal/couchcmd -run ^TestContinuationWriterPublishesExactCheckpointAcrossWorktrees$ -count=1 -v -timeout=2m then PASS all five subtests (10.81s parent, 11.096s package; warm-reattached 2.17s), exit 0. No source edits or suite retry. Investigate retention writer lifetime versus helper TempDir teardown; root cause not yet established.

## Spec

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ]

## Log

### 2026-10-19
