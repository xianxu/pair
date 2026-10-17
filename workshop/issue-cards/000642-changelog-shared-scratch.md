---
id: 000642
status: open
created: 2026-10-17
updated: 2026-10-17
estimate_hours:
github_issue:
---

# Changelog Lua test shares reload scratch path across concurrent suites

## Problem

During pair#637 PR #396 recorded suite on b7f361ec1a8a097d500baa1241e9b88bf35a3b41, nvim/changelog_test.lua failed `reload replaced buffer with file contents` and `reload put cursor at newest entry`. This happened after a 1466 s queue wait; the test execution stopped after 27 s, before Go tests. The fixture uses the fixed TMPDIR filename pair-changelog-reload-test.md, writes it, reloads it, then removes it. Another concurrent invocation shares/removes that path. Other fixture paths in the same file are fixed too.
