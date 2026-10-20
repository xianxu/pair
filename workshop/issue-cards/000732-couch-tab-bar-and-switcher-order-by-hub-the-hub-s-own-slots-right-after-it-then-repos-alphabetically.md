---
id: 000732
status: open
created: 2026-10-20
updated: 2026-10-20
estimate_hours:
github_issue:
---

# couch: tab bar and switcher order: by hub, the hub's own slots right after it, then repos alphabetically

## Problem

Couch orders the tab bar (and the switcher) so a hub repo's own worker slots land far from the hub: today
`ops  ariadne :1 :2 … brain  ops:1  pair :1 …`, with ops:1 after brain, away from ops:0 (operator, 10-20, screenshot
in ops's session file). Now that ops repos carry code (ops's charter, 10-20), hubs have worker slots of their own,
and they belong next to the hub.
