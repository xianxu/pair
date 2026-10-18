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
flow: {kind: quick, provenance: inferred, spec: "80c4bffa", done: "e7714447"}
---

# Review observation diagnostic positive control times out in real BufWritePre resolver

## Problem

The normal make test shell/review phase can fail in ObservationFixtureDiagnosticsTest.test_positive_control_retains_wanted_fresh_and_request_order. The fixture calls a real readiness resolver during BufWritePre; a late synchronous resolver was killed at its existing 2500 ms deadline, producing an empty result and a write refusal. It blocked pair#658’s whole-device after measurement before the final Go recipe.

## Spec

Determine why this positive control sometimes exhausts the resolver budget, distinguishing host capacity, the fixture’s own synchronization, and product RPC reentrancy. Preserve genuine write-refusal and branch-identity safety; no blanket timeout bump or blind retry.

First test the concrete budget-composition hypothesis: identity.lua waits 2500 ms, resolveIdentityWithSnapshot owns a shared two-second context, and OSRuntime.GitContext allows another second of os/exec WaitDelay after cancellation. Retained call-level evidence does not establish which Git/pipe phase consumed the historical timeout. Add phase evidence only at an owned test seam, and demonstrate the candidate with controlled process/descriptor handshakes before selecting its fix. Preserve the existing successful brief-output-holder case from #613 and all final branch/HEAD/index checks.

If the controlled reproduction confirms a product budget defect, keep command execution and output draining within one explicit owner/budget rather than increasing the Lua deadline. If it instead demonstrates legitimate product refusal under host delay, isolate the fixture's ordering assertions from real resolver speed and preserve real resolver conformance separately; bring that scope revision to the hub before code. In either case, a timeout with empty stderr must produce a useful refusal diagnostic while leaving the document and activation unchanged.

## Done when

- The flaky ordering oracle's dependence on real resolver speed is demonstrated by retained timeout evidence and a deterministic native-execution escape control. No unseen historical Git phase is claimed as the product cause.
- The positive control passes under that controlled condition while changed-branch, unknown identity and deadline refusal controls remain effective.
- Real Git collection and native sync/async deadline conformance remain separate tests; production deadlines and final identity/index checks are unchanged.

## Plan

- [x] Post the plan before code; retain the budget mismatch as a hypothesis, not the historical root cause.
- [x] Resolve the descriptor-holder proposal: rejected as artificial historical evidence. Retain the one real Git phase trace instead; no stress or repeated suite was used to chase the flake.
- [x] Exercise controlled timeout/unknown/changed-branch refusals through decoding and the guard, retaining exact bytes, activation, callback-gate ordering and brief-holder success coverage.
- [x] Obtain hub approval for fixture isolation instead of an unproved production budget fix. Install a shared-catalog stateful resolver at the existing seam, trap native execution escapes and count waits instead of 250 ms ceilings. Preserve production deadlines and final checks.
- [x] Keep decisions pure and use the existing callback gate (ARCH-PURE/DRY). Separate stateful fixture ordering from real Git/native timeout coverage (ARCH-MOCK/ORDER). Every owned native child and fake callback has teardown (ARCH-FUNERAL).
- [x] Bound the real phase check and native tests; touch no operator UI or slot (ARCH-SECURE). No production query/fan-out changes; fake queries have counted linear reads/waits at 8 and 16 (ARCH-CONSTRAINTS).
- [x] Record the Lua/Python fixture mistake class in the Log; hub waived a Go-only catalog rule. No peer repository edits.
- [ ] Run focused Go/Lua/diagnostic tests, then the recorded full suite and normal close/merge gates. A fixture-only fix uses owned headless validation; any product/editor behavior change needs a live-check arrangement approved by the hub. Report the landing.

Existing visible states remain unchanged: a verified current identity permits the guarded write; unknown/deadline/changed-branch observations refuse it, retaining edits and activation. Recovery is retrying the action after a fresh successful observation. No new user or automation state is proposed. This plan fixes the refusal diagnostic as part of the same failing path, not a new authorization policy (ARCH-PURPOSE).

## Revisions

- 2026-10-18: Hub approved fixture isolation (receipt 955f5902-cbcc-4534-a458-f6ebee3bb6d8). Use a shared two-commit catalog and actual owned HEAD/ref/document files to back Python wanted observations and the Lua vim.system resolver double. Capture async identity/bytes at dispatch, preserve the completed-result gate, and refuse unknown refs/heads. Replace the fixture's 250 ms UI ceilings with counted synchronous-wait assertions. Keep real Git collector tests and native Neovim timeout conformance separate. Production resolver budgets and identity/index checks are unchanged. The hub explicitly waived a Go catalog rule because this mistake class is in Lua/Python; record it below instead.

- 2026-10-18: Reject the artificial descriptor-holder reproduction as historical evidence: it would manufacture a missing condition and require extra orphan ownership machinery. One real observation run with test-only Git Trace2 logging completed instead. Do not select a production drain-budget fix without evidence that the failure used that path. Next decision is whether to isolate the fixture's ordering checks behind a stateful resolver double while retaining real resolver and native-timeout conformance; bring that mechanism/scope choice to the hub first.

- 2026-10-18: Hub assigned #662 after parking #663. Initial investigation identifies a falsifiable deadline/drain composition issue and independent empty-stderr diagnostic loss. Prove the phase interaction before choosing a product versus fixture fix; no historical-phase attribution is claimed.

## Log

2026-10-18 pre-suite contract correction: inspected the installed Neovim 0.11 vim/_system.lua, where kill still produces its exit callback. The new double incorrectly dropped that callback; its new regression failed with "killed process lost its completion callback", then passed after retaining completion with the cancellation signal. Full fixture diagnostics pass again (13 tests). Cancelled only this checkout's queued suite on 831585aa via sdlc local-check cancel --name suite before it executed; the runner confirmed lock-wait context cancellation and exited, leaving other checkouts alone. This superseded-code cancellation is not a flake retry. All server checks on the earlier PR head were green; a new pushed head and recorded suite will cover this correction.

2026-10-18 CI overlap: draft PR https://github.com/xianxu/pair/pull/425 opened on tested implementation head 831585aa62982b956259bee39d3e46fcd9dac82d under the hub's standing early-draft authorization. Server conformance and merge checks started; local-gates guard is green. The recorded suite was requested on that pushed head via sdlc local-check run --name suite. This docs-only note does not replace its required record, close review or merge gates. No operator staging is needed for the fixture; separate native conformance uses the installed Neovim with owned children.

2026-10-18 implementation verification: positive ordering-control regression first failed because the old fixture had no stateful backend, then passed after wiring the shared catalog. All 13 diagnostics pass after final native-escape trapping, including both original refusal controls, new timeout refusal, actual checkout movement, stateful 8/16 query counts, callback/trace lifecycle and separate native deadline conformance. Native sync and async outcomes are code 124 with the shipped 2500 ms limit; startup timing is not required to beat that limit in the async control. Focused real Git collector tests passed: go test ./cmd/internal/reviewcmd -run 'TestResolveIdentity|TestIdentity|TestOSGitContext' -count=1 (6.297 s). Guard/identity Lua tests passed earlier. Oversized fake-input fixtures use sparse files instead of writing 16 MiB of unused payload. Atlas and lessons describe the separation; no product timeout, Git query or authority rule changed. Full recorded suite and boundary review remain pending.

2026-10-18 mistake class: an ordering/evidence fixture made success depend on repeated real subprocesses beating the product's deadline. A legitimate native timeout is a correct write refusal, not an ordering regression. The fix separates deterministic stateful fixture ordering from real Git/native deadline conformance; it does not increase a timeout, accept unknown authority, or treat a passing rerun as proof. Native timeout diagnostics remain useful. Hub waived the Go-only lint catalog requirement for this Lua/Python fixture class.

2026-10-18 native phase check: ran the unchanged observation fixture once with an owned Lua init enabling GIT_TRACE2_EVENT only for pane descendants. It passed in 19.332869 s; supervisor exit 0, direct child reaped, no diagnostic deadline fired. All 44 retained resolver completions succeeded; maximum elapsed 328.413916 ms, sum 10,767.188416 ms (overlapping async/sync intervals are not additive wall time). Git Trace2 recorded 423 complete processes, no nonzero exits: 188 rev-parse, 94 symbolic-ref, 47 log, 47 diff-index and 47 ls-files. Their summed internal Git durations were 1.160788 s, maximum 6.548 ms. These clocks begin inside Git and exclude native startup/parent waiting; the difference is not a causal partition. The historical slow phase remains unknown. Local evidence: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair662-phases.w2s73r/{init.lua,run.py,result.json,fixture.log,observations.json,git-trace.jsonl,summary.json}. Only one run, no stress, no suite retry, no runtime change. Source already explains a functional fixture risk: it treats every legitimate fresh-resolver refusal as an ordering-test failure, including 16 synchronous guards across eight snapshot writes. A product timeout remains a correct refusal, not permission to reuse stale authority.

2026-10-18 diagnostic regression: new nvim/review/identity_test.lua first failed with diagnostic="" for native code 124 / signal 9 / empty stderr. Decode now preserves nonempty native stderr and supplies an actionable timeout or exit-code reason when stderr is absent/empty. The regression passes without changing the 2500 ms sync/async deadline. Added guard checks for timeout, malformed identity and changed branch retaining exact buffer/disk bytes and activation. Focused identity, restore-controller, observation-trace and observation-gate tests all pass. This fixes diagnostic loss only; native budget-composition reproduction remains in progress and the original flake's inner phase is still unproved.

2026-10-18 planning: claimed and ran start-plan from main on 000662-review-observation-resolver-timeout. Read-only call-graph inspection: BufWritePre -> restoration:guard -> identity.resolve -> review readiness --resolve -> resolveIdentityWithSnapshot. This route does not invoke the headless-Neovim classifier. Successful resolution performs nine sequential Git calls, plus two index-generation checks; ordinary writes call guards before and after. The async callback gate holds completed results and is already released before the failing synchronous write, so it does not deliberately hold that native process. The historical trace records whole resolver calls only; it cannot distinguish native startup, Git execution, pipe draining and scheduler delay. OSRuntime.GitContext sets WaitDelay to one second despite the collector's shared two-second context and outer 2500 ms Lua wait. The empty diagnostic is explained by result.stderr or fallback accepting the empty string in Lua. These are source findings; no new reproduction, test run or implementation has occurred yet. Peer ariadne AGENTS.local.md was read to inspect its lint catalog; MEMORY.md is absent. The catalog handles Go ASTs, not arbitrary Lua/Python files.

Observed 2026-10-18 around01:02 PDT in pair#658’s same-body make -k WF_TEST_WEIGHT=2 test, source HEAD91d46ca2f5c0c3b1d5537fbb5b28c2a95b079bb1. Owned quiet two-token device diagnostic: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair658-device-after.SjFHBp/device-suite/make.log lines855–905. Make exit2 after312.427012s; all ten diagnostic tests enumerated, exactly this positive control failed. Remaining Make prerequisites completed; final go test ./... recipe was suppressed by failed prerequisites.

Error: nvim/review.lua:910 BufWritePre, review: write refused. Fresh trace’s final synchronous call23 ran2578.387417ms, timeout2500, code124, signal9, stdout_bytes0, decode_error invalid review resolver JSON syntax. Earlier asynchronous call22 succeeded in2092.532375ms. The original full-suite baseline passed this diagnostic. git diff from baseline b08a927f6fed942fdbb4aef84b09cf07b2a32f47 to failing source shows no changes to nvim/review.lua, tests/review-observation-test.sh, tests/review-observation-diagnostics-test.py or tests/lib/review_observation_gate.lua. These are intermittent controls, not proof of the exact root cause. No rerun or deadline change yet. Retain raw failure JSON embedded in log and source/body/queue/result receipts alongside it.

The initial native Zellij CI flake in #660 is separate; this is not its missing-session signature. Pair#659’s external-tool budget work is also not assumed to fix this review resolver.

Cross-reference pair#623 (native-process timeout budget family) and pair#659 (external-tool budgets; owned by pair:7). Hub approved exactly one issue-backed unchanged device rerun after the recorded suite; if that rerun fails before final Go again, stop and report, with no repeated attempts.
