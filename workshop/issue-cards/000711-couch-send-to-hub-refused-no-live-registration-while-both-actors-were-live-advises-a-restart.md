---
id: 000711
status: working
created: 2026-10-19
updated: 2026-10-20
estimate_hours:
github_issue:
started: 2026-10-20T05:51:05-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# couch --send-to hub refused 'no live registration' while both actors were live; advises a restart

## Problem

2026-10-19 ~13:15, pair:7 (Codex) ran `couch --send-to hub --message '<a long plan summary>'` and got exit 1 with no receipt:
`couch: unavailable: Couch cannot find a live caller or recipient registration; if the slots are running, restart Couch to
reconnect them`. Right after, `couch --actors --json` showed both pair:7 and ops:0 live with their recorded incarnations, and
a shorter message sent to `ops:0` by exact address succeeded (receipt 07decfe3…). Earlier the same day the hub hit a related
transient refusal on a send: `couch: unavailable: observe listed session: session listing observations disagree`.

Either the `hub` alias resolution fails transiently (or for this caller) while exact addressing works, or a transient
registry/listing disagreement is reported as "no live registration" and tells the slot to restart Couch. The advice is wrong
either way: restarting Couch is the most disruptive action available, and the slots were live.
