---
id: 000738
status: codecomplete
created: 2026-10-20
updated: 2026-10-21
estimate_hours:
github_issue:
started: 2026-10-20T19:58:34-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
actual_hours: 9.91
tracker:
    version: 1
    completion:
        token: close-28b26ab254ef
        repository: github.com/xianxu/pair
        reviewed_head: 71623e8e30520581a091fdd4536230ed43190e09
        evidence_commit: cb4c3354739b9e0c32e97a7fcc6df338601b0ae4
---

# Affected package graph falls back to full suite before runtime bundle preparation

## Problem

During #726's first recorded affected check after integrating #725, a fresh local-check worktree could not load the Go package graph:

`cmd/internal/runtimebundle/embed.go:8:41: pattern assets/runtime/files: no matching files found`

`sdlc local-check run --name affected` selected the full suite instead, with reason `package graph unavailable`. The checkout had not generated the runtime bundle before the Go graph query. This is safe (the suite still runs) but prevents the affected runner from selecting the intended smaller check in that fresh-checkout case.
