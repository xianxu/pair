---
id: 000842
status: working
deps: [pair#849]
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '7fa2d1a578fcca44f477dac7b9681b350f7eaa23' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T22:15:11-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
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

- [ ] Declare the existing send chord once and qualify pane/fullscreen routing without changing payload semantics.
- [ ] Derive acknowledged pending-feedback eligibility and render independently clipped status Send spans.
- [ ] Bind status clicks to the exact accepted actor endpoint and existing PairSendDOM transaction; prove stale/refusal controls.
- [ ] Update help/atlas, run inventory and recorded pre-close, obtain the hub-coordinated live look, close and land.

## Log

### 2026-10-23 — claim, source audit and #856 coordination

Claimed in pair:7 and ran bare sdlc start-plan; branch is based on main, no #739 commits stacked. #739 separately verified PR510 merged at9e6cc9d7a781c9b73acc0e3b3b72116c004b5a4b, tracker done/archive present; outcome reported to hub. Refresh completed; only this issue details changed, no generated vocabulary refresh to discard or file here. The seeded workflow refresh was deferred by the tool and not committed.

Read-only worker audits traced Shift+Ctrl+Alt+Return to nvim PairSendDOM -> existing DOM state/transport -> submissionManager/Board press-time selection. It does not submit terminal NoteBoard sticky notes; no terminal sketch-submission counterpart exists in current source. No new terminal-sketch gap is filed from a hypothetical feature. Existing draft transaction owns snapshot, busy suppression, request IDs, uncertain delivery, ack and clearing; a status control must enter that action rather than duplicate payload construction or read a draft file.

Read #856 joint plan and mock HTML/CSS, then viewed its actual synthetic1280x900 rendering via a disposable headless Chrome profile (exit0, no visible window/live profile). Pair:4 confirms browser-column layout ownership and no send/tooltip behavior assigned to #856. Source surfaces differ: #842's LIVE/Chrome groups are Couch status rows; #856's protected column belongs to the browser page. Hubddb54395 resolves the ambiguity: #842 owns only the two Couch groups and exactly the existing hotkey behavior/eligible feedback; no terminal-note transaction extension and no browser-column button. Terminal-only notes therefore cannot make a submission-eligible Send appear in this issue. #843 remains next, #850 is pair:4's item.

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
