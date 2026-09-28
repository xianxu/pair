---
id: '000237'
status: done
started: 2026-09-12T17:39:11-07:00
created: 2026-09-12
updated: 2026-09-12
estimate_hours: 0.35
actual_hours: 0.04
---

# relaunch refuses once a thread's ledger passes 8 MiB: the session query caps the whole ledger file instead of each record

## Problem

Operator report, 2026-09-12, from couch's relaunch action on `couch-5003fd4f6c74f514`:

```
error: relaunch couch-5003fd4f6c74f514: could not resolve its native session binding: session inventory read exceeds limit
```

Measured: that thread's ledger, `ledger-couch-5003fd4f6c74f514.jsonl`, is
8,482,765 bytes — 39 rows, of which 14 `launch` rows carry 8.47 MB between
them (each ~600 KB; longest line 677,523 bytes). The 8 MiB it just crossed
is a **whole-file** cap:

- `sessioninventory.QuerySessionContext` (`query.go:99`) reads the owner
  ledger with `runtime.ReadFile(ledger, 8<<20)`. Over the cap `ReadFile`
  returns `ErrReadLimit`, the query returns it as an error, and every caller
  turns it into a refusal: couch relaunch (`couchcore/relaunch.go:120`), and
  also the launcher's own resume (`launcher/osruntime.go:705`), `reviewcmd`,
  `opener`, `slugcmd`, `titlepoller` (on a polling cadence), `contextcmd`, and
  the activity CLI — nine callers, one read.
- `RecoverPairBindings` (`pair_inventory.go:69`) reads the same ledger family
  with a 64 MiB whole-file cap; same class, later cliff.

The ledger is append-only JSONL, one record per line. The package already
has the right bound for that shape: transcripts are read through
`visitJSONLinesAt`, which caps **each record** (`jsonRecordLimit`, 8 MiB)
and never the file. The ledger reads predate that helper and kept a whole-
file cap, so a thread that has been relaunched enough times stops being
resumable — the file grows with every launch by construction, so this is a
cliff every long-lived thread reaches, not an anomaly.

Why the rows are 600 KB each is a separate defect — every launch record
embeds a boundary snapshot of the whole storage root — filed as #238. This
issue is the read: a per-file cap on a file that is defined to grow is the
wrong bound regardless of row size.
