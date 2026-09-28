---
id: '000154'
status: done
started: 2026-08-27T16:23:37-07:00
created: 2026-08-27
updated: 2026-08-27
estimate_hours: 3.35
actual_hours: 12.21
---

# decouple Pair from Couch state

## Problem

Pair is the lower-level wrapper around coding harnesses. Couch is an integrated
workspace shell which hosts Pair, but Couch's configuration and durable thread
model must not become prerequisites for using Pair directly.

The #149 implementation crossed that boundary: every direct Pair launch reads
the Couch thread-store manifest, and ordinary Pair launches register themselves
into that store. A valid Couch manifest written after M5 added
`legacy_migration_version`; Pair's private strict projection did not add the
field, so `pair` now exits before launching any harness. Under Couch this appears
secondarily as an `await Pair registration` timeout.
