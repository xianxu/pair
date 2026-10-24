---
id: 000863
status: working
deps: []
github_issue:
created: 2026-10-24
updated: 2026-10-24
estimate_hours: 1.333
card_mirror: '4fd092766f657f1e4bcb64c796571240b59e037a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T00:18:23-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# Native fleet: explicit-hub Fleet.Start broken since #739; second explicit hub fails attach at a fresh start

## Problem

Found by pair:6 while writing #830's T6 fleet test (2026-10-24).

1. **Harness:** since #739, `testcouch.Fleet.Start` cannot start an explicit-hub fleet. `Start` waits for the primary hub's agent (`WaitAgent(ctx, 0)`), but #739's startup launches no hub: the switcher shows each configured hub as `hub · Ctrl+Space: start DIR`, with the notice `startup: full-shutdown; missing hubs: …`. This happens even in a fresh namespace. Every `ExplicitHub` fleet test therefore times out at "steward startup": `TestDisposableEditedRestart`, `TestDisposableStartupArguments`, and `TestDisposableCouchSYNTHETICLimitAndBusyForcePreserveDirtyFiles`. Reproduced on clean origin/main 54c4a59ca:
   `PAIR_NATIVE_COUCH_FLEET=1 go test ./cmd/couch -run '^TestDisposableEditedRestart$/claude' -count=1 -v` → `startup_restart_fleet_test.go:44: steward startup: context deadline exceeded; screen=threads`.
   These tests are outside the normal suite, so nothing caught it.
2. **Likely real bug:** with two explicit hubs (`couch PRIMARY WORKER1_DIR`) in a fresh namespace, the second hub's `start-hub` fails. The notices were `hub failed; open it in the switcher to retry: …` and `clean after a failed attach of : attach record/handle process identity mismatch`. The attach target name was empty. The failure evidence from the harness read `final="attach record/handle process identity mismatch"`, with slot and stage empty. A fresh namespace has no records, so `StartInteractive` should have nothing to attach.

Screen at the failure:

```
threads
Ctrl+Y copy last notice / selected error · Ctrl+G hub automation on/off
  hub · Ctrl+Space: start /…/fleet/fixture
  hub · Ctrl+Space: start /…/fleet/worktree/fixture-slot1/fixture
  (no match)
startup: full-shutdown; missing hubs: /…
```

## Spec

- `Fleet.Start` drives the same start gesture the operator uses after #739 (the switcher's start for the primary hub), so explicit-hub fleet tests run again.
- Find and fix the second explicit hub's attach failure at a fresh start.
- Decide whether a fresh namespace (no records at all) should count as a full shutdown. The startup notice currently says `full-shutdown` for a Couch that never ran.

## Done when

- The `ExplicitHub` fleet tests pass natively on Claude and Codex.
- A fresh start with two explicit hubs raises no `hub failed` notice, and a test covers it.
- The fresh-namespace classification is decided, and the notice says what actually happened.

## Plan

See [durable plan](../plans/000863-fresh-hub-startup-plan.md).

- [x] Qualify pure fresh and no-child boundaries, integrate existing lifecycle launch/adoption, test duplicate Enter, and verify native two-hub callers before close.

## Log

### 2026-10-24 — intake and production-case diagnosis first

Claimed in pair:7, then ran bare start-plan from main-slot7 after #842 finalized SHIP and entered external check/stage wait. Branch base99846a5015f753f05edaf4778618849f377a2f71; no #842 commits stacked. Refresh only deferred the seeded workflow; no generated vocabulary edits to discard. #842 retains recorded lease3238290542 and its own branch; do not duplicate or cancel it for #863.

Hub563a3e8a decides zero history is fresh: hubs start normally, hub automation OFF, no automatic agent input. Full shutdown is known prior records whose sessions are absent; retain #739 no-auto-resume there. Hub843a4c5e prioritizes confirming or ruling out the production caller failure before any fix plan. No production edits until measurement and approved design.

Pair:6 f29ca79c reports second-hub error at375adcb6c with no gesture: fresh couch PRIMARY WORKER1_DIR, automatic queueExplicitHubs -> start-hub -> StartInteractive. Original /Volumes/TestScratch/pc-3413210420 was removed, so only reported teardown text remains (identity mismatch; empty slot/stage, no wrapper log). Single explicit hub separately times out in Fleet.Start. Pair:6 later8175ab45 reports manually starting primary with Ctrl+Space form produces a ready Pair-wrapped Claude fixture, but inherited ActorCLI --actors returns caller lookup: recipient is not live for120s. Treat these as reported, not retained/replayed evidence.

#830 has unlanded Hubs and StartCold helper options. Earlier no-cycle advice was corrected by9bfb9542: its T6 is blocked by caller registration; TL decides sequencing. Do not import/stack #830 source. Production-case experiment will reuse landed Fleet.Start in an owned goroutine, drive the real form through existing Input/Press, cancel/join only Start's observer after fixture readiness, then invoke real caller CLI from inherited fixture credentials. This measures current main without changing production or inventing a duplicate fleet owner. Save full bounded test output outside the cleanup root, and join cleanup.

### 2026-10-24


### 2026-10-24 — first native production-case result: one hub registers

No production changes. Landed shared Fleet owner plus real start-form gesture on main runtime99846a5015: native Pair-wrapped Claude fixture becomes ready, first inherited --actors returns not-live, second observation2s later returns Code ok with fixture:0 registered. PASS27.29s, exact owned generations absent/direct child and IO joined. This rules out a permanent caller failure only for this exact single explicit-hub scenario; the reported two-explicit-hub case remains unclassified. Fixture readiness is not broker readiness. A subsequent fresh read can legitimately succeed; do not cache its first refusal as permanent failure.

Retained synthetic native captures (outside the removed owner roots):
- [Final one-hub pass](../evidence/000863/pair863-main-fresh-hub-caller-observed.log), SHA25622439a0d75f7c50d43b0c591547fb9f063a27ab7449aad6d4abd64e1771d2d73.
- [First probe setup error](../evidence/000863/pair863-main-fresh-hub-caller.log), SHA25653fc119a7c37a7b855c82bbfa76336a5f18027c45c2dffbae7a6f293b3b5685b: waited for parameters from a different form; actual start form had args repo-default, never launched an agent. Not production defect evidence.
- [Second probe oracle error](../evidence/000863/pair863-main-fresh-hub-caller-corrected.log), SHA25649a5c34214ca97aa8304767772c96de779f18c16a1c161d66855b3a598b9e943: reached ready fixture and known exit1/not-live response, but mistook ActorCLI's known nonzero result error for cleanup uncertainty and stopped too early. CleanupUncertain was false. Not proof of persistent refusal.

The final probe corrected those consumer predicates based on observed/source evidence, without deadline increase or production changes. Read-only audit independently traces same four inherited registration/caller fields and rules out generic ambient credential contamination. Full wrapper peer log is not represented by harness startup excerpts; absence of PEER-start-fail in those excerpts proves nothing. No fix selected yet; plan follows the exact two-hub measurement. Test file is diagnostic and currently opt-in.

### 2026-10-24 — exact two-hub production sequence admits primary caller

Hub c1133e63 approved the missing Hubs option without waiting for #830. Coordinated exact contract with pair:6 f78a1caf: Options.Hubs []int, distinct provisioned worker indices1..Workers, requires ExplicitHub, ordered after primary argv. Added only that harness seam, not StartCold/QuitConsole or unlanded production. Test first failed on absent Hubs, then ordered argv and invalid-option controls passed0.201s.

[Exact two-hub native log](../evidence/000863/pair863-two-hub-caller.log): PASS54.57s (Claude34.74s), production still main99846a5015. Fresh couch PRIMARY WORKER1 reproduced extra hub attach failure. The real form-started PRIMARY became ready; after joining only the Start readiness observer, its inherited --actors returned Code ok on observation0 with admitted fixture:0 wrapperPID17437. Cleanup proved all owned generations absent and direct child/IO joined. This disproves a necessary causal link from failed extra-hub adoption to persistent primary caller refusal. It does not explain or invalidate pair:6's reported120s outcome; that remains unclassified pending harness comparison.

Read-only independent audit: hub-effect gate is released when empty StartInteractive returns, before adoption; AbortStarted with nil Handle returns before registry access; failure recording only changes diagnostics. No evidence warrants a speculative registration fix. Confirmed production root: start-hub returns empty StartResult,nil for absent hub, Started() says true unconditionally, and adoption refuses zero record/nil handle as process identity mismatch. Fresh namespace also classifies full-shutdown today. Plan will address these confirmed boundaries and the approved fresh-start semantics, with registration native acceptance retained.

## Revisions

2026-10-24 — Hub563a3e8a: the Spec's harness-only start-gesture repair is superseded by fresh-start policy: zero-history configured hubs start normally, automation off; known shutdowns stay parked. Preserve no automatic hub input. Hub c1133e63 authorizes matching #830 Hubs input for measurement without stacking. Hub bc2786cd adds conditional production duplicate-Enter correction if two launches are demonstrated. Durable plan records both proven defects and the still-unclassified peer observation; no speculative registry change.

2026-10-24 — Hub7c48d75b approved the plan. Pinned full flow because startup policy crosses core/orchestration/native acceptance. Plan-quality PQ-1 asked for function-level adversarial strategies rather than prose case inventories; revised the whole test-strategy class, preserving design. No production implementation before the gate passes.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; estimate-source marks calibration stale, so provisional. Three well-specced smaller Go extensions (classification/child contract, startup orchestration, form/harness qualification) each use upper v2 impl0.5h scaled40%=0.2h; design0.3h discounted by the approved detailed spec to0.06h. One native-boundary discovery budget uses0.6h x40%=0.24h. Docs use design0.2h x0.2=0.04h, impl0.2h x40%=0.08h; one review chunk uses0.4h x40%=0.16h. Existing lifecycle and native harness eliminate any novel-stack/library work. Familiarity1.0; thorough-plan design buffer15%. Total0.22 x1.15 +1.08 =1.333h.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: real-api-discovery design=0.0 impl=0.24
item: atlas-docs design=0.04 impl=0.08
item: milestone-review design=0.0 impl=0.16
design-buffer: 0.15
total: 1.333
```

2026-10-24 — Pair:6 b5498b2e explicitly retracts unverified double-launch hypothesis: no second-launch evidence, corrected helper failed to launch and had no captured screen. It polled actor0; producer context unchanged. Shared our accepted-preview predicate and single-start Enter sequence (Down requests preview, args repo-default means accepted). Persistent peer not-live remains unclassified/likely helper, not production proof. Held-effect reducer qualification stays in scope without new state absent a failing invariant.

### 2026-10-24 — implementation and first native qualification

Full change-code passed after PQ-1 strategies and calibrated1.333h estimate (advisory qualification budget risk). One earlier estimate review was rejected stale because I edited this Log while it ran; reran unchanged inputs, no bypass. Pure startup/child TDD red then green; combined cmd/core/tty startup, automation and form tests PASS6.045s/15.807s/0.733s. Core generated192 contradictory/uncertain/order combinations. Repeated Enter while completion/preview held gives exactly one start; the independent lost-state negative control produces two and is rejected. Existing form state suffices, no production form edit. Pair:6 retracts its unverified double-launch hypothesis and adopts our actual accepted-preview predicate.

First fixed-source native run at25a6c4847 has two classified failures: Claude's primary Zellij0.45.1 server panics before wrapper startup; extra managed hub refuses Open as resume-no-survivor. The latter is our path selection mistake: Open explicitly repairs/resumes and cannot create a never-started managed slot; hub7cf1b120 approves existing Fresh for proven zero-history only. Red queue-intent control expected fresh but observed open; changed permission, targeted cmd tests PASS2.835s. Added a declared-dispatch managed-slot check (trusted implicit call) to exercise the actual domain path, not merely argv construction.

Zellij evidence: [server log](../evidence/000863/fresh-fixed-claude-zellij.log), [independent extra-hub refusal](../evidence/000863/fresh-fixed-claude-launch-failures.json). Installed Homebrew0.45.1 source tag commit efd8fd5a89a20c07a111d248ad7fce53848d2c18, zellij-server/src/lib.rs1462: RemoveClient unwraps absent session_data after remove_client!. This is internal client-lifecycle handling, not parsing a malformed Pair argument or geometry. Pair's normal startup probes may trigger early-client removal timing; exact timing causation remains unmeasured. No new Zellij argv/environment introduced, no speculative Pair workaround. Hub directs retaining this classification and no upstream report without standalone non-Pair reproduction; do not chase that tonight.

Reader inventory audited: new native consumer uses shared Build and real runtime bundle/native tools, therefore non_go_full_fallback; fleet_test uses literal paths/private structs only, local_or_literal_fixture. Recorded only those named hashes, then make test-affected-inventory passes. Atlas documents fresh versus known shutdown, immutable automation evidence, and empty result no adoption.

2026-10-24 — First native matrix completed FAIL456.39s: Claude214.17s Zellij RemoveClient metadata unwrap, Codex217.00s primary ready but extra managed hub's Open refused; exact owned cleanup passed both. Full capture [native-first-fixed.log](../evidence/000863/native-first-fixed.log), SHA2562320f9563c197a28f32160e7150bcf7e0d685be60ed1896f28aa760fbde4e38c. No native acceptance claimed. Corrected Fresh-authority head44dd64305 has one two-personality native qualification running; no deadline extension. Recorded pre-close lease1828715990 / attempt b91e09de6459b14844859cc26f3e5f8b queued on44dd64305; #842 separate lease3238290542 retained, not superseded.

Hub4161aeef requests a separate Pair follow-up AFTER #863: avoid startup probe client attach/detach before Zellij session metadata readiness, using retained v0.45.1 panic/source pointer. Do not fold that mechanism into #863. Owed filing after landing. Pair:6 1a632b61 explicitly retracts its not-live reading as its own helper; adopting our accepted-preview gesture allowed T6 through all three actors, quit and cold-start; other blank-screen failures remain separate startup flakiness, not evidence of registration poisoning.

2026-10-24 — Second path correction, stopped and approved by hub9a53d319 before code: Fresh is illegal for a real ordinary primary (slotstart.go162); legacy no-SlotCatalog fake concealed this. Consolidated StartFreshHub resolves the exact target once with Open, routes numbered target through StartFreshSlot and ordinary target through shared startInteractiveResolution. Generic StartInteractive is not relaxed. OS SlotCatalog-backed primary+numbered tests red on missing helper, then core/cmd fresh checks PASS2.852s/0.772s. No third path change allowed without re-plan. The second native matrix at44dd64305 is a failing-source capture, not acceptance; no further native run until it joins and final source is committed. Cancelled queued #863 lease1828715990 because production changed; #842 lease3238290542 remains valid and retained.

2026-10-24 — Final target-aware source b5534bb1f0c106475dbe093fb7241805e6ec805b: combined targeted core/cmd/tty/testcouch startup, fresh targets, form, automation and Hubs tests PASS12.702s/4.270s/0.579s/3.945s; inventory PASS. Final-source native matrix running once after the prior failing-source matrix joined cleanup. Prior blanket-Fresh failure capture [native-blanket-fresh-failed.log](../evidence/000863/native-blanket-fresh-failed.log), SHA2562a67b1d33710f903d2099712f8a45a3625e1fd8849ddc532e891ea83d343e817: ordinary primary rejected Fresh before any Zellij/agent started; not an external startup flake. Both owned generations absent and IO joined.

Current recorded check: lease679043877, attempt451b040be2f87c45e29114b37238acec, checkout /Volumes/TestScratch/local-checks/sdlc-local-3218857594, headb5534bb1; pre-close affected correctly falls back to suite for the native reader. Retain runner/session91224 and queue position; no duplicate. Prior lease1828715990 cancelled while queued after source changed. #842 remains independently queued on27f192b8, lease3238290542/session37701; never cancel it for #863. Next: classify final native result, retain it, finish plan/atlas evidence, wait final recorded pass, inventory then close/review and land. After #863 land, file the hub-requested Zellij probe/readiness follow-up. No third path correction without re-plan.

2026-10-24 — Final-source native acceptance PASS39.45s (Claude15.61s; Codex14.95s), sourceb5534bb1f0c106475dbe093fb7241805e6ec805b. [Capture](../evidence/000863/native-target-routing-passed.log), SHA2569d8b7d54e924f7ff384577630f537a1fd073c4c2e179c2733b08e8494779841b. Both configured hubs automatically started, each inherited caller queried --actors successfully; automation status is off with startup:fresh; Accepted input empty for both; no false hub-failed/identity-mismatch notice; all exact owned generations absent and direct child/IO joined. Codex extra hub's first early caller read was notlive and the next fresh observation2s later succeeded, matching readiness/admission separation. No retries of a mutation, new broker policy, third path correction, timeout bump or live fleet/pair:0 use. Recorded final-source check remains lease679043877/session91224, then inventory and close/land.

2026-10-24 — Contract shadow sweep found old attachment-only startup prose in README and the embedded Couch SKILL.md, not just atlas. Updated both to proven fresh startup versus known shutdown/unknown; README uses inline review markers. This mirrors the native-qualified behavior, adding no automation permission. Read the writing-skills reference; this is an existing binary-contract reference correction, not a new discipline/workflow skill. Verified old blanket assertion absent and new fresh/off/no-input distinction present. Because the embedded skill ships with the binary, cancelled queued lease679043877 before running tests and will record the final head once; #842 remains retained.

2026-10-24 — Final recorded pre-close lease1020585962 / attemptdebe789b4aeddd5b5d7855568d635b33 onfd333cd67b3c078855c4141f2210c069618fdc89 remains queued (session18587; checkout /Volumes/TestScratch/local-checks/sdlc-local-2846087787). Replaced prior queued lease679043877 after embedded skill mirror changed; both superseded parent runners joined with context-cancelled queue result, no test executed. Fresh worker help explicitly permits close review overlapping a queued check if final-head pass precedes landing and any code fix requeues. Using that permission; no suite pass claimed. Product implementation and native proof complete; no-plan-check acknowledges only the final self-referential close/merge procedure checkbox. TL owns project updates. After SHIP, wait recorded pass before merge; if review changes runtime, cancel/requeue only this issue's exact lease. Keep #842 lease3238290542/session37701.

2026-10-24 — Close stopped at actual measurement before review (session9931 exit1). Exact refusal: branch attribution for068eb706-6618-4e56-a4bd-be215a560ce4 (/Users/xianxu/workspace/worktree/pair-slot7/pair): concurrent claims have unknown, resting or ambiguous active branch during2026-10-24T07:44:22Z →07:44:38Z. Binary names close recovery --no-actual with evidence in --verified and says ask TL to restore named evidence. Reported to hub; no invented attribution, bypass, review or close success. Current final recorded check lease1020585962/session18587 and separate #842 lease3238290542/session37701 remain queued. No model restart. Next: hub-directed actual recovery/exclusion, then inventory and close review overlapping the retained check; no landing before recorded pass.


### 2026-10-24 — close REWORK and third-delta re-plan

Close at 99846a5015..68bc00e8 returned REWORK: Critical BR-1 automatic-launch-requires-empty-authority and Important BR-2 native-acceptance-evidence. Accepted BR-1: unenrolled configured local history is invisible to the namespace snapshot and public StartFreshSlot permits replacement. Read-only audit also finds general primary Open can cold-resume history appearing after observation. No production code edits after REWORK. Hub39d48cc4 approves direction only; the consolidated appended plan Revision is sent for approval before change-code/implementation. Previous Task2 held fresh coverage and overall completion claims are superseded, not acceptance evidence.

BR-2 is unclassified: the reviewer passed FreshHubCaller, EditedRestart and dirty/force, but StartupArguments/claude failed after initial startup in switch-agent adoption (attach terminal input has ended; terminal listing omitted a live server), then skipped Codex. The re-plan names bounded pinned-main/branch classification with retained diagnostics. No claim this is a branch defect, flake or the earlier Zellij metadata panic. Review/gate sidecars are retained.

Cancelled only queued #863 lease1020585962 / attemptdebe789b4aeddd5b5d7855568d635b33 through local-check cancel; parent18587 joined context-cancelled without executing tests. Separate #842 runner37701 was preserved and finished: sole deterministic failure is artifact inventory omitting couchtty/console_send.go, log cfb07485d03d337d82c60275ecc8c0a9, sha256 prefix dc9fb4d0b9b4; other packages completed. Reported separately, no passing suite claimed. Next: await consolidated plan approval; audit/fix #842 inventory independently on its own branch while waiting.


### 2026-10-24 — approved re-plan, gate and first safety correction

Hub9dfc3c84 approved consolidated scope. change-code full returned exit0: plan-quality CLEAN high (round4), estimate-quality INFO only (original1.333h omits added guarded-store work, renewed review and expanded native qualification; retained as original estimate, not claimed current scope forecast). Read-only lock audit confirmed coordinator -> global -> local, matching EnrollSlotRepository; no local-to-global inversion or new lock. Existing primary AllocateThreadTag only proves address uniqueness, so empty-only primary allocation now checks history inside the existing locked publication and bypasses Open/resume.

Red observation test reproduced unenrolled local current/archive classified fresh. Red stopped-current test showed StartFreshHub actually archived/replaced and launched. New direct guard test initially failed missing API. Implemented metadata-only bounded history observation, configured stores independent enrollment, guarded global/local commits, lock-assuming extraction of existing journal writers, numbered requireEmpty startup mode and ordinary create-only allocation. Pending journals are refused before existing lock code can recover them; absent-store preview captures physical host witnesses without initializing storage.

Targeted core/cmd/tty startup/fresh/Fresh-slot checks PASS15.825s/0.756s/0.309s (tty pattern matched none); focused new safety cohort PASS6.840s. Explicit operator Fresh replacement control remains green. Appearance injection uses the real artifact-claim boundary after resolution, not a fabricated callback: primary current/archive/legacy and numbered current/archive/legacy arrivals refuse without effects or changed old bytes. Temporarily bypassing the empty guard makes five schedules launch and fail the oracle (numbered current additionally has existing exact-byte CAS); source restored in finally. This negative control is not a passing production result.

No native acceptance or suite pass yet. Real fresh Console.Run integration tests are in progress as a bounded delegated test-only task; BR-2 still needs classified native evidence. #842 inventory failure remains separate. Next: complete held orchestration tests, native BR-2 classification and corrected matrix, then inventory/pre-close/re-close.


### 2026-10-24 — controlled Run qualification and native comparison setup

New actual Console.Run tests exercise production startFreshPrimary/beginConsole/startExplicitHubs, real core/store/dispatcher, catalog-backed targets and stateful process holds. Launch/adoption are held separately; cancellation joins each phase; empty results attach nothing; later live inventory cannot enable original fresh automation. Plain cohort PASS0.636s. One initial race observation waited for a missing third adoption with workers idle; SIGQUIT digest retained as fresh-run-race-summary.txt. Cause remains unclassified, not asserted fixture-only or fixed. Test diagnostics now fail on actual dispatcher error and await the final attach-completed marker as well as Hosted publication. Bounded count10 race diagnostic PASS33.142s; full log fresh-run-race-count10.log. No full message-service bootstrap is claimed by these tests.

The same history enumeration also includes local recovery/*.json raw conversation backups. An isolated recovery-only unit test failed first, then passed with the shared bounded nofollow flat-directory traversal. Other raw record families are current, archive and journal images (pending journal already refuses); continuation/grace/preferences are derived prose/metadata. Updated atlas, README inline addition and embedded skill; new Couch-only storage files classified NonArtifactSources after direct input audit. Artifact inventory targeted check precedes completion claims.

Hub79aa31fa approves main-baseline explicit operator startup gesture, then unchanged switch-agent steps; main otherwise cannot reach that stage because #863 fixes its fresh bootstrap. Branch retains automatic-fresh qualification. Added the matching shared Fleet.StartCold seam (same #830 name/semantics), preserving Start's wait behavior. For comparison, compile the native test driver on this issue branch, then temporarily detach this SAME slot checkout to pinned main for BuildOwned candidate source and regenerate that source's runtime bundle; run the compiled driver with baseline source explicit, join cleanup, restore issue branch and regenerate its bundle. This avoids a second checkout or unlanded production stack. Log both actual Git pins and the baseline-only gesture. No timeout increase or blind repeat is authorized.


### 2026-10-24 — first bounded native baseline result

Compiled native consumer at branch5d5a6014cd4db610a31b2d3ab5db9d32ea5c1649, detached this same checkout to Git-verified main4053f3da995746efcc1c6296cffbde4a67a7f343, regenerated main's runtime bundle and used BuildOwned with that exact source. Main baseline used the hub-approved explicit operator start gesture only; switch-agent steps are unchanged. TestDisposableStartupArguments FAIL42.33s, Claude34.11s; Codex skipped by existing break-on-failure. It reached hub startup, then StartWorker1 failed before switch-agent: Pair exited1 before registering. Retained server log explicitly contains the known Zellij0.45.1 lib.rs1462:26 absent-session-data unwrap panic. Exact owned generations absent and direct child/IO joined. Full capture main-startup-arguments-4053f3da.log retained; this is a different failure from BR-2's post-startup adoption report, not evidence of matching cause. No main rerun.

Restored issue branch5d5a6014c and regenerated its runtime bundle. Next is the one corrected-code required matrix; it must either pass both personalities or produce classifiable evidence before any repeat. The old BR-2 occurrence remains unclassified until passing acceptance or independent matching evidence satisfies its verification requirement. No change to scenario deadline or live fleet.


### 2026-10-24 — corrected native matrix finished; two failures, no repeat

Corrected source7a71f335350082c048aca4c84058cd350c6d0eb8, compiled consumer5d5a6014c (runtime-identical, later head adds evidence only). Required native matrix completed all four top-level tests, exit1; full capture branch-required-native-7a71f335.log, SHA256217ecb09534658cc33ca98bdc08f71ff87141e1305d0ca28b8ba42af424f0f11.

- FreshHubCaller PASS46.96s: Claude14.34s, Codex20.08s; both hubs live callers, automation off, no automatic input, exact cleanup.
- StartupArguments FAIL58.64s: Claude52.31s. First switch Claude->Codex succeeded with exact literal argv; second switch to Claude failed Pair exit1. Its server log contains lib.rs1462:26 unwrap panic at02:49:13 PDT, the same panic as pinned main's initial-worker failure. Codex-starting cohort skipped by existing break. This occurrence is #866; it is NOT a passing required matrix or proof that the older reviewer occurrence had the same cause.
- Dirty/force PASS68.20s: Claude30.84s, Codex32.98s, existing dirty file preservation and confirmations held.
- EditedRestart FAIL80.69s: Claude PASS51.53s, Codex FAIL22.08s. Codex switch-agent was refused 'source must be known idle; retry after its turn finishes' despite the existing startupArgumentsWaitIdle precondition. This second failure remains unclassified; no assertion of branch causation, fixture-only cause, flake, or relation to #866. Exact owner cleanup passed every fleet.

At TL8029db79's direction filed unclaimed pair#866 now, not after landing: workshop/issues/000866-zellij-startup-probe-readiness.md contains both initial panic captures and source pointer; creation published on main, no dependency. Third panic capture above is available for its owner. No fourth native attempt or deadline increase. Report both native failures to TL and pause further qualification pending direction; deterministic safety implementation remains committed and passing. BR-2 verification requirement remains open.


### 2026-10-24 — bounded Codex idle investigation authorized

TL1233e4d0 directs retained-run inspection then ONE EditedRestart/Codex run on pinned main with explicit-gesture baseline. The retained failure lacks the successful resource sample JSON, so its precise sampled timestamp/settled value cannot be reconstructed. The executed helper guarantees its predicate returned true: one exact scope/tag row, IdleKnown && Idle, a resource snapshot newer than helper entry, and unchanged native PID/generation. Production resources computes Idle from the liveness Settled pointer plus broker/picker state at sampling; switch admission re-reads the current liveness Settled pointer. Refusal means it was nil/false then. Thus the facts establish a sample-to-admission disagreement, not its cause or branch attribution.

Added content-free diagnostic logging at the successful precondition (resource sample/observation timestamps, exact binding/native generation and rows), without changing that predicate. Reused the same explicit baseline candidate/gesture helper for EditedRestart; production and scenario deadlines unchanged. Both changed native readers remain non_go_full_fallback after individual input audits. Main comparison will select only the Codex subtest; no Claude repeat. If main reproduces, file independent issue with both captures; if branch-only, investigate in #863. StartupArguments second-switch panic is recorded as #866.


### 2026-10-24 — pinned-main Codex idle comparison passed

The one TL-authorized main comparison used main4053f3da995746efcc1c6296cffbde4a67a7f343 and consumer c9dbf637d18c869d1ecae545201b2e5d1ab755b2, explicit main-only operator gesture. EditedRestart/Codex PASS48.85s (case39.22s), resume and positive-absence fallback both succeeded, old transcript retained, exact cleanup passed. Successful idle samples logged. Capture main-edited-restart-codex-4053f3da.log SHA256f13c712825482e0ea0216f679b2d2fdf41ce4cf13833c56be0e93c00750eddae. Restored issue branch and its runtime bundle. No independent idle issue filed: there is no matching main failure. A single main pass is not proof of introduced causation; the disagreement stays #863's to diagnose.

Read-only audit finds no changed settling, fixture painting, resource projection, cue producer or switch admission implementation. Resources samples a Settled value, not a reservation. Two plausible unchanged transitions are delayed worker cue delivery and asynchronous transcript task_started/task_complete consumption after the fixture accepts the seed. Fresh launch timing may expose them, but the retained branch capture lacks the transition. Also auditing chrome resize/progress as an output trigger; no conclusion or production fix yet.

Prepared test-only failure diagnostics: fresh resources/peek, accepted input timestamps/kinds/hashes and bounded private wrapper event tails (256KiB total,64KiB/file,4096 entries) before owner cleanup; source mutations/retries are not added. This makes a further diagnostic capture reviewable, but it is not run without TL direction. No production idle guard or deadline changed.


### 2026-10-24 — one authorized instrumented branch Codex run: not reproduced

TLc6927449 authorized exactly one branch EditedRestart/Codex run with test-only failure capture. Source eb2922b2206443b0fee6dd379ea4d084630fe9d8, command PAIR_NATIVE_COUCH_FLEET=1 go test ./cmd/couch -run '^TestDisposableEditedRestart$/^codex$' -count=1 -v -timeout=12m. PASS99.20s (case82.31s; go package99.417s). Resume and positive-absence fallback succeeded, prior transcript preserved, exact owned generations absent and child/IO joined. Successful idle samples logged at03:08:03.360499 and03:08:16.357717 PDT; current operation receipts succeeded. Capture branch-codex-idle-instrumented-eb2922b2.log retained.

The prior Codex idle disagreement was NOT REPRODUCED. No causation claim, no production idle fix, no classification as unrelated or known flake, and no further native run. Read-only audit confirms settling/admission/fixture/cue implementations are unchanged; timing and deferred lifecycle/cue hypotheses remain unproven. Sent evidence to TL before any close decision as required. BR-2 disposition awaits that decision; StartupArguments second-switch panic remains #866. Next independent authorized work is #842's audited artifact inventory repair on its own branch, with no #863 checkout/runner left active.
