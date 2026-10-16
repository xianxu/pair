---
id: 000614
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
severity: low
card_mirror: '917c5cf98e2ae28ab51eff24d859e30371b444b8' # card fields mirrored from issue-cards; edit via sdlc
---

# One shared rule for native conversation IDs and lock holder labels

## Problem

Three rules validate the same native conversation ID and holder label in different ways (pair#595 close review): `artifactpath.NativeLockFilename` (lock key), `nativelock.validHolder` (holder label) and the watcher's `validBindingNotice` (sessionwatch binding notice). A binding notice can carry an ID (space or Cf characters) that the notice accepts and `NewKey` later rejects; the wrapper now drops it as ignored, but the rules disagree.

## Spec

One shared rule for a native conversation ID (and one for a holder label), used by the lock key, the binding notice and the holder parser, so an ID is accepted or rejected the same way at every boundary. No behavior change for valid IDs.

## Done when

- One definition, used at all three sites; tests cover the disagreeing cases (space, Cf, oversize) at each boundary.

## Plan

- [ ]

## Log

### 2026-10-16
