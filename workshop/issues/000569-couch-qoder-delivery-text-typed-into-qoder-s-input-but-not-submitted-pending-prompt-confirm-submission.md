---
id: 000569
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours: 1.924
card_mirror: '1d2707e14319bb29e117b2c8557388ade9b4d055' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-15T16:14:38-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
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

- [x] Add Qoder submit retry and fresh rendered confirmation through the existing peer-delivery reducer.
- [x] Bound unreconciled automatic-input recovery under the TL-approved policy and prove next-delivery recovery.
- [ ] Verify focused/race/full suites, document behavior and prepare for the TL-requested close review; live acceptance remains separately pending under the revision below.

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

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Provisional because the calibration source is flagged stale. Three smaller existing Go extensions (delivery reducer, automatic-input reducer/reporting, wrapper adapter) each use 0.2h design x0.2 and 0.4h implementation x0.4. One focused terminal-evidence/TUI concern uses 1h design x0.2 and 0.8h implementation x0.4; existing terminal model, composer parser and ANSI library cover the stack. Issue/spec: 1h x0.2 design and 0.2h x0.4 implementation. Docs: 0.1h x0.2 plus 0.1h x0.4. Review: 0.1h x0.2 plus 0.3h x0.4. Native discovery/conformance: 0.6h implementation x0.4. Familiarity 1.0, thorough-plan design buffer 15%; approximate range 1.2–2.8h, selected 1.924h (0.56 x1.15 +1.28).

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.04 impl=0.16
item: smaller-go-module design=0.04 impl=0.16
item: tui-screen design=0.20 impl=0.32
item: issue-spec design=0.20 impl=0.08
item: atlas-docs design=0.02 impl=0.04
item: milestone-review design=0.02 impl=0.12
item: real-api-discovery design=0 impl=0.24
design-buffer: 0.15
total: 1.924
```

### 2026-10-15 — implementation RED/GREEN checkpoint

New stateful receiver tests reproduced missing retry (one CR, zero acceptance), false turn-edge confirmation with occupied composer, and absent terminal reporting. They now pass with reducer-owned three-attempt budget, >=500 ms key-only retry under exact envelope/input ownership, and post-submit rendered-empty confirmation. Captured only the known native pending restart composer from source bytes [31229919,31238201), SHA256 6271e9fb9dc3faa89ab56bdff75e51220a090b9767fe68beff684a44fe4b24ad; the fixture proves the wrapped notice matches the complete expected layout. Terminal publication emits one authenticated quarantine report before polling stops; later deliveries cancel promptly with recovery advice, and normal render wakes release on later complete empty evidence. No late submit, repaste or deletion.

Core tests observed RED then GREEN, including 1,000 generated event sequences, short writes and terminal no-ops. Full couchmessage passed (7.458s), focused core race passed (1.238s), parser fuzz 63,024 executions. Focused wrapper/message/supervisor tests passed (6.993s/0.179s/0.916s), focused races passed (4.315s/1.226s/1.900s). Supervisor instruction/auth/dedup tests observed RED then GREEN and race passed. Matcher fuzz passed 14,738 executions; artifact inventory package passed 12.516s; make build passed. Additional focused regression controls and full required suite remain to be recorded; live idle/mid-turn pair:4 check is still owed, no native key-drop root cause or live acceptance claimed.

### 2026-10-15 — live check deferred by TL

TL directs required suite and close review on candidate 545ac416, then HOLD (receipt 172b9532-1e96-4fc7-ab4b-a838f7469fea). pair:0 is staged with #544; replacing it would discard that smoke candidate. pair:4 relaunch is blocked by #584's ownership guard. No staging, relaunch or landing now. TL will coordinate the combined stage/restart and native idle/mid-turn/subsequent-delivery checks in the next Qoder cheap window after #584 lands. The live Done-when remains owed and is not claimed by close review. PR #352 is open; required suite runs on 81624e3d, which adds only issue docs beyond candidate code.
