---
id: 000642
status: working
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
---

# Changelog Lua test shares reload scratch path across concurrent suites

## Problem

During pair#637 PR #396 recorded suite on b7f361ec1a8a097d500baa1241e9b88bf35a3b41, nvim/changelog_test.lua failed `reload replaced buffer with file contents` and `reload put cursor at newest entry`. This happened after a 1466 s queue wait; the test execution stopped after 27 s, before Go tests. The fixture uses the fixed TMPDIR filename pair-changelog-reload-test.md, writes it, reloads it, then removes it. Another concurrent invocation shares/removes that path. Other fixture paths in the same file are fixed too.
