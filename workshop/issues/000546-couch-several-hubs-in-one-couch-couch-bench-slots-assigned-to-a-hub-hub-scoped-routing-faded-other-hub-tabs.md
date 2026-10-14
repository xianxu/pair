---
id: 000546
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours: 4.19
card_mirror: 'd648fecd289840c29b9bb8cddd036de56cdadd25' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T23:46:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# couch: several hubs in one Couch (couch . ../bench): slots assigned to a hub, hub-scoped routing, faded other-hub tabs

## Problem

Couch has exactly one hub (atlas/couch.md: "Several hubs: not built"). The operator wants to split kinds of work: feature work with the main TL hub (ops:0), and a second hub supervising a benchmark of ariadne changes (ops workshop/pensive/2026-10-13-02-pensive-model-eval-by-replay.md), both inside one Couch. Today a bench TL in an ordinary slot can't receive `--send-to hub` reports, nudges or resend prompts, and can't use hub-only commands.

## Spec

Operator's model (2026-10-14), refined with the TL:
1. **`couch . ../bench`**: each argument is a hub directory (a hub slot). No arguments behaves exactly as today (`.` is the one hub).
2. **Slot → hub assignment is a persisted fact** on the slot's record: inherited from the hub or slot that started it, survives restarts, changeable in the switcher. Assignment is per slot, not per repo.
3. **Hubs are isolated by default, as if two Couches:**
   - `couch --send-to hub` resolves to the sender's hub; pair#530's stuck-slot nudges and the restart resend prompt go to the slot's own hub. Explicit cross-hub sends (`--send-to ops:0`) still work.
   - Hub-only commands: read-only ones (`--peek`, `--actors`) see every slot; mutating ones (`--switch-agent`, `--relaunch`, `--set-model` from pair#544, reap/reboot) act only on the calling hub's own slots.
   - `couch --notify` is allowed from any hub; the desktop title names the hub.
   - Session-sync files are already per owner; no change.
4. **UI:** the switcher shows one section per hub (the renderer already starts a section at each Hub chip after the first). While a slot of one hub is focused, every chip of the other hub(s), including their hub, is drawn slightly faded in the tab bar.
5. **Shared on purpose:** one Couch process (a restart restarts all hubs; pair#528's continuity rules apply to each), the machine-wide test lock, and the terminal/web casts (singletons that follow the focused pane, whichever hub it's in).

## Done when

- `couch . ../bench` starts with two hubs; a slot started from the bench hub reports to it with `--send-to hub`, and its nudges/resend prompt reach the bench hub, not ops:0.
- A mutating hub-only command against another hub's slot is refused with a clear message; read-only ones work.
- Assignment survives a Couch restart and can be changed in the switcher.
- The switcher shows two sections; other-hub chips render faded while one hub's slot is focused.
- `couch` with no arguments is unchanged (regression tests).

## Plan

Durable plan: [Several hubs](../plans/000546-several-hubs-plan.md).

- [x] Implement stable slot ownership, explicit hub launch and scoped authority.
- [x] Project hub sections/CLI reassignment/fading; verify supervision, persistence and unchanged default behavior.
- [x] Run affected checks and full make -k. Close review, green CI and landing follow through the SDLC gates.

## Log

### 2026-10-13

- 2026-10-14 planning: claim/start-plan complete. Reused current authenticated workspace, broker, store-journal and console-operation seams in the design (ARCH-DRY). Stable slot ownership is separate from replaceable thread records to avoid #528 recovery changes. Hub notified that cli.go and narrow run.go startup wiring are unavoidable. Asked how bare couch treats assignments to an omitted hub; proposed retaining unavailable assignments, never silently reassigning. No implementation edits.

## Revisions

### 2026-10-14 — operator replaces slot ownership persistence with hub claims

The original Spec's persisted slot→hub field is superseded. The first command-line hub is main and owns everything by default, with nothing stored. Each started hub reads its own `.hub`: workspace-relative repository entries claim all its slots; `repo:N` entries claim one slot. Exact slot claims beat repository claims across hubs; equal-specificity conflicts use command-line order and warn. An omitted hub contributes no claims; bare Couch remains cwd as sole hub exactly as today.

Runtime `couch --move-slot repo:N[,…] --to-hub HUB` is ephemeral unless `--persist` is supplied. Operator or any verified hub TL may move slots; a hub may move its own or main's. Persistent moves add the target's exact claim and remove obsolete source exact claims; broad source repo claims remain for other/future slots. Moving back removes the exact override. No exclusions syntax, per-slot persisted field or unavailable-hub state. Switcher shows ownership and triggers the same move. Other Spec requirements remain.

Hub explicitly approved a separate startup helper reusing StartInteractive, CLI parsing and narrow run.go wiring, with no recovery/startup-policy changes in #528's core files. The active revised plan is appended under Revisions in the durable plan; initial draft is explicitly superseded, preserved for decision history. Receipts: 8624c810-08e2-4b4c-8606-b305400e94c2, dc433d5b-8668-446d-be63-09fd002bc92d, f6911942-31a8-4b9a-b87c-923f890f7e25.

### 2026-10-14 — approved CLI-first scope cut

Ops:0 approved plan a5c4842799590f5dfc3aaa09d93a55757810a603 (receipt 676cf232-af54-4067-b179-874215cf0fba), with the switcher move action and persist toggle removed. Those are now dependent follow-up pair#548. This overrides the active plan’s item 7, Hub move reducer/menu_hub.go entity, menu action changes and move-UI task: do not implement them in #546. Keep per-row hub display, sections and other-hub fading. CLI move-slot and all routing, authority, supervision, startup and persistence behavior remain in scope. The operator UI origin is future work in #548; #546’s remote CLI requires a verified hub TL. Proceed through full change-code review before implementation.

- 2026-10-14 plan review: PQ-1 clarified mode-specific idempotency (persist reconciles claims even after an ephemeral move); PQ-2 now names parser/resolver/edit/state/IO/authorization/routing/supervision/presentation function strategies. No code yet. Re-running change-code.

## Estimate

Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only. Calibration is marked stale by estimate-source; this is provisional ship wall-clock, not a deadline.

Decomposition: issue shaping 0.6 design + 0.2 base implementation; pure claims/ownership 1.0 + 0.8; five bounded Go extensions (claim files, startup, authority/routing, supervision, CLI operation) each 0.3 + 0.5; display-only TUI 0.5 + 0.7; docs 0.1 + 0.2; close review 0 + 0.5; integrated multi-hub fixture 1.0 + 1.5. Thorough approved specification discounts implementation-phase design by 0.2; shaping remains undiscounted. Reuse durablefile, operation queues, message fixtures and console renderer; no external library replaces the domain-specific resolver, and there is no novel external API discovery. Familiar stack multiplier 1.0; implementation values below are already scaled by 0.40. Design subtotal 1.42; implementation subtotal 2.56; 15% design buffer yields 4.193 hours, rounded 4.19. Verification is included in each component and the final integration fixture; no extra test surcharge.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec design=0.6 impl=0.08
item: greenfield-go-module design=0.2 impl=0.32
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: smaller-go-module design=0.06 impl=0.20
item: tui-screen design=0.10 impl=0.28
item: atlas-docs design=0.02 impl=0.08
item: milestone-review design=0 impl=0.20
item: api-integration design=0.20 impl=0.60
design-buffer: 0.15
total: 4.19
```

- 2026-10-14 implementation checkpoint: change-code admitted full flow after CLEAN plan review and INFO estimate review (authority/integration allowances may be low). Added pure claim parser, immutable ownership projection/revision transitions and durable edit planner. Initial TestHub run failed on absent APIs; focused suite now passes (0.220s). Additional RED found already-satisfied exact claims moving past comments; GREEN preserves their position. Batch authority, independent runtime/persistence, copy isolation and restart fallback covered. Focused race verification running. #548 details published and claim released.

- 2026-10-14 implementation checkpoint: file adapter covers missing/malformed/symlink configuration, expected-content conflicts, partial writes and lost write acknowledgments. Explicit startup uses the existing queue through declared start-hub, with bare invocation structure unchanged. Added typed move-slot transport/receipt identity (destination plus persistence), authenticated queued dispatch and revision-fenced runtime publication. Focused hub/parser/operation documentation tests pass across couchcmd/core/message (0.768/0.232/0.183s); move persistence/restart/failure tests pass (0.500s). Family filtering preserves global ambiguity checks and exact cross-hub routes (RED/GREEN). UI subagent full couchtty PASS 40.930s and focused race PASS 1.474s; supervision/notification focused race PASS 9.655s. Integration/conformance, complete package runs and full make -k remain outstanding.

### 2026-10-14 — active acceptance after approved revisions

This replaces the original Done-when inheritance/persisted-slot/switcher-move
clauses. Started hubs load their own bounded `.hub` claims; exact wins over broad,
then CLI order resolves equal specificity with a warning, then main is fallback.
Unclaimed new slots belong to main. CLI moves are ephemeral by default, persistent
only on confirmed claim writes. Only a verified TL moves own/main slots; no new
operator-identity wire bypass. Switcher ownership display/sections/fading remain;
its move action is #548. No thread-record hub field or recovery-policy changes.
The original routing, mutation isolation, shared singleton and bare-launch
acceptance clauses remain, interpreted using this ownership model.

- 2026-10-14 integration checkpoint: stateful socket tests cover own/main/third-hub move authorization, batch refusal, queued incarnation/revision changes, source changes after prepare, receipt intent conflicts and lost-reply recovery. CLI output names destination and persistence; uncertainty names operation-status. Generated edit sequences cover all 81 combinations of three hub claims and destinations. Focused all-hub races passed couchcmd 27.843s/core 1.304s/message 1.220s; isolated startup/layout shuffle+races passed 17.031s; build passed. First broad package run caught in-progress fixture timing/layout assumptions plus new operation dispatch/inventory omissions; those are corrected, not waived. First make -k entered Go while the trusted execution-guard helper was still absent and reported its compile error; remaining checks continue, then full verification will be rerun from a committed tree. No success claimed for either broad run.

- 2026-10-14 startup ordering: chain extra-hub queue admissions after each prior adoption, preserving explicit argument order even with concurrent queue workers. Stateful queue/adoption test and startup integration pass (4.158s). Main merge 3c23f136f7da692a9b916ed5b90b0cdbd78c6a74 contains #544 details only; no set-model implementation has landed to reconcile. No #528 startup/recovery-policy files changed.

- 2026-10-14 verification: full couchcmd package now PASS (49.006s) after updating its obsolete public-help expectation. Clean build reports vcs.revision=2b0c2457c39bd33a7a9f4bfef74141169cdf94c2 and vcs.modified=false. Parser fuzz PASS (11,348 executions, two workers); specification-bound checks confirm 128 runtime overrides refuse an additional batch without a partial move and name restart recovery, and the 1024-line claim budget names file repair. Final make -k rerun has entered the machine gate and is executing.

### 2026-10-14 — verification complete and close handoff

Full `make -k test` exited 0 on committed HEAD 699e3ec42 (log `/tmp/pair-546-full-final.log`); couchcmd 57.820s and couchcore 240.733s. All shell, editor, inventory and Go targets passed. Focused races, full couchtty, parser fuzz and clean build evidence above remain applicable. Final fetch found no new main commits. The isolated terminal integration uses real console/start/dispatch paths with stateful fake external processes and private stores; this is not a live operator smoke. Active implementation and verification are complete; close review, CI and landing remain gate-owned. Plan checklist wording now separates completed verification from those pending gate actions, and records the approved CLI-only move scope.
