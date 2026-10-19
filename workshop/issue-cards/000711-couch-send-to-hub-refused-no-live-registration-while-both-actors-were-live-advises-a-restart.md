---
id: 000711
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
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
