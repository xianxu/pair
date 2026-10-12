---
id: 000463
status: done
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '895aae701981a275974eec09f4f7b6de69fe0669' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-12T00:15:37-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "a00f3e48", done: "2343f58b"}
actual_hours: 0.31
---

# couch --send-to hub: a reserved target the supervisor resolves live to the hub slot

## Problem

ariadne#336 adds a constitution rule: every agent in a Couch slot reports each item's outcome to the hub (the slot Couch started from, where the TL works) and asks the hub instead of an in-pane question prompt, whether the hub dispatched the work or the operator started it by hand. A slot has no way to learn the hub's address. Couch already knows it (`cmd/internal/couchcmd/message_service.go` `hubWorktree`, used for the TL's 64-message allowance and for waiting notices), but exposes it to no slot: no environment variable, no query.

## Spec

- `couch --send-to hub --message TEXT` resolves `hub`, a reserved target, in the running supervisor to the hub's connected binding (`hubBinding`), and delivers as an ordinary peer message (same receipt, admission and allowance rules).
- It resolves live, at send time. An environment variable such as `COUCH_HUB` was rejected: it goes stale when a detached thread is resumed under a Couch started from a different tree.
- Refusals name the reason: no hub (Couch started outside a checkout), the hub isn't connected, or the sender is the hub itself.
- `hub` is reserved ahead of repository-name and prefix resolution. If a repository could be named `hub`, its slots stay reachable as `hub:N`, and `couch --skill` documents the reservation.
- `couch --skill` documents `couch --send-to hub` in its addressing section.

## Done when

- `couch --send-to hub --message …` from a non-hub slot delivers to the hub slot: covered by a service test here; the live check in a running Couch is delivered by ariadne#336's Done-when (see Revisions).
- Each of the three refusals is covered by a test and names its reason.
- `couch --skill` documents the target and the reservation.

## Plan

- [x] `messageService.handle` rewrites a send to `couchmessage.HubTarget` to the hub's exact slot via `hubBinding()` before the broker's repository-name routing (`hubRecipient`); three refusals: no hub (`refused`), hub not connected (`unavailable`), sender is the hub (`refused`).
- [x] Tests: `TestMessageSendToHubReachesTheHubSlot`, `TestMessageSendToHubRefusalsNameTheReason`, `TestMessageHubTargetIsReservedAheadOfRepositoryNames` (message_service_test.go).
- [x] `couch --skill` table row + reservation paragraph; `--help` usage line.
- [x] Live check deferred by the TL to ariadne#336's Done-when (ariadne:1 runs it after the next batched Couch restart).

## Log

### 2026-10-12
- 2026-10-12: closed — make -k test and go test ./... green; TestMessageSendToHubReachesTheHubSlot, TestMessageSendToHubRefusalsNameTheReason (all three refusals) and TestMessageHubTargetIsReservedAheadOfRepositoryNames pass; README/atlas/skill document the target; live --send-to hub check deferred by TL (ops:0) to ariadne#336 Done-when after the next batched Couch restart.; review verdict: SHIP
- Filed from ariadne#336 (ariadne:1), dispatched by the TL (ops:0). ariadne#336 stays open with a dep on this issue, and closes with its live check once this ships.
- Rewriting happens in the service, not `couchmessage.ResolveRecipient`: the hub is the service's knowledge (`authority.hub`), and a rewrite to an exact `repo:N` keeps the broker's admission, allowance and receipt rules unchanged (ARCH-DRY: one delivery path).
- Verified: `make -k test` green, `go test ./...` green, new hub tests pass. The live Done-when is deferred: the supervisor resolves `hub`, so it needs a Couch restart onto this build. The TL (ops:0) batches restarts and moved the live check to ariadne#336's Done-when.

## Revisions

- 2026-10-12: Done-when 1's live check moved to ariadne#336 (its Done-when, run by ariadne:1 after the next Couch restart), on the TL's call; this issue lands on unit tests.
