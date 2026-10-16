---
id: 000600
status: working
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
severity: critical
started: 2026-10-15T22:46:02-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Fresh reboot onto a new address fails recovery authority: SourceRevision names the old record

## Problem

`couch --reboot ariadne:4 --confirm` (Fresh onto a new address) fails every time with `pair: recovery authority changed: explicit recovery source revision, time or agent changed`. In couchcore `launchRecoveryFreshAddress`, the intent keeps `SourceRevision` = the OLD record's revision, while the authority's `RecordRevision` is the NEW address record's revision, which starts at 1. The launcher's `validateRecoveryRecord` refuses when `SourceRevision > RecordRevision`, so any reboot whose old record is past revision ~1 is refused. On disk, ariadne:4: new record 1-ariadne-52, Fresh intent source_revision 3 (old 1-ariadne-51, archived at rev 4). The tests missed it because the couchcore fixtures fake `ValidateRecoveryAuthority`.
