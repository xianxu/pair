---
id: 000666
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'dd660943cd13ce670525e811484e768725fec953' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T10:34:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
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

- [ ] Add a Pair-local cross-file native-fixture-roots rule in internal/repolint, consuming the three protected source files and emitting located findings through the ordinary repository lint test.
- [ ] Qualify clean source fixtures, the existing six mutations, all seven omitted root reads, and missing/unreadable/malformed source failures; preserve the real poisoned parent/nested child and socket-budget tests.
- [ ] After equivalent lint coverage passes, remove the old couchcore source matcher/mutation test so there is one owner; keep environment builders and runtime regression unchanged.
- [ ] Update atlas ownership/interface documentation, run targeted lint and runtime tests plus races, then recorded suite and sole close review/normal landing.

## Log

Hub ded49f45 requested this one follow-up after #636 lands and leaned Pair-local. Pair:6 delivery2fd83608 supplied landed paths and evidence; independently verified the merge and inspected the source. This is outside #665's implementation scope and does not block it.

## Design

Recommend a Pair-local repository rule: the defect depends on Pair's empty-environment builder and a cross-file helper handoff. Shared golint.Rule matches one AST, so putting these identifiers there would impose Pair contracts on other repos. Keeping only the existing guard would miss the requested lint ownership. A generic configurable cross-file lint framework is unnecessary for one known contract.

nativeFixtureRootFindings consumes source bytes keyed by the three protected repository paths and returns located findings for rule native-fixture-roots (incident #636). It is pure parsing/inspection; absent or malformed required sources fail visibly. A thin reader in internal/repolint reuses the repository test root and injected readFile seam, reading each of those files once without extra Git/native queries. TestTrackedGoFilesPassCatalog consumes this local check alongside the unchanged shared catalog/debt scan. No suppression path is added.

The manifest protects warm setup, park setup, the child cmd.Env handoff and all seven explicit root reads in recoveryLiveChildEnvironment. This is bounded source-contract checking, not a general data-flow proof or automatic discovery of future native fixtures. Keep the existing matcher until the new rule passes its six original mutations and a real-source equivalent; then remove the duplicate matcher/mutation test in the same issue (ARCH-DRY). The actual poisoned direct/nested-child, restoration/cleanup and socket-budget regressions stay in couchcore (ARCH-PURPOSE).

Pure rule tests use source fixtures with no IO (ARCH-PURE); the reader uses injected reads for missing/unreadable inputs. The existing repo catalog boundary owns Git and its stateful fake; no new binary/service dependency is introduced (ARCH-MOCK). Cost is one pass over three bounded source files per repository lint run; no runtime hot path or additional process query (ARCH-CONSTRAINTS). Source parse/read failures refuse rather than treating unknown as clean (ARCH-SECURE). The check holds no state between events and creates no durable resources because it is one synchronous in-memory lint invocation (ARCH-ORDER, ARCH-FUNERAL).

Visible results are clean (continue), finding (restore the named setup/handoff/root read and rerun), or unreadable/malformed source (restore the source and rerun); no user data is changed or lost. No native window, deadline change, operator-environment access or shared ariadne edit is needed.

## Revisions

- 2026-10-18 — Dispatched by verified hub receipt39404527; unowned card checked, claimed and start-plan completed. Chose Pair-local cross-file lint ownership; equivalent deterministic qualification precedes retiring the old duplicate guard. Quick flow, no milestones or estimate.
