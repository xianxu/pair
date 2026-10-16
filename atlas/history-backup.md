# History backup

`pair archive` is an explicit batch backup path. It archives selected stable
regular files as tar → zstd `-19` → GPG public-key encryption, reads the published
ciphertext back to verify its SHA256 and size, and records verified bundles in a
private local catalog. It preserves every source file. The entrypoint is
[`archivecmd`](../cmd/internal/archivecmd/run.go); the implementation is
[`historyarchive`](../cmd/internal/historyarchive/archive.go).

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
Metadata is bounded and file bytes stream; one zstd process uses one compression
thread per invocation. zstd `-19` has its own working memory, so these limits do
not imply a strict low total RSS. There are no host-wide scans or parallel
compressions within an invocation. The operation is cancellable without a fixed
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

`pair archive continuous [--kind KIND]` is one pass over the continuous kinds
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
  never opens the live file: it copies the `-wal` and then the database into a
  private directory, refuses the copy if either changed meanwhile (a checkpoint
  mid-copy), runs SQLite's `.backup` on the private copy, switches it to
  `journal_mode=DELETE` and passes `PRAGMA integrity_check` before publishing
  it under `$PAIR_DATA_DIR/archive-snapshots/` (outside the catalog directory).
  A skipped database is retried on the next pass.
- **Never deletes;** writes only the archive store, the snapshot root, the sink
  and the catalog. One pass at a time (`continuous.lock`); a completed full pass
  writes `continuous.last`.
- **Schedule:** Couch checks hourly and runs the command at `nice 10` when
  configured and the last pass is over 24 hours old (`couchcmd/archive_schedule.go`),
  logging to `$PAIR_DATA_DIR/archive/continuous.log`. A manual run uses the same
  lock.
- Files over their kind's raw bound are skipped with a note. Database snapshots use the snapshot bound (`rawBundleLimit`, 8 GiB): they compress well (Codex `thread_history_1.sqlite` 1.60 GB to 115 MB), so the 2 GiB ciphertext bound is their real limit, and a snapshot whose compressed bundle exceeds it is skipped with a note (`ErrCiphertextLimit`), not a failed pass (#616).

