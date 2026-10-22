---
id: 000788
status: open
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
github_issue:
---

# flake: TestAnOperatorSwitchWaitsBehindAtMostTheRunningAttempt

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/couchtty.TestAnOperatorSwitchWaitsBehindAtMostTheRunningAttempt: Targeted same-exact-head fail->pass proof on plain origin/main d2540fd6ae6eaa06e8a90217b285c22116247558 exported with git archive (no source edits): count100 JSON probe has98 PASS/2 FAIL, both resume:couch-second dispatched before switch (console_reattach_test.go298). Failed probe /Volumes/TestScratch/tmp/pair-779-main-probe.NSzmu1/probe.log SHA2566171b58601875415f0cdf0a93a5b7d2de38c4d888a7d6be11826f87d2b7f323f. Subsequent exact single-test run on that unchanged tree exited0; /Volumes/TestScratch/tmp/pair-779-main-probe.NSzmu1/probe-pass.log. Candidate #779 integrated Console package hit the same unchanged failure, count50 reproduced2/50; wrap/native/confirmation-specific targets PASS2.348s. This is targeted local evidence, not a claimed full-suite pass. Track separately from #779; investigate whether the cross-lane strict dispatch-order oracle remains valid. No timeout changes or product fix.
