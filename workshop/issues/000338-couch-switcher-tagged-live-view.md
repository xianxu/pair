---
id: 000338
status: open
deps: [pair#337, pair#173]
github_issue:
created: 2026-09-28
updated: 2026-09-28
estimate_hours:
card_mirror: '7f975e63ce5024dea72196143892ef1d4e3ec7e4' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch switcher space toggles a focus view of tagged live threads

## Problem

#337 lets the operator tag a couch thread's description from pair's draft
(`! start working on #xxx`). Those tags only pay off if there is a quick way to
see them together: "what are the active things I marked?" The couch switcher
lists every thread, dead or alive, tagged or not. It has no view that shows
just the live threads the operator has tagged.

## Spec

The switcher gets two named views. The **normal view** is the existing
switcher. The **focus view** is a narrower filter over it: the live threads the
operator has tagged. Pressing `space` in the root view switches between them:

1. The focus view shows only **live** threads, using the same notion of
   liveness the switcher and status row already use.
2. It shows only threads whose **description is non-empty**.
3. It is a filter on the normal view, not a re-sort. Rows keep the normal
   view's order and only non-matching rows disappear. If #236 lands first, the
   normal view's order is #236's single shared order, and the focus view follows
   it automatically.
4. Each row renders as `name ◆ description`. `name` is the same thread label
   the normal view shows. `description` goes through couch's existing
   untrusted-text sanitization, and the row is truncated to the menu width.

Interaction:

- The root view already treats every typed character as filter text
  (`reduceRootKey`, `cmd/internal/couchtty/menu.go`). Space switches views
  **only when the filter is empty**. Once a filter has text, space is still
  appended to it as a literal character, so multi-word searches keep working.
- Selecting a row and pressing Return switches to that thread, exactly as in
  the normal view.
- Space works both ways: with an empty filter, space in the normal view opens
  the focus view, and space in the focus view returns to the normal view.
  Filtering by typing works the same in both views.
- When nothing is tagged, the focus view shows a one-line placeholder
  (e.g. "no tagged live threads — tag one with `! …` in a pair draft"), not an
  empty box.
- The switcher reopens in the view it was last left in. If the operator
  switched to the focus view and jumped to a thread from it, the next
  `ctrl+space` opens the focus view again. The last view is couch console
  state held in memory for the life of the couch process; it is not persisted.

## Done when

- With an empty filter, pressing space in the switcher shows only live threads
  with a non-empty description. The rows are in the normal view's order and
  read `name ◆ description`. Pressing space again returns to the normal view.
- After leaving the switcher from the focus view, the next `ctrl+space` opens
  the focus view; after leaving from the normal view, it opens the normal view.
- With a non-empty filter, space is added to the filter as before.
- Return on a tagged-view row switches to that thread.
- Tests at the menu reducer and render boundary cover: switching views, the
  live-and-tagged filtering, order preservation, row format and truncation,
  literal space inside a non-empty filter, and the empty-view placeholder.
- The operator smoke-tests it live together with #337 tagging.

## Plan

- [ ] Find where the root rows are built and how liveness and description
  reach them (`thread_presentation.go`, `menu.go`).
- [ ] Add a view mode to the root frame. Make the row filter a pure function of
  (rows, mode) and give it unit tests.
- [ ] Add the space toggle in `reduceRootKey`, gated on an empty filter, and
  the `name ◆ description` render with truncation.
- [ ] Add reducer and render tests, then ask the operator to run a live smoke
  test.

## Revisions

- 2026-09-28: the operator changed the starting view. The switcher now reopens in
  the last view used, instead of always opening in the normal view. Spec and
  Done when updated to match.

## Log

### 2026-09-28
- Operator confirmed the design: the views are named "normal view" and "focus view"; space toggles both ways when the filter is empty; the focus view is a pure filter on the normal view's order; there is an empty-view placeholder; the switcher always opens in the normal view.
