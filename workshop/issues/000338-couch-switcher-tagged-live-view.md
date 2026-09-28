---
id: 000338
status: working
deps: [pair#337, pair#173]
github_issue:
created: 2026-09-28
updated: 2026-09-28
estimate_hours:
card_mirror: '6f2f8f0c6f1ae23dba55b5830b53383528d79151' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-09-28T13:44:11-07:00
flow: {kind: quick, provenance: inferred, spec: "f04ec6d5", done: "91674aef"}
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
- The switcher reopens in the last view used, remembered for the life of the
  Couch process. It initially opens in normal view; no setting is persisted.

Implementation design:

- Root frame owns the normal/focus mode. Toggle only on a root-space event
  with an empty filter, then reconcile selection through the existing reducer.
  Keep the mode when returning from actions, refreshing inventory, and reopening
  with Ctrl-Space; retain selection by row identity where still visible.
- Filter the existing ordered rows with `Live()` and a non-empty sanitized
  `DisplaySummary()` (published summary takes precedence over operator
  description). Share the visible-row path across rendering, keyboard selection,
  and mouse extents. Normal view behavior and search rules remain unchanged.
- Derive labels against the complete inventory so filtering cannot rename a
  row. Focus rows replace normal path/status details with `name ◆ description`,
  using existing sanitization and terminal-cell clipping. Keep attention lines
  and address-based dispatch. Label the focus view and document the space key.
- Reuse the existing inventory refresh to observe description/liveness changes.
  No new IO, durable storage, timers, or services are needed (ARCH-DRY,
  ARCH-PURE). The one mode value dies with its owning console (ARCH-FUNERAL).

## Done when

- With an empty filter, pressing space in the switcher shows only live threads
  with a non-empty description. The rows are in the normal view's order and
  read `name ◆ description`. Pressing space again returns to the normal view.
- With a non-empty filter, space is added to the filter as before.
- Return on a tagged-view row switches to that thread.
- Reopening remembers normal/focus mode for the running console. Refreshes
  update membership without changing mode or routing hidden rows.
- Tests at the menu reducer and render boundary cover: switching views, the
  live-and-tagged filtering, order preservation, row format and truncation,
  literal space inside a non-empty filter, and the empty-view placeholder.
- Tests cover published-description precedence, untrusted text and Unicode
  width, identity-based selection after toggle/refresh, submenu return, mouse
  extents, and reopen persistence. README and switcher key help describe Space.
- The operator smoke-tests it live together with #337 tagging.

## Plan

- [x] Find where the root rows are built and how liveness and description
  reach them (`thread_presentation.go`, `menu.go`).
- [x] Add a view mode to the root frame. Make the row filter a pure function of
  (rows, mode) and give it unit tests.
- [x] Add the space toggle in `reduceRootKey`, gated on an empty filter, and
  the `name ◆ description` render with truncation.
- [x] Add reducer and render tests, then ask the operator to run a live smoke
  test.

## Revisions

- 2026-09-28: restore the operator's already-approved last-view preference
  from commit `0d973f84` (preserved in #337's landing log). The original issue
  was handed off before that edit could land. Update Spec and Done when to
  retain mode in console memory, and name existing reducer/render seams and
  boundary tests. No change to the approved normal/focus toggle interaction.

## Log

### 2026-09-28
- 2026-09-28: closed — Operator live smoke passed. Fresh full couchtty suite passed. Twenty race-enabled focus reducer/render and real-console refresh/reopen repetitions passed. Literal Space tests enumerate normal/focus with unchanged mode and no effects; broken normal-view guard mutation rejected. README checks and build passed; production unchanged since smoke.; review verdict: SHIP
- Operator confirmed the design: the views are named "normal view" and "focus view"; space toggles both ways when the filter is empty; the focus view is a pure filter on the normal view's order; there is an empty-view placeholder; the switcher always opens in the normal view.
- Started work after #337/#329 landed. The last-view decision above supersedes
  the initial always-normal entry. #173 is still open, but this view can consume
  existing `ActionableThreadSummary.DisplaySummary()` directly; #337's stored
  tag does not require a new metadata writer or a separate #173 implementation.
- Spec review approved the existing seams. Implemented root-frame mode,
  post-search focus membership using the reattach overlay and sanitized
  `DisplaySummary`, stable full-inventory labels, focus rows and placeholder,
  and Space key help. README and atlas cover the mode and clarify that direct
  newest-notification jumps remain independent of focus filtering.
- TDD: initial reducer/render tests failed on missing focus behavior; the real
  console input test failed waiting for focus view. After implementation, the
  full couchtty suite passed, and `go test -race ./cmd/internal/couchtty -run
  'TestMenuFocus|TestConsoleFocusView' -count=3` passed. The console test covers
  live description update/removal/restoration plus normal/focus reopen through
  real key input and switching. README control tests and `make build` passed.
- Built in pair:0 and requested a fresh-Couch operator smoke. Automated tests
  are complete; live smoke and the SDLC close review remain pending.
- Operator confirmed: "#338 smoke test passed." Live acceptance is complete;
  the SDLC close review and landing remain pending.
- Close round 1 found no implementation defect; BR-1 requested literal-Space
  coverage in normal view as well as focus view. The shared test now enumerates
  both modes and asserts `r `, unchanged mode, and no emitted effects. A
  mutation that toggled normal view despite a nonempty filter failed this test;
  production was restored byte-for-byte. No production behavior changed.
- The follow-up race run exposed an early test observation: actor focus moves
  before the asynchronous switch completion clears `InFlight`. The console
  test now waits for both before reopening and issuing the next switch. Twenty
  race-enabled repetitions of the focus tests passed with the complete outcome
  observed; this changes test sequencing only.
