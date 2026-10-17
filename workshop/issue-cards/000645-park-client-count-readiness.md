---
id: 000645
status: done
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T10:19:44-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
actual_hours: 0.64
tracker:
    version: 1
    completion:
        token: close-ad8c3b980a7d
        repository: github.com/xianxu/pair
        reviewed_head: 0c0e924a7c8323e60f7490c540c7947d292d264d
        evidence_commit: 904406d1e12d8217c91430da27df94a5d51f29f3
        landed_commit: 8f99f3b477be55f0318a7612cc34294daee1fcf6
---

# Park fixture accepts empty client baseline as attachment readiness

## Problem

Read-only #636 audit found that cmd/internal/couchcore/park_lifecycle_live_test.go zellijClientCount directly invokes list-clients, counts every nonempty stdout line (including the header), and accepts successful empty stdout as count0. startProductionParkHelper snapshots that baseline, starts a helper, and accepts any later count greater than baseline as attachment. If baseline is an empty successful reply and the next response is only CLIENT_ID / ZELLIJ_PANE_ID / RUNNING_COMMAND header, the count moves0 to1 even with no attached client. Source behavior establishes this false-readiness sequence; no native reproduction or causal link to #636/#640 is claimed.
