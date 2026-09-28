---
id: 000236
status: open
created: 2026-09-12
updated: 2026-09-21
estimate_hours:
github_issue:
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
