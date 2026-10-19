---
id: 000688
status: done
created: 2026-10-18
updated: 2026-10-19
estimate_hours: 4.27
github_issue:
started: 2026-10-19T01:53:02-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: 2.67
tracker:
    version: 1
    completion:
        token: close-2bee4c9cb7c3
        repository: github.com/xianxu/pair
        reviewed_head: f4b39f45d45569edd2f716ef9cd5e2efc2010d91
        evidence_commit: b642b986f9761816025cd8e57d57b6895d4de8db
        landed_commit: 55bc07db58b8ef1bb257eef1b90f2622464e31ae
---

# Codex database snapshots can starve under continuous writes

## Problem

During #678 operator live verification on 2026-10-18, pass 1 skipped one Codex database and pass 2 skipped logs_2.sqlite, state_5.sqlite and thread_history_1.sqlite with 'changed during copy'. The current collector copies then compares the live database/WAL before producing a private backup; continuous Codex writes can repeatedly invalidate that observation.

This is not a claim that all database backup always fails. Read-only local catalog inspection found 10 codex-database records: logs_2.sqlite and state_5.sqlite have fresh bundles verified 2026-10-19T01:53:42Z and 01:54:02Z (18:53/18:54 PDT). The latest thread_history_1.sqlite bundle is still 2026-10-16T18:22:42Z (11:22 PDT), roughly 55 hours old at the investigation, 114658949 encrypted bytes. #616 fixed compressed-size admission and demonstrated one successful thread-history bundle; it did not change snapshot consistency or guarantee freshness under ongoing writes.
