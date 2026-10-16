---
id: 000610
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
severity: high
---

# couchcore package takes ~10 min, at the go test timeout

## Problem

Couchcore takes nearly Go's default ten-minute per-package test timeout. This blocks otherwise verified pair landings under fleet load, including #591 and #589. In #591's recorded integration suite abe0a5ed883f06ffe58305cb45a29c09, couchcore timed out at 600.245 s; every other recipe/package passed. The automatic main comparison 2064bde22de3d40f5d15091fe41ef443 passed, but couchcore alone took 580.585 s. This is a pair item in ops/test-economy; the hub owns that project.
