---
id: 000628
status: working
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T11:26:22-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# Idle UI invariant intermittently returns only 16 of 20 activity results

## Problem

During pair#619 close review on 2026-10-16 around 18:39 PDT, head 3f5f7fed4f03aa2829be24ae339cb473260bf000, the affected-package run failed TestWorkbenchPerformanceIdleUITick (27.72s): workbench_activity_invariant_test.go:77 reported pass results observed=16 and four failed rows (work-01, work-03, work-04, work-20). Couchtty package failed in 65.202s. This is a different assertion from #626's outer 30-second activity-worker wait; shared causality is not established.

Retained failing tool record: /tmp/pair619-review-activity-failure.json; source review session 01a14781-9960-7871-8ade-430a4a8c28fa. The version-controlled #619 close-review sidecar records the failing package run and unchanged targeted passing repeat. First recorded full suite on the same code (f500a1ccf3b165f070d40a2b5478605b6a02b48b) passed in 12m46s, log SHA256 prefix 6746e0ae12fd. Later changes are documentation only. No code or timeout fix was made in #619.
