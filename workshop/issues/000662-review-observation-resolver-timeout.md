---
id: 000662
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a2c912999900020be4226aee8c56320392c7aa87' # card fields mirrored from issue-cards; edit via sdlc
---

# Review observation diagnostic positive control times out in real BufWritePre resolver

## Problem

The normal make test shell/review phase can fail in ObservationFixtureDiagnosticsTest.test_positive_control_retains_wanted_fresh_and_request_order. The fixture calls a real readiness resolver during BufWritePre; a late synchronous resolver was killed at its existing 2500 ms deadline, producing an empty result and a write refusal. It blocked pair#658’s whole-device after measurement before the final Go recipe.

## Spec

Determine why this positive control sometimes exhausts the resolver budget, distinguishing host capacity, the fixture’s own synchronization, and product RPC reentrancy. Preserve genuine write-refusal and branch-identity safety; no blanket timeout bump or blind retry.

## Done when

- Root cause is demonstrated with retained resolver phase evidence and a deterministic regression.
- The positive control passes under that controlled condition while changed-branch, unknown identity and deadline refusal controls remain effective.

## Plan

- [ ]

## Log

Observed 2026-10-18 around01:02 PDT in pair#658’s same-body make -k WF_TEST_WEIGHT=2 test, source HEAD91d46ca2f5c0c3b1d5537fbb5b28c2a95b079bb1. Owned quiet two-token device diagnostic: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair658-device-after.SjFHBp/device-suite/make.log lines855–905. Make exit2 after312.427012s; all ten diagnostic tests enumerated, exactly this positive control failed. Remaining Make prerequisites completed; final go test ./... recipe was suppressed by failed prerequisites.

Error: nvim/review.lua:910 BufWritePre, review: write refused. Fresh trace’s final synchronous call23 ran2578.387417ms, timeout2500, code124, signal9, stdout_bytes0, decode_error invalid review resolver JSON syntax. Earlier asynchronous call22 succeeded in2092.532375ms. The original full-suite baseline passed this diagnostic. git diff from baseline b08a927f6fed942fdbb4aef84b09cf07b2a32f47 to failing source shows no changes to nvim/review.lua, tests/review-observation-test.sh, tests/review-observation-diagnostics-test.py or tests/lib/review_observation_gate.lua. These are intermittent controls, not proof of the exact root cause. No rerun or deadline change yet. Retain raw failure JSON embedded in log and source/body/queue/result receipts alongside it.

The initial native Zellij CI flake in #660 is separate; this is not its missing-session signature. Pair#659’s external-tool budget work is also not assumed to fix this review resolver.

Cross-reference pair#623 (native-process timeout budget family) and pair#659 (external-tool budgets; owned by pair:7). Hub approved exactly one issue-backed unchanged device rerun after the recorded suite; if that rerun fails before final Go again, stop and report, with no repeated attempts.
