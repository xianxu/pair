---
id: 000825
status: working
created: 2026-10-22
updated: 2026-10-23
estimate_hours:
github_issue:
flake_tests: [github.com/xianxu/pair/cmd/internal/testcouch/agentfixture.TestCancellationRetirementAllowsIDReuseWithoutInheritedRevocation]
started: 2026-10-23T19:43:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
---

# flake: TestCancellationRetirementAllowsIDReuseWithoutInheritedRevocation

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/testcouch/agentfixture.TestCancellationRetirementAllowsIDReuseWithoutInheritedRevocation: First-occurrence registration explicitly authorized by ops:0 receipt 40b13625-8fe1-42b2-910b-5611746c09b9 after diff classification: #791 does not modify agentfixture. Recorded attempt 6a7b76c66a1bd111dc09a1e528491529 on 8171265c7567f5abadd78882590bf2c380b9997d failed only request_lifetime_test.go:188 cycle10 leaked results: 1 <nil>. Targeted whole top-level test passed 3.428s under Go1.27.3 on docs-only descendant 318e1a91b5cfa301067c5c5c01e4a3b80e963f3d; agentfixture tree is identical c6e34e3c913580f23de43c0a3394aa91ceceb959 and no Go/module changes between heads. This is targeted diagnostic evidence, not an exact-original-head full-suite pass. Original-head local-check resume must independently prove the registered test before any passing status.
