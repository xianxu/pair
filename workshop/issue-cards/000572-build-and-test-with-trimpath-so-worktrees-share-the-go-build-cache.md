---
id: 000572
status: working
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
started: 2026-10-15T10:52:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Build and test with -trimpath so worktrees share the Go build cache

## Problem

On 2026-10-15 the shared Go build cache (`GOCACHE=/Volumes/AgentScratch/go-build`) filled the 186 GB scratch volume
(184 GB), failing every slot's suite. Main multiplier: builds embed each worktree's absolute source path, so the same
package is cached separately for each of ~20 slot worktrees; after a `go clean -cache`, ~30 GB came back within
minutes. The operator approved building with `-trimpath`.
