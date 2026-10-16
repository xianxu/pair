---
id: 000599
status: codecomplete
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
github_issue:
started: 2026-10-16T08:22:54-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 0.08
tracker:
    version: 1
    completion:
        token: close-0b4ea77c31b6
        repository: github.com/xianxu/pair
        reviewed_head: 876f0f80fe55981b58d7fb82a9995e6413cc19a5
        evidence_commit: 610d1b68732feff121320c400bcd56507ffa1b8b
---

# Couch: plain peer deliveries become owed on every agentic document of the owner (duplicate session-sync pings)

## Problem

Since the 10-15 evening Couch restart (pair#573, agentic documents), every peer delivery to ops:0 becomes an owed
event on **every** registered document ops:0 owns, not only the document it concerns. Example: worker reports
b9c9305a and 4e51644c (about pair#597 and #586) were owed on both the TL session file and the QA file
`ops/teams/devinfra/qa/xian-xu/2026-10-15-39-why-reboot-refuses.md`, and Couch pinged "owes a response" for each file
separately. The TL has to fold the same IDs once per file; with several agentic documents open, pings multiply.
