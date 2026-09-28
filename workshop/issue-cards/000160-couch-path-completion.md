---
id: '000160'
status: done
started: 2026-09-01T12:02:48-07:00
created: 2026-09-01
updated: 2026-09-01
estimate_hours: 3.60
actual_hours: N/A
---

# Couch start path tab completion

## Problem

The Couch “start thread” form accepts a filesystem path as unassisted text. An
operator must already know and type the exact directory, which makes navigating
to a nearby or unfamiliar working directory unnecessarily slow and
error-prone. `Tab` currently moves focus from the path field to the agent field,
so completion also needs an explicit field-navigation model.
