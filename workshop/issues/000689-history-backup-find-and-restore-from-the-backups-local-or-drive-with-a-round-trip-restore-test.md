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
flow: {kind: full, provenance: inferred}
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

### 2026-10-19 — implementation admitted

- Plan quality CLEAN after PQ-1 disposition; change-code passed with derived estimate 4.93 h. Estimate review INFO: cross-component fixture/race/Linux/full-suite work and active operator qualification are tracked explicitly in this Log; idle operator scheduling is excluded. The library discount covers existing mechanics; remaining ownership/evidence/publication contracts stay custom work.
- M1 RED/GREEN: compatible label/key codecs, index refresh/history/unknown-field codecs, UTC selection, read-only catalog, local catalog-loss discovery, ownership-stamped staging and atomic index store. Focused catalog/store tests passed with race; codec fuzz passed 51,035 executions. M1 is not closed yet; CLI/Drive integration and normal guards remain.
- Installed rclone v1.75.2 refused files-from-raw with max-size. Switched the new exact-name download seam to tested rooted filter rules, preserving actual max-size/hard max-transfer caps. Conformance downloaded the selected 5-byte label, skipped a 9-byte object under a 4-byte max-size, and refused a 4 MiB transfer under a 1 MiB cap without retained partial output. Upload transport unchanged.
- Independent existing fixture defect filed unclaimed as pair#696. Fractional mtimes make pair byte sizes unequal, but TestDriveMirrorBatchBudgetDefersCompletePairs budgets only pair 0. A controlled .1Z/.123456789Z case produces totals 3481/4009 and reproduces second-pass deferral with byte-identical legacy labels. No unrelated fix/assertion weakening. Hub message 62c9378c-5ebc-4036-a3a0-1b40c6310719 submitted after an initial pre-admission unavailable refusal.

### 2026-10-19 — M1 qualification checkpoint

- CLI create/find/local-sync fixture passed with public JSON output and no private filename/root exposure; find does not publish an index. An invalid optional Drive locator no longer blocks local discovery (RED invalid-remote refusal, then GREEN focused view/sync tests).
- Counted local N=8/16 work: two shard passes and 2N record visits, one catalog directory enumeration and N label reads. Catalog-loss fallback performs three kind/year/month enumerations and 2N label reads. A repeated-lookup control increases measured counters. Combined catalog/local/view/sync races passed (20.261 s).
- Drive label sync N=8/16 uses <=4 subprocess calls, two flat listings and one label batch, zero ciphertext downloads/checksum fallback. Partial/conflicting observations preserve the prior snapshot. Index codec fuzz passed 1,194 executions; broader M1 focused race passed (50.155 s).
- Full changed-package run: archivecmd and archivetest passed; historyarchive failed only the previously filed #696 fixture at the same deferred-retry boundary. Artifact coverage initially found three stale ignored runtime mirror files absent from current source; regeneration through runtimebundlegen cleared it and artifactpath passed (13.461 s). Repository lint passed (11.748 s). No assertion/deadline change, retry qualification or broad-suite success is claimed.
- Preparing the pushed checkpoint and recorded full suite. M1 mandatory review and M2 implementation/live acceptance remain outstanding.

## Revisions

- 2026-10-18: clarified the later sync-index Spec supersedes remote queries from find. Recorded the hub-approved label/location evidence and atomic refresh contract; original Done-when obligations are retained without reduction. Added the durable plan with M1/M2 review boundaries.

- One-pass read-only plan editing aid identified two gaps before code: Drive-only source-root overlap is knowable only after decrypting the manifest, and a staging pathname/lock alone does not prove ownership of existing contents. The durable plan now requires the post-manifest check and an ownership-stamped staging protocol with refusal on unknown contents; tests preserve a preexisting operator staging directory and a target below a recovered source root. Created files/directories also share a whole-request bound so small-file/directory growth cannot bypass byte admission.

- Hub d9acf9a3-51ac-4eda-9dc9-1a70065e04ac approved the full plan as written and instructed change-code. They will arrange the operator real-day Drive restore when M2 is ready; that acceptance remains mandatory before landing.

- 2026-10-18 PQ-1 (function-level-test-contract): consolidated the class of repeated unit-case inventories into named-function adversarial strategies and mechanical guards. Kept catalog-loss and real-key/operator acceptance drills and all approved invariants; no implementation begun.
