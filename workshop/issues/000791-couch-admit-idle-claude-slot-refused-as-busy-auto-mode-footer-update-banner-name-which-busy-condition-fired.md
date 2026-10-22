---
id: 000791
status: codecomplete
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-22
estimate_hours: 1.597
collaboration-mode: collaboration
card_mirror: 'e314a8e0135326547e264739d52bb22ca60a386d' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T12:23:59-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
actual_hours: 0.64
---

# Couch delivery: capture delayed refusal evidence

## Problem

## Spec

From ops-echo:0 (10-21, 18:5x–20:1x): `couch --prepare echo:3 --confirm` refused three times with
`admit refused (busy) echo:3: the agent has a turn open, the composer holds text, or input was seen in the last few
seconds`, while `couch --peek echo:3` showed Claude idle at an empty `❯` prompt and `couch --actors` showed it
available. The peek footer read `⏵⏵ auto mode on (shift+tab to cycle)   ✔ Update installed · Restart to update`.

Hypothesis (unverified): the admission check doesn't recognise Claude's idle composer under the **auto mode**
footer (other Claude slots show `bypass permissions on`), or the update banner, so it falls back to "busy". Find
which of the three busy conditions fired; the refusal should name the one that did.

## Done when

- A forced pre-paste refusal at the active delivery head produces exactly one local wrap-events diagnostic after 15 seconds, containing its refused screen and precise blocking condition; no early or repeated capture.
- The complete encoded diagnostic is at most 64 KiB; any truncation is explicit. Encoding and writer IO occur outside admission/mailbox locks.
- Tests show diagnostic failure leaves receipts/input safety unchanged, for Claude and Codex, including Claude batch preparation.
- The capture includes the latest resize attempt/completion timing, requested geometry, changed flag and outcome, coherently copied with the refused screen generation. A visible Claude pane is exercised through one/two/one tab-bar rows.
- Recovery is tracked in follow-up #810; the approved resize-then-silence investigation supplies its red case, without changing recognition here.

## Plan

Implementation: [durable plan](../plans/000791-couch-admission-plan.md), final diagnostic-only revision approved by TL.

- [x] File recognition follow-up and checkpoint approved diagnostic-only scope.
- [x] Forced-refusal tests first, bounded capture state/encoding and dispatch integration.
- [x] Lock/failure/cap/growth/personality tests, atlas and normal source inventory.
- [x] Run targeted capture/preflight checks and submit the recorded affected check on the pushed implementation.
- [x] Approved resize addition: native wrap/repaint and controlled-silence fixtures, coherent resize timing, generated/concurrent tests, targeted race/fuzz and atlas updates.

The initial diagnostic implementation has a SHIP close review; recorded affected pass and landing remain pending. The approved resize addition below requires fresh implementation verification and boundary review before landing.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; calibration reported stale by estimate-source, so provisional. Derived after plan-quality CLEAN (round 2). Familiar Go/terminal/BufferedWriter seams, no novel stack/library needed. Thorough approved design applies x0.2 design discount and +15% design buffer. For the original scope, two smaller Go units cover eligibility/encoding with generated/fuzz tests and production dispatch/lock/failure tests respectively. Those module and review implementation items pick 0.5h primitive upper endpoint x0.4; issue-spec and atlas use 0.20h and 0.10h base selections respectively; familiarity1.0. Spec0.5x0.2, state0.3x0.2, integration0.2x0.2, atlas0.1x0.2, review0.2x0.2 design.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.10 impl=0.08
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0.04 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 1.597
```

## Log
- 10-21 22:0x (ops:0): **a Codex case, possibly the same class.** parley.nvim:3 (Codex, tag 1-parley-nvim-83) idle at an
  empty `› Ask Codex to do anything` composer, but every delivery since the 21:33 Couch restart sits at "waiting for
  recognized composer" (ops:0's 9d16fde1 at position 2, behind the restart notice). Its screen differs from Codex
  slots that receive normally: a **blank line between the composer and the footer**, and the footer reads
  `⚠ 1 warning · f2 to view` (others show `⚠ 2 warnings`). Capture that screen as a second fixture.

- 10-21 ~20:40 (ops-echo:0): **cause pinned: the `✔ Update installed · Restart to update` banner** in Claude's footer
  blocks composer recognition. After the operator restarted echo:3's Claude (banner gone, auto mode still on), a
  resend delivered. While the banner showed, sends also sat at "waiting for recognized composer", `couch --relaunch`
  refused busy, and `couch --reboot` failed at recovery confirmation: "slot zellij session 1-393 kill not confirmed:
  ownership unresolved, unknown pane owner, pane has no unambiguous actual command" (operation
  e630974ef114e3fe95e51db5a22971b0, launch-diagnostics/failures.json); that reboot failure may be its own issue.
  Fixture: footer `⏵⏵ auto mode on (shift+tab to cycle)   ✔ Update installed · Restart to update`.

### 2026-10-21

### 2026-10-22 — recurring hub stalls; investigation dispatch
- 2026-10-22: closed — TL-approved diagnostic-only delivery. BR-1 regression reproduced 37009-byte fitting styled row omission; exact escaped-JSON budgeting now passes it and all capture tests under race2.933s, fuzz400/6.498s. Prior full wrapper package PASS196.078s; diagnosticlog race PASS102.837s; corrected-source full wrapper package running; repolint36.564s/artifactpath21.673s pass. New-head recorded affected queued, landing waits. BR-2 measured tracker proof in Log/dispute confirms #810 open/unclaimed and awaiting first real capture. No recognition changes.; review verdict: SHIP

- Operator report via ops:0: hub deliveries have repeatedly stalled until an operator keystroke, then flushed in a batch, both before and after #735 landed. The hub is where the operator watches; worker slots may have the same symptom, but that is not yet observed evidence. This is not a newly introduced hub-only failure.
- Latest observed ops:0 footer: `⏵⏵ bypass permissions on (shift+tab to cycle)       ✔ Update installed · Restart to update`. Read-only live capture at investigation start confirms an empty `❯` prompt between horizontal rules and that combined footer. Claude version in the authenticated dispatch identity: 2.1.304.
- The update banner is confirmed for this observation and reported for echo:3; its presence on every earlier hub stall is unknown. Treat the earlier “cause pinned” entry as evidence of recovery after restart, not proof that the banner explains all stalls. Reproduce both Claude footer modes with the banner, compare banner-free controls, and investigate a possible second trigger including the Codex blank-line/warning-footer case.
- TL dispatch and correction authenticated as submitted to this slot: 6af1c55a-9494-4f6e-8c48-b43e5a8c4c2a and c13682ef-f911-427f-93cc-cd9fe557d319. Plan goes to TL before production changes. #737 remains parked on its pushed issue branch awaiting a separate explicit stage go.

- 12:23 operator clarification (authenticated dispatch 95e4c949-c196-485c-ac04-007c139910c9): intermittent, perhaps twice daily, with delivery otherwise working while the update banner remains for hours. Detection: session-file submit gives no immediate agent-bar message, or the usually busy hub goes quiet. Banner presence alone cannot explain this. Investigate redraw/resize/queued-message/spinner/turn-end transients. Add bounded automatic capture of the refused screen and precise blocking condition after a delivery wait threshold.
- Claimed and started planning from current main, independently of #737. start-plan refreshed its ariadne substrate; weave generated an unrelated merge-check workflow change. Preserved that generated change in the named stash `#791: preserve unrelated weave refresh workflow output`, excluded from this issue.
- Initial trace: live-restart admission consumes only a settled boolean and conflates its causes; peer delivery has separate pre-paste conditions and does not require the turn to be inactive. Claude footer wording is outside its ruled-box composer recognizer. Therefore do not equate restart-busy with a peer-delivery turn gate, and do not declare the banner the root cause without replay evidence.

- Scratch-only native replay completed: `go test -overlay /Volumes/TestScratch/pair791-replay.v_daojgb/overlay.json ./cmd/internal/wrapcmd -run '^TestInvestigation791CaptureReplay$' -count=1 -v` PASS, package 4.596s. The retained ops:0 bypass/banner screen (54,648,427 bytes / 92 resize events), echo:3 earlier 1-echo-87 auto/banner pre-exit screen (80,031 bytes / 5 resizes), and parley.nvim:3 blank-line/one-warning screen (9,662,972 bytes / 48 resizes) all yielded PeerComposerEmpty, complete=true, cursor visible=true, at (2,33), (2,34), (2,35) respectively. This is a passing control, not reproduction of a stalled moment. The unconditional wait-detail helper is meaningful only on unknown composer; its generic string printed by this scratch probe does not describe a refusal when the composer enum is Empty.
- Durable plan: workshop/plans/000791-couch-admission-plan.md. No production code changed. Proposed one active-delivery capture at 15s, 64KiB total record through existing bounded wrap-events writer, exact refusal and cursor/frame/style evidence; narrow read-only settle-observe for restart diagnostics without changing admission policy or session dialect. Root-cause fix remains blocked on a failing captured transition, not on a guessed banner match.

## Revisions

### 2026-10-22 — operator/TL expanded reproduction and capture scope

The original auto-footer/banner-only acceptance is insufficient: recurring stalls survive #735, delivery usually works with the banner present, and a Codex case exists. Add bypass-footer/banner and banner-free controls, the captured Codex case, explicit actual refusal at restart admission/effect time, and bounded automatic capture of the refused screen/condition after a delivery wait threshold. Preserve draft/picker/unknown-state refusal. Completion requires a failing captured transition, regression and narrow fix, diagnostics failure/cap/ordering tests, and TL-coordinated real acceptance; passing static controls alone do not establish completion. Earlier Log entries remain historical evidence rather than an all-stalls root-cause claim.

### 2026-10-22 — diagnostic-only approval

TL authenticated approval 4333f27a-2f1d-4e8a-b9dd-039d84a610c0 explicitly makes capture the #791 deliverable now: forced refusal fires a capture at 15s, refusal names its condition, total record <=64KiB, existing wrap-events writer, no IO under admission locks. Recognition fix moves to an unclaimed follow-up waiting for the first failing capture. The broader proposed restart endpoint and static-fixture deliverables are superseded, not implemented.

### 2026-10-22 — revised completion contract

Supersedes original auto-mode/banner idle-admission Done-when and the earlier expanded root-cause acceptance: #791 now ships diagnostics only per TL. Current Done-when and Plan above reflect this approval; original contracts and rationale remain in Git/history and the durable plan's preceding sections. No recognition/safety-policy change or claim that the historical fault is fixed.

- Filed recognition follow-up #810 (unclaimed, waiting for first real failing capture). Plan-quality round 1 requested named risky functions and strategy-based test contract (PQ-1); final revision now names advancePeerWaitCapture and encodePeerWaitCapture with generated-sequence and fuzz oracles. No production changes before gate acceptance.

### 2026-10-22 — diagnostic implementation and targeted evidence

- change-code accepted full flow, plan-quality CLEAN after PQ-1 and estimate-quality INFO. The estimate is provisional; issue-spec includes already completed claim-to-approval work, and module/review allowances include active verification/repair time, not just typing. Full native checks may exceed that allowance; actual remains measured, not back-fitted.
- TDD behavioral red: TestPeerWaitCaptureForcedRefusal reported “missing capture at threshold: unexpected end of JSON input”; implementation green then passed in 0.341s. Capture selection is under the existing dispatch owner, and emission is deferred until inputAdmission and mailbox mutexes are released. No recognition or restart policy changes.
- Added advancePeerWaitCapture/PeerWaitCaptureState, encodePeerWaitCapture/PeerWaitEvidence, and terminalModel.SnapshotGeneration to retain copied cells and their exact generation. Diagnostic JSONL is bounded as a whole, including escaping/envelope/newline; bottom-first ANSI rows retain styles, and explicit truncation marks incomplete screens. Existing wrap-events BufferedWriter owns storage and lifecycle.
- Forced capture/threshold/repeated-poll/recovery, early guards, generated state sequences, styled-screen replay, caps, Claude/Codex/Qoder and Claude batch preparation, disabled/failing writer and actual managed-file/lock tests pass under race (2.821s). Baseline #735/peer-wait controls pass (0.393s); repolint 17.016s and artifactpath 19.072s pass; affected-reader inventory check passes. Full wrapcmd/diagnosticlog race checks and bounded encoder fuzz are running; not yet claimed passed.
- Follow-up #810 remains unclaimed and waits for the first real failing capture. #791 was retitled through the card setter to “Couch delivery: capture delayed refusal evidence”. #737 remains parked and unstaged.

- Encoder fuzz completed PASS (2,408 executions, 6.820s). PR #546 opened; pushed code head b01de4e4d953b1574fe3340516a3dac914c47e58. Recorded affected lease-2648858986 is queued (initial position 11), exact-head checkout /Volumes/TestScratch/local-checks/sdlc-local-3376230752. Full wrapcmd/diagnosticlog race check remains running. Close review may overlap these checks under tl-worker policy; landing still requires successful final evidence. No manual main integration or live-stage action.

- First close attempt measured 0.45h (window 2026-10-22T12:23:59-07:00 to 12:47:08-07:00; 1 ownership interval, 6 admitted sessions, 216 sessions excluded without verified provenance), then refused before review because the Plan's last checkbox combined pending recorded checks, close review and landing. No review ran and no close finalized. Revised that row to the completed targeted-check/submission step and retained the pending acceptance transitions explicitly in prose/Log, rather than claiming they passed or waiving the plan gate.

### 2026-10-22 — separate implementation from pending gate-owned acceptance

The final Plan row now records targeted-check execution and recorded-check submission, which are complete. Actual check pass, close review and landing remain pending gate-owned transitions explicitly recorded above; the prior combined checkbox could not truthfully be checked before invoking close. No acceptance requirement was removed and no gate bypass was used.

### 2026-10-22 — close round 1 and corrections

- Close round1 REWORK: BR-1 found conservative raw/6 row budgeting can omit a fitting styled screen; scratch overlay regression reproduced a 2000x1 alternating-bold row whose JSON is37,009 bytes yet capture had zero rows. New TestPeerWaitCaptureFittingStyledRow failed before correction. Encoder now accounts actual escaped JSON bytes incrementally, including row envelopes, separators, full metadata and JSONL newline, with a raw-byte lower-bound check before escaping oversized graphemes. Temporary construction remains bounded and linear.
- BR-2 is an unverifiable finding due to the disposable review checkout lacking origin/issue-tracker, not a missing issue. Host commands re-run 2026-10-22T19:55:19Z: `sdlc issue show 810 --json` reports tracker 7ba6cc0eb5f81693864fc39260d0866794d87cab, card d7f96db618cdea2cd2770c66f859d20e43262bb4, status open, assignment relation unattributed, no holding workspace/branch. git show of that tracker card has no owner. Details are pinned on origin/main by 4a7ab2ca1a4c0b0d4aa34e8cada1d8c2917427bc at workshop/issues/000810-captured-idle-stall.md; git show verifies Spec “Wait for the first real failed-admission capture from pair#791 diagnostics” and Plan “no implementation before a demonstrated failing transition”. Filed unclaimed per TL; BR-2 disputed with this measured host evidence for next-review disposition.
- Manual full-race wrapper run exceeded its default10m deadline in existing TestHarnessTTYFixtureConformance/qoder/1.1.60/menu.raw (CPU-active replay, no assertion failure reported). diagnosticlog full-race PASS102.837s. Complete wrapper package without race, with explicit30m limit, PASS196.078s before encoder correction; targeted new capture race tests had already passed. Final corrected-source verification remains pending.
- Old-head recorded affected lease-2648858986 was explicitly cancelled successfully because BR-1 requires a code change; it never ran and records no code failure. New-head recorded check will be queued after targeted correction verification.

- Corrected encoder: targeted capture race tests PASS2.933s; fresh bounded fuzz PASS400 executions/6.498s. Final complete wrapper package and preflights running; old-head package checks are preserved above with their precise limits.

### 2026-10-22 — SHIP; queued verification and landing

- Close round2 SHIP, no open findings: BR-1 addressed with exact byte accounting and a regression that the reviewer independently made fail using the pre-fix overlay; BR-2 withdrawn after corroborating the pinned tracker/details evidence. No gate bypass. Measured actual adopted0.64h (1 interval, 6 admitted sessions, candidate window12:23:59–12:58:44 local; metadata exclusions retained by the gate).
- Corrected-source full wrapper package PASS200.483s; complete diagnosticlog race PASS102.837s. Capture-specific race PASS2.933s; final fuzz PASS400 executions/6.498s; repolint PASS36.564s, artifactpath PASS21.673s; reader inventory passes. The full wrapper race attempt's default10m timeout remains explicitly recorded above and is not represented as a pass. Reviewer additionally ran complete wrapper package134.270s, full diagnosticlog/artifactpath, targeted race, fuzz342 and affected inventory.
- Corrected implementation is ecc76e29ec0cc2d0ca5252453ebe19883bdeca86; close evidence7ee1c86aab4bcfb5e14df0bfa8957c1da70eb810, codecomplete mirror5c24e78918ed72a4922e9ac9f452660b9e8c838c. Required recorded affected lease-402578803 remains queued on that exact code at last observed position11; no continuation/restart while own check is queued.
- Entered sdlc merge --yes promptly after SHIP: landing line position3, held by fleet-blocking #773 grant until14:32:08 local (its own integration/suite and operator shutdown-check sequencing). Preserve this runner and recorded check; do not bypass/revoke another issue's grant. Next: await recorded check and landing turn; resolve genuine failures if any, then let the same merge complete.
- Hub notified with PR546 and SHIP/pending-check state. Rollout note: capture is in pair-wrap, so existing wrappers need the normal authorized refresh/relaunch after landing; no live slot was driven/staged/restarted here. #810 remains unclaimed awaiting first actual failed-capture evidence, and #737 remains parked/unstaged.

- Authenticated TL dispatch988fcfe0-745c-418c-9482-00b3eb887790 authorizes starting independent pair#807 from main while this exact queued check remains alive. No queue priority requested/granted. Merge runner has ended with readiness refusal (CI/local check pending); CI subsequently all passed including native conformance3m55s. Required affected lease-402578803 remains queued, lastposition11. At every natural pause on #807 check retained exec session55235; once it passes, preserve #807, switch back to this issue branch and re-run sdlc merge --yes to land #791 first. This is not a continuation/restart; current agent/check remain alive.


### 2026-10-22 13:31 — resize lead and approved diagnostic addition

- Authenticated TL lead f0f48a53-7be6-4572-bbd6-080b43fac429: operator links a current stall to multiline tab-bar resize, always the visible slot. Add visible Claude pane replay through one/two/one bar rows and latest resize timing to capture. Approval d06e1506-63a0-4335-85b4-96087a40510b accepts one fixed-size latest-attempt/completion/geometry/changed/outcome observation under the existing terminal mutex, copied with screen generation; no policy or IO-under-lock change. Durable-plan final revision is the implementation contract. #807 is independently checkpointed/pushed and paused, not stacked.
- This addition supersedes the previous final-head acceptance claim; initial SHIP and all old check receipts remain historical. Do not land until the resize investigation is reported and updated code has fresh verification/review. #810 may proceed from a demonstrated failing native transition rather than waiting for the diagnostic to fire; no symptom reproduction claimed yet. Existing queued check remains alive pending a measured need to replace its code head.


### 2026-10-22 — resize addition evidence and updated verification

- Updated change-code: plan-quality round4 CLEAN (PQ-1 remains addressed), estimate-quality INFO requested explicit incremental scope. Derived increment after that gate: smaller-go-module design0.06/impl0.20 for transaction observation/coherent snapshots/native fixture; atlas design0.02/impl0.04; renewed review/verification design0.04/impl0.20; same1.15 design buffer and familiarity1.0 gives0.578h increment, total1.597h via set-estimate. No actuals invented; initial0.64h predates this addition.
- Behavioral red TestPeerWaitCaptureIncludesLatestResize failed all four outcomes with “forced refusal lost latest resize evidence”; fixed-size terminal-owned evidence and SnapshotDiagnostic made it PASS0.430s. Subsequent changed flag explicitly means actual dimensions differ; a same-size retry after abort still invalidates cursor by existing policy. SnapshotGeneration's API and every cursor/admission rule remain unchanged.
- Native ops:0 replay (57MB,105 resizes,563 sampled post-resize frames) PASS4.902s: actual94x39→38→39 paints restored complete/visible empty composers with bypass footer/update banner; no persistent complete-idle recognition failure. Committed sanitized native control fixture removes transcript glyphs and OSC payloads, preserving geometry/control bytes and footer/banner, with source ranges/hashes. Visible-pane replay derives body heights from the real status-bar planner and exercises ordinary/batch admission.
- Controlled resize then withheld output reproduces precise cursor-hidden refusal at0/15/30/300 simulated seconds; the retained native repaint immediately restores admission, PASS0.379s scratch reproduction. Native silence cannot be timed from retained logs: resize events lack timestamps; adjacent output markers span919s at candidate53541962 but guaranteed post-resize silence is0s. No input timing proves keystroke recovery. TL accepts this mechanism plus operator observations as enough for #810 (authenticated9a7378bb-68de-4a95-bb1c-129f827a79c1), directs claim and separate plan before code. #810 no longer waits exclusively for a live capture; no recovery effect is added to #791.
- Obsolete exact recorded affected lease-402578803 cancelled successfully due to this approved production change. Its runner ended context-cancelled while still queued; no test failure status was produced. Replacement final-head check is pending. Native-fixture reader inventory correctly refused the new reader; audited only tracked testdata raw/events and registered non_go_full_fallback before rechecking.

- New resize/capture/native replay and #735 controls PASS under race7.429s; bounded encoder fuzz PASS397 executions/6.820s. repolint27.888s and artifactpath20.956s pass; updated reader certificate check passes. During concurrent verification, generated transaction test initially overclaimed that aborting a same-size no-op must invalidate a complete screen; corrected its oracle to preserve the pre-resolution completeness, matching existing policy. Production changed-flag now separately reports actual dimension difference, including same-size retry after aborted resize. Complete wrapper package run remains pending; any earlier compiled test-oracle failure will be retained and rerun on corrected source before claiming full-package success.

- Complete wrapper run183.695s failed only the earlier compiled version of the generated transaction test (“abort confirmed geometry”). Its same-size no-op abort oracle had already been corrected and the final targeted race run passed7.429s; no production admission defect was found. A new complete wrapper run on committed cc59a202d409d523f11dd747e64c028744d1389a is running (session62920); do not count the earlier package run as a pass.
- Pushed resize implementation cc59a202d409d523f11dd747e64c028744d1389a and updated PR546. New recorded affected selection falls back to the full suite due to committed native fixture files, lease-1905342880, session96595, checkout /Volumes/TestScratch/local-checks/sdlc-local-2912210349, initial position15. Preserve it; no continuation/restart while queued. Earlier lease cancelled only because code changed, not to skip the line.
- Ready for renewed whole-issue close review on the approved expanded diagnostic scope; final full-package/recorded results still pending and landing remains gated. TL9a7378bb directs #810 claimed/planned independently from main while verification queues, then land #791 first once green. #807 remains checkpointed/pushed with implementation not begun.
