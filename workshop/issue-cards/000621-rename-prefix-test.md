---
id: 000621
status: codecomplete
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T10:48:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: 3.38
tracker:
    version: 1
    completion:
        token: close-dcf883cb86dc
        repository: github.com/xianxu/pair
        reviewed_head: 0df8e47804a65518ba75f0d4fa4ed06bf611998b
        evidence_commit: 198e5c3b44db69b78ef018312067c05c592705a5
---

# Rename shell test fails legacy prefix case

## Problem

The standalone tests/pair-rename.sh fails at T7, which expects pair rename pair-t1 pair-t2 to strip the pair- prefix. The current CLI looks for the literal pair-t1 and reports no files found although the fixture contains tag t1. This is pre-existing: the original script from the pre-isolation #617 branch HEAD fails identically in a disposable empty environment. The isolation sweep found it; other cases T1–6/T8–9 pass57 assertions when run separately.
