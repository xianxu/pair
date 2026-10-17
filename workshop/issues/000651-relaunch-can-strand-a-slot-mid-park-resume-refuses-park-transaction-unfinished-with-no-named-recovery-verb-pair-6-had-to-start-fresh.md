---
id: 000651
status: working
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-17
estimate_hours: 2.5965
collaboration-mode: collaboration
card_mirror: '449c114d1e05bbf8433f23a06d9ebdc1dd8d58a1' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T13:48:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Relaunch can strand a slot mid-park: resume refuses 'park transaction unfinished' with no named recovery verb (pair:6 had to start fresh)

## Problem
A failed relaunch can retain an open park after its Pair/terminal has gone. Resume then refuses with “recover that park or Start fresh”, but park recovery is not a public CLI verb and no retry/recover menu action is offered. Even direct controller Recover cannot resolve an absent terminal with missing or failed completion.


## Spec
Preserve the conversation through interrupted relaunch without bypassing exact process identity, terminal ownership, store revision or completion authority. Before request publication is attempted, cancellation/failure can roll back the newly begun park. After publication may have occurred, retain the transaction and reconcile its exact attempts: never invent a successful park or undo unknown external effects.

Expose a confirmed, idempotent recover-park operation through the hub CLI and the thread action menu. It completes a matching successful park, drives an existing live attempt when safe, or aborts an interrupted dead-owner transaction and retires only its exact positively dead incarnation. This last outcome preserves native binding/history and explicitly reports incomplete cleanup/scrollback; it does not manufacture VerifiedPark or mean filesystem effects were reversed. Later resume uses its normal native-conversation checks.

Every unfinished-park refusal and partial relaunch receipt names the recovery verb. All paths classify the durable state after a partial write; a failed pre-begin commit must not claim Pair stopped, and a finalized park must not claim an open transaction.


2026-10-17 ~11:50: `couch --relaunch pair:6 --confirm`, a Codex slot on tag 1-pair-55 whose binding had just been confirmed, didn't finish. Its output ended with a recovery menu ("abandon — give up on it and leave the thread occupied"), and the slot had no live wrapper afterwards. `couch --resume pair:6` then refused: "resume-parking: a park transaction is unfinished; recover that park or Start fresh". `couch --help` names no verb for recovering a park, and the resume candidates listed only "Resume from disk (confirmed)", which refuses, and "Start fresh". `scrollback-1-pair-55-codex.raw` was missing, apparently moved by the half-done park. The hub had to discard the conversation with a fresh reboot. That was cheap here because pair:6 was idle between items, but it would lose real work mid-item.

Earlier the same day, pair:4's reboot failed similarly at zellij session ownership ("ownership unresolved"), which may be pair#640's reply misrouting.

Wanted:
- A relaunch either completes or rolls back its park, so the slot stays resumable from disk.
- An interrupted park is recoverable by a named verb, shown in the refusal: finish it, or restore the scrollback and binding, then resume.
- The relaunch's failure output names the step that failed (park, kill, spawn) and the recovery command.

## Done when
- Isolated cancellation/fault and journal-restart tests cover every durable park boundary plus all six cleanup stages; each outcome retains independently checked ownership and conversation invariants.
- Pre-publication rollback leaves the original live incarnation intact. Matching success finalizes once; dead-owner interruption can be retired without fresh conversation or forged cleanup success. Unknown/live competing owners remain protected.
- A real executable, confirmed recover-park command and matching menu action are available from each applicable refusal; repeated recovery is harmless and never resumes/submits automatically.
- Required suite, mandatory close review, documentation and TL-coordinated live relaunch/interruption/recovery acceptance pass before landing (unless TL explicitly revises live scheduling).


- A test interrupts a relaunch at each park step; afterwards `couch --resume` succeeds, or the refusal names a recovery verb that, once run, makes it succeed.
- The relaunch failure text names the failed step and the recovery command.

## Plan
See [durable implementation plan](../plans/000651-interrupted-park-recovery-plan.md). Await TL approval before change-code or production edits.

- [x] Promote the isolated interruption matrix to regressions and implement bounded rollback/recovery over the existing park authority.
- [x] Expose recover-park through confirmed CLI/menu/receipts and make all interrupted-park advice actionable.
- [ ] Complete the required suite and prepare close review with targeted/race and native fixture proof; record operator acceptance explicitly deferred by the hub.


## Log

### 2026-10-17

### 2026-10-17 — diagnosis before implementation
Claimed and started from main. start-plan refreshed the stale ariadne tool build and explicitly requested an unchanged second invocation; the second invocation created the canonical branch successfully. No gate bypass.

Read-only audit traced Relaunch → ParkExpected → BeginPark → PublishRequest → request acknowledgement → quit trigger → completion/death → FinalizePark → Resume. Existing controller modes are not a public CLI: park is PresentationTUI and slot admission rejects park. Menu does not expose recovery modes. Existing startup reconciliation inspects/publishes but deliberately does not trigger another quit.

Diagnostic source base from Git: 6df6562411120a83c21262ffc95dfe6e7fa496d6. Temporary same-package probe used existing real ThreadStore plus stateful Pair lifecycle/terminal/process fakes and store hooks. Twelve boundary cases and six cleanup-stage faults pass as reproductions: before begin; canceled after begin; before/after request publication; before request acknowledgement; before/after trigger; completion before child death; dead child without completion; before finalization; interrupted finalization journal; after finalization cleanup; each RunCleanup stage. Missing completion with absent terminal, and failed cleanup with dead owner, both retain open Park and Recover refuses; resume reproduces ResumeParking with no executable advice. Before-begin failure and journal-finalized/canceled returns misleadingly say the park is open. Recover completes the other exact-evidence cases. These are controlled fake/IO failures and context cancellation, not native process-kill evidence or proof of the original pair:6 ordering.

Retained directory: /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair651-repro.6hqyct. Command: go test ./cmd/internal/couchcore -run '^TestProbe651' -count=1 -v; exit0,8.858s. probe.log SHA256 70667c6c8714d99f23ad0137e60337e50a7672ac830d0ce38407e62b24cc70bf; probe651_test.go SHA256 33ede8c19255653279f4270e931e689192437453b308456c388837778d9ebbd0. evidence.json records source head/command. Temporary probe removed from checkout after retention; no production code changed.

Verified pair:6 receipt033e6b3d: its previous recovery screen is unavailable and it never received the original failed relaunch receipt. It can substantiate only TL's earlier “stopped partway through parking scrollback” report, not a prior tag/phase or exact ordering. Do not infer original identity from its new tag.

ARCH-ORDER/SECURE shape the proposed distinction between unpublished rollback, matching completion and dead-owner retirement; ARCH-DRY reuses existing worker, park transition, exact ownership and prepared resume rather than adding a parallel launch or destructive Fresh fallback. TL approval remains pending.

### 2026-10-17 — plan approved before code

Verified TL receipt4ba546a0: public confirmed recover-park/menu and exact-dead-owner retirement policy approved. Add the three operator outcomes after retirement (continue live, resume from disk, start fresh), with executable next-command advice on success and refusal. No automatic operation. Hub confirms it did not retain pair:6's original failed receipt; proceed on the isolated fixture proof and preserve that provenance limit. Durable plan Revisions records the delta; enter change-code before implementation.

### 2026-10-17 — plan-quality refinement PQ-1

First change-code plan review accepted all architectural lenses but blocked on prose test inventories. Replace them with named function-level adversarial strategies and mechanical guards, retaining transition model/acceptance invariants. No production code written. Refined plan names the existing checked-store primitives required for the already-approved bounded rollback policy; ordinary writer semantics stay unchanged. Rerun change-code for remembered disposition.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only. estimate-source flags this calibration stale, so provisional. Plan-quality cleared PQ-1 CLEAN before deriving numbers.

Issue-spec investigation uses design0.5×0.5 and implementation0.3×0.4. Three smaller-Go extensions are recovery decision/controller, atomic checked store transition, and relaunch rollback/outcome classification; each uses design0.3×0.2 and implementation0.5×0.4. The TUI operation/menu extension uses design0.5×0.2 and implementation1.0×0.4. Confirmed CLI/socket/receipt integration is a smaller-Go extension at0.3×0.2 design and0.5×0.4 implementation. Stateful fake/native conformance is another at the same rates. Atlas/help uses design0.1×0.2 and implementation0.15×0.4. One close-review primitive includes full verification/landing at design0.2×0.2 and implementation0.5×0.4. Existing Go context, journal, lifecycle fake, native helper and typed-operation seams cover the stack; no greenfield dependency/library or cross-repo propagation. Familiarity1.0. Thorough-plan design buffer15%. Design subtotal0.71; implementation subtotal1.78; 0.71×1.15+1.78=2.5965 hours. This estimates the whole issue including diagnosis, tests and live acceptance, not only remaining typing; queue/operator delays are not fabricated active work.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.25 impl=0.12
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: tui-screen design=0.1 impl=0.4
item: smaller-go-module design=0.06 impl=0.2
item: smaller-go-module design=0.06 impl=0.2
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.04 impl=0.2
design-buffer: 0.15
total: 2.5965
```

### 2026-10-17 — implementation checkpoint before recorded suite

change-code admitted on01cbcdda1 after PQ-1 CLEAN and estimate-quality INFO. INFO correctly cautions that native conformance, sequential verification and live acceptance are compressed in the provisional estimate; no gate waived. Implementation adds the confirmed declared recover-park operation/menu, authenticated admission-time exact address+nonce binding, context-bounded checked journal recovery and retirement, unpublished rollback, truthful relaunch outcomes and explicit next-action projection. Operator outcomes use positive terminal/native authority; missing terminal identity and unavailable binding refuse rather than guessing Fresh. Busy-store tests exposed guidance's legacy routing lock; changed guidance routing to a read-only bounded view so even error advice honors cancellation. Production wrappers retain ordinary writer semantics; new recovery paths opt into checked locks.

Targeted core recovery/rollback7.312s and extended journal/authority/lock3.588s pass. Eighteen permanent boundary/cleanup matrix cases pass2.468s; delegated relaunch tests2.221s and focused race2.350s pass. CLI/socket/menu/protocol target tests pass across command/message/tty; admission tests pin known-absent nonce against a newly opened transaction. A malformed-address test briefly used a legal component as its negative input; corrected to traversal, not a relaxed validator. Subsequent core race4.632s passes. Final broad targeted run and latest narrow bounds checks are in flight at this checkpoint; required recorded suite not yet run.

Verified native lane grant3e82aae1; three private real-helper cases ran once and failed71.030s before completion/scrollback gates. Log /tmp/pair651-park-recovery-live.log SHA256 a1b58656462bf63222063a6e8d7f8238a9d688f36effc0362110af6b31170936. All private fixtures cleaned, lane released in submitted reportfa3e1619. No native acceptance claimed. First log lacked post-attach PTY diagnostics; bounded diagnostics were added afterward, not rerun. Source audit finds fixture rejects nonzero AttachExistingSession code before consuming quit intent, while production completes handed-off cleanup for code!=0 with err=nil; existing deterministic launcher tests pass this production policy. This is a proved fixture-policy divergence, not yet proof of the native timeout cause. No speculative production repair.

Verified hubc3d1f43e: diagnose retained logs before requesting another scheduled native run after pair:1; continue required suite and close review on fake/journal proof, native conformance explicitly owed. Hub3e82aae1 defers operator live acceptance to next Couch restart because pair:0 cannot be staged; disposable slot to be requested by hub. No live operator slot, :0 staging or restart performed.

### 2026-10-17 15:16 PDT — native diagnosis isolates test-only handoff

Verified hub992ddb22 granted one named finalization diagnostic, unchanged 30s private fixture bound. TestParkRecoveryLiveFinalizationInterruption failed30.04s before constructing/calling RecoverPark. Retained /tmp/pair651-finalization-diagnostic.log SHA2566fa176ad117cb09b1c9c7177d56f73338b99a528372bfbf36dff7a518dee2a96 includes exact helper dead, absent cleanup stages, and PTY diagnostic: Lost connection to the Zellij server; park helper attach returned code=1 error=<nil>. Lane released in submitted report946ecc3f. No native acceptance claimed.

The early-return helper is TEST ONLY: runParkLifecycleHelper in park_lifecycle_live_test.go, invoked by couchcore's launchhelper_test.go TestMain under PAIR_TEST_COUCH_PARK_HELPER. It is not the shipped pair-launch-helper: cmd/pair-launch-helper/main.go invokes LaunchHelperMain, which gates then execs the actual target. Shipped launcher/createflow.go treats an err=nil attach as handedOff regardless of child exit code and runs cleanup; test-only helper incorrectly required code0. Verified hub38ade0a0 asked to distinguish these; submitted reply486224dc records it. This cannot establish pair:6's original failure ordering.

Added a test-only attach seam and deterministic regression: exit1/7 initially failed (helper returned3 before consumer instead of4 for known absent intent); exit0 reached the consumer and execution error refused. Removed only fixture's code!=0 guard to match production. Same regression plus existing production handoff/attach tests pass (core0.261s, launcher0.203s). No production behavior changed by this diagnosis. A new native run requires a separate hub lane grant.

Final four-package targeted run passed (core124.603s, command7.960s, message0.208s, tty2.052s); focused races passed (7.513s/1.517s/1.229s/1.506s). Narrow bounds/publication check passed4.318s. All PR CI checks passed on b4b25013. Recorded full suite remains queued on that head; the fixture/test change requires fresh full-suite evidence before close.

### 2026-10-17 15:18 PDT — native progress and explicit live-owner refusal

Verified hube73ba424 authorized one fixed finalization case, then each other case once if finalization passed. Finalization passed1.01s; /tmp/pair651-finalization-fixed.log SHA256329ce82b730ae524ba226ee29f3d0baccaa3a770063cb3675e97739db1245133. Publication passed0.70s. Scrollback case reached the synchronized gate, then failed its live-owner refusal assertion10.71s: completion reconciliation blocked on the cleanup-held lifecycle lock, helper hit its own10s deadline and died, then recovery safely retired that now-dead owner. No explicit kill occurred. /tmp/pair651-other-native-fixed.log SHA256926363d474841ffdd9f936e8664385f8b09b5d51a627c5439bb3d6719117356c. Lane released in report124fc680; failed case retained, not passed by retry.

Hub5175c159 correctly identified the wait as a production policy difference from the approved live-owner refusal. It allowed deliberate immediate refusal or surfaced bounded wait; chose immediate refusal before completion reconciliation for absent-terminal/live-or-unknown owner. Hub767c5636 explicitly agrees and requires finishing/retry advice. Plan Revisions records rationale: existing blocking lock does not promise a truthful wait bound. Deterministic TestRecoverParkAbsentTerminalRefusesBeforeCompletionLock initially fails for both live and unknown owners because completion callback is entered; it models death while that lock is held. Early owner screening now refuses without entering it or changing the record. Live error says park is still finishing; common error wrapper supplies exact retry command/menu action. Terminal-present recovery and exact-dead completion reconciliation remain unchanged. Native negative assertion now independently checks owner still live before the owned kill; a new native lane is required.

Recovery/interruption/retention focused races passed (core7.097s/storagegc1.184s); final wording assertion checked separately. New retention regression uses actual InventoryRoot and unchanged Decide policy to prove incomplete historical request/trigger/lock/journal evidence remains enumerated, visible-protected, archive-graced and eligible after ordinary retention. It proves eligibility, not physical deletion. No new retention mechanism or production GC change.

### 2026-10-17 15:19 PDT — native fixture conformance complete

Verified hub7bf18e25 granted exactly one scrollback run after the deliberate immediate-refusal fix. TestParkRecoveryLiveInterruptedScrollbackCleanup passes0.98s on8fc9b2112424f455bab793f543ca8aa47cd17930. /tmp/pair651-scrollback-fixed.log SHA25687aedfff047532e70f7b93c58a2a960b8e6f3036744c2582f51a44a22c899214 records the exact helper/group interruption and retained tombstoned history without VerifiedPark. It independently verifies live-owner refusal before interruption, fake/live operator-outcome agreement, retained conversation configuration and harmless repeat recovery. Lane-release report22705cf9 verified submitted. Finalization and publication already passed their granted runs; all earlier failed attempts remain retained.

These fixtures use actual native terminal/helper, lifecycle files and journal IO, with synthetic native-conversation resolution. They do not establish actual agent resume or the original pair:6 ordering. Operator normal relaunch/interruption/recovery/same-conversation/subsequent-delivery acceptance is still owed at the next hub-coordinated Couch restart.

Latest pushed code head8fc9b2112424f455bab793f543ca8aa47cd17930 has recorded suite queued as session45085, worktree sdlc-local-548023597, inflight623465d9cc0aa46db70ba021a7a7ce9b. Earlier b4b25013 suite session72622 was already queued before code changed and continues unchanged; its result cannot satisfy the newer code. No manual main chasing or duplicate same-head run. CI on the new head is pending.

## Revisions

### 2026-10-17 15:19 PDT — acceptance scheduling and plan bookkeeping

Hub3e82aae1/c3d1f43e permitted suite and close on fake/journal proof with native/live owed; subsequent scheduled native fixtures now pass. Real operator acceptance stays deferred to next restart under the same explicit TL scheduling exception in Done when. Tick completed implementation rows, remove the empty scaffold checkbox, and replace the combined verification/landing row with a concrete pre-close verification row. This avoids representing future landing or deferred operator acceptance as already observed; the Log and durable plan retain those commitments.

### 2026-10-17 — required suite catches integration declarations

The running b4b25013 full suite exposed branch defects: TestHelpAndSkillCarryEveryVerbCost panicked because recover-park was absent from verbCosts; TestProductionArtifactReferencesAreExactlyClassified found three new files absent from the exhaustive inventory; TestNoProductionSymbolIsReferencedOnlyByTests identified the exported RecoverPark convenience wrapper used only by tests. Targeted unchanged reproduction /tmp/pair651-suite-guards-red.log confirms all three. Record the heavy cost already documented by the skill, classify the three files as non-artifact sources (they route existing store/lifecycle authority without constructing artifact filenames), and remove the production convenience wrapper. Tests use a private test-only wrapper over the same shared recovery core; real CLI/menu retains RecoverParkExpected and captured identity.

Targeted guard plus recovery/matrix run /tmp/pair651-suite-guards-green.log passes: command0.365s/artifactpath2.121s/core14.231s. No allowlist waiver, gate configuration change or policy relaxation. Full b4b suite remains running to collect every failure; do not count partial output as final. Cancelled only the superseded8fc queued run before execution (session45085, no suite assertions ran), after verifying PID66448 exact start/command, parent chain through this session45374, its test-lock child and private548023597 worktree. It exited with suite lock interrupted/context canceled; no running suite signalled. Corrected code will get a new recorded suite.
