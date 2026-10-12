---
id: 000509
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '7a042fd0373cf28e3c76ec7a251711d9ddc1df91' # card fields mirrored from issue-cards; edit via sdlc
---

# broadcast: lock replay share links behind a code (follow-up to #495)

## Problem

Replay share links (pair#477; `webcast/shares.go` `NewReplayShares`, `ReplayServer`) are mounted on the shared `broadcast.Host` without a code: their `LockFor` returns nil by design in #495 (the operator's D4: replay is read-only, lower severity). A replay link can still expose a recording's page content, typed values (passwords excepted) and notes to anyone who holds it.

## Spec

Lock replay share links like live links: mint a castlock with each share token through the `broadcast.LinkLocks` registry, show the code in Couch's operator-only layer where the share link is copied, and end the lock when the share is removed (or Couch exits). Same lock page, session cookie, wrong-try budget and rotation as #495.

## Done when

- A replay share link opens locked until its code; the server refuses the replay's routes without a session; the code shows only in the operator-only layer; tests cover each guard.

## Plan

- [ ]

## Log

### 2026-10-12
