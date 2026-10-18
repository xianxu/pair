---
id: 000651
status: codecomplete
deps: []
github_issue:
created: 2026-10-17
updated: 2026-10-18
estimate_hours: 2.5965
collaboration-mode: collaboration
card_mirror: 'a94c3002bf2c3b35e61fe135cf6f84bb3523ed73' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T13:48:48-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 3.86
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
- [x] Complete the required suite and prepare close review with targeted/race and native fixture proof; record operator acceptance explicitly deferred by the hub.


## Log



- 2026-10-18: closed — Main integration retains private title authority and exact park identity; readiness-cued deadline fixture passes targeted races and full required suite. BR-3 measured-host evidence: exact requested TestTrackedGoFilesPassCatalog passed exit 0 on reviewed 0cb2e215 in owning checkout with verified remote main authority, scanned 2597 sources; retained output /tmp/pair651-br3-exact-head-catalog.log SHA256 ae28d21a9ff22358b1995a3a8ddfd0192c4394904b50db23261163c52ccef81c. No source or guard change after that suite. Prior native fixture proof retained; operator disposable-slot acceptance remains owed next coordinated restart; TL owns project. (fixed-then-ship; full suite: Full required suite exit 0 on 0cb2e215d6087eea67e96e9cc7a3500727c3cdd6; attempt 9dbe4d89404e5331e7db045f17888636; log SHA256 d02c105757cfc0ee4b7d4dad94bbf9929c0ad91e37ebbce4e37b3e5aedb30b43; successfully posted sdlc-local/suite. Subsequent commits contain only review ledger and retained verification evidence.); review verdict: SHIP
- 2026-10-18: closed — Required full suite exit0 on 948470412f7252834ae432f26234a6c6223d9972, attempt17840d24c5974d633e014875abd98cbd, recorded sdlc-local/suite; 868.24683025s execution after1240.741971s queue; log SHA2564ba094c4bb7c93664bfc9ed0089d40bd68b00ce312ed870e09bafdb2f2532533. Real lifecycle/store lock contention regressions prove canceled worker settlement and reservation release; publication uncertainty, exact ownership and production-entrypoint advice coverage pass; three hub-granted native helper/terminal interruption cases pass with synthetic conversation resolution. BR-1 and BR-2 corrected with red/green proof in Log; complete failed attempts retained. TL owns project, hence --no-project; operator same-agent acceptance explicitly deferred to next coordinated restart.; review verdict: SHIP
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

- 2026-10-18: Integrated full suite passed on exact pushed HEAD `0cb2e215d6087eea67e96e9cc7a3500727c3cdd6`, attempt `9dbe4d89404e5331e7db045f17888636`, exit 0, no timeout; queue 2364.320551 s, execution 796.4564145 s. Log SHA256 `d02c105757cfc0ee4b7d4dad94bbf9929c0ad91e37ebbce4e37b3e5aedb30b43`; successfully posted as `sdlc-local/suite`.
- 2026-10-18: Fix review round 3 found no code defect, disposed BR-1/BR-2, and raised BR-3 (`verified-catalog-execution`, Important/unverifiable): its disposable checkout had no remote main authority, so the catalog test refused before scanning. Re-ran the exact requested consumer on unchanged HEAD `0cb2e215d6087eea67e96e9cc7a3500727c3cdd6` in the configured owning checkout: `go test ./internal/repolint -run '^TestTrackedGoFilesPassCatalog$' -count=1 -v`, exit 0, package 2.355 s; checked 2597 tracked Go sources, five shared rules plus Pair native-fixture-roots, 100 exact sleep debt sites. Verified remote main ref was `refs/remotes/origin/main 9c1228cc6f01b47eb82bad0fc16e58a9d913c5f2`. Retained output `/tmp/pair651-br3-exact-head-catalog.log`, SHA256 `ae28d21a9ff22358b1995a3a8ddfd0192c4394904b50db23261163c52ccef81c`. This is the same reviewer-environment class already recorded by #666 BR-1 in lessons; use the measured-host-evidence dispute protocol, preserving the guard and source code unchanged.

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

### 2026-10-17 — complete failed suite inventory and coverage domains

The first full b4b25013 attempt finished exit2 in12m55s after785s queue. Complete retained log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/58a58ba1c3675fe49e82c9f6df20f029.log SHA256ef0ae6e0eff374f6f599d9c964f344fa2b7e94df6fc98e7746af72522170633d. All failures were branch omissions: the three declaration/dead-symbol tests already recorded, TestTheContinuationGuardsReachMatchesEveryRowAction and TestRowActionDeclarationsAndTheMenuAgreeInBothDirections. Every other package completed, with no unrelated failure. This is a failed required suite, not a flake or passing evidence.

The latter two fail again in /tmp/pair651-row-domain-red.log. Add recover-park to the continuation dispatch sweep (already-settled live recovery succeeds without starting a replacement), and extend the shared menu row domain over open-park presence, with an independently stated expected action. No exemption or waived assertion. /tmp/pair651-row-domain-green.log passes all five failed guard names plus offered-action reachability and menu tests (core1.152s/tty0.678s/artifactpath3.747s/command0.373s). Cancelled the superseded4c queued suite before execution after exact start/parent/queue-child/worktree verification (PID35387/private3911013580), preserving its cancellation; it ran no suite assertions. Only the corrected complete head will be counted. Wait for full attempt exhaustion before starting another corrected suite, so a late failure does not force needless requeueing.

### 2026-10-17 — expanded row-domain checks completed

The shared open-park domain also reached three older expectations outside the initial targeted-name filter: primary Enter, generic unconfirmed Recover, and whole-action advice matching. Keep open-park Enter's existing no-effect Tab→Recover park guidance where no ordinary Enter operation exists, distinguish confirmed transaction recovery from generic unconfirmed recovery, and recognize longer action labels before their constituent names. No operation exemption. Normal primary add-slot expectations remain scoped to ordinary primary rows; an open transaction has its own recovery priority.

The expanded status assertion found a production display gap: live rows hid transaction advice behind Continue. Show the common unfinished-transaction advice in threadStateText, preserving rootStateText's appended continuation/missed status. Full couchtty package passes58.743s in /tmp/pair651-couchtty-complete.log. An additional status-preservation regression then went red because an initial early return hid continuation status; moving the advice into threadStateText fixes it. Final affected-domain/status/menu tests pass16.118s. Plan Revisions and atlas record the shared guidance behavior. These are display/test-domain fixes, not a changed native recovery mechanism.

### 2026-10-17 — second complete suite and executable skill example

The corrected013c1f717e61508f3874aa18627ee3a0484fe6e5 required suite completed exit2 in14m32s after1448s queue. Every package passed except TestSkillDocumentsRecovery: the new embedded skill example included literal optional-argument notation [--json], so its verbatim command did not parse. Complete retained log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/d4babe47e5c7bcb0b0f4c45707e11306.log (hash recorded below). Core475.057s, tty51.605s, artifactpath4.146s and all other packages passed. Do not call this a passing suite or a flake; it is an executable documentation defect.

Unchanged targeted TestSkillDocumentsRecovery reproduced the refusal. Use the literal executable command without bracket notation, and describe optional --json separately; no parser relaxation. A complete command-package run on13c plus this working-tree document change failed only TestSessionSyncWriteTelemetryBestEffort/timeout at509.32225ms against500ms (package176.028s). This matches existing open pair#596; source still uses the unchanged200ms telemetry context and the test measures the whole durable write against500ms. Scheduling/filesystem contribution remains hypothesis, not proved timeout cause. Retain /tmp/pair651-couchcmd-complete.log SHA256a2285ecee14f28619cadd76e224005a36fe340020c6ea9fb5ec15a5f80641135. One unchanged complete-package rerun on the identical working tree passes106.043s in /tmp/pair651-couchcmd-complete-rerun.log. No source/timeout/flags changed between those two attempts, no new flake filing because#596 exists, no blind retry. The required suite on13c did not report#596; its only failure is the separately fixed skill example. Commit the example and verification before queueing the new required suite; do not chase main.

Second full-suite SHA2569791680ca3d9fa193fed700af5a07aa67a7b983b4b73ee1921926aeac7b06866; unchanged command-package repeat SHA256d7b9c202d4be50fc7880d05d738db384c112ae02b3074ae0bdc72684485c99d1.

### 2026-10-17 — full suite passes; status publication gap recorded

The required suite on80063e6d8f7f29e4cc83ad0eb8d7d757b551c393 PASSED, exit0, no timeout or failures: command78.663s/core466.413s/tty50.972s and all remaining packages/recipes pass. Attemptd79d57fd973df98f92b03617bf772697 records1763.113822s exact queue wait and851.610315667s execution (14m11.61s), with UTC endpoints retained in .git/sdlc/test-stats/attempts/d79d57fd973df98f92b03617bf772697.json. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/80063e6d8f7f29e4cc83ad0eb8d7d757b551c393/suite.log SHA256b52194a2dbbd18ddfccd70c0157df248c9f33bf90c7938fd25b946b818d61c9b. This is the actual complete test pass; the CLI exits1 afterward because its GitHub status POST failed (error connecting to api.github.com). A subsequent successful read-only API query confirms no sdlc-local/suite status on that SHA. No manual status publication or gate waiver.

Inspected current local-check help and source: run always executes a suite; no publication-only recovery exists. Reportbc03fb50 verified submitted. Verified hub9f2f29b6 agrees, filed recovery gap as ariadne#431 (pair:7 also hit it), explicitly authorizes close review now and normal merge to run/record its suite. No source change in ariadne. Native fixture conformance remains passed; genuine operator acceptance remains explicitly deferred per3e82aae1/c3d1f43e to the next coordinated restart. TL owns the project file; close uses --no-project for that ownership, not to omit operator acceptance.


### 2026-10-17 17:27 PDT — close round 1 REWORK and approved correction

The binary-owned close review on59c8d4f0 returned REWORK; close is not finalized and the card remains working. BR-1 Critical identifies lifecycle flock and drive/finalization legacy threadstore paths that can retain a worker reservation after caller cancellation. BR-2 Important identifies incomplete advice across continuation/recovery/switch callers. Hub3bae113e approves the context propagation correction, forbids goroutine escape, and asks for any unavoidable unbounded legacy operation to be named. The durable plan Revisions enumerates every reachable IO and advice caller plus updated integration concepts and regression strategies. No new persisted lifecycle state or authority relaxation.

TestRecoverParkCancellationSettlesContendedLifecycleWorker initially fails for both dead-absent and terminal-present cases: canceled recovery retained its worker behind the real lifecycle flock. The contextual fix settles and releases the reservation while the other lock stays held. TestRecoverParkSettlesWhenThreadStoreBecomesBusyAfterAdmission passes with contextual drive/finalization calls; restoring their legacy calls reproduces both one-second reservation failures. Initial focused four-package races passed core11.906s/lifecycle1.166s/command1.573s/tty1.524s. More complete final targets and a new recorded full suite follow; no stale suite counted for changed code.

Actual publication cancellation regressions distinguish before-write NotCommitted from after-rename Indeterminate with reader reconciliation. The before-write residue assertion exposed canceled cleanup skipping its own temporary file; keep synchronous owned staging cleanup while published request/completion cleanup explicitly checks cancellation. No detached IO goroutine. The existing interruption matrix's late hook assignment was lost by a context-routed backend snapshot; install its armed journal hook before routing, preserving the same durable-finalization interruption point and invariants.

Review's complete-package run also reported the pre-existing registration-boundary failure already recorded in pair#432. Its explicit-recovery/retry coalescing failure passed alone, but the50ms sleep did not prove retry admission; replace it with a gated same-nonce worker join before releasing the trigger. Keep this review attempt distinct from the prior complete required suite pass. Hub856c5ced granted exactly one short rerun of the three existing native cases, after targeted proof, with the same private/no-focus/30s bounds. Operator acceptance is still owed next coordinated restart.


### 2026-10-17 — review corrections targeted and native proof

BR-1 final focused races pass core2.588s/lifecycle1.200s, including actual flock cancellation, later threadstore drive/finalization contention, canceled evidence observation/cleanup and publication-outcome/staging-residue tests. /tmp/pair651-br1-final-race.log SHA2564511f3bb663a4ae18d9bf075dbfa3c2d801503b67c655b97dc3561508023b64d. The broad recovery/interruption/store targets pass core17.688s/lifecycle0.179s. Explicit recovery/retry coalescing now synchronizes on the actual same-nonce join; its focused race passes1.426s.

BR-2 production-entrypoint tests reproduced each missing refusal class before the shared advice sweep; failed targets retained at /tmp/pair651-br2-red.log, -extra-red.log, -guard-red.log and -stop-red.log. Final focused core race run passes39.050s in /tmp/pair651-br2-complete-race.log SHA256109215eff3ffcf57f9a59cf115dcda48f092278fdb8b798af52a25a227de5806. This includes plain/pending/running resume and switch guards, direct recovery, continuation publication/source parking, foreign live/unknown owner and abandonment conflict, detach, Park/Leave new/existing source stop and prepared recovery. The existing typed error and ownership decisions are retained.

Verified hub856c5ced grants one three-case native window. Complete native run /tmp/pair651-review-fixes-native.log SHA256f83761333371503118e6d17a6739050a0179e29ea2c6d8358e4c1f325bb1dc28 fails only finalization's missing fault hook (publication0.71s and interrupted scrollback0.91s pass). Retained diagnostics prove cleanup completed and VerifiedPark was published; its late hook assignment was lost by context routing's copied hooks, exactly the deterministic matrix's diagnosed test issue. Install its armed hook before routing, no production change; matrix/journal targets pass1.938s. Hub567ebdc6 pre-grants exactly one finalization-only rerun after that proof. It PASSES1.03s (package1.208s), observes the injected lost journal reply, reopens to the durable settled state and repeats harmlessly. /tmp/pair651-review-finalization-fixed.log SHA2567ba8a237e579ceee977b3821e76e42e5f1a866221765e628f9cfe64009b9f8f5. Lane released in5ec7c480; fb3cfb4e corrects an initially misstated0.78s timing to the observed1.03s. No additional native process/test remains. These native fixtures still use synthetic conversation resolution; real operator acceptance stays owed.

Commit/push the complete correction before the next recorded full suite. No stale80063e6d status or test pass is counted for these code changes; no main chasing or review/CI bypass.


### 2026-10-17 — final pre-suite exhaustive guards

The exhaustive dead-symbol guard caught three wrappers introduced during BR-1 correction: all production consumers now use contextual mutations, so legacy AdvancePark/AppendParkAttempt/FinalizePark wrappers had only fixture users. Remove the unused API layer; existing exported methods now take the explicit context and production/tests all call the same mutation bodies. No allowlist waiver. /tmp/pair651-review-final-guards.log passes artifactpath1.704s/command0.339s/core8.744s/tty0.287s/gccmd0.141s, including dead-symbol/artifact/help/skill guards and recovery/interruption/domain targets. The already granted native proof used those identical contextual bodies before the mechanical API rename. git diff --check is clean.


### 2026-10-17 19:22 PDT — corrected-head suite stopped by dependency DNS outage

Attempt43569e230dd84d0f7fd7d7a0db666453 on ecf055bee02b7abcc6c738993d49dcd0062d25ed completed exit2 at17:55:53 PDT after1281.613219s exact queue wait and203.799178542s execution. Retained complete log .git/sdlc/test-stats/failed/43569e230dd84d0f7fd7d7a0db666453.log SHA25600ef8a5dcf89025a687478068d06427fa5a0f911879626f4d952a2603b994f19. The install-layout fixture's fresh module cache could not resolve proxy.golang.org (no such host / i/o timeout) for multiple independent dependencies. make stopped there before the complete Go package suite; this is not a passing or complete coverage run. Subsequent GitHub status POST also failed. No assertion failure or branch defect is identified by this attempt.

At19:21 PDT unchanged read-only GitHub API query succeeds and the exact module proxy endpoint returns HTTP200; network reachability has recovered. Preserve the failed attempt and make one unchanged recorded suite run on the same pushed code. Its declared argv is fixed to make PAIR_GO_TRIMPATH=-trimpath test; local-check exposes no keep-going override. Do not alter gate configuration or hand-post status; use the complete recorded repeat to cover every target rather than adding an unrecorded full suite plus another recording run. Report67d334e5 and clarification to hub record the decision. Any further failure stays blocking, with the complete failure inventory examined before another decision; no blind passing retry.

Pair:7's #576 planning request36031e75 verified submitted. Replied c1bd27d8 with exact confirmed action, captured address/nonce admission, receipt/result API, worker reservation semantics and menu/row projection changes to preserve; corrected a mistaken extra filename in23b3455f. Peer will implement from main after #651 lands, not stack here.


### 2026-10-18 — complete unchanged rerun found three branch omissions

Hub check-in e10f4caf verified submitted. Attempt94496de57bcf61774df26fcd5b7f0452 on ecf055bee completed all recipes/packages, exit2, and successfully recorded its failed GitHub status:703.616224s queue,712.8783325s execution. Retained failed/94496de57bcf61774df26fcd5b7f0452.log SHA256f86ee104496f8cfb8f445d66297d13cf8a8141c5bda0e91785067049148bfc23. Exactly three failures: TestRelaunchFinalizedJournalWithLostReplyReportsParked, TestEveryLifecycleTransitionIsDrivenIntoItsOwnRefusal and TestNoDeclarationCarriesTwoStackedGodocs. No unrelated assertion failures; DNS failure did not repeat. This is a failed complete run, not a pass needing republish. Hub report8b9ed5b8 identifies the full inventory.

Focused unchanged reproduction /tmp/pair651-suite94496-red.log fails all three. Relaunch outcome fixture still installed its journal hook late, so context-routed copied hooks never received the intended fault. An AST scan across every cmd test found exactly this remaining nested AfterJournal assignment and one AfterGetThread assignment; the latter executes on the original store after reads, is independently covered by the unreadable-outcome test and remains valid. Install the relaunch journal hook before routing and arm it at finalization, retaining all durable-state/no-launch/outcome assertions. No production behavior change.

The refusal inventory's call-graph root was the legacy updateExistingThread wrapper, missing methods using updateExistingThreadContext. Derive from their shared updateExistingThreadOnBackend journal/CAS sink instead, covering both routes without an exemption. This also brings RetireInterruptedPark into the inventory; add its real exact-owner refusal and unchanged-revision proof. Finally, the new Remove doc opened with another declaration's name (Store), tripping the existing Godoc guard; make it describe Remove directly, without weakening the guard. Targeted relaunch/refusal/Godoc verification follows before another pushed-head required suite.


### 2026-10-18 — SHIP and current-main integration conflict

Required suite17840d24c5974d633e014875abd98cbd on948470412f7252834ae432f26234a6c6223d9972 passed exit0 in868.24683025s after1240.741971s exact queue wait; sdlc-local/suite was published. Log SHA2564ba094c4bb7c93664bfc9ed0089d40bd68b00ce312ed870e09bafdb2f2532533. Close round2 returns SHIP, zero new findings, both BR-1/BR-2 addressed; actual3.16h is adopted, codecomplete bound to f7b2c18b0de058aae6610d9f9e05c25c0d20c1d0 and mirror328c098c1b23f68eb508e4dc55a121349af52704. Normal merge entered immediately and waited first behind#666.

When admitted, sdlc merge attempted current main db5ddb4e2bf5af6ec85c3c916e51b17523c12085 and stopped on one conflict in actionableinventory.go; it restored the branch and explicitly requires manual integration, recorded full suite, then close --fixed-then-ship --full-suite. Resolve the adjacent struct additions by keeping HasOpenPark/ParkNonce and main's private TitleTarget pointer; retain both open-park projection and applyTitleTargets. No behavior is selected against the other. Focused integrated recovery/actionable/title/open-park targets pass core8.485s/command0.444s/tty0.394s/gccmd0.190s. Full cached whitespace check flags an existing imported main history review line in#647; leave unrelated archived peer evidence intact. The conflict file's check is clean. Integration must get its own full suite and fix review before landing.


### 2026-10-18 — integrated catalog catches matrix deadline shortcut

Integrated061a4108c8cdab32433203e121f48ad1e7a69c4b suite attempt8acc664b04cd7e939e2fc6d5185a13ce completed exit2 after859.105854s queue and779.186064792s execution, failed status recorded. Retained failed/8acc664b04cd7e939e2fc6d5185a13ce.log SHA2562c0776ac18c45293148e8cbde14fcbd563fe8646b685a30af8487a74c6663d99. Every behavioral package passed, including core335.656s, command57.975s and TUI52.145s. Sole failure is internal/repolint TestTrackedGoFilesPassCatalog: main's new test-shrunk-deadline rule identifies matrix line74's one-millisecond completion timeout before readiness. No unrelated failures. Hub report5bc76c57 records the complete inventory; this is not a passing integration run.

Focused catalog reproduction fails unchanged in /tmp/pair651-deadline-lint-red.log. The diagnostic names ariadne's internal testfix.DeadlineOnCue, which Pair cannot import across the internal-package boundary. Reuse Pair's existing spendableBudget and completionDeadline seam already used by park_wait_test.go instead of copying that helper or suppressing the rule. Keep production wait settings; spend each fresh test budget from Wait only after independently observing matching completion and the exact child still live. Assert the cue ran and the relaunch error includes context.DeadlineExceeded, while retaining the existing proof that recovery cannot retire the live child and succeeds only after exact death. This removes a scheduling race rather than weakening the timeout proof.

Focused race verification passes core3.724s/repolint17.599s in /tmp/pair651-deadline-cue-green.log. No production code, native behavior or public API change; existing native proof remains applicable. Commit/push this fixture correction, run its required recorded suite, then the already-required fix review over the main conflict and this test correction before landing.
