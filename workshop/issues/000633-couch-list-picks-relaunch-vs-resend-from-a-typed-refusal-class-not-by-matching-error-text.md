---
id: 000633
status: working
deps: [pair#618]
github_issue:
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
card_mirror: 'db13b95160a940dffccd319e223d42f48bf881b7' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-17T07:29:55-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "f9d13e8d", done: "ef574bea"}
---

# couch --list picks relaunch vs resend from a typed refusal class, not by matching error text

## Problem

`couch --list` shows a refused slot's receive state with one action (pair#618). `receiveStateText`
(`cmd/internal/couchcmd/list_refusals.go`) picks "relaunch" versus "send again" by matching five phrases ("does not
own", "conversation changed", "launch registration", "not positively listed", "no retained checkout") against the
refusal's free-text reason. Those phrases live in error strings in `message_service.go` and `registry.go`, so
rewording an error silently flips the advice from relaunch to resend. This is the second finding in the
duplicated-request-classification family (pair#618 close review, BR-14). The rule: classify a result once, at its
source, into a typed value, and have consumers read that value.

## Spec

- The ownership/registration failures that only a relaunch clears carry a typed class: a sentinel error or a typed
  error that `errors.Is`/`errors.As` can see, wrapped where they are raised.
- `refuseLocked` records that class on the refusal. The `refusals` op answers `{reason, class}` per slot: still
  reasons only, with no sender binding or body, since the op is unauthenticated.
- `receiveStateText` chooses the action from the class. No text matching.

## Done when

- A test rewords each ownership error string and the `--list` action stays "relaunch".
- A test shows an unclassified reason still gets "send again".
- `list_refusals.go` has no phrase list.

## Plan

- [ ] Add couchmessage.RelaunchRequiredError (preserving cause text/unwrapping) and RefusalInfo {reason,class}; classify once in refuseLocked, with cancellation/deadline remaining retryable. Retain class beside the existing bounded refusal record.
- [ ] Wrap all five source families in message_service.go, including both registration paths. Return the safe reason/class projection and make list rendering read the class only (ARCH-DRY/PURPOSE).
- [ ] Red/green regressions: reworded typed reasons still relaunch; unclassified old phrases, unknown classes and timeout/cancellation still resend; source checks carry the type; broker JSON excludes credentials/body. Keep existing expiry/supersession tests.
- [ ] Update atlas/README, run targeted tests and recorded full suite, close and land through SDLC. Headless acceptance: unchanged visible wording/actions, typed data path exercised through broker and list fixtures; no staging requested.

The structured values replace the old refusal strings only on this best-effort diagnostic op. Mixed-version decode failure keeps the ordinary list intact and omits receive annotations, as already documented for unsupported/older supervisors; no heuristic compatibility classifier. Unknown/missing classes use send-again. Other response shapes are unchanged. No new IO, timers or retention owners (ARCH-PURE/FUNERAL); existing 250ms call, 64-entry cap and 10-minute expiry remain. Unauthenticated output stays canonical slot plus reason/class only (ARCH-SECURE). Existing ordered storage/supersession is reused (ARCH-ORDER); no new state machine. Existing fake endpoints and local protocol fixtures suffice (ARCH-MOCK); no external integration changes (ARCH-CONSTRAINTS).

## Log

### 2026-10-16

Planning: all matching strings originate in message_service.go; registry.go has no corresponding error source. Existing refusalReason normalizes context failures, so classification preserves retry advice for those even inside a typed registration wrapper.
