---
id: 000662
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9d181b2aaecb076f0f5feb58fdcf7111d0759dab' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T03:46:30-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# Review observation diagnostic positive control times out in real BufWritePre resolver

## Problem

The normal make test shell/review phase can fail in ObservationFixtureDiagnosticsTest.test_positive_control_retains_wanted_fresh_and_request_order. The fixture calls a real readiness resolver during BufWritePre; a late synchronous resolver was killed at its existing 2500 ms deadline, producing an empty result and a write refusal. It blocked pair#658’s whole-device after measurement before the final Go recipe.

## Spec

Determine why this positive control sometimes exhausts the resolver budget, distinguishing host capacity, the fixture’s own synchronization, and product RPC reentrancy. Preserve genuine write-refusal and branch-identity safety; no blanket timeout bump or blind retry.

First test the concrete budget-composition hypothesis: identity.lua waits 2500 ms, resolveIdentityWithSnapshot owns a shared two-second context, and OSRuntime.GitContext allows another second of os/exec WaitDelay after cancellation. Retained call-level evidence does not establish which Git/pipe phase consumed the historical timeout. Add phase evidence only at an owned test seam, and demonstrate the candidate with controlled process/descriptor handshakes before selecting its fix. Preserve the existing successful brief-output-holder case from #613 and all final branch/HEAD/index checks.

If the controlled reproduction confirms a product budget defect, keep command execution and output draining within one explicit owner/budget rather than increasing the Lua deadline. If it instead demonstrates legitimate product refusal under host delay, isolate the fixture's ordering assertions from real resolver speed and preserve real resolver conformance separately; bring that scope revision to the hub before code. In either case, a timeout with empty stderr must produce a useful refusal diagnostic while leaving the document and activation unchanged.

## Done when

- Root cause is demonstrated with retained resolver phase evidence and a deterministic regression.
- The positive control passes under that controlled condition while changed-branch, unknown identity and deadline refusal controls remain effective.

## Plan

- [ ] Post this plan to the hub before code. Treat the budget mismatch as a hypothesis, not the historical root cause.
- [ ] Add a deterministic owned process/descriptor-holder regression beside TestOSGitContextToleratesBrieflyHeldStdout. Handshake on child start, held output, context cancellation and release; retain phase events. Use a package-derived hang ceiling only for a stuck test, not a fixed performance assertion. Do not introduce CPU stress or repeat suites to chase the flake.
- [ ] Exercise the same controlled failure through identity decoding and the write guard: timeout/unknown identity refuses with a nonempty reason; changed branch remains refused; exact document bytes and activation remain intact. Retain the positive completed-result gate sequence and brief-holder success control.
- [ ] With that evidence, choose the smallest fix at the budget owner. No deadline increase, skipped final identity checks, stale-authority cache or blind retry. If the candidate is disproved or requires a new mechanism, stop and revise with the hub.
- [ ] Keep the core decision pure (ARCH-PURE), reuse the existing Runtime and callback gate seams (ARCH-DRY), and retain real controlled native-process coverage (ARCH-MOCK). Enumerate execution, cancellation, pipe drain and teardown as distinct phases (ARCH-ORDER). All owned children/descriptor holders have unconditional release/termination/reaping (ARCH-FUNERAL).
- [ ] Bound each real diagnostic invocation at 60 seconds with owned cleanup, preserving the product's two-second context and 2500 ms Lua wait. No UI or operator slot is touched (ARCH-SECURE). No new production fan-out or scan is planned: resolved observations use nine Git queries; normal pre/post-write guards use eighteen. If collection changes, count calls at N and 2N rather than asserting host speed (ARCH-CONSTRAINTS).
- [ ] If a mechanical budget mistake class is confirmed, send the hub the precise ariadne/pkg/golint rule proposal with #662 evidence; the catalog is Go-AST based and peer-owned, so do not silently add a Lua-only exception or edit that repository under this issue.
- [ ] Run focused Go/Lua/diagnostic tests, then the recorded full suite and normal close/merge gates. A fixture-only fix uses owned headless validation; any product/editor behavior change needs a live-check arrangement approved by the hub. Report the landing.

Existing visible states remain unchanged: a verified current identity permits the guarded write; unknown/deadline/changed-branch observations refuse it, retaining edits and activation. Recovery is retrying the action after a fresh successful observation. No new user or automation state is proposed. This plan fixes the refusal diagnostic as part of the same failing path, not a new authorization policy (ARCH-PURPOSE).

## Revisions

- 2026-10-18: Hub assigned #662 after parking #663. Initial investigation identifies a falsifiable deadline/drain composition issue and independent empty-stderr diagnostic loss. Prove the phase interaction before choosing a product versus fixture fix; no historical-phase attribution is claimed.

## Log

2026-10-18 planning: claimed and ran start-plan from main on 000662-review-observation-resolver-timeout. Read-only call-graph inspection: BufWritePre -> restoration:guard -> identity.resolve -> review readiness --resolve -> resolveIdentityWithSnapshot. This route does not invoke the headless-Neovim classifier. Successful resolution performs nine sequential Git calls, plus two index-generation checks; ordinary writes call guards before and after. The async callback gate holds completed results and is already released before the failing synchronous write, so it does not deliberately hold that native process. The historical trace records whole resolver calls only; it cannot distinguish native startup, Git execution, pipe draining and scheduler delay. OSRuntime.GitContext sets WaitDelay to one second despite the collector's shared two-second context and outer 2500 ms Lua wait. The empty diagnostic is explained by result.stderr or fallback accepting the empty string in Lua. These are source findings; no new reproduction, test run or implementation has occurred yet. Peer ariadne AGENTS.local.md was read to inspect its lint catalog; MEMORY.md is absent. The catalog handles Go ASTs, not arbitrary Lua/Python files.

Observed 2026-10-18 around01:02 PDT in pair#658’s same-body make -k WF_TEST_WEIGHT=2 test, source HEAD91d46ca2f5c0c3b1d5537fbb5b28c2a95b079bb1. Owned quiet two-token device diagnostic: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair658-device-after.SjFHBp/device-suite/make.log lines855–905. Make exit2 after312.427012s; all ten diagnostic tests enumerated, exactly this positive control failed. Remaining Make prerequisites completed; final go test ./... recipe was suppressed by failed prerequisites.

Error: nvim/review.lua:910 BufWritePre, review: write refused. Fresh trace’s final synchronous call23 ran2578.387417ms, timeout2500, code124, signal9, stdout_bytes0, decode_error invalid review resolver JSON syntax. Earlier asynchronous call22 succeeded in2092.532375ms. The original full-suite baseline passed this diagnostic. git diff from baseline b08a927f6fed942fdbb4aef84b09cf07b2a32f47 to failing source shows no changes to nvim/review.lua, tests/review-observation-test.sh, tests/review-observation-diagnostics-test.py or tests/lib/review_observation_gate.lua. These are intermittent controls, not proof of the exact root cause. No rerun or deadline change yet. Retain raw failure JSON embedded in log and source/body/queue/result receipts alongside it.

The initial native Zellij CI flake in #660 is separate; this is not its missing-session signature. Pair#659’s external-tool budget work is also not assumed to fix this review resolver.

Cross-reference pair#623 (native-process timeout budget family) and pair#659 (external-tool budgets; owned by pair:7). Hub approved exactly one issue-backed unchanged device rerun after the recorded suite; if that rerun fails before final Go again, stop and report, with no repeated attempts.
