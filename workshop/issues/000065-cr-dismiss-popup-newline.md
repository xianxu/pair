---
id: 000065
status: done
deps: []
github_issue:
created: 2026-06-18
updated: 2026-06-18
estimate_hours: 0.5
---

# Return key dismisses completion popup + inserts newline when nothing selected

## Problem

In the draft pane, when a completion popup is showing and the user has NOT
Tab-selected an item, pressing Return does nothing useful — the keystroke just
closes the menu and the expected newline never lands. The draft's `<CR>` is
fundamentally "insert a newline" (send is `Alt+CR`), so a swallowed Return is a
real papercut while typing.

Root cause: `nvim/init.lua`'s insert-mode `<CR>` expr map returned a bare `<CR>`
for the popup-visible-but-nothing-selected case:

    return (pum_visible() and pum_has_selection()) and '<C-y>' or '<CR>'

A bare `<CR>` fed while the pum is up only dismisses the menu — the newline is
eaten. This is the well-known nvim gotcha under `completeopt=...,noselect`.

## Spec

`<CR>` in the draft must ALWAYS produce its intended effect:

- popup not visible            → `<CR>` (plain newline) — unchanged.
- popup visible + a selection  → `<C-y>` (accept highlighted item) — unchanged.
- popup visible, no selection  → dismiss the menu AND insert a newline. Treat it
  as "I didn't pick anything; keep what I typed and break the line."

The fix is the canonical idiom: feed `<C-e>` (cancel completion, preserving the
typed text exactly) followed by `<CR>` (now processed as a normal newline).

## Done when

- Return inserts a newline when the popup is up with nothing selected, keeping
  the typed text on screen.
- Tab-select-then-Return still accepts the highlighted item (no regression).
- Return with no popup is an ordinary newline (no regression).
- Regression test asserts the expr output for all three states.

## Plan

- [x] Extract a pure `cr_keys(visible, has_selection)` decision in `nvim/init.lua`,
      add the `<C-e><CR>` branch, route the `<CR>` map through it, expose it as
      `_G.PairCRKeys` for testing.
- [x] Add `tests/cr-newline-test.sh` (autopair-style headless driver) asserting
      the three states; wire it into `Makefile.local` (`test-cr` + `test`).
      Deliberately the headless-driver shape, not a bare `nvim -l` pure test:
      `cr_keys` lives in monolithic `init.lua` (can't be `dofile`'d standalone
      like `slug.lua`/`scrollback.lua` without its side effects), and booting the
      real `init.lua` lets the test ALSO assert the live `<CR>` maparg callback
      routes through `cr_keys` (returns `<CR>` headless, where no popup is up) —
      proving the wiring, not just the extracted function.
- [x] Run `make test-cr` (5/5 green). Live-dogfood deferred to operator: an
      end-to-end keystroke test needs an interactive pum, which a non-interactive
      agent can't drive and headless pum rendering is unreliable here.

## Log

### 2026-06-18

- Confirmed current map at `nvim/init.lua:3301`; `completeopt` is
  `menu,menuone,noinsert,noselect` (`init.lua:443`), so nothing is ever
  auto-selected — the no-selection Return path is the common case, not an edge.
- Behavioral headless test of popup rendering is flaky (pum needs a UI;
  feedkeys timing). Following the repo's autopair-test precedent: assert the
  returned expr-string, trust nvim's `<C-e><CR>` semantics, verify newline live.
- Implemented `cr_keys` + `_G.PairCRKeys`, routed the `<CR>` map through it,
  corrected the now-inaccurate atlas line (#254) that claimed the old "(else
  newline)" behavior. Updated README/atlas etc. were NOT touched here — they
  belong to the WIP committed separately.
- Verification: `make test-cr` 5/5; `test-autopair`/`test-queue`/`test-statusline`
  + full `go test ./...` green. Full `make test` green EXCEPT `test-changelog`,
  which fails IDENTICALLY on the pre-WIP baseline `4042686` (LLM-distiller env
  issue, unrelated to this fix).
- Tried twice to add an automated behavioral (newline-lands) headless check;
  both failed at the luafile compile stage under UI-attach + feedkeys — the
  flakiness the repo's expr-string precedent exists to avoid. Decision-table +
  wiring test + canonical idiom stand; operator does the 10-sec live confirm.
- Side note for operator: the WIP README added a duplicate `Alt+l` keybind row
  (it also said "First invocation is slow", contradicting line 72's "Opens
  instantly"). I dropped the dup and kept the detailed line 72; if the
  slowness note is accurate, fold it into line 72.
- 2026-06-18: closed — test-cr 7/7 (4 decision branches incl. the z= momentary
  case + wiring); operator live-confirmed the keystroke (newline lands on
  Return with a popup up & nothing picked; z= dismiss adds no newline).
  test-autopair/queue/statusline + `go test ./...` green; full `make test` green
  EXCEPT pre-existing `test-changelog` (fails identically on baseline 4042686,
  LLM-distiller env). FORCE on close: `active-time-v3.py` absent (uncloned
  data-dep) → ACTUAL unmeasurable, not hand-guessing per AGENTS.md §5;
  estimate_hours 0.5.

## Revisions

### 2026-06-18 — milestone review caught a shared-chokepoint regression

**Reason:** The `## Spec` three-state table was written for ONE consumer of the
insert `<CR>` map (as-you-type draft completion), but the map is a shared
chokepoint: it also serves the momentary normal-mode `z=` spell popup
(`spell_suggest_popup`, gated by `spell_popup_active`), whose contract is
"dismiss leaves the text intact" — NO newline. The first cut's `<C-e><CR>` would
inject a spurious newline into the draft when a `z=` popup is dismissed with
Return (the deferred `stopinsert` keeps us in insert mode when the `<CR>` lands).
Critical found by `sdlc judge milestone-review`.

**Delta:**
- `cr_keys` gains a third `momentary` arg (still pure): momentary + no selection
  → bare `<CR>` (clean dismiss); typing + no selection → `<C-e><CR>` (newline).
- The `<CR>` map passes `momentary = spell_popup_active`.
- `tests/cr-newline-test.sh` adds two `z=` momentary cases (now unit-testable
  since the distinction is a pure arg — closes the review's test-gap note).
- Atlas + the `cr_keys` comment document the two consumers.
