---
id: 000809
status: codecomplete
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
github_issue:
flake_tests: [github.com/xianxu/pair/cmd/internal/scribecmd.TestRunPropagatesChildExitCode]
started: 2026-10-22T12:41:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:8
    worktree: /Users/xianxu/workspace/worktree/pair-slot8/pair
    repository: github.com/xianxu/pair
actual_hours: 1.75
tracker:
    version: 1
    completion:
        token: close-89a97a48c971
        repository: github.com/xianxu/pair
        reviewed_head: 1266347a901798d554eb428c2b65801d190923fb
        evidence_commit: d127857e888c054cd5ae15bcc8bc6fa05b223931
---

# flake: TestRunPropagatesChildExitCode

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/scribecmd.TestRunPropagatesChildExitCode: pair#797 recorded suite attempt 5df63acf2f4b4b91b3ec17108300a892 on exact ceda3da54a7bd9577dfd529da9d8f89995715053 failed only TestRunPropagatesChildExitCode: exit 42 propagated as 1 at scribecmd_test.go:94; every other package and all script phases finished. Suite duration 12m5s, raw log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/5df63acf2f4b4b91b3ec17108300a892.log SHA256 e2caee9abf54beb8f1ea404ccad0245b5f0bc7828612d57b2f50489ce9027d35. Exact-commit detached checkout qualification go test ./cmd/internal/scribecmd -run ^TestRunPropagatesChildExitCode$ -count=100 -json then passed 100/100, package 4.067s; raw /Volumes/TestScratch/tmp/pair797-scribe-qualification.yhdzFn/qualification.jsonl SHA256 9fcf974492cf9205203ed137b05d66f0d910a89cb931a47398f746e3e4358807. No scribe changes in #797; root cause unclassified, original helper discards Run stderr, so failing fatal error unavailable. An additional code-identical close-head qualification passed100/100 in1.658s but exact-commit qualification above is authoritative. This first unrelated intermittent suite failure is registered per TL-worker #394; no source, timeout or assertion changed.
