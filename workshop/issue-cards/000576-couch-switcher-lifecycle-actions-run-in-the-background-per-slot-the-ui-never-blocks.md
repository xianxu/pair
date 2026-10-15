---
id: 000576
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
---

# Couch switcher: lifecycle actions run in the background per slot; the UI never blocks

## Problem

While the operator runs a lifecycle action in Couch's switcher (bring a slot up, relaunch, resume), the switcher holds
the screen until the action finishes or fails: the operator can't switch to another slot or the hub meanwhile
(operator, 2026-10-15). A slot's lifecycle is already modelled as states, and startup brings many slots up in the
background, so a single slot's action needn't block the UI.
