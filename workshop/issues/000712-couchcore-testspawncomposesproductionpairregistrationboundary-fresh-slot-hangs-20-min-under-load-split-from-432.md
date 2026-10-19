---
id: 000712
status: working
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'c0c63385006a8e2a5fc68b8463f9f934dc4d5797' # card fields mirrored from issue-cards; edit via sdlc
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcore.TestSpawnComposesProductionPairRegistrationBoundary]
started: 2026-10-19T14:08:10-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "ac6c2475", done: "1f82abd6"}
---

# couchcore: TestSpawnComposesProductionPairRegistrationBoundary/fresh-slot hangs 20 min under load (split from #432)

## Problem

`couchcore.TestSpawnComposesProductionPairRegistrationBoundary/fresh-slot` hung in pair#689's integrated suite (2026-10-19,
f2333e34): 1172.59 s, LaunchNative reported "terminal not listed live", the cleanup spawn goroutine failed to exit, then the
package hit its 20-minute timeout. The whole suite took 26 min (median 12.5). pair#432 records the same test "failing
identically on origin/main" (10-10) and asks to triage it separately; most pair suites since pass it, so it's intermittent,
probably load-sensitive. It now blocks a landing as an unattributable failure.

## Spec

- Split from #432 (its Spec asks for this). Reproduce under load (the suite's own concurrency, or `stress`), with the
  goroutine dump at the hang.
- Root-cause the hang: what waits forever when the terminal isn't listed live, and why the cleanup goroutine can't exit.
  A wait without a deadline in production code is a bug in its own right (the product side, not just the test).
- Register it as a known flake under this issue meanwhile (same-head fail→pass evidence from pair#689's run), so landings
  rerun it instead of blocking; the registration ends when this closes.

## Done when

- The hang is explained and fixed; the test passes 20/20 under a loaded run; no unbounded wait remains on that path.
- Persistent session-quiescence refusal returns within one shared five-second session budget, retains the exact unresolved thread/start ownership, and records/surfaces a named unproven cleanup failure through existing lifecycle diagnostics and hub-visible receipt/show output. A timed-out observation never permits rollback, a foreign-session signal, or a fresh replacement launch.

## Plan

- [ ] Reproduce the retained failure deterministically with an isolated Couch and the existing fake Zellij. Capture the exact repeated session-quiescence error through the existing artifact/owner-probe seam, and prove helper death separately from session absence.
- [ ] Repair the fixture's session lifecycle at its existing persisted-state seam: queries must read the fixture server's session identity and lifetime, independent of the querying caller's environment; fixture release/deletion must make exact absence observable. Preserve #698's listing-before-agent-readiness order. Add deterministic failed-launch/cleanup regressions for ordinary and fresh-slot starts, including restored parent environment, stale readiness and a delayed release. If measurement instead identifies a production ownership defect, stop and revise with the hub before changing that mechanism.
- [ ] Run targeted race/source checks and 20/20 composed runs under bounded owned load; retain load cleanup on every exit. Run the recorded suite, close review and normal merge gates, then report landing. No live Couch, native Zellij lane or pair:0 staging.

The likely change is test-only and fits the quick-flow shell. ARCH-MOCK: extend the existing registration terminal fake rather than bypassing production registration or owner checks. ARCH-ORDER: model server creation -> listing publication -> agent readiness -> release/deletion -> observed absence; a ready marker alone is not a live server. ARCH-FUNERAL: every owned fixture child, socket and goroutine is reaped/joined before its isolated roots are removed. No new user-visible state or persisted production format is planned. Production post-ack retry deliberately retains ownership while absence is unproved; do not replace it with a timeout that abandons writers. The regression must establish eventual exact absence and a joined launch, not merely stop waiting for it.

## Log

### 2026-10-19

Recovery/triage: verified canonical submitted dispatch 02f3d46f-c737-4231-a2a4-bea5e71c97fa; claimed and ran start-plan from fresh main after #708. The retained attempt's exact code is f2333e342cc325ded2344fe69dd0d7309224dca7. Its couch.go:966 is the session cleanup retry after quiesceThreadSession, not the helper retry. FakeHandle's unconditional KILL already ends its child/group and Wait channel at that revision; the frozen fixture clock is unrelated. The launch error is consistent with the agent-ready/listing publication race repaired by #698: that revision published agent readiness first, current main publishes the listing first. A second latent fixture defect remains plausible: list-sessions uses a persistent ready marker plus the querying process's PAIR_SESSION_NAME, while the composed hook restores the parent's environment before cleanup, and release can leave a stale ready marker. The retained log omits the repeated session error, so capture that exact error before selecting the repair. No production code changed during this investigation.

Measurement after verified hub approval ba6e71aa-42d2-416a-a7ab-fe8ee2cf9829: unchanged composed baseline passed 2.71s (ordinary 1.32s, fresh-slot 1.39s). The existing production retry contract test passed 0.63s, confirming it returns only once the fake changes refusal to success. An owned throwaway probe then forced 128 consecutive QuiesceNamed refusals with the retry sleep injected: production made 129 attempts, every context lacked a deadline, and returned only on the probe's eventual success (0.82s). Thus a permanently refusing real observer can spin forever; no total production session deadline exists. A separate fake subprocess with a stale ready marker and restored caller environment emitted exactly ` [Created 1s ago]\n`, exit 0, losing its session identity (0.07s). Diagnostic probe passed in 1.145s total and is removed after recording these facts; it is not a regression test. No production code has changed. R1 durable plan now includes bounded production session cleanup and retained ownership; wait for hub approval and re-enter full-flow change-code before production edits.

## Log

- 2026-10-19 (hub, from pair:6): registered as a known flake. Same head f2333e342cc325ded2344fe69dd0d7309224dca7: pair#689's
  integrated suite d963805085758c53cfbbab3363ad6432 failed after 26m12s (log 2f2ab4699a85; `fresh-slot` hung 1172.59 s,
  "terminal not listed live", cleanup goroutine didn't exit, package hit its 20 min timeout); the unchanged targeted test
  then passed in 6.666 s (ordinary 2.94 s, fresh-slot 3.44 s), same 20 min timeout, fake Zellij. The suite ran at twice its
  median time (machine loaded).

## Revisions

2026-10-19 R1: the hub approved the initial fixture investigation and required a production bound if the real session loop was unbounded. Measurement confirms it is. The executable scope is now workshop/plans/000712-bounded-session-cleanup-plan.md: five-second session budget after helper quiescence, typed unproven failure, exact durable ownership retention, existing lifecycle/hub-visible diagnostic surfaces, plus the approved fixture lifecycle correction. The earlier test-only Plan is retained as history and superseded by this durable plan. Exact helper drain remains its existing ownership-retaining mechanism; the observed hang occurs after that phase has completed.

2026-10-19 R2: plan-quality PQ-1 found that a new persisted phase would be rejected by normalizeLaunchFailure. Reuse accepted Phase launch and Outcome unproven; keep the named session cleanup uncertainty in existing error/detail fields, with validator/codec and public persistence/readback tests. No format/status extension or behavioral scope change. Hub approved R1 in canonical submitted receipt 8937d10b-c9c6-4bbd-a243-b5163bdc0a36 and required existing continue/resume-from-disk/start-fresh recovery outcomes.
