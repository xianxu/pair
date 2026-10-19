---
id: 000712
status: codecomplete
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours: 1.9
collaboration-mode: collaboration
card_mirror: '3175020942209307c355692212310949ec9a77b1' # card fields mirrored from issue-cards; edit via sdlc
flake_tests: [github.com/xianxu/pair/cmd/internal/couchcore.TestSpawnComposesProductionPairRegistrationBoundary]
started: 2026-10-19T14:08:10-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 1.87
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

- [x] Reproduce the retained failure deterministically with an isolated Couch and the existing fake Zellij. Capture the exact repeated session-quiescence error through the existing artifact/owner-probe seam, and prove helper death separately from session absence.
- [x] Repair the fixture's session lifecycle at its existing persisted-state seam: queries must read the fixture server's session identity and lifetime, independent of the querying caller's environment; fixture release/deletion must make exact absence observable. Preserve #698's listing-before-agent-readiness order. Add deterministic failed-launch/cleanup regressions for ordinary and fresh-slot starts, including restored parent environment, stale readiness and a delayed release. If measurement instead identifies a production ownership defect, stop and revise with the hub before changing that mechanism.
- [ ] Run targeted race/source checks and 20/20 composed runs under bounded owned load; retain load cleanup on every exit. Run the recorded suite, close review and normal merge gates, then report landing. No live Couch, native Zellij lane or pair:0 staging.

The likely change is test-only and fits the quick-flow shell. ARCH-MOCK: extend the existing registration terminal fake rather than bypassing production registration or owner checks. ARCH-ORDER: model server creation -> listing publication -> agent readiness -> release/deletion -> observed absence; a ready marker alone is not a live server. ARCH-FUNERAL: every owned fixture child, socket and goroutine is reaped/joined before its isolated roots are removed. No new user-visible state or persisted production format is planned. Production post-ack retry deliberately retains ownership while absence is unproved; do not replace it with a timeout that abandons writers. The regression must establish eventual exact absence and a joined launch, not merely stop waiting for it.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; calibration is marked stale by estimate-source, so values are provisional. Uses the v2/v2.1 primitive ranges, thorough-plan design discount x0.2, implementation scale x0.40, familiar existing Couch/fixture territory x1.0, and +15% design buffer. No novel stack/library search is needed: standard context and existing recorder, thread transactions and fake-process seams already supply the mechanisms.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.15 impl=0.08
item: api-integration design=0.20 impl=0.40
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.06 impl=0.20
item: cross-cutting-refactor design=0.10 impl=0.16
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.02 impl=0.16
design-buffer: 0.15
total: 1.90
```

Rows cover measurement/spec, one-budget native session cleanup and interaction tests, unknown-state/error projection, fixture lifecycle and failed-start regressions/load harness, context propagation and consumer/source checks, docs/lessons, and one issue-close review. Raw design picks are respectively 0.75/1.0/0.2/0.3/0.5/0.1/0.1h; raw implementation picks 0.2/1.0/0.4/0.5/0.4/0.15/0.4h, each within its table range. Adjusted design 0.59h x1.15 = 0.6785h plus implementation 1.22h = 1.8985h, rounded 1.90h. Suite capacity waits are not active work; controlled-load setup and test diagnosis are included.

## Log

### 2026-10-19
- 2026-10-19: closed — BR-1: shared five-second session context covers current/legacy metadata, owner proof and delete; expiry rereads nothing; separate five-second disposition context retains exact occupied claim on refusal and exposes both uncertainties. Held-lock, live/replaced identity, history/show/receipt and 20-second total-envelope tests pass, including focused race checks. BR-2: nested Git/slot TMPDIR six-case family passes normal13.434s/race61.337s; pinned 587e5cbdfe7206c7026c7342c4f9a77f9ef4d0f5 original family passes20/20 under20 owned CPU workers (135.963s), all reaped. Source/lint and existing regressions pass. Current tl-worker permits review overlap with queued suite cab04757dce899d76310d33aafe65b21; no-plan-check acknowledges only unfinished suite/close/merge checklist. Recorded pass on final code remains required before landing.; review verdict: SHIP

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

2026-10-19 implementation checkpoint: deterministic virtual-time RED proved the old loop needed 7 virtual seconds and 66 attempts before the test released refusal; the repaired loop returns after its one five-second session budget, and every attempt receives the identical nonzero deadline. Public Spawn persistent-refusal regression passes with exact retained binding/StartNonce/PID/identity, unknown occupied incarnation, persisted launch/unproven history, named CleanupError and diagnostic nonce. Receipt regression passes with existing failed/unconfirmed classification and the same diagnostic identity. StartRecoveredUnknown uses the existing promoted SessionBinding representation described in plan R3. The operator-line test caught and repaired cleanup uncertainty being overwritten by observed-client-exit wording. Focused fixture family passed normally (15.141s) and with race instrumentation (66.670s); no race report, and owned server/wrapper/agent generation death is asserted. Loaded repetitions, source guards and full suite remain pending.

Verification checkpoint: removing contextual deletion dispatch makes TestSessionCleanupCarriesDeadlineThroughNativeDelete fail (legacy delete called once, missing deadline, nil error); restored implementation passes. Removing the unconfirmed-knowledge routing makes the public persistent-refusal regression fail with a creating rather than unknown retained incarnation; restored implementation passes. Public history is reopened through NewLaunchFailureStore and the actual show dispatcher must project the same nonce and CleanupError. Production artifact/source guards passed for artifactpath and termcmd; slot resource coverage passed. Full lint, loaded repetitions and focused race checks remain in progress. An early draft PR is authorized by the hub standing CI-overlap exception; suite, close review and merge gates still precede landing.

Focused completion: virtual-time cleanup/persistence/show and receipt tests pass with race detection (couchcore 2.974s, couchcmd 2.124s). Existing post-ack/start-cleanup/ownership/switch-agent regressions pass (11.672s). Source guards, slot resource coverage and repository lint catalog pass. Draft PR 465 is open for CI overlap; required suite attempt 625706108325e6c00e36ab4cdcbbdebb is queued on 2cfbb831b, normal FIFO. Loaded repeated composed cases remain running with twenty owned CPU workers and a 330-second outer watchdog; all workers are killed and waited in finally on every exit.

Loaded verification PASS: original ordinary/fresh composed family plus deterministic failure variants each repeated 20 times with 20 owned CPU workers; package passed in 293.821s, outer elapsed 314.29s, exit 0, within the existing five-minute package bound and 330-second outer watchdog. Finally killed and Wait-reaped all 20 workers. No native/live terminal lane used. This proves the formerly hanging family completes under load with exact fixture server lifetime; no unchanged flake retry was taken. All server CI checks passed on the tested code. Full recorded suite remains in normal FIFO (attempt 625706108325e6c00e36ab4cdcbbdebb); no duplicate run or process intervention.

Review arrangement: refreshed tl-worker help explicitly permits the close review to overlap a queued recorded suite, provided the final-head suite passes before landing and any review code fix requeues it. Begin the binary-owned close review now. The precise no-plan-check acknowledgment applies only to the still-unfinished recorded-suite/close/merge checklist row; no verification bypass is requested, and merge must await the authoritative recorded pass.

BR-1 verified: held store-lock read ignored a 20ms context and returned only after the regression released the lock at 2s. The expiry reread independently hit the test's 20s watchdog with its dump in GetThread -> blocking Flock at couch.go:1001. Repaired session metadata reads use PreviewThreadContext with the same session context, and the retry loop retains the observed session name instead of rereading on expiry. Both contention cases plus the shared-budget regression pass (0.482s). The expiry test now has an external real watchdog that can release a broken kernel-flock wait even inside virtual time; it joins that owner on every exit. Unknown-disposition persistence contention is still awaiting the hub decision sent in submitted receipt 45232ab6-8a1b-4448-bfe2-1432e5c1534f; do not implement that dependent fallback without the answer.

Turn checkpoint after BR-1 read repair: current contention regressions pass normally (0.482s) and with race detection (2.152s). Implementation commit 4f11579fc44a3accda482c3a3a5bcf37ebeb8971 is pushed to draft PR 465. The original suite attempt was cancelled only through sdlc local-check cancel; no suite is active for this checkout. BR-1 remains open until the disposition-write path is bounded too. The hub has visibly replied in its live screen, but canonical Couch approval has not been delivered into this busy turn; dependent persistence fallback is not implemented. End the turn to admit that delivery, then verify its canonical receipt/body before continuing. The registration_fixture_712 agent is independently reproducing/fixing BR-2 nested-review-TMPDIR Git discovery in its prior three fixture files; collect its report before editing those files or requeuing. New code needs a fresh suite, close review disposal, and normal merge gates. No live terminal or process intervention.

R4 implementation and proof: canonical submitted approval 3f878b0c-bada-408b-bfcf-c7cdbe82fdbf authorizes separate bounded disposition persistence. Added one five-second recovery-scale context covering PreviewThreadContext, routed update/journal locking and CAS conflicts. Refusal joins ErrCleanupPersistenceUnconfirmed to the session error; the original occupied claim stays untouched. RED tests blocked until the outside lock watchdog released storage both before read and between read/write (2.19s/2.16s). The repaired public tests pass, preserve the exact original record revision and Start.Nonce, and expose both uncertainties in retained diagnostic evidence and operator text. A diagnostics-stalled variant consumes its existing ten-second cap and checks the full 5+5+10=20s post-helper budget. Combined cleanup, decision, persistence, show and receipt regressions pass with race instrumentation (couchcore 2.665s, couchcmd 1.636s). Atlas now states the total budget and retained-claim fallback. BR-2 fixture agent is completing explicit disposable Git ceiling plus explicit workspace identity, with a permanent nested-slot checkout regression; no production parser change.

R4 safety tail: added exact live-incarnation cases for live -> unknown, already unknown (no rewrite), and replaced birth identity (refuse persistence, preserve exact replacement). Targeted cases pass (0.461s). Real-time watchdogs are only a broken-kernel-wait unwind; their cap now derives from the documented 5+5+10-second product envelope. Main behavioral assertions use virtual time and require completion while the store holder remains locked, avoiding a host-speed band. Root source guards/lint pass (artifactpath 3.240s, termcmd 0.703s, repolint 4.411s), and existing post-ack, ownership, switch-agent plus slot-state coverage pass (10.233s).

BR-1 enumeration completion: records lacking embedded session bindings still use a scoped external index. Their compatibility Quiesce path previously reset context; new TestSessionCleanupLegacyIndexCarriesBudget reproduced an owner probe with no deadline and a seven-virtual-second watchdog release despite the caller's two-second budget. The production indexed path now reads PairSessionName with the supplied context and uses QuiesceNamed for the ownership/revalidation/delete chain; exact observed name is returned for cached expiry reporting. Generic no-IO/fake controllers retain their compatibility behavior. Combined cleanup/persistence/legacy regressions pass (1.410s). This repairs the full known production class rather than only the first GetThread site.

BR-2 verified and fixed in couch_test.go only: with TMPDIR inside this real Git/worktree checkout, the initial command failed ordinary before readiness with repository-scope change and fresh with unknown workspace. Explicit Git ceiling fixes ancestor scope discovery; exact current WorkspaceIdentity in SlotCatalogFake.Workspaces fixes incomplete fake authority instead of changing the production slot parser. Permanent regression owns an outer Git checkout under worktree/pair-slot3/pair and an inner TMPDIR; its fresh case reproduced unknown workspace before the fake completion. All six ordinary/fresh success, transient-failure and outer-Git isolation cases pass under actual checkout-nested TMPDIR normally (13.434s) and with race instrumentation (61.337s), no race report. The agent's temporary root is removed; final pinned-head loaded repetitions and fresh suite follow.

Final pinned-code loaded proof for BR-2: before launch, git rev-parse printed 587e5cbdfe7206c7026c7342c4f9a77f9ef4d0f5. On exactly that code, go test ./cmd/internal/couchcore -run '^TestSpawnComposesProductionPairRegistrationBoundary$' -count=20 -timeout=5m passed (135.963s package; 139.66s outer). Twenty owned CPU workers ran for that bounded invocation; finally killed and Wait-reaped all twenty. No source changed during the run (only atlas wording committed afterward). Together with the nested-TMPDIR six-case normal/race pass, this disposes BR-2. BR-1 is covered by contextual thread reads, cached expiry identity, legacy index/probe/delete context propagation, independent bounded disposition read/CAS/journal writes, both held-lock seams, exact unchanged-claim fallback, live/unknown/replaced identity tests, and the diagnostic-stall total envelope. Final cleanup/receipt race checks pass (couchcore 2.982s, couchcmd 1.651s); source/lint guards and existing regression family pass (4.195s/0.665s/4.990s and 10.347s). Fresh required suite cab04757dce899d76310d33aafe65b21 is queued on the same code at 587e5cbd; previous attempt 625706108325e6c00e36ab4cdcbbdebb was intentionally cancelled before running because review changed code. Close review may overlap per current tl-worker rule; final-code recorded pass still precedes landing.

Close outcome: round 2 SHIP, high confidence; BR-1 and BR-2 explicitly addressed, no blocking findings. Evidence commit 46b40db395481e0cc55ac2dc4cc2c75ab7247908; measured actual 1.87h adopted by sdlc (estimate 1.90h). Reviewer independently repeated the pinned composed family 20/20 under twenty owned CPU workers (106.456s), reaped them all, and performed metadata/Git-ceiling omission checks. Minor BR-3 (single-incarnation-transition-authority) is preserved as unclaimed follow-up #713, dependent on #712, rather than churning the reviewed source and queued final suite. PR 465 is ready for review. Required suite cab04757dce899d76310d33aafe65b21 is still queued (normal FIFO; six with a newer fleet-priority entry ahead); no pending product decision or source change. Wait for its recorded pass, then enter sdlc merge; do not duplicate or cancel the valid queued run.

Environment replacement, not a flake retry: verified submitted hub instruction a19af2df-e986-4778-82d4-b328f1c41c44 says the queued runner predates ariadne#474 and would use HFS+ TestScratch, invalidating timestamp/write-budget checks. Cancelled only this checkout's queued suite through sdlc local-check cancel; cab04757dce899d76310d33aafe65b21 never ran. Replaced immediately with the updated installed sdlc on unchanged reviewed source at c636e5fa11867718c9fe436313cd0272c30cf453. New required attempt a3864cece67094352778f2cff95d5403 uses SSD worktree /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/sdlc-local-3608107206, showing it skipped the HFS+ disk; normal queue position six. The operator, not this slot, owns recreating TestScratch as APFS. No live process or filesystem migration was attempted.

2026-10-19 crash recovery and landing blocker: previous Couch screen is unavailable; recovered from this clean issue branch and fresh ops:0 dispatch. Published the previously local log commit b1cc1870b39b9111f41a86c27ddd16afdb730735 through sdlc pr (PR 465). APFS recorded suite d7b39428f6f4d3de2182c233873dbb15 waited 1463s, then ran to completion in 8m36s. Couchcore passed in 152.062s; all other packages passed except couchcmd.TestContinuationWriterPublishesExactCheckpointAcrossWorktrees/warm-reattached. Its nested launcher.TestCouchLaunchPromptTransportHelper failed TempDir RemoveAll cleanup: .retention directory not empty. The unchanged exact suite checkout rerun of the complete parent test passed all five cases (11.096s package, 10.81s parent). Registered this same-head fail/pass as unclaimed flake #715. However SDLC classified the completed suite as unattributable-failure, saying couchcmd failed without a failing test, despite explicit parent/subtest FAIL lines 810-811 and nested helper FAIL line 815 in the saved log. No automatic known-flake rerun occurred; no blind suite retry or merge bypass was attempted. Saved log: shared Git directory sdlc/test-stats/failed/d7b39428f6f4d3de2182c233873dbb15.log (sha256 prefix 2c6fa8c9c88a). Landing remains blocked on SDLC failure attribution; report the peer-tooling gap to the hub for its disposition. Reviewed product source is unchanged.
