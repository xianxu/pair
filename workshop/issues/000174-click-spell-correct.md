---
id: 000174
status: open
deps: []
github_issue:
created: 2026-09-02
updated: 2026-09-02
estimate_hours:
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

## Spec

**Clicking a misspelled word in insert mode opens the existing suggestion menu.**

**Hook it into the existing insert-mode `<LeftMouse>` map** (`:3571`), which
already handles the click-inside-a-visible-popup case and otherwise falls
through to a plain `<LeftMouse>`. That fall-through is the exact moment, and it
is a better site than a `CursorMovedI` autocmd on two counts: a click is never
typing, so no `changedtick` discrimination is needed; and it does not fire on
arrow keys, which are not part of this request. The popup-already-visible case
is handled in the branch above.

The map is `expr = true`, so the cursor has not moved when it runs — the
spell check must be scheduled after the click lands.

**Reuse `spell_suggest_popup`** rather than writing a second suggestion path
(`ARCH-DRY`). It already finds word bounds, checks `spellbadword`, calls
`spellsuggest`, and drives `complete()`.

### The two things that decide whether this survives

**1. Cursor displacement — fix as part of this, not after.** `spell_suggest_popup`
parks the cursor at end-of-word before `complete()` (`:1920`), because
`complete()` replaces from `start_col` to the cursor. So clicking mid-word to
fix a single letter both pops a menu and moves the caret. Dismissing must
restore the clicked column, or the gesture fights the ordinary reason the
operator clicked there.

**2. False positives on technical vocabulary.** The draft is prompts to a coding
agent — `couchcore`, `ptychild`, paths, flags — nearly all spell-flagged. Every
click landing on one pops an unwanted menu *and* (per 1) moves the caret. The
as-you-type completer avoids this by firing only when path/word completers
decline; a click has no prefix to filter on, so there is no equivalent guard.
**This is the acceptance risk, not a detail** — the feature is correct and still
unwanted if it fires on most clicks.

**Do not set `spell_popup_active`.** That flag makes bare digits 1-9 pick
suggestions, and it is deliberately restricted to the explicit `z=` gesture so
digits stay literal mid-type (`:1589-1596`, `:1943`). It clears on
`CompleteDone`/`InsertLeave`, not on "the operator resumed typing", so setting
it here would make typing `2` right after a click insert a suggestion instead of
a digit. Accept Tab/CR-only picking.

**Retreat, if (2) bites in use:** demote to a dedicated gesture. Double-click is
unclaimed. Right-click is the universal idiom but reverses a settled decision —
`mousemodel = 'extend'` (`:458-462`) deliberately disabled nvim's context menu
as "confusing inside the pair draft pane." That rejected a generic Copy/Paste
menu rather than a spell menu, so it is reopenable, but it is a reversal and
should be argued as one.

## Done when

- Clicking a misspelled word in insert mode opens the suggestion menu; picking
  replaces the word, dismissing leaves it intact.
- Dismissing restores the cursor to the clicked column, not end-of-word —
  asserted for a click in the middle of a word.
- Clicking a correctly-spelled word does nothing at all: no menu, no cursor
  movement, no flicker.
- Typing a digit immediately after a click-triggered menu inserts that digit.
- Clicking inside an already-visible popup keeps its current select-and-confirm
  behavior (`:3571`) — unchanged.
- `<CR>` semantics are unchanged in every existing case; the new popup goes
  through the same `cr_keys` decision (`:1979-1997`) without adding a state.

## Plan

- [ ] Schedule a post-click spell check in the `<LeftMouse>` fall-through.
- [ ] Cursor restore on dismiss, with the mid-word click test.
- [ ] Confirm no `spell_popup_active`, and pin the digit-stays-literal case.
- [ ] Live it for a few days and record the false-positive rate in `## Log`;
      demote to double-click if it fires on most clicks.

## Log

### 2026-09-02

Raised as "trigger the menu when the cursor moves over a misspelling in insert
mode". Narrowed on inspection: the typing case and the deliberate case are
already built, so only the click-back case is missing, and `<LeftMouse>` is a
better hook than cursor movement because it excludes typing and arrow keys by
construction.
