---
id: 000681
status: done
created: 2026-10-18
updated: 2026-10-19
estimate_hours:
github_issue:
severity: high
started: 2026-10-18T18:19:31-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
actual_hours: 0.86
tracker:
    version: 1
    completion:
        token: close-6f2ae7d894a6
        repository: github.com/xianxu/pair
        reviewed_head: 13bab42b594631c392438fcb89aac5837d3dece8
        evidence_commit: 31d3bbe5100b3172d72ad6abe62728aff2bc1552
        landed_commit: 8059ab717462c1553fce5afc66dea265c03be888
---

# Recover park inventory exceeds CLI admission deadline

## Problem

On the operator build containing #651, confirmed CLI recover-park can time out before receiving a receipt because it builds the full fleet inventory inside a two-second admission exchange. The same inventory during ordinary resume preparation took 3.055 and 5.233 seconds. Two requests returned context deadline exceeded / outcome uncertain, but caller-scoped operation-status returned unknown for both; neither recovery ran. This strands an otherwise recoverable interrupted park behind its named CLI recovery command.
