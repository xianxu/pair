---
id: 000320
status: wontfix
created: 2026-09-24
updated: 2026-10-02
estimate_hours:
github_issue:
---

# Resume every parked thread when couch starts

## Problem

Parking is the cheap way to put a fleet of threads away (batch park frees
their agents and zellij sessions), but there is no matching way to bring the
fleet back. When couch starts it attaches one thread (`SelectResumableRoot`)
and the #206 background pass reattaches only *detached* threads — agents still
running behind a client-less session. That pass is `warm-only` by design: it
refuses a thread that is parked and never starts an agent. So every parked
thread has to be resumed by hand, one switcher action at a time.

Goal: park a batch, quit couch, start couch later, and have the batch back —
batch park paired with batch recover.
