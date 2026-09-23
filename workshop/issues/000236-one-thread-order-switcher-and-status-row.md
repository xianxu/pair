---
id: 000236
status: open
deps: []
github_issue:
created: 2026-09-12
updated: 2026-09-21
estimate_hours:
---

# One thread order for the switcher and the status row: repo load order, slots grouped under their repo, operator-reorderable with Alt+Up/Down and persisted; the status row is its projection

## Problem

The two surfaces that list threads order them by unrelated rules, and neither
rule is one a person can predict.

- **Switcher**: `ActionableThreadInventory` sorts by `(RepoScope, Tag)`
  (`couchcore/actionableinventory.go:239`) and `visibleRootThreads`
  (`couchtty/menu.go:291`) returns that verbatim. `RepoScope` is a hex hash of
  the repo path and `Tag` is `couch-<16hex>`, so the list is **hash order**:
  stable across runs, but not alphabetical, not recency, not by state.
  Threads of one repo cluster only because they share a hash. `LastActiveAt`
  is on every row and unused.
- **Status row**: `c.order = append(c.order, handleID)` at attach
  (`couchtty/console.go:376`), removed on exit (`:942-944`), the first
  survivor becomes active when the active pane dies (`:956`), painted in that
  order (`:1059`, `:2120`). Attach order — but keyed by **pane handle, not
  thread**, so relaunch, resume and reattach move a thread to the end.

So the bar says one thing, `ctrl-space` says another, and neither survives a
relaunch. With slots (pair project `couch-slots`) a repo becomes a *group* —
`pair :1 :2` — and the group has to be a unit on both surfaces or the slot
model reads as noise.

**`couch-slots` is `defined`, not committed** (as of 2026-09-12): no
deadline, no planned finish, and the operator is undecided on it. This issue
does not depend on it. Everything here is worth doing with today's one
thread per repo — the hash-order switcher, the pane-handle bar, and the
relaunch jump are all present now. Where the text says "slot" or ":1", read
"a second thread of the same repo, if and when one exists"; with one thread
per repo every group is one row and the slot-specific parts (within-group
order, the second highlight level) are simply inert. The switcher also lists parked and detached threads the
bar never shows, so "same list" is impossible; "same order" has to mean
something stricter.

## Spec

**One ordering function over the inventory; the status row is its
projection.** The bar shows the attached subset in the same relative order,
so the bar's order is always a subsequence of the switcher's and the two
cannot disagree about relative position. A parked row occupies a switcher
position the bar skips.

**The order.**

1. **Groups are repo/path.** A group is everything at one repo — primary and
   its `:1`, `:2` slots (today: every thread sharing a working tree's repo;
   with slots, the primary path plus its slot paths).
2. **Group order is repo load order**: the order in which a repo first got an
   attached thread in this couch instance. A new group goes to the **end**.
   Persisted (below), so it survives restarts and a group that had a thread
   yesterday keeps its place today.
3. **Within a group**: primary first, then slots by number. State does not
   reorder — a parked `:1` sits under its repo between two live rows, marked
   as parked, not in a parked pile at the bottom.
4. **Operator reorder.** In the switcher, `Alt+Up` / `Alt+Down` move the
   selected row's **group** one position; the new order is persisted
   immediately and the bar re-projects. Inside the switcher couch owns input,
   so these bytes (`\x1b[1;3A/B` and the `1;9` Option forms already in the
   chord table, `workbenchshortcut/shortcut.go:368-369`) never reach pair,
   where the same chord means grow/shrink draft; no collision, but say so in
   `pair keys`.
5. **New threads** insert by rule 2: an existing group's new thread joins its
   group in slot order; a new repo's thread creates a group at the end.

**Persistence.** The group order is per couch store, beside the
`NamingTable` that `Store.Save`/`Load` already round-trip
(`couchcore/store.go:47,63`): an ordered list of group keys. Unknown keys
(archived repos) are dropped on load; keys not in the list (new repos) append
in first-attach order. Pure model + one file, same shape as naming.

**Visual cue for the group** (lands with slots; design it now so the
highlight model has room). Two levels of background: the **selected row**
(strong, today's `selectedMenuLine`, `menu_render.go:327,358`) and the
**selected row's group** (weaker), so that with the cursor on `:1` the whole
`pair` block reads as one thing. `#225`'s shared bar style owns the palette;
this issue owns the two-level structure. Until slots exist every group is one
row and the second level is invisible, which is fine.

**The bar**, concretely: replace `c.order`'s pane-handle list with a
projection of the master order filtered to attached threads, and derive
"next active when the active pane dies" from the same order. This removes
the relaunch-moves-you-to-the-end behaviour as a side effect.

Out of scope: reordering *within* a group (slot numbers are the order, by
design — `couch-slots`, occupancy rule), reordering from the bar (click to
switch stays; the switcher is where structure is edited), and any
notification-driven reordering (`ctrl-space` focusing the latest notification
is arrival, not order, and stays).

## Done when

- Bar order is a subsequence of switcher order at all times (a test walks the
  inventory, projects, and asserts subsequence for every attach/detach/park/
  resume/relaunch sequence generated from the operation table).
- A relaunched or resumed thread keeps its position on both surfaces.
- `Alt+Up`/`Alt+Down` in the switcher move the group; after a couch restart
  the order is as left; a new repo appears last.
- Slots (`:1`, `:2`) render under their repo in slot order regardless of
  state, once slots exist; the two-level highlight has a rendering test.
- `pair keys` documents the switcher-local meaning of `Alt+Up`/`Alt+Down`.

## Plan

- [ ] Pure `ThreadOrder` model: group key derivation, load-order list, insert-at-end, move-group, project-to-attached; table-driven tests including the subsequence property
- [ ] Persist beside `NamingTable` in `Store.Save`/`Load`; drop-unknown / append-new on load
- [ ] Switcher: `visibleRootThreads` reads the model; `Alt+Up`/`Alt+Down` move the group and save
- [ ] Bar: `c.order` becomes the projection; next-active from the same order; relaunch/resume regression test
- [ ] Two-level highlight in `selectedMenuLine`'s caller — structure now, palette from `#225`
- [ ] `pair keys` + README keybinding row

## Log

### 2026-09-12

- Filed from the brain advisor session after two rounds: the operator chose
  repo load order with slots grouped under their repo, then pointed out the
  switcher lists parked threads the bar doesn't, which is what produced the
  projection rule. Alt+Up/Down for group moves, persisted, and the two-level
  group highlight are the operator's; the end-insertion rule for new threads
  is theirs too.
- Related: `#197` (label derivation → `pair :1 :2`), `#225` (shared bar
  style), project `couch-slots` (defines slots and the occupancy rule the
  within-group order assumes — **not committed**; listed in its scope so the
  two stay consistent, not because this waits on it).

## Revisions

### 2026-09-21 — group identity comes from the couch contract

The grouping remains repo-first for the operator: primary `pair`, then durable
secondary slots `pair :1`, `pair :2` in numeric order. The filesystem spelling
(`worktree/pair-1` versus `worktree/pair-slot1`) is an implementation contract
owned by couch and must not leak into ordering or labels. The ordering model
should consume the normalized `(repo, slot-number)` identity produced by the
shared label/identity derivation, with primary represented as slot zero. This
keeps the status-row projection and switcher stable if the on-disk spelling is
chosen later (ARCH-DRY).

Agent profiles do not create a second ordering axis. Primary `pair` (`:0`),
design `pair :1`, and implementation `pair :2` remain ordered by slot number;
the effective harness/model is metadata on each row. An explicit profile
override must not move a thread or make it appear to be a different workspace.
