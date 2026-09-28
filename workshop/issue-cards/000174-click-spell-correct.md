---
id: 000174
status: open
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
github_issue:
---

# Click a misspelled word to correct it in insert mode

## Problem

Spell correction in the draft covers two of three cases:

- **While typing a word** — the as-you-type spell typeahead (`nvim/init.lua:1933-1943`)
  runs as a fallback in `run_completers`, after `path_complete` and
  `word_complete` decline, so a word matching nothing that is also misspelled
  gets a suggestion menu. No mode change.
- **Deliberately, in normal mode** — `z=` is already remapped
  (`spell_suggest_popup`, `:1891`) to the standard completion menu over
  `spellsuggest()`, picked with Tab/CR or bare digits 1-9.

The uncovered case is the common one: **a word you already finished, noticed was
red, and went back to with the mouse.** The completer chain runs on typing, not
on cursor movement, so clicking onto a misspelling in insert mode does nothing.
Correcting it means leaving insert mode for `z=`.
