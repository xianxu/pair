---
id: 000724
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '4c3a2c99e28b69c8e1d373792d4936dcfdf5c245' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T03:50:24-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/wrapcmd.TestTerminalModelControlObserverBoundsStringStorage, github.com/xianxu/pair/cmd/internal/wrapcmd.TestWrapperTitleLeaseNotInherited]
flow: {kind: quick, provenance: inferred, spec: "c166e865", done: "a0bbe98b"}
---

# flake: TestTerminalModelControlObserverBoundsStringStorage, TestWrapperTitleLeaseNotInherited

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/wrapcmd.TestTerminalModelControlObserverBoundsStringStorage: Same exact head 44d7bdccf8124f25d15082a7286ba1c0ff0f4a46, no source edits: broad go test ./cmd/internal/couchcmd ./cmd/internal/couchmessage ./cmd/internal/wrapcmd -count=1 failed wrapcmd after196.827s (session62009): BoundsStringStorage large OSC allocations26 versus small6; TitleLeaseNotInherited timed out3.07s before leased title appeared. Focused go test ./cmd/internal/wrapcmd -run ^(TestTerminalModelControlObserverBoundsStringStorage|TestWrapperTitleLeaseNotInherited)$ -count=3 -v then passed both three times in3.814s (session76552). Source and test files are unchanged from origin/main. This is observed fail-to-pass evidence, not a root-cause diagnosis; investigate allocation measurement interference and process startup budget under concurrent machine load. Recorded full suite on this head remains queued; no suite failure has been waived.
- github.com/xianxu/pair/cmd/internal/wrapcmd.TestWrapperTitleLeaseNotInherited: Same exact head 44d7bdccf8124f25d15082a7286ba1c0ff0f4a46, no source edits: broad go test ./cmd/internal/couchcmd ./cmd/internal/couchmessage ./cmd/internal/wrapcmd -count=1 failed wrapcmd after196.827s (session62009): BoundsStringStorage large OSC allocations26 versus small6; TitleLeaseNotInherited timed out3.07s before leased title appeared. Focused go test ./cmd/internal/wrapcmd -run ^(TestTerminalModelControlObserverBoundsStringStorage|TestWrapperTitleLeaseNotInherited)$ -count=3 -v then passed both three times in3.814s (session76552). Source and test files are unchanged from origin/main. This is observed fail-to-pass evidence, not a root-cause diagnosis; investigate allocation measurement interference and process startup budget under concurrent machine load. Recorded full suite on this head remains queued; no suite failure has been waived.

## Spec

Fix two test-fixture/oracle defects without changing wrapper or parser behavior. The allocation oracle uses process-global Mallocs, so unrelated goroutines can produce an apparent size-dependent allocation increase. Keep its strict small/large bound, run the measurement in one self-exec test helper with no competing package tests, and verify the reported measuring PID is the actual distinct helper PID. Include 1 MiB and 2 MiB payloads and retain visibility rejection. Reject weakening the check to retained capacity only.

The title fixture reads its agent PID once and samples its identity only in that same branch. An unavailable identity is therefore cached forever, even after the title is live. Keep observing until a nonempty identity is established; then retain that exact PID/identity for cleanup. Add a test-only observation seam and phase diagnostics, preserving the existing 3 s startup/owner-death bounds and the real native-child non-inheritance assertion. Unknown identity remains pending, not gone or safe to kill. No product state/API or persisted format changes (ARCH-PURE, ARCH-ORDER, ARCH-SECURE, ARCH-EVOLVE).

Prove the fixes through deterministic interference, then a bounded cohort on compiled test binaries with at most four workload processes; use at most two extra load workers, no fleet-wide load flood. Count failure/pass outcomes and record actual host load. Native children remain the existing disposable fixtures, and every additional workload/helper is joined or cancelled. One fixed number of allocation measurements per payload, independent of package activity; one identity observation per fixture poll, stopping once established (ARCH-CONSTRAINTS, ARCH-FUNERAL). Reuse existing fixture/dispatch seams (ARCH-DRY, ARCH-MOCK). Do not claim these controlled probes identify the exact historical interleaving; improve diagnostics to distinguish unknown agent identity from genuinely absent title publication (ARCH-PURPOSE).

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.

## Plan

- [ ] Keep controlled red evidence: process-wide allocation contamination and one unavailable agent identity followed by a live title. Preserve original failure receipts.
- [ ] Isolate only allocation sampling in a self-exec test helper; retain the strict allocation comparison, 1 MiB/2 MiB controls and visibility checks. Verify the measuring process identity; a temporary unbounded-parser mutation must fail the oracle.
- [ ] Extract the title fixture's sampling step in test-only code, reobserve unavailable agent identity until established, and add a deterministic unknown-then-live test plus useful timeout phase diagnostics. Keep existing 3 s bounds and generation-fenced cleanup.
- [ ] Run targeted tests and a bounded before/after cohort with no more than four workload processes. Record outcomes and load; then publish, recorded full suite, close review and land on SHIP through SDLC. No visible staging needed for test-only changes; report outcome to hub.


## Log

### 2026-10-20

### 2026-10-20 — claim, root-cause probes and quick plan

Claimed unowned card in pair:7; start-plan succeeded after restoring only the two CI workflows regenerated by its weave refresh (known pending #699; initial tree was clean). No CI workflow changes are part of this issue. A read-only subagent traced the title startup chain: zellij is after title PID publication and is not a valid pre-publication delay injection. The fake native shell publishes PID before exec sleep; Darwin identity can transiently be unavailable during exec.

Baseline compiled wrapcmd binary, the two exact tests, count=3: all six invocations passed, allocation 0.16–0.19s, title 0.34–1.09s. Host load before probes: 38.83/37.26/29.85. Build used -p=1. No broad load generator ran.

Disposable Go overlay probes (no repository implementation edits) under /Volumes/TestScratch/tmp/pair-slot7/pair724.RgEPoV: TestProbe724GlobalAllocationInterference failed in 0.622s wall, with small=6 and large=26 after synchronously scheduling exactly twenty unrelated allocations per large call. The parser and its allocation code were unchanged. This reproduces the historical numeric symptom as global-counter misattribution. TestProbe724TransientAgentIdentity failed in 3.357s wall (test 3.03s): identity reader was called exactly once, title PID file existed, and a subsequent real identity was available for fenced cleanup. Thus one unavailable observation poisons fixture readiness. These are deterministic red controls, not a claim that the original broad log recorded these exact interleavings.

Quick test-only scope; no production timeout, allocation tolerance, sandbox setting, native fixture policy or persisted format changes. Plan goes to hub before change-code. If the diff reveals production behavior defects or requires new state/mechanism, stop and revise rather than stretching this plan.
