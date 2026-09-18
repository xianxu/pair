---
id: 000286
status: open
deps: []
github_issue:
created: 2026-09-18
updated: 2026-09-18
estimate_hours:
---

# Couch switcher controls: derive menuControls from couchkeys and document Ctrl+Space's switcher meaning

## Problem

Two advisories from `pair#282`'s close review, deferred so #282 could merge.

1. **`couchtty.menuControls` (`menu.go:22`) still restates Couch's chords by
   hand.** #282 made `couchkeys` the single source for Couch's chords.
   couchtty's framing and routing, `couch --help` and Pair's Alt+h page all
   derive from it. `menuControls` keeps a parallel list of six of the same
   chords with its own wording. Its Alt+d reads "detach this thread · all +
   leave couch here", while couchkeys says "detach every live thread and leave
   Couch". Only `TestREADMEDocumentsEveryPanelControl` consumes the list.
   Its comment, and `readme_test.go`'s, claim the panel renderer consumes it
   too, which is stale.
2. **Ctrl+Space's in-switcher meaning is undocumented.** With the panel
   focused, Ctrl+Space starts a thread (`console.go` `onHotkey` → `KeyCtrlSpace`
   → `openStartForm`). couchkeys declares Ctrl+Space only at `ScopeEveryPane`,
   so neither `couch --help` nor the Alt+h page lists that second meaning. The
   "Couch switcher" section still reads as the switcher's key list. A
   switcher-scope row would follow the page's `(key, context)` rule, but
   couchkeys rows also drive framing, and `TestEncodingsAreUnique` forbids two
   rows with the same bytes. It needs a help-only concept, or another way to
   declare it.

## Spec

To be designed.

## Done when

- [ ] Couch's chord rows in `menuControls` derive from `couchkeys` (or are
      dropped from it). The stale "panel renderer consumes it" comments are
      corrected.
- [ ] The switcher section documents Ctrl+Space's in-switcher meaning, and a
      test ties it to the panel handler.

## Plan

- [ ] Design with the operator.

## Log

### 2026-09-18

- Filed from `pair#282`'s close review (advisories 2 and 3 of round 1).
