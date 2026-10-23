---
id: 000845
status: open
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
github_issue:
started: 2026-10-23T14:16:27-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# Native Couch conformance isolation omits Git upload-pack helper

## Problem

Pair's native Couch conformance bridge fails before readiness on macOS because isolated workspace provisioning cannot execute git-upload-pack for its local private upstream. This blocks ariadne#577 native Couch service conformance evidence; no TL/worker replay or successful service qualification was claimed.
