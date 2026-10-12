---
id: 000491
status: working
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '70696d29f10a2e207c80fa47d00a3bda730eb48f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T10:32:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "2850032f", done: "83c6e1bb"}
---

# couch: group the tab bar by hub — the hub first, its slots to its right, grouped by repo

## Problem

Couch is becoming a TL (the hub) plus the slots it leverages. The tab bar doesn't show that structure: the hub sits wherever its repo sorts, and slots are a flat list. The operator (10-12) wants lightweight grouping, designed so a later non-singleton Couch can show several hub sections.

## Spec

The operator's sketch: `[ops]  ariadne  :1  :2  blog  pair  :1  :2  [ops:1] …`
1. **The hub first:** the hub slot (the checkout Couch started from, pair#463) is the leftmost cell, bracketed (`[ops]`; `[ops:1]` if the hub is a numbered slot).
2. **Its slots to its right, grouped by repo:** each repo's name once, standing for its `:0` slot, followed by its numbered slots as `:1 :2 …` (`pair :1 :2` = pair:0, pair:1, pair:2). Repos in a stable order (alphabetical unless the operator says otherwise). Slots that don't exist aren't shown.
3. **Sections:** a hub plus its slots is one section. Today there is one hub, so every slot is in its section. The design must allow more sections later (a non-singleton Couch, or several TLs): another bracketed hub starts a new section. Decide in the plan what makes a slot a member of a hub when there are several (e.g. the hub that dispatched or prepared it), but build only the single-hub case now.
4. **Everything else unchanged:** each cell keeps its state rendering (red behind a picker, bell, fade, placeholder), the `Chrome ▶/⏸`/broadcast cells keep their place at the end, clicks and keys select the same slots, and slot-switching order follows the display order.
5. **Width:** when cells don't fit, degrade gracefully (e.g. collapse a repo's numbered slots into a count) without hiding a red slot.

## Done when

- The hub is leftmost and bracketed on startup; slots are grouped by repo with the repo name once and `:N` cells after it; state rendering and selection unchanged.
- Tests for the layout (one hub, several repos, missing slots, narrow widths with a red slot kept visible).
- The operator smoke-tested it before close.

## Plan

Durable plan: `workshop/plans/000491-couch-group-the-tab-bar-by-hub-the-hub-first-its-slots-to-its-right-grouped-by-repo-plan.md`.

- [x] Hub root reaches the Console (`SetHub` → `MenuState.Hub`)
- [x] `PresentThreads`: sections (hub row, then members grouped by repo), `hubOf` membership = the single hub today, `Hub` flag
- [x] Tab bar: hub leftmost of its section, no marking, ` │ ` between sections; switcher: hub first line, its section indented one level; `[…]` stays the active marker
- [x] Narrow widths: collapse `:N` into `+N`, then whole groups; red/hub/active/bell never collapsed
- [x] Atlas, README, skill; full suite; live check; operator smoke test via ops:0

## Log

### 2026-10-12
- Design: repository grouping (`pair :1 :2`) already exists in `RenderStatusRow`, ordered by `PresentThreads`, the one order source for the tab bar and the switcher. New: the hub (couchcmd's `hubWorktree`) goes to the Console, `PresentThreads` puts its group first, and the row brackets it. Conflict: `[…]` already marks the active chip; proposed reverse video for active. Broadcast/Chrome/capture cells lead today (#395 requires it), not at the end as the Spec says; they stay put. Sent the layout and narrow-width design to ops:0 before building.
- Operator review: the hub's repo is not grouped with it; the hub is a standalone section header (`[ops]  ariadne  :1  :2 … [ops:1]  ariadne:3  :4 …`). Hub as a concept: designated slots, slots assigned to one hub, a new slot inherits the hub of the slot it was started from. Build only the implicit singleton hub now, with membership as a function so designation and assignment slot in later. Plan Revisions updated.
- Operator, second review: no special hub marking. The hub is leftmost in its section on the tab bar (a separator between sections, none with one hub) and the first line in the switcher, with its section's slots and their notifications indented one level. `[…]` stays the active marker.
- Build: `Console.SetHub` → `MenuEventHub` → `MenuState.Hub`. The event re-sorts the installed inventory through a new allowlisted installer, `reorderMenuInventory`; the inventory-reader guard refused a direct read in `ReduceMenu`. `PresentThreads(rows, git, hub)` ends in `hubSection`: hub row first, the rest +2 indent. Membership is "every row belongs to the one hub" inside `hubSection` rather than a separate `hubOf` function, which nothing would vary yet.
- Narrow widths: the plan's step 3 folded plain tabs into `+N` too, which broke `TestAChipClippedToOneColumnIsStillClickable` (a cut-short tab must stay clickable). Narrowed: whole groups fold only while a kept chip (hub, active, red, bell) lies past the edge. `+N` draws unstyled, because the width test measures SGR bytes.
- Mutation checks: the hub group split in `chipGroups`, the red keep in `chipKept`, the kept-chip fold loop, and the +2 indent each fail a #491 test when reverted. A redundant `!a.Hub` reset (the hub already forms its own group) survived its mutation and was removed.
- The hub root is `hubWorktree`'s value, the same one `isHub` already matches against slot `WorktreeRoot`s in production, so its form matches the rows' paths.
- Full suite: `make -k test` (retention env scrubbed, default TMPDIR) exits 0, 102 Go packages ok, no FAIL lines. Built and verified; waiting for the operator smoke test in the next batched Couch restart (with #488, #489), per ops:0.
- Operator smoke test passed (TL relay: tab bar and switcher look right). Full suite green again on a59916070 after merging main (#488, #493).
