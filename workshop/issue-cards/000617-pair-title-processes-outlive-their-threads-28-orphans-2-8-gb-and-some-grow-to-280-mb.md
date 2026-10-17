---
id: 000617
status: done
created: 2026-10-16
updated: 2026-10-16
estimate_hours: 3.33
github_issue:
started: 2026-10-16T12:54:47-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
actual_hours: 8.28
tracker:
    version: 1
    completion:
        token: close-7eeae96c0967
        repository: github.com/xianxu/pair
        reviewed_head: 20e3a811d207d6f6cd73dbb2df2df18733bd6837
        evidence_commit: eb58397bfe50e5da21a84d52c1ad75f19a5d34ae
        landed_commit: 0fe70c1c7f131335a0ea287de353d95142d2b8b0
---

# pair title processes outlive their threads (28 orphans, 2.8 GB) and some grow to ~280 MB

## Problem

`pair title` processes (one per agent thread) outlive their threads: on 2026-10-16 there were 28 of them, all reparented
to launchd (ppid 1), **2.8 GB resident** in total. Some belong to threads long gone (`1-parli-12`, `1-ducks-11` from
Oct 6; `1-xianxu-dev-22` from Oct 9; `couch-9d6fdf5e…` from Oct 13), and several of the live ones have grown to
~280 MB (`1-ops-33`, `1-pair-8`, `couch-9d6fdf5e…`), so there may also be unbounded growth inside one. Couch's resource
monitor now flags them (orphan-rss alerts for ops:0 and pair:2).
