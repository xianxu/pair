---
id: 000666
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'bc97ce9902fd9422129fda400f275a555c24ef5a' # card fields mirrored from issue-cards; edit via sdlc
---

# Guard native fixture root handoffs

## Problem

#636 fixed live recovery/park fixtures that accidentally lost disposable native socket and runtime roots: testpair.ApplyEnvironment clears inherited values, and nested helper environments need explicit propagation. The local source guard prevents recurrence only at its named entrypoints. Capture this Pair-specific mistake class in the repository lint path without making the shared catalog depend on Pair helper names.

## Spec

Decide whether this belongs in a small Pair-local catalog extension or ariadne's shared pkg/golint. Prefer Pair-local unless a general, independently useful shared rule is demonstrated: testpair's environment-clearing and helper-handoff contracts are Pair-specific. Keep the existing #636 guard and behavioral regression until equivalent coverage is qualified; do not simply delete the local protection.

Evidence is landed in PR431, merge 1e90001c06796ba600c77d3cde3255ad6818f611, verified reachable from origin/main. See cmd/internal/couchcore/recovery_live_environment_test.go: recoveryLiveEnvironmentViolations, TestRecoveryLiveEnvironmentSourceGuard (six mutations), and TestRecoveryLiveEnvironmentOwnsRoots (poisoned ambient roots, direct/nested children, cleanup/restoration, real AF_UNIX endpoint-budget bind). Actual entrypoints are recovery_conformance_live_test.go and park_lifecycle_live_test.go. Archived #636 details retain RED/GREEN and mutation evidence. No native run is needed merely to reproduce the guard's deterministic behavior.

## Done when

- Rule ownership and the consumption interface are documented, with Pair-specific identifiers kept local unless a justified shared abstraction is approved.
- Ordinary repository lint catches omitted disposable-root setup and nested helper propagation at the protected entrypoints.
- Stateful or source fixtures prove clean and missing setup/handoff cases, including all six existing mutations; poisoned direct/nested child and socket-budget coverage remains intact.
- No blanket suppressions, native deadline changes or operator-environment access are introduced; required review and recorded suite pass.

## Plan

Unassigned follow-up after landed #636. Design the local extension boundary before code; any ariadne change is routed through the hub as a cross-repository proposal.

## Log

Hub ded49f45 requested this one follow-up after #636 lands and leaned Pair-local. Pair:6 delivery2fd83608 supplied landed paths and evidence; independently verified the merge and inspected the source. This is outside #665's implementation scope and does not block it.
