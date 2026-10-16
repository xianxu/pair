---
id: 000600
status: open
deps: []
github_issue:
target: slot-recovery
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
severity: critical
card_mirror: '6c9aa8e88b256fac5cd9df62abf0b4c9673f006e' # card fields mirrored from issue-cards; edit via sdlc
---

# Fresh reboot onto a new address fails recovery authority: SourceRevision names the old record

## Problem

`couch --reboot ariadne:4 --confirm` (Fresh onto a new address) fails every time with `pair: recovery authority changed: explicit recovery source revision, time or agent changed`. In couchcore `launchRecoveryFreshAddress`, the intent keeps `SourceRevision` = the OLD record's revision, while the authority's `RecordRevision` is the NEW address record's revision, which starts at 1. The launcher's `validateRecoveryRecord` refuses when `SourceRevision > RecordRevision`, so any reboot whose old record is past revision ~1 is refused. On disk, ariadne:4: new record 1-ariadne-52, Fresh intent source_revision 3 (old 1-ariadne-51, archived at rev 4). The tests missed it because the couchcore fixtures fake `ValidateRecoveryAuthority`.

## Spec

A Fresh recovery onto a new address writes an intent whose `SourceRevision` describes the new address record it is committed into, so the launcher's existing validation holds. No new checks or refusals; the validation rule itself is unchanged. Resume and relaunch paths are unaffected.

## Done when

- A test runs the REAL `validateRecoveryRecord` on the authority that the fresh-address path builds, with an old record past revision 1, and goes RED before the fix, GREEN after.
- The fake validator is dropped for that path if it is cheap to do so.
- Targeted tests, the recorded suite and close review pass; land, and report to the hub for the live `couch --reboot ariadne:4 --confirm`.

## Plan

- Set `nextIntent.SourceRevision` to the new record's revision in the `launchRecoveryFreshAddress` Commit callback, before the record and its start claim copy the intent.
- Add the regression test.

## Log

### 2026-10-15

Filed from the read-only diagnosis of the TL's failed ariadne:4 reboot after #597 landed (TL GO 88674e9e, priority 1: slot startup).
