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
actual_hours: 2.88
tracker:
    version: 1
    completion:
        token: close-d19892a0e029
        repository: github.com/xianxu/pair
        reviewed_head: 716e5f77d63ecceb0121a3dd796f203722959e7b
        evidence_commit: 31ef133911333fb88b5ef3994da1b1c9f84e4b69
---

# Live reboot refuses owned pane as ambiguous

## Problem

The owning hub can obtain a live worker's exact reboot confirmation offer, but acknowledging it fails during session quiescence: `ownership unresolved: unknown pane owner: pane has no unambiguous actual command`. The current incarnation remains unreplaceable through the public recovery path. This is separate from #791's busy/banner admission problem, where the same reboot refusal appears only in the Log.
