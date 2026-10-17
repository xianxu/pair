---
id: 000633
status: open
deps: [pair#618]
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '468c22e4d284ab83cf811db134bb8fa6d5dcf7aa' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-16
