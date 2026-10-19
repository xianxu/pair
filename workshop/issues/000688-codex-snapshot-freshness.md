---
id: 000688
status: working
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '979ecc1c7c2f5a67b440f0b77ac4725c0cda1dfb' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T01:53:02-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "ddb4876c", done: "16f91cb5"}
---

# Codex database snapshots can starve under continuous writes

## Problem

During #678 operator live verification on 2026-10-18, pass 1 skipped one Codex database and pass 2 skipped logs_2.sqlite, state_5.sqlite and thread_history_1.sqlite with 'changed during copy'. The current collector copies then compares the live database/WAL before producing a private backup; continuous Codex writes can repeatedly invalidate that observation.

This is not a claim that all database backup always fails. Read-only local catalog inspection found 10 codex-database records: logs_2.sqlite and state_5.sqlite have fresh bundles verified 2026-10-19T01:53:42Z and 01:54:02Z (18:53/18:54 PDT). The latest thread_history_1.sqlite bundle is still 2026-10-16T18:22:42Z (11:22 PDT), roughly 55 hours old at the investigation, 114658949 encrypted bytes. #616 fixed compressed-size admission and demonstrated one successful thread-history bundle; it did not change snapshot consistency or guarantee freshness under ongoing writes.

## Spec

Investigate snapshot freshness per database under sustained writes, and provide a coherent bounded collection route that can make progress on active SQLite databases while preserving original files. Evaluate SQLite online backup or VACUUM INTO against the current private-copy safety contract; do not weaken integrity/stability checks or infer liveness from one successful sample. Report per-database last successful backup age and classify skips honestly.

This follow-up is independent of Drive replication #678 and does not block its landing. No live database or snapshot code was changed during this filing.

## Done when

- A portable stateful SQLite writer fixture demonstrates the current starvation and a coherent replacement snapshot that makes progress under sustained writes, preserving source bytes/ownership semantics.
- A bounded live check records fresh verified bundles for logs_2.sqlite, state_5.sqlite and thread_history_1.sqlite without disabling the writer or changing operator sessions.
- Snapshot limits, cancellation, storage cost and visible failure/freshness semantics are documented; tests preserve consistency and source isolation.

## Plan

Single-pass full flow: [durable implementation plan](../plans/000688-codex-snapshot-freshness-plan.md). Send the complete plan to the hub before change-code.

- [ ] Replace copy-then-compare with a bounded pinned read-only SQLite backup, verified private output, stateful fake and active-writer conformance.
- [ ] Report typed per-database acquisition/publication state and last verified bundle age using the existing one-pass catalog projection.
- [ ] Coordinate the three-database live check with the hub, complete the recorded suite and close review, then land on SHIP.

## Log

- Filed from #678 hub direction cb95f294-af33-4a95-99d0-b583012e94da. Evidence is read-only catalog records and operator-reported changed-during-copy skips; the precise frequency/root cause remains to be measured. Existing #616 is a size-limit fix, not the same issue.
- 2026-10-19 initial investigation: current SQLiteSnapshotter requires unchanged DB/WAL size+mtime across WAL-first/DB-second raw copies, then runs SQLite only on the private copy. The native source is never opened by SQLite. Busy skips return nil errors and can still advance continuous.last; catalog VerifiedAt means last verified bundle, not last successful unchanged snapshot observation. Keep these meanings distinct in the plan. A read-only pipeline exploration found exact source identity/staging/dedup and lifetime seams in sqlite_snapshot.go/continuous.go; no production edits made.
- Official references: https://www.sqlite.org/backup.html warns that ordinary incremental backup can restart indefinitely under concurrent writes; https://www.sqlite.org/lang_vacuum.html documents consistent VACUUM INTO output but possible implicit-rowid changes; https://www.sqlite.org/wal.html documents that read-only opens can still involve/create SHM. Evaluate a pinned read transaction plus the existing CLI backup against VACUUM INTO, retaining integrity validation and bounded cancellation. No live Codex DB, credentials, or operator session touched.
- Private planning probe retained at /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair688-probe.0AoX2l/probe.py. Installed /usr/bin/sqlite3 reports3.54.0 Apple; Python sqlite3 reports3.53.4. An8.5MB fixture quiescent pinned backup passed quick_check, preserved the two-row transaction invariant and256 future-schema rows, but changed native-fixture SHM bytes. Ordinary active backup hit its5s probe deadline while the writer committed1498 transactions. Concurrent pinned probes then crashed the Python fixture writer with SIGBUS in walIndexAppend, FS pagein error22; retained crash reports Python-2026-10-19-015751.ips and015814.ips in the local DiagnosticReports directory. This is an unresolved fixture/runtime interaction, not evidence that live backup is safe or a production SQLite defect. Isolate it before choosing a mechanism. Probe uses only owned temporary databases.
- #689 peer seam: BundleLabel and its separate label/location index gain raw unknown-field preservation, but CatalogRecord/Manifest/SourceIdentity/FileEntry remain strict. Manifest JSON is ID input, so adding extensions requires explicit identity compatibility. PR450 remains unmerged, awaiting morning restore drill; do not build atop its branch or duplicate its codec work.
- 2026-10-19 probe resolution: the fixture hashed database sidecars with raw open/read/close in the Python process that owned SQLite connections. Closing another descriptor for that inode cancels the process's POSIX advisory locks; this invalidated SQLite coordination and caused the SIGBUS. Moving byte inspection into a separate process fixed the fixture. SQLite documents this hazard at https://www.sqlite.org/howtocorrupt.html#posix_advisory_locks_canceled_by_a_separate_thread_doing_close . Do not inspect native bytes inside a process holding SQLite connections.
- Corrected private trial7: pinned read-transaction backup of an 8,531,968-byte database completed in 1.0245s while the writer committed 125 transactions; integrity and the two-row invariant passed, and all 256 future-schema rows survived. VACUUM INTO completed in 1.0756s with 135 commits; ordinary unpinned backup exceeded the 5.0127s deadline with 647 commits. Quiescent pinned backup took 0.4495s and changed only SHM coordination bytes; DB/WAL hashes, sizes and identities remained unchanged. These are private mechanism measurements, not live acceptance or universal timing guarantees.
- Hub receipt c8f5b589-027a-4270-a6bc-8d66c03a6ef5, verified submitted from ops:0: approved replacing never-open with SQLite's normal read-only WAL-reader protocol, including SHM read marks; a short bounded pinned read transaction, no hot-loop retries, and pinned .backup instead of VACUUM INTO to preserve implicit rowids. Approved format-neutral scope: no catalog/manifest codec changes in #688. ARCH-EVOLVE applies to formats actually changed; opaque SQLite pages retain unknown schema and values.
- Read-only metadata measurement (no native database opened): logs DB 83,636,224 bytes plus WAL 107,564,992; state DB 346,783,744 plus WAL 4,375,472; thread history DB 1,989,320,704 plus WAL 4,342,512. Proposed 5s acquisition deadline must be measured against the approximately 2GB history database; it is not permission to lengthen the read pin if the machine cannot meet it. Native live acceptance remains to coordinate with the hub.
- SQLite backup implementation confirms a pre-existing source read transaction is retained across backup steps: https://sqlite.org/matrix/ev/src/backup.html (sqlite3_backup_step begins and later closes a source transaction only when it owns that transaction). A BEGIN followed by PRAGMA schema_version pins the source before .backup; the subprocess deadline releases that pin on cancellation. Private output validation happens after source reader exit.

## Revisions

- 2026-10-19, hub-approved reader contract and scope: replace copy-then-compare acquisition with a bounded SQLite read-only transaction and pinned online backup. DB/WAL content is never written by the reader; normal SHM coordination is allowed. Preserve unknown SQLite schema and implicit rowids through opaque backup. Keep catalog, manifest, label and configuration formats unchanged; no codec expansion or dependency on unlanded #689.
- 2026-10-19, plan-quality PQ-1/PQ-2/PQ-3 before implementation: the Go parent-owned deadline alone cannot bound an orphaned SQLite reader after collector death. Proposed revision is a kernel alarm armed by a tiny Perl exec launcher, plus an inherited flock on each private work directory so Sweep cannot remove a surviving reader's scratch. A private actual-sqlite3 probe killed its collector, proved the inherited lease remained held, then proved the alarm independently ended the SQLite reader and released the lease. Probe retained at /tmp/pair688-owner-death-probe.py. Perl is present here; this would add a runtime prerequisite, not an installation, and is awaiting hub approval. Simpler parent-only cancellation is insufficient; a new SQLite library or platform-specific Go timer launcher is a larger alternative. Ambiguous generic SQLite exit failures must be Failed/unknown, never inferred busy from stderr; only context/alarm evidence supports a timeout classification. Compress duplicate prose test inventories into the existing function/strategy table. No implementation started.
