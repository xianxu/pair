---
id: 000868
status: open
created: 2026-10-24
updated: 2026-10-24
estimate_hours:
github_issue:
---

# flake: TestCancellationRetirementAllowsIDReuseWithoutInheritedRevocation

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/testcouch/agentfixture.TestCancellationRetirementAllowsIDReuseWithoutInheritedRevocation: failed in attempt de7d151179704cbaed6e949992bc8d8b, passed on the same head in attempt 3fabfaabea51c1ee78825a876be040a6
