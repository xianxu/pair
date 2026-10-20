---
id: 000725
status: codecomplete
created: 2026-10-20
updated: 2026-10-20
estimate_hours: 2.21
github_issue:
started: 2026-10-20T02:22:02-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 3.89
tracker:
    version: 1
    completion:
        token: close-a600b6fa92a5
        repository: github.com/xianxu/pair
        reviewed_head: 55d81f74276404be06a6b7ab86e534a4e26bb393
        evidence_commit: 70d7c719a8df48c6e2e4c1f51163371569c8f38e
---

# pair affected-only tests: .sdlc/affected.json + make test-affected, per ariadne#473's settled contract

## Problem

ariadne#473 (test-economy, option B′) lets a close and a landing-train car use an **affected-only** test run as
evidence, with the full suite running once per train. It's off per repo until the repo ships `.sdlc/affected.json`
and a command that runs only the affected tests. pair has neither, so pair keeps running a full suite (~9 min) for
every car, and a docs-only branch (pair#723 on 10-19) can't vouch for a train at all.
