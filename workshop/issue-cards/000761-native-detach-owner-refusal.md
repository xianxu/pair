---
id: 000761
status: done
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
actual_hours: 6.28
tracker:
    version: 1
    completion:
        token: close-cf95b0ec91a2
        repository: github.com/xianxu/pair
        reviewed_head: 01c4f854f9588a64ae956d14c962a09770b24766
        evidence_commit: e88939dab432dbb0a2b1af0d1567d58c9255fdde
        landed_commit: e68a6c2a35e078f673b86ecad9543e4cebc096d3
---

# Couch native detach intermittently loses post-client ownership proof

## Problem

During #744 native qualification, real Couch leave-detach refused after stopping a worker client: `Couch: leave couch: detach 1-fixture-2: observe Pair session before retiring its incarnation: session "📁 1-2" ownership unresolved`. The remaining diagnostic was clipped by the terminal row. This preserves a real post-client-exit ownership refusal; no owner check was weakened.
