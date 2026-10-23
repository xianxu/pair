---
id: 000812
status: open
deps: []
github_issue:
created: 2026-10-22
updated: 2026-10-22
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'd0a40df2f80d44ead435780c873bdffb7cb5ac7a' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch tab bar: drop the trailing last lifecycle/error message (switcher keeps it)

## Problem

## Spec

Operator request (10-22, screenshot in ops `assets/session/2026-10-22.13-05-53.800.jpg`): the Couch tab bar's second
row shows the last lifecycle or error message after the slot tabs (e.g. `· pair [couch-dd48472d] exited (1)`). The
operator doesn't find it useful there and wants it removed from the tab bar. It stays in the switcher. Anything that
needs the operator's attention goes through the notification (attention list / `couch --notify`), not the tab bar.

- Remove the trailing last-message text from the tab bar (both one-row and wrapped two-row layouts); the space goes
  back to slot tabs.
- The switcher keeps showing it, unchanged.
- If any path relied on the tab-bar text as its only operator-visible signal, route it to the attention list instead
  and say which.

## Done when

- A tab-bar render test with a pending lifecycle/error message shows no trailing message text; the switcher test
  still shows it.
- README/atlas wording updated.

## Plan

- [ ]

## Log

### 2026-10-22
