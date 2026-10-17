---
id: 000633
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
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
