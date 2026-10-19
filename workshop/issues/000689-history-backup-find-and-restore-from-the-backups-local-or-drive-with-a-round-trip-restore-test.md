---
id: 000689
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'f03d43448f7e1c0f9910ab253ce0a066159a0c1f' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-18T23:32:01-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:6
    worktree: /Users/xianxu/workspace/worktree/pair-slot6/pair
    repository: github.com/xianxu/pair
---

# history backup: find and restore from the backups (local or Drive), with a round-trip restore test

## Problem

pair archive can create and upload backups (continuous pass, Drive sink #678) but has no way to read them back: no restore command and no index. The operator asked (2026-10-18) whether we have tested reading the backups back, for example fetching the logs from a few days selectively. We have not: verification today proves the bytes in Drive match the local ciphertext, not that a bundle decrypts, unpacks, and yields the files someone wants. Untested restore means unproven backup.

## Spec

Each bundle already has a plain JSON label (kind, from/to time range, file_count, raw and encrypted bytes, recipients, sha256), stored beside it locally and in Drive. Add: (1) `pair archive find --kind K --since T --until T` listing matching bundles from the labels (local catalog first, Drive listing when asked); (2) `pair archive restore ID... --to DIR [--from drive]` fetching the bundle (from Drive by its flat content name if not local), verifying its sha256 against the label, decrypting with the operator's GPG key via gpg-agent, and unpacking into DIR (never over live data); optional path filter inside the bundle. (3) A restore drill: an automated test with a throwaway GPG key round-trips create, upload (fake rclone), find, restore, and compares files byte for byte; plus a documented one-time live drill the operator runs against Drive (decryption needs their key).

**Label index across locations (operator, 2026-10-18).** The `.json` labels stay local too (they already sit beside
each bundle and in the catalog), and they are the index: `find` queries the local labels, never a remote. With more
than one destination (local, Drive, later others), each label records where its bundle is present and verified (the
#678 upload receipts already hold Drive presence). `pair archive sync-index [--from drive]` downloads only the labels
(small: about 1 KB each) from a destination and rebuilds the local index of which bundle is where, so a machine that
lost its local catalog can reconstruct it from Drive alone, then restore selectively.

## Done when

- find/restore exist; the round-trip test passes with a throwaway key.
- sync-index rebuilds the location index from Drive's labels alone (test: delete the local catalog in a fixture, rebuild, and find the same bundles).
- The operator restores one real day of Claude transcripts from Drive into a scratch folder and the files match the originals.

## Plan

Durable plan: [find, restore and label/location index](../plans/000689-find-restore-index-plan.md).

- [ ] M1 — Preserve additive label/index fields; implement local-only find and atomic local/Drive index synchronization, including catalog-loss rebuild.
- [ ] M2 — Implement verified fresh-destination restore, throwaway-key full round trip, and operator-owned real Claude-day Drive drill; qualify and land before retention.

## Log

### 2026-10-18


### 2026-10-18 — design checkpoint

- Claimed while unowned; start-plan refreshed the stale ariadne tooling, then was rerun as instructed to start this issue branch. No implementation edits or tests/builds initiated by this worker; the tool-owned catch-up rebuilt its own binaries.
- Read the entire Spec, including the local-only find/sync-index addition. Existing CatalogRecord requires a private Manifest, so public labels cannot reconstruct its authority. Uploaded labels are immutable and cannot gain location fields without invalidating #678 hashes.
- Hub bb8ab7c9-1c77-43ae-9d13-d2e28ff82dd7 approved separate index envelopes, observed/verified evidence, no ciphertext download during sync, complete refresh marking unavailable while retaining history, and no changes from failed/partial refreshes. The plan uses one atomically replaced snapshot, preserves unknown fields at every label/index nesting level, and distinguishes publication uncertainty from a confirmed failed observation.
- Restore always verifies staged ciphertext against the label before GPG/zstd and validates all tar members, including unselected files, before exclusive publication to a new scratch target. The real Drive/day/key drill remains the operator's. No retention or live-data import is included.
- Read-only component mapping reused existing safe paths, catalog/receipt/hash helpers, rclone runner/fake and isolated GPG fixture. Full design and initial batch limits are awaiting hub approval before change-code.

## Revisions

- 2026-10-18: clarified the later sync-index Spec supersedes remote queries from find. Recorded the hub-approved label/location evidence and atomic refresh contract; original Done-when obligations are retained without reduction. Added the durable plan with M1/M2 review boundaries.

- One-pass read-only plan editing aid identified two gaps before code: Drive-only source-root overlap is knowable only after decrypting the manifest, and a staging pathname/lock alone does not prove ownership of existing contents. The durable plan now requires the post-manifest check and an ownership-stamped staging protocol with refusal on unknown contents; tests preserve a preexisting operator staging directory and a target below a recovered source root. Created files/directories also share a whole-request bound so small-file/directory growth cannot bypass byte admission.

- Hub d9acf9a3-51ac-4eda-9dc9-1a70065e04ac approved the full plan as written and instructed change-code. They will arrange the operator real-day Drive restore when M2 is ready; that acceptance remains mandatory before landing.
