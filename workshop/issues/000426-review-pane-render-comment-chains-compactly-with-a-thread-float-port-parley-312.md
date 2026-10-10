---
id: 000426
status: open
deps: [parley.nvim#312, ariadne#316]
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: 'cc5681722319be5dc662eecb9a5b1686b7f30029' # card fields mirrored from issue-cards; edit via sdlc
---

# Review pane: render 🤖 comment chains compactly with a thread float (port parley#312)

## Problem

The review workbench pane (`nvim/review/`, atlas `review-workbench.md`) shows
🤖 review markers raw. A long comment chain (`🤖<X>[q]{a}[q2]{a2}…`) floods
the line and is hard to read. parley#312 made these chains first class in
parley's markdown buffers. Pair's review pane should render them the same way,
so a review reads the same in both tools.

## Spec

Port parley#312's behavior (parley `lua/parley/comment/`; atlas
`parley.nvim/atlas/modes/review.md`):

- **Compact display** (extmark conceal, `conceallevel=2`):
  - every turn collapses to `[…]` / `{…}`, except a last human turn, which
    stays visible and editable inline (`🤖[…]{…}[this is ok]`);
  - `🤖<X>…` highlights X with the 🤖 and `<>` hidden;
  - `🤖~D~{N}` shows D struck through plus `{…}`;
  - turn brackets are colored by speaker.
- **Edit protection:** the cursor snaps off hidden bytes. Insert mode only
  allows these points: before the 🤖, inside the anchor, inside the last human
  turn, after the marker. A marker broken by a stray edit paints visibly
  (`ParleyReviewBroken`-style).
- **Thread float:** `<CR>` on a marker opens the chain like a chat: `💬: `
  human turns, `🤖: ` robot turns, unprefixed lines continue a turn. `:w`
  writes it back as one line, `q` saves and closes, `:q!` discards. Save
  re-parses the joined marker and refuses unbalanced brackets.
- **Single-line grammar** (ariadne#316, review-convention §3): a newline inside
  a turn is `<br>`, a literal `<br>` is `\<br>`, and a backslash run before
  `<br>` doubles (odd = literal, even = newline). Only turn text is decoded;
  anchors stay verbatim.

Open questions, to settle at start-plan:

- **Multi-line markers.** Pair #89 made `markers.lua` (`spans_multiline`) and
  `reconcile.conflict_marker` multi-line aware. ariadne#316 now says markers
  are single-line. Should the agent's conflict markers move to `<br>`, with
  legacy multi-line markers painting as broken?
- **Escaping.** Pair's `marker_codec.lua` already escapes `>`, `]` and `\`
  inside quotes. Reconcile it with the `<br>` codec, keeping one codec per
  delimiter set.
- **Reuse vs port.** Pair ports parley's parser (`markers.lua`) rather than
  depending on parley. Decide whether to port `comment/view.lua` (pure layout
  and snap) and `comment/thread.lua` (pure float layout) the same way.
- **Other viewers.** Should the scrollback and change-log viewers
  (`nvim/annotate.lua`) get the compact display too, or only the review pane?

## Done when

- In the review pane, a chain of 3 or more turns displays as
  `🤖[…]{…}[last human turn]`, checked on a real render (a pty
  `screenstring()` probe, not just extmark assertions; see parley's lesson on
  treesitter link conceal hiding `[text]` brackets).
- `<CR>` on a marker opens the thread float. A reply saved with `:w` lands as
  one `<br>`-encoded line, and the agent's next round reads it.
- The codec and float layout round-trip under property tests that draw input
  from their own delimiters.

## Plan

- [ ]

## Log

### 2026-10-10
