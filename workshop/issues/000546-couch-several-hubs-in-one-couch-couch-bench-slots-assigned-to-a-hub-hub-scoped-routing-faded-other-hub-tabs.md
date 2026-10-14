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
