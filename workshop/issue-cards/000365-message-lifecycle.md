---
id: 000365
status: done
created: 2026-10-01
updated: 2026-10-02
estimate_hours: 6.91
github_issue:
started: 2026-10-01T13:09:21-07:00
actual_hours: 1.60
tracker:
    version: 1
    handoff:
        token: move-acd89b761440
        repository: github.com/xianxu/pair
        source_branch: refs/heads/main-slot2
        source_base: 5d3ab19488efe71e27961132ecbed41c66006f21
        source_head: 5d3ab19488efe71e27961132ecbed41c66006f21
        source_path: workshop/issues/000365-message-lifecycle.md
        source_blob: 17a70d1ccf95de2451873d79dbf5f3a2fe580c5c
        destination: workshop/issues/000365-message-lifecycle.md
        main_commit: ac48a27fba4699393507e92e1670aaac66a0d9a2
    completion:
        token: close-c564759b69c7
        repository: github.com/xianxu/pair
        reviewed_head: c44c786ef19a798509d47c58e7cbab714f58cff2
        evidence_commit: 64666d92b8a0c8f885f8d13586d45adf592db954
        landed_commit: 45b1f60c02418796020143dfaf02e745f50b6905
---

# Replace messaging liveness polling with lifecycle events

## Problem
