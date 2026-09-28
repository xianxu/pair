---
id: '000250'
status: done
started: 2026-09-14T10:31:03-07:00
created: 2026-09-14
updated: 2026-09-14
estimate_hours: 5.056
actual_hours: 7.25
---

# Recover stale Couch threads without losing live sessions or checkpoints

## Problem

A failed hosted restart or lost Couch helper can leave a thread recorded as
live while Couch refuses to host it. The operator cannot resume work, and
archive can also refuse because the persisted incarnation is still occupied.
Recovery must be available without hand-editing the thread store or discarding
a surviving agent conversation.

Reported 2026-09-14 for Pair address
`e108517d46ab4575/couch-36b623f6869ebaa2`, following the continuation failure
tracked in #249. Read-only `couch --show pair` reported recorded live PID 5330
and `unusable: stale — couch exited unexpectedly`. At inspection time,
`zellij list-sessions --no-formatting` showed no live Pair session. The operator
had reported an archive refusal attributed to a live Zellij session; the exact
UI error was not captured, so distinguish that report from current observation.

Source inspection found `archivableRecord` rejects occupied persisted
incarnations before `Couch.ArchiveThread` reaches session quiescence. Thus a
stale live record can block the documented manual escape even when its process
has died. Recheck runtime evidence and the exact refusal before repair.

The archived proposal
`workshop/history/issues/000171-reconcile-stale-incarnations-after-crash.md`
was punted because manual archive was considered sufficient. This incident
invalidates that premise. Refer to that file by path: another archived issue
also uses ID 000171. #214 covers a related launch-race recovery case; #249
prevents the continuation failure. This issue owns recovery from an already
stranded thread, independently of those triggering bugs.
