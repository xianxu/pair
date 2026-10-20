---
id: 000691
status: done
created: 2026-10-18
updated: 2026-10-20
estimate_hours:
github_issue:
started: 2026-10-19T10:56:26-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 2.79
tracker:
    version: 1
    completion:
        token: close-5a68b6127917
        repository: github.com/xianxu/pair
        reviewed_head: a5c7d1cc16da78c133c7ebf4740255058f6d57eb
        evidence_commit: 94975fb9103aad5cbb50901824e5a50e0b6097c2
        landed_commit: 42f9a77973f998a272c2f21362fc21e9e3335d90
---

# couch --keys: allow Esc to dismiss a slash-command panel the hub opened (e.g. /usage)

## Problem

The hub reads Claude quota by running /usage in an idle Claude slot with `couch --keys SLOT clear-line text:/usage submit` and reading it with `couch --peek`. The /usage panel then stays open ("Esc to cancel"), and `couch --keys SLOT esc` refuses ("wait until the agent composer is empty and its turn finishes"), so only the operator can close it. On 2026-10-18 two slots (brain:1 since 12:38, parley.nvim:4) sat with open panels, raising Couch's 10-minute waiting notice and needing the operator's Esc.
