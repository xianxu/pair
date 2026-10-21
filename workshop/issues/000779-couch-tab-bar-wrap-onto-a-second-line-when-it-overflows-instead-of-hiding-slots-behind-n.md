---
id: 000779
status: working
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'acde1c7c4bcef86fcee82625f62d87d799503e2a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-21T16:42:14-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# Couch tab bar: wrap onto a second line when it overflows, instead of hiding slots behind +N

## Problem

## Spec

Operator report (10-21 13:08, ops session file item 12, screenshot `ops/assets/session/2026-10-21.13-08-21.150.jpg`):
Couch's tab bar ran out of space. With ops, ariadne, blog, pair, parley, you-decide, ops-echo and echo groups and their
slots, the row overflows and the tail collapses into "+3", so slots are hidden.

Wrap the tab bar onto a **second line** when it doesn't fit (like text wrapping), instead of truncating to "+N":
- Wrap at group boundaries where possible (keep a repo's slots together); break inside a group only if one group alone
  is wider than the line.
- At most two lines; if two lines still don't fit, keep today's "+N" overflow on the second line's end.
- The Chrome/cast status group stays where it is (left of line one).
- The rest of the layout (panes, footer) shrinks by the extra line only when the second line is in use.
- Resizing the terminal re-wraps without flicker; the active tab stays visible.

## Done when

- Render tests: fits on one line (unchanged), needs two lines (wraps at a group boundary), one oversized group, and
  overflow beyond two lines (+N), at several widths; active tab always visible.
- Pane geometry accounts for the second line (no overlap), with a test.
- Attention chips take priority over ordinary chips; any hidden attention is explicit in +N !. At physically impossible widths, the active chip remains clipped but clickable and a non-fitting whole overflow marker is omitted.
- Both-row input/anchors and broadcast LIVE/privacy/annotation guards derive from the same accepted chrome height, including resize and private switcher frames.

## Plan

- [ ] Deliver the approved two-row layout through shared compositor, Console and broadcast geometry, with render/interaction/native pins; see workshop/plans/000779-tab-bar-wrap-plan.md.

## Log

### 2026-10-21

Claimed and start-plan completed in pair:4 after pair#757 landed (PR532). Includes the manual #757 archive repair, ariadne#532 class: commit a2d91a471ac3bb0b10984194b0990fbde8fdb71f moves two unchanged browser evidence logs from noncanonical plan.md.side into workshop/history/plans/000757-mock-to-real-plan.side. Both renames are100%; no content changes. TL receipt c1d5a9d6-7e31-44f6-a670-5823fcadadf7 explicitly approves riding with this issue and requires this note in the PR description.

Initial read-only audit: one-row assumptions exist in couchtty rendering/geometry/input/anchors, terminal composition/presenter resizing, and broadcast LIVE/pointer checks, private-frame filtering and annotation guards. This leaves the quick-flow code shell; prepare one atomic full-flow plan with pure packing and a shared trusted chrome-height contract before implementation. No product code written. Geometry and packing audits delegated read-only under AGENTS section3; plan approval will go through the hub. Private #757 GOCACHE removed; new Go calls use the repaired machine default on main SSD.

## Revisions

- 2026-10-21: TL c5c04f63 approves full-flow shared trusted chrome height, row1 attention badge, final-row notices and clipped-but-clickable active at absurd widths. Attention must be prioritized over ordinary chips; overflow explicitly marks hidden attention (+N !). Durable plan records the full consumer sweep and awaits TL approval.
