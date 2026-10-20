---
collaboration-mode: discussion
---

# Find, synchronize and restore history backups

> **For agentic workers:** Follow AGENTS.md §3. Use bounded subagents for independent components; keep shared schema and integration decisions in the owning session. Implementation remains gated on hub approval and `sdlc change-code`.

**Goal:** Find archived time ranges without network access, reconstruct label/location knowledge from Drive without downloading bundles, and restore verified files into a new scratch destination before retention is introduced.

**Architecture:** Keep public labels immutable and the manifest-bearing local catalog authoritative for local archival publication. Add one atomic, local-only label/location index; discover and restore through existing safe filesystem and rclone seams. Restore authenticates ciphertext and every tar member before publishing a complete new output directory.

**Tech stack:** Go, existing Catalog/LocalSink/safe descriptor helpers, encoding/json, archive/tar, existing rclone runner/stateful fake, installed GPG/zstd through an isolated decoder seam.

## Accepted requirements and alternatives

- Read the entire #689 Spec, including its later sync-index addition. The addition supersedes the original suggestion that find may list Drive: **find never queries a remote**.
- Hub bb8ab7c9-1c77-43ae-9d13-d2e28ff82dd7 approves: labels embedded unchanged in a separate index; locations beside them; observed versus verified evidence; no ciphertext downloads during sync-index; full verification on every restore; complete refresh may mark unavailable while retaining history; failed/partial refresh changes nothing.
- Recommended index: one bounded snapshot with atomic replacement. A manifest-bearing CatalogRecord cannot be reconstructed from labels, so synthesizing one would manufacture local authority. Rewriting uploaded labels to add locations would conflict with #678's immutable objects and receipt hashes. Per-record mutable files would make an all-or-nothing refresh unnecessarily transactional.
- No retention, source deletion, automatic overwrite, import into live agent roots, remote repair, key setup, remote index upload, or unrequested background service. Retention remains blocked until this item lands and the operator's live drill passes.

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| BundleLabel with preserved extensions | cmd/internal/historyarchive/model.go; label_json.go | modified |
| LabelIndex / LabelIndexRecord / BundleLocation / LocationProof | cmd/internal/historyarchive/label_index_model.go | new |
| DestinationObservation | cmd/internal/historyarchive/label_index_model.go; label_index_json.go | new |
| FindQuery / SelectLabels | cmd/internal/historyarchive/find_model.go | new |
| IndexRefresh / ApplyIndexRefresh | cmd/internal/historyarchive/label_index_model.go | new |
| RestoreRequest / RestorePlan / RestoreState / NextRestore | cmd/internal/historyarchive/restore_model.go | new |
| BundleKeys | cmd/internal/historyarchive/plan.go | new shared derivation |
| StagingOwner | cmd/internal/historyarchive/staging_model.go | new |

- **BundleLabel:** Retain known field meaning, version 1, legacy bytes when no extensions exist, and ciphertext SHA-256 separately from manifest-derived ID. Preserve unknown JSON members using RawMessage extensions; no extension grants location or verification authority. Extract the existing kind/from/to/ID filename derivation into BundleKeys and reuse it from Manifest.Keys and local-label discovery.
- **LabelIndex:** Versioned root with records keyed by full bundle ID; each record embeds the label and locations keyed by validated destination identity (backend/locator, with pinned folder ID in Drive evidence). A complete refresh of one logical Drive locator also marks its prior folder identity unavailable when that binding has changed. One label may have many locations; incompatible known label fields for one ID are conflicts, never last-writer-wins. Unknown extension conflicts are refused rather than silently discarded. Existing label values and all unknown index/record/location/proof members survive refreshes.
- **BundleLocation:** Destination identity, object keys, current availability, last observation, and historical verification facts. Two evidence levels: observed from unique name plus expected size; verified only from a matching backend SHA-256 or a validated existing #678 upload receipt. Keep first/last observation and latest verification timestamps, not an unbounded event journal. An unavailable location may retain historical verification; display that as history, not a current success.
- **FindQuery:** Optional exact kind and UTC time bounds. Accept RFC3339 UTC and YYYY-MM-DD (UTC midnight); require since < until when both exist. Overlap uses label.To >= since and label.From < until. Sort by kind/from/ID; output JSON Lines labels with their locations/evidence, without private manifests/source filenames.
- **IndexRefresh:** One complete observation set for a destination. ApplyIndexRefresh clones input, merges known label identity and extension values, replaces only that destination's current observations, retains other destinations and historical verification, and marks missing members unavailable. Incomplete/failed refreshes have no publish effect.
- **RestorePlan:** Full 64-hex IDs, deduplicated request, explicit absolute fresh target, auto/local/drive selection, optional repeated literal `--path REL` exact-file filters. Index evidence selects a candidate only; it never skips verification. Default auto tries the configured local copy, then configured Drive only if local is absent. A corrupt local copy is an explicit refusal; `--from drive` selects Drive intentionally. Unknown IDs advise find/sync-index first. Only configured destinations can be contacted; index extensions cannot introduce remote connections.
- **StagingOwner:** A bounded namespace/version/nonce/parent-identity stamp is durably created before any payload enters a fresh exclusively created staging directory. Cleanup recognizes the exact namespace and parent identity and traverses only known children without following links. An existing unrecognized directory (including a crash before a durable stamp) is preserved and refused for inspection; it is never inferred to be ours by its filename. Unknown owner members may be carried without reinterpretation; a future cleanup-protection semantic requires an incompatible owner version that this reader refuses. Shared OwnedStaging keeps index/restore ownership checks consistent.
- **RestoreState:** Verified-input, extracting, validated, published, refused/cancelled, publication-uncertain states. All transitions use NextRestore; only a validated complete output and successful decoder exits permit publication. Cancellation before publication produces cleanup; uncertainty after publication requires inspection/retry, never a false claim that output is absent.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| LabelIndexStore | cmd/internal/historyarchive/label_index_store.go | new | safe local snapshot IO, flock, fsync/rename |
| LocalLabelInventory | cmd/internal/historyarchive/label_index_local.go; catalog.go | new / modified | read-only catalog/label/receipt projection |
| DriveIndexSync | cmd/internal/historyarchive/label_index_drive.go | new | existing pinned flat rclone inventory and exact label batch |
| Rclone download operations | cmd/internal/historyarchive/rclone.go | modified | remote-to-private-directory batch copy |
| FakeRclone | cmd/internal/historyarchive/archivetest/rclone.go | modified | stateful remote objects and download outcomes |
| Decoder / CommandDecoder | cmd/internal/historyarchive/decode.go | new | GPG-agent decryption and zstd stream |
| Restorer / ExtractBundle | cmd/internal/historyarchive/restore.go; restore_extract.go | new | verified private ciphertext and bounded safe tar extraction |
| OwnedStaging | cmd/internal/historyarchive/owned_staging.go | new | private-directory owner stamp, bounded no-follow cleanup |
| Restore publication | cmd/internal/historyarchive/restore_publish_{darwin,linux}.go | new | descriptor-relative exclusive directory rename |
| Archive CLI | cmd/internal/archivecmd/find.go; sync_index.go; restore.go; run.go | new / modified | argument parsing, config, IO/reporting |

- Reuse ParseDriveDestination, ResolveDestination, ObserveFlatInventory, flat-name/identity validation, LocalSink.Get/Verify, fileDigest, safe descriptor walkers and commitoutcome. Do not duplicate config/store resolution or introduce a Google API client.
- LocalLabelInventory projects public labels and location facts, never retains every private Manifest. Add an existing-only read path to the catalog scanner: no creation, tail repair, or metadata writes during find. A missing catalog is not an instruction to create one. Local sync can also rebuild from the sink's known kind/year/month label layout using bounded no-follow traversal; no arbitrary recursive walk outside the configured sink.
- `find` reads local snapshot plus existing local catalog/labels/upload receipts; it performs no remote call and writes no index. This exposes already-created #678 backups without an initial migration. Missing local metadata is visible; sync-index is the explicit rebuild command.
- `sync-index` refreshes local evidence; `sync-index --from drive` reconstructs Drive evidence from labels alone. It does not append records to the private catalog or create upload receipts. Existing receipt proofs are checked against label, configured destination, folder identity and receipt validation rules.
- Extend the existing fake's copy seam to remote-to-local. It persists the same object multiplicity/data across listing, partial batch, retry and cancellation; counts actual subprocess requests and label/ciphertext bytes. It must reject unexpected operations instead of returning canned success.

## Formats, authority and compatibility (ARCH-EVOLVE / ARCH-SECURE)

- `<catalog>.index.json` is the sole new durable index snapshot. Store no keys, passphrases or private manifest/source-file content there. It can be rebuilt and is never local backup authority or a deletion receipt.
- BundleLabel, index root, index record, location and proof types capture unknown members with `map[string]json.RawMessage`. Custom decoders require the known fields/types and reject duplicate keys, including case aliases, trailing JSON, null where a record is required, invalid identities and unsupported interpretation versions. Unknown values survive decode/merge/re-encode at every nested level; JSON formatting need not be byte-identical. Keep the original uploaded label files unchanged.
- Reuse decodeJSONNoDuplicateKeys rather than weakening decodeStrict globally. Sweep label readers in inspectBundle, readDriveLabel, catalog decoding and new sync/find/restore routes. Keep request/config and existing manifest interpretation contracts separate; restore uses the existing validated manifest identity algorithm, not ciphertext SHA-256 as a bundle ID.
- Tests pin old label serialization and legacy archive IDs, then decode and rewrite newer label/index fixtures with nested object/array extensions. Validate missing/wrong-type known fields and duplicate aliases still fail. Unknown extension members that resemble locations/proofs do not confer evidence.
- Incoming labels with one ID but different SHA-256/size/summary are conflicting copies. Refuse the refresh and retain the old index; do not guess which ciphertext an ID should select. Future destinations remain preserved opaque extensions until explicitly supported.

## Transaction ordering and state

### Index synchronization

One fail-fast exclusive `<catalog>.index.lock` is held for the entire sync. It serializes snapshot mutation and owns the one staging family; find and backup creation do not acquire it. Catalog access is short and closes before network work, with no inverse lock order. The network phase never holds the local catalog's lock.

1. Open the index/store safely, take index.lock, recover only ownership-stamped interrupted staging, then read/validate the old snapshot.
2. Gather a complete local or pinned-Drive observation. Drive uses a before flat hash listing, a single exact `.json` download batch into private staging, and an after listing. Validate unique names, stable relevant IDs/metadata, raw label hashes when supplied, ID/filename correspondence and complete batch membership. No bundle/hash-download fallback is allowed in sync-index.
3. Name/size presence supports observed. Matching backend bundle SHA-256 or an existing validated upload receipt supports verified. MD5 can validate downloaded label bytes; it cannot promote the bundle's SHA-256 evidence. Missing hashes do not cause a hidden ciphertext download.
4. Any malformed label, conflict, missing requested file, duplicate, timeout/cancel or partial download aborts the observation and leaves the index unchanged. Successful complete observations alone reach ApplyIndexRefresh.
5. Encode/validate the entire replacement under reader/writer bounds, write private temp, fsync it, atomically replace the snapshot and fsync its parent. Before-commit failure leaves old bytes. After-rename durability failure is explicitly publication-uncertain: the visible snapshot is wholly old or wholly new, never a partial index. Reopen/compare the intended snapshot on retry; preserve uncertainty until durability is confirmed.
6. Clean owned label/filter/temp staging on every return; the next lock owner sweeps crash residue. No remote repair or deletion.

### Restore

1. Parse all requested IDs and load their validated local label view. Admit the whole batch against ciphertext/raw limits before downloading. Refuse existing target, unsafe parent, overlap with archive storage or known manifest source roots. No output path comes from a source root in the archive.
2. Hold one no-follow stable `.pair-restore.lock` in the selected target parent for the operation. This serializes owned staging/cleanup and publication within that parent. Use one private sibling `.pair-restore-staging` directory with a durable StagingOwner stamp before writing payloads; collect only a recognized owned family under the same lock on next attempt. Refuse preexisting unrecognized contents unchanged, and test an operator directory bearing that name. The lock file remains with the user-owned scratch parent; it is not a background resource.
3. Copy local ciphertext or fetch selected remote flat names into owned private staging. Drive resolution/listing is once per batch, with exact requested names and no folder recursion. Hash/size-check the staged bytes against labels before feeding them to Decoder. Never hash a mutable source and reopen it for decryption.
4. CommandDecoder permits normal gpg-agent/pinentry, uses no passphrase argument or private-key setup, disables automatic key retrieval, and streams GPG to zstd. The existing encryption helper's pinentry-mode=error is not reused unchanged. Close pipes, cancel and join both children on every failure. A controlled Decoder fake can emit complete plaintext then return a late integrity error.
5. Extract into private output staging as `<ID>/MANIFEST.json` and `<ID>/files/<relative path>`. Manifest is first, bounded and validated; compare Manifest.ID and every label summary field. Recheck overlap against each decrypted Manifest.Source.Root before extracting its first selected file, because Drive-only admission has no private manifest. Refuse a scratch parent/target inside a source root, and test this after catalog loss. Validate the complete declared member set, size/hash, effective paths and regular-file types. Reject duplicates, unexpected/missing members, links, devices, unsafe paths, excessive hidden tar metadata and non-padding trailing bytes. Verify filtered-out entries too. Count files plus newly created directories against a 100,000-entry whole-request extraction limit, including hidden staging/output members; refuse before creating an entry beyond the limit. Output files are 0600/directories 0700; never honor archived owner, executable bits or link targets.
6. Bound the entire decompressed stream, including tar/PAX headers and padding; file-size accounting alone is insufficient. Consume to validated EOF and require both decoder children to exit successfully before NextRestore permits publication.
7. Fsync files/directories and publish the whole batch with an exclusive directory rename: Darwin RenameatxNp(RENAME_EXCL), Linux Renameat2(RENAME_NOREPLACE). A target created concurrently is refused, never replaced; unsupported exclusive-rename behavior is refused rather than weakened to check-then-rename. Parent-sync uncertainty reports complete-output-may-exist and requires inspection. No partial plaintext becomes the successful target; all owned uncommitted plaintext/ciphertext is removed on error/cancel and next-owner recovery.
8. Restore does not rewrite remote labels, private catalog or index verification evidence. Report restored files/bundles and target only after publication is confirmed. Restored output persists until its owner removes that explicit scratch tree.

Actions: find, sync-index, restore to a fresh path, retry after fixing the reported conflict/configuration, inspect uncertain committed output.

| Internal outcome | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| No index, local catalog exists | local labels available | find / local sync | no archive loss |
| No local metadata, Drive available | index needs rebuild | sync-index --from drive | no archive loss |
| Complete presence, no usable bundle SHA-256 | observed location | restore for full byte verification | no archive loss |
| Missing on complete refresh | unavailable now; historical verification retained | choose another location | no automatic deletion |
| Partial/failed/cancelled sync | old index retained; refresh failed | retry sync-index | discarded staging only |
| Index commit durability uncertain | complete snapshot may have committed | reopen/retry synchronization | no partial index |
| Bad key/ciphertext/tar/filter or cancelled restore | refused; no target published | correct input/key or choose another copy | discarded owned staging only |
| Unrecognized preexisting staging | preserved, ownership unclear | inspect it or select a different parent | none |
| Existing/concurrent output target | target untouched | choose a fresh path | none |
| Output commit durability uncertain | complete target may exist | inspect target before retry | no automatic overwrite |

## Operating envelope, counts and lifecycle

This is explicit batch CLI work, not a UI/keystroke hot path. Reuse the current 30-minute remote-phase and 60-second metadata-query budgets; cap the complete restore at 30 minutes including key unlock. No extra detached workers or polling loops.

- Index: at most 100,000 records, 16 locations per record, 4 MiB per label/record, 64 MiB serialized snapshot and 64 MiB aggregate downloaded labels. Enforce the same limits before writing as on reading; overflow refuses the refresh, keeps the old snapshot and reports the bound. One snapshot supersedes another, so no generation directories or index journal accumulate.
- Remote inventories retain #678's 100,000-entry/64-MiB wire limits. Before sync download, sum label sizes and require complete pair-name/size observations. Configure the batch's actual transfer cap (`--max-transfer`, hard cutoff, per-object `--max-size`) and reject missing/partial outputs; do not merely discover an overflow after writing arbitrary bytes to disk. Confirm these installed-rclone semantics in local conformance.
- Sync cost: one folder resolution, two flat listings and at most one exact label-copy batch (4 rclone invocations independent of N); zero ciphertext downloads and zero downloaded checksum batches. N=8 and 16 tests count invocations, downloaded bytes and visited labels. Each pass scans each index/catalog/inventory a constant number of times; no per-record directory/catalog rescans.
- Restore proposal: at most 64 IDs per command; existing 2 GiB ciphertext and 1 GiB raw/8 GiB snapshot limits per bundle; aggregate 4 GiB ciphertext, 16 GiB raw and 100,000 selected files admitted from labels. Extraction additionally caps all created filesystem entries at 100,000, so empty-directory/inode growth cannot bypass byte limits. Whole tar overhead is separately capped at 32 MiB per bundle (10,000 headers/padding plus bounded manifest/path extensions). Exceeding batch limits asks for smaller selections. These are explicit initial batch limits, not performance claims; the Claude-day drill is far below them.
- Restore network cost is independent of selected file count: one resolution/listing and one selected ciphertext batch, followed by a constant streaming pass over each bundle and all its members. N/2N tests count runner invocations/member bytes and prove no source-root scan or per-file rclone call.
- Both download flows use at most eight rclone transfers, bounded diagnostics inherited from #678, isolated private staging and one owned process at a time. Decoder owns two children and one bounded streaming consumer; all are joined before return. No whole plaintext tar is buffered/spooled.
- Index lock/staging: one stable lock and one bounded staging family per configured catalog; cleanup every return/next owner. Restore staging: one bounded family per explicit parent under its lifetime lock; the last plaintext consumer is confirmed publication or cleanup. Published restored output/parent lock are user-requested durable artifacts removed with the scratch parent, with cost bounded by the admitted request. No retention mechanism is introduced.
- Memory grows with bounded metadata plus one streaming buffer per active transfer; measure representative N/2N metadata counts and record timings without claiming a fixed RSS independent of Go/runtime allocation.

## Implementation and review boundaries

### M1 — compatible labels, local find and atomic index synchronization

Files: add label_json.go, label_index_model.go, label_index_store.go, label_index_local.go, label_index_drive.go, find_model.go, staging_model.go, owned_staging.go and colocated tests under historyarchive; extend model.go/plan.go/catalog.go/rclone.go and the stateful fake; add archivecmd/find.go and sync_index.go with CLI tests; update help/README/atlas and artifactpath source inventory.

- [x] Use the named-function strategies below in RED/GREEN order to implement compatible label/index codecs, SelectLabels and ApplyIndexRefresh.
- [x] Implement existing-only local projection, bounded label discovery, atomic LabelIndexStore and shared OwnedStaging through their function-level contracts below.
- [x] Extend Rclone/FakeRclone downloads and implement complete Drive label observation; qualify the fixed-cost, labels-only synchronization contract below.
- [x] Fixture: create/upload archives, delete local catalog/index/labels/bundles (keep only fake remote and unrelated source originals), sync from Drive labels alone, find identical IDs/ranges/locations, and prove no private CatalogRecord or upload receipt was invented.
- [x] Add CLI validation/help/JSON output/config resolution and README/atlas/index ownership notes. Register all production files. Run focused/race/guard tests and `sdlc milestone-close --issue 689 --milestone M1` with evidence; fix the binary-owned review findings before M2.


### M2 — verified restore, real-key round trip and operator drill

Files: add restore_model.go, restore.go, restore_extract.go, decode.go, restore_publish_darwin.go/restore_publish_linux.go and tests; extend archivetest codec fixtures and rclone downloads; add archivecmd/restore.go and tests; extend codec_live_test.go; document operator drill in atlas/history-backup.md and root README.

- [x] Implement RestorePlan/NextRestore, ExtractBundle and Decoder/Restorer in RED/GREEN order against the named-function strategies below. Keep literal path selection deterministic and publication conditional on complete validation.
- [x] Implement exclusive publication and qualify its OS contract below; compile/test the Linux path through normal supported CI. Unsupported operations remain refusals.
- [x] Reuse/refactor the existing isolated GPG key fixture, with explicit temp-root template and agent cleanup. Gated real-binary test uses production create -> fake Drive upload -> removal of all local archive metadata/copies -> sync-index -> find -> production restore; compare independent original hashes and bytes, including Unicode/binary/empty files and exact-path filtering. No operator keys/config or real remote in automated tests. Quote/sanitize untrusted paths in bounded diagnostics; never print decrypted payloads or key material.
- [x] Run installed-rclone local-backend download conformance against the same fake contract and the GPG/zstd round trip in isolated homes. Update docs/atlas/source guards, run targeted races, then push and record the full suite normally.
- [x] Send a private candidate and exact commands to the hub for the operator's one real Claude-day Drive restore into a fresh scratch child. Include explicit PAIR_DATA_DIR from a Pair terminal, find/sync-index steps, byte comparison against originals, no live-data import, and cleanup of only that owned scratch tree. Retain observed counts/hashes/verification outcome; await operator acceptance before close/landing. No :0 stage, restart or live remote execution by this worker without separate authorization.
- [ ] Run `sdlc milestone-close --issue 689 --milestone M2` with accepted live and automated evidence, then close through `sdlc close --issue 689 --verified ...` with measured actuals; its mandatory fresh review is the boundary. Push final evidence, enter `sdlc merge --yes`, let the landing turn integrate/qualify main, and report outcome to hub. TL owns the history-backup project and the retention dependency.

## Risk functions and mechanical verification

| Risk function | Adversarial strategy | Mechanical guard |
|---|---|---|
| DecodeLabel / DecodeLabelIndex and nested codecs | bounded schema mutation across known and future members | golden legacy bytes/IDs; recursive unknown-value round trip; reject invalid known fields, duplicate aliases and conflicting extensions; bounded parser fuzz |
| ApplyIndexRefresh / reconcileReceiptLocation | generated refresh and receipt histories across empty destinations, folder bindings, completeness and proof times | immutable prior snapshot; durable complete markers; retained history and unrelated destinations; no receipt outranks complete absence/binding, newer matching receipt promotes only its own location |
| SelectLabels / LocalLabelInventory / BundleKeys | generated time ranges and existing/missing local metadata trees | deterministic overlap/order and keys; find performs zero writes/network calls; counted 8/16 scans and no absent-catalog creation |
| OwnedStaging | interruption at ownership/payload boundaries with preexisting foreign contents | recognized-only no-follow cleanup under lifetime lock; unknown directories and source bytes unchanged; created-entry bound |
| LabelIndexStore.Commit | injected commit-boundary failures and contended/cancelled lifetimes | old-or-new complete snapshot; before-commit old bytes unchanged; reopen durability/uncertainty checks; one lifetime owner |
| DriveIndexSync / Rclone download | generated inconsistent inventories and interrupted/over-limit transfers | stateful fake and installed local conformance; <=4 invocations at N=8/16, 0 bundle bytes, actual transfer cap, unchanged prior snapshot on incomplete observation |
| RestorePlan / NextRestore | generated request/event sequences crossing admission, proof and publication boundaries | aggregate bounds; evidence never skips SHA-256; no publication before validated decoder success; explicit uncertainty; reject out-of-order events |
| ExtractBundle | generated malformed tar structure/content including hidden metadata and excluded members | full-stream/entry caps; exact expected-set, manifest/label and all-member hash accounting; no unsafe types/paths or non-padding tail; post-manifest source overlap refusal; guard-removal RED checks |
| CommandDecoder / Restorer | readiness-controlled cancellation and late integrity failure after valid plaintext | no publication before both successful exits; every child/owned handle collected; source/remote/target bytes unchanged; N/2N runner/member-byte counts |
| PublishRestore | preexisting and concurrently created output, moved parent, sync failure | exclusive OS rename conformance; no-replace invariant and explicit uncertain result |

## Architecture account

ARCH-DRY reuses publication/path/hash/Drive primitives and extracts filename derivation once. ARCH-PURE owns selection, evidence merge, admission and state transitions. ARCH-PURPOSE covers local-only find, catalog-loss sync, actual decrypted restore and the real operator drill before retention. ARCH-MOCK extends the same stateful rclone/codec seams with isolated real-tool conformance. ARCH-CONSTRAINTS gives metadata/byte/process/time budgets and counted 8/16 cases. ARCH-SECURE authenticates every input and never grants deletion/live-import authority. ARCH-ORDER uses one index mutator lock, one restore-parent lifetime lock, explicit transitions and old-or-new publication. ARCH-FUNERAL bounds/supersedes the index and cleans staging, while requested restored output belongs to its operator. ARCH-EVOLVE preserves every unknown label/index member across read/merge/write with legacy compatibility tests.

## Revisions

- 2026-10-18: initial full plan following dispatch 06dd38d3-b87e-451d-a896-cef079ff0929. Start-plan refreshed the stale ariadne tool build and was rerun as instructed; no source implementation started.
- 2026-10-18: hub bb8ab7c9-1c77-43ae-9d13-d2e28ff82dd7 approved separate immutable-label envelopes, observed/verified evidence and complete-only refresh replacement. Specified one atomic snapshot so the approved no-partial-index promise is structural rather than a rollback across per-record files.

- 2026-10-18 one-pass plan editing aid: made post-manifest source-root overlap checks explicit for Drive-only recovery, added durable staging ownership/refusal before cleanup, and bounded created filesystem entries in addition to byte sizes. Added preexisting-operator-directory, missing-catalog overlap and entry-overflow regressions. No implementation was written.

- 2026-10-18: full plan approved as written by hub d9acf9a3-51ac-4eda-9dc9-1a70065e04ac, including snapshot/restore caps and lifetime locks. The hub will arrange the required operator Claude-day Drive drill when M2 is ready. Proceed to change-code; automated plan quality and derived estimate remain required before implementation.

- 2026-10-18 PQ-1: consolidated repeated milestone case inventories into the named-risk-function strategy table, including selection, staging ownership and restore transitions. Behavioral contracts, resource bounds, end-to-end recovery and operator acceptance are unchanged.

- 2026-10-19 installed-rclone conformance: v1.75.2 rejects files-from-raw combined with max-size. The exact canonical-name download batch therefore uses rooted include filter rules plus the closing exclude, retaining max-size and hard max-transfer. Selected-byte, oversized-file and transfer-cutoff probes passed; upload transport is unchanged.

### 2026-10-19 — BR-1 proposed revision, awaiting hub approval

The M1 reviewer found that a complete empty Drive observation has no durable representation: a later offline projection can resurrect an older receipt whose bundle/folder identity was never indexed. Per-record availability cannot encode destination-wide absence. Add one optional, unknown-preserving `complete_destinations` map to the index root, keyed by backend/logical locator, with a `DestinationObservation` value containing the pinned folder ID and complete observation time. Bound it to 1,024 destinations under the existing whole-snapshot 64 MiB cap. Older snapshots default to no complete observations; older readers preserve the additive field unchanged.

`IndexRefresh` carries the pinned folder ID explicitly, including empty observations. ApplyIndexRefresh atomically updates the destination marker with every successful complete Drive refresh; stale refreshes refuse against that marker even when there are no records. Failed or partial refreshes update neither records nor marker. A later complete folder rebinding replaces that logical destination's current binding while preserving per-location history.

Offline receipt projection uses one precedence rule across indexed and previously unseen identities: a historical receipt cannot claim availability against a newer complete observation, or a different known complete folder binding. Preserve its dated proof as unavailable history. A receipt newer than both the complete marker and that location's last observation may establish only its own location's availability when it matches the complete binding; another complete sync is required to admit a changed folder binding. With no complete marker, existing validated receipts keep their original dated evidence behavior. No extra remote queries or per-record scans are introduced.

Verification extension: sequence complete-empty initial sync -> find -> local sync; complete folder rebind -> previously unseen old receipt projection; stale refresh against an empty marker; genuine later upload in the same binding; unknown marker fields and legacy snapshot read/rewrite; marker count/byte bounds. Run each relevant sequence through LoadLabelView and SyncLocalLabelIndex, with old snapshot unchanged on partial/failed refresh. The shared named-function strategy table now also covers destination-marker precedence within ApplyIndexRefresh and receipt composition.

- 2026-10-19 hub b322a666-7176-43fc-baa1-2bd67c49571c approved the BR-1 refinement above, including empty observations, binding/time precedence, newer same-binding receipts and sequence tests. This is the third refinement on index evidence: if another review hole appears, stop and bring it to the hub before another patch; simplify the model rather than adding another incremental mechanism.

- 2026-10-19 M2 BR-2: corrected the implementation/docs to enforce the already promised 32 MiB overhead relative to each validated label's declared raw payload, rather than adding it to the kind's worst-case raw ceiling. The regression admits the exact total and refuses one extra byte for both normal and snapshot kinds. This restores the approved bound; index evidence is unchanged.
- 2026-10-19 M2 BR-3: the review's long TMPDIR exceeded native GPG socket-path limits. Requalified the exact pinned code under an isolated short private TMPDIR (both real-tool tests passed, native agent/home/root cleanup measured), and documented that invocation. Linux remains cross-compiled only. The finding is disputed with measured host evidence for explicit reviewer disposition.
- 2026-10-19 M2 final diagnostic check: reproduced terminal control emission from a corrupt member filename and the CLI error seam. Quote the member identity and reuse rowtext.Sanitize at the CLI diagnostic owner so arbitrary child/filesystem error text is safe too; preserve the existing single-line normalization and readable Unicode. No new index evidence mechanism, source mutation or restoration mode.

- 2026-10-19 M2 resource-bound class sweep: reuse the admitted remote subset's sum and largest object as the existing DownloadBatch hard total/per-object limits. Stateful post-listing growth now refuses before payload transfer; installed-rclone local conformance proves exact-bound single/batch success and overflow cleanup. No index evidence change, additional network query or transfer mechanism.
