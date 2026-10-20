---
id: 000732
status: open
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'c9c020ea19c94c4471db562d722de8412fadf374' # card fields mirrored from issue-cards; edit via sdlc
---

# couch: tab bar and switcher order: by hub, the hub's own slots right after it, then repos alphabetically

## Problem

Couch orders the tab bar (and the switcher) so a hub repo's own worker slots land far from the hub: today
`ops  ariadne :1 :2 … brain  ops:1  pair :1 …`, with ops:1 after brain, away from ops:0 (operator, 10-20, screenshot
in ops's session file). Now that ops repos carry code (ops's charter, 10-20), hubs have worker slots of their own,
and they belong next to the hub.

## Spec

One ordering rule, shared by the tab bar and the switcher (one function, ARCH-DRY):
1. **Grouped by hub:** each hub's section, in today's hub order (the steward hub first, then later hubs after a
   separator, as now).
2. **The hub's own slots first:** the hub repo's :0, then its own :1, :2, … immediately after it.
3. **Then alphabetical:** the remaining repos owned by that hub in alphabetical order, each followed by its slots in
   numeric order.

Target: `ops  :1  ariadne  :1  :2 …  brain  pair  :1 …`, then `ops-echo  echo  :1 …`.

## Done when

- A unit test pins the order for a fixture with two hubs, a hub with its own slots, and several repos (including one
  whose name sorts before the hub's), for both the tab bar and the switcher.
- The operator's live look at the next Couch restart shows ops:1 right after ops.

## Plan

- [ ]

## Log

### 2026-10-20
