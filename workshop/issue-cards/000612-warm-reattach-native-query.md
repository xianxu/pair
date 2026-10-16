---
id: 000612
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
severity: medium
---

# Warm reattach acceptance intermittently resolves native binding

## Problem

TestSwitcherWarmReattachWithoutNativeBindingReachesTerminal intermittently fails at cmd/internal/couchcmd/warm_reattach_test.go:142: warm acceptance resolved native binding 1 times (expected zero). The first observed run was #589's integrated-head full suite at 1a2e5f787b4021ff410f83896e2745f14c6b8f0c. It was the only failing test; all other packages passed, including couchcore in 548.732s. Full failure log: /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/a3db77884e3a91eb92e51b263f4d6a8a.log (sha256 prefix 118c2364bb97).

One automatic full-suite main baseline at d59420ed5eee5e3b94113bb8cac0c82cff5d4a59 passed. That single passing comparison was insufficient: the requested repeated whole-package matrix reproduces the same failure on both main and #589's head. The failure therefore does not require #589's changed dormant-readmission fixture.
