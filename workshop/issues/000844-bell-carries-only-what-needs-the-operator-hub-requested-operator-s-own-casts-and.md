---
id: 000844
status: codecomplete
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-24
estimate_hours: 4.12
collaboration-mode: collaboration
card_mirror: '467a51e4c246fb1653b465a0df4fae43768d69ed' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T10:30:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: N/A
---

# Bell carries only what needs the operator (hub-requested, operator's own casts and drives, no hub able to act); the rest goes to the hub

## Problem

## Spec

**The bell carries only what needs the operator; everything else goes to the hub (operator, 10-23).** pair#812
routed 34 producer classes to attention (list: ops `session.side/2026-10-23-11-bell-contents.md`). Many are things
the hub already receives and acts on. New default, **deny by default, add as we operate**: an event reaches the bell
only if one of these holds:
1. **A hub asked for it** (`couch --notify`). The hub decides what the operator must see.
2. **It's about something the operator is directly doing**: a cast or Chrome cast they started failing or ending
   unexpectedly, a cast code lockout, remote driving they're using failing, a cast or drive action they took failing.
3. **No hub can act on it**: Couch itself crashed or couldn't start its hub, or no live hub exists.

Everything else (supervision lines such as "N min with no later hub message", worker "waiting on a prompt" and
judged questions, lowered-priority relaunch, lifecycle cleanup and metadata risks, history-backup and continuation
problems, hub claim conflicts, trace and crash-capture setup failures) goes to the owning hub as a Couch system
message, and to the feed; the hub notifies the operator when a human is needed. Keep the routing table in one place
so adding a class back is a one-line change.

## Done when

- Only classes 1 to 3 add bell items; every other class reaches the owning hub and the feed (table-driven test over
  every producer class).
- With no live hub, class-3 events still reach the bell (test).

## Plan

Durable plan: [human attention routing](../plans/000844-human-attention-routing-plan.md).

- [x] Implement the pure typed routing table and bounded delivery reducer after TL approval.
- [x] Integrate Console startup/shutdown and existing guarded system-message delivery.
- [x] Sweep every producer, direct prompt/question/archive/stall path and duplicate-effect case.
- [x] Complete atlas, audited reader inventory and start recorded pre-close verification.

- [x] Repair owner-scoped generation and refused-disposition transitions under the approved revised contract, then rerun verification.

Workflow postconditions: binary-owned close review, PR/merge, then verify landing/archive/runner and report to hub.

## Log

### 2026-10-23


### 2026-10-24 — claim, audit and proposed plan
- 2026-10-24: closed — Approved scoped-generation/disposition re-plan passed change-code round4; current code passes integrated race x3 (core14.412s tty3.952s cmd17.884s helper1.160s), repolint6.555s, reader inventory and diff check. Main old-model semantic overlay fails4 refused-disposition cases plus revoked-handoff; real protocol handoff tests and N16/32 observation bounds pass. BR-2/3 disposed addressed in round2; BR-1 repaired across exact-root handoffs, revoked generations, unknown readiness and non-admitting confirmed-refusal disposition, with no fourth design delta. Normal measurement refusal: measurement branch proof: measurement sources exceed snapshot limit 128 at candidate4e5772e3855d0126ad3a3fe943183181114eb319, candidate window2026-10-24T10:30:06-07:00 through2026-10-24T11:42:28-07:00; engine printed no window/sessions. --no-actual authorized ops:0 delivery8ed4d3ff-1e37-4edf-9c60-5ecaf47edafe; no guessed hours. New recorded pre-close check on pushedd7a07ecba is pending; recorded pass and pair:4 combined live acceptance still required before landing.; review verdict: SHIP

Started independently from main after #812 landed; #853 is blocked on ariadne#598 and contributes no unlanded commits to this branch. Two bounded read-only audits enumerated all warning producers and existing hub-routing seams. Findings: status/warning Kind is not a policy class; two sites directly call publishWarningLocked; prompt/question and archive/stall paths bypass Warn; wait questions already send a hub message; startup warnings precede service creation. The plan addresses the full enumerable class, preserving existing admission/automation guards (ARCH-DRY, ARCH-PURPOSE).

TL approved policy in delivery 84c76ec1-c90f-40f0-885d-bfeafba9b898: automation OFF gives feed only/no replay; connected disabled hub is not absent; missing connected owning hub gives actionable fallback while routine feedback stays feed-only. The detailed design proposes typed classes, one table, buffered startup and bounded asynchronous hub delivery with exact occurrence IDs; global diagnostics retain primary ownership. Capacity fallback and live-hub picker interpretation are explicit approval questions in the plan. No code or tests changed and no passing verification claimed.

**NEXT ACTION:** receive TL approval of the durable design, resolve the bounded dispatch/overflow and hub-picker details, then run change-code (plan-quality review before estimate/implementation). No code before that boundary.


### 2026-10-24 — detailed design approved

Ops:0 delivery 12d7cfb1-b715-4546-b159-69f6c8cde4bb authenticated as submitted from the bound hub: detailed plan approved, including one table, startup buffer/reducer, 128 x 4 KiB/four admissions/two-second contexts/two-minute expiry, no uncertain resend, primary global ownership, one aggregated capacity-loss bell, and connected-hub picker rule. Proceed to change-code. Initial invocation waited on #838's live close lock and was canceled before acquiring it to record the approval and select the full flow; no implementation/gate bypass occurred.

NEXT ACTION: run change-code --issue 844 --flow full; address plan-quality findings, derive the estimate only after that review passes, then execute the test-first tasks.


### 2026-10-24 — plan gate round 1

Full-flow plan review returned one Important finding PQ-1 (executable-test-strategy): name the production functions under test and compress prose scenario inventories. Revised the entire test contract and task section to name RouteNotice, AdvanceNoticeDelivery, resolveNoticeOwner, runNoticeDelivery, publishWarningLocked, applyNoticeRoute and the existing condition/notification functions, with one adversarial strategy each. State sequences call the production reducer and check independent no-duplicate, epoch-fencing and resource-bound invariants; stateful shell fixtures control effect completion ordering. No policy change, implementation or estimate yet. Existing Attention unit baseline passed (go test ./cmd/internal/couchcore -run TestAttention -count=1, 0.350s).



### 2026-10-24 — implementation checkpoint, integration verification running

change-code full flow passed: PQ-1 addressed, round 3 clean, estimate-quality INFO (producer/integration budgets optimistic). Implemented typed routing, bounded pure delivery state and four-worker shell, Console feed-once projection, startup/close binding, exact owner/epoch broker backend and the complete producer/condition/archive/stall sweep. Existing picker/question wait watcher retains canonical sends, thresholds, mute and deduplication; HubHandled denotes that effect ownership rather than proof of successful submission. This avoids introducing immediate duplicate picker traffic. Startup waits on surviving/explicitly attached wrappers, never treating a nil boot service as absence.

Test-first failures covered missing APIs, typed producer facts, UTF-8 clipping, admitted receipt tracking, full-capacity condition coalescing, shutdown cancellation, and OFF/ON before async dispatch. Publication now captures cached owner readiness/epoch under the documented memory-only lock order; broker IO runs only outside Console/publication locks. Whole-package AST guards cover Warn, locked warnings and direct attention additions outside audited projections.

Verified focused race aggregate: go test -race ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'TestNotice|TestConsoleNotice|TestCondition|TestPicker|TestWaitNotice|TestWaitAutomation|TestArchive|TestAutomaticInputStall' -count=1 -timeout=120s PASS (core 20.671s, tty 3.021s, cmd 13.406s). Adapter race x3 PASS 4.092s; startup real-broker integration race PASS 1.920s; unknown-class fuzz PASS 10,811 executions. Full tests for those three packages are currently running; no full-package or recorded check pass claimed yet.

Reader audit: wait_notice_test.go still reads the same Claude/Codex wrapper screen fixtures and retains its registered fixed-scanner guard; only fixture cleanup/expected routing changed. New notice_class_inventory_test.go files and tty notice_producer_inventory_test.go scan only their own Go package's tracked source or literal AST fixtures, so normal package ownership covers their inputs; each is explicitly local_or_literal_fixture. Individually recorded those four named certificates; make test-affected-inventory and git diff --check PASS. No selector or local-gates configuration changes.

NEXT ACTION: read the retained three-package test result, fix/classify failures, complete atlas/plan verification, push and run recorded --pre-close, inventory then close. The full recorded check has not started.


### 2026-10-24 — broad-check diagnosis and supervision fixture correction

The three-package broad run was explicitly stopped with SIGQUIT for diagnosis at about six minutes, not counted as passing. The cmd stack was waiting in TestSupervisionWarningCompositionRaisesAttention -> openWarningAttention for the old raw supervision.notice callback, which is now routine/feed-only in production. Updated that existing test to invoke the typed supervision.warn path for its no-connected-hub fixture; class-3 fallback passes (standalone 1.897s). Retained its original registered test name. Its only inputs remain disposable fixture directories, so individually refreshed its existing local_or_literal_fixture certificate.

The core stack was inside existing TestFreshStartupPostClaimPreparationFailuresRollBack -> startupCleanupFixture -> OSSlotCatalog.Discover/ProvisionFixture.Run. Those files are byte-identical to this branch's main base; isolated current-head run passed 7.083s. This is an interrupted broad observation, not a registered flake or claimed baseline suite execution. A prior targeted run compiled before startup fixture barrier correction was also terminated diagnostically; current startup integration has the passing race evidence above. No recorded pre-close run has been canceled or started.

A new complete couchcmd run is in progress with JSON events retained at /Volumes/TestScratch/tmp/pair-slot1/pair844-targeted-7nn8ata4/couchcmd.jsonl, so progress/failure can be inspected without assuming that silence means a hang. The final recorded check remains necessary.


### 2026-10-24 — final focused verification before recording

Final repeated race check PASS: go test -race ./cmd/internal/couchcore ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'TestNotice(Route|Delivery|Owner|Class)|TestConsoleNotice|TestSupervisionWarningCompositionRaisesAttention' -count=3 -timeout=90s (core 2.794s, tty 2.572s, cmd 6.920s). make test-affected-inventory and git diff --check PASS after the supervision fixture correction. Complete couchcmd JSON run remains active; next inspect its exact result, then record pre-close on the pushed final head.

### 2026-10-24 — complete-package result and final readiness correction

The complete couchcmd JSON run finished in 502.363s with 1,982 passing test/subtest events and only TestWaitOriginalPermitReachesMailboxCommit/false and /true failing (parent included). Both failures expected passive bell attention under a connected owner after an OFF/ON transition. Updated those expectations to retain the judged episode while requiring no bell; unchanged mailbox suppression, epoch and no-replay assertions remain. Focused automation/original-permit race verification passed (5.538s), and final owner/agent-kind/automation race verification passed (3.748s). This was a branch test-expectation defect, not a classified flake or passing complete-package run.

Source audit also found registry liveness can precede EffectConnect's identity/broker commit. resolveNoticeOwner now treats that in-flight witness as booting rather than absent; the regression failed with Unavailable before the correction and passes after identity commitment. Repeated owner/startup/condition race verification passed x3 (5.810s). The new real Console-to-adapter-to-registry/broker test covers Claude and Codex, exactly one system-origin delivery and matching canonical submitted receipt, independently rerun in the final race selection. Its inputs are literal fixture data and disposable service state, without external reader inputs.

Standing authorization from ops:0 delivery 8ed4d3ff-1e37-4edf-9c60-5ecaf47edafe applies only if measurement refuses with 'measurement sources exceed snapshot limit 128': use --no-actual with the exact refusal and measured window in --verified, never guessed hours. Measurement has not yet been attempted for this issue.

Integrated producer selection passed with race detection: go test -race ./cmd/internal/couchtty ./cmd/internal/couchcmd -run 'Test.*(Notice|Attention|Warning|Picker|Question|Notify|AutomaticInputStall|Startup)' -count=1 -timeout=180s (tty 16.913s, cmd 16.274s). make test-affected-inventory passed after staging the new agent-kind test; git diff --check passed.

NEXT ACTION: commit/push this final checkpoint, then run recorded local-check --pre-close. No recorded check or live acceptance is claimed. Coordinate any :0 stage/live check through the hub before touching that slot; close/landing remain outstanding.

### 2026-10-24 — recorded check queued; close review may overlap

Pushed candidate 1be62e2f568c9d8b17f87700456b549d934be446. local-check --pre-close resolved to affected, then selected the full suite because tests/affected-readers/cmd/internal/couchcmd/notice_attention_test.go.json changed (the known #853 ownership-policy limitation). Retained suite run lease-3505757571, terminal session 40123, disposable checkout /Volumes/TestScratch/local-checks/sdlc-local-2682892674; currently queued, not passed. Current tl-worker rules expressly allow close review to overlap this queue; a pass on final code head remains required before landing, and review code fixes require a fresh check.

Ops:0 delivery c7464d76-eef8-46c9-b642-0300e78316af authenticated submitted: the live look joins the combined pair:0 stage coordinated by pair:4. After review and green suite, send pair:4 the reviewed head plus three or four operator steps; do not touch pair:0. Live acceptance remains outstanding. Adjusted task bookkeeping to distinguish starting the recorded check from the required pre-landing pass, without asserting unobserved success.

NEXT ACTION: inventory then binary-owned close review while the suite waits/runs; retain the check and classify its complete result. After both gates pass, send pair:4 the head/checklist for combined stage, obtain live acceptance, then PR/merge and outcome report. Do not restart this slot while its check is queued/running.

### 2026-10-24 — authorized actual-measurement refusal

Normal close on candidate 4e5772e3855d0126ad3a3fe943183181114eb319 refused before review: "can't auto-compute (measurement branch proof: measurement sources exceed snapshot limit 128)." It printed no measured window or sessions. Candidate window reconstructed from the authoritative card's started anchor and pinned last issue commit: 2026-10-24T10:30:06-07:00 through 2026-10-24T11:42:28-07:00 (tracker claim commit 2013428b9 at 10:30:07; no release). This identifies the failed candidate, not numeric actuals or a successful measurement. Using only --no-actual under ops:0's standing authorization 8ed4d3ff-1e37-4edf-9c60-5ecaf47edafe; no typed hours. The suite remains queued and live acceptance remains outstanding.

### 2026-10-24 — boundary round 1 REWORK

Review returned BR-1 Critical (notice-owner-loss-fallback), BR-2 Important (user-docs-behavior-parity) and BR-3 Minor (shared-analysis-single-source). Verified BR-1: pending epoch mismatch conflates unavailable owner epoch zero with automation revocation; confirmed SendResult Refused also retires without unavailable-owner routing. Repair covers pending and immediately-before-admission disconnect while preserving OFF/ON and admitted/uncertain no-replay. README's two obsolete unconditional bell descriptions are being corrected; duplicated producer AST analysis is being consolidated. Revisions and prevention lessons record the class sweep.

Canceled old suite lease-3505757571 through sdlc local-check cancel while it was still queued (never executed), so obsolete code will not consume a full run; terminal confirmed context-canceled suite-lock wait. No check pass or flake classification. Queue a fresh check only after the review fixes are verified and pushed. Hub informed in submitted message 6f903a07-b5c1-442d-b57b-ea269732cc4d.

### 2026-10-24 — BR-1 identity retention approved

The class sweep found a second loss ordering: removing both source workspace and owner leaves Thread-only occurrences without the root needed to retain automation provenance. Hub approved one Root string in the existing owner fact, copied to existing occurrence.OwnerRoot at publication (authenticated submitted delivery 600b1333-b9a6-46a5-8dc8-ce8f929e811c). Current verified ownership takes precedence; removed sources retain their exact prior root. No queue, retry, wire or durable state added; regression before production correction. This complements the pending/refused absence-vs-revocation fix.

BR-2 corrected README's summary and bell interaction text, plus architecture map parity. BR-3 consolidated both AST analyses and rejection fixtures into cmd/internal/testnotice, keeping package-specific audited projections/forwarders. Main independently reran helper/inventory tests PASS (helper 0.062s, cmd 0.335s, tty 0.365s). Individually refreshed certificates still cover each caller's own Go package; helper parses supplied bytes and literal fixture tests only. Targeted repolint found three pre-existing branch fixed sleeps in notice_delivery_test.go; replaced with bounded production reconciliation/event observations in the same repair round, with final lint still required.

### 2026-10-24 — BR-1 repair verified

Observed deterministic REDs: reducer confirmed-owner-loss produced no effect; backend disconnected epoch changed 2 to 0; four held admissions plus one pending occurrence produced no fallback; refusal immediately before the real guard produced zero projections; Thread-only source-plus-owner removal produced zero fallback in both orderings; startup-buffered owner resolution failed to retain exact root. Fixes retain verified Root at publication and OwnerChanged, preserve the cached automation epoch on proven unavailability, and route only confirmed Refused results through RouteNotice when the frozen generation still matches. Pending and guarded OFF/ON-plus-disconnect cases remain feed-only; admitted/uncertain/late duplicate cases never bell or resend.

Agent final broad race x3 passed (core 1.786s, cmd 3.842s). Main independently reran integrated race selection across core/tty/cmd/testnotice (TestNotice, ConsoleNotice, Condition, Picker, WaitNotice, WaitAutomation, Archive, AutomaticInputStall, Findings): PASS core 4.117s, tty 2.042s, cmd 5.112s, helper 1.146s. Main repolint catalog PASS 4.871s after the three fixed sleeps were removed. make test-affected-inventory and git diff --check PASS. README/architecture map and shared analyzer corrections are committed separately. No recorded check pass or live acceptance yet.

NEXT ACTION: commit/push this verified fix, rerun the binary close review with the authorized precise actual waiver; satisfy any noon freshness refusal before queuing a new final-head recorded pre-close check. After review and recorded pass, send pair:4 head plus short operator checklist for the combined stage; do not touch pair:0 independently.

### 2026-10-24 — round 2 REWORK; stop and re-plan

Round 2 marked BR-2/BR-3 addressed, retained BR-1 Critical: pending source handoff from /repo1 epoch2 to /repo0 epoch0 wrongly compares different root-scoped counters and clears the notice; destination loss then gets zero fallback. Existing disconnect regressions were independently verified red against pre-fix overlays. No full check queued.

Independent scratch audit while review ran (no shared source edits) also reproduced unknown-readiness losses: a pending identity witness returns Booting epoch0 and clears known-epoch pending work; retaining epoch alone lets a later OFF/ON overwrite the generation because Booting is exempted; a confirmed guarded refusal during unknown readiness retires without later fallback. Main independently reproduced TestNoticeOverlayUnknownReadinessPreservesPending RED via scratch overlay. Scratch-only two-part pending correction passes race x3, but guarded-refusal case remains RED and is deliberately not patched incrementally.

Stopped under the worker third-delta rule and appended a revised owner-scoped/disposition transition contract to the durable plan. Alternatives reject freezing the old owner forever or silently retiring unknown work, because both violate existing approved policy. Proposal adds only one phase in the existing bounded reducer for confirmed-refused local disposition, plus memory-only root-scoped generation observation. Awaiting TL approval before production changes and change-code re-admission. Hub stop report 5b0d9e91-f370-4824-bdf5-5674e7ad26ba sent; receipt must be checked.

NEXT ACTION: present the committed revised plan to hub and await approval. No source changes, suite, stage or new close until approval; afterward change-code, deterministic red/green transition matrix, final pushed recorded check, close review and combined live look.

### 2026-10-24 — revised contract approved

Ops:0 approved exact-root generation validation, accepted-handoff adoption of destination Root/epoch, and one non-admitting AwaitingDisposition phase after confirmed refusal, with connected/feed, unavailable/bell, unknown/wait dispositions and unchanged bounds (authenticated submitted delivery a5bd7b97-5321-4e2f-8353-ab458864d057). Any fourth design delta must stop for scope reduction rather than another mechanism extension. Plan-quality owns the fresh-eyes check; run change-code before implementation. Revised plan request c4442aa0-0594-42f3-ae97-0ed0ba50b4b8 was submitted. No production changes have been made since round2's reviewed head.

### 2026-10-24 — revised change-code admitted

change-code passed revised plan-quality round 4 with no open findings; accepted-design commit 07101ce59. Estimate-quality INFO requested explicit revised-work/live/landing allocation, without blocking. Updated the estimate derivation from 3.24 to 4.12 h using the same v3.1 primitive calibration, adding only the approved pure extension and integration qualification, not duplicating the original producer sweep. Actuals remain unmeasurable under the recorded authorized refusal, and no estimate is used as actuals. Implementation now proceeds test-first under the approved contract; a fourth design delta must stop for hub scope reduction.

### 2026-10-24 — revised scoped/disposition implementation verified

Implemented the approved contract without a fourth delta. Pure reducer owns Revoked observations, root-scoped handoff adoption and AwaitingDisposition. Backend NoticeOwnerCurrent validates only the retained Root's cached automation epoch; known-root Booting retains that epoch. Shell rechecks exact Owner/Token/Phase before using observations. Confirmed-refused entries can wait for local disposition but never Admit; admitted/uncertain retain only canonical receipt ownership.

Main independently ran the old-model semantic Go overlay: four TestNoticeDeliveryRefusalAwaitsOnlyDisposition cases fail 'confirmed refusal lost unknown-readiness disposition', and PendingScopedGeneration fails 'revoked prior generation followed handoff' (0.211s). Existing tests also reproduced five old-shell handoff/readiness failures. New real-protocol hub-move fixtures hold four admissions and a fifth pending, prove enabled exact-root admission, destination-loss fallback, disabled/no replay, pre-handoff revocation and post-handoff old-root non-authority; guarded refusal does one attempt, then feed/bell/unknown disposition. N=16/32 and doubled history count at most two cached observations per active entry with zero additional IO/admissions/workers. No new external reader inputs; paths are disposable service fixture state.

Main final integrated race run x3 PASS: core 14.412s, tty 3.952s, cmd 17.884s, testnotice 1.160s, covering notice/Console/condition/picker/wait/automation/archive/stall/analyzer tests. Main repolint catalog PASS 6.555s; make test-affected-inventory and git diff --check PASS with the new test staged. Atlas updated for scoped generations, accepted handoff and the non-admitting disposition phase. These are targeted passes; no recorded suite or live acceptance claimed.

NEXT ACTION: commit/push repaired head; finish plan bookkeeping, recorded pre-close check and close round3 (overlap allowed). Any fourth design delta stops for TL scope cut. Once review and recorded check pass, send pair:4 exact head and short operator steps for combined stage; live acceptance precedes PR/merge/landing report.

### 2026-10-24 — close SHIP; recorded verification pending

Close round3 SHIP, confidence high, all BR-1/2/3 addressed and zero open findings. Reviewer independently mutation-verified scoped revocation, witness retention and refused disposition; architecture checks pass. Card codecomplete bound to evidence c0214f319186f78bce7337300719c49fa5d7f203; close mirror HEAD b3e5ae728c12befc397151eb5434d4972414036e. No fourth design delta. Numeric actuals remain N/A under the exact authorized measurement refusal recorded above.

Recorded pre-close selected full suite because a reader certificate changed, on code d7a07ecba6348d1d1a9d9de3c3704c45926c19c6. Retained terminal session 91413, run lease-1632073207, disposable checkout /Volumes/TestScratch/local-checks/sdlc-local-994618917; queued, not passed. Do not cancel/restart merely for waiting. Close commits are bookkeeping only; code unchanged. PR #593 opened for CI overlap: https://github.com/xianxu/pair/pull/593. No merge runner or landing ticket entered before the required recorded pass/live look. Hub report 2a65b2d0-0a34-42a8-ab43-e77989cb9c51 verified submitted.

Reviewer noted intermittent existing TestFreshStartupRunSerializesLaunchThroughAdoption in its broader startup selection (isolated/repeated pass); checking existing flake registration rather than attributing an unrelated base cancellation failure to it. No recorded failure has occurred on this run.

NEXT ACTION: monitor retained suite to completion and classify every failure; use registered-flake resume only when its conditions hold. On green, send pair:4 current Git head and three short operator steps for the combined stage (routine restart/feed, explicit hub --notify, expected cast start/stop remaining quiet); no unilateral pair:0 change. Record live verdict, then enter sdlc merge --yes for PR593 and verify tracker done/archive/runner exit before hub outcome. If further design is required, stop for TL scope cut.

## Estimate

Produced via brain/data/life/42shots/velocity/estimate-logic-v3.1.md against baseline-v3.1.md. Method A only. Calibration is flagged stale by estimate-source, so these are provisional ship-wall-clock hours. Read v2/v2.1 procedure and table: retain issue/spec design cost; use x0.2 design discount for approved module/integration/refactor decisions, 40% implementation conversion, familiarity 1.0 and 15% design buffer.

Reuse check: existing AttentionList/feed, cached authority, guarded sendAutomaticSystem, canonical receipts and stateful fixtures cover the stack; no novel external library/API discovery is needed. Policy is one small extension, delivery reducer one single-concern greenfield module, dispatcher an integration over the existing broker, producer migrations two bounded cross-cutting surfaces (TTY and service), and condition projection one existing TUI seam. No double-counted new screen or transport.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.5 impl=0.04
item: smaller-go-module design=0.04 impl=0.16
item: greenfield-go-module design=0.2 impl=0.32
item: api-integration design=0.2 impl=0.4
item: cross-cutting-refactor design=0.1 impl=0.2
item: cross-cutting-refactor design=0.1 impl=0.2
item: tui-screen design=0.1 impl=0.2
item: atlas-docs design=0.05 impl=0.04
item: milestone-review design=0.0 impl=0.2
item: smaller-go-module design=0.04 impl=0.2
item: api-integration design=0.2 impl=0.4
design-buffer: 0.15
total: 4.12
```

2026-10-24 revised allocation after plan-quality round 4 passed: the previous 3.24 h estimate remains recorded in the Log; the approved owner-scoped/disposition repair adds a well-specced smaller Go extension (design 0.04, impl 0.20) and integration/test qualification (design 0.20, impl 0.40), using the same table, approved-plan discount and 40% conversion. These fund the new pure phase/revocation fact, existing backend seam extension, handoff/readiness matrix and counted observations; no new greenfield module or producer sweep is counted again. Design subtotal 1.53 x 1.15 + implementation subtotal 2.36 = 4.1195, rounded 4.12 h. This is a revised estimate, not actuals.

Original milestone-review/process impl 0.20 allocation includes the one final boundary plus coordinated live acceptance and PR/merge/archive/report; repeated convergence reviews remain convergence rather than new milestones. Focused/recorded tests are included in component implementation, and the new integration allocation covers the expanded deterministic qualification. The provisional stale calibration caveat still applies.
