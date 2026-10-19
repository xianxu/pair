---
id: 000693
status: codecomplete
created: 2026-10-18
updated: 2026-10-19
estimate_hours:
github_issue:
severity: medium
started: 2026-10-19T03:57:11-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
actual_hours: 0.50
tracker:
    version: 1
    completion:
        token: close-cfc2ae50e856
        repository: github.com/xianxu/pair
        reviewed_head: 877401b241a651bd16c3dc6c14def7447e0917e5
        evidence_commit: aab4df76cf610fa78618155aa8bc89a3441f75a2
---

# flake: lifecycle primary-slot notice can race verified workspace publication

## Problem

The #681 integrated suite on 807115355d853aa5ef041dd212fd90f0142d4994 failed TestLifecyclePublisherRegisteredPrimarySlotIsNotOrdinaryContent after1139.55s with "no lifecycle message" (lifecycle_notices_test.go:556). The package reached its20m deadline as the next, unrelated TestMessageAdmissionRejectsMismatchesThenGoesDormant/root began (0s). All other Go packages and prior script/editor checks completed successfully.
