---
id: 000672
status: done
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
actual_hours: 0.24
tracker:
    version: 1
    completion:
        token: close-60e30ab0845d
        repository: github.com/xianxu/pair
        reviewed_head: dd8355c685a473acd7ba76a1d803cf36a6c7898b
        evidence_commit: 155ad43082b79e778aa5687d8f8dcd7faeca84dd
        landed_commit: 5c8030d2545a89c26cd57411695a5d4b3aca229a
---

# Accept native Zellij client replies without blocking launches

## Problem

Operator launches/relaunches fail on the new build containing #647. Observed by the hub: exact session name 📁1-292 is rejected because ProbeSessionName's successful list-clients reply fails strict decoding with invalid Zellij clients header. Installed Zellij is0.45.1. This blocks new slots and takes priority over #665 M2.
