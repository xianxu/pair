---
id: 000791
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-22
estimate_hours: 1.019
collaboration-mode: collaboration
card_mirror: '1bfbb6063c31abd9a740abed11bfcbc57f03912c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T12:23:59-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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
- Recognition changes are tracked in an unclaimed follow-up waiting for the first real failing capture.

## Plan

Implementation: [durable plan](../plans/000791-couch-admission-plan.md), final diagnostic-only revision approved by TL.

- [ ] File recognition follow-up and checkpoint approved diagnostic-only scope.
- [ ] Forced-refusal tests first, bounded capture state/encoding and dispatch integration.
- [ ] Lock/failure/cap/growth/personality tests, atlas and normal source inventory.
- [ ] Recorded checks, mandatory close review, normal landing line.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only; calibration reported stale by estimate-source, so provisional. Derived after plan-quality CLEAN (round 2). Familiar Go/terminal/BufferedWriter seams, no novel stack/library needed. Thorough approved design applies x0.2 design discount and +15% design buffer. Two smaller Go units cover eligibility/encoding with generated/fuzz tests and production dispatch/lock/failure tests respectively. Implementation picks 0.5h primitive upper endpoint x0.4; familiarity1.0. Spec0.5x0.2, state0.3x0.2, integration0.2x0.2, atlas0.1x0.2, review0.2x0.2 design.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.10 impl=0.08
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.04 impl=0.20
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0.04 impl=0.20
design-buffer: 0.15
total: 1.019
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
