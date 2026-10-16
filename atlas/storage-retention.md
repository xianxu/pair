# Pair and Couch storage retention

Pair owns the collector; Couch supplies manifest membership and archived-record
receipts. Agent-native stores and repository files are outside collection.

| Data | Expiry clock | Protection |
| --- | --- | --- |
| Pair drafts, queued/submitted prompts, ledger, current terminal data and recovery state | 365 days since meaningful use | Live users and Couch switcher membership, including parked or unreadable threads |
| Archived Couch records and associated Pair session data | Fresh 365-day grace at archive, extended by later meaningful use | Live users; a restored manifest reference protects again |
| Immutable old parked captures | 365 days per capture | Exact readers and unfinished handoffs; tag reuse and Couch visibility do not renew age |
| Debugging logs | Keep every generation pending verified cloud backup | KeepDiagnosticsUntilBackup; ops history-backup piece 6: archive before evict |

Meaningful use is explicit create/attach/resume/view, or a changed authored-content
write. Unchanged saves, background scans, repaint, diagnostics and distillation do
not extend the session clock. Foreground viewing remains protected until the
viewer closes. Existing session data without use metadata starts a new 365-day
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

Couch singleton adoption is a separate decision from retention migration.
`couch --adopt-store /absolute/store` previews an in-place active-inventory
selection; repeat the request with `--apply <digest>` after reviewing its JSON
report. Add custom sources with repeated `--legacy-store /absolute/path`.
Multiple populated inventories remain **UNMIGRATED** until explicitly resolved;
there is no automatic merge. `--exclude-store /absolute/path` acknowledges a
retired active inventory without deleting its files or removing identity or
retention registrations. It cannot bypass a live or unknown supervisor owner.
Consequently, an excluded but unavailable retention registration can still block
collection. Adoption does not count as acknowledgment of a complete GC inventory.

Stop and upgrade old Couch binaries before adoption, and keep authority and state
on machine-local filesystems. Production singleton authority is found through
the real UID's account home at `.local/share/pair-host/singleton`, independently
of HOME/XDG overrides. The selected Couch store, Pair data root and identity root
remain stable. For isolated retention experiments, use `COUCH_ISOLATED_ROOT`
with all effective roots beneath it and `tests/with-isolated-pair.sh` to clear
ambient Pair overrides. Existing C/N/M and retained conversations are unchanged.

Couch startup only observes slots; it does not enroll stores or schedule
collection. Explicit reference publication enrolls the store before writing.
After migration, running Pair schedules bounded background sweeps after
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
  64 MiB. `KeepDiagnosticsUntilBackup` defaults to true: current and closed
  diagnostics, including quarantines and legacy logs, remain protected from
  count and age eviction until ops project `history-backup`, piece 6, supplies
  archive-before-evict authority. Total disk usage is intentionally unbounded;
  64 MiB bounds new individual generations, not retained history. Rotation
  continues beyond the former three-generation and 64-candidate limits without
  a pruning inventory. Persisted schema-v2 size/count fields remain readable;
  count limits confer no deletion authority while the backup fence is active.
  Unknown writer proof still refuses rotation and growth at the byte cap.
  Preview reports the backup fence and unverified/oversized evidence.
  Exact writer registrations discover external traces; unknown external paths
  are not globbed. Registry publishers serialize under a permanent lock and
  skip unchanged entries while reconfirming directory durability. Per-log state
  and generation metadata serialize under the log lock. Existing deletion
  intents cannot bypass the backup fence; ambiguous ownership retains data.
  `creation.go` publishes a reserved inode without replacement; `append.go`
  reserves a bounded batch with its length/hash in place, verifies complete
  appended bytes, then publishes one checkpoint. Legacy payload intents recover
  before migration. Uncertifiable partial batches are preserved whole as
  explicitly unverified generations; writer proof permits a fresh current log.
  Wrapper diagnostics coalesce at 64 KiB or dirty five-second ticks, bounded to
  256 KiB outstanding including flight. Ordinary appends avoid per-record fsync;
  process crashes can lose accepted pending buffers. Host power-loss durability
  is not promised. Rotation syncs its payload/metadata boundaries durably.
- `gcruntime/` composes Couch, process evidence, binding cleanup and diagnostic
  collection. `gccmd/` exposes the command; `storagegc/schedule.go` owns bounded
  scheduling and its durable completion/cursor state. Diagnostic path discovery
  bypasses session clock and liveness evaluation; collection supplies its own
  exact writer and generation proof. The maintenance context propagates into
  process inspections, their subprocess deadlines and recovery tree walks.
  Protocol-aware runtime rows skip irrelevant data-root canonicalization while
  retaining both process-generation fences; legacy missing roots still refuse.
  This proof also feeds parked-capture expiry. End-to-end maintenance tests
  collect expired raw/events/metadata together and preserve exact readers,
  unfinished handoffs and the existing CaptureRetentionPeriod boundary. That
  one constant remains 365 days; diagnostic policy never selects raw
  recovery captures.

Malformed metadata, ambiguous ownership, unknown liveness, symlinks, incomplete
inventories and uncertain filesystem effects retain data. Exceptionally large
single-owner journals exceeding 1 MiB are blocked before detachment. Manual file
moves/restores require stopping managed users and collection; use Couch's typed
restore path when restoring archived records.

Verification lives beside these packages, in `nvim/retention_test.lua`, and in
`tests/retention-test.sh` and `tests/adapt-schema-test.sh`.

## Resource monitor snapshots (#587)

Couch's resource sampler owns one best-effort namespace-local aggregate history,
separate from diagnosticlog's 365-day retention. ResourceHistoryStore writes
resources/history.json (up to 4 MiB), with one fixed .history-stage replacement.
Current file plus stage are bounded at 8 MiB. Existing safe-path, regular-file
and no-follow checks protect reads and writes; oversized or invalid history is
reported as lost continuity and a fresh bounded in-memory ring starts.

History retains at most 24 hours, 1440 frames and 4 MiB of aggregate slot usage,
orphan/unattributed summaries and coverage/cost metadata. The memory ring keeps
advancing when a write fails. A crash may lose its tail; history failures never
gate alerts. Ownership, rate baselines, episodes and delivery outcomes are only
in memory. Restart re-arms advisory alerts, so a repeat is possible.

While Couch is stopped the file remains bounded; the next load/write prunes
expired frames. No cleanup daemon or checkpoint/intent files are created.
Sources remain classified as namespace-owned non-Pair artifacts in artifactpath.
See [resource monitoring](couch.md#resource-monitoring-587).
