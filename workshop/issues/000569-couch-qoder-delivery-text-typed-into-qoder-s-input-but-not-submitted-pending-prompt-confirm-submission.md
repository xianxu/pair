---
id: 000569
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '2e8caee3602676ed455179c518bea7aabf0f4554' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T16:14:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# Couch → Qoder delivery: text typed into Qoder's input but not submitted (pending * prompt); confirm submission

## Problem

2026-10-15 ~07:55 PDT: a hub message to pair:4 (Qoder CLI) was typed into Qoder's input box but never submitted. The
pane showed it as a pending `* [Couch peer from ops:0; delivery b2fb1254…]` prompt while Qoder sat idle (no generation
running), so the worker never received the hub's answer and stalled. Earlier deliveries to the same Qoder session were
submitted and acted on, so it's intermittent; it likely happens when the delivery lands while Qoder is busy (Qoder
queues typed input as a `*` pending prompt) and Qoder doesn't auto-submit the queue when its turn ends, or the submit
key is dropped by a paste-burst guard (compare #544's Codex paste-burst fix).

## Spec

- Reproduce: deliver to a Qoder slot while it's mid-turn, and while it's idle; record whether the text is submitted.
- Qoder's receiver profile must confirm submission (the prompt leaves the input box, or a turn starts) and retry the
  submit key if not, the same way the Claude and Codex profiles confirm.
- Delivery status must not report "delivered" for text that's only typed.

## Done when

- A live Qoder conformance check: deliveries mid-turn and idle are both submitted and acted on.
- A unit test of the Qoder profile's confirm-or-retry logic over captured screen states (pending `*` prompt vs. empty).

## Plan

- [ ] Add Qoder submit retry and fresh rendered confirmation through the existing peer-delivery reducer.
- [ ] Bound unreconciled automatic-input recovery under the TL-approved policy and prove next-delivery recovery.
- [ ] Verify focused/race/full suites, document behavior and complete TL-coordinated idle/mid-turn live checks on pair:4.

Durable plan: [Qoder submit confirmation](../plans/000569-qoder-submit-confirmation-plan.md).

## Log

### 2026-10-15

### 2026-10-15 — initial investigation

Claimed/start-plan in pair:5 after pair#573 landed. Live peek confirms the restart notice 44c73545-5fed-4759-9c68-fd4865c10085 is still in the `*` Qoder composer. Its original receipt is unavailable after restart, so no exact historical receipt outcome is inferred. The shared delivery reducer has one CR submit and confirmation, with no retry for any receiver; #544 is a separate native-drive pacing fix. Occupied composers are excluded from the existing automatic-input tripwire, explaining why this residual state does not alert or release. Qoder's native first-CR suppression cause remains unproven. Proposed retry/confirmation and terminal-residue policy is in the durable plan; TL clarification requested by receipt 791e44c3-d7a6-4e49-b3cb-de6f3240b303. No live input or staging performed.

Planning catch-up initially refused because weave refresh sees the unrelated untracked pensive file as dirt. Refreshed the slot-private ariadne substrate directly (its own clean checkout), preserving the pensive in place; rerun start-plan succeeded. No operator checkout modified.

## Revisions

### 2026-10-15 — explicit bounded recovery

TL dispatch adds the repeated later-delivery expiry to the issue's purpose. Planned coverage includes bounded key-only retry, fresh rendered acceptance evidence, explicit unreconciled-input escalation and positive-empty next-delivery recovery. Automatic clearing or execution after an expired delivery is not assumed authorized; the terminal recovery policy is being agreed with TL before implementation.

### 2026-10-15 — TL approval

TL approved plan 967b13cd and the terminal policy (receipts c4ac5b82-afee-4dfe-8a1e-70a9c5c367db and 84d5bf3e-f2ba-4f7b-a483-c65b528689f6). Retries require exact complete-envelope ownership in addition to fresh rendered evidence and existing input fences; no automatic deletion or late execution. Live idle/mid-turn checks on pair:4 must be coordinated with TL before landing, ideally before 17:00 PDT. Baseline focused Qoder/delivery/automatic-input tests pass (wrapcmd 3.461s, couchmessage 0.267s); the missing behavior needs new RED tests.
