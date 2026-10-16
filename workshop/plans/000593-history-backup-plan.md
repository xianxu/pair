# History backup foundation implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy); use bounded subagents where suitable and superpowers-executing-plans for session-warm integration. Steps use checkboxes.

**Goal:** Produce verified, encrypted, independently readable history bundles and their catalog, reusable by #394 and #592.

**Architecture:** A pure inventory/manifest/planner defines the format and grouping. One synchronous IO coordinator streams selected files through tar → zstd → GPG, publishes through a sink, verifies it, and records the result. The durable facts are bundle, label and catalog only; no job checkpoint, intent ledger, background worker or cleanup authority.

**Tech Stack:** Go archive/tar, SHA256 and JSON; installed zstd -19 (single worker) and GPG batch public-key encryption; local files and existing durablefile helpers.

## Scope and choices

The approved ops design is the requirements source. Prefer a small Go package plus thin CLI over a shell-only pipeline (harder to reuse and verify across #394/#592), or a general backup engine (unnecessary scheduler/state/plugin surface). Implement source declarations and explicit selection now; scanners, quiet-period policy execution, SQLite snapshots, GC archived phase, Drive and restore/search commands remain their named project pieces. SQLite source declarations require snapshots; this issue does not accept live SQLite bytes through the manual route.

CLI: `pair archive inventory` prints the embedded inventory; `pair archive create --request FILE` accepts a strict versioned JSON request with source kind, host identity, root, explicit relative file list, recipient fingerprints, local sink directory and catalog path. Backup destinations and selected files are explicit; root tokens resolve from request overrides or XDG/agent defaults. No implicit production scan or default backup write. JSON request is bounded and unknown fields refuse. The library accepts the same validated request with injectable sink/codec/clock. This command is the end-to-end consumer, not an unreferenced library.

## Format and inventory

One embedded `cmd/internal/historyarchive/sources.json` table declares source kind, owner, root token, path pattern(s), on-expiry/continuous mode, existing local-retention rule (descriptive only), snapshot requirement and sink rule. Root tokens are values such as pair-data, claude-home, codex-home, qoder-home, ops-worktree and git-common; no developer home paths or private key material. Sink selection is a pure ordered kind/age rule evaluator; current default is a named local sink supplied explicitly. Validate missing/duplicate kinds and ambiguous rules. Inventory output and create admission derive from this table; future collectors consume it rather than copying it.

Kinds cover Pair terminal events, parked captures, scrollback, Couch session/ledger/thread history, webcast recordings, adapt/fullscreen diagnostics, Claude/Codex/Qoder transcripts, Codex database snapshots, operator QA/assets/local side documents, sdlc runtime data and this catalog. Pair-owned declarations reference artifactpath's existing family identities where applicable; do not duplicate its filename classification engine. Operator-local side documents keep host/repository/worktree identity in the source metadata; actual git-tracking selection belongs to #592.

Version 1 bundle: `<kind>/<UTC year>/<month>/<kind>-<from>-<to>-<id>.tar.zst.gpg`, neighboring `.json` label. ID is SHA256 of canonical manifest content (not ciphertext, which is randomized). The manifest contains version, kind, host, source root/optional repository and worktree identity, time range, compression, recipient fingerprints, and sorted file entries `{path,size,mtime,sha256}`. Files live under `files/<original relative path>`; `MANIFEST.json` is reserved at tar root. No absolute/traversal/duplicate paths, symlinks, devices or hardlink entries. Tar records contain regular data, normalized modes and no user/group identity. Label includes version/id/kind/range/file count, raw and ciphertext bytes, zstd settings, recipients and ciphertext SHA256; no paths/host/root/file hashes. Catalog includes label, sink identity/keys, verification time and manifest file list/source identity; it is private local metadata.

## Core concepts

| PURE entity | Lives in | Status |
|---|---|---|
| SourceInventory, SourceRule, SinkRule; validation/route selection | cmd/internal/historyarchive/inventory.go, sources.json | new |
| ArchiveRequest, SourceIdentity, FileEntry, Manifest, BundleLabel, CatalogRecord | cmd/internal/historyarchive/model.go | new |
| PlanBundles, canonical identity and format validators | cmd/internal/historyarchive/plan.go | new |
| PublicationDecision (absent, verified-existing, conflict) | cmd/internal/historyarchive/publication.go | new |

Inventory owns N source rules; request selects one kind/root and N files. PlanBundles partitions one request into N manifests. Manifest has one content identity and one label per encrypted representation. Catalog records reference one verified bundle on one sink. All transformations validate explicit input values with no IO or clock reads. These are the first shared definitions for two imminent consumers (ARCH-DRY). PublicationDecision interprets observed complete/partial/conflicting object evidence; it owns reuse/refusal decisions rather than adapter booleans.

| INTEGRATION entity | Lives in | Status | Wraps |
|---|---|---|---|
| SelectedFiles | cmd/internal/historyarchive/files.go | new | descriptor-relative regular-file reads under explicit root |
| Codec, CommandCodec | cmd/internal/historyarchive/codec.go | new | tar writer, zstd/GPG processes and bounded diagnostics |
| Sink, LocalSink | cmd/internal/historyarchive/sink.go | new | private local directory; put/get/list/verify |
| Catalog | cmd/internal/historyarchive/catalog.go | new | locked append-only JSONL, sync and validated tail |
| Archiver | cmd/internal/historyarchive/archive.go | new | synchronous orchestration, injected sink/codec/clock |
| MemorySink, TestCodec | cmd/internal/historyarchive/archivetest/fakes.go | new | stateful bytes/object map, recipient availability, failure barriers |
| Run | cmd/internal/archivecmd/run.go | new | strict CLI request and signal cancellation |
| Families, runStreamingSubcommand | cmd/internal/dispatcher/dispatcher.go, cmd/pair-go/main.go | modified | route CLI through standard dispatcher |

Pure functions receive observations, not interfaces that secretly perform IO. Archiver injects adapters and calls the pure functions. Reuse commitoutcome for indeterminate publication diagnostics and durablefile.SyncDirectory/WriteAtomic for small metadata and existing Unix/context filesystem patterns; streaming bundle publication must not load a GiB into WriteAtomic. Stateful test sink retains bytes across failed puts, corruption, retries and fresh Archiver instances. TestCodec retains its public-key set and generated artifacts and can pause/fail a stream; it is never presented as encryption. Real codec conformance uses an isolated temporary GPG home/keypair and actual zstd, running at implementation/close and whenever codec changes.

## Flow, interruption and retries (ARCH-STATE)

One invocation acquires a nonblocking catalog lock; concurrent callers using that catalog receive a clear busy error. LocalSink also uses atomic no-replace publication so callers with different catalogs cannot clobber objects. No long-lived process state survives return.

1. Validate request, inventory, recipients and bounds; open explicit safe source root. Hash selected regular files and obtain identity/size/mtime. Plan by month and size; derive manifest IDs.
2. If valid existing catalog record and sink objects match this manifest ID and ciphertext checksum, return the verified record without another append. Missing/corrupt objects fail explicitly.
3. For an uncataloged ID, inspect sink bundle/label. A complete matching label+bundle is re-read and adopted; conflicting metadata or bytes refuse. A bundle without a valid label is incomplete and is never treated as a backup. Report its key for inspection; do not overwrite or infer ownership from name alone.
4. For absent objects stream tar → zstd → GPG into a private temporary ciphertext file. Rehash payload while writing tar and compare the prepass hashes/identity; change during read fails. Wait for every process, require successful exits, fsync ciphertext. No plaintext tar temporary file.
5. Publish ciphertext and label without replacement, read back both, compare full encoded label and ciphertext size/hash. Only verified evidence permits a catalog append. Append one bounded JSON line and fsync before returning success.
6. Any error/cancel stops and joins subprocesses, closes descriptors, removes this invocation's temporary ciphertext, and preserves sources and published objects. Error identifies phase and one next action. Unconfirmed publication is inspected on retry, never blindly overwritten. Catalog append failure can leave verified uncataloged bytes; retry can adopt them. The catalog cannot be selected as its own input, and first creation syncs its parent directory. A torn trailing catalog line is reported and requires explicit repair; it is not silently trimmed or skipped.

The most dangerous ordering is successful bundle publication followed by failed label/catalog publication: absence of the catalog never permits source deletion. #394 must require a successful verified result; it cannot use mere file existence. A crash with an incomplete orphan does not require background recovery in this foundation: the next explicit call reports it. Tests inject faults at publication/verification/catalog boundaries and exercise fresh-process retry observations. No process may outlive the invocation.

## Operating envelope and artifact lifetime

This is an explicit batch path, never a Couch UI/startup critical path (ARCH-CONSTRAINTS). Proposed conservative initial bounds: 10,000 selected files/request, 4MiB request/manifest/catalog-record, 256MiB target raw bundle size, 1GiB hard raw bundle/file bound, 2GiB ciphertext bound, 64MiB catalog shard. Month grouping uses file mtime UTC; zero/small bundles allowed, oversized single files refuse without mutation. Bound checks precede reading payloads and encoding metadata; safe subtraction prevents overflow. Metadata remains bounded; data streams with a fixed copy buffer and one zstd process/thread per invocation. zstd -19 has its own working memory, to be measured in the real fixture; do not claim a strict low total RSS. No host-wide scans or parallel compressions; no network. Slow work is cancellable, no arbitrary success deadline. The operator controls batch frequency; default full suite uses tiny fake fixtures, real bounded conformance is opt-in and explicitly run at close.

Catalog writer enforces its 64MiB shard bound before append and refuses with an action to select a new catalog file; never silently loses old records. Bundle/label/catalog are intentionally retained until an explicit later retention policy: disk cost is compressed/encrypted payload plus bounded metadata per bundle, printed in result. Temporary ciphertext is owned by the invocation and removed on normal failure/success; interrupted processes can leave one `.partial` file per active invocation. Under the catalog lock, next invocation removes only its dedicated catalog staging-directory remnants; it never removes published sink objects. No staging directory may overlap sources/sink payloads. Document these lifecycle owners and accounting in atlas/storage-retention-io.md (ARCH-FUNERAL).

Trust boundary (ARCH-SECURE): local user supplies roots and trusted recipient fingerprints; selected payload bytes/names and sink objects are untrusted. Validate JSON/version/path/hash lengths; use root-confined, no-symlink regular-file opens (including parents), shell-free argv, bounded stderr, batch/no-tty GPG. Never log payloads or key secrets. New directories0700/files0600. GPG encryption requires no secret-key operations. Validate fingerprint syntax and exact public-key availability, including encryption capability/expiry; no keyserver fetch or network discovery. Checksums prove returned bytes match the written ciphertext, not authenticity against someone who can rewrite both local catalog and sink. No signed-manifest/security-boundary claim.

## Implementation and tests

### 1. Pure format and inventory
- [ ] Write failing table tests in inventory_test.go/model_test.go/plan_test.go: all declared kinds, malformed JSON/version, age boundaries, source identity, canonical ID under reordered input, path traversal/duplicates, month/size partitioning and all budget edges.
- [ ] Implement the declared inventory and pure contracts; run `go test ./cmd/internal/historyarchive -run 'Inventory|Manifest|Plan|Publication' -count=1` (PASS); commit.

### 2. Local publication and catalog
- [ ] Write stateful sink/catalog tests in sink_test.go/catalog_test.go: short writes, no-replace collisions, symlink parents, corruption, get/list consistency, lock contention, duplicate record, torn tail, writer byte cap and directory durability failures.
- [ ] Implement stream-to-temp+fsync+no-replace publication and strict catalog append. Test PublicationDecision independently on missing/partial/complete/conflict observations. Run focused tests, commit.

### 3. Streaming archiver and codec
- [ ] Add files_test.go/codec_test.go/archive_test.go with stateful fakes: changed file between passes, cancellation at each barrier, missing/expired recipient, compressor/encryptor exit errors, corrupt verify, catalog failure, complete uncataloged adoption, incomplete bundle refusal, retry with no duplicate catalog and preserved source bytes.
- [ ] Implement safe source reads, tar stream, joined subprocess pipeline and coordinator. No shell pipeline; propagate each command's exit and bounded diagnostic. Test injected failure orderings deterministically, then `go test -race ./cmd/internal/historyarchive/... -count=1`; commit.
- [ ] Add codec_live_test.go opt-in isolated zstd/GPG round-trip. Generate a disposable test key, archive unicode/empty/binary fixtures, decrypt/decompress independently, inspect tar+manifest and compare every SHA256. Assert label has no source paths; bound/measure fixture duration, output and process resource use. Automated tests use no real source directories or personal key. The separate issue-required live smoke below uses the configured public recipient only.

### 4. CLI, docs and qualification
- [ ] Implement archivecmd/run.go and run_test.go; wire dispatcher and streaming entrypoint with route tests. Strict JSON request drives the same production Archiver; help/inventory perform no backup writes. Test user-facing phase/action errors and cancellation.
- [ ] Update README.md, atlas/history-backup.md, atlas/index.md, atlas/storage-retention-io.md and cmd/internal/artifactpath/manifest.go for every production reference. Document format, JSON example, bounds, partial objects, catalog privacy and operator-only key recovery checklist. Include dependency contract for #394/#592.
- [ ] Run targeted package/race/classification tests and bounded real codec conformance. Run `make build`; invoke built `pair archive inventory` and a scratch-only create round trip. Record concrete evidence and no source mutation.
- [ ] Run the issue-required live smoke on a small explicitly selected real ops QA source, encrypted to the approved public fingerprint, with a private temporary local sink/catalog. Verify by read-back SHA256 and label, report paths/counts only; do not decrypt using a personal private key, print source contents, alter ops or export keys. Preserve the small verified bundle for operator inspection and name its temporary location/lifetime.
- [ ] Push PR, run recorded suite and `sdlc close --issue 593 --verified ...`; fix findings with negative controls. Merge main and `sdlc merge --yes` on SHIP. Report real hashes/PR and measured costs to hub. No live backup schedule, operator key export or machine configuration edits.

## Revisions

None yet. Proposed file/budget details await TL approval; changes will be appended here rather than erasing the original design.

### 2026-10-15 — TL approval and catalog simplification

TL receipt4042bd3a approves dc721f5f with two superseding changes: under the catalog lock, repair an incomplete final JSONL line by truncating to the last complete record and syncing before append. Complete records are retained and validated; a complete record conflicting with the stored bundle refuses. Any bundle whose catalog line was torn is a verified uncataloged object, handled by the existing retry adoption. Roll a full64MiB shard automatically to the next numbered shard; no operator shard selection. Shards are one logical catalog and duplicate lookup spans them by streaming reads; no unbounded in-memory index. New-shard creation is synced and serialized by the same catalog lock. Tests cover tail truncation before retry, complete-line conflict, boundary rollover, crash/retry across rollover and preserved earlier shards. These rules supersede the earlier tail-refusal/full-shard-refusal text and tests; all other approved bounds/scope remain.

### 2026-10-15 — plan-quality round 1 refinements

PQ-1: adoption/reuse always reconfirms durability; readable bytes alone are not success. `LocalSink.Verify` opens and hashes each final object, syncs the opened file and every directory publication edge through the configured sink root, and only then returns verified evidence. `Catalog.Open`/`Catalog.Append` sync the selected shard and newly created shard directory edge before reporting an existing or new record committed. Tail repair syncs the truncated shard before any append; rollovers sync new shard and parent. PublicationDecision consumes confirmed durable verification, not mere presence. An error at any of these barriers returns no success result, including on repeat. This is unconditional retry verification of ordinary files, with no extra persisted state. Test faults after successful write/link/append but before sync; retry in a fresh instance, fail sync again, assert no success/catalog advancement; remove fault and assert one durable record. Applies equally to new objects, adopted objects, existing catalog entries and rolled shards.

PQ-2: the following function-level strategy table is the authoritative test decomposition; it consolidates the task case lists above (not a second set of tests).

| Production function/method | Adversarial input/event class | Mechanical guard / independent oracle |
|---|---|---|
| DecodeInventory, SourceInventory.Resolve, SourceInventory.Route | malformed/duplicate source or overlapping sink rules, missing/default roots | strict decoder plus table-driven boundary examples; every checked-in kind resolves |
| DecodeRequest, ValidateManifest, Manifest.ID | unsafe paths, repeated entries, malformed versions/fingerprints/hash, reordered input | validator tables; canonical ID unchanged by selection order and changed by payload/source identity |
| PlanBundles | month/size/count/metadata edges and oversized member | assert every admitted file occurs exactly once, month and byte limits hold, sum preserved |
| DecidePublication | absent/partial/matching/conflicting observed objects, unconfirmed sync | exhaustive observation table; only durable verified matching evidence can reuse |
| SelectedFiles.Describe, SelectedFiles.WriteTar | symlink/special/replaced/mutating files | descriptor confinement and before/after identity/hash; inspect independent tar bytes and preserved source |
| CommandCodec.CheckRecipients, CommandCodec.Encode | missing key/tool, bad exit, partial startup, blocked pipe and cancellation | stateful codec/process fixtures with barriers; every child joined and no successful output after any failed stage; real isolated GPG/zstd round-trip |
| LocalSink.Put, Get, List, Verify | collision, traversal, corruption, short write, late sync error | portable real directory and fault seams; final byte digest/size, no replacement and repeat durability confirmation |
| Catalog.Open, Find, Append | incomplete tail, conflicting complete records, rollover and concurrency | exact bytes before/after repair; lock barriers; automatic next shard; streaming search across all shards and repeated sync failures |
| Archiver.Create | failures/interruption between transform, publication, verification and catalog | stateful sink/codec and actual temporary catalog; new Archiver observes same objects, never returns success early, preserves sources and avoids duplicate record |
| archivecmd.Run, dispatcher.Resolve, runStreamingSubcommand | malformed/unknown options and requests, cancellation, command routing | fake-backed CLI end-to-end and actual scratch create; one phase/action error; inventory/help do not write |
