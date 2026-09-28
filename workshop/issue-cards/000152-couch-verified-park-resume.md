---
id: '000152'
status: done
started: 2026-08-27T12:04:27-07:00
created: 2026-08-25
updated: 2026-08-30
estimate_hours: 9.90
actual_hours: 13.40
---

# couch: verified park and resume lifecycle

## Problem

`#149` makes a durable work thread distinct from its optional live actor
incarnation. The thread menu needs a normal way to make a live thread inactive
without deleting its draft, ledger, transcript, native-session identity, name,
or description. Signalling couch's top child is insufficient: Pair hosts a
zellij server session whose panes can outlive that client, so freeing the
concurrency slot at that point would admit a second writer beside the first.

The same quiescence proof is needed by `#135` before transferring a live tag to
another agent. This issue owns that reusable lifecycle protocol; the menu in
`#151` is only a client.
