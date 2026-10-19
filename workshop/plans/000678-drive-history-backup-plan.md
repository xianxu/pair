---
collaboration-mode: discussion
---

# Verified Drive copies of history backups

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Execute with superpowers-executing-plans; use bounded subagents only where Section 3 applies. Checkboxes below are tasks within one atomic close boundary, not separate milestones.

**Goal:** Copy every catalogued encrypted bundle and public JSON label to the configured Drive destination, verify both objects, retain local backups, and make unavailable remote backup visible without failing local backup.

**Architecture:** Finish the existing local archive first, then run an optional batched replication phase from a bounded in-memory projection of its catalog; one copy subprocess performs at most eight parallel transfers. A thin rclone adapter uploads only verified ciphertext/labels; a pure transition model permits a durable upload receipt only after both remote checksums match. Destination-specific upload receipts form a separate upload catalog, so remote evidence cannot masquerade as local backup evidence.

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

NextUpload is the sole owner of legal operation decisions. States are InventoryBefore, PreparedForBatch, InventoryAfter, ReadyToRecord, Verified, Pending and Refused. Events contain an exact bundle/destination identity plus observed absent/matching/conflicting/unknown status or operation outcome. Caller executes returned effects and feeds their observations back; it cannot manufacture Verified from command exit status. A batch command error is not terminal for every pair: final observations may verify its completed members. Caller reports the batch outcome as an event before those observations. Cancellation ends the attempt after joining the child, leaving already published remote bytes for reconciliation.

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
5. For each candidate read the bounded public label, validate it against the local CatalogRecord and hash its actual bytes. An exact durable receipt skips the object work. Missing receipt is pending; corrupt/conflicting receipt is visible refusal. Pending ciphertext/label is read through LocalSink into one private regular staging file at a time; SHA256 and size must match local authority before it can reach rclone. Compute MD5 alongside SHA256 on this same stream, for backends returning only MD5. Only encrypted payload and its public label reach the remote. Their existing local keys include kind/year/month directories; files-from on that tree would retain those directories, so the batch needs a flat staging view. Use descriptor-checked hard links for same-filesystem immutable objects, verified copy fallback on EXDEV, and validate staged bytes/identity before the child reads them. This avoids rewriting the measured 790 MB on the normal same-filesystem path.
6. Use flat content-addressed names `<bundle-ID>.tar.zst.gpg` and `<bundle-ID>.json`. Preserve initial duplicate/mismatch refusals. Admit complete pending pairs up to a 4 GiB logical staged-byte budget (ciphertext plus labels), bounded by existing object/count limits; when the next pair does not fit, leave it and later pairs pending for the next pass. Never admit half a pair at the budget boundary. Hard links count their logical sizes toward the same cap; across filesystems, copied data is capped at 4 GiB. On staging IO/space failure, abort before the copy child and clean owned files.
7. Run at most one `rclone copy <flat-staging-folder> <pinned-remote-root> --transfers 8 --immutable --ignore-existing --checksum --no-update-modtime --retries 1 --low-level-retries 1`. The staging folder contains only admitted encrypted objects/public labels. A nonzero batch exit is retained as an error but does not bypass final observation: proceed to the second flat hash listing while context remains usable. Duplicate or mismatching names get no receipt and no automatic repair. Complete verified pairs get receipts even when another member failed.
8. After that second listing, prefer backend SHA256 then MD5 against verified local bytes. Collect only unique, size-matching names without usable hashes into one bounded private files-from-raw list, outside the upload folder, and run at most one `rclone hashsum SHA256 <pinned-remote-root> --download --files-from-raw <list> --checkers 8`. Decode its bounded multi-line result keyed by exact flat names, preserving missing/duplicate/invalid results as unverified. No per-object child processes. Report per-bundle backend-sha256/backend-md5/downloaded-sha256 methods and fallback counts. Publish immutable receipts separately only for pairs whose two final objects verify; uncertain members remain pending. Command failure/cancellation cannot manufacture verification. Cancel and join a running child before cleaning staging or returning.
9. Report local counts separately from Drive verified/already-recorded/pending/unavailable state. Remote failure alone does not change local success exit status or marker. Later daily/manual passes retry; no in-pass retry storm. Flat folder and serialized pass ownership are the hub's operating assumption under its configured drive.file client; arbitrary concurrent external writers are outside the admitted model. Receipts are observations at verification time, not an ongoing remote lease.

### Remote uniqueness observation (PQ-1; hub-directed simplification)

ObserveFlatInventory is exactly the two nonrecursive folder-content listings described above. Index `name -> []entry`, never `name -> last entry`, and require multiplicity one before adoption or receipt. ResolveDestination identifies the configured folder once and pins its ID; ambiguous destination names refuse. The pass holds continuous.lock through this sequence, and retries re-list the actual objects. This directly covers duplicates created by our interrupted or uncertain uploads without general recursive namespace machinery. FakeRclone retains multiple entries with the same name, stable IDs, and controlled mutation between the two listings.

### State decisions (ARCH-ORDER)

| State and event | Next state / effect |
| --- | --- |
| InventoryBefore + valid pair prepared under byte budget | PreparedForBatch / enqueue its two flat objects |
| InventoryBefore + duplicate/mismatch | Refused / report; exclude pair |
| InventoryBefore + batch budget exhausted | Pending / defer complete pair |
| PreparedForBatch + batch success, failure or unknown outcome | InventoryAfter / one shared final listing |
| InventoryAfter + unique matching pair checksums | ReadyToRecord / publish receipt |
| InventoryAfter + missing/uncertain member | Pending / report and reconcile next pass |
| InventoryAfter + duplicate/mismatching member | Refused / report, no receipt |
| ReadyToRecord + durable exact receipt | Verified / count |
| Receipt publication + unknown/error | Pending / reconcile next pass |
| Any active state + cancel/deadline | Pending / cancel/join child, clean owned staging |
| New pass + absent receipt after partial batch | InventoryBefore / observe retained bytes |


Invariant: receipt publication requires two matching remote object checksums after the final flat listing for this destination and bundle. An old or late result for another identity is rejected. At most one subprocess is active; its internal transfers/checkers are capped at eight, and no owned child survives return. A process death may leave one remote object or both objects without a receipt; the next pass observes them before choosing effects. The existing continuous.lock excludes two active continuous owners; LocalSink immutable publication protects receipt races. Out-of-band remote writers remain external authority: no destructive repair is attempted.

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

ARCH-CONSTRAINTS: this is optional daily batch work, never a presenter/UI operation. The measured backlog in the issue is about 758 MB; actual bundle count is not yet measured. One subprocess at a time with at most eight internal transfers/checkers. Each ciphertext keeps MaxCiphertextBytes (2 GiB), labels/records keep MaxMetadataBytes (4 MiB), and complete staged pairs total at most 4 GiB per pass. Same-filesystem hard links avoid duplicate data allocation; EXDEV fallback uses at most that same byte budget. Remaining pairs are explicitly deferred, not reported as verified. Set a 30-minute remote-phase budget and 60-second metadata-command cap as initial design choices; transfer/checksum uses remaining phase time. Deadline leaves a visible pending backlog. There is no claim that 758 MB must finish within a particular network duration. Local success is already durable before the budget is spent. CPU hashing is O(bytes); network normally upload bytes plus metadata, with checksum download fallback adding one readback. Completed receipts avoid subsequent ciphertext reads and object-specific remote commands; the two flat folder listings remain phase-wide.

Catalog projection is O(catalog bytes), retaining compact candidates only, at most 100,000 / 64 MiB. Each flat remote index has the same entry/byte bounds; process the initial index, release it, then decode the final index. No catalog/inventory spooling is introduced. Source catalog is read a constant number of times per pass; keyed receipts and remote object checks remain O(bundle count) while subprocess count is constant: one resolution, two flat content listings, at most one copy batch and at most one filtered downloaded-hash batch. Count N=8 and 2N=16 actual fake invocations, not only copied objects: at most five commands, one upload batch (zero on rerun), one fallback batch only when needed, and linear catalog/entry visits. Exercise byte-budget admission with a test-injected smaller logical budget using the same production admission function, without allocating multi-GiB fixtures. Time measurements supplement counts.

ARCH-FUNERAL: upload receipts are one bounded metadata file per bundle per configured destination; their last consumer is retry/dedup until a future explicit retention policy. Retention is deliberately manual with local/remote copies kept, as required; cost is receipt bytes and one filesystem entry per bundle/destination, reported in tests and live evidence. Cap receipt serialization to 8 KiB; refuse new receipt publication visibly if its managed upload catalog exceeds 100,000 receipt entries per destination. Count existing destination entries once at phase admission and update that count on successful publication; never enumerate the receipt directory per bundle. Enforce writer and reader bounds, never silently prune verification evidence. Encrypted object links/copies and one checksum filter list are pass-owned private files; clean synchronously on success/failure/cancel, sweep only this family's stale files after acquiring continuous.lock next pass. Remote objects and existing local bundles have the existing retain-until-explicit-retention policy; no automatic deletion is introduced. Register both artifact families and every new production source with artifactpath inventory/retention contracts.

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
| NextUpload / AdmitBatchPair | generated sequences of observations/outcomes, cancellations and stale identity results | append receipt implies two matching remote object observations; uncertain outcomes never imply absence; transition/effect enum assertions |
| DriveMirror.Run | barrier-controlled child completion and receipt/staging IO faults at batch publication and per-pair receipt boundaries | retain and compare local files; no returned live child; no receipt before checksum success; rerun converges from actual fake bytes |
| PublishUploadReceipt / PrepareBatchObject / SweepUploadStaging | symlink replacements, corrupt complete receipts, interrupted publication and stale stage | LocalSink safe-open/no-overwrite contract; ordinary cleanup plus next-owner sweep; destination/bundle mismatch refuses |

## Approval checkpoint

Full flow approved by hub f7e7fe62-5f41-4133-ba43-4296e82fd885; simplified flat-folder shape directed by hub 892e9607-9074-46ed-9d53-a2e5de49637d. Rerun change-code with these PQ-1/PQ-2 corrections. No remote call has been made during planning.

## Revisions

- 2026-10-18: hub approved the full design, receipt family and bounds (delivery f7e7fe62-5f41-4133-ba43-4296e82fd885). Added its requested per-bundle verification-path reporting and fallback counts. Hub will arrange the real Drive pass when fake-backed implementation is ready.

- 2026-10-18 plan-quality PQ-1/PQ-2: specified bounded before/after namespace inventories, pinned Drive folder identity and deferred receipt publication, with checksum observations fenced to exact IDs. Compressed duplicated prose case inventories into named function strategies. This identity mechanism delta is submitted to the hub before rerunning change-code.

- 2026-10-18: hub 892e9607-9074-46ed-9d53-a2e5de49637d rejected the heavier identity machinery. Superseded it with one pinned flat destination, ID-addressed bundle filenames, two flat hash listings and multiplicity-one receipt admission. Removed all spooling, recursive inventories and depth limits. Added backend MD5 verification on the same verified source stream; downloaded SHA256 remains the fallback only when no usable backend hash is returned. This meets the existing backlog/checksum/local-authority requirements without another mechanism.

- 2026-10-18 implementation conformance: installed rclone v1.75.2 local backend replaced conflicts with --immutable alone. Added --ignore-existing to the same copy operation; installed-local conformance proves preserved bytes and final hash verification still refuses conflicting objects. No new query, state or mechanism. Corrected the repository lint package path to internal/repolint.

- 2026-10-18 live-driven batch revision (hub b4588b23-0c2c-4250-bf98-508602961c6a; approval/refinement 89180927-8227-4f02-8f4f-a093e1a771d2): operator observed 116 serial copyto processes for 58 bundles / 790 MB. Replanned transport to one copy batch with eight transfers and one optional downloaded-hash batch, with per-pair final receipts despite a partial batch error. Flat local staging is needed to remove existing local key directories; same-filesystem hard links avoid a second 790 MB data copy, EXDEV uses verified copying. Added 4 GiB complete-pair admission and visible overflow deferral. Supersedes previous single-object/constant-per-object cost assumptions; retains separate receipt authority and two flat listings. Operator live instructions must explicitly set PAIR_DATA_DIR="$HOME/.local/share/pair" when invoked from a Pair slot. Rerun change-code before code.

- 2026-10-18: hub 99090e02-f6db-4b07-b87d-0f9677fc3286 explicitly approved the flat hard-link view, verified-copy fallback, 4 GiB logical pair cap, one copy and optional hash batch, and mandatory cleanup on every exit. Hub 64519fc8-111a-425a-8607-423a59680640 requested bounded readable stderr in pending reasons; retain terminal-cleaned tail diagnostics with credential values redacted rather than content-free exit codes. Hub cb95f294-af33-4a95-99d0-b583012e94da accepted the real Drive pass/rerun and directed finishing batch transport and landing. This acceptance establishes Drive behavior; the earlier serial pass is not batch timing evidence.

## Batch revision qualification

- [x] Integrated fake N/2N actual command counts, complete-pair budget deferral, hard-link and verified-copy paths, partial-batch receipts, and cleanup on success/failure/cancel.
- [x] Installed-local batch conformance and race/guard checks on the revised code.
- [ ] Final pushed-head recorded suite and binary-owned close review, then landing.


### 2026-10-18 close BR-1 — destination capacity

The initial per-batch entry cap did not reserve capacity against objects already in the destination. Reserve the actual before-inventory entry count once, including unrelated entries and duplicate members, and charge only absent members of admitted complete pairs. Defer pairs before copying would exceed the 100,000-entry reader bound; existing-only verification and receipt recovery remain possible at the boundary. Test near-full inventory, one missing member, full-inventory adoption and repeated passes. This corrects ARCH-CONSTRAINTS / ARCH-FUNERAL within the existing admission mechanism. The 64 MiB external wire limit remains a fail-closed metadata observation bound: backend-added fields and formatting are external, so it is not a guaranteed remote-storage capacity allowance.

Artifactpath clarification: its Families vocabulary is explicitly tag-bearing session artifacts, while the history backup uses content-addressed bundles under a configured sink. Upload receipts and pass staging therefore follow the existing historyarchive NonArtifactSources classification; their owners, bounds, consumers and retention are registered in this plan and atlas/history-backup.md. Register every new production source there, without inventing tag-bearing family tokens for backup metadata. This supersedes the earlier sentence promising both families in artifactpath itself; artifactpath package guards validate the source classification.
