---
id: 000049
status: working
deps: []
github_issue:
created: 2026-06-04
updated: 2026-06-04
estimate_hours: 0.5
---

# Spell popup: bare-digit pick + stay in normal mode

## Problem

The `z=` spell-suggestion popup in draft nvim had two UX rough edges:

1. Pressing a number while the popup is shown inserted that digit into the
   buffer instead of selecting the corresponding suggestion. (Picking was only
   on `<M-i>`, mirroring the path/word completers.)
2. `z=` is a normal-mode trigger, but it had to enter insert mode to host the
   completion popup — and after picking a suggestion the user was stranded in
   insert mode rather than returned to the normal mode they started in.

## Spec

- While the `z=` popup is up, bare digits `1`–`9` pick the matching suggestion.
- Everywhere else (ordinary insert-mode completion, plain typing) bare digits
  stay literal — the spell behavior must not leak.
- Accepting *or* dismissing a spell suggestion leaves the user in normal mode.

## Done when

- Bare `1`–`9` pick the Nth suggestion while the spell popup is visible.
- Typing digits in a normal insert session still inserts the literal digits.
- After a pick (or dismiss) the editor is in normal mode.

## Plan

- [x] Gate the new behavior on a `spell_popup_active` flag set by
      `spell_suggest_popup`, cleared on `CompleteDone` / `InsertLeave`.
- [x] Add insert-mode expr mappings `1`–`9` (`spell_pick_digit`) that pick only
      when `spell_popup_active` + a visible menu; otherwise return the literal.
- [x] Label the spell popup with plain numbers (`1 the`) not `⌥1` — add an
      optional `label_prefix` arg to `indexed_items` (`'⌥'` stays for path/word).
- [x] On `CompleteDone`, when the popup was a spell popup, `stopinsert` so we
      return to normal mode.

## Log

### 2026-06-04

- All changes in `nvim/init.lua`. The popup-showing mechanism (`startinsert` +
  `vim.schedule(complete)`) is the pre-existing shipped path — only layered the
  flag, digit mappings, plain-number label, and the normal-mode handoff on top.
- Verified headless (`nvim --headless -u init.lua`): init parses (`luac -p`) and
  loads clean (exit 0); the `1`–`9` insert mappings + `z=` normal mapping
  register; `spellsuggest('teh')` → `the` as #1; regression guard — typing
  `abc123def` in a normal insert session yields literal `abc123def` (digit
  mappings don't interfere when no spell popup is active).
- Headless `vim.wait` can't reproduce nvim's interactive
  `startinsert`→`schedule`→`complete()` event ordering, so the full popup-pick
  round-trip needs a live manual confirm in draft nvim.
