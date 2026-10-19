# History backup

`pair archive` is an explicit batch backup path. It archives selected stable
regular files as tar → zstd → GPG public-key encryption, reads the published
ciphertext back to verify its SHA256 and size, and records verified bundles in a
private local catalog. It preserves every source file. The entrypoint is
[`archivecmd`](../cmd/internal/archivecmd/run.go); the implementation is
[`historyarchive`](../cmd/internal/historyarchive/archive.go).

Compression uses `-9 -T0` for `codex-database` and `-19 -T0` for every
other kind. The manifest and label record the actual level; readers accept
canonical positive zstd levels independently of this writer policy. Encoding
rejects levels above 22 before spawning and uses `--ultra` for levels 20–22.
Existing level-19 bundle IDs remain readable, and unchanged source content is
skipped rather than recompressed. On the measured 2 GB history snapshot, level 9
produced 205 MB in 2.2 s versus level 19’s 147 MB in 56.3 s (both `-T0`),
saving about 454 CPU-seconds for 58 MB more per changed daily bundle. This
trades roughly 21 GB/year of storage for much less daily compression work.
`PAIR_ARCHIVE_LIVE=1` enables the isolated-key
codec round-trip conformance test for levels 9, 19, 20 and 22.

## Commands and request

```sh
pair archive inventory
pair archive create --request /private/tmp/pair-archive-request.json
```

Help and inventory perform no backup writes. Inventory prints the embedded
[`sources.json`](../cmd/internal/historyarchive/sources.json): source kinds,
root tokens, indicative file patterns, retention owners, snapshot requirements,
and sink routing policy. Patterns select matching files or directory subtrees; future collectors must
recurse into selected directories. It does not scan those roots. Create reads a regular
JSON request file; it refuses symlinks at the final component, nonregular files,
unknown fields, trailing JSON and extra CLI arguments. Files are explicit paths
relative to one source root, not glob expressions or an instruction to discover
all matching files.

A request for one operator-selected document looks like this:

```json
{
  "version": 1,
  "kind": "ops-documents",
  "source": {
    "host": "my-machine",
    "root": "/private/tmp/archive-demo-source",
    "repository": "example-repository",
    "worktree": "main"
  },
  "files": ["qa/example.md"],
  "recipients": ["0123456789ABCDEF0123456789ABCDEF01234567"],
  "sink": "/private/tmp/archive-demo-sink",
  "catalog": "/private/tmp/archive-demo-catalog/catalog.jsonl"
}
```

The fingerprint is a placeholder: replace it with the trusted full fingerprint
of an installed, usable public encryption key. The example does not create its
source file. Set the actual host and absolute source, sink and catalog paths;
`repository` and `worktree` are optional identity metadata. The source tree must
be separate from both the sink and catalog directory. On macOS, use canonical
paths such as `/private/tmp`, since source and storage traversal rejects symlink
components. Temporary example destinations survive only until explicitly removed
or the host's temporary-file policy removes them; choose durable storage for a
real backup.

When `source.root` is omitted, inventory resolves it using these defaults:

| Root token | Resolution |
| --- | --- |
| `pair-data` | `PAIR_DATA_DIR`, then `XDG_DATA_HOME/pair`, then `~/.local/share/pair` |
| `claude-home` | `CLAUDE_CONFIG_DIR`, then `~/.claude` |
| `codex-home` | `CODEX_HOME`, then `~/.codex` |
| `qoder-home` | `QODER_HOME`, then `~/.qoder` |
| `couch-store` | `COUCH_STORE_DIR`, then `XDG_DATA_HOME/pair/couch`, then `~/.local/share/pair/couch` |
| `slot-store` | Explicit absolute path to one slot’s `.couch` store required |
| `ops-worktree`, `git-common` | Explicit absolute root required |

The inventory covers Pair terminal events, parked captures, scrollback and
diagnostics; Couch session bindings, ledgers and threads; webcast recordings;
Claude, Codex and Qoder transcripts; ops documents; SDLC runtime files; and the
archive catalog. Pair declarations cover both legacy root-level and repo-scoped
artifacts. Couch threads cover the global store and the separately declared
`couch-slot-threads` backend, whose explicit root is a slot’s `.couch` directory.
Transient locks/control files are excluded from declared artifact globs.
`codex-database` is declared but refuses manual selection because
it requires a snapshot collector. Inventory patterns describe policy; this CLI
uses the supplied selection. All currently admitted kinds use public-key
encryption. A public key is enough to create backups: no private key operation,
keyserver retrieval or passphrase prompt is part of creation. `gpg` and `zstd`
must be on PATH.

## Format, verification and retry

[`PlanBundles`](../cmd/internal/historyarchive/plan.go) groups files by UTC mtime
month and size. A canonical manifest determines the bundle ID, including source
identity, selected file paths, hashes and mtimes, recipients and compression.
The encrypted tar contains `MANIFEST.json` and `files/<relative-path>` entries.
Objects are stored below `<kind>/<YYYY>/<MM>/`, with kind, UTC time range and ID
in their names; bundle suffixes are `.tar.zst.gpg`, paired with `.json` labels.

The plaintext label and command output expose kind, time range, file count,
byte counts, recipients, bundle ID and ciphertext SHA256; they omit source paths
and contents. Object names also reveal kind and time range. The local catalog
contains the full manifest, including source identity and file paths, so it is
private metadata even though payloads are encrypted. New directories use mode
0700 and files 0600. Checksums verify returned bytes against the recorded hash;
they do not establish authenticity against an actor able to rewrite both the
objects and the catalog. There is no signed-manifest guarantee.

[`LocalSink`](../cmd/internal/historyarchive/sink.go) publishes without replacing
an existing final object, then verification reconfirms file and directory
durability. Catalog append happens only after both bundle and label verify.
An interruption can leave a published bundle without its label or catalog:
existence alone is never successful backup evidence and never deletion authority.
A complete matching pair without a catalog entry can be verified and adopted by
retrying the same request. An incomplete or conflicting published pair refuses
and requires inspection; retry does not overwrite or delete it. Successful
retries reuse matching verified objects and do not duplicate catalog records.
Changed source bytes or metadata can produce a different manifest and ID.

[`Catalog`](../cmd/internal/historyarchive/catalog.go) holds one cross-process
lock across its shards. The first shard is the requested path, followed by
`.000001`, `.000002`, and so on. The catalog inventory includes both the base
JSONL and all six-digit numbered shards, excluding locks and staging. It rolls automatically before exceeding 64 MiB;
lookup streams all shards. On open, an incomplete final JSONL line is truncated
to the preceding complete record and synced under the lock. Complete malformed
or conflicting records refuse; they are not silently dropped. A bundle whose
catalog line was torn follows the verified-object adoption path on retry.
Durability is reconfirmed for existing records as well as newly written records.

## Bounds and lifecycle

| Resource | Bound |
| --- | --- |
| Selected files per request | 10,000 |
| Request, manifest or catalog record | 4 MiB |
| Target raw bundle payload | 256 MiB |
| Hard raw bundle payload / individual file | 1 GiB; database snapshot kinds 8 GiB, bounded in practice by the 2 GiB ciphertext (#616) |
| Ciphertext per bundle | 2 GiB |
| Catalog shard | 64 MiB, automatic rollover |
| Sink listing result | 10,000 entries |

A file larger than the target may occupy its own bundle up to the hard limit.
Tar headers and compression overhead are additional to raw payload accounting.
Metadata is bounded and file bytes stream. Each bundle uses one zstd process
with `-T0` (all available CPU threads); bundles are compressed serially within
an invocation. Compression working memory depends on level and thread count,
so these limits do not imply a strict low total RSS. There are no host-wide
scans or concurrent bundle compressions within an invocation. The operation is cancellable without a fixed
success deadline; cancellation joins the transform children and tar writer.

The invocation owns its temporary ciphertext and removes it on ordinary return.
Subsequent serialized operations recover their owned staging remnants after a
crash, without removing published objects. Catalog staging is protected by the
catalog lock; sink publication owns its separately serialized staging recovery.
Each sink destination directory retains one private `.archive-put.lock` inode;
cleanup scans at most 10,000 temporary entries in a reserved `.archive-staging`
subdirectory; retained final objects do not consume that budget. Internal
lock/staging/partial names are unavailable through object Get/Put and hidden from List.
Bundles, labels and catalog shards are retained until an explicit future
retention decision. Disk cost includes encrypted bundles, plaintext labels and
private catalogs, plus temporary staging while a write is active. See
[storage I/O ownership](storage-retention-io.md#history-backup-storage).

## Recovery ownership and current scope

The operator owns decryption-key recovery. Before depending on backups:

1. Confirm the intended recipient fingerprint and keep the corresponding public
   key available to the backup machine.
2. Preserve the matching private decryption key, its passphrase where applicable,
   and recovery instructions in the operator's 1Password vault according to their
   key-handling policy. Agents must not export keys or edit personal GPG settings
   as part of this command.
3. Verify that authorized recovery access survives loss of the backup machine;
   record the fingerprint and which vault item holds its recovery material.
4. Perform an operator-controlled recovery drill on a disposable encrypted
   fixture and compare recovered file hashes. Keep the private catalog protected
   and recoverable along with the bundles.

This foundation provides no deletion, Drive transport or restore command; the
schedule and the SQLite snapshot collector are continuous mode's (below). A future archive-before-eviction consumer
must require successful verified catalog evidence; mere object existence is
insufficient. It does not change current source retention policies or lift the
diagnostic deletion fence.

Normal tests use stateful fake codecs and stores. The opt-in conformance test
creates only a disposable keyring, encrypts a small tar fixture, independently
decrypts/decompresses it, and compares exact tar bytes:

```sh
PAIR_ARCHIVE_LIVE=1 go test ./cmd/internal/historyarchive -run '^TestCodecLiveRoundTrip$' -count=1 -v
```

It needs `gpg`, `gpgconf` and `zstd`; it does not use personal private keys or
source directories. The test cleans its temporary keyring and isolated agent.

## Continuous mode (#592)

`pair archive continuous [--kind KIND] [--json]` is one pass over the continuous kinds
(`historyarchive.ContinuousKinds`): Claude, Codex and Qoder transcripts, Codex
databases, ops documents (`teams/*/qa`, `teams/*/assets`,
`teams/*/tl-status-*.side`) and each repository's git-common `sdlc/`. It is
configured by `$PAIR_DATA_DIR/archive/continuous.json` (`ContinuousConfig`:
recipients, sink, catalog, `ops_root`, `workspace`); without that file a pass
does nothing.

- **Selection:** regular files unmodified for 3 days (`ContinuousQuiet`);
  symlinks are never followed. A pattern matching a directory takes every file
  beneath it (Codex nests sessions by date).
- **Dedup:** per file, by source identity (host, root, repository, worktree),
  relative path and content hash, from the catalog's manifests
  (`ArchivedIndex`); an unchanged size and mtime skips rehashing. A changed file
  is a new version. Worktree slots share their repository's git common dir, so
  sdlc data is archived once per repository.
- **Databases:** snapshotted daily regardless of quiet. `SQLiteSnapshotter`
  captures an observed cold WAL database as described below; other inputs
  open through SQLite's read-only WAL-reader protocol,
  pins a read transaction and runs `.backup` into a private directory. DB/WAL
  content is never written by the reader; SQLite may create/update SHM read
  marks. The source reader has a size-scaled 5–30 second budget and a 250ms busy timeout,
  with no retry inside the pass. The pair binary's internal `exec-alarm`
  launcher arms a kernel timer before execing SQLite, so collector death
  cannot leave an unbounded read pin. Its periodic interval also protects a
  launcher whose first alarm expired before exec. A scratch flock inherited
  by the child prevents cleanup while an orphaned reader is still alive.
  Source replacement refuses publication. After the source reader exits,
  only the private candidate is switched to `journal_mode=DELETE` and checked
  with `PRAGMA integrity_check` (separate 10-minute ceiling), then published
  under `$PAIR_DATA_DIR/archive-snapshots/`. Opaque backup preserves unknown
  SQL schema/values and implicit rowids. Invalid/truncated tool output never
  counts as integrity evidence. Cancellation discards the candidate and keeps
  the previous stage and verified bundles. A deferred database waits for the
  next scheduled pass or an explicit operator rerun; generic CLI failures are
  unknown failures, never guessed to be busy from their text.
  The read budget is twice the estimated DB+WAL copy time at 190 decimal MB/s,
  rounded up and clamped to 5–30 seconds. The measured approximately 2GB
  history database gets 21 seconds; the context and kernel alarm use the same
  value, never renewed. For a complete WAL header with no WAL, SHM or rollback
  journal files, the collector instead makes a private plain copy and checks
  it against a second complete read through the same held source descriptor.
  Held database and parent identities, size, mtime and ctime must stay unchanged;
  the parent check detects even transient sidecars. This is observed quiescence
  on a reliable local filesystem, not a writer lock. It avoids macOS system
  SQLite 3.54's closed-WAL readonly failure without opening that native source
  through SQLite or creating native sidecars. The two reads share one 5–30 second
  deadline and fixed-size buffers. Changed or unknown capture evidence defers
  without retry and keeps the previous verified backup; let the next scheduled
  pass try again. Unrelated directory churn can conservatively defer this path.
  Proven unsafe paths, size violations and private output failures are rejected;
  ask the hub to repair the reported source or output configuration.
  Active databases retain native backup; a failed cold capture never falls
  through to another acquisition in the same pass. Both paths share private
  DELETE/integrity validation, atomic staging, opaque schema/rowid preservation
  and leased cleanup. Later writes do not invalidate a completed cold capture.
- **Database freshness:** each pass prints archived, unchanged, deferred or
  failed per database, plus its last verified catalog timestamp/age (or never).
  A checked-unchanged source is current even when its stored bundle is old;
  a skipped source has no successful check timestamp. Acquisition success
  alone is not a verified backup. These are runtime reports, with no catalog
  or manifest format changes. `continuous.last` still means pass completion,
  not that every database has a fresh bundle.
- **Drive file count (#705):** the human and version-1 JSON reports reuse the
  pass's existing listings, counting every non-directory entry, including
  duplicate names and unrelated files. A successful final listing replaces the
  explicitly labeled before count; later failure retains the before evidence.
  Missing, skipped or failed initial listings remain unknown, never zero; no
  extra remote query fills the gap. `PAIR_ARCHIVE_DRIVE_MAX_FILES` is a positive
  integer (default `20000`); invalid settings fail before work. A count strictly
  above the threshold asks the hub to arrange year subfolders on Drive.
  Couch uses the same notification path and daily gate as database warnings,
  keyed by the canonical configured destination root, not its physical folder
  ID. Counts, threshold changes and uncertain output do not rearm that day's
  attempt; restart allows one fresh warning. The optional JSON observation
  carries destination, folder ID, before/after phase and count, plus the envelope's
  `max_drive_files`; legacy reports remain readable with an unknown count.
  This reports capacity only and never moves files. Future month-folder
  aggregation can retain the configured-root contract.
- **Freshness warnings (#700):** `PAIR_ARCHIVE_DATABASE_MAX_AGE` is a positive
  Go duration (default `48h`); malformed or nonpositive settings fail before
  work. Only failed/deferred checks warn when the catalog's last verified
  bundle is strictly older than the threshold, or immediately when never
  verified. Successful unchanged validation is healthy, retaining the old
  verified timestamp without recompression. Each warning names the database
  and asks the hub to repair its backup.
  Couch reserves one attempt per exact source/kind/path per UTC day, in memory
  for its lifetime. Restart permits a fresh warning; a failed or uncertain
  delivery consumes the reservation, without retry. Database and Drive reservations share a bound
  of 10,000 keys/day with one overflow warning. Warnings use status, desktop
  output and attention on the connected primary hub's exact thread, never
  inferred source ownership or broadcast. Without that hub, status and desktop
  output explicitly say it is unavailable; no attention item is raised. The
  report and attempted warning remain in `continuous.log`. The small
  `archiveWarningOwner` seam owns future machine-wide steward routing.
  The scheduler requests `--json`: stdout is one version-1 report, bounded to
  4 MiB, including partial database results on a failed pass; stderr remains
  diagnostic. Readers tolerate additive fields, refuse ambiguous/invalid
  known data, and retain original JSON bytes in the existing latest-pass log.
  No catalog, manifest, config or completion-marker format changes.
- Snapshot admission rejects canonical overlap of every native source with scratch or any snapshot staging destination across source kinds and regardless of the pass filter before cleanup or directory creation, including symlink aliases and output directories not yet created. A configuration error names the overlapping roots. Ordinary database publication failures are accumulated while later validated databases continue; cancellation still stops the pass.
- **Never deletes sources or retained copies;** writes the archive store, snapshot root, local sink/catalog, and the configured remote after local success. One pass at a time (`continuous.lock`); a completed full pass
  writes `continuous.last`.
- **Schedule:** Couch checks hourly and runs the command at `nice 10` when
  configured and the last pass is over 24 hours old (`couchcmd/archive_schedule.go`),
  logging to `$PAIR_DATA_DIR/archive/continuous.log`. A manual run uses the same
  lock.
- Files over their kind's raw bound are skipped with a note. Database snapshots use the snapshot bound (`rawBundleLimit`, 8 GiB): they compress well (Codex `thread_history_1.sqlite` 1.60 GB to 115 MB), so the 2 GiB ciphertext bound is their real limit, and a snapshot whose compressed bundle exceeds it is skipped with a note (`ErrCiphertextLimit`), not a failed pass (#616). Each snapshot is its own request, so a skip never drops another database. Cost: a database that changes daily is a new version every pass, and the sink never deletes; at the measured 115 MB per `thread_history_1.sqlite` bundle that is roughly 40 GB a year for that one database. Pruning old snapshot versions is a separate, not-yet-made retention decision.

The raw bound is checked before acquisition and on the private output; it is
not a filesystem quota. One candidate runs at a time. Temporary candidate growth
and native WAL growth behind a pinned reader are limited by reader lifetime,
not an exact transient byte cap. Disk-full fails without replacing prior data.
Scratch `sqlite-*` directories are removed on completion or swept under the
continuous lock once their work lease is free; unreadable leases are left intact
with an error for inspection. Staging keeps the latest validated file per path;
unchanged bytes retain its mtime for dedup. Encrypted versions retain the existing
storage policy above.

### SQLite conformance and native acceptance

The ordinary historyarchive suite uses disposable SQLite writers and kernel-alarm
owner-death fixtures; it never opens native databases. Closed-WAL conformance
qualifies the configured CLI and macOS system CLI, preserving the complete native
fixture image. Controlled real SQLite writers exercise both capture reads,
including rollback with only transient sidecars. Counted tests bound source reads
and observations at B/2B bytes and N/2N databases. The opt-in
`TestSQLiteSnapshotLiveContinuous` requires `PAIR_SQLITE_LIVE_ACCEPTANCE=1` and
`PAIR_SQLITE_LIVE_LAUNCHER` pointing at the tested pair-go binary. Coordinate it
with the hub/operator first: it writes normal encrypted bundles to the configured
local sink/catalog, uses only the codex-database kind, omits Drive, and has a
five-minute outer budget. Select the normal configured `PAIR_DATA_DIR` explicitly
when a Couch slot injects a repository-scoped data root. It reports per-source
read budget, acquisition duration, snapshot bytes and WAL size changes.

If only publication exceeded that outer budget, a separately approved
`TestSQLiteSnapshotPublishStagedHistory` run (`PAIR_SQLITE_PUBLISH_STAGED=1`)
publishes the retained validated history staging file under the existing
continuous lock. It never opens a native database or takes another snapshot.
Its transform has a 30-minute bound; run it at low priority and give the Go test
runner a longer timeout (for example `-timeout=31m`). Optional
`PAIR_STAGE_ZSTD`/`PAIR_STAGE_GPG` paths permit content-free tool timing wrappers;
an external owner must bound and clean up the complete process group when using
such wrappers. Normal suites skip both native acceptance helpers. Large snapshot
compression policy is tracked separately in #702.


## Drive copies (#678)

`pair archive continuous` completes local backup and writes its local success marker before optional Drive replication. Install/configure rclone independently: Pair never reads or creates its credentials. Add `drive_remote` and `drive_folder` string fields to continuous.json to override `pair-backup` and `backups/pair-history`. Omitted/empty fields use those defaults; invalid remote names or nonrelative/unsafe folders produce a visible Drive-unavailable note after local success. The remote must be a configured named rclone remote and the folder must already exist.

[DriveMirror](../cmd/internal/historyarchive/drive.go) projects every record for the configured local sink, including earlier bundles when no new sources were archived. It releases the local catalog lock before network work. The resolved destination folder ID is pinned, then two flat hash listings bracket uploads. Remote filenames are `<bundle-ID>.tar.zst.gpg` and `<bundle-ID>.json`; the public label is uploaded, the private catalog/Manifest is not. Duplicate names and checksum/size conflicts get no new receipt and are reported; they are never repaired or deleted automatically.

Each source object is staged privately and checked against local SHA256/size before rclone reads it. Upload uses one `copy --transfers 8 --immutable --ignore-existing --checksum` batch: `--ignore-existing` is required because the installed rclone v1.75.2 local-backend conformance showed that `--immutable` alone replaced a conflicting file. Final listing/checksum verification handles a skipped conflict honestly. Prefer returned SHA256, then MD5 computed alongside the verified local stream; when neither usable hash is returned, [hashsum --download](https://rclone.org/commands/rclone_hashsum/) reads the missing-hash objects in one filtered `hashsum --download --files-from-raw` batch and computes SHA256. Each pass uses one resolution, two flat listings, at most one copy batch and at most one downloaded-hash batch, independent of object count. A failed copy batch can still yield receipts for its independently verified complete pairs; incomplete/mismatching pairs stay pending. Output names each bundle's `backend-sha256`, `backend-md5`, or `downloaded-sha256` path and the total downloaded checksums.

The separate upload catalog is `<catalog>.uploads/<destination-SHA256>/<bundle-ID>.json`. Receipts contain identities/hashes/sizes and verification time, never local manifest paths, and never grant local archive authority. Each receipt is at most 8 KiB and there are at most 100,000 per destination; evidence is retained with the copies until an explicit retention policy is introduced. A changed destination uses a fresh receipt namespace. An interrupted upload may leave matching remote objects without a receipt; rerun adopts them after fresh listing/checksum verification. A corrupt/conflicting receipt requires inspection, not overwrite. A receipt records verification at its timestamp; ongoing remote audits are outside this command.

The optional remote phase runs for at most 30 minutes with one subprocess at a time and at most eight internal transfers/checkers, with a 60-second metadata-command cap. Compact candidate and flat listing projections are each bounded to 100,000 entries / 64 MiB; local files retain their existing ciphertext/metadata limits. Exceeding a bound leaves remote work pending while preserving local success. The flat private staging view under the archive store's `.drive-staging/objects/` uses verified hard links to immutable source files on the same filesystem; EXDEV falls back to a verified copy. This removes the local kind/year/month path components without rewriting the whole backlog. Complete pairs are admitted up to a 4 GiB logical-byte cap per pass. Admission also reserves against the complete before-inventory count, charging only missing members so uploads cannot cross the 100,000-entry reader bound. Overflow stays visibly deferred; existing-only verification and receipt recovery still work at that boundary. The 64 MiB wire limit is a fail-closed observation limit for externally supplied metadata, not a remote storage allowance. Every exit sweeps the owned view and checksum list, including errors/cancellation, and the next continuous-lock owner also removes interrupted staging. Cleanup unlinks the view and never removes the original backup path. No recursive inventory or catalog spool is created. A pass reports local counts separately from Drive verified/already-recorded/pending counts; rerun after correcting missing rclone/remote or interruption. Inspect duplicate/mismatch notes before changing remote files.

Deterministic tests use stateful `archivetest.FakeRclone` behind the production runner. Installed-rclone local-backend conformance is opt-in and uses a private HOME/config:

```sh
PAIR_RCLONE_LOCAL_CONFORMANCE=1 go test ./cmd/internal/historyarchive ./cmd/internal/historyarchive/archivetest -run 'TestRclone|TestExecRclone|TestFakeRclone' -count=1
```

For the one operator-run real Drive check, the hub supplies the qualified candidate binary after fake/local conformance passes. Keep the existing continuous.json and run `PAIR_DATA_DIR="$HOME/.local/share/pair" /path/to/candidate/pair archive continuous` (the explicit data root matters when the terminal is itself a Pair slot), retain its local/Drive summary and per-bundle verification paths, then run the same command again: it must report already-recorded pairs with zero new uploads. Confirm bundle and label filenames in Google Drive and retain before/after local bundle counts/bytes. A lock-busy result means no pass ran; wait for the owning pass. Do not print rclone configuration or credentials into evidence. The operator's initial serial-transport candidate passed live acceptance in #678: pass 1 verified 56 pairs with 2 pending; pass 2 verified 7 (2 retries plus 5 new pairs), skipped the 56 existing receipts, and left 0 pending, with 0 downloaded checksums in both passes. Batch transport is separately qualified against the stateful fake and installed-rclone local backend; do not describe the earlier live result as a measured batch throughput result.
