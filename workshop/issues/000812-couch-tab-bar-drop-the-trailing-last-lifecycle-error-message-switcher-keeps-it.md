---
id: 000812
status: codecomplete
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-23
estimate_hours: 1.7435
collaboration-mode: collaboration
card_mirror: 'c1f7ffef01d3b793dd0ee49184b88a27a94ff160' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T05:01:56-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 3.85
---

# Couch tab bar: drop the trailing last lifecycle/error message (switcher keeps it)

## Problem

## Spec

Operator request (10-22, screenshot in ops `assets/session/2026-10-22.13-05-53.800.jpg`): the Couch tab bar's second
row shows the last lifecycle or error message after the slot tabs (e.g. `· pair [couch-dd48472d] exited (1)`). The
operator doesn't find it useful there and wants it removed from the tab bar. It stays in the switcher. Anything that
needs the operator's attention goes through the notification (attention list / `couch --notify`), not the tab bar.

- Remove the trailing last-message text from the tab bar (both one-row and wrapped two-row layouts); the space goes
  back to slot tabs.
- The switcher keeps showing it, unchanged.
- If any path relied on the tab-bar text as its only operator-visible signal, route it to the attention list instead
  and say which.

## Done when

- A tab-bar render test with a pending lifecycle/error message shows no trailing message text; the switcher test
  still shows it.
- README/atlas wording updated.
- Actionable asynchronous failures from the producer audit appear in existing attention; routine progress/success/key feedback does not create obligations. Existing prompt/question/hub-notify/backup attention is not duplicated.
- Global or unavailable-thread attention events open the switcher rather than redirecting to an unrelated focused slot; exact attached-thread selection remains exact. TL approved this navigation detail (receipt db842971-53f5-4660-a836-870b0d14de1a).

## Plan

[Durable implementation plan](../plans/000812-tab-bar-notice-plan.md). TL-approved full-flow plan; enter change-code before implementation.

- [x] Remove general feed notice from both tab-bar styles; retain latest full notice for switcher display and Ctrl+Y with menu precedence/geometry tests.
- [x] Route the audited asynchronous actionable notice classes through existing attention; keep progress/success/key feedback feed-only, and qualify exact/global selection and bounded lifetime.
- [ ] Update README/atlas, run targeted/race/inventory and recorded checks, obtain coordinated live acceptance, then close/review/land.


## Operator check (after TL stage go)

Original checklist below is superseded by the TL live-evidence allocation in Revisions; exit and detached-target cases use deterministic tests.

1. Open the staged disposable Couch in collapse, then multiline mode; confirm only controls and slot chips use the tab bar, with no trailing message.
2. Trigger the prepared disposable exit/error; open the bell and confirm one readable warning. Normal progress and success should add none.
3. Open the switcher and press Ctrl+Y; confirm the retained message is visible and its full text copies, while operation and copy feedback keep precedence.
4. Select exact-thread, global and detached-target warning examples; confirm the attached thread opens exactly, the others open the switcher, and dismiss/clear works.

## Log

### 2026-10-22


### 2026-10-23 — claimed and planned from main
- 2026-10-23: closed — Post-SHIP one-literal correction from recorded full-suite failure: reuse existing continuation notice Kind instead of invented continuation-scan, which matched artifact filename vocabulary. Actual scan episode/source/body unchanged; artifactpath classification PASS2.923s, warning episode/publication race x10 PASS3.347s, make test-affected-inventory and diff check PASS. Prior recorded full run on d2942011 completed all remaining targets, sole deterministic artifact failure, not a flake; fresh recorded affected/full submitted on pushed 5eb99587cf9010715c99837953d7b81cd4d2eae4 and must pass before landing. Existing TL authorization closes before held stage/live; only pending composite plan row waived, coordinated operator check still required. (fixed-then-ship); review verdict: SHIP
- 2026-10-23: closed — BR-6 full async assertion class sweep in durable plan: exact exit warning postcondition and controlled assertion-first/publication-first, stop negatives join terminal callbacks and feedback. Exact unlock-to-Warn gated overlay RED0.397s then GREEN x10 0.565s; comprehensive classification/source/ordering race x10 PASS113.174s. Complete couchtty PASS57.447s (4362 test/subtest passes), couchcmd PASS113.354s (1784 passes),0 failures; final extra test-only stop join named race x10 PASS3.856s. Production unchanged since cafbf526. make test-affected-inventory and diff check PASS. Fresh recorded affected check submitted on pushed d29420112ffe74bae7e2b7bb443c93a7a5dcbf8f; pass required before landing. TL1e4df14f authorizes close before held stage/live: only unchecked composite plan row waived, not real-console acceptance or explicit stage go.; review verdict: SHIP

Authenticated ops:0 dispatcha5d6fa63-e13d-47f0-a2c9-03f5643533b7 assigns #812 after #791 landed. Claimed pair:1 (claim waited for a peer close's legitimate repository lock); start-plan created this independent branch from mainc0dbb32decc052500172ded01ff36e3fa8e49cc9. Catch-up refreshed the local ariadne dependency toee4dd7fa1e8488e3230dc462971be8796f317e41; seeded workflow refresh was explicitly deferred by the new tooling, no local-gates edit to ship.

Read-only producer audit delegated to812_notice_audit, no code changes. The current feed's latest text reaches StatusModel.Notice through statusModelLocked and both layouts; CapabilityNotice is a separate live control and stays. Contrary to the initial “switcher keeps it” assumption, the general feed is currently only available through Ctrl+Y; renderedMenuNotice draws MenuState.Notice instead. The plan explicitly adds a bounded latest-feed switcher projection without replacing operation errors/progress/copy feedback. ARCH-DRY keeps Feed as history owner; ARCH-PURPOSE covers asynchronous warnings before deleting their tab visibility.

Audit identified unexpected exits, crash/trace failures, hub/cleanup/stale/supervision warnings, continuation failures and async cast/guessing-lockout failures without existing attention. Blanket routing all setNotice calls would promote progress and harmless feedback to false obligations. Recommended explicit producer classification reuses AttentionNotify's cap50/select/dismiss/12-hour expiry. Subjectless events are global Couch items; proposed selection opens switcher, never attributes them to the focused actor. The durable plan records this visible-state decision for TL approval. Existing notify/prompt/question/archive routing remains single-source. No new core enum, persisted format, worker, queue or native-input effect proposed. No estimate or change-code run yet.

Next: send committed plan to TL and await approval, especially the global-selection choice; then run change-code and its plan-quality gate before deriving any estimate or implementing. No live stage or operator-slot action authorized.


### 2026-10-23 — plan approved; enter implementation gate

Authenticated TLdb842971-53f5-4660-a836-870b0d14de1a approves renderer/collapse/wrap removal, latest-feed switcher display with operation/copy precedence, explicit actionable async attention routing, and global/detached-target selection opening the switcher. Extra noise guard: routinely firing normal-operation classes do not belong in attention. Durable plan updated with that constraint; no production changes yet. Next run change-code full flow, answer plan-quality findings and derive estimate only after it clears. Use executing-plans in this warm session and delegate isolated test/producer tasks where context is capturable. Stage/live acceptance still requires a later TL go.



- Plan-quality round1 REWORK (PQ-1, function-level-test-strategy): codebase claims and approved architecture verified, but tests were written as case inventories and the new projection/publication functions were unnamed. Swept the test plan into named risky-function strategies with fuzz/generated and controlled mutex/channel interleaving proofs; no behavior or scope change. No source/test implementation yet. Re-run change-code before deriving estimate.



- Plan-quality round2 CLEAN; PQ-1 addressed, no further findings. Estimate derived only after this result, from v3.1/v2.1/v2 primitive tables and baseline-v3.1 (calibration marked stale/provisional). Existing Feed/AttentionList/renderers cover the work; no greenfield library or external API needed. Small module projection: design0.15×0.2=0.03/impl0.50×0.4=0.20; TUI attention and navigation: design1.0×0.2=0.20/impl1.0×0.4=0.40; cross-cutting producer routing: design0.8×0.2=0.16/impl0.5×0.4=0.20; docs: design0.1×0.2=0.02/impl0.15×0.4=0.06; review/verification: design0.1×0.2=0.02/impl0.5×0.4=0.20; coordinated UX iteration: design0.3×0.2=0.06/impl0.3×0.4=0.12. Design0.49×1.15 + implementation1.18×familiarity1.0 =1.7435h. Exact collapse function anchor corrected before gate rerun.


### 2026-10-23 — implementation checkpoint

change-code full-flow completed after round3 plan-quality CLEAN and estimate-quality INFO. Verification work is included in the projection/TUI/routing primitives plus boundary review; ux-rename-iteration is the coordinated acceptance-feedback round, not a proposed rename. No actual hours hand-entered.

Implemented tab-model/common-render removal and derived latest retained Feed in both switcher snapshot/publication paths; operation/refresh and independent clipboard feedback keep precedence. Explicit Warn source routing reuses AttentionNotify with a bounded pure projection, exact optional source, console-local IDs, c.mu → attention mutex, repaint after unlocking. Global/detached selection opens the root switcher directly (a second Ctrl+Space would instead open Start). Routine lifecycle delivery diagnostics, scratch opt-out/fallback, normal cast stops, manual code rotations and key feedback remain feed-only. Archive pass/retention failures use a distinct actionable callback; existing archive notifications retain their single attention owner. Continuation scan failure retains one bounded latest-error episode, cleared only by a healthy scan. No new queue, worker, storage or native input.

Behavioral RED evidence: render/feed/switcher regressions; unexpected exit (zero warnings), global navigation (actor left focused), continuation scan (zero warnings), cast start failures (zero warnings), composition-root crash/hub/supervision/cleanup producers. Initial waiting-fixture runs were interrupted and are not passes. Projection failure bounds and control safety subsequently fuzzed for 11,380 executions, PASS 6.425s. Counted40/80 warnings retain cap50 and expire via existing12h lifetime; channel-controlled publication/selection orders match a sequential AttentionList model; expected exit/shutdown negative controls pass. Added geometry test caught a new warning relayout using raw screen rows instead of existing attentionScreen geometry; corrected to the same existing formula (RED→GREEN). Focused final core tests PASS0.551s and race PASS2.294s. Source tests cover all three trace failures; secret-safe guessing lockouts plus stale/duplicate/manual controls; real screen/Chrome failure vs normal stop; recovery episode controls. Delegated targeted source race PASS1.943s; render broader PASS2.184s/race1.821s; couchcmd broader PASS6.011s/race3.109s. Complete couchtty/couchcmd run still pending; none of these targets is a recorded suite pass.

README and atlas updated with executable anchors and removed contradictory trailing-row prose. Inventory initially named only console_notice_expiry_test.go: audited its inputs (all os.ReadFile/Stat operands are generated clipboard/call markers inside t.TempDir; PATH tools are literal test-created shell fixtures; no repository/external input). Recorded only that reader as local_or_literal_fixture; inventory recheck PASS. No bulk refresh.

Next: finish complete package checks, repair any failures, push this candidate and run recorded affected check on its exact head. Coordinated real-console stage/live acceptance still pending TL go; no :0 action, close or landing yet.

### 2026-10-23 — complete-target failure classified; stale oracles repaired

The manual complete couchtty/couchcmd target used a5m package timeout, shorter than the repository's declared20m. It is NOT a pass: obsolete TestCodesMenuRotatesAndDisconnects tab-row expectation consumed283.44s; TestGuessRotationAlertAndFlash and TestConsoleMenuResumeLandsOnExactReturnedHandle also assumed removed chrome text. Updated only those three oracles to retained feed/switcher and explicit attention, preserving private-flash/generation, viewer disconnection, code safety and exact-returned-handle assertions. Focused repair run PASS0.783s; inventory PASS with no additional certificates.

Remaining Drive/ctrl-revoke teardown and later timer timeout happened with the package deadline nearly exhausted. Unchanged isolated TestDriveActionsDoNotBlockTerminal PASS0.575s; TestCouchRestartReplyAdmission (where couchcmd hit its5m ceiling mid query) PASS1.037s. testwait derives all windows from the package deadline; these are consequences of the consumed/undersized global budget, not an established same-head flake. No blind full passing retry or flake registration used. Final recorded affected check will run complete selected packages with the declared20m budget and retain its log. Core warning race after geometry correction PASS2.003s.

PR558 is open as an early candidate; close/review and real-console acceptance remain outstanding. Next push the oracle correction, then run the recorded affected check on that head. No live :0 stage yet.

### 2026-10-23 — recorded check queued; close sequencing approved

Recorded affected selection on afee7fb0eea8bb6c3156567d1ca744e596396859 escalated to full suite because the audited reader certificate changed. Owned run lease-3535874828, inflight11fd085bbff5b4739e288a00f96e6eb5, retained log under the common Git sdlc/test-stats/inflight directory. Natural FIFO position8 when admitted to the queue; keep this run, no cancellation/requeue or restart. PR558 updated. Final four-test stale-oracle/drive target PASS2.965s.

Authenticated TL1e4df14f-c092-4148-8ad7-6921cd401d30 says pair:0 holds #811's morning check and #812 stages next only on TL go; meanwhile finish recorded check and local close. Four short operator steps added above. Local close/review can precede live acceptance under this explicit direction; passing recorded proof and real-console acceptance still precede landing. The remaining third Plan row therefore stays unchecked: it includes pending live acceptance and landing. If close requires it, acknowledge only that plan-check guard with this exact sequencing reason; never claim those actions passed.

### 2026-10-23 — boundary round1 REWORK; source-class correction

Binary-owned full review returned REWORK with BR-1 Critical actionable-producer-coverage (local queue finishLifecycleAttempt still used Notify; direct-dispatch callback correctly skips queue-owned attempts) and BR-2 Critical attention-requires-actionable-source (generic Chrome Logf includes healthy fidelity summaries). No close finalized. The independent bounded source audit also confirmed recovered overflow/resync, aggregate Close loss summaries and transient drive503 diagnostics are informational, and uncertain remote input has an existing changed-Recovery attention owner. The original producer-routing checkbox is qualified until these whole-class corrections pass; do not treat earlier targets as proof of these missing cases.

Cancelled the owned old-head queued suite with sdlc local-check cancel --run lease-3535874828, confirmed terminal cancellation; executable source correction needs a new exact-head run. It had not started its suite; no passing result or retained FIFO place claimed. No restart occurred. Typed callback correction uses optional Warnf alongside Logf through CastConfig/RecordOptions/recorder.Options, legacy fallback instead of double publication, with explicit actionable source sites. No new queue, lifecycle/status or persisted state. Local/remote/direct terminal ownership and typed successful-result warning fragments are being swept separately; routine recovery/fallback strings remain feedback.

### 2026-10-23 — BR-1/BR-2 class fixes verified

BR-1: real queued local completion regression reproduced both cleanup and diagnostic-persistence missing alerts, then passed with exact origin and terminal once ownership. Swept typed successful-result warnings: StartedChild adoption/config publication; Archive/Reboot session-left-running; ParkRecovery diagnostic; switch cleanup. A single transient non-serialized SwitchAgentResult.CleanupWarning preserves the producer fact separately from ordinary recovery Notice text; a real successful switch with post-park cleanup failure proves it survives. Full generic feedback still publishes once, and attention only when one explicit typed actionable fact exists. Routine recovery/fallback/NextCommand and even error-looking untyped strings raise none. No duplicate ParkResult diagnostic owner. Targeted lifecycle race PASS1.539s; switch/edited-restart race PASS24.214s; main combined warning source target PASS0.559s and real switch cleanup PASS0.313s.

BR-2: actual recorded-cast end with a frame reproduced healthy fidelity falsely raising1 attention (RED); healthy summary now raises0, failed fidelity exactly1, while both retain feedback. Optional typed Warnf falls back to Logf for legacy callers rather than invoking both; console wires informational and actionable paths separately. Recorder write/open/snapshot/segment and real sink.Close failure sites warn; recovered resync and aggregate Close loss account remain informational. Remote-input uncertainty retains changed Recovery as its sole attention owner. Drive classification follows drive.Answer's typed response:503 stays informational,502/500 warn, preserving ErrInput precedence even when wrapping a deadline error (separate RED→GREEN). Recorder targeted PASS0.465s/full-package race11.021s; actual cast/drive source target race PASS5.553s; console recording/warning-source race PASS4.040s. Inventory and diff check PASS without additional reader repairs. Lessons and atlas updated for both finding families.

Next: commit/push the correction, run a fresh recorded affected/full check on its exact head, and re-run full close review for explicit BR-1/BR-2 disposition. Live operator checklist still waits for TL stage go after #811; no landing before acceptance.

### 2026-10-23 — boundary round2 REWORK; complete source and oracle sweep

Round2 verified BR-1 and BR-2 addressed by independent reversal tests, but raised BR-3 Critical (second actionable-producer-coverage instance: unrecoverable low-priority-agent background relaunch instruction) and BR-4 Important (remaining removed-tab oracles; success fixture with legitimate borrowed ownership). No close finalized. Cancelled owned queued lease-1373756955 through SDLC and observed terminal cancellation; source correction needs another exact-head run. No suite pass claimed.

The broader manual couchtty run was quiet because an additional stale oracle waited for inactive-exit text in chrome. Captured the owned manual test stack with SIGQUIT (interrupted, not pass), then repeated with JSON progress to identify TestInactiveChildExitKeepsFocusAndRecordsNotice precisely; stopped that owned manual test after diagnosis. The JSON log is /Volumes/TestScratch/tmp/pair-slot1/pair812-couchtty.NsTHdj. Current repaired complete-package run uses JSON progress rather than hiding an obsolete wait until the deadline. Parser-isolation/publication/resize/focus oracles were migrated to actual warning badges, retained feed and switcher, with committed-state channel sequencing instead of incidental sleeps. Focused RED2.746s, GREEN0.614s, race1.779s; inventory PASS.

The durable plan now records the full console source enumeration under the general rule, rather than another isolated callback fix. TL3a019860-25e1-4345-8318-72004b300014 explicitly keeps layout-caused pointer-watch stop feed-only as routine safety; failed accepted async capability/drive operations warn. Composition-root callback enumeration and fixes are continuing in the same round. Qualification remains pending complete-package proof, recorded check and clean close.

### 2026-10-23 — remaining class enumeration and whole-console proof

The durable plan now enumerates all remaining couchtty and couchcmd producer branches under the shared source rule, including mixed lifecycle/scratch callbacks, renicer and absent-hub archive fallback. Automatic scratch fallback/retained-entry cleanup stays feed-only because coding remains usable and automatic retry owns recovery; typed metadata setup errors warn. Missing/disconnected archive hub must still create one global event through its existing notification owner. No English severity inference or blanket mixed callback promotion.

TLd40218f2-3381-4372-b703-fbbf844e9c74 approved exactly one bool under existing lifecycle publisher mutex at first128-hub overflow, at most one capacity warning per publisher lifetime; no other new state. This prevents the capacity source from becoming a routine notification storm. Pointer-watch classification remains the earlier explicit feed-only TL decision.

Complete JSON couchtty verification then found TestCouchProductionSoak's hardcoded right-edge chrome gesture opened the newly legitimate attention badge after an intentional attachment replacement. Isolated RED25.269s stopped at receipt7234 (r4, first iteration after replacement). Changed only that test's neutral press to its fixed fixture's midpoint, with assertions excluding all interactive chip/control/badge/group spans; all original receipt, mouse-origin, resize, focus and reattachment assertions remain. Narrow GREEN0.513s/race2.352s. Complete couchtty package now PASS57.651s (4360 passing test/subtest events), retained JSON /Volumes/TestScratch/tmp/pair-slot1/pair812-couchtty.MLp9aX, with write budget40091648/67108864. Earlier interrupted manual probes and cancelled queued suites remain non-passes.

The accepted async capability/drive source correction has RED→GREEN and race14.835s including current/stale generation/cast, ordinary typed races/cancel/deadline, success/copy/retry and nonblocking controls. Pointer-watch negative regression confirms feedback/phase change without attention. Await final composition-root package proof and inventory before committing/requeuing/round3.

### 2026-10-23 — complete source/oracle sweep verified before round3

BR-3 class sweep is complete against the durable plan's console/composition tables: exact-binding renicer relaunch warnings; typed lifecycle publication/metadata/capacity failures; startup-only metadata configuration errors; and one global archive event when its hub is absent/disconnected. Routine callback branches remain feedback. TL-approved capacity latch observed RED100 warnings across100 drained overflow passes with128 roots, then GREEN1 warning/128 roots. Accepted async capability/drive failure and pointer safety controls are covered. Only the approved capacityWarned suppression bool is added; no new queue/lock/condition owner or persistence.

BR-4 complete package verification exposed one further edited-restart acceptance oracle expecting a full joined warning in raw terminal bytes. It now observes an actual VT-emulated stable visible banner prefix and copies byte-for-byte full receipt.Warning through real Ctrl+Y/OSC52, with native clipboard tools hidden only after restart completion and PATH restored before cleanup. All12 cases across Claude/Codex and resume/fresh/publication outcomes passed twice (4.789s); focused race5.414s. Whole couchcmd PASS66.169s,1780 test/subtest passes plus package pass,0 failures; retained JSON /Volumes/TestScratch/tmp/pair812-couchcmd-round3-verified.LEAoXW/run.jsonl. Renicer exact-target navigation strengthening then passed race1.548s; no production changed after whole-package compilation. Whole couchtty PASS57.651s,4360 test/subtest passes plus package pass as above. Existing targeted source races also pass (2.800s final composition group,14.835s async-control group,2.352s repaired native soak).

Inventory named only renicer_test.go after these edits. Owner audited its entire source: literal typed fake process tables/bindings; native test launches its own literal sleep/wait child and observes that process; benchmark uses native process table; no repository/non-Go file input edge changed. Recorded only that certificate as local_or_literal_fixture and rechecked inventory PASS; diff check PASS. No bulk refresh.

Next: commit/push this tested class correction; submit a fresh recorded affected/full run on its exact head; re-run full close for explicit BR-3/BR-4 disposition. No recorded check currently runs: both earlier queued old-head runs were explicitly cancelled for subsequent executable corrections, not passing retries. Local close may precede live acceptance per TL, but recorded proof and coordinated real-console check still precede landing. Stage remains held for #811; await explicit TL go.


### 2026-10-23 — round3 verifies sources; async test ordering blocks

Round3 reviewed cafbf526b5ca0ecfbd4b75518de7ac3758585ebf: BR-3 and BR-4 addressed, independently reversed; complete couchtty PASS54.4s and focused five-package/race checks pass. REWORK BR-5 (Important, regression-oracles-follow-contract): lifecycle_diagnostic_attention_test.go queues a diagnostic then opens attention before publication; the helper toggles an empty list, which stays closed after later publication. Reviewer complete couchcmd stalled470.7s at this test and was interrupted, not passed. Our earlier whole-package pass does not refute the controlled ordering gap. ARCH-ORDER correction: require publication acknowledgment in the shared helper, enumerate every caller, and force both publication/input orders without production changes. Bounded test repair delegated; complete couchcmd rerun follows.

Recorded suite lease-2349723115 on cafbf526 never started (FIFO position6 after697s); cancelled with sdlc local-check cancel and runner terminal context-canceled confirmed. No suite pass or retained queue position claimed. Fresh run must use the corrected pushed head. Close measured2.99h, window2026-10-23T05:01:56-07:00 to06:16:43-07:00,7 admitted sessions,1 ownership interval; code cafbf526b5ca0ecfbd4b75518de7ac3758585ebf, tracker51698e2f7f3866a4baad5443693d03b62a3d5c1e. Close did not finalize. Stage/live acceptance remain pending TL go after811.


### 2026-10-23 — BR-5 controlled orderings and complete package pass

The shared openWarningAttention helper now acknowledges the current VT-rendered badge before input; historical accumulated bytes cannot satisfy publication. The queued lifecycle diagnostic also joins its actual Warn completion via a channel. Controlled RED helper-first fails with “attention input preceded warning publication acknowledgment” (0.381s). Three GREEN cases force publication-first, helper-first and dismissed historical badge before fresh publication, without sleeps or production seams.

Audited all ten original call sites (correcting the initial nine count): crash installation; explicit hub failure; supervision; captured lifecycle queue completion; archive tick; unhealthy renicer; normal-priority renicer's independent warning on a separate console; scratch metadata; common later-hub-failure helper under both warm-first orders; queued lifecycle diagnostic. Synchronous sources return after Warn; asynchronous hub sources retain existing badge joins plus shared current-screen acknowledgment; queued diagnostic now has its exact callback join.

All callers and controlled orderings x10 PASS17.231s; all callers and three orderings race x10 PASS28.732s. Complete couchcmd JSON count1/timeout20m PASS88.520s,1784 test/subtest passes,14 skips,0 failures; agent-owned session70765 terminal0, retained /Volumes/TestScratch/tmp/pair-slot1/pair812-couchcmd-br5.SThf9P/run.jsonl. make test-affected-inventory PASS (no certificate change), git diff --check PASS. Two test files only; production unchanged from cafbf526. Next commit/push, fresh recorded affected/full on exact corrected head, then full close for BR-5 disposition. Recorded proof, TL stage go and coordinated real-console acceptance still precede landing.


### 2026-10-23 — round4 requires full async assertion audit

Round4 on58b2e43e6af14e82aa9413c7a960c46d6b44a0ed verifies BR-1 throughBR-5 addressed, including BR-5 reversal. Reviewer complete couchtty42.096s/couchcmd72.989s/recorder4.692s/webcast12.164s PASS. REWORK BR-6 Important: unexpected-exit test waits pane removal, but onExit unlocks before Warn; a channel-gated Go overlay deterministically observes0 warnings in this legal gap. The helper-only sweep was too narrow. Extend ARCH-ORDER rule to all asynchronous warning assertions and negative controls: acknowledge the actual published postcondition or owning callback completion, never preceding pane/phase state. Repair/audit delegated across couchtty/couchcmd; source behavior unchanged.

Recorded58b2 suite lease-3296915880 remained queued, last position5 after232s, never ran; explicitly cancelled and terminal context-canceled confirmed. No passing record claimed. Round4 measured3.43h (05:01:56 to06:37:40 local),8 admitted sessions,1 ownership interval, trackerfcce9acc42fa09b74ad2c2a471c96f382f7d2853. Close did not finalize despite Important finding being demoted at the round cap; repair it and re-review rather than using that demotion to ship.


### 2026-10-23 — BR-6 full assertion sweep verified

The plan now enumerates every affected async source, assertion acknowledgment and negative completion join, plus synchronous controls. Unexpected-exit waits actual warning source/body, with assertion-first and publication-first worker channels and existing SetForget completion. Stop negatives join following terminal commands; normal webcast end also joins final feed. No production change or new seam.

Temporary console.go overlay gates exact unlocked pane-removal-to-Warn gap: original test RED0.397s (“unexpected exit produced0warnings”); repaired test under identical forced gap x10 PASS0.565s. Focused exit/source x10 PASS2.645s; comprehensive warning classification/source/ordering race x10 PASS113.174s (session95070 terminal0). Complete JSON packages PASS: couchtty57.447s/4362 test-subtest passes, couchcmd113.354s/1784 passes,0 failures; session73808 terminal0, log /Volumes/TestScratch/tmp/pair-slot1/pair812-br6-packages.CzdDTf/run.jsonl. Only final guessing-stop test join followed compilation; that named test final-source race x10 PASS3.856s. Inventory and diff check PASS, no certificate changes. Next commit/push, fresh recorded affected/full and full close explicitly disposing BR-6. No recorded check active; stage remains held for TL go and operator acceptance.


### 2026-10-23 — restart retained-suite result and category correction

Authenticated Couch restart9cff332a-717a-4146-b23b-4e70c69dd224 prompted retained-check inspection. Lease-126562997 terminal exit1: full suite ran11m38s on d29420112ffe74bae7e2b7bb443c93a7a5dcbf8f; sole failure artifactpath.TestProductionArtifactReferencesAreExactlyClassified: console_continuation.go invented Kind continuation-scan matches the continuation- artifact filename prefix. All remaining suite targets passed (including couchtty41.721s/couchcmd62.389s). Failure log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/failed/c4af0d6972bf28d7255e8b19512cd5e2.log, sha256 d9597e7c122f. This is deterministic branch debt, not a registered flake or passing suite.

Use the existing continuation notice category already used by the other continuation failures, instead of inventing a scan-specific category or broadening the artifact inventory. One production literal changes; episode suppression/source/body behavior stays the same. Artifact classification GREEN2.923s; warning episode/publication ordering race x10 PASS3.347s; make test-affected-inventory PASS. Next commit/push, fresh recorded suite on corrected head, and narrow post-close review. No stage/go/live acceptance has occurred. Updated restart report21eb7d70-2192-4a76-af4c-0aff4f75c29b submitted to hub.


### 2026-10-23 — staged build and TL live-evidence allocation

Authenticated TL stage gobda966e4-a6d1-4b04-8516-35059ffa9bab. First stage refused #811's held lease without effects; TL e7b13fc1-02ea-4a24-b04e-660ced3f8b18 confirmed holder clear. sdlc stage --build succeeded on pair:0 at3dedd7b6de3cb3d0944d08e2e00fa81883d98f07. bin/couch, couch-conformance-bridge, pair, pair-launch-helper and pair-slug each verified that exact vcs.revision with vcs.modified=false. Independent post-build target Git HEAD equals staged pin and target/source are clean. Restart note/checklist e59e68e9-3588-40f7-8b4a-af2e839ac843 submitted. Existing processes retain old binary until operator restart; live acceptance not yet claimed.

TL be810f58-06be-43c0-ad23-910fa12dd692 explicitly allocates live proof to tab bar, global startup-warning selection, switcher Ctrl+Y and attached hub notification selection; no available disposable worker and no new slot for this. Exit and detached-target evidence is deterministic, not live: TestUnexpectedExitRaisesExactWarning covers exact retained source/body with assertion-first/publication-first and forced unlock-to-Warn overlay; TestGlobalAndDetachedAttentionOpenSwitcher covers global/detached navigation instead of focused actor. These pass in repeated/race and complete package proof recorded above, and review SHIP independently reversed the exit oracle. Do not report these two cases as operator-tested.

Global trigger sent to TL: start staged Couch with COUCH_INPUT_TRACE=/dev/null/pair812-input.trace plus normal launch arguments; failed trace open raises a genuine global warning, no trace opened. Attached trigger from authenticated ops:0: couch --notify --title '812 attached-target check' 'Select this item: it should open ops:0 exactly.' Existing CLI has no arbitrary/global/detached injection. These are TL-triggered, operator-observed; no worker killed/provisioned. Live results pending. Retained recorded suite lease-4010227337 on5eb99587cf9010715c99837953d7b81cd4d2eae4 remains queued, now first in line; terminal session34961 retained, log .git/sdlc/test-stats/inflight/f8a5f237aa232d426ddc445512650b0d/run.log. Next inspect suite to completion and wait for TL/operator live result; no merge until both pass. Stage remains pinned at3dedd7b6, irrespective of this evidence-only follow-up commit.

## Revisions

### 2026-10-23 — BR-3/BR-4 exhaustive completion

Apply the single actionable-source rule to every remaining callback/branch in console and composition root, preserving routine safety/recovery/feedback exclusions and existing attention owners. Migrate every removed-tab regression oracle while retaining its original parser, ordering, resize, focus and persistence contract; ordinary-startup negative controls need confirmed ownership and completed background work. No acceptance requirement is waived.

### 2026-10-23 — BR-1/BR-2 producer-class correction

Distinguish all three lifecycle terminal owners (local queue, remote queue, direct dispatch). For Chrome, classify successful/recovered/transient reports at typed sources rather than inferring severity from Logf text. Retain exactly one attention owner for uncertain remote input. This corrects the implementation against the already-approved attention/noise contract; no new product behavior or acceptance exemption.

### 2026-10-23 — TL resequences close before live acceptance

TL1e4df14f-c092-4148-8ad7-6921cd401d30 explicitly requests recorded check and close while #811 owns the stage, leaving #812's real-console acceptance as the final landing prerequisite. This supersedes the earlier close-after-live order, not the acceptance requirement. No code/scope change.

### 2026-10-23 — executable preservation and attention contract proposed

The operator's removal decision remains unchanged. Discovery shows the switcher does not actually draw the general Feed, so preserving that surface requires explicit projection rather than merely leaving MenuState.Notice alone. Add Done-when criteria for the already-required actionable-signal audit, and propose global/unavailable-target attention selection opening the switcher. The durable plan enumerates affected producer classes and keeps this navigation detail pending TL approval; no implementation scope is silently accepted.

### 2026-10-23 — TL approval

The preceding pending navigation choice is approved, with a producer-level noise exclusion for routine normal operation. No new scope beyond the recorded plan.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only; calibration source currently stale, values provisional. Spec resolves decisions (design ×0.2), implementation ×0.4 per v3.1, familiarity1.0; thorough-plan design buffer15%. No external API discovery or cross-repo propagation.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.03 impl=0.20
item: tui-screen design=0.20 impl=0.40
item: cross-cutting-refactor design=0.16 impl=0.20
item: atlas-docs design=0.02 impl=0.06
item: milestone-review design=0.02 impl=0.20
item: ux-rename-iteration design=0.06 impl=0.12
design-buffer: 0.15
total: 1.7435
```

### 2026-10-23 — TL allocates live and deterministic acceptance evidence

TL be810f58-06be-43c0-ad23-910fa12dd692 replaces the setup-heavy original live checklist with these four operator observations, after restart on the staged build:

1. Check collapse and multiline tab bars have no trailing message text.
2. Select the TL-triggered global startup trace warning; the switcher opens.
3. In the switcher, check retained message visibility and full Ctrl+Y copy, preserving operation/copy feedback precedence.
4. Select the TL's attached-target notification; ops:0 opens exactly.

Unexpected exit and detached-target navigation remain covered by deterministic controlled-order/race/native console fixtures, explicitly not operator-tested. No new disposable slot is created. Recorded-suite proof and these four live observations remain required before landing; acceptance results are pending.
