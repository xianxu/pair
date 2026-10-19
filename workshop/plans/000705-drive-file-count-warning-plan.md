---
collaboration-mode: discussion
---

# Drive file-count warning implementation plan

> **For agentic workers:** Consult AGENTS.md §3. Keep this one atomic close. Use bounded delegation for independent pure tests or fixture qualification; SDLC owns the fresh boundary review.

**Goal:** Report the Drive destination's observed file count and warn once daily above 20,000, using only listings the pass already performs.

**Architecture:** Add an optional runtime observation to DriveReport. Project both human output and the existing bounded JSON report from that observation. Extend #700's exact-key daily gate and extract its existing delivery/log body once so database and Drive warnings share one effect path.

**Tech Stack:** Go; existing Rclone/FakeRclone, continuous command report, archive scheduler and primary-hub notification seam.

## Scope and decisions

- PAIR_ARCHIVE_DRIVE_MAX_FILES is an inherited positive decimal integer, unset = 20000. Zero, negative, malformed and overflowing settings fail before config/source/network work, like the database-age setting. No config file or machine setting change.
- Count every returned non-directory entry, including unrelated files and duplicate file names. Directories are excluded from this report count, while existing capacity accounting still counts all entries. No estimate from bundle/receipt counts.
- Reuse successful ObserveFlatInventory results. A successful final listing replaces the before observation; a failed final listing keeps the explicitly labeled before observation. Failure before any complete valid listing, empty candidate sets, unavailable Drive or skipped remote work leave the count unknown, never zero. No added remote call even to fill missing observations.
- Warning boundary is strictly Files > configured threshold. Equality does not warn. Wording names the destination/count and says: Ask the hub to arrange year subfolders on Drive. #706 owns layout changes; this item never moves files.
- Daily identity is the exact destination string plus pinned FolderID, separate from database identity. Repeated before/after counts, threshold/health changes, uncertain notification, or reconnect do not rearm that UTC day's reservation. Existing rollover/backward-clock/cap/restart rules remain unchanged.
- Reuse primary-hub resolution, status/attention/desktop delivery and continuous.log retention from #700, including no-hub fallback and consuming admission before IO. No new store, queue, worker, timer or retry.
- Runtime report v1 grows additively. Older reports without the new observation remain readable as unknown count. MaxDriveFiles is optional for legacy reports but must be positive whenever an observation exists; null or missing required observation fields, invalid count/phase/destination, duplicates/aliases and no-pass outcomes carrying observations refuse the whole report. Unknown additive fields remain accepted and original bytes retained. Persisted catalog/manifest/config/receipts stay unchanged.
- Expected production diff may exceed the quick shell's 100 added lines once explicit protocol validation/shared delivery are included; use single-pass full flow, no Mx boundary.

## Core concepts

### Pure entities

| Name | Lives in | Status |
| --- | --- | --- |
| DriveInventoryObservation | cmd/internal/historyarchive/drive_count_model.go | new: Destination, FolderID, Files, Phase (before/after); optional in DriveReport |
| Drive count threshold/rendering | cmd/internal/historyarchive/drive_count_model.go | new: ParseDriveMaxFiles, observation Line/Warning and count projection from decoded entries |
| ContinuousCommandReport | cmd/internal/historyarchive/continuous_report.go | modified: MaxDriveFiles plus strict optional observation validation/presence |
| archiveWarningKey / archiveWarningGate | cmd/internal/couchcmd/archive_alert_model.go | modified key: explicit destination and folder identity; reuse unchanged Reserve transition |

One Drive report owns at most one current observation. Absence means unknown; observed zero is a valid separate value. Phase records the point observed, not an upload outcome. The shared threshold predicate/rendering serves CLI and Couch (ARCH-DRY). The gate remains single-owner volatile state, capped at 10,000 keys, including both warning classes.

### Integration points

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| DriveMirror.Run | cmd/internal/historyarchive/drive.go | modified | existing before/after ObserveFlatInventory; no new Rclone invocation |
| Continuous command | cmd/internal/archivecmd/continuous.go | modified | threshold env, human row and typed report |
| archiveScheduler.tick / warning delivery helper | cmd/internal/couchcmd/archive_schedule.go | modified | existing admission, callback and private log; common effect path for both classes |
| FakeRclone / FakeRemoteObject | cmd/internal/historyarchive/archivetest/rclone.go | modified | retained objects/duplicates/copy failures; add directory representation to qualify the actual listing seam |

The new model file joins artifactpath/manifest.go. No changes to archive_notify owner resolution are needed. Fake objects and all backing stores live only in test temp directories. Existing real rclone local-backend conformance remains the adapter check; no live Drive or desktop operation during development.

## Visible states and recovery

Actions: ask the hub to inspect unavailable Drive reporting; ask the hub to arrange year subfolders.

| Internal evidence / failure | Visible state | Recovery action | Possible loss |
| --- | --- | --- | --- |
| Valid final observation at/below limit | Count labeled after pass; no capacity warning | None | No new data-loss claim |
| Valid before observation only, later listing/copy fails | Count explicitly labeled before pass; existing remote failure note remains | Ask hub to inspect remote reporting | Newer count unknown; local backup remains intact |
| No valid listing, no candidates, remote skipped/unavailable | Drive file count unavailable; no fabricated zero or capacity warning | Ask hub to inspect remote reporting if unexpected | Capacity unknown |
| Valid observation above limit, eligible day | One primary-hub warning naming destination and count/limit | Ask hub to arrange year subfolders | No files moved or deleted; flat-directory growth remains |
| Same destination warns again that day or notification is uncertain | Count remains in report/log; no second attempt | Use the existing warning to ask hub for year subfolders | Desktop display may be unconfirmed |
| Primary hub unavailable | Existing status + desktop attempt and explicit unavailable text/log; no attention item | Ask hub to arrange year subfolders | Same delivery uncertainty; reservation consumed |

Before/after observations survive later remote verification/cleanup errors. Cancellation stops output attempts; it does not turn a retained observation into a successful upload. A Couch restart permits one fresh warning, as #700 documents.

## Constraints and architecture

- Daily batch path only. For N entries per existing listing, counting is one O(N) pass and O(1) extra observation state. Reuse the before loop's capacity accounting where appropriate; at most one counting pass per already-returned inventory. Existing 100,000-entry / 64 MiB reader limits remain authoritative; malformed/overflow listings give no new observation.
- Counted stateful tests at N=8 and 16 preserve exactly two content listings per ordinary or receipt-only pass, one destination resolution, existing copy/hash counts, and report 2N uploaded files. No extra source/catalog reads.
- ARCH-DRY/PURE: one pure count/threshold/rendering model and one existing warning effect path. ARCH-MOCK: retained FakeRclone plus actual subprocess/fake presenter seams.
- ARCH-PURPOSE: both daily human/JSON reports and Couch warning are covered; only folder migration belongs to #706. ARCH-CONSTRAINTS: bounded existing inventories/protocol and no extra IO/worker.
- ARCH-SECURE: external inventory is decoded at the existing strict boundary; runtime report validates exact observation identity and required count before automation, with no credentials read or created. ARCH-ORDER: preserve last valid evidence and reuse reservation-before-effect; no time-based retry authority.
- ARCH-FUNERAL: creates no durable family; optional observation dies with its pass, shared gate/log keep their existing bounds/lifetimes. ARCH-EVOLVE: additive runtime report; no persisted codec change, unknown fields retained in raw log.

## Implementation tasks (one close)

- [ ] Add failing pure observation/threshold/codec tests using the strategies below; implement the small model and optional report shape; run green and commit.
- [ ] Add failing stateful Drive fixture qualification, then observe before/after counts at the existing seams, preserving capacity and call counts; run green and commit.
- [ ] Add failing CLI/scheduler qualification, wire shared delivery and destination admission, document behavior and run targeted/source inventory guards; commit and open PR.
- [ ] Run the pushed recorded full suite; close through SDLC's one fresh review, publish final evidence before entering merge, and land on SHIP. Hub owns project tick and natural live observation.

### Function-level test strategies

| Function / seam | Adversarial strategy and concrete regression |
| --- | --- |
| ParseDriveMaxFiles | Table/fuzz: default and positive boundary; reject zero/negative/noninteger/overflow before CLI work. |
| DriveInventoryObservation.Line / Warning | Below/equal/above default and override; before/after labels; zero vs absence; bounded Unicode destination with one recovery instruction. |
| Count from decoded inventory | Pure map with duplicate non-directory names and directories; exact entry count, not map key count; never mutate input or change capacity semantics. |
| Encode/DecodeContinuousReport | New observation roundtrip; legacy absent observation; required fields including zero-count presence; negative/overbound counts, bad phase/identity, invalid threshold and no-pass combination; existing malformed/duplicate/alias/oversize and unknown-field raw-log guarantees remain. |
| DriveMirror.Run | Extend TestDriveMirrorCountedBacklogAndRerun at N=8/16; zero added list/resolve/copy/hash invocations. TestDriveMirrorFinalListingRejectsLateDuplicate proves final observation replaces before. Early/final listing failure, partial copy failure, cancellation and no candidates preserve exact known/unknown evidence. Fake directory entries exercise production DecodeFlatInventory. |
| runContinuous CLI | Fake pass/report callback records no work on invalid env; configured limit reaches typed report and human warning; partial/unavailable Drive keeps database report useful. No real source/network opens. |
| archiveScheduler.tick / shared warning helper | Fake clock and stateful fake pair report with DB + Drive warnings. Exact destination+FolderID dedup, different destinations, below/above repeat and day/restart, unavailable primary and uncertain output share existing admission/output semantics; no duplicate retry or second log writer. Keep #700 counted row regression. |

Targeted commands: go test ./cmd/internal/historyarchive ./cmd/internal/archivecmd ./cmd/internal/couchcmd -run 'Test(Drive|Continuous|Archive|RunArchivePass)|Fuzz' -count=1; then the exact artifact source/reference/dead-symbol/documentation guards. Behavioral red must identify missing count/warning or incorrect listing/capacity rather than relying only on undefined symbols. Full suite: sdlc local-check run --name suite on pushed head, with normal machine queue. No pair:0 staging or real alert requested; propose fake-backed acceptance and next natural daily count as live confirmation to the hub.

## Revisions

- 2026-10-19: dispatched after #700 landed. No-candidate passes currently do not list Drive; preserving that unknown observation is required by the no-extra-call contract. Successful before evidence survives failed final listing, clearly labeled. Keep #706 year/month migration out of this item; it waits for #689.
