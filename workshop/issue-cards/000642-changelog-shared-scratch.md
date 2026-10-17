---
id: 000642
status: codecomplete
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T06:23:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
actual_hours: 0.41
tracker:
    version: 1
    completion:
        token: close-52d0dc501d2b
        repository: github.com/xianxu/pair
        reviewed_head: c03fe7d4d8121e1b1b1850fb250da0d0c2b121f6
        evidence_commit: 3e3ca5ebe6d8e70281eb0abf1471cd74d0e1b2e0
---

# Changelog Lua test shares reload scratch path across concurrent suites

## Problem

During pair#637 PR #396 recorded suite on b7f361ec1a8a097d500baa1241e9b88bf35a3b41, nvim/changelog_test.lua failed `reload replaced buffer with file contents` and `reload put cursor at newest entry`. This happened after a 1466 s queue wait; the test execution stopped after 27 s, before Go tests. The fixture uses the fixed TMPDIR filename pair-changelog-reload-test.md, writes it, reloads it, then removes it. Another concurrent invocation shares/removes that path. Other fixture paths in the same file are fixed too.
