---
id: 000761
status: codecomplete
created: 2026-10-21
updated: 2026-10-21
estimate_hours: 2.3535
github_issue:
started: 2026-10-21T00:55:41-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
actual_hours: 5.72
tracker:
    version: 1
    completion:
        token: close-71ddea7fad07
        repository: github.com/xianxu/pair
        reviewed_head: 1c3cfb28316d17d49f983716a0c13e28fb18c842
        evidence_commit: d2c4d1e057e67eadebbcae68f6510af453864e79
---

# Couch native detach intermittently loses post-client ownership proof

## Problem

During #744 native qualification, real Couch leave-detach refused after stopping a worker client: `Couch: leave couch: detach 1-fixture-2: observe Pair session before retiring its incarnation: session "📁 1-2" ownership unresolved`. The remaining diagnostic was clipped by the terminal row. This preserves a real post-client-exit ownership refusal; no owner check was weakened.
