---
collaboration-mode: discussion
---

# Live SQLite backup freshness implementation plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy). Follow @superpowers-test-driven-development; this is one atomic change with one close review, not separate milestone boundaries.

**Goal:** Make coherent Codex database backups progress during writes, with bounded reader lifetime and honest per-database freshness output.

**Architecture:** SQLite opens the source read-only and pins a read transaction before its online backup. A short-lived subprocess owns that transaction; a separate subprocess validates only the private result before the existing staging, encryption and verified catalog publication. A typed pass report combines acquisition outcomes with existing catalog timestamps without changing any persisted format.

**Tech Stack:** Go, the existing sqlite3 CLI, Python sqlite3 for disposable conformance fixtures, existing historyarchive storage/catalog/codec seams.

## Decision and evidence

The old collector requires unchanged DB/WAL metadata across two raw copies. A sustained writer can invalidate every pass. Unpinned incremental online backup can itself restart when the writer changes its source. A pre-existing read transaction fixes the view for the backup's duration. SQLite owns the consistency protocol; we do not infer coherence by comparing live file mtimes.

Corrected private 8.5MB probe: pinned backup 1.0245s with 125 concurrent commits; ordinary backup timed out after 5.0127s with 647 commits. Integrity and transaction invariants held in the pinned output. These are mechanism measurements, not production latency guarantees. The fixture's earlier SIGBUS was raw descriptor closes in a process owning SQLite locks; all byte inspection now happens outside that process.

Hub approval c8f5b589-027a-4270-a6bc-8d66c03a6ef5 replaces the never-open contract: native DB and WAL contents remain read-only, but SQLite may create/update SHM read marks. Prefer .backup to VACUUM INTO, which may change implicit rowids. No catalog/manifest codecs or dependency on unlanded #689.

Sources: [online backup](https://www.sqlite.org/backup.html), [backup API](https://www.sqlite.org/c3ref/backup_finish.html), [retention of an existing source transaction in backup.c](https://sqlite.org/matrix/ev/src/backup.html), [WAL readers/checkpoints](https://www.sqlite.org/wal.html), [VACUUM rowids](https://www.sqlite.org/lang_vacuum.html), [raw-close locking hazard](https://www.sqlite.org/howtocorrupt.html#posix_advisory_locks_canceled_by_a_separate_thread_doing_close).

## Core concepts

### Pure entities

| Name | Lives in | Status |
|---|---|---|
| SnapshotResult / SnapshotStatus | cmd/internal/historyarchive/sqlite_snapshot_model.go | new |
| SnapshotObservation | cmd/internal/historyarchive/sqlite_snapshot.go | modified |
| SnapshotStable | cmd/internal/historyarchive/sqlite_snapshot.go | deleted |
| DatabaseBackupStatus / DatabaseBackupSummary | cmd/internal/historyarchive/continuous_snapshot_model.go | new |
| ArchivedIndex | cmd/internal/historyarchive/continuous_index.go | modified |

- **SnapshotResult** replaces the bool/free-form-note protocol with Ready, Deferred or Rejected and bounded detail. Ready means a coherent validated staging file, not a verified encrypted backup. Deferred means a confirmed context/alarm deadline; Rejected includes invalid input, integrity/size refusal and ambiguous generic CLI failure. The CLI does not expose a reliable structured busy result: exit 1 remains failed/unknown, even if its diagnostic mentions a lock. Infrastructure errors and caller cancellation remain errors. No status is inferred from message text. Unit tests use literal values.
- **SnapshotObservation** retains preflight size/WAL-presence facts for admission; size/mtime equality stops being consistency authority. Delete SnapshotStable and its old behavioral dependency rather than keeping a second consistency algorithm.
- **DatabaseBackupSummary** combines one acquisition result, dedup/publication result, current observation time and optional last verified bundle time. It projects Archived, Unchanged, Deferred or Failed. A source checked unchanged today is distinguishable from one not successfully observed since an old bundle. Tests cover unknown timestamp, clock skew, publication failure and previous verified bundles.
- **ArchivedIndex** remains the one per-pass catalog projection. Add latest verified time keyed by the existing exact source identity/path, populated alongside existing hashes/stamps from each CatalogRecord. Preserve its manifest-only Add use for current tests/callers; add AddRecord for timestamp-bearing records. No second catalog scan per database. Future report fields extend this runtime projection, not stored records.

### Integration points

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| SQLiteSnapshotter | cmd/internal/historyarchive/sqlite_snapshot.go | modified | source paths, scratch and staging filesystem |
| SQLiteRunner / ExecSQLiteRunner | cmd/internal/historyarchive/sqlite_snapshot_exec.go | new | sqlite3 child process, bounded streams and cancellation |
| Snapshotter | cmd/internal/historyarchive/continuous.go | modified | typed acquisition outcome |
| ContinuousPass | cmd/internal/historyarchive/continuous.go | modified | discovery, existing archive publication and catalog scan |
| runContinuous | cmd/internal/archivecmd/continuous.go | modified | existing command output |
| fakeSQLiteRunner | cmd/internal/historyarchive/sqlite_snapshot_test.go | new | disposable stateful SQLite protocol model |

- Runner injection belongs to SQLiteSnapshotter. A request carries typed source-read or private-validation purpose, database path, private working directory and script; result carries bounded output/overflow plus exit/context error. Keep command construction in one runner. The fake models source generations, pinned generation, writer advancement, partial destination, lock contention, cancellation and integrity outcome across calls. It does not merely return a predetermined successful call.
- Real conformance runs the same collector with sqlite3 and an independently owned Python writer. Its controls use pipe/barrier acknowledgments, not sleeps to infer readiness. All fixture roots are disposable; no normal suite reads native Codex directories.
- ContinuousPass retains its one continuous.lock, one active database at a time, existing Archiver/Codec/Sink seams and exact staging SourceIdentity. Update fixture Snapshotter implementations to the typed return. No new scheduler, lock, persistent retry queue or global executor.

## Acquisition and publication protocol

1. Validate relative selection, canonical source root and regular source DB; refuse symlink DB/WAL/SHM entries and non-regular sidecars. Capture source DB identity before opening; recheck exact DB/root identity after backup. Ordinary DB/WAL appends and checkpoints are allowed. A replaced/disappeared DB is rejected rather than published under the earlier identity. Source directory ownership is trusted; this is not a sandbox against a malicious actor doing undetectable ABA directory swaps.
2. Apply the existing 8GiB raw DB+WAL admission before starting sqlite3. Create one mode-0700 work directory under configured scratch, outside source/staging; its fixed output basename is snapshot.sqlite. Use a fixed basename to avoid interpreting caller paths inside a dot command. Pass the absolute source as an argv element and use the work directory as child cwd. Ignore sqlite startup files with -init /dev/null, and pass -bail -batch -readonly -nofollow.
3. Run this fixed stdin script with a **5s total source-process deadline** and **250ms SQLite busy timeout**:

   ```sql
   .timeout 250
   BEGIN;
   PRAGMA schema_version;
   .backup main snapshot.sqlite
   ROLLBACK;
   ```

   BEGIN plus the schema read pins the transaction before .backup. This script does not inspect application tables or change source pragmas. Busy timeout is per lock wait; the process context is the total bound. Never rerun backup inside the pass after busy/timeout. Reap the direct child before returning; existing-style WaitDelay is at most 1s. Do not claim a cancelled/uncertain child produced a publishable snapshot.
4. After the source process exits successfully and identity checks pass, require a regular private output within the 8GiB raw bound. Run only on that private output: PRAGMA journal_mode=DELETE; PRAGMA integrity_check;. Keep a separate bounded validation context (existing 10-minute ceiling), which never pins the native database. Require exact expected output and no stdout overflow; retain bounded stderr for a failure notice. Source and validation streams remain separate. Bound stdout to 64KiB and stderr to 8KiB, draining/discarding excess; overflow can never become a successful integrity result.
5. Keep sameContent dedup and staging rename semantics. Identical content keeps staging mtime. Old staged data is never offered for this pass when current acquisition failed. Staging replacement happens only after successful validation; any partial output stays private. Caller cancellation propagates and prevents publication. Existing cryptographic publication/catalog receipts remain the only verified backup evidence.

The process call stack owns acquisition/validation/publish ordering; no durable workflow state or background job outlives Snapshot. Most likely interrupt: cancellation after a partial backup but before validation. It discards private output and keeps prior verified bundles; there is no automatic retry and no inferred completion. Cancellation during existing encrypted publication retains that publisher's existing uncertain-outcome/reconciliation contract.

## Visible state and recovery

Actions: wait for the next scheduled pass; rerun the continuous command explicitly; repair the named configuration/storage/tool problem and rerun.

| Internal state / failure mode | Visible state | Recovery action | Possible loss |
|---|---|---|---|
| Validated snapshot, new encrypted bundle and verified catalog record | Archived; last verified timestamp/age | Wait for next scheduled pass | Writes after the pinned snapshot are absent from this version |
| Validated snapshot matches a cataloged content hash | Unchanged; checked this pass, last stored bundle timestamp/age | Wait for next scheduled pass | No new version; an old bundle timestamp alone is not failure |
| Confirmed source deadline; prior verified bundle exists or none exists | Deferred with reason; last verified age or never | Explicitly rerun once when ready | New writes remain unbacked-up; old verified versions remain |
| Size refusal, invalid/replaced source, tool or integrity failure | Failed/skipped with bounded reason; prior age or never | Repair the named cause and rerun | Latest state unavailable; prior bundles preserved |
| Caller cancels or process dies with partial private output | Incomplete pass, no claim of freshness | Rerun the command | Partial private candidate discarded/swept; prior versions retained |
| Encryption/catalog publication fails after acquisition | Failed publication, prior verified age retained | Repair the named publication cause and rerun | Private coherent acquisition is not a verified bundle |

Do not change continuous.last: it still records full-pass completion, not freshness of every source. Print one deterministic per-database line in the existing continuous command output/log, also on partial failure. Show only existing public database selection names, status, times and bounded tool failure detail; no application rows or credential content. Unchanged/Archived both say current data was checked. A skipped source never says checked or archived. A partial failure still lets unrelated databases run unless the caller cancelled.

Covered: sustained writes, bounded source pin, partial acquisition/publication, honest older backups, unknown SQL schema/rowids. Not covered: automatic immediate retries, background freshness alerts, new history UI, codec evolution, scheduler cadence changes, or a guarantee that every size/workload meets the five-second read budget. A slow real 2GB history backup is a measured bound failure to discuss with the hub, not grounds to extend the pin silently.

## Operating envelope, costs and cleanup

- Workload is a serial batch, off the UI path. One pass owns continuous.lock; zero extra worker pools or retry timers. Native read pin: 5s deadline, plus at most 1s pipe/reap allowance. Private validation: separate 10-minute ceiling. Context cancellation shortens either.
- Native sizes measured by metadata only: logs DB84MB + WAL108MB; state DB347MB + WAL4MB; history DB1.99GB + WAL4MB. The private 8.5MB result does not prove the 2GB deadline. Live check must measure all three.
- Preserve 8GiB raw admission and accepted-output limits, 2GiB ciphertext limit and existing metadata limits. Scratch contains one candidate at a time; latest staging and prior encrypted versions remain under existing policy. These are admission/output limits, **not a filesystem quota**: transient candidate growth and native WAL growth while the read mark is held are bounded by reader lifetime, not an exact byte ceiling. Disk-full must fail safely with the old staged/verified bundle untouched. Do not introduce a new shell resource-limiter in this fix.
- Memory: streamed SQLite IO and existing bounded comparison buffers; only bounded tool text retained. No DB/WAL whole-file Go buffer. Encryption/compression streaming unchanged. No network added by snapshot acquisition; existing optional Drive behavior unchanged.
- For D databases of B bytes and C catalog records: one O(C) catalog scan per pass; at most one acquisition and one private validation process per admitted DB, O(D) launches and O(sum B) SQLite/compare work. No raw recopy and no application-level backup restart loop. Count N=3 and 2N=6 through production ContinuousPass with the stateful runner, checking attempts/processes/catalog scans and refusing >2D calls or per-DB catalog rescan. A single rejected DB does not starve the rest.
- Scratch creator and last user: SQLiteSnapshotter, cleaned by defer on all ordinary exits; existing Sweep under continuous.lock removes sqlite-* residue after crash. Staging creator: successful validated candidate; last user: archive/dedup; replaced by next validated changed snapshot. Catalog and encrypted bundles retain existing lifetime/cost per successful version; no new durable artifact family. Probe fixtures are owned temporary directories, removed after retained evidence is recorded.

## Boundary strategies and mechanical guards

| Function / seam | Adversarial strategy | Mechanical guard and regression |
|---|---|---|
| SQLiteSnapshotter.Snapshot | Writer commits between every old copy boundary | Pinned source generation in stateful fake plus real writer invariant; old algorithm negative control |
| SQLiteSnapshotter.Snapshot | Symlink, missing, non-regular or replaced source; unusual path characters | Regular/nofollow and pre/post identity checks; absolute argv, fixed dot-command filename; path/replacement tests |
| ExecSQLiteRunner.Run | Held lock, hung child, partial stdout, stderr flood, cancellation | Context deadline, bounded separate streams, reaping and no retry; executable-fixture tests |
| Private output validation | Partial/corrupt/over-limit candidate; truncated apparent ok | Exact complete validation result plus regular/size check; old staging sentinel remains unchanged |
| ContinuousPass.archiveRoot | Failed acquisition leaves old staged candidate | Only typed Ready candidates rediscovered; failure/cancel tests exclude old stage |
| ArchivedIndex.AddRecord / DatabaseBackupSummary | Old, absent, newer, future timestamps; identical content | Pure exact-source/path projection; literal table tests and catalog dedup integration |
| runContinuous | Partial report, skipped source, missing timestamp | CLI assertions name every database/status and age, never claim last-run marker is per-source freshness |

## Implementation tasks

### Task 1: Acquire a pinned snapshot through one bounded seam

Files: modify sqlite_snapshot.go/sqlite_snapshot_test.go; create sqlite_snapshot_model.go/sqlite_snapshot_model_test.go and sqlite_snapshot_exec.go/sqlite_snapshot_exec_test.go under cmd/internal/historyarchive/. Modify cmd/internal/artifactpath/manifest.go for the new production-source classifications.

- [ ] Add a portable stateful writer fixture and negative control that advances the source between old copy boundaries, proving changed-during-copy starvation. Retain the failing run before replacing the implementation.
- [ ] Add failing policy and stateful-runner tests using the function-level strategies above. Run go test ./cmd/internal/historyarchive -run 'TestSQLiteSnapshot|TestExecSQLite' -count=1; record the expected failures.
- [ ] Implement typed acquisition, fixed script, exact identity checks, separate bounded runner streams/deadlines and private validation. Delete raw copy stability authority. Retain copyRegularFile only where existing unrelated test fixtures need it; remove unused production hooks.
- [ ] Implement the runner/identity guards against the named adversarial classes, using readiness-controlled subprocess fixtures rather than external startup timing assertions.
- [ ] Run the same collector contract against a real disposable SQLite writer; keep SQLite-owning writer and raw inspectors in separate processes, with barriers and counters proving concurrent commits. Record the conformance output.
- [ ] Exercise rejection/corruption/size/disk-error paths and crash sweep. Run targeted snapshot tests and go test ./cmd/internal/artifactpath ./cmd/internal/testpair; commit the verified mechanism with issue/agent trailers.

### Task 2: Project freshness through the existing continuous pass

Files: modify cmd/internal/historyarchive/continuous.go, continuous_index.go, continuous_external_test.go, continuous_select_test.go, bounds_internal_test.go; create continuous_snapshot_model.go/continuous_snapshot_model_test.go; modify cmd/internal/archivecmd/continuous.go and continuous_test.go. Classify the new production file in artifactpath/manifest.go.

- [ ] Write failing projection/index and production-pass tests using their named boundary strategies; record the red result before implementing.
- [ ] Implement Snapshotter typed return and runtime database summaries; populate latest verification times during the existing single catalog scan and successful record adoption. Update all fixture Snapshotter implementations. Preserve dedup/source identity and other source-kind output.
- [ ] Render one deterministic freshness line per DB in runContinuous and test its strategy through the CLI entry point.
- [ ] Run counted production-path N=3/6 cases: one catalog traversal, ≤D acquisitions, ≤D private validations, no retry, continued progress after one deferred DB. Run go test ./cmd/internal/historyarchive ./cmd/internal/archivecmd ./cmd/internal/artifactpath ./cmd/internal/testpair -count=1 and commit.

### Task 3: Document, verify live with the hub, close and land

Files: modify atlas/history-backup.md and its existing atlas/index.md entry if the map description changes; update this plan and the issue Log. Do not edit hub-owned project files.

- [ ] Document SHM/read-only distinction, five-second pin, deferred behavior, per-source freshness, scratch/output admission limits versus transient disk use, unchanged bundle age and existing scheduler meaning.
- [ ] Ask the hub to coordinate the bounded live check, with no pair:0 staging unless explicitly requested. Use the tested collector from this slot, existing configured encryption/sink/catalog, and one codex-database pass. No credential inspection, operator-session changes or writer stoppage. Record per-DB acquisition durations, output sizes, verified catalog records and writer continuity; no raw application rows. Any timeout is an honest failed acceptance and a hub decision, not a hot retry or silent deadline increase.
- [ ] Confirm fresh verified bundles for logs_2.sqlite, state_5.sqlite and thread_history_1.sqlite. If a database is genuinely unchanged, show the coherent observation and existing verified content; do not fabricate a newer version. Have the hub decide any mismatch with the issue's fresh-bundle acceptance wording.
- [ ] Push through SDLC and run sdlc local-check run --name suite on the final pushed head. Registered flakes follow current tl-worker rules; unknown failure blocks. Close once with exact targeted, recorded-suite and live evidence; let sdlc own the fresh-eyes review.
- [ ] On SHIP run sdlc merge --yes; its landing turn integrates main and owns the authoritative integration checks. Report Git-printed merge commit and PR to hub; verify submitted receipt and stop.

## Architecture accounting

ARCH-DRY: one SQLite runner, existing lock/publication/dedup/sweep and one catalog pass. ARCH-PURE: typed result/index/report policy separated from IO; fake and real DB work explicitly integration. ARCH-PURPOSE: prove active-writer progress and all three native DBs, not just private-copy coherence. ARCH-MOCK: stateful SQLite fake and disposable actual-SQLite contract checks. ARCH-CONSTRAINTS: bounded native reader, one attempt, serial batch, counted growth and honest transient disk limits. ARCH-SECURE: read-only/no-follow native opening, ignored startup file, fixed script with argv paths, private scratch, no row/credential output. ARCH-STATE: lexical acquisition/validation/publication ownership and typed outcomes; cancellation cannot promote a partial candidate. ARCH-FUNERAL: existing sweep/staging replacement and existing bundle retention, no persistent retries. ARCH-EVOLVE: no persisted format changes; opaque .backup preserves unknown SQL schema, values and rowids.

## Revisions

### 2026-10-19 — PQ-1/PQ-2/PQ-3 and hub-directed Go alarm launcher

Reason: collector death invalidates a parent-only deadline; generic SQLite exit codes do not distinguish busy. This revision supersedes the direct sqlite3 launch and unconditional scratch sweep described above. Hub c6bca0a7-27cc-4445-a53a-a28b509b2102 explicitly directs the same-binary Go launcher, with no Perl/runtime dependency. Ambiguous exit classification and condensed test tasks address PQ-2/PQ-3.

Additional INTEGRATION concepts:

| Name | Lives in | Status | Wraps |
|---|---|---|---|
| execalarm.Run / armTimer | cmd/internal/execalarm/run.go, timer_darwin.go, timer_linux.go | new | kernel ITIMER_REAL and syscall.Exec |
| internal exec-alarm routing | cmd/pair-go/main.go | modified | early private CLI dispatch |
| snapshot work lease | cmd/internal/historyarchive/sqlite_snapshot.go | new | inherited flock descriptor and guarded sweep |

- The existing pair binary gets a private early route: internal exec-alarm SECONDS -- PROGRAM ARGS. Parse a positive bounded integer (1–600), resolve PROGRAM before arming the timer, arm ITIMER_REAL, then syscall.Exec. This route bypasses launch/session setup. The source request always supplies 5 seconds; private validation may use its existing bounded 600-second ceiling. The caller's context remains the earlier cancellation path. No new installed executable or interpreted dependency.
- Linux uses unix.Setitimer. Darwin uses syscall.Syscall(SYS_SETITIMER, ITIMER_REAL=0, pointer to unix.Itimerval, nil) with a short documented unsafe bridge and runtime.KeepAlive. A disposable Go probe on this Darwin arm64 machine armed a one-second timer and execed sleep; it exited with signal: alarm clock. If the syscall fails on any supported platform, refuse before opening the DB and report the cause; never fall back to parent-only supervision. Unsupported platforms fail clearly.
- Kernel ownership is the key resource: the timer belongs to the process that becomes sqlite3 and survives exec/collector death. No watchdog process is needed. SQLite must not clear SIGALRM/the timer; actual CLI owner-death conformance qualifies that dependency. The deadline begins before exec (and may reject slow startup), never after the source is opened. Acquisition remains one attempt.
- Before spawning either child, Snapshotter creates/locks a regular no-follow work/lease file, holds it through its own cleanup, and passes the open description as fd 3 via ExtraFiles. The helper preserves that fd across exec. Sweep acquires the same per-work flock nonblocking before deletion; held/uncertain leases are skipped, never deleted. Parent death releases its reference but not the child's. Child death releases the last reference; a later sweep collects it. Legacy work without a lease remains safe to collect because old snapshot children opened only private sources; lease-present but unreadable/malformed work is not removed speculatively. This changes scratch coordination only, not archive/catalog formats.
- Process and lease ownership end on exec failure, normal child exit, context cancellation or kernel alarm. An orphan may survive until its own alarm but cannot pin the source past that kernel budget; no later pass deletes its live scratch. Reaping/lease cleanup tests use controlled process ordering, not host-speed ceilings.

Additional function-level strategies:

| Function / seam | Adversarial strategy | Mechanical guard and regression |
|---|---|---|
| execalarm.Run / armTimer | Invalid budget/argv, exec failure, killed collector | Parse before effects, kernel timer before exec; owned-process probe verifies SIGALRM and lease release after collector death |
| main.runWithLegacyRuntime | Internal helper mistaken for public pair launch | Exact early internal route; test asserts no launch/runtime-root setup |
| SQLiteSnapshotter.Sweep | Collector dead but native reader alive | Nonblocking same-description work lease; controlled owner-death fixture proves sweep skips until reader exits |
| SnapshotResult classification | CLI returns generic exit 1 with misleading lock text | Typed context/alarm evidence only; real CLI failure and fake agree on unknown failure |

Implementation inserts the helper/lease red-green tests before Task 1's collector wiring, with targeted go test ./cmd/internal/execalarm ./cmd/pair-go ./cmd/internal/historyarchive. Register every new production source in the artifactpath inventory and regenerate runtime-bundle mirrors through the repository's existing generator if required. No production implementation starts before the revised plan gate passes.

### 2026-10-19 — implementation qualification

The helper implementation lives in execalarm.go/run_unix.go/run_other.go and timer_darwin.go/timer_linux.go. ITIMER_REAL is periodic at the same budget so an expiry consumed by Go before exec cannot leave an unbounded native child; thread-local unblocking and a caught signal disposition also qualify inherited mask/ignore cases. The collector's context still supplies total elapsed cancellation while alive. Actual SQLite owner-death and held-lease cleanup conformance passed.

Cost wording above refers to one freshness index projection: existing OpenCatalog validates shards, and existing archive publication performs its own catalog verification. This change adds no per-database catalog projection. Counted 3/6-database tests defend the new acquisition/validation work and prove failed acquisitions cannot reuse staging. Native acceptance is an opt-in test of the production ContinuousPass with the configured local sink/catalog, a separately built pair-go launcher, no Drive phase and a five-minute outer cancellation bound; it is skipped in ordinary suites. The hub coordinates when to run it.

### 2026-10-19 — hub-approved size-scaled read budget after live measurement

The first actual native pass acquired logs in 0.543s and state in 1.816s; history stopped after 5.111393s with 985,505,792 partial bytes and did not publish. WAL file size increased by zero on all three. Logs/state produced fresh verified bundles; goals/queue were coherent unchanged. Memories failed at the schema-read step with unable-to-open, which must be diagnosed before landing. The pass completed in 98.76s; no automatic retry occurred.

Hub cd188ea1-2521-45f2-ace9-8c511f0fac6d explicitly replaces all fixed five-second acquisition limits above with **ceil(clamp(2 × size / 190 MB/s, 5s, 30s))**, decimal MB. Size is the preflight DB+WAL sum, which conservatively budgets WAL folding. Source admission remains 8GiB; calculate clamping before integer arithmetic can overflow. Add the PURE function SnapshotReadSeconds(rawBytes int64) in sqlite_snapshot_model.go, with literal boundary/monotonicity tests and an adversarial strategy: arbitrary sizes → bounded nondecreasing seconds, exact measured 1.99GB example → 21s. Snapshotter passes this budget in SQLiteRequest.ReadSeconds; ExecSQLiteRunner validates 5..30 and applies the same value to both Go context and kernel launcher. Validation's private 600s bound is unchanged. Existing fake/counted and actual alarm tests continue to qualify the mechanism. No new executor, retry, state or persisted format.

The hub authorizes exactly one further native pass after the revision and memories diagnosis. It again uses the normal local sink/catalog, no Drive, five-minute outer cancellation and one size-scaled pin per source. A timeout remains Deferred and must be reported; it never renews its budget.
