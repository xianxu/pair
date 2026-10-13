---
id: 000510
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '9d60fe58b1d923f5223b2a851fcbaa26c113ec42' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T17:14:14-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
---

# couch: a whole-repo fold's +N count can read as the count of the kept tab before it

## Problem

Advisory from #491's close review (round 2), filed at ops:0's request. When the tab bar is too narrow, #491 folds whole repositories, from the right, into one trailing `+N` until every kept tab (hub, active, red, belled) is in view (`fitChips`/`layoutChips` in `couchtty/reserve.go`). That count is appended after the last drawn chip, which is often a kept tab of another repository drawn with its full label:

`ops  ariadne  +2  pair:2  +3`

Here `+3` counts blog, pair and pair:1, but it sits right after `pair:2` and reads as pair's count. At some widths (30–31 columns in the review's probe) the fold count doesn't fit and is dropped, so slots vanish with no count at all. `fitChips`' comment claims a count always follows its own repository; that holds for the per-group collapse count since #491's round-1 fix, not for the fold count.

## Spec

- Every `+N` either directly follows a chip of the one repository it counts (the per-group collapse count, as today), or reads unmistakably as the row's overflow, not as the neighbour's count. For example, it is set apart at the row's end the way sections are (`│ +3`), or it stands where the folded groups were.
- An overflow count that can't be drawn means folding one more group, not dropping the count: hidden slots are always counted.
- One rule for both kinds of count in `layoutChips`/`fitChips`; fix the comment to match.

## Done when

- At every width from the full row down to the kept chips alone, the row's counts add up to the hidden slots, and no fold count sits directly after another repository's chip in a way that reads as its count. Covered by a width-sweep test over #491's hub model with a red `pair:2`.
- The operator sees it at a smoke test.

## Plan

- [ ]

## Log

### 2026-10-12
- 2026-10-12: unclaimed: filed for ops:0 as an unassigned follow-up to #491; nothing pending
