---
id: 000497
status: open
deps: []
github_issue:
created: 2026-10-12
updated: 2026-10-12
estimate_hours:
card_mirror: '3a52d7d0c72932c9029e66835d3fc471bc8da596' # card fields mirrored from issue-cards; edit via sdlc
---

# storage: rotate and cap pair's wrap-events logs (41 GB under ~/.local/share/pair/repos)

## Problem

`~/.local/share/pair/repos` holds ~41 GB, mostly `wrap-events-*.jsonl` logs (the largest 4.6 GB, last written Sep 16). Nothing rotates or caps them.

## Spec

Rotate wrap-events logs (by size and age) and cap the total per repo; compress or delete old segments; make it part of pair's storage GC (`storagegc`), with the retention stated in the atlas. Report what a first sweep would remove before removing anything.
**Operator permission (the operator, 10-12):** every operation that changes this machine (creating or mounting a volume, moving or deleting caches or logs, changing launch settings, renicing processes) is proposed first, with the exact command, and runs only after the operator approves it, through the TL. Code changes and tests in the repo don't need that; anything outside the repo does.

## Done when

- Rotation and a cap are in place with tests; a first sweep (approved by the operator after a dry-run listing) reclaims the space; retention documented.

## Plan

- [ ]

## Log

### 2026-10-12
