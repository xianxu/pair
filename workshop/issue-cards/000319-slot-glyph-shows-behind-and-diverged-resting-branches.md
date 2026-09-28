---
id: '000319'
status: done
started: 2026-09-24T14:13:32-07:00
created: 2026-09-24
updated: 2026-09-24
actual_hours: 0.35
---

# Slot glyph shows behind and diverged resting branches

## Problem

The #317 slot glyph marks a resting branch that is *ahead* of its upstream
(`+`) but says nothing when it is *behind*. A slot that has fallen behind
origin/main is a poor place to start work or move a branch into without a pull
first, and one that has diverged (as `main-slot1` did during #317: ahead 3,
behind 34) needs a rebase or a publish before either direction is safe. Today
both look clean.
