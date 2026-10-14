---
id: 000546
status: working
deps: []
github_issue:
created: 2026-10-13
updated: 2026-10-13
estimate_hours:
card_mirror: '8033390a8dde4737db481cce03a0848239f51b3a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-13T23:46:06-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
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

- [ ] Implement stable slot ownership, explicit hub launch and scoped authority.
- [ ] Project hub sections/reassignment/fading; verify supervision, persistence and unchanged default behavior.
- [ ] Run affected checks and full make -k, close review, green CI and land.

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
