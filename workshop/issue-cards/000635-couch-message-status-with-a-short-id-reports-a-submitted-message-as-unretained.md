---
id: 000635
status: open
created: 2026-10-16
updated: 2026-10-17
estimate_hours:
github_issue:
started: 2026-10-17T06:48:43-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:3
    worktree: /Users/xianxu/workspace/worktree/pair-slot3/pair
    repository: github.com/xianxu/pair
---

# couch --message-status with a short ID reports a submitted message as unretained

## Problem

`couch --send-to` prints the message's full ID. A sender who checks it with the 8-character prefix the
fleet uses in reports (`couch --message-status 97fcc5ef --json`) gets
`{"Code":"refused","Error":"receipt unavailable: no connected recipient retains this message"}`, even while the
broker holds the receipt as `submitted`. Querying the full ID returns the real status. Observed 2026-10-16 on pair:2:
three hub reports read as lost were in fact delivered, and the TL had replied to one of them. A false
"unretained" answer leads senders to resend, or to treat delivery as broken.
