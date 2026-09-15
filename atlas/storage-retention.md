# Pair and Couch storage retention

Pair owns the collector; Couch supplies manifest membership and archived-record
receipts. Agent-native stores and repository files are outside collection.

| Data | Expiry clock | Protection |
| --- | --- | --- |
| Pair drafts, queued/submitted prompts, ledger, current terminal data and recovery state | 60 days since meaningful use | Live users and Couch switcher membership, including parked or unreadable threads |
| Archived Couch records and associated Pair session data | Fresh 60-day grace at archive, extended by later meaningful use | Live users; a restored manifest reference protects again |
| Immutable old parked captures | 7 days per capture | Exact readers and unfinished handoffs; tag reuse and Couch visibility do not renew age |
| Debugging logs | 7 days per generation | Writer coordination and complete legacy-process evidence |

Meaningful use is explicit create/attach/resume/view, or a changed authored-content
write. Unchanged saves, background scans, repaint, diagnostics and distillation do
not extend the session clock. Foreground viewing remains protected until the
viewer closes. Existing session data without use metadata starts a new 60-day
grace; old atime/mtime does not prove session inactivity. Legacy archived Couch
records missing a grace sidecar also receive a full grace on apply; malformed
sidecars remain blocked.

`pair gc` previews counts, logical bytes and retention reasons; `--json` includes
exact paths and decisions. Preview never initializes clocks or recovers journals.
`pair gc --apply` initializes missing clocks and collects eligible data once the
Couch-store inventory has been acknowledged. `--root PATH` selects explicit
storage, including a scoped Pair directory.

Migration is explicit because old custom Couch namespaces cannot be discovered
reliably. Register each existing namespace with `pair gc --register-store PATH`.
Acknowledge the complete list with `pair gc --complete-migration --store PATH`,
repeating `--store` for every registered namespace. An intentionally empty list
uses `--complete-migration` alone. New managed Couch stores register before
publishing references. An unavailable registered store blocks session collection.

After migration, running Pair/Couch schedules bounded background sweeps after
readiness, at most once per completed daily pass. Work resumes from a cursor;
automatic pages visit at most 100 owners, including retained owners, with a
cooperative two-second budget and nonblocking root-lock acquisition. Contention
or budget expiry preserves the last durable cursor and retries later. Discovery
is metadata-only and capped at 100,000 entries; incomplete inventory retains
payloads, and very large inventories may require explicit collection. Filesystem
calls can exceed the cooperative budget. Explicit apply scans past retained
owners while limiting deletions, so repeated commands do not starve later owners.
Failures preserve evidence and leave data retained. Expiry means collection on a
successful eligible sweep, not deletion at an exact wall-clock instant. No global
disk ceiling is promised: active session history remains retained.

The implementation map:

- `artifactpath/gc.go` owns exact file membership and independent retention
  classes. Shared scope metadata and permanent diagnostic lock inodes are excluded.
- `storagegc/policy.go` and `capture.go` decide from injected clocks and evidence.
  `coordinator.go`, `use.go`, `lease.go`, `start.go` and `runtimes.go` record actual
  process identity, pending content effects and handoffs under stable root locking.
  Confirmed dead pre-spawn reservations are reaped; uncertain spawned reservations
  require explicit `pair retention resolve-start` evidence and acknowledgment.
  Unpublished metadata stages centrally under `.retention/pending`; recovery
  removes exact unpublished residues under coordination.
- The internal `pair retention` command serves Lua editors. A durable intent must
  precede authored-content effects; uncertain completion remains protected.
- `couchcore/retention.go` journals archive grace with membership removal.
  `archive_gc.go` owns exact cross-store detach receipts. Lock order is Pair root
  before Couch store, or Pair root before diagnostic log. Nested maintenance
  acquisition is nonblocking; cancellation propagates through journal entries.
  Coordinated Couch publications use one reserved `.thread-store-publication`
  stage reclaimed under the store lock. Unlocked continuation materialization
  uses a separate publisher; generic temporary names are never swept.
- `storagegc/collector.go` inventories metadata; `transaction.go` journals exact
  quarantine identities before rename. The journal precedes its unique quarantine
  directory, so interrupted creation remains replayable. Durable references and
  owner metadata discover scoped owners even when payload directories are absent.
  `transaction_model.go` owns the pure
  prepared → detached → finalized transition reducer. Empty eligible activity
  records use the same retirement journal. Recovery never deletes a replacement
  source after detachment. Raw captures, event sidecars and creation metadata detach together. New captures
  use a recorded UTC creation clock with checked payload identities; legacy
  captures use the latest plausible filename time or file mtime conservatively.
- `diagnosticlog/` manages wrapper events, adaptation records and optional
  Pair/Couch traces. Current paths stay stable. Generations rotate at 24 hours or
  64 MiB; segmentation preserves young data. Closed-generation expiry uses last
  write, allowing up to one extra day of record-age slack. Logging is optional
  and may skip under contention. External traces are discovered only through
  exact writer registrations; unknown old external paths are not globbed.
  Registry enumeration reports traversal completion separately from filtered
  entries. Registry publishers serialize under a permanent registry lock;
  per-log state and generation metadata serialize under the log lock. Reserved
  unpublished stages are reclaimed only under their owning lock. Deletion
  intents replay through missing payloads, metadata and removed ancestor
  directories, while replacement identities and unsafe ancestors still refuse.
- `gcruntime/` composes Couch, process evidence, binding cleanup and diagnostic
  collection. `gccmd/` exposes the command; `storagegc/schedule.go` owns bounded
  scheduling and its durable completion/cursor state. Diagnostic path discovery
  bypasses session clock and liveness evaluation; collection supplies its own
  exact writer and generation proof. The maintenance context propagates into
  process inspections, their subprocess deadlines and recovery tree walks.

Malformed metadata, ambiguous ownership, unknown liveness, symlinks, incomplete
inventories and uncertain filesystem effects retain data. Exceptionally large
single-owner journals exceeding 1 MiB are blocked before detachment. Manual file
moves/restores require stopping managed users and collection; use Couch's typed
restore path when restoring archived records.

Verification lives beside these packages, in `nvim/retention_test.lua`, and in
`tests/retention-test.sh` and `tests/adapt-schema-test.sh`.
