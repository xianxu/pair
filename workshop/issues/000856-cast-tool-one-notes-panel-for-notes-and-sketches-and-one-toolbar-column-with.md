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

### 2026-10-23 — Plan approved; mock ready

TL approved the joint plan (receipt bdff0bdf-86f8-4099-94c7-8de3cfac6ab4). Created synthetic mock-1 under the canonical plan.side/mocks directory with one shared fixtures.json. Served at http://127.0.0.1:51106/ by retained terminal 71609. Node syntax and diff-whitespace checks passed. A temporary native qualification reused the repository's headlessPage CDP fixture and passed non-skipped in 1.49s (package 1.677s): mixed entries, finished read-only label, unsaved/save, unavailable anchor, tab return, same-load app update, reload partition, tool reveal, and independent rectangle assertions for expanded/minimized Notes against the toolbar at 1280/768/320×800. Inspected screenshots; no panel/toolbar overlap. The temporary qualification source was removed after use; this is mock evidence, not production feature qualification. Next: send the mock URL to the hub for operator layout feedback, then implement only after acceptance and landed #850/#849. #850's existing lease remains queued without duplication.

### 2026-10-23 — Operator mock refinement

Applied authenticated feedback b5c92d0c-2c24-4098-bb02-e83190857da9 to mock and joint plan: replace thumbnails with compact sketch/annotation type icons; hover or focus a row glows its corresponding page geometry without scrolling. Missing anchors do not guess. Highlight clears when leaving/reconciling/minimizing/changing page. Coordinate #842/#843 with pair:7: #856 owns protected column/layout, while send-button and tooltip behavior remain their issues.

### 2026-10-23 — Icons and hover supersede initial thumbnail scope

The operator's latest design removes sketch thumbnails from the original Spec. Entries use type icons and page-side hover glow instead. The plan records the corresponding lifecycle, current-load identity and no-selection-mutation requirements. Production code still waits for the layout look.

### 2026-10-23 — Revised mock qualification

Non-skipped native Chrome check passed 0.63s (package 0.789s): four compact type icons and no thumbnail element; sketch hover glows only its anchor; annotation hover glows its box; hover does not scroll; missing anchor clears prior highlight; actual keyboard focus highlights; minimize and reload clear transient glow. Initial keyboard-focus probe exposed that property-assigned focusin/focusout handlers did not run; switched to explicit event listeners and reran with actual focus (no synthetic focus event). Temporary fixture removed after qualification. Mock server remains at http://127.0.0.1:51106/ with live reload. Node syntax and diff checks rerun. #850 retained check acquired its slot after 4776s and is now running; no duplicate or cancellation.

### 2026-10-23 — Actual tools and mock viewing boundary

Applied operator feedback 96da76d2-2673-40a2-b1aa-38ef59fe0206: mock now includes Pointer and Annotation (Note), names Sketch as Pen, retains Live/Pause, colors, conditional Undo/Redo and Done, and removes Stroke. Marking a synthetic card in Pen mode supplies a history action. Instructions say open in a non-cast Chrome window: the real cast injection otherwise overlays the mock; no capture/origin exclusion changes were made. Added a new-tab scenario starting empty. #857's target+loader+root scope and native same-URL new-target regression cover the live cross-tab leak.

### 2026-10-23 — Tool mock verification

Non-skipped native Chrome qualification passed 0.80s (package 1.036s): actual tool IDs and no Stroke action; Pen page action exposes Undo, Undo exposes Redo; Note page action opens an unsaved editor and Cancel discards its draft; same-URL New tab starts empty and returning restores four original rows; panel/toolbar rectangles remain separate at 1280/768/320×800. Temporary fixture removed after use. Node syntax and whitespace checks pass. Next: operator layout review in a non-cast Chrome window at the existing mock URL. Production code remains held.
