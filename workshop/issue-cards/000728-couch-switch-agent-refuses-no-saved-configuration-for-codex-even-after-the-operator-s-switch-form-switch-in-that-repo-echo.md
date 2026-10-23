---
id: 000728
status: codecomplete
created: 2026-10-20
updated: 2026-10-22
estimate_hours: 14.58
github_issue:
started: 2026-10-20T23:31:35-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
actual_hours: 6.89
tracker:
    version: 1
    completion:
        token: close-aa945e44c061
        repository: github.com/xianxu/pair
        reviewed_head: fba4fd7863e67c7f0df87005aa967fa5bc18040f
        evidence_commit: 3f7f7732cb93ccb9022986d52bf6152f83dce2ef
---

# couch --switch-agent refuses 'no saved configuration for codex' even after the operator's switch-form switch in that repo (echo)

## Problem

ops-echo:0 (10-20 ~09:20): `couch --switch-agent echo:2 --agent codex --confirm` refuses with "no saved configuration
for codex; the operator must configure it in the switch form first" (requests 8e69e8d8 and 0ad9d2c3), and still
refused after the operator switched echo:0 to Codex by hand in the switch form. So the form's switch doesn't save the
per-repo configuration the CLI reads (or saves it under a key the CLI doesn't look up). In the devinfra repos the
same CLI switch worked (ariadne:0, ariadne:5, parley.nvim:3, :4 on 10-20), so a saved codex configuration exists there.
