---
id: 000668
status: done
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
started: 2026-10-18T14:16:34-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 0.66
tracker:
    version: 1
    completion:
        token: close-13a0a2fac3af
        repository: github.com/xianxu/pair
        reviewed_head: 3853781e7f2f3c95c6b6e6b31448dbb6df3f46fe
        evidence_commit: 4d8ce9832b61a7a061b5f93ba82661f9dcd83d6b
        landed_commit: c5245132b9eb26588c56b05a19f733261368f69e
---

# Native terminal oracle subprocess crashes with SIGSEGV in macOS conformance

## Problem

While landing #659, macOS CI run 39111981159/job 117629869893 failed TestNativeConsoleWrapperZellij/wrapped-zellij at terminal_native_test.go:345: independent xterm oracle: signal: segmentation fault (empty subprocess output). First observed 2026-10-18 08:36:46 UTC, branch head 46e0bd0379ccf8ccabb972fa3b62aea2a9a36871; tested PR merge fb2be7a7ff1a4779bdf61884229748d808e2ae1d. Full failed-job log retained at /tmp/pair659-ci-39111981159.log; public run is https://github.com/xianxu/pair/actions/runs/39111981159/job/117629869893 .
