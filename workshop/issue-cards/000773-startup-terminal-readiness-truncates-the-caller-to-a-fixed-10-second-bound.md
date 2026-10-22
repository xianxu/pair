---
id: 000773
status: done
created: 2026-10-21
updated: 2026-10-22
estimate_hours:
github_issue:
severity: high
started: 2026-10-21T08:12:54-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: 6.84
tracker:
    version: 1
    completion:
        token: close-2e1e62eccde1
        repository: github.com/xianxu/pair
        reviewed_head: be30575e0c8d797c4babc08a478f00cdaebbf089
        evidence_commit: 94fcc08b4acb865d518692fd7e9222ccd02e881b
        landed_commit: 4fc8025cd3cbd5b8e2dc6fb3d48992ac21a4dcf6
---

# startup: terminal readiness truncates the caller to a fixed 10-second bound

## Problem

Production terminal readiness imposes a fixed10s deadline even when its caller has a longer remaining deadline. A cold native agent launch under load can be refused before the caller budget expires. Discovered by #744 final native matrix, independently from #772's attach InputEnded failure. This is a startup-budget design defect; do not classify it away as a flake or increase a numeric timeout.
