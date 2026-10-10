---
id: 000424
status: working
deps: []
github_issue:
created: 2026-10-10
updated: 2026-10-10
estimate_hours:
card_mirror: '22f9e830b74ffa58f224e72ef4eb043ca7b87ce4' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-10T09:55:49-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "7201e287", done: "9817e9e2"}
---

# couch CLI routes only four slot verbs to the slot-operation path; --relaunch/--reload-context fail

## Problem

Found live by TL ops:0 on 2026-10-10, right after #421 landed: `couch
--relaunch pair:3 --confirm` fails with "relaunch requires only a request ID
and an exact slot". `runMessageCLIWithCall` (`cmd/internal/couchcmd/messages.go`)
sends only resume, reboot, reap and recover to `runSlotOperationCLI`. Relaunch
and reload-context fall through to the message path, which sends no request ID.
#421 made `couchcore.IsSlotOperation` the one verb list, but this router kept
its own list. The slot-operation tests call `runSlotOperationCLI` directly, so
nothing exercised the router.

## Spec

The router asks `couchcore.IsSlotOperation`. A dispatch test drives every
declared slot verb from parsed argv through `runMessageCLIWithCall` and
asserts that each one reaches the broker as an operation request with an ID.

## Done when

- `couch --relaunch` and `couch --reload-context` reach the slot-operation
  path.
- A router-level test fails if any declared slot verb is not routed.

## Plan

- [x] Router asks `couchcore.IsSlotOperation` (`messages.go`).
- [x] `TestEverySlotOperationIsRoutedToTheSlotPath`: argv → router → broker,
      for every declared slot verb; it failed for exactly relaunch and
      reload-context before the fix.
- [x] Swept the other hand-kept verb lists. They are different concepts and
      correctly exclude the new verbs: local-CLI scope and live ownership
      (`run.go`), switcher argument shape (`menu_slot.go`), and hosted-child
      ending (`menu.go`, where reload re-execs inside its pane).

## Log

### 2026-10-10
- 2026-10-10: closed — Pure routing bugfix, no new surface (--no-atlas): the atlas already documents both verbs. TestEverySlotOperationIsRoutedToTheSlotPath drives every declared slot verb from parsed argv through runMessageCLIWithCall to the broker; it failed for exactly relaunch and reload-context with the TL-reported error before the fix and passes after. couchcmd and couchmessage green under clean env. Other hand-kept verb lists swept: run.go (local CLI scope/ownership), menu_slot.go (switcher arg shape), menu.go (hosted-child ending) correctly exclude the new verbs. Full suite ran on the #421 tree under an hour ago; this diff is one routing line plus a test.; review verdict: SHIP
