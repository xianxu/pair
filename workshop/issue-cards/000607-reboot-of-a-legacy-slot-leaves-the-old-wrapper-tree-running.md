---
id: 000607
status: open
created: 2026-10-15
updated: 2026-10-15
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
---

# Reboot of a legacy slot leaves the old wrapper tree running

## Problem

`couch --reboot` of a legacy slot (no proven binding) starts the fresh wrapper but leaves the old process tree running: pair:1 (wrapper 86287, couch-9d6fdf5eecace248), pair:6 (34570, 1-pair-47) and ariadne:3 (94921, 1-ariadne-32) kept wrapper, codex, session-watch and engine; #587's monitor flagged a 363 MB orphan; the TL SIGTERMed them by hand. Cause: in executePreparedRecovery, a Fresh selection whose confirmed-terminal preparer fails (legacy readiness/attribution) proceeds with an EMPTY stop set ("unproved remnants left untouched").
