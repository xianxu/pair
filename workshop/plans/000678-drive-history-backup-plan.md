---
collaboration-mode: discussion
---

# Verified Drive copies of history backups

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Execute with superpowers-executing-plans; use bounded subagents only where Section 3 applies. Checkboxes below are tasks within one atomic close boundary, not separate milestones.

**Goal:** Copy every catalogued encrypted bundle and public JSON label to the configured Drive destination, verify both objects, retain local backups, and make unavailable remote backup visible without failing local backup.

**Architecture:** Finish the existing local archive first, then run a serial optional replication phase from a bounded in-memory projection of its catalog. A thin rclone adapter uploads only verified ciphertext/labels; a pure transition model permits a durable upload receipt only after both remote checksums match. Destination-specific upload receipts form a separate upload catalog, so remote evidence cannot masquerade as local backup evidence.

**Tech Stack:** Go, existing historyarchive Catalog/LocalSink, exec.CommandContext, installed rclone, Google Drive, stateful fake command runner.

## Requirements and alternatives

The existing local CatalogRecord remains unchanged. Its full Manifest contains private source paths; it is never uploaded. The issue's public .json companion is the existing BundleLabel, not the private manifest. Configure optional string fields `drive_remote` and `drive_folder` in continuous.json; omission defaults to `pair-backup` and `backups/pair-history`. Invalid remote settings produce Drive-unavailable status after local work, rather than invalidating otherwise valid local configuration. No rclone configuration, credentials, or private GPG keys are read by Pair.

Do not replace LocalSink: that would make Drive availability a prerequisite for local backup. Do not add destination receipts to the existing JSONL catalog: its identity and Find currently key only by bundle ID, and its continuous index treats every record as local evidence. Changing that contract would unnecessarily expand this item. Instead persist small, immutable, versioned upload receipts keyed by destination and bundle ID under `<catalog>.uploads/`. This is explicitly a new upload catalog, not an undocumented mutation of the existing schema. Its records contain destination, bundle identity, both flat remote object names, local SHA256s, remote verification methods/hashes, sizes, pinned folder ID and verification time; no full private Manifest.

Use existing LocalSink safe open/publication/Verify machinery for the receipt catalog. Each receipt is one content-validated JSON object at `<destination-SHA256>/<bundle-ID>.json`; an exact existing receipt is idempotent, conflicting/corrupt evidence refuses that remote operation visibly. This provides O(1) keyed lookup/publication per bundle without repeated scans of a growing JSONL receipt journal. Destination changes select a different namespace; old copies and receipts stay intact.

## Core concepts

### Pure entities

| Name | Lives in | Status |
| --- | --- | --- |
| DriveDestination | cmd/internal/historyarchive/drive_model.go | new |
| UploadReceipt | cmd/internal/historyarchive/drive_model.go | new |
| UploadState / UploadEvent / NextUpload | cmd/internal/historyarchive/drive_model.go | new |
| ContinuousConfig / ContinuousReport | cmd/internal/historyarchive/continuous_config.go; continuous.go | modified |

DriveDestination validates a named remote (no inline backend, flags, colon or controls), slash-separated relative folder, and content-addressed flat object names. It owns canonical destination and receipt namespace derivation. UploadReceipt is one receipt per bundle per destination; ValidateUploadReceipt requires explicit version, identity, hashes, sizes, keys and verification timestamp. MatchReceipt compares exact expected object identities rather than trusting a filename. These are pure functions with direct unit/property tests. Natural extension is another destination using the same receipt contract, not a new framework now. ARCH-DRY reuses existing BundleLabel validation. RemoteObjectNames derives `<bundle-ID>.tar.zst.gpg` and `<bundle-ID>.json` from validated ID; local record keys stay unchanged. The remote folder has no kind/year/month subdirectories.

NextUpload is the sole owner of legal operation decisions. States are InventoryBefore, CheckingBundle, CopyingBundle, CheckingLabel, CopyingLabel, InventoryAfter, ReadyToRecord, Verified, Pending and Refused. Events contain an exact bundle/destination identity plus observed absent/matching/conflicting/unknown status or operation outcome. Caller executes returned effects and feeds their observations back; it cannot manufacture Verified from command exit status. Terminal interruption ends this attempt, not the durable remote objects.

### Integration points

| Name | Lives in | Status | Wraps |
| --- | --- | --- | --- |
| RcloneRunner / ExecRcloneRunner | cmd/internal/historyarchive/rclone.go | new | bounded child processes |
| DriveMirror | cmd/internal/historyarchive/drive.go | new | local verified files, remote observations and upload receipt catalog |
| FakeRclone | cmd/internal/historyarchive/archivetest/rclone.go | new | stateful remote objects, injected faults and cancellation barriers |
| ContinuousPass.Run | cmd/internal/historyarchive/continuous.go | modified | optional replication after local completion |
| runContinuous | cmd/internal/archivecmd/continuous.go | modified | production adapter wiring and visible summary |

Runner accepts argv arrays and context, returns bounded structured output/exit information; no shell evaluation. FakeRclone implements exactly that boundary with retained object bytes and stable IDs (including duplicate names/folders), missing remote/tool, partial writes, immutable conflicts, checksum corruption, missing hashes, output truncation and held completions. Test flow and production use the same adapter and NextUpload. Existing local storage is portable private temp roots; no test inherits operator data or remote configuration. ARCH-MOCK conformance uses disposable installed-rclone local backend plus one operator-run real Drive check.

## Full flow and ordering

1. Acquire the existing continuous.lock for this pass. Finish local collection/archive and preserve existing local failure behavior. Write continuous.last when local full pass succeeds, before optional network work.
2. Under the existing catalog lock, project validated records for the configured local sink into compact candidates (label and local keys only, no full Manifest). Retain at most 100,000 candidates / 64 MiB, with visible Drive-unavailable status on exhaustion. Release the catalog lock before network IO. There is no catalog snapshot file or spool.
3. Resolve defaults and validate remote settings; locate rclone. Report unavailable and return local success when tool/settings/remote cannot be used. Resolve the configured destination folder once per pass, checking the matching directory entries at the configured parent and refusing multiple matches; pin its stable ID using the adapter's internally generated `--drive-root-folder-id <ID>`. Do not recursively walk ancestors or introduce depth limits. A missing configured folder/remote remains unavailable; never create credentials or repair remote structure. ResolveDestination is a one-time metadata operation, separate from the two folder-content listings.
4. Read one flat `lsjson` of the pinned folder, with hashes, into a bounded index of name to entries. Preserve duplicates in this index. Drive entries require stable object IDs. DecodeFlatInventory rejects malformed/overflowing metadata; duplicates are represented as ambiguous, not collapsed. Local-backend conformance uses native path identity instead of Drive IDs. Each listing has at most 100,000 entries / 64 MiB, and metadata deadline 60 seconds. No recursive listing, per-object directory enumeration, inventory spool or namespace fencing framework.
5. For each candidate read the bounded public label, validate it against the local CatalogRecord and hash its actual bytes. An exact durable receipt skips the object work. Missing receipt is pending; corrupt/conflicting receipt is visible refusal. Pending ciphertext/label is read through LocalSink into one private regular staging file at a time; SHA256 and size must match local authority before it can reach rclone. Compute MD5 alongside SHA256 on this same stream, for backends returning only MD5. Only encrypted payload and its public label reach the remote.
6. Use flat content-addressed names `<bundle-ID>.tar.zst.gpg` and `<bundle-ID>.json`. Before-list entries with a unique matching size/hash permit adoption. A missing name permits `rclone copyto <private-file> <pinned-remote:name> --immutable --ignore-existing --checksum --no-update-modtime --retries 1 --low-level-retries 1`; duplicate/mismatching names receive no copy and a visible report. Check backend SHA256 when returned, otherwise MD5 against the staged digest. If neither usable hash is returned, defer verification to downloaded SHA256. No delete, overwrite, dedupe, sync or move.
7. After upload effects, obtain one more flat `lsjson` of that same pinned folder. Publish a receipt only when each of its two names has exactly one entry and that entry's size and checksum match the expected local bytes. Prefer backend SHA256, otherwise backend MD5. Only if neither hash is available run `rclone hashsum SHA256 <pinned-remote:name> --download`; require its exact expected SHA256 and size from the unique final listing. Report per-bundle paths `backend-sha256`, `backend-md5`, or `downloaded-sha256` (including mixed pairs), plus aggregate fallback counts. No child stderr or credentials enter logs.
8. Keep compact candidate verification facts in memory, not a receipt spool. Publish and verify immutable UploadReceipt through LocalSink only after final unique-name/checksum checks. Duplicate or mismatching names get no receipt and are reported without automatic repair; unaffected pairs can still receive receipts. A command failure/timeout leaves uncertain objects pending; reconcile by listings on the next pass. A listing/probe failure stops this optional phase without receipts based on incomplete evidence. Cancellation cancels and joins an owned subprocess before return.
9. Report local counts separately from Drive verified/already-recorded/pending/unavailable state. Remote failure alone does not change local success exit status or marker. Later daily/manual passes retry; no in-pass retry storm. Flat folder and serialized pass ownership are the hub's operating assumption under its configured drive.file client; arbitrary concurrent external writers are outside the admitted model. Receipts are observations at verification time, not an ongoing remote lease.

### Remote uniqueness observation (PQ-1; hub-directed simplification)

ObserveFlatInventory is exactly the two nonrecursive folder-content listings described above. Index `name -> []entry`, never `name -> last entry`, and require multiplicity one before adoption or receipt. ResolveDestination identifies the configured folder once and pins its ID; ambiguous destination names refuse. The pass holds continuous.lock through this sequence, and retries re-list the actual objects. This directly covers duplicates created by our interrupted or uncertain uploads without general recursive namespace machinery. FakeRclone retains multiple entries with the same name, stable IDs, and controlled mutation between the two listings.

### State decisions (ARCH-ORDER)

| State and event | Next state / effect |
| --- | --- |
| CheckingBundle or CheckingLabel + unique matching before-entry | next object, then InventoryAfter |
| CheckingBundle or CheckingLabel + confirmed absent | corresponding Copying state / immutable copy |
| Copying + successful request | next object, then InventoryAfter / final flat listing |
| Any check + duplicate/mismatching bytes | Refused for this pair / report; no copy or receipt |
| Inventory parsing + malformed authority | Pending / stop optional phase |
| Any operation + unavailable or unknown outcome | Pending / report; no receipt |
| Upload/adoption effects complete | InventoryAfter / one final flat listing |
| InventoryAfter + exactly one entry per name, size/hash match | ReadyToRecord / publish receipt |
| ReadyToRecord + durable exact receipt | Verified / count |
| Receipt publication + unknown/error | Pending / reconcile next pass |
| Any active state + cancel/deadline | Pending / cancel and join child, clean private staging |
| New pass + receipt absent after partial remote write | CheckingBundle / observe and adopt exact bytes |

Invariant: receipt publication requires two matching remote object checksums after the final flat listing for this destination and bundle. An old or late result for another identity is rejected. Serial effects have no background worker surviving return. A process death may leave one remote object or both objects without a receipt; the next pass observes them before choosing effects. The existing continuous.lock excludes two active continuous owners; LocalSink immutable publication protects receipt races. Out-of-band remote writers remain external authority: no destructive repair is attempted.

### Actions and visible states

Actions: run a continuous pass; configure/fix rclone or remote settings; inspect a reported conflict; rerun after interruption.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
| --- | --- | --- | --- |
| Local completed, tool/remote/settings unavailable | Local backed up; Drive unavailable | configure/fix remote, then run pass | off-host protection delayed |
| Pending objects, timeout or child failure | Local backed up; Drive pending, verified count retained | rerun after interruption | remote partial bytes may remain; no local loss |
| Hash/receipt/identity conflict or malformed output | Local backed up; Drive verification refused | inspect reported conflict | remote copy untrusted; no local loss |
| Both remote objects verified and receipt durable | Local backed up; Drive verified | run next scheduled pass | later out-of-band remote deletion/corruption is outside this receipt's observation |
| Local archive failure | existing local failure plus any existing evidence | run a pass after resolving local failure | unchanged local backup gap |

Covered: backlog with no new local sources, partial bundle/label writes, lost acknowledgment, corruption, missing tool/remote, changing destination, cancellation and repeat invocation. Not covered: remote retention/deletion, ongoing remote integrity audits, automatic conflict repair, credential setup or changing Couch scheduling. Receipts attest successful verification at their timestamp, not eternal remote existence.

## Operating envelope and lifecycle

ARCH-CONSTRAINTS: this is optional daily batch work, never a presenter/UI operation. The measured backlog in the issue is about 758 MB; actual bundle count is not yet measured. One subprocess at a time; one ciphertext stage at a time bounded by existing MaxCiphertextBytes (2 GiB), labels/records by MaxMetadataBytes (4 MiB). Set a 30-minute remote-phase budget and 60-second metadata-command cap as initial design choices; transfer/checksum uses remaining phase time. Deadline leaves a visible pending backlog. There is no claim that 758 MB must finish within a particular network duration. Local success is already durable before the budget is spent. CPU hashing is O(bytes); network normally upload bytes plus metadata, with checksum download fallback adding one readback. Completed receipts avoid subsequent ciphertext reads and object-specific remote commands; the two flat folder listings remain phase-wide.

Catalog projection is O(catalog bytes), retaining compact candidates only, at most 100,000 / 64 MiB. Each flat remote index has the same entry/byte bounds; process the initial index, release it, then decode the final index. No catalog/inventory spooling is introduced. Source catalog is read a constant number of times per pass; keyed receipts and remote effects are O(bundle count). Count N=8 and 2N=16: two flat content listings, O(N) entries/visits, at most two copies per previously absent bundle, zero copies on rerun. Time measurements supplement counts.

ARCH-FUNERAL: upload receipts are one bounded metadata file per bundle per configured destination; their last consumer is retry/dedup until a future explicit retention policy. Retention is deliberately manual with local/remote copies kept, as required; cost is receipt bytes and one filesystem entry per bundle/destination, reported in tests and live evidence. Cap receipt serialization to 8 KiB; refuse new receipt publication visibly if its managed upload catalog exceeds 100,000 receipt entries per destination. Count existing destination entries once at phase admission and update that count on successful publication; never enumerate the receipt directory per bundle. Enforce writer and reader bounds, never silently prune verification evidence. Encrypted object staging files are pass-owned private files; clean synchronously on success/failure/cancel, sweep only this family's stale files after acquiring continuous.lock next pass. Remote objects and existing local bundles have the existing retain-until-explicit-retention policy; no automatic deletion is introduced. Register both artifact families and every new production source with artifactpath inventory/retention contracts.

ARCH-SECURE: remote names, folders, persisted receipts, local public labels and external JSON are untrusted. Strict typed decoders require fields/version and reject truncation, null, wrong identity, overflow and unsafe keys; receipt unknown versions refuse with advice. External lsjson may have documented extra backend fields but cannot omit required observation fields or contain duplicate required JSON keys. Subprocess argv contains only executable, public remote/key paths and private encrypted staging paths. Capture metadata listings at 64 MiB, other stdout at 64 KiB and stderr at 16 KiB, classify exit status without echoing arbitrary child stderr or config. Pair creates/reads no credentials; rclone alone reads operator-managed credentials. Tests isolate HOME/config and local backend. No new secret is generated.

ARCH-PURE separates typed decisions/parsers from IO; ARCH-DRY reuses LocalSink and Catalog streaming rather than cloning safe filesystem code; ARCH-PURPOSE includes the entire historical backlog and live operator proof, not just new bundles. ARCH-MOCK, ARCH-ORDER, ARCH-CONSTRAINTS, ARCH-SECURE and ARCH-FUNERAL are exercised above and by the tests below.

## Implementation tasks (one close boundary)

### Task 1: Pure contracts and configuration

Files: create drive_model.go and drive_model_test.go under cmd/internal/historyarchive; modify continuous_config.go and continuous_config_test.go.

- [x] Write failing unit tests for Destination parsing/defaults and receipt validation/identity; config with absent remote fields preserves existing local config, invalid remote settings remain remote-only failure.
- [x] Run `go test ./cmd/internal/historyarchive -run 'TestDriveDestination|TestUploadReceipt|TestContinuousConfig' -count=1`; retain meaningful RED.
- [x] Implement typed contracts, strict decoding and pure NextUpload transitions; add sequence tests that vary duplicate, cancellation, missing and uncertain observations.
- [x] Run that command GREEN and commit with issue reference and session Agent trailer.

### Task 2: Stateful rclone boundary and conformance

Files: create historyarchive/rclone.go, rclone_test.go and archivetest/rclone.go.

- [x] Write TestRcloneObservation and TestRcloneCopyRecovery against the stateful runner; use the adversarial strategy/guard table below.
- [x] Run `go test ./cmd/internal/historyarchive/... -run 'TestRclone|TestFakeRclone' -count=1` RED; implement minimum adapter and stateful fake, then GREEN.
- [x] Add an opt-in installed-rclone local-backend conformance test using isolated config/private roots; run it once against installed rclone, retaining version and command results. It never contacts Drive.
- [x] Commit the independently verified boundary.

### Task 3: Safe mirror and continuous integration

Files: create historyarchive/drive.go, drive_test.go; modify historyarchive/continuous.go and continuous_external_test.go; archivecmd/continuous.go and its tests; artifactpath/manifest.go plus applicable artifact-family declarations/tests.

- [x] Write TestDriveMirrorRecovery and TestContinuousDriveIsolation using the DriveMirror.Run strategy and independent guards below.
- [x] Run `go test ./cmd/internal/historyarchive/... ./cmd/internal/archivecmd -run 'TestDrive|TestContinuous' -count=1` RED; implement bounded catalog projection, flat listings, verified staging, receipt publication and optional post-local phase; rerun GREEN.
- [x] Add counted 8/16 bundle cycle tests and second-pass zero-copy assertions using actual production catalog and fake retained remote state.
- [x] Run targeted packages with race detector where appropriate, and `go test ./cmd/internal/artifactpath ./cmd/internal/testpair ./internal/repolint -count=1` (verify package paths before execution); commit.

### Task 4: Docs, operator conformance and normal landing

Files: README.md; atlas/history-backup.md (already indexed); issue Log and this plan.

- [x] Document config defaults/validation, independent local/Drive status, receipt location/lifetime, checksum fallback and pending/conflict recovery. Include an exact operator live command and prerequisites without secrets.
- [ ] After fake-backed qualification, ask hub to arrange the one operator-run real Drive pass. It uses production continuous configuration, uploads historical encrypted pairs, verifies checksums and receipt counts, then reruns to prove zero new copies and unchanged local files. Operator checks visible Drive bundle/label names. Retain counts/bytes, adapter version, command exit/status and receipt evidence; never include credentials. Do not stage :0, restart or relaunch without hub coordination.
- [ ] Push with sdlc pr, run recorded `sdlc local-check run --name suite`, close with verified evidence and binary-owned fresh review. Actuals measured by sdlc, no typed estimate before plan-quality gate.
- [ ] Push final Log updates before `sdlc merge --yes`. Let landing turn integrate main and validate; report PR/landed hash from returned Git output to hub.

## Adversarial strategy and mechanical guards

| Risky function | Adversarial strategy | Independent mechanical guard |
| --- | --- | --- |
| ParseDriveDestination / RemoteObjectNames | generate Unicode/control/colon/absolute/dot-segment spellings and long segments | parsed destination roundtrip plus all keys remain beneath exact configured prefix; argv never changes option count |
| DecodeUploadReceipt / MatchReceipt | mutate every required field, truncate at every boundary, duplicate keys, null/overflow/version skew | no mutation authorizes Verified for an independently constructed expected pair; bounded serialization/read |
| DecodeFlatInventory / ResolveDestination / VerifyRemoteEntry | generated unordered entries with varied multiplicities/IDs and malformed or overflowing fields | no malformed observation authorizes copy-as-absent or receipt; multiplicity one, size and independently computed checksum must match; two flat listings at N/2N |
| NextUpload | generated sequences of observations/outcomes, cancellations and stale identity results | append receipt implies two matching remote object observations; uncertain outcomes never imply absence; transition/effect enum assertions |
| DriveMirror.Run | barrier-controlled child completion and receipt/staging IO faults at each publication boundary | retain and compare local files; no returned live child; no receipt before checksum success; rerun converges from actual fake bytes |
| PublishUploadReceipt / StageVerifiedObject / SweepUploadStaging | symlink replacements, corrupt complete receipts, interrupted publication and stale stage | LocalSink safe-open/no-overwrite contract; ordinary cleanup plus next-owner sweep; destination/bundle mismatch refuses |

## Approval checkpoint

Full flow approved by hub f7e7fe62-5f41-4133-ba43-4296e82fd885; simplified flat-folder shape directed by hub 892e9607-9074-46ed-9d53-a2e5de49637d. Rerun change-code with these PQ-1/PQ-2 corrections. No remote call has been made during planning.

## Revisions

- 2026-10-18: hub approved the full design, receipt family and bounds (delivery f7e7fe62-5f41-4133-ba43-4296e82fd885). Added its requested per-bundle verification-path reporting and fallback counts. Hub will arrange the real Drive pass when fake-backed implementation is ready.

- 2026-10-18 plan-quality PQ-1/PQ-2: specified bounded before/after namespace inventories, pinned Drive folder identity and deferred receipt publication, with checksum observations fenced to exact IDs. Compressed duplicated prose case inventories into named function strategies. This identity mechanism delta is submitted to the hub before rerunning change-code.

- 2026-10-18: hub 892e9607-9074-46ed-9d53-a2e5de49637d rejected the heavier identity machinery. Superseded it with one pinned flat destination, ID-addressed bundle filenames, two flat hash listings and multiplicity-one receipt admission. Removed all spooling, recursive inventories and depth limits. Added backend MD5 verification on the same verified source stream; downloaded SHA256 remains the fallback only when no usable backend hash is returned. This meets the existing backlog/checksum/local-authority requirements without another mechanism.

- 2026-10-18 implementation conformance: installed rclone v1.75.2 local backend replaced conflicts with --immutable alone. Added --ignore-existing to the same copy operation; installed-local conformance proves preserved bytes and final hash verification still refuses conflicting objects. No new query, state or mechanism. Corrected the repository lint package path to internal/repolint.
