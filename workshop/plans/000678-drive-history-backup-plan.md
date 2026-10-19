---
collaboration-mode: discussion
---

# Verified Drive copies of history backups

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Execute with superpowers-executing-plans; use bounded subagents only where Section 3 applies. Checkboxes below are tasks within one atomic close boundary, not separate milestones.

**Goal:** Copy every catalogued encrypted bundle and public JSON label to the configured Drive destination, verify both objects, retain local backups, and make unavailable remote backup visible without failing local backup.

**Architecture:** Finish the existing local archive first, then run a serial optional replication phase from a private snapshot of its catalog. A thin rclone adapter uploads only verified ciphertext/labels; a pure transition model permits a durable upload receipt only after both remote checksums match. Destination-specific upload receipts form a separate upload catalog, so remote evidence cannot masquerade as local backup evidence.

**Tech Stack:** Go, existing historyarchive Catalog/LocalSink, exec.CommandContext, installed rclone, Google Drive, stateful fake command runner.

## Requirements and alternatives

The existing local CatalogRecord remains unchanged. Its full Manifest contains private source paths; it is never uploaded. The issue's public .json companion is the existing BundleLabel, not the private manifest. Configure optional string fields `drive_remote` and `drive_folder` in continuous.json; omission defaults to `pair-backup` and `backups/pair-history`. Invalid remote settings produce Drive-unavailable status after local work, rather than invalidating otherwise valid local configuration. No rclone configuration, credentials, or private GPG keys are read by Pair.

Do not replace LocalSink: that would make Drive availability a prerequisite for local backup. Do not add destination receipts to the existing JSONL catalog: its identity and Find currently key only by bundle ID, and its continuous index treats every record as local evidence. Changing that contract would unnecessarily expand this item. Instead persist small, immutable, versioned upload receipts keyed by destination and bundle ID under `<catalog>.uploads/`. This is explicitly a new upload catalog, not an undocumented mutation of the existing schema. Its records contain destination, bundle identity, both object keys, SHA256s, sizes, and verification time; no full private Manifest.

Use existing LocalSink safe open/publication/Verify machinery for the receipt catalog. Each receipt is one content-validated JSON object at `<destination-SHA256>/<bundle-ID>.json`; an exact existing receipt is idempotent, conflicting/corrupt evidence refuses that remote operation visibly. This provides O(1) keyed lookup/publication per bundle without repeated scans of a growing JSONL receipt journal. Destination changes select a different namespace; old copies and receipts stay intact.

## Core concepts

### Pure entities

| Name | Lives in | Status |
| --- | --- | --- |
| DriveDestination | cmd/internal/historyarchive/drive_model.go | new |
| UploadReceipt | cmd/internal/historyarchive/drive_model.go | new |
| UploadState / UploadEvent / NextUpload | cmd/internal/historyarchive/drive_model.go | new |
| ContinuousConfig / ContinuousReport | cmd/internal/historyarchive/continuous_config.go; continuous.go | modified |

DriveDestination validates a named remote (no inline backend, flags, colon or controls), slash-separated relative folder, and existing derived object keys. It owns canonical destination and receipt namespace derivation. UploadReceipt is one receipt per bundle per destination; ValidateUploadReceipt requires explicit version, identity, hashes, sizes, keys and verification timestamp. MatchReceipt compares exact expected object identities rather than trusting a filename. These are pure functions with direct unit/property tests. Natural extension is another destination using the same receipt contract, not a new framework now. ARCH-DRY reuses existing BundleLabel validation and derived bundle keys.

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
2. Under the existing catalog lock, stream validated records into a private snapshot file. Release catalog lock before any network operation. Only records for the configured local sink are candidates. Stream the snapshot afterward; retain at most one record and one object buffer/stage at a time.
3. Resolve defaults and validate remote settings; locate rclone. Report unavailable and return local success when tool/settings/remote cannot be used. Never configure a remote or create credentials. Use one bounded destination probe, not one failing probe per bundle.
4. For each candidate, read the bounded local public label, validate it against the local CatalogRecord, and hash its actual bytes. A matching durable upload receipt skips further work. Receipt absence is pending; corrupt/conflicting receipt is visible refusal, never fabricated success.
5. For each pending object, safely read through LocalSink into a private regular staging file; hash/size must match expected ciphertext or label before a subprocess can read it. Never pass an unvalidated mutable source path to rclone. No plaintext source or private catalog enters staging/upload.
6. Observe exact remote object. Matching hash and size adopts an earlier upload without copying; explicit missing object permits copy; conflict refuses replacement. Use `rclone copyto <private-file> <remote-key> --immutable --checksum --no-update-modtime --retries 1 --low-level-retries 1`. Treat command success as a request outcome only, then obtain independent remote checksum evidence. Never use sync, move, purge, delete, overwrite, or automatic dedupe.
7. First establish unique remote identity using ObserveRemoteInventory below; `--stat` alone is never uniqueness evidence. Prefer `rclone lsjson <remote-key> --stat --hash-type SHA-256 --no-modtime --no-mimetype`: validate exactly one file, expected name/path, nonnegative matching size and a valid hash. Where SHA256 is unavailable, `rclone hashsum SHA256 <remote-key> --download` computes it by reading the remote bytes. Never substitute size-only verification. Report the verification path for each bundle (backend SHA256 versus downloaded SHA256, including mixed bundle/label paths) and aggregate downloaded-checksum counts without logging credentials or child stderr. Missing remote/auth and ambiguous/invalid output do not prove an absent object. Duplicate-name evidence refuses adoption.
8. After bundle and label checks match, spool candidate receipts privately. Publish and verify each immutable UploadReceipt using LocalSink only after the final ObserveRemoteInventory fence confirms the same unique remote object IDs, sizes and available hashes. Receipt candidates retain their exact downloaded-checksum observation when backend SHA256 is absent. An uncertain receipt write is pending; rerun reconciles its exact contents. Stop remote phase on its first failure with verified-so-far count and an actionable pending/unavailable note; keep all local files. Cancellation cancels and joins any owned subprocess before returning.
9. Report local counts separately from Drive verified/already-recorded/pending/unavailable state. Remote failure alone does not change local success exit status or marker. A later daily/manual pass retries; there is no in-pass automatic retry storm.


### Remote uniqueness observation (PQ-1 refinement; awaiting hub approval)

ObserveRemoteInventory resolves the configured folder component by component with nonrecursive lsjson listings. At each component require exactly one matching directory with a nonempty stable ID for Drive; duplicate or uncertain names refuse. Bound depth to 16 and each listing to 4 MiB / 10,000 children. Pin the resolved Drive destination with internally constructed `--drive-root-folder-id <ID>` for the rest of this phase, so later commands do not resolve ambiguous ancestor names again. The disposable local-backend conformance uses its single filesystem path identity instead of Drive IDs.

Inside that pinned folder, perform one recursive lsjson inventory before effects and one after effects, including directories and files with relative paths and IDs, rejecting any duplicate path, directory/file collision, malformed identity or overflow. Stream inventories to private spool files (2 GiB each), retaining a compact path/ID/hash/size index capped at 100,000 entries / 64 MiB. Exceeding any bound leaves Drive visibly pending/unavailable. This additional temporary family is owned and swept with the existing upload staging family. Root resolution and its IDs are checked again before receipt publication. Ordinary new archive subdirectories are allowed only when the final inventory proves one directory per path; no receipt is published on ambiguous final structure.

Before and after a downloaded hashsum observation, stat the exact selected file and require the same ID, size and available hashes. Its candidate receipt carries that ID and downloaded SHA256. The final inventory must resolve that path to the same unique ID with matching size/hash metadata before publication; an unknown/change/duplicate prevents the receipt. This is bounded observational evidence, not a lease over arbitrary external Drive writers: a later mutation remains outside the receipt timestamp, as already specified. No remote deletion or dedupe is attempted.

Receipt publication is deferred until this final fence. A failed/uncertain phase keeps all remote objects but no new receipts; the next pass adopts matching objects after a fresh unique inventory. Existing committed receipts are still skipped. Count inventory traversals and entries at N/2N as well as commands: exactly two inventories, no per-object directory enumeration, constant ancestor depth, and O(N) retained index/work. FakeRclone models duplicate ancestor names, duplicate files, replacement IDs and namespace changes at both inventory barriers. Required guards are no receipt without unique final identity and no destructive effect for unknown/ambiguous state.

### State decisions (ARCH-ORDER)

| State and event | Next state / effect |
| --- | --- |
| CheckingBundle or CheckingLabel + exact matching observation | next object's check, or ReadyToRecord |
| CheckingBundle or CheckingLabel + confirmed absent | corresponding Copying state / immutable copy |
| Copying + successful request | corresponding Checking state / fresh checksum observation |
| Any check + conflicting bytes or malformed authority | Refused / report; no write |
| Any operation + unavailable or unknown outcome | Pending / report; no receipt |
| Both object observations match | InventoryAfter / final unique inventory fence |
| InventoryAfter + exact unique identities | ReadyToRecord / publish candidate receipt |
| ReadyToRecord + durable exact receipt | Verified / count |
| Receipt publication + unknown/error | Pending / reconcile next pass |
| Any active state + cancel/deadline | Pending / cancel and join child, clean private staging |
| New pass + receipt absent after partial remote write | CheckingBundle / observe and adopt exact bytes |

Invariant: receipt publication requires two exact independent remote checksum observations for this destination and bundle. An old or late result for another identity is rejected. Serial effects have no background worker surviving return. A process death may leave one remote object or both objects without a receipt; the next pass observes them before choosing effects. The existing continuous.lock excludes two active continuous owners; LocalSink immutable publication protects receipt races. Out-of-band remote writers remain external authority: no destructive repair is attempted.

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

ARCH-CONSTRAINTS: this is optional daily batch work, never a presenter/UI operation. The measured backlog in the issue is about 758 MB; actual bundle count is not yet measured. One subprocess at a time; one ciphertext stage at a time bounded by existing MaxCiphertextBytes (2 GiB), labels/records by MaxMetadataBytes (4 MiB). Set a 30-minute remote-phase budget and 60-second metadata-command cap as initial design choices; transfer/checksum uses remaining phase time. Deadline leaves a visible pending backlog. There is no claim that 758 MB must finish within a particular network duration. Local success is already durable before the budget is spent. CPU hashing is O(bytes); network normally upload bytes plus metadata, with checksum download fallback adding one readback. Completed receipts avoid subsequent ciphertext reads and remote commands.

Catalog snapshot streaming is O(catalog bytes), bounded to 2 GiB of temporary metadata with a visible unavailable note on exhaustion, preserving local success. RAM is bounded to record/stream buffers; no full catalog or all-object listing is retained. Source catalog is read a constant number of times per pass; receipt operations and remote commands are O(bundle count), independent of receipt history scans. Count N=8 and 2N=16 runs: catalog visits linear, at most constant commands per object, exactly two copy effects per previously absent bundle, zero copy effects on rerun. Time measurements supplement counts, not replace them.

ARCH-FUNERAL: upload receipts are one bounded metadata file per bundle per configured destination; their last consumer is retry/dedup until a future explicit retention policy. Retention is deliberately manual with local/remote copies kept, as required; cost is receipt bytes and one filesystem entry per bundle/destination, reported in tests and live evidence. Cap receipt serialization to 8 KiB; refuse new receipt publication visibly if its managed upload catalog exceeds 100,000 receipt entries per destination. Count existing destination entries once at phase admission and update that count on successful publication; never enumerate the receipt directory per bundle. Enforce writer and reader bounds, never silently prune verification evidence. Staging/snapshot files are pass-owned private files; clean synchronously on success/failure/cancel, sweep only this family's stale files after acquiring continuous.lock next pass. Remote objects and existing local bundles have the existing retain-until-explicit-retention policy; no automatic deletion is introduced. Register both artifact families and every new production source with artifactpath inventory/retention contracts.

ARCH-SECURE: remote names, folders, persisted receipts, local public labels and external JSON are untrusted. Strict typed decoders require fields/version and reject truncation, null, wrong identity, overflow and unsafe keys; receipt unknown versions refuse with advice. External lsjson may have documented extra backend fields but cannot omit required observation fields or contain duplicate required JSON keys. Subprocess argv contains only executable, public remote/key paths and private encrypted staging paths. Capture stdout at 64 KiB and stderr at 16 KiB, classify exit status without echoing arbitrary child stderr or config. Pair creates/reads no credentials; rclone alone reads operator-managed credentials. Tests isolate HOME/config and local backend. No new secret is generated.

ARCH-PURE separates typed decisions/parsers from IO; ARCH-DRY reuses LocalSink and Catalog streaming rather than cloning safe filesystem code; ARCH-PURPOSE includes the entire historical backlog and live operator proof, not just new bundles. ARCH-MOCK, ARCH-ORDER, ARCH-CONSTRAINTS, ARCH-SECURE and ARCH-FUNERAL are exercised above and by the tests below.

## Implementation tasks (one close boundary)

### Task 1: Pure contracts and configuration

Files: create drive_model.go and drive_model_test.go under cmd/internal/historyarchive; modify continuous_config.go and continuous_config_test.go.

- [ ] Write failing unit tests for Destination parsing/defaults and receipt validation/identity; config with absent remote fields preserves existing local config, invalid remote settings remain remote-only failure.
- [ ] Run `go test ./cmd/internal/historyarchive -run 'TestDriveDestination|TestUploadReceipt|TestContinuousConfig' -count=1`; retain meaningful RED.
- [ ] Implement typed contracts, strict decoding and pure NextUpload transitions; add sequence tests that vary duplicate, cancellation, missing and uncertain observations.
- [ ] Run that command GREEN and commit with issue reference and session Agent trailer.

### Task 2: Stateful rclone boundary and conformance

Files: create historyarchive/rclone.go, rclone_test.go and archivetest/rclone.go.

- [ ] Write TestRcloneObservation and TestRcloneCopyRecovery against the stateful runner; use the adversarial strategy/guard table below.
- [ ] Run `go test ./cmd/internal/historyarchive/... -run 'TestRclone|TestFakeRclone' -count=1` RED; implement minimum adapter and stateful fake, then GREEN.
- [ ] Add an opt-in installed-rclone local-backend conformance test using isolated config/private roots; run it once against installed rclone, retaining version and command results. It never contacts Drive.
- [ ] Commit the independently verified boundary.

### Task 3: Safe mirror and continuous integration

Files: create historyarchive/drive.go, drive_test.go; modify historyarchive/continuous.go and continuous_test.go; archivecmd/continuous.go and its tests; artifactpath/manifest.go plus applicable artifact-family declarations/tests.

- [ ] Write TestDriveMirrorRecovery and TestContinuousDriveIsolation using the DriveMirror.Run strategy and independent guards below.
- [ ] Run `go test ./cmd/internal/historyarchive/... ./cmd/internal/archivecmd -run 'TestDrive|TestContinuous' -count=1` RED; implement private catalog snapshot, verified staging, receipt publication and optional post-local phase; rerun GREEN.
- [ ] Add counted 8/16 bundle cycle tests and second-pass zero-copy assertions using actual production catalog and fake retained remote state.
- [ ] Run targeted packages with race detector where appropriate, and `go test ./cmd/internal/artifactpath ./cmd/internal/testpair ./cmd/internal/repolint -count=1` (verify package paths before execution); commit.

### Task 4: Docs, operator conformance and normal landing

Files: README.md; atlas/history-backup.md (already indexed); issue Log and this plan.

- [ ] Document config defaults/validation, independent local/Drive status, receipt location/lifetime, checksum fallback and pending/conflict recovery. Include an exact operator live command and prerequisites without secrets.
- [ ] After fake-backed qualification, ask hub to arrange the one operator-run real Drive pass. It uses production continuous configuration, uploads historical encrypted pairs, verifies checksums and receipt counts, then reruns to prove zero new copies and unchanged local files. Operator checks visible Drive bundle/label names. Retain counts/bytes, adapter version, command exit/status and receipt evidence; never include credentials. Do not stage :0, restart or relaunch without hub coordination.
- [ ] Push with sdlc pr, run recorded `sdlc local-check run --name suite`, close with verified evidence and binary-owned fresh review. Actuals measured by sdlc, no typed estimate before plan-quality gate.
- [ ] Push final Log updates before `sdlc merge --yes`. Let landing turn integrate main and validate; report PR/landed hash from returned Git output to hub.

## Adversarial strategy and mechanical guards

| Risky function | Adversarial strategy | Independent mechanical guard |
| --- | --- | --- |
| ParseDriveDestination / object-key derivation | generate Unicode/control/colon/absolute/dot-segment spellings and long segments | parsed destination roundtrip plus all keys remain beneath exact configured prefix; argv never changes option count |
| DecodeUploadReceipt / MatchReceipt | mutate every required field, truncate at every boundary, duplicate keys, null/overflow/version skew | no mutation authorizes Verified for an independently constructed expected pair; bounded serialization/read |
| DecodeRemoteObservation / ObserveRemoteInventory | varied optional backend fields, duplicate/omitted required fields, wrong size/hash/name/IsDir, oversized output | no malformed observation authorizes copy-as-absent or receipt; size and independently computed SHA256 must both match |
| NextUpload | generated sequences of observations/outcomes, cancellations and stale identity results | append receipt implies two matching remote object observations; uncertain outcomes never imply absence; transition/effect enum assertions |
| DriveMirror.Run | barrier-controlled child completion and receipt/staging IO faults at each publication boundary | retain and compare local files; no returned live child; no receipt before checksum success; rerun converges from actual fake bytes |
| PublishUploadReceipt / StageVerifiedObject / SweepUploadStaging | symlink replacements, corrupt complete receipts, interrupted publication and stale stage | LocalSink safe-open/no-overwrite contract; ordinary cleanup plus next-owner sweep; destination/bundle mismatch refuses |

## Approval checkpoint

Full flow: send this plan and state/coverage tradeoffs to the hub and wait for approval before sdlc change-code or production edits. The separate upload catalog and explicit operating limits are proposals requiring that approval. No remote call has been made during planning.

## Revisions

- 2026-10-18: hub approved the full design, receipt family and bounds (delivery f7e7fe62-5f41-4133-ba43-4296e82fd885). Added its requested per-bundle verification-path reporting and fallback counts. Hub will arrange the real Drive pass when fake-backed implementation is ready.

- 2026-10-18 plan-quality PQ-1/PQ-2: specified bounded before/after namespace inventories, pinned Drive folder identity and deferred receipt publication, with checksum observations fenced to exact IDs. Compressed duplicated prose case inventories into named function strategies. This identity mechanism delta is submitted to the hub before rerunning change-code.
