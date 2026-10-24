---
id: 000842
status: codecomplete
deps: [pair#849]
github_issue:
created: 2026-10-23
updated: 2026-10-24
estimate_hours:
collaboration-mode: collaboration
card_mirror: '1ea07666e4b3d6bddc27ed6483f8b9d9e8cc2248' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T22:15:11-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 1.23
---

# Cast broadcast bar: a ⏎ send button when sketches or annotations are pending

## Problem

## Spec

**A ⏎ send button on the cast broadcast bar when sketches or annotations are pending (operator, 10-23).** In the
terminal cast (`LIVE ⏸ …`) and the Chrome cast (`Chrome ⏸ …`), when there are sketches or annotations pending
submission, a ⏎ button appears on the broadcast bar. Clicking it does exactly what Shift+Ctrl+Alt+Return does:
send the draft nvim content plus the pending sketches and annotations. The hotkey is hard to discover; the button
makes the action visible. It disappears when nothing is pending.

## Done when

- With pending sketches or annotations, the ⏎ button shows on both broadcast bars and clicking it sends the same
  payload as the hotkey (test); with nothing pending it's absent (test).
- A live look on a real cast (operator, briefly).

## Plan

Durable design: [Cast Send Button](../plans/000842-cast-send-button-plan.md).

- [x] Declare the existing send chord once and qualify pane/fullscreen routing without changing payload semantics.
- [x] Derive press-selection pending-feedback eligibility (per Revisions) and render independently clipped status Send spans.
- [x] Bind status clicks to the exact accepted actor endpoint and existing PairSendDOM transaction; prove stale/refusal controls.
- [ ] Update help/atlas, run inventory and recorded pre-close, obtain the hub-coordinated live look, close and land.

## Log

### 2026-10-23 — claim, source audit and #856 coordination

Claimed in pair:7 and ran bare sdlc start-plan; branch is based on main, no #739 commits stacked. #739 separately verified PR510 merged at9e6cc9d7a781c9b73acc0e3b3b72116c004b5a4b, tracker done/archive present; outcome reported to hub. Refresh completed; only this issue details changed, no generated vocabulary refresh to discard or file here. The seeded workflow refresh was deferred by the tool and not committed.

Read-only worker audits traced Shift+Ctrl+Alt+Return to nvim PairSendDOM -> existing DOM state/transport -> submissionManager/Board press-time selection. It does not submit terminal NoteBoard sticky notes; no terminal sketch-submission counterpart exists in current source. No new terminal-sketch gap is filed from a hypothetical feature. Existing draft transaction owns snapshot, busy suppression, request IDs, uncertain delivery, ack and clearing; a status control must enter that action rather than duplicate payload construction or read a draft file.

Read #856 joint plan and mock HTML/CSS, then viewed its actual synthetic1280x900 rendering via a disposable headless Chrome profile (exit0, no visible window/live profile). Pair:4 confirms browser-column layout ownership and no send/tooltip behavior assigned to #856. Source surfaces differ: #842's LIVE/Chrome groups are Couch status rows; #856's protected column belongs to the browser page. Hubddb54395 resolves the ambiguity: #842 owns only the two Couch groups and exactly the existing hotkey behavior/eligible feedback; no terminal-note transaction extension and no browser-column button. Terminal-only notes therefore cannot make a submission-eligible Send appear in this issue. #843 remains next, #850 is pair:4's item.


### 2026-10-24 — first implementation and targeted proof
- 2026-10-24: closed — BR-1: README now describes Chrome Send visibility, wider chord availability and terminal limitation, alongside help/atlas. BR-2: replacement control flushes and observes original AND successor; deliberate current-target misrouting fails with successor received 1 send chord, production restored unchanged; positive StatusSend -race passes 16.344s. Prior targeted Go and real headless nvim transaction checks passed; current inventory passes. Precise no-plan-check acknowledgement approved by hub563a3e8a: only remaining task combines queued final-head pre-close and externally scheduled live acceptance/landing. New --pre-close record is queued on 27f192b8930fe672722112a425c97ca21a5d774e; final pass and pair:4 coordinated combined morning cast stage still required before landing. No suite or live pass claimed; hub owns projects.; review verdict: SHIP
- 2026-10-24: flow upgraded quick → full — 176 added lines in code files (limit 100); an earlier round of this close already ran the full review

Change-code admitted the inferred quick flow; the binary will upgrade the close review if the real diff leaves the shell. Shared chord red failed because ESC[13;8u was undeclared, then workbenchshortcut/keyhelp tests passed after declaration, generated mapping, exact reservation-policy updates and removal of the duplicate manual map. make test-submission-transaction passed all existing real headless draft/DOM transaction tests. Focused wrapcmd/termcmd chord tests passed after updating the old four-reserved-chord expectation for the explicitly approved fifth chord.

Bounded Board/Cast subagent observed missing-API red, then green for HasSubmissionFeedback and related feedback/selection tests. It reuses selectionLocked and exact delivered sketch mask; no filesystem/browser/preview read. Render subagent observed absent-field red, then green for presence and width0–100 clipping/Unicode/non-overlap tests and all TestRenderStatus cases. Console red likewise exposed absent Send state; first integration wait identified a wrong visibility predicate (messaging registration/process proof rather than the existing live endpoint input capability). Corrected that predicate; no messaging authority is relaxed. Focused StatusSend now passes without any submission manager/preview, shows after a note and disappears on exact delivered mask wake; actual mouse path admits exactly one chord for plain left press and none for right/modifiers/release/drag/wheel/stale pane, thread, endpoint, cast or panel. One diagnostic run was stopped with SIGQUIT after46s to inspect the initial missing-span wait; not passing evidence or a flake.

Latest combined targeted command: go test ./cmd/internal/couchtty ./cmd/internal/wrapcmd ./cmd/internal/termcmd -run 'StatusSend|RenderStatusSend|Workbench|Chord|Shortcut|PassesThrough' -count=1 passed (couchtty4.016s, wrapcmd1.261s, termcmd1.980s). No visible window, stage or real-cast acceptance yet. Existing cast update worker observes candidate wakes and caches one bool; repeated paint consumes no Board/Store projection. Board calls happen outside Console.mu; accepted frame binds exact endpoint/cast and input FIFO admission is bounded.

Reader inventory audit: changed workbenchshortcut/shortcut_test.go retains its one repository input nvim/workbench_actions.lua plus local t.TempDir sidecars. No new path or scanner; nvim/** is already a non-Go full fallback input. Refreshed only its fixed_scanner certificate after inspection. Final inventory, recorded pre-close and mandatory close review remain.


### 2026-10-24 — race qualification and final-head queue

Added an actual active-pane replacement negative control (in addition to stale witness fields); the full StatusSend test under -race passed14.759s. Production unchanged from087446b70. Pre-close resolved affected correctly, then selected the full suite because nvim/init.lua is a declared full-fallback input; its older-head lease3817618192 was still queued, not running, and was cancelled through the owned SDLC lease before queuing this final test head. No completed suite is discarded, no parallel duplicate runner, no flake retry. Hub7d702e30 assigns acceptance to pair:4's combined morning #856/#857/#842 stage after a recorded pass and SHIP; pair:0 remains #851's. Do not stage alone.


### 2026-10-24 — recorded-check checkpoint

Final pushed executable/test head c6ec05d934e5dcddf003c10c9f25c123d4bf3a21. One owned pre-close runner remains: lease-3817920737, attempt3571627d7f82767774fde43939990c2f, log /Users/xianxu/workspace/pair/.git/sdlc/test-stats/inflight/3571627d7f82767774fde43939990c2f/run.log; tool session25514 at allocation, FIFO position8. It selected affected, then the declared nvim/init.lua full-suite fallback. Do not launch a duplicate. Next: read this run's terminal result; on pass run make test-affected-inventory and close's mandatory review (full review expected from actual diff; no-project since the hub owns projects). Hand pair:4 the verified SHIP head and smoke steps for the combined morning stage. No independent pair:0 stage or landing before that authorized acceptance. This checkpoint adds documentation only, no runtime/test changes after c6ec05d9.


### 2026-10-24 — close round1 corrections (BR-1 / BR-2)

Full review found no Critical or production defect, but returned FIX-THEN-SHIP with two Important findings still blocking ledger finalization: missing README coverage of the Send control/wider chord (BR-1), and an incomplete successor-endpoint oracle (BR-2). Added the user-facing README correction with inline review markers, and strengthened the mouse-route test to retain/flush/count/clean both old and successor child endpoints. Qualified it by temporarily replacing accepted-frame routing with current-pane routing: the replacement case failed specifically with "successor received 1 send chord(s)"; restored production source byte-for-byte afterward. This proves the corrected oracle rejects the misroute the old assertion missed. Positive corrected StatusSend race run passed16.344s. No production change. Added the prevention rule to lessons and a current PURE/INTEGRATION entity table to the plan, leaving superseded design history intact. Inventory passes.

The old recorded lease3817920737 was still queued and was cancelled through SDLC because the reviewed test fix needs a final-head record (tl-worker review-overlap contract). No suite had run; requeue only after this correction is pushed. The live look remains the combined morning stage with pair:4 (hub7d702e30); hub563a3e8a explicitly approves the precise no-plan-check acknowledgement for that external wait. #863 separately claimed, not started; do not switch this checkout while the next #842 review is active.


### 2026-10-24 — SHIP; parked for final recorded result and combined stage

Close round2 returned SHIP, high confidence; BR-1 and BR-2 addressed, zero new/open findings. The reviewer independently repeated the wrong-destination mutation and observed the successor assertion fail. SDLC published codecomplete bound to evidence de17daf3044e6d45c7e1335db2fc6237fd371ef7, then mirrored it; pre-checkpoint HEAD57e65c33cf0c8e43ddeaac7c147effce238dd2cf. PR581 remains unmerged; no live acceptance claimed.

One final recorded suite is still queued: lease-3238290542, attemptcfb07485d03d337d82c60275ecc8c0a9 on tested runtime/test head27f192b8930fe672722112a425c97ca21a5d774e, tool session37701, detached checkoutsdlc-local-951911802. Follow this retained runner/result, not the cancelled older leases. Close evidence and this checkpoint are docs-only descendants. Next: on its pass hand pair:4 the branch/head and operator steps for the approved combined morning stage; no independent pair:0 action. Only after that acceptance, run sdlc merge --yes and verify landing.

Hub843a4c5e made #863 urgent after #842 enters this wait: verify the production fresh-hub broker caller case first in the disposable shared harness. #863 is claimed/open, with no code/design stacked on this branch. Switching this slot to main-based #863 leaves this detached recorded suite and #842 branch intact.

## Revisions

### 2026-10-23 — scope clarification before design

Per hubddb54395, show Send only for feedback that the current hotkey would send, in the existing Couch LIVE/Chrome status-row group. Keep existing transaction data and permissions unchanged. Terminal notes unsupported by that transaction do not become eligible merely because they are counted on a bar. #856's browser-page column and its Notes panel do not absorb this work. No implementation has started; pending action-routing design and hub approval.

### 2026-10-23 — approved routing and separate terminal gap

Hub6491d0e0 approved the existing chord as the single routing owner, including wider agent/review/right-terminal availability and reservation of this one chord from fullscreen terminal apps; help and atlas must say so. Filed unclaimed #859 with source evidence for the missing terminal-sketch submission path, without claiming existing terminal sketches were lost. The durable plan keeps terminal NoteBoard notes ineligible, browser column untouched and payload authority in PairSendDOM. No implementation started; send the plan to the hub before change-code.

### 2026-10-23 — hub approval conditioned on #849 press-selection seam

Hub7dcaf909 approves the plan with a visibility correction: use #849's non-empty cast press selection (open notes at current revisions, finished labelled sketches), never acknowledged preview state. Preview may never complete on live pages. The durable plan's matching Revision supersedes all preview-based visibility/witness tasks; routing approval remains. Sent pair:6 receipt2270e3b0-cfcb-4260-9d26-b3d44009e271 requesting exact API, locking/invalidation and landing status. Wait for #849 on main before code; no stacking. No implementation started.

### 2026-10-23 — exact #849 seam received; landing prerequisite remains

Pair:6 c68595da reports #849 is not on main (pushed5cd9a3e78, pre-close queued). Cast.SubmissionSelection takes an in-memory Board selection atomically: open current-revision notes plus finished labelled sketches. Selection.PressSelection(delivered) applies exact delivered versions and the oldest-eight limit, but Store.DeliveredFeedbackVersions scans manifests on disk and must never run per paint/refresh for this button. Raw selection can retain a delivered sketch until the live display mask hides it. #842 will therefore reuse/add a Board-owned read-only non-empty projection using the same exact sequence/digest feedback-display mask, so acknowledgement cannot leave a false Send. Do not duplicate a durable delivered ledger or call the store.

SubscribeSubmissionCandidates supplies a coalesced wake for note create/edit/state, sketch label and clear; delivery mask changes also need explicit qualification. Observe the projection outside Couch locks; publish/read only its small result in presentation. Exact scheduling and file names will be reconciled against landed main at change-code, reusing an existing owner where available. Count actual event-driven reads/work rather than promise an unmeasured O(1) selection copy. Tests must include a permanently unavailable preview, delivered-sketch masking, current note revisions, unfinished sketches and cast replacement. No code or #849 branch changes requested. Wait for pair:6's verified landing receipt.

### 2026-10-23 — dependency landed and integrated

Verified PR578 MERGED at5d8fd7648caa64ecfcb6d9e2d20716a881d0397c and merged fresh origin/main into #842. Hubf896637c authorizes change-code. Plan Revision pins Board-owned nonempty selection plus exact live delivered-mask exclusion, observed through the existing cast update worker's candidate wake and cached in webcastState (no new worker, no per-paint IO/Board scan). No implementation before the gate.
