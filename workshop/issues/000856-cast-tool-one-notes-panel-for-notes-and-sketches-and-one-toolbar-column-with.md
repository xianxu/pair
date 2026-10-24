---
id: 000856
status: working
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
card_mirror: 'a7e5c077af7623ef075ad9433c25a9e636d90c09' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-23T21:35:40-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:4
    worktree: /Users/xianxu/workspace/worktree/pair-slot4/pair
    repository: github.com/xianxu/pair
---

# Cast tool: one Notes panel for notes and sketches, and one toolbar column with tools revealed on toggle

## Problem

## Spec

**Cast tool: one Notes panel for notes and sketches, and one toolbar column (operator, 10-23, from their echod
annotations).** Agreed design, from ops `session.side/2026-10-23-24-cast-tool-learnings.md` (with screenshots
described there):
- **One Notes panel for everything:** each note and each sketch is an entry (a sketch entry shows its thumbnail, its
  note, add/edit note, and "show on page"); the panel is the only place notes and sketch notes are read or opened.
  Replaces the per-sketch buttons and pair#850's separate "Sketch notes (N)" button.
- **One toolbar, one column:** live/pause first; Sketch as a toggle that, when on, reveals the palette, Undo, Redo and
  Done below it; Undo and Redo hidden when they can't act. Nothing (notes, popups, status) is drawn over the toolbar.
- **Status messages** (such as "Pending feedback is out of date") show in the Notes panel's header, not as a floating
  box.
The exact positions get settled against a mock flow (the operator's note: "once we have the mock flow set up, we can
be a lot more precise how things are laid out"), so the plan starts with a mock of the layout for the operator to
react to before code.

## Done when

- Notes and sketches are entries in one panel; there are no per-sketch on-page buttons (tests).
- The toolbar is one column with the reveal-on-toggle behaviour, and no other cast UI overlaps it at common viewport
  sizes (tests), plus the operator's live look.

## Plan

Durable joint design: [Cast feedback workspace](../plans/000856-cast-feedback-workspace-plan.md). #857 remains a dependent issue on its own eventual main-based branch.

- [ ] Deliver the clickable layout mock and record operator approval.
- [ ] Implement the unified panel and protected one-column toolbar after #850/#849 land.
- [ ] Qualify, close, obtain the live look and land through SDLC.

## Log

### 2026-10-23

### 2026-10-23 — Joint design checkpoint

Claimed #856 and #857; started #856 from main while #850's final-source recorded check runs from its own detached snapshot. No #850 runner holds this checkout. #857 remains dependent and will get its own branch from landed main. Read the recorder/Board/UI seams and a bounded page-load survey; the durable plan records the state table, counted growth guards and codec/digest compatibility risk. No production changes yet. Next: commit the plan, send it to the hub tonight, then build the synthetic clickable layout mock for operator feedback. Preserve #850's retained check lease-3551124609 and return for merge when it passes.

## Revisions

### 2026-10-23 — TL's morning scope and #849 seam

Finished sketch labels are read-only for this delivery; unfinished labels keep Add note. Show on page scrolls/highlights only a confirmed current-load anchor and reports unavailable without guessing, navigation or reload. #857 is designed here but lands separately. Keep #849's noteDialog.unsaved getter and press-time saved selection; the preview stays informational. These decisions supersede the original sketch add/edit wording for finished labels. Exact panel placement is still subject to operator review of the mock before implementation.
