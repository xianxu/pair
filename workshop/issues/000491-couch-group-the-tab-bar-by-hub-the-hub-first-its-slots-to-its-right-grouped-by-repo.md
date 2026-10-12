---
id: 000491
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '64f4a461e11756084393b600d61b777d3fe34475' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-12
