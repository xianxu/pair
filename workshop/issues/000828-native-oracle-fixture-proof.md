---
id: 000828
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
severity: medium
collaboration-mode: collaboration
card_mirror: '7771c8fd521d394d491e9167fcc9eeed86d651c7' # card fields mirrored from issue-cards; edit via sdlc
---

# Native oracle requires fixture proof

## Problem

During #759 shared-module qualification, the opt-in native Zellij history oracles dumped a shell prompt instead of the fixture child content. The helper accepts a successful dump even without positive fixture readiness. An equality-only oracle can then pass without exercising its intended input. This is a coverage integrity gap, not an established dependency regression.

## Spec

Qualify the owned fixture child before treating its dump as an oracle: require positive readiness and source identity or an expected fixture nonce. Keep a fully explicit private environment and owned HOME/config/socket/data roots, never operator configuration. Preserve source/PTY stderr and launch evidence when a successful dump contains the wrong source. Keep bounded startup and exact joined cleanup. Diagnose why the fixture fell back to a shell; missing --config-dir is a hypothesis, not a proved cause or prescribed fix.

## Done when

- A dump from a shell or unrelated pane cannot pass any native oracle, including the erased-background equality case.
- Fixture-ready absence and wrong-source output fail with preserved launch/PTY evidence.
- Private HOME/config/data/socket isolation and joined cleanup are pinned in regressions.
- The seven affected native cases exercise and confirm their intended source on the supported installed Zellij.

## Plan

- [ ]

## Log

2026-10-22, found during #759. Source bfa3f4a22c1e2583e1b1b2eefc118e19ec54d70d, Go1.27.3, installed Zellij0.45.1. Candidate contains the ten shared dependency upgrades only, no tsnet. One declared run with PAIR_TERMINAL_NATIVE=1: six failures, one unqualified equality-only pass. Failures: TestPresenterHistoryEvictionNativeAppendAndRebuild, TestPresenterHistoryEvictionKeepsSoftLineOracle, TestHistoryWireNativeOracle, TestHistoryOneHostRowOracle, TestHistoryStyledLinkedWrapWireOracle, TestHistoryDefaultWideViewportWrapOracle. TestErasedBackgroundSoftGapNativeCopyOracle passed equality despite the shell prompt. TestHistoryWireNativeOracle failed at history_oracle_test.go:132 with literal native history "sh-3.2$\n".

One matched original-module control of TestHistoryWireNativeOracle on the SAME source with equivalent fresh private HOME failed identically at line132. No candidate retries, assertion changes or shortened deadlines. The one-row case also reported no active session. The helper tests/terminal-oracle/discovery/zellij_oracle.py waits for readiness but does not require it before accepting dump-screen; it discards source/PTY evidence on that success path. Exact cause remains unproved.

Durable evidence is being committed on #759 under workshop/evidence/000759/tsnet-dependency-20261022/native/: additional-native.log/results/command/cleanup, original-module-control/test.log/comparison/results/cleanup, and SHA-256 manifest. Main requested native cohort and complete ordinary terminal package passed separately. No operator sessions, live fleet, visible windows or OS clipboard touched. Follow-up is unclaimed; do not broaden #759 into fixture repair.
