---
id: 000797
status: codecomplete
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
github_issue:
severity: high
started: 2026-10-22T09:32:31-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 3.27
tracker:
    version: 1
    completion:
        token: close-0ebfe8955817
        repository: github.com/xianxu/pair
        reviewed_head: 993b4106bb1ed8790c059e7fa247195361721b3f
        evidence_commit: 2846810c195d47e79649f4b178d81754e3ded5b7
---

# Live reboot refuses owned pane as ambiguous

## Problem

The owning hub can obtain a live worker's exact reboot confirmation offer, but acknowledging it fails during session quiescence: `ownership unresolved: unknown pane owner: pane has no unambiguous actual command`. The current incarnation remains unreplaceable through the public recovery path. This is separate from #791's busy/banner admission problem, where the same reboot refusal appears only in the Log.
