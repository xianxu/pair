---
collaboration-mode: discussion
---

# Drive month folders implementation plan

> For agentic workers: consult AGENTS.md Section 3. Use superpowers-executing-plans for this session-warm archive work. The SDLC binary owns the fresh-context reviews at M1, M2 and close.

**Goal:** Put new Drive bundle/label pairs under the same UTC year/month as their local bundle keys, while preserving discovery, verification and restore of legacy flat pairs.

**Architecture:** One shared layout model derives physical destinations and validates recorded locations. Upload visits the legacy root and only the current/pending months; explicit sync-index enumerates the bounded two-level layout to rebuild after local loss. Each physical folder has one pinned Drive ID and flat uniqueness checks. Reuse the existing pass lock, immutable pairs, staging, upload state machine, receipt namespace and index evidence rules.

**Tech stack:** Go, existing RcloneRunner/ExecRcloneRunner, stateful archivetest.FakeRclone, existing safe filesystem and raw-extension JSON helpers. No new dependency or remote service.

## Scope and choices

- Recommended: new writes in month folders, dual-layout reads, no automatic move or deletion of existing remote files. This delivers the requested layout without risking a migration of the only remote copy.
- Moving the whole legacy tree first makes cutover depend on a destructive remote migration; leave that optional step separate.
- Recursive listing on every upload is simpler but grows with all archived history and violates the listing contract. Explicit disaster-recovery sync-index may enumerate all supported year/month folders under its existing metadata bounds.
- Month is label.From in UTC, exactly the calendar rule used by BundleKeys. Content IDs, local keys, immutable labels, ciphertext and manifest ID calculation do not change.
- Receipt files stay at configured-root Namespace()/bundle-ID.json. Destination inside each receipt names its actual physical folder (legacy root or root/YYYY/MM), and FolderID identifies that folder. This preserves old lookup keys and avoids scanning every receipt subdirectory. No schema version bump or new location field is needed.
- Existing receipt attestation retains its historical meaning; an AlreadyRecorded count is not a claim of a new remote verification. Restore always observes and hashes actual ciphertext.

## Core concepts

| Entity | Kind | File | Status |
|---|---|---|---|
| DriveArchiveLayout, month grouping and recorded-location validation | PURE | cmd/internal/historyarchive/drive_layout.go | new |
| DriveFolderState/Event/Effect and transition function | PURE | cmd/internal/historyarchive/drive_folder_model.go | new |
| UploadReceipt and UploadObject compatible codecs | PURE | cmd/internal/historyarchive/drive_model.go; drive_receipt_json.go | modified/new |
| UploadState transition model | PURE | cmd/internal/historyarchive/drive_model.go | reused; extend only if a new upload effect is necessary |
| RestoreBundlePlan recorded locations and grouping | PURE | cmd/internal/historyarchive/restore_model.go | modified |
| LabelIndex/IndexRefresh completeness and receipt precedence | PURE | cmd/internal/historyarchive/label_index_model.go | reused, per physical destination |
| Pinned child-directory resolution/creation | INTEGRATION | cmd/internal/historyarchive/rclone_folders.go | new; wraps RcloneRunner |
| DriveMirror multi-folder coordinator | INTEGRATION | cmd/internal/historyarchive/drive.go; drive_months.go | modified/new; catalog, receipts, staged files, rclone |
| FakeRclone hierarchical folders | INTEGRATION | cmd/internal/historyarchive/archivetest/rclone.go | modified; portable retained folder/object map |
| DriveIndexSync multi-folder snapshot | INTEGRATION | cmd/internal/historyarchive/label_index_drive.go | modified; index lifetime lock and rclone |
| Receipt-location projection | INTEGRATION | cmd/internal/historyarchive/label_index_view.go | modified; root-scoped receipt reads |
| Restorer input acquisition | INTEGRATION | cmd/internal/historyarchive/restore_input.go | modified; pinned folder groups and ciphertext hashing |
| Per-folder count report and warning consumers | PURE/INTEGRATION | drive_count_model.go; continuous_report.go; couchcmd/archive_schedule.go | modified; report validation is pure, scheduler output is integration |

Layout is the single owner of root/UTC-month relationships, used by uploads, receipt projection, sync-index and restore (ARCH-DRY). RcloneRunner remains the only external-command seam. Directory planning and folder transition tests use values only. FakeRclone models folder IDs/parents, duplicate sibling names, remote bytes, partial effects and uncertain acknowledgments across calls (ARCH-MOCK).

## Invariants and execution flow

### Folder binding

1. Resolve the configured root once. Never create the operator-configured root implicitly.
2. Observe the root flat inventory once, retaining its legacy files and year-directory entries. For each needed year, observe its child directories once. Resolve month directories through the already pinned parent ID; never resolve an ambiguous name by choosing the first entry.
3. Create a missing year/month only for an admitted upload, through a pinned parent. Re-observe after the one mkdir attempt. Current-month inspection alone does not create empty folders.
4. Exactly one directory with a valid ID yields Ready. Duplicate folders or a file with the expected directory name yields Conflict. A failed or unconfirmed lookup/create remains Pending, with no receipt or success count.
5. No second mkdir in the same pass after an uncertain result. A later pass observes first; any observed duplicate is refused and reported, never repaired automatically. Folder observations prove identity at that observation, not permanent absence of late remote effects. This uses the existing serialized-client operational model of #678; no new mutation journal or background reconciliation loop.

The pure folder transition function owns Planned -> Observing -> Ready/Missing/Conflict/Unknown, Missing -> Creating -> Reobserving -> Ready/Conflict/Unknown. Cancellation/IO failure emits no subsequent create or copy effect. The IO shell executes effects and feeds results back; it cannot assign Ready directly (ARCH-ORDER).

### Upload

1. Under the existing continuous lifetime lock, project all catalogued candidates for the configured sink once, irrespective of selected source kinds. Read each candidate receipt at the unchanged configured-root key once.
2. Validate any recorded destination as exactly the configured root or that label's canonical month child, with matching object names and hashes. Valid old receipts stay old; do not re-copy or move them merely to change the layout.
3. Pending candidates are grouped by month before remote work. Visit the current month if it exists and months containing pending admitted pairs. Do not enumerate other historical months. A fully empty catalog can retain the existing no-network behavior.
4. Check each pending pair against the legacy flat inventory and its month inventory. A complete unique matching legacy pair can be adopted in place. A partial, duplicated or conflicting legacy pair remains pending with a diagnostic; do not silently create a second copy in the month folder. Check uniqueness independently inside every used physical folder; identical IDs at two valid recorded locations are not a global uniqueness invariant.
5. Fresh pairs with no legacy objects target the month folder. Reuse PrepareBatchObject, hard links/verified cross-filesystem copies, one global 4 GiB logical staging/admission budget and the current complete-pair rule. Process folder batches serially, each with one copy, eight transfers and immutable/ignore-existing options. Per-folder batches preserve exact folder-ID pinning; batching across relative subpaths would lose that fence.
6. Reuse before/after flat listings and one optional downloaded-checksum batch per active physical folder. Even a partially failed copy may publish receipts only for pairs whose two unique objects verify. Cleanup all owned staging on success, failure and cancellation; no remote deletion or repair.
7. Aggregate Pending/Deferred/Verified/AlreadyRecorded and checksum-path counts without losing failures in one month when later months can proceed. Folder ambiguity is local to that folder; root resolution or global bounds failure ends remote work visibly. Local backup success remains independent of Drive availability.

### Index and restore

1. Receipt projection reads the same root namespace and uses receipt.Destination as the physical locator after shared layout validation; keep complete-observation versus receipt precedence unchanged.
2. Explicit sync-index lists root plus strict YYYY/MM directory levels, no recursive lsjson. Preserve legacy root labels. Accept only the supported two-level grammar, valid IDs and unique sibling folder names; unexpected directories are ignored without traversing them. Ambiguous supported folders fail the refresh.
3. Under one existing index-store lifetime lock, collect all per-folder label observations in private staging. Apply the existing per-destination IndexRefresh rules in memory and publish one snapshot only after the entire requested inventory is complete and stable. Empty folders still receive complete markers. A failed/partial folder pass publishes nothing.
4. Compare directory bindings before/after the full observation. Previously known physical folders absent from a complete tree observation mark their locations unavailable while retaining prior proof/history and advancing the relevant complete observation; no failed listing can invalidate anything. Use the existing per-locator marker model; do not introduce a second evidence clock or root receipt authority. Test disappearance, root rebind and stale/new receipt ordering explicitly.
5. PlanRestore retains validated Drive locations for each selected label. Acquisition groups selected IDs by physical locator/FolderID; prefer current available evidence deterministically, preserve the legacy path, and require live binding/size/uniqueness confirmation before download. Refuse a stored locator outside the configured root/canonical month. If no recorded remote location exists, require sync-index rather than guessing another remote path.
6. Download only selected ciphertext in bounded per-folder batches into the existing private restore stage. Share aggregate byte/entry limits across all groups. Hash every acquired ciphertext before any decryption; keep full member verification and exclusive fresh-directory publication unchanged. A corrupt or ambiguously bound source fails rather than silently switching to another copy.

## Compatibility and trust

- Labels and the label index already preserve unknown members; retain their existing byte caps, strict required fields, duplicate-key rejection and raw unknown values. No rewrite of uploaded immutable labels.
- Add extension maps and compatible JSON methods to UploadReceipt and nested UploadObject using existing decodeExtensibleObjectSchema/marshalExtensibleObjectLimit helpers. Keep 8 KiB receipt bound, strict known-field types/required fields, case-fold collision and duplicate rejection. Preserve old no-extension bytes/field order. Replace direct Go struct equality with explicit known-field comparison and appropriate preserved-extension handling.
- Destination continues to mean the physical rclone folder. Receipt storage namespace continues to mean the configured replication root. An old flat receipt remains valid without conversion; a newer month receipt is read by the new compatible reader. Never use a Drive record as local archive authority (ARCH-EVOLVE).
- Treat folder names/IDs, listing JSON, labels, receipts and index bytes as untrusted. Shared typed layout parsing precedes IO. Truncation, malformed known fields, unsupported paths or checksum conflicts produce visible refusal with no receipt/index promotion. Use argv arrays, sanitized bounded diagnostics and private fixtures; never read rclone credentials or GPG private key material.

## Operating envelope and counted cost

This is an optional batch after local archive work, not a UI path. Retain 30-minute Drive phase, 60-second individual inventory resolution, serial folder batches, eight rclone transfer/checker workers, global 4 GiB staged bytes, existing 100,000-candidate/64 MiB projection bounds and per-flat-inventory byte/entry caps. Sync-index retains one global 64 MiB snapshot/label budget, 100,000 records and 1,024 physical complete-destination cap; exceeding a bound fails without publishing a partial snapshot. Folder creation and local projections check destination growth bounds before new remote effects; no unbounded traversal or downloaded-ciphertext fallback in sync-index.

For N candidates, M needed month folders, Y needed years and E total entries in visited flat inventories: one catalog pass plus receipt lookups O(N), grouping O(N + M log M), directory work O(Y + M), inventories O(E), copy/hash calls O(M), no per-object process. Root legacy inventory is O(legacy files + year entries); it no longer gains new bundle files. Explicit index rebuild visits all supported bounded folders because recovery after local loss cannot know omitted months.

Count tests at 8/16 pending pairs in one month: fixed listing/copy/hash invocation counts independent of N. Repeat at one/two months: per-folder listing, copy and optional hash counts grow linearly in M. Populate many unrelated historical months and prove upload never lists them; add confirmed receipts in those months and prove no extra historical IO. Assert staged bytes <=4 GiB across groups and metadata bounds across index groups.

Durable residue: remote year/month directories are archival namespace retained with their immutable pairs; no new process or per-pass durable registry. Existing receipt/index writer bounds remain enforced. Upload and restore private stages are swept by their current owned cleanup paths. Empty directories created before a canceled upload may remain; a later pass reuses them, never recursively removes foreign remote content (ARCH-FUNERAL).

## Visible states and recovery

Actions: retry the pass, run sync-index, inspect/repair remote ambiguity, select an explicit restore source.

| Internal state/failure | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| New month absent, creation and copy confirmed | Verified pairs with physical month location | Retry pass normally | None |
| Valid legacy receipt or adopted legacy pair | Already recorded / verified at root | Select explicit restore source | None; no automatic migration |
| Remote/tool unavailable or uncertain mkdir/copy | Pending with folder/error and retained local copy | Retry pass, observing first | No local loss; partial remote objects may exist |
| Duplicate folder/object, partial legacy pair or hash conflict | Conflict/pending; no new receipt | Inspect/repair remote ambiguity | No automatic deletion; existing copies retained |
| Byte/entry/time limit reached | Deferred/pending counts with bound | Retry pass | No local loss; progress already verified remains |
| Complete index refresh with removed/rebound folder | Unavailable current location, historical proof retained | Run sync-index after repair | Discovery availability only; history retained |
| Failed/partial index refresh | Prior snapshot retained, error | Run sync-index | No snapshot loss |
| Restore stale binding or corrupt bytes | Explicit failure, no published output | Select explicit source after inspection | No live-root change; owned scratch removed |

Covered: cancellation and late/uncertain external effects, duplicate names, partial copy, unavailable service, missing/changed folders, catalog loss and stale receipts. Not covered: automatic flat-file migration, source-kind expansion, retention, unattended key use or a new scheduler. These are separate issues or explicitly optional work.

## #710 coordination

The shared upload path must project the whole catalog, not only bundles created in this pass or selected by --kind. Add cross-kind/older-backlog tests and preserve pending counts/errors under month grouping. Future catalog bundles use their labels' month through the same generic path; no list of six current source kinds is added here.

Existing ContinuousPass returns on local collection errors before calling DriveMirror; #710 gap 5 owns that collection/upload scheduling diagnosis and stale-backup policy. #706 does not silently expand into new source kinds, Couch snapshots, catalog encryption/bootstrap or launchd. Send the hub this overlap explicitly so #710's owner builds on the month coordinator rather than replacing it.

## Function-specific test strategies

| Risky function | Adversarial input class and mechanical guard |
|---|---|
| DriveArchiveLayout.MonthDestination / ValidateRecordedDriveLocation | Fuzz arbitrary roots, UTC ranges, IDs and stored locators; independently assert BundleKeys month agreement, root containment and rejection without IO. |
| StepDriveFolder | Generate bounded event sequences, including invalid and late events; assert independently that Ready requires a unique positive observation, no second create occurs in one attempt, and failure/cancel cannot emit copy or receipt effects. Exhaust the state/event product, not just hand-picked happy sequences. |
| DecodeUploadReceipt and receipt/object MarshalJSON/UnmarshalJSON | Fuzz malformed/duplicate/case-folded JSON under fixed byte limits; assert strict known fields, raw nested unknown preservation, deterministic re-encoding and byte identity for legacy no-extension fixtures. |
| DriveMirror.Run | Stateful fake with injected interruption before/after each remote effect; independently require unique matching pair evidence for every published receipt, retained progress without receipt for uncertain objects, a global byte cap, all-catalog backlog accounting and N/2N command bounds. |
| Rclone.ResolveChild / EnsureChild | Stateful fake and private local-rclone conformance; assert commands stay below the supplied pinned parent, ambiguous observations never select a child and unconfirmed effects never become Ready. |
| DriveIndexSync.Run | Stateful folder graph with generated observation changes and injected group failures; assert failed refresh leaves snapshot bytes identical, successful complete snapshots retain unknown/history fields, and global caps apply across groups. Reuse the existing receipt-precedence sequence oracle. |
| PlanRestore / Restorer.acquireRestoreInputs | Fuzz malformed recorded locators in the pure planner; use stateful fake group mutations in acquisition, asserting exact selected IDs and pinned folder IDs, shared aggregate budgets, full actual-byte verification and no published output after any acquisition failure. |
| DecodeContinuousReport / archiveScheduler result handling | Fuzz additive/malformed folder-count reports; assert strict known counts, no fabricated whole-tree count, one warning key per observed physical destination and compatibility with legacy Inventory-only reports. |

## M1 — Layout, compatible receipts and month-aware uploads

Files: new drive_layout.go/drive_layout_test.go, drive_folder_model.go/_test.go, rclone_folders.go/_test.go, drive_receipt_json.go/_test.go, drive_months.go/_test.go; modify drive.go, drive_model.go, archivetest/rclone.go and its tests, drive_external_test.go, drive_count model/report tests. Register every new ordinary Go file in cmd/internal/artifactpath/manifest.go.

- [x] Write failing tests for the named pure functions in the strategy table below; malformed inputs are fuzzed and compatible values obey independent round-trip invariants.
- [x] Implement the shared layout and codecs; make those regressions pass. Exercise MatchReceipt without losing extension fields.
- [x] Write generated event-sequence tests for StepDriveFolder using the independent invariants below. Extend the stateful fake to model folder hierarchy and IDs while keeping existing flat fixtures meaningful.
- [x] Write failing DriveMirror.Run integration tests using retained fake state, injected effect boundaries, independent receipt/budget invariants and the counted-cost assertions above.
- [x] Implement only the minimal folder adapter/coordinator changes. Use one flat staged batch per physical folder and the existing UploadState transitions for receipts.
- [x] Add N/2N operation-count assertions; run existing Drive/receipt/staging regressions and fake self-tests. Run the full artifactpath package and repolint before M1 close.
- [x] Update atlas/history-backup.md and atlas/index.md for physical folder/receipt semantics. Run sdlc milestone-close --issue 706 --milestone M1 with measured behavior evidence; the binary owns its review.

## M2 — Discovery, restore, reporting and full qualification

Files: label_index_drive.go, label_index_view.go, restore_model.go, restore_input.go and colocated/external tests; drive_count_model.go, continuous_report.go, cmd/internal/archivecmd/continuous.go, cmd/internal/couchcmd/archive_schedule.go and warning/protocol tests; README.md and atlas/history-backup.md; extend restore_live_external_test.go and local-rclone folder conformance.

- [x] Write failing DriveIndexSync.Run tests through the stateful hierarchy using the atomicity and compatibility strategy below.
- [x] Implement bounded two-level discovery, per-physical-folder refreshes and atomic single-snapshot publication without a new index-evidence mechanism. Re-run all #689 receipt precedence tests.
- [x] Write failing PlanRestore and Restorer.acquireRestoreInputs tests using the location-validation and publication invariants below.
- [x] Wire recorded locations into restore planning/acquisition; leave hashing, decoder isolation, member checks and exclusive publication in their existing owners.
- [x] Extend report validation and Couch warning consumption for observed physical folders, retaining the old Inventory field for older reports. Never label a subset count as a total tree count. Update warning guidance now that month layout exists.
- [x] Run historyarchive/fake/archivecmd/couch warning tests, targeted race checks, artifactpath/repolint and meaningful RED/GREEN controls for new behavior.
- [x] Qualify installed rclone folder resolution/creation with a private local backend; extend the throwaway-key end-to-end round trip to recover one legacy and one month pair after deleting local catalog/receipts/index. No real credentials or operator sessions in automated tests.
- [ ] Ask hub to arrange an operator real Drive pass: a fresh month upload, retry without re-copy, legacy restore, month restore, folder IDs and checksum path/counts; normal GPG pinentry, scratch deletion confirmed. No worker-owned real-remote mutation or :0 stage without hub direction.
- [x] Update README/atlas with new layout, dual-layout recovery, observed-folder counts, bounds and no automatic migration. Preserve human review markers unless explicitly actionable.
- [ ] Push candidate, run the recorded full suite, then M2 close and whole close through SDLC reviews. On SHIP run sdlc merge --yes, verify Git/GitHub/tracker landing and report to hub. TL owns the project file.

## Planning evidence

- #706 was unowned, claimed by pair:6, then started on its own main-based branch.
- Bare start-plan caught up/rebuilt the sibling Ariadne as required, but weave refresh changed two unrelated tracked workflows. Preserved only those generated workflow changes in stash cf709527564ea17b03504c0fb767c56a76d70504; neither workflow enters this issue or receives a local-gates label. The second start-plan succeeded. Stash stays parked for separate toolchain/CI ownership, not silently restored onto this feature.
- Current code stores receipts under one configured-root namespace, but hardcodes that root as both receipt location and restore remote acquisition. That is why a write-only folder change would not fulfill #706.
- No production code changed during planning. No estimate before the plan-quality gate. Await hub approval before change-code.

## Revisions

### 2026-10-19 — PQ-1: function-specific adversarial verification

The first plan-quality review accepted the architecture and scope but required executable test contracts. Named the risky layout, folder-state, receipt, upload, index, restore and reporting functions; replaced prose case inventories with fuzz, generated event-sequence, atomicity and independent invariant strategies. Kept counted N/2N and global resource guards. No scope, mechanism or operating-envelope change.

### 2026-10-20 — M1 BR-1/BR-2: parent capacity and boundary documentation

The intended pre-effect parent bound must span the whole pass, including an uncertain create. Retain a projected capacity slot after any admitted missing-child attempt, replacing it with the confirmed ID when available. A reserved empty ID never proves existence; later attempts may not reuse it as Ready. Apply the same rule to root/year and year/month attempts. This repairs the existing parent-bound implementation without a new persisted state or evidence mechanism. Regression sequences cover two missing years and two missing months at 99999 entries, including lost confirmation.

Upload-layout and per-folder batch documentation belongs at M1 in both README and atlas; M2 adds only its delivered discovery, restore and reporting behavior. Existing human review markers remain pending.


### 2026-10-20 — Operator restore regression: own exact sizes, coarse transport backstop

The operator's root-only and mixed restore failures exposed rclone v1.75.2 rounding exact byte-valued SizeSuffix flags through three-decimal KiB values. A valid 3109-byte selected object was excluded by max-size while copy exited successfully; streaming also proved an exact max-transfer could abort the same valid object. TL approval db57f65a-d54f-4bbf-8016-f3d07bfc5ef3 supersedes the exact native cap assumption: preserve exact selected names, refuse each listed ciphertext size differing from its catalog label before copy, verify downloaded size/SHA-256 before any decoder, omit max-size, and give max-transfer only a coarse backstop (next power of two strictly above the admitted group total, minimum 1 MiB). There is no instantaneous request-wide ceiling or split-request mechanism. The adapter additionally checks staged per-file/batch limits; caller cleanup removes failed stages. Private installed streaming tests cover odd sizes above 1 KiB, exact power-of-two boundaries, root/month/mixed acquisition, and listed-size refusal with zero copies. Contextual errors preserve the physical source, staged path and wrapped cause.
