---
id: 000658
status: open
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T21:32:25-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# pair test suites write ~20 MB/s (couchcmd.test): measure per package and cut, with a counted budget

## Problem

Full test suites are now nearly all of the fleet's disk writes (ops qa 2026-10-17-06): after the rolling relaunch the disk averaged 35.5 MB/s with suites running in 107 of 108 ten-minute samples, about 3.1 TB a day. The SSD reports 7% of rated life used at 89.4 TB written (rated near 1.3 PB), so this pace reaches rated wear in about a year. In a pair suite, `couchcmd.test` alone writes about 20 MB/s of its own (measured 10-16 23:13 with fs_usage-style per-process sampling).
