---
id: 000627
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
started: 2026-10-16T18:58:29-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Live shortcut fixture races Zellij cache removal during TempDir cleanup

## Problem

First observed 2026-10-17T00:28:29Z on pair#604 PR #381 integrated head f8e3e1c7970f0ab6505478e74ec823007d2c3a5e, GitHub Actions run 38986306428, conformance job 117258474451. TestAgentShortcutInputConformanceLive passed every input subtest, then failed during Go TempDir cleanup: unlinkat .../002/Library/Caches/org.Zellij-Contributors.Zellij: directory not empty.

Failed log: /tmp/pair604-integrated-conformance-failure.log. Durable CI evidence: https://github.com/xianxu/pair/actions/runs/38986306428/job/117258474451 . Previous PR-head conformance run 38983105918 passed unchanged clipboard implementation. No repeat of the failed job yet.
