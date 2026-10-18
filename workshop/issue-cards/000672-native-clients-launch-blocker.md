---
id: 000672
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
severity: critical
started: 2026-10-18T11:27:02-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Accept native Zellij client replies without blocking launches

## Problem

Operator launches/relaunches fail on the new build containing #647. Observed by the hub: exact session name 📁1-292 is rejected because ProbeSessionName's successful list-clients reply fails strict decoding with invalid Zellij clients header. Installed Zellij is0.45.1. This blocks new slots and takes priority over #665 M2.
