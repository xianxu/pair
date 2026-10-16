---
id: 000607
status: done
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
github_issue:
severity: critical
started: 2026-10-15T23:54:34-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 2.22
tracker:
    version: 1
    completion:
        token: close-95a2accfb462
        repository: github.com/xianxu/pair
        reviewed_head: c8e2c29a75356cb65202fd8b0f1afcc52f4b262f
        evidence_commit: f2b234218027ab0319c25c92e2dd705602819ec4
        landed_commit: e6019bc5e0fe2ccf9317b78e2ccbe9c9b026b17e
---

# Reboot of a legacy slot leaves the old wrapper tree running

## Problem

`couch --reboot` of a legacy slot (no proven binding) starts the fresh wrapper but leaves the old process tree running: pair:1 (wrapper 86287, couch-9d6fdf5eecace248), pair:6 (34570, 1-pair-47) and ariadne:3 (94921, 1-ariadne-32) kept wrapper, codex, session-watch and engine; #587's monitor flagged a 363 MB orphan; the TL SIGTERMed them by hand. Cause: in executePreparedRecovery, a Fresh selection whose confirmed-terminal preparer fails (legacy readiness/attribution) proceeds with an EMPTY stop set ("unproved remnants left untouched").
