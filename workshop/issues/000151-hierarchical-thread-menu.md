---
id: 000151
status: open
deps: [149]
github_issue:
created: 2026-08-24
updated: 2026-08-24
estimate_hours:
---

# couch: hierarchical work-thread menu

## Problem

`#146` proves deterministic tty switching, but its flat transitional panel has
no discoverable home for secondary actions after the unintuitive `:` command
namespace is removed. Adding actor-specific actions there would encode the
wrong identity: actors are live harness incarnations, while `#149` makes the
durable work thread the thing an operator names, describes, parks, resumes, and
returns to.

## Spec

**A filtered hierarchical menu over work threads.** The root list is one row per
durable work thread from `#149`, including parked historically active threads.
Rows lead with the human name when present and the opaque tag fallback when not,
then show path, live/parked state, notification state, and—when parked—relative
last-active age. A live row says `live` rather than presenting a historical age.
Multiple threads at one path therefore remain visibly distinct.

Every menu level is a frame with its own stable item identities, filter text,
and selected identity. Printable input filters the current list. Filtering keeps
the selected item when it remains visible, otherwise selects the first match;
zero matches means no selection. Up/Down move within the filtered rows. Tab
pushes the selected item's submenu, Escape pops one frame, and the parent frame
returns with its filter and selection unchanged. Enter invokes the selected
item's primary action. With no selection, Enter and Tab report `no selection`
and do not change levels.

Escape at the root follows the transitional panel's familiar two-step rule: a
non-empty filter is cleared while retaining the selected thread when possible;
with an empty filter, Escape returns to the active thread through the forced
clear-and-replay attach path. If no live thread can receive focus, the root menu
stays visible and reports why. Escape from a nested list always returns one
level even when that child has a filter; the parent frame is restored exactly.

At the thread level, Enter attaches to a live thread or resumes a parked one.
Tab opens that thread's secondary actions. The first actions are `park` for a
live thread, `rename`, and `describe`; a parked thread offers `resume` instead
of `park`. Actions capture the durable thread ID when the submenu opens and
revalidate it at dispatch. A thread that disappears produces a notice and pops
back to the refreshed root list; it can never redirect an action to whichever
row moved into the old screen position. Dispatch also revalidates action
applicability against current live/parked state. If the state changed, no stale
action runs; the menu returns to the refreshed root list with that thread
selected and reports the change.

`park` always enters a confirmation frame. Its rows are `cancel` (selected by
default) and `park <thread label>`; Escape or Enter on cancel returns to the
action frame, while Enter on the explicit park row dispatches. Success returns
to the root list with the same thread selected and marked parked. Failure leaves
the action frame intact and reports through the notice feed.

`rename` and `describe` open text-input frames. Printable keys append and
Backspace edits. Escape cancels back to the unchanged action frame. Enter sends
the exact text through couch's shared operation surface; validation remains the
operation's responsibility. Failure keeps the input open with its text intact
and reports a notice. Success refreshes thread data and returns to the action
frame. No terminal UI action gets a private dispatch path unavailable to the
advisor (ARCH-DRY, ARCH-PURE).

Ctrl-Space is the global start action while any menu-list frame is visible: it
opens a start-path input without discarding the menu stack. Escape restores that
stack; Enter starts with `.` for empty input or the exact entered path, then
returns to the refreshed root list with the new thread selected. Ctrl-Space is a
no-op while any text input is already active. A failed start keeps the input,
typed path, originating menu stack, filters, and selections intact and reports
through the notice feed.

The current selection uses reverse video plus `▸`, so it remains visible without
color. A child menu renders to the right of its parent row when space permits
and below the parent list in a narrow terminal. Live threads render normally;
parked age adds a relative text label and one of three progressively dimmer ANSI
grays: less than 24 hours, 1–7 days, and more than 7 days. Terminals without
256-color support retain the state and age text, so gray is never the sole cue.

The menu transition reducer, filtering, age-band choice, and wide/narrow layout
decision are pure. Rendering and operation dispatch are thin injected edges.
Thread summaries, resolution, and operations come from their existing shared
sources rather than being restated in the terminal package (ARCH-PURE,
ARCH-PURPOSE).

## Revisions

### 2026-08-24 — root exit, stale state, and failed start are total transitions

**Reason:** spec review found three missing edges in the menu state machine and
one inconsistent recency sentence.

**Delta:** root Escape now clears filter then returns to the active thread;
nested Escape pops exactly one level. Dispatch revalidates both durable identity
and current action applicability. Failed start preserves its input and the full
originating stack. Only parked rows show historical age/grayscale; live rows say
`live`. These rules make every reported edge deterministic without adding a
second operation or state source (ARCH-DRY, ARCH-PURPOSE).

## Done when

- Enter attaches/resumes the selected work thread; Tab enters its action menu;
  Escape restores the exact parent filter and selection.
- Park requires explicit confirmation and preserves the durable work thread.
- Rename and description operate on the selected durable thread, survive a
  harness restart, and expose operation failures without losing typed text.
- Multiple threads at one path are distinct rows; parked threads remain listed
  with textual state/age and progressively dimmer age bands.
- Ctrl-Space opens start from every list level without destroying the menu stack.
- Stale targets and zero-match lists never dispatch against a different row.
- Wide and narrow terminal layouts are readable, with selection visible without
  relying on color.

## Plan

- [ ] Build the pure menu-stack reducer and filtered selection model.
- [ ] Render thread rows, recency, selection, and nested wide/narrow menus.
- [ ] Wire attach/resume, confirmed park, rename, describe, and global start
      through the shared couch operation surface.
- [ ] Add transition, stale-target, rendering, and full-console regression tests.

## Log

### 2026-08-24

Split from `#146` after operator smoke rejected the `:` command namespace and
design clarified that historical work threads—not live actors or worktrees—are
the durable menu rows. Depends on `#149` so the menu never invents a second
identity or metadata store.
