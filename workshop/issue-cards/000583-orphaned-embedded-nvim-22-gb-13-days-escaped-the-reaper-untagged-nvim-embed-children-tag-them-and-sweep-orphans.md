---
id: 000583
status: done
created: 2026-10-15
updated: 2026-10-17
estimate_hours: 8.1
github_issue:
started: 2026-10-16T10:00:03-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
actual_hours: 11.15
tracker:
    version: 1
    completion:
        token: close-ed27ce752615
        repository: github.com/xianxu/pair
        reviewed_head: c1f399ca89b85cf38b1284f5a7966973f4a75cd7
        evidence_commit: 273cbeae06cf984cde97a54021781bf0608d281a
        landed_commit: c1a84d097b9582f06e33813a17ca899f3175bce8
---

# Orphaned embedded nvim (22 GB, 13 days) escaped the reaper: untagged 'nvim --embed' children; tag them and sweep orphans

## Problem

2026-10-15: an orphaned `nvim --embed` (pid 15395, started 2026-10-02, cwd `worktree/ariadne-slot2/ariadne`, parent
`launchd`) had grown to **22 GB** over 13 days, contributing to 63 GB of swap. pair already reaps leftover embedded
nvims (`launcher/osruntime.go` ~L945–970: it scans for `nvim … --embed` and reaps those whose slot tag isn't live),
but it only recognises processes whose argv carries a slot tag (`scope.TagFromNvimEmbedArgv`). This one's argv was bare
`nvim --embed`, so it was invisible to the reaper. Whatever spawned it (a review pane, a GUI client or a test harness)
exited without closing it.
