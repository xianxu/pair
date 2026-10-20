---
id: 000706
status: codecomplete
created: 2026-10-19
updated: 2026-10-20
estimate_hours: 4.73
github_issue:
started: 2026-10-19T20:44:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
actual_hours: 2.29
tracker:
    version: 1
    completion:
        token: close-ceb1850f0037
        repository: github.com/xianxu/pair
        reviewed_head: edad1fb9759600c2ad25aceb49e787dce7deffb7
        evidence_commit: d7d2b39f705eab6a8dcd4c33cbb3a13c96a9ff18
---

# Drive backup: store bundles in year/month folders instead of one flat folder

## Problem

The Drive backup destination `backups/pair-history/` is one flat folder (pair#678), growing by roughly 5,000-10,000 files a year and never shrinking (retention deletes local copies only). The operator (2026-10-19): rather than wait for the 20,000-file warning (#705), put files into year/month folders now.
