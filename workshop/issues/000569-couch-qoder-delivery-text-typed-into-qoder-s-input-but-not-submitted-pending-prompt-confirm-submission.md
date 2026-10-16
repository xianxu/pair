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
- [x] Verify focused/race/full suites, document behavior and prepare for the TL-requested close review; live acceptance remains separately pending under the revision below.

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

### 2026-10-15 — uncertainty wording correction and suite replacement

An additional RED test showed generic quarantine refusal incorrectly called unknown residue definitely unsubmitted. The automatic-input reducer now retains its typed cause; known pending envelopes get the approved instruction while unverifiable residue preserves unconfirmed wording. Focused wrapper/core tests pass (2.929s/0.226s). The initial suite on 81624e3d acquired after 127s; it is superseded by this code correction and was cancelled during shell checks, not claimed passing.

During runner selection I incorrectly signalled Parley slot2 runner PID13969 despite its cwd identifying another slot. Notified parley.nvim:2 immediately by receipt b9785032-7aeb-4ef4-bdd5-a840cb4d8279 to inspect/restart its check; no checkout file changes. Then positively identified our own runner PID47867 by cwd pair-slot5, parent41373 and its retained log FD before separately signalling it. The mistake and process-ownership rule are recorded in lessons; no further peer-process actions.

### 2026-10-15 — process interruption disclosure and measurement preview

Parley confirmed the mistakenly interrupted ff50e564 suite exited130 without assertion failure or checkout changes, preserved the partial log and restarted the same head; it requests no further recovery actions (receipt 43e03727-be7d-4ca1-8145-b668bf807681). TL requested exact cause; response 43723a93-58f7-4fe7-89f5-c0bfa6d70fcc states kill -INT13969, first global matching command selection, absent parent-chain verification, and an unchecked batched cwd probe/kill. The lesson now explicitly requires tracing ownership to this slot's shell and prefers owned handles/groups; no guard issue or machinery changes were made.

Refreshed sdlc actual preview succeeds normally (0.59h candidate, not the final close value), warning and skipping incomplete old Qoder metadata instead of blocking. Close should measure/adopt fresh inputs; no no-actual bypass is currently needed. Latest focused races pass on corrected code: wrapcmd 4.210s, couchmessage 1.199s, couchcmd 1.972s. All PR #352 CI checks pass, native conformance 3m37s; required suite remains queued on 24653fa376710248082065aa8102d624743d1bbc.

### 2026-10-15 — quarantine precedence regression

An additional stateful regression showed that an open picker could turn a later quarantined delivery into Held instead of the required immediate refusal. RED receipt: held / blocked by a picker. Quarantine refusal now precedes picker holds (without any input effects), and focused Qoder/evidence/automatic-input/picker race tests pass 4.802s. Required suite on 24653fa3 was cancelled while still queued, after ownership verification: PID78181 command local-check, cwd pair-slot5, direct parent41373 matching this conversation's Codex invocation; probe inspected before separate SIGINT. No other runner signalled. The corrected code requires a replacement recorded suite; the older queued run provides no pass claim.

### 2026-10-15 — required suite passed; close review requested, landing held

Recorded required suite on a7b330872a80118ff1756edde74104805b07225d passed exit0 in 12m29s after 998s queued. Full shell/editor/environment and Go suites pass: couchcmd 76.459s, couchcore 435.549s, couchmessage 7.496s, wrapcmd 114.247s, sessionsync 17.111s. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/a7b330872a80118ff1756edde74104805b07225d/suite.log; SHA256 4d6a4760b008e2410324de2b8f660f8905cdfb00710ae5b02a2a4d21076e2895. All server checks pass on the code head, native conformance 2m26s. Main added only issue details #587–#591, integrated without a code change.

TL explicitly requested close review now, then HOLD (172b9532-1e96-4fc7-ab4b-a838f7469fea). The native idle, mid-turn and subsequent-delivery Done-when is still unverified: no stage, relaunch or landing. It remains TL-coordinated after #544/#584 in the next cheap window. This review assesses the implementation and automated evidence; it must not be read as native acceptance.

### 2026-10-15 — close round 1 corrections

Review returned FIX-THEN-SHIP with BR-1/BR-2 Important still open, so close explicitly refused finalization and requested another close after corrections. No codecomplete publication or native acceptance occurred. BR-1: the earlier 'press Enter' advice follows Qoder's newline remap, not submission. New routing-backed regression observed RED: the actual refusal's advised key emitted backslash-CR. Current advice is Alt+Enter or clear it, shared by receipt and supervisor via AutomaticInputRecoveryHint; the regression follows the chord through production translateChunk, verifies native CR and observes receiver consumption/quarantine release. BR-2: README now documents Qoder fresh-empty confirmation, exact-envelope bounded key-only retry, prompt quarantine refusal and Alt+Enter recovery; body replay is distinguished from key retry. Atlas agrees. Historical TL quotes remain untouched; the plan revision supersedes their Enter label without changing the keybinding. Focused normal tests pass (wrapcmd 3.236s, couchmessage 0.179s, couchcmd 2.152s). A corrected-head suite and another close review remain required; live/landing stay held.

### 2026-10-15 — corrected-head required suite passed

Recorded suite on 8186d8abd3e02528189a1809f3793f57d682bf07 passed exit0 in 14m49s after 1719s queued. It includes BR-1/BR-2 corrections. Full shell/editor/environment and Go checks pass: couchcore 562.167s, couchcmd 109.090s, couchmessage 8.372s, wrapcmd 150.264s, sessionsync 21.252s. Log /Users/xianxu/workspace/pair/.git/sdlc/local-checks/8186d8abd3e02528189a1809f3793f57d682bf07/suite.log; SHA256 56304faa904ba90b1cc393de569f97a8999bb27c34d72cc82068fb954f757dc9. Corrected focused races also passed (wrapcmd 5.399s, couchmessage 1.251s, couchcmd 3.388s); all CI checks pass, including native conformance 2m49s. Another close review must dispose BR-1/BR-2. Native idle/mid-turn/subsequent-delivery acceptance remains owed; TL's live/landing hold is unchanged.
