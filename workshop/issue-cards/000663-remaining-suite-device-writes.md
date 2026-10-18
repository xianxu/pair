---
id: 000663
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
started: 2026-10-18T02:55:47-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Attribute remaining final-Go suite device writes after fixture and cache reductions

## Problem

After pair#658’s fixture/cache reductions, the same complete Make suite still wrote15,098,630,144 physical-device bytes in643.772099s. The largest remaining measured interval is final go test ./... through completion:13,312,307,200 bytes (about88.2% of the device total). Its producer remains unproved. Sampled own-process counters across the entire suite total only1,038,913,536 bytes/825 generations, leaving a14,059,716,608-byte difference that includes missed short-lived processes, filesystem work and other host activity. It must not be assigned wholesale to tests, Git or APFS metadata.
