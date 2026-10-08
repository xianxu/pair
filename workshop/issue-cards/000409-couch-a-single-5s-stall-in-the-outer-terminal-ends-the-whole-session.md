---
id: 000409
status: codecomplete
created: 2026-10-07
updated: 2026-10-07
estimate_hours: 2.87
github_issue:
started: 2026-10-07T21:52:21-07:00
claimant:
    operator: T
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
actual_hours: 1.80
tracker:
    version: 1
    handoff:
        token: move-01efb205fef9
        repository: github.com/xianxu/pair
        source_branch: refs/heads/main
        source_base: f566d361d9f2e20270446ca4c7b612fd0f39c5b2
        source_head: f566d361d9f2e20270446ca4c7b612fd0f39c5b2
        source_path: workshop/issues/000409-couch-a-single-5s-stall-in-the-outer-terminal-ends-the-whole-session.md
        source_blob: c3c7b585811684aba99fc403cb783730a859853e
        destination: workshop/issues/000409-couch-a-single-5s-stall-in-the-outer-terminal-ends-the-whole-session.md
        main_commit: 3d84cd5133544b1515845a505b25ef12fc028101
    completion:
        token: close-398a12fcc69c
        repository: github.com/xianxu/pair
        reviewed_head: 765deca0b68d2b31629538b800695a0198f0fc76
        evidence_commit: 1995dc48fcd06eb6b1793aa18a1f3d98c539aa37
---

# couch: full-screen repaint per frame overruns a backgrounded terminal, and the resulting exit is silent

## Problem
