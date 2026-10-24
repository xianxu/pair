---
id: 000866
status: codecomplete
created: 2026-10-24
updated: 2026-10-24
estimate_hours: 5.5
github_issue:
severity: high
started: 2026-10-24T02:48:13-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
actual_hours: N/A
tracker:
    version: 1
    completion:
        token: close-7e31234274ac
        repository: github.com/xianxu/pair
        reviewed_head: bdbf7367ba61debbe2017f5cf5656e38a7c2e829
        evidence_commit: 0b1ca63d1ac0704b9cab3d0c5bbc90cac727cee0
---

# Wait for Zellij readiness before startup probe clients

## Problem

During pair#863's few shared disposable native runs, Zellij 0.45.1 twice panicked at zellij-server/src/lib.rs:1462:26 (Option::unwrap on absent session_data). A startup probe can connect/remove a client before session metadata exists. Pair should investigate a readiness-qualified probe so its own startup observation does not trigger that upstream lifecycle window. Exact standalone causation is not yet proven; this is the bounded Pair-side avoidance issue, not an upstream bug report.

First occurrence: pair#863 source25a6c4847 native Claude fresh primary startup, before a wrapper started; full matrix FAIL456.39s, exact cleanup passed. Retained source-branch evidence:
- workshop/evidence/000863/native-first-fixed.log, SHA2562320f9563c197a28f32160e7150bcf7e0d685be60ed1896f28aa760fbde4e38c
- workshop/evidence/000863/fresh-fixed-claude-zellij.log (full server capture).

Second occurrence: pinned main4053f3da995746efcc1c6296cffbde4a67a7f343, baseline explicitly started its primary through the operator form, then initial worker1 startup failed before any switch-agent step. TestDisposableStartupArguments FAIL42.33s (Claude34.11s); Pair exited1 before registering. The retained server log contains the same lib.rs1462:26 panic at02:44:40 PDT on2026-10-24. Exact owned generations absent and direct child/IO joined; Codex skipped by existing test loop. Evidence workshop/evidence/000863/main-startup-arguments-4053f3da.log, SHA256d17dbd817ceb9225e85e8ec6ab48f5c10b465af8f2e8390eb2d9e17570692497.

The evidence is committed on pair#863's pushed branch (latest evidence7a71f3353), not yet main; this issue has no dependency on #863 and can use git show that branch. Do not confuse these panics with #863 BR-2's different post-startup switch-agent adoption failure.
