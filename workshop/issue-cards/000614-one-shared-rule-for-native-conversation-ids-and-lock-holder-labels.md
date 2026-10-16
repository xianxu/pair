---
id: 000614
status: open
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
github_issue:
severity: low
---

# One shared rule for native conversation IDs and lock holder labels

## Problem

Three rules validate the same native conversation ID and holder label in different ways (pair#595 close review): `artifactpath.NativeLockFilename` (lock key), `nativelock.validHolder` (holder label) and the watcher's `validBindingNotice` (sessionwatch binding notice). A binding notice can carry an ID (space or Cf characters) that the notice accepts and `NewKey` later rejects; the wrapper now drops it as ignored, but the rules disagree.
