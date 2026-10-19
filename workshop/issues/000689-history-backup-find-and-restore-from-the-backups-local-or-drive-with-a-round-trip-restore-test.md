---
id: 000689
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours: 4.93
collaboration-mode: collaboration
card_mirror: 'ab8c526eefb220e8dbf1e618cd0980751889f545' # card fields mirrored from issue-cards; edit via sdlc
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


## Estimate

*Produced via `brain/data/life/42shots/velocity/estimate-logic-v3.1.md` against `baseline-v3.1.md`. Method A only.* Calibration is stale, so this is provisional. The approved detailed plan earns the 0.2 design multiplier and 15% design buffer. Familiarity is 1.0: #678's Drive, filesystem and fake seams are already familiar.

Library check: encoding/json, archive/tar, existing safe filesystem/commitoutcome helpers and x/sys exclusive-rename bindings cover the novel modules' mechanics; use half the greenfield design midpoint (1.25/2), then the 0.2 spec discount = 0.125 per concern. Smaller extensions use 0.15*0.2=0.03. Drive integration uses design midpoint 2*0.2=0.4. Implementation values are already 40% of the v2 table (0.4 for smaller modules, 0.7–0.8 for greenfield modules, 1.0 for API integration, 0.4 per review). No second scale is applied. Total design 1.26 h, scaled implementation 3.48 h, yielding 1.26*1.15+3.48=4.929 h (rounded 4.93 h). Operator scheduling/idle time is excluded.

- compatible label codec/shared keys: smaller-go-module.
- index entities, merge and find: greenfield-go-module.
- atomic store and owned staging: greenfield-go-module.
- read-only catalog and local discovery: smaller-go-module.
- batched Drive label synchronization/stateful fake: api-integration.
- isolated streaming decoder: greenfield-go-module.
- bounded tar verifier/extractor: greenfield-go-module.
- restore orchestration/exclusive publication: greenfield-go-module.
- find CLI: smaller-go-module.
- sync-index CLI: smaller-go-module.
- restore CLI: smaller-go-module.
- docs and drill: atlas-docs.
- M1 review: milestone-review.
- M2 review: milestone-review.
- final close review: milestone-review.
- installed rclone conformance and operator drill preparation: real-api-discovery.

```estimate
model: estimate-logic-v3.1
familiarity: 1.0
item: smaller-go-module design=0.03 impl=0.16
item: greenfield-go-module design=0.125 impl=0.28
item: greenfield-go-module design=0.125 impl=0.28
item: smaller-go-module design=0.03 impl=0.16
item: api-integration design=0.4 impl=0.4
item: greenfield-go-module design=0.125 impl=0.28
item: greenfield-go-module design=0.125 impl=0.32
item: greenfield-go-module design=0.125 impl=0.32
item: smaller-go-module design=0.03 impl=0.16
item: smaller-go-module design=0.03 impl=0.16
item: smaller-go-module design=0.03 impl=0.16
item: atlas-docs design=0.025 impl=0.08
item: milestone-review design=0.02 impl=0.16
item: milestone-review design=0.02 impl=0.16
item: milestone-review design=0.02 impl=0.16
item: real-api-discovery design=0 impl=0.24
design-buffer: 0.15
total: 4.93
```

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

- 2026-10-18 PQ-1 (function-level-test-contract): consolidated the class of repeated unit-case inventories into named-function adversarial strategies and mechanical guards. Kept catalog-loss and real-key/operator acceptance drills and all approved invariants; no implementation begun.
