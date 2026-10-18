---
id: 000660
status: working
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
started: 2026-10-18T06:05:07-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 0.86
tracker:
    version: 1
    completion:
        token: close-907a5c8db81a
        repository: github.com/xianxu/pair
        reviewed_head: b575589513eb20a7eec304f2ef803c55a209a069
        evidence_commit: b8fe3d0451f156fca0275100835a52eb4f74076d
        landed_commit: b386574636297f0529f0e4c0d7cbc29b1ff58e4d
---

# Native direct Zellij session disappears after detach

## Problem

MacOS conformance intermittently loses the private direct-Zellij session at persistent reattach. First retained failure: PR422, run39105463917 attempt1, head7de3dec6684644b5114278ac92c53106d45345f2, 2026-10-18 06:57UTC. TestNativeConsoleWrapperZellij/direct-zellij-baseline waited285.86s at terminal_native_test.go:312 for the persistent frame after reattach. child.raw contained only "No session with the name ... found!"; receipts had BOOT/CPR/input and STATE=<same boot>:1 before detach. The wrapped subtest subsequently retained PID:nonce and advanced counter1 to2, before the package's five-minute alarm exhausted the remaining budget. Treat that alarm as a cascade from the first wait, not a separate causal finding.

Run: https://github.com/xianxu/pair/actions/runs/39105463917/job/117610220213

Same issue code passed conformance in runs39105228904 (d9c9bab5bb60d3cf68b753bbe079606e8139ac0f) and39105295728 (eeedca6731051353d4314ab056da6ba2b200d0f9); only documentation differs through the failed head. The terminal fixture is unchanged by #658. This establishes intermittency, not its cause.

This differs from #603's wrapped real-shell-actor discovery wait, and #361's initial layout/fixture mismatch. Here initial launch/input/state succeeded and reattach reported an absent session.
