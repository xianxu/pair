---
id: '000146'
status: done
started: 2026-08-22T12:14:19-07:00
created: 2026-08-21
updated: 2026-08-25
estimate_hours: 10.32
actual_hours: 37.03
---

# couch: tty switching and attach

## Problem

With a registry of named actors (`#145`), the operator still has no way to move
between them except terminal tabs, which know nothing about what a session is.
The switching experience is what determines whether couch gets used at all: if
getting back to a known place is ever slow or flaky, the operator reverts to tabs
and everything above it is dead weight.
