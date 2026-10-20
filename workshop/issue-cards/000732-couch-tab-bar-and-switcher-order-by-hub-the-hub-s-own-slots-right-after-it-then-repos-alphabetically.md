---
id: 000732
status: codecomplete
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
started: 2026-10-20T10:15:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
actual_hours: 1.84
tracker:
    version: 1
    completion:
        token: close-725c386cf6dd
        repository: github.com/xianxu/pair
        reviewed_head: 0ab34d5f2bc870cc2b2b5a5ac9e6c17d85b32b46
        evidence_commit: 1e185c3880f16b2bb99203c4d4c73ab6ddc67630
---

# couch: tab bar and switcher order: by hub, the hub's own slots right after it, then repos alphabetically

## Problem

Couch orders the tab bar (and the switcher) so a hub repo's own worker slots land far from the hub: today
`ops  ariadne :1 :2 … brain  ops:1  pair :1 …`, with ops:1 after brain, away from ops:0 (operator, 10-20, screenshot
in ops's session file). Now that ops repos carry code (ops's charter, 10-20), hubs have worker slots of their own,
and they belong next to the hub.
