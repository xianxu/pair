---
id: 000825
status: working
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '4791762c50f1085e7a427f9538712a95156867ae' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T19:43:32-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:9
    worktree: /Users/xianxu/workspace/worktree/pair-slot9/pair
    repository: github.com/xianxu/pair
flake_tests: [github.com/xianxu/pair/cmd/internal/testcouch/agentfixture.TestCancellationRetirementAllowsIDReuseWithoutInheritedRevocation]
flow: {kind: quick, provenance: inferred, spec: "210d6aa8", done: "8753b161"}
---

# flake: TestCancellationRetirementAllowsIDReuseWithoutInheritedRevocation

## Problem

Known flaky tests (#394), registered with same-head fail→pass evidence:

- github.com/xianxu/pair/cmd/internal/testcouch/agentfixture.TestCancellationRetirementAllowsIDReuseWithoutInheritedRevocation: First-occurrence registration explicitly authorized by ops:0 receipt 40b13625-8fe1-42b2-910b-5611746c09b9 after diff classification: #791 does not modify agentfixture. Recorded attempt 6a7b76c66a1bd111dc09a1e528491529 on 8171265c7567f5abadd78882590bf2c380b9997d failed only request_lifetime_test.go:188 cycle10 leaked results: 1 <nil>. Targeted whole top-level test passed 3.428s under Go1.27.3 on docs-only descendant 318e1a91b5cfa301067c5c5c01e4a3b80e963f3d; agentfixture tree is identical c6e34e3c913580f23de43c0a3394aa91ceceb959 and no Go/module changes between heads. This is targeted diagnostic evidence, not an exact-original-head full-suite pass. Original-head local-check resume must independently prove the registered test before any passing status.

## Spec

Retirement assertions count mailbox records, excluding atomic publication scratch while it is still owned by the publisher. Preserve the command/result/cancellation fences, joined-child guarantee, exact ID reuse and capacity behavior. Reuse countMailboxEntries rather than changing runtime publication/retirement or removing another producer's scratch. A genuine leftover result or cancellation must still fail. No production state, format, deadline or retry changes.

Alternatives: joining or locking the publisher inside RetireResult would add a runtime mechanism to satisfy a test oracle that disagrees with the existing capacity contract; waiting/retrying the directory-empty assertion would hide that mismatch. Use the existing counter and deterministic held-scratch controls instead (ARCH-DRY, ARCH-PURE, ARCH-PURPOSE).

## Done when

- The root cause is fixed (never a retry loop or a longer timeout); closing this issue unregisters the tests.
- Held publisher scratch cannot fail either live-publication retirement assertion; real mailbox records still count as leaks.
- Existing cancellation, uncertain-result preservation, ID reuse, joined-child and bounded-capacity controls pass, with race and the required recorded pre-close check.

## Plan

- [ ] Pin the false-positive state with a held .publish-* file in the reuse and joined-child retirement controls; observe RED with the raw-directory oracle.
- [ ] Use the existing countMailboxEntries for both live-publication retirement assertions, include entry names on failure, and add scratch-versus-genuine-record negative controls.
- [ ] Run targeted lifecycle/capacity controls, repeated reuse and race, lint and reader inventory; prepare pushed pre-close verification and close. Recorded pre-close pass, mandatory fresh review and landing remain downstream gates until their own results.

Quick flow, test-only correction. ARCH-ORDER: producer owns temporary hard-link cleanup after final result becomes visible; RetireResult removes cancellation then command then final result under the admission lock. No new state or effects. ARCH-CONSTRAINTS: preserve MaxRequests+1 cycles and existing limits; no new hot path or growth driver, and no timing-based waits. ARCH-FUNERAL: test-owned held scratch dies with t.TempDir; never unlink another publisher's temporary file. ARCH-EVOLVE/SECURE: production wire fields and identity validation unchanged. Existing stateful fixture remains the integration seam (ARCH-MOCK).

## Log

### 2026-10-22

### 2026-10-23 — Root-cause work with retained #841 check

Claimed and started from fresh main84d84dd0298c7b1cf2d02ed7b90d127fcc4952bb after #852 landed. Latest #852 integration suite reproduced this test at cycle66 with one leaked result, followed by managed same-head exact-test pass3.297s (parent82a29a2a767de75660985214b5feb1e8, child64235cd95e13c908ddccc3466535cc55); root-cause repair is required, never a retry or timeout bump.

TL295d506f-33e6-4f3b-a09a-89a60e9a0537 authorizes both open claims: #825 active root-cause work and #841 accepted SHIP/high awaiting recorded check/landing. #841 check lease-1223841125 remains live in runner81109 at detached HEAD954e3caea7078371c91c7523e86a70bdad751658 (/Volumes/TestScratch/local-checks/sdlc-local-1464763110); verified detached and matching dependency/policy/build inputs before switching. Its docs-only tip is af5785f92. Return to board #841 on pass; do not cancel/restart it. ariadne#580 tracks overlapping-time attribution. A bounded read-only agent investigates result publication/retirement ordering while this session reproduces and traces the lifecycle.
