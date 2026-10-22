---
id: 000743
status: codecomplete
created: 2026-10-20
updated: 2026-10-22
estimate_hours: 5.06
github_issue:
started: 2026-10-20T14:15:16-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 18.48
tracker:
    version: 1
    completion:
        token: close-b3de59f22535
        repository: github.com/xianxu/pair
        reviewed_head: 7fd996711dbfeed70263e272238bc85ea168b0c7
        evidence_commit: ad6a69d9178a3d72500cd9b930bb272bdb9b3c75
---

# conformance: run couch-zellij-conformance in a local Tart VM as a declared local check; GitHub job becomes a non-required main/weekly backstop

## Problem

`couch-zellij-conformance` (real Zellij 0.45.1, nvim and PTYs on a clean macOS runner) is a required PR check for
Couch, terminal, launcher and Zellij paths. On 10-20 GitHub's macOS arm64 capacity cancelled it three times before
any runner started ("not acquired by Runner of type hosted"), gating pair landings (#726, #732, #735) on GitHub's
capacity instead of on our tests. The operator's rule is local-first verification.
