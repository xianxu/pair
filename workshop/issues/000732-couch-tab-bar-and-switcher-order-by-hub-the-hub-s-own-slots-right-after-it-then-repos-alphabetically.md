---
id: 000732
status: working
deps: []
github_issue:
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
collaboration-mode: collaboration
card_mirror: '0c3670d57e42e629b0e8beeb7a8a57930b49e2b8' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-20T10:15:12-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "0dd72e62", done: "fe5614c4"}
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
- Land on SHIP after automated verification. The operator checks ops:1 immediately after ops at the next Couch restart after landing; any live defect is filed separately (TL revision, 2026-10-20).

## Plan

- [ ] Add failing two-hub tests for shared projection, switcher order, and tab order, with shuffled input and numeric slots.
- [ ] Promote the hub repository group in existing pure hubSection (ARCH-DRY, ARCH-PURE); preserve ownership, hub order, and alphabetical remainder.
- [ ] Update atlas; run targeted tests and recorded suite; close review and report for next-restart live acceptance.

## Log

### 2026-10-20

- Quick plan posted to hub; proceeding unless objected. Both consumers already derive from PresentThreads; changing their shared hubSection avoids a second ordering implementation. No IO, durable state, lifecycle, or format changes. Existing sort remains O(N log N); section promotion is a bounded number of linear passes, with no nested row scans.

## Revisions

### 2026-10-20 — acceptance timing (TL ops:0)

The original Done when required the operator live look before acceptance. TL instructed no staging because pair:0 is held by #722; the shuffled two-hub regression can land on SHIP. Done when now permits close and landing after automated verification, with the operator live look at the next Couch restart after landing. Any defect found there becomes a new issue.
