---
id: 000819
status: working
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: '1d7e8fe1a54cdf6a0137d5190d300d9fddfa7176' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-22T16:54:59-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# Collapse tab bar: a one-glyph-per-slot state strip next to +N

## Problem

Collapsed counts hide the checkout-state evidence operators can see on full chips. Expose one compact marker per hidden live slot while retaining the default one-row layout and exact group navigation.

## Spec

Operator request (10-22, after accepting pair#796's collapse tab bar): next to a collapsed group's `+N`, show **one
glyph per hidden slot** for its state, so the worktrees' activity reads at a glance without opening the menu.
The operator's sketch (using ⌥ for the branch glyph): `pair⌥  +9 ⌥⌥+-⌥*--⌥`, nine glyphs for nine hidden slots.

- Reuse the state glyphs the full chips already show (e.g. the branch glyph for a slot on an issue branch, the
  attention/bell and other existing markers); one cell each, in slot order; no new state vocabulary.
- Width: the strip counts toward the collapse planner's width budget (grapheme-correct, per pair#796's width fix); if
  it doesn't fit, fall back to `+N` alone (attention still forces `!`).
- Clicking anywhere on `+N` or its strip opens that group's menu (whole-chip hitbox).
- Live slots only, consistent with pair#796's counts.

## Done when

- A collapsed group renders one glyph per hidden live slot in order, using the existing chip glyphs (test).
- When the strip doesn't fit, the chip falls back to `+N` with no clipping mid-glyph (test).
- The strip is part of the clickable chip (physical-cell click test).

## Plan

- [ ] Derive one compact existing checkout-state marker per exact hidden live member in slot order.
- [ ] Admit complete strips only into spare width after the shared base packing; draw/hit-test count and strip together with existing owner styling.
- [ ] Verify projection, width, physical clicks, privacy/geometry, pending/live and counted-work bounds; stage for the operator, close and land.

Detailed design is on this issue branch at workshop/plans/000819-folded-slot-state-strip-plan.md; implementation awaits TL approval.

## Log

### 2026-10-22 — planning under authorized parallel queue

Claim/start-plan from clean main09c8964de39cdb5e6bd9db20601c232446af6949, independent of789. TL62d7df8e authorizes819 planning while789 isolated suite waits, returning for789 fixed review/merge on its result. Current789 lease-2426150993/session41982 remains untouched; no merge runner active. start-plan refresh hit the known seeded merge-check workflow change (ariadne557); saved /Volumes/TestScratch/tmp/pair-slot4/pair819-seeded-wpn5rb4w/merge-check.diff, restored only that generated file per existing TL instruction and retried. No workflow change is included here.

Initial issue-show ran against the older resting checkout and appeared empty. After start-plan caught up, origin/main and local details both contain TL's original sketch/spec; no missing-origin filing repair is needed. TLebda3dbd clarified checkout state and exact single-marker priority, neutral dot, retained dimming, whole-chip navigation and +N fallback. The durable plan recommends optional decoration after base packing, retaining the single measured renderer and confirmed click route (ARCH-DRY/PURE). Alternatives are recorded. No code changed; full-flow design awaits TL review before change-code. Estimate intentionally not derived before plan-quality.

## Revisions

### 2026-10-22 — state meaning and single-cell priority (TLebda3dbd)

Checkout state only, not agent idle/working. Reuse existing SlotGlyph evidence; choose dirty, diverged, ahead/behind, then issue-branch marker, in that priority. Clean/resting/unknown becomes a neutral dot. One cell for each hidden live slot in slot order; exact count membership, owner dimming and full count/strip group navigation remain. If a complete strip cannot fit, use +N alone; existing hidden-attention ! remains. This clarifies the original attention/bell example: attention actors stay explicit under796 and do not enter hidden strips.

### 2026-10-22 — TL plan approval (c4f6db71)

Approved optional strip allocation after base packing, left-to-right using spare columns: each strip fits whole or the group remains plain +N; detail never displaces active/attention/counts/controls. One hit span, unchanged multiline. Proceed to change-code; the real staged operator look remains required before landing. #789 retained snapshot suite is independent and will be resumed for its fixed review/merge on result.
