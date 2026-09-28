---
id: '000317'
status: done
started: 2026-09-23T23:22:18-07:00
created: 2026-09-23
updated: 2026-09-24
actual_hours: 1.91
---

# Slot quick-status glyph in Couch tab bar and switcher

## Problem

The switcher and tab bar (#307) show each slot's address (`pair:1`, `:2`) and
liveness, but not whether the slot has work in it. Deciding where to start
something, or which slot can take a moved branch ("move this branch to :N",
ariadne#248), means opening each slot or running git by hand.
