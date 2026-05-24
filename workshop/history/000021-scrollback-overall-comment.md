---
id: 000021
status: working
deps: [000018]
created: 2026-05-23
updated: 2026-05-23
related: [nvim/scrollback.lua]
---

# Overall comment affordance at end of scrollback

## Problem

#000018 lets the user drop inline `🤖[]` markers on individual lines via
`Alt+q`. After annotating, the user often wants to add an *overall* /
summary comment that isn't attached to any one line — "this whole turn
went sideways", "focus on X next". Today there's nowhere natural to put
that.

## Spec

Append a single styled affordance line at the very end of the
scrollback buffer:

    For overall comment, Alt+q on this line.

Rendered with a dim/italic highlight (link to `Comment`) so it reads as
an affordance, not transcript content. `nvim` virt_lines aren't cursor-
navigable, so this is a real (but visually distinct) trailing line.

**Behavior:**

- `Alt+q` on the affordance line opens the marker prompt with no
  inline-quote context (just a "(overall comment for this scrollback)"
  caption) and `default` = the previously stored overall comment, if
  any.
- Submit with text → store comment; rewrite the line to
  `Overall comment: <text>` (still dimmed).
- Submit empty → clear comment; restore the hint text.
- Cancel (Esc) → no change.
- `Alt+q` on the line again edits the existing comment (same edit-in-
  place semantics as inline markers).
- Visual-mode `Alt+q` whose selection touches the affordance line is
  rejected with a notice.

**Extraction:** on exit, the stored overall comment is appended to the
markers block (separated by blank line), with no `> quote` prefix —
it's a standalone block, not tied to a quoted line. If there are no
inline markers, the block ships as just the overall comment.

**Exit prompt:** the confirm message folds in the overall comment:

    Exit scrollback? 3 🤖[] markers + overall comment will be sent.

(or just "overall comment will be sent" if there are no inline
markers.)

## Plan

- [ ] M1: append affordance line on `BufReadPost`, apply
  `PairScrollbackFooter` line highlight; track `footer_row_by_buf`.
- [ ] M2: `add_marker_normal` branches to `add_footer_comment` when
  cursor is on the affordance row. Empty submit clears, non-empty
  stores + rewrites the visible line.
- [ ] M3: `add_marker_visual` rejects selections touching the
  affordance row.
- [ ] M4: `emit_pending` strips the affordance row from the marker
  scan and appends the stored overall comment to the shipped block.
- [ ] M5: `Esc` exit-confirm prompt counts inline markers + overall
  comment separately in the message.
- [ ] Manual verify: open `Alt+/`, walk to bottom, `Alt+q`, type
  "summary", Return → line updates. `Alt+q` again to edit. Submit
  empty → reverts. Inline marker + overall comment both ship in the
  draft sidecar.

## Log
