---
id: 000465
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours: 3.66
card_mirror: '495720593b3184954a224e55499ee530fb4f468c' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T00:27:53-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: operator}
---

# couch: a slot blocked by a picker shows red in the tab bar, notifies the hub with the picker's text, and receipts say so

## Problem

When a slot's agent shows a picker that holds input (a permission prompt, the auto-mode classifier's question, an agent's question or plan approval), Couch correctly holds messages back, but nobody learns of it: the hub gets a generic waiting notice only after 10 minutes, the operator sees nothing, and a held message expires as `delivery deadline elapsed: waiting for picker to close`. 10-11/12: pair:2 sat on a question picker twice, and pair:3 was unreachable for an hour behind a latch armed by a permission prompt that auto-denied (pair#464).

## Spec

Couch reports facts; the hub and the operator decide (the operator's principle).
- **Red in the tab bar** (the operator, 10-12): while a slot is behind a picker that holds input, its tab-bar entry renders red, so the operator sees it at a glance. It clears when the picker closes.
- **Hub notice, early and specific:** after about a minute behind a picker (not 10), one notice per episode to the hub with what the picker is: its kind as far as the screen tells (permission prompt vs question), its title or command, and its options. Same mute file as pair#451's waiting notices.
- **Receipts say why:** a message held by a picker stays queued while the picker is up and reports `blocked by a picker: <title>` in `--message-status`, instead of expiring silently at the deadline; delivered when the picker closes, or expired with that reason.
- Uses pair#464's corrected picker state (the composer coming back closes it).
- Not in scope: answering pickers from the hub (a possible later verb for question pickers only; permission prompts stay the operator's).

## Done when

- A slot behind a picker turns red in the tab bar within seconds and clears when it closes (tests on recorded screens; operator smoke test).
- The hub gets one specific notice per episode after about a minute, with the picker's text.
- `--message-status` for a held message names the picker; delivery happens when it closes.
- The operator smoke-tested it before close.

## Estimate

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: issue-spec           design=0.8 impl=0.08
item: smaller-go-module    design=0.15 impl=0.14
item: smaller-go-module    design=0.1 impl=0.14
item: smaller-go-module    design=0.3 impl=0.25
item: tui-screen           design=0.5 impl=0.25
item: smaller-go-module    design=0.2 impl=0.14
item: atlas-docs           design=0.1 impl=0.05
item: milestone-review     design=0.0 impl=0.14
design-buffer: 0.15
total: 3.66
```

Items in order: issue/plan; `ReadPicker` parser; `Picker` on frame/registry/liveness; wrapper settle report + receipt hold; red chip + switcher row; watcher picker notice; atlas; close review.

> *Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.*

## Plan

Durable plan: `workshop/plans/000465-couch-a-slot-blocked-by-a-picker-shows-red-in-the-tab-bar-notifies-the-hub-with-the-picker-s-text-and-receipts-say-so-plan.md` (full flow: the code is over the quick-flow shell).

- [ ] Merge main once #464 lands; baseline green
- [ ] `couchmessage.ReadPicker` (one parser for kind/title/detail/options)
- [ ] `Picker` on session frame → registry → liveness
- [ ] Wrapper settle probe reports the picker
- [ ] Receipt: picker = hold, `blocked by a picker: <title>`
- [ ] Console: red chip + switcher row
- [ ] Watcher: picker notice at ~1m, 15s tick
- [ ] Atlas; full tests; live check; operator smoke test via ops:0

## Log

### 2026-10-12
- Design: the wrapper reports `Picker` beside `Settled` (#421 frames). Couch paints the slot red, the #451 watcher gets a picker reason with a 1m threshold, and the wrapper's receipt treats a picker as a hold (the #435 mechanism) with detail `blocked by a picker: <title>`. A single `couchmessage.ReadPicker` parser serves both the notice and the receipt (ARCH-DRY). Waiting for #464 (picker-state fix) to land before building.
