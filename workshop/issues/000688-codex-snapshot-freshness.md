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

Investigate and plan before code; coordinate any real-environment check with the hub.

## Log

- Filed from #678 hub direction cb95f294-af33-4a95-99d0-b583012e94da. Evidence is read-only catalog records and operator-reported changed-during-copy skips; the precise frequency/root cause remains to be measured. Existing #616 is a size-limit fix, not the same issue.
- 2026-10-19 initial investigation: current SQLiteSnapshotter requires unchanged DB/WAL size+mtime across WAL-first/DB-second raw copies, then runs SQLite only on the private copy. The native source is never opened by SQLite. Busy skips return nil errors and can still advance continuous.last; catalog VerifiedAt means last verified bundle, not last successful unchanged snapshot observation. Keep these meanings distinct in the plan. A read-only pipeline exploration found exact source identity/staging/dedup and lifetime seams in sqlite_snapshot.go/continuous.go; no production edits made.
- Official references: https://www.sqlite.org/backup.html warns that ordinary incremental backup can restart indefinitely under concurrent writes; https://www.sqlite.org/lang_vacuum.html documents consistent VACUUM INTO output but possible implicit-rowid changes; https://www.sqlite.org/wal.html documents that read-only opens can still involve/create SHM. Evaluate a pinned read transaction plus the existing CLI backup against VACUUM INTO, retaining integrity validation and bounded cancellation. No live Codex DB, credentials, or operator session touched.
- Private planning probe retained at /var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair688-probe.0AoX2l/probe.py. Installed /usr/bin/sqlite3 reports3.54.0 Apple; Python sqlite3 reports3.53.4. An8.5MB fixture quiescent pinned backup passed quick_check, preserved the two-row transaction invariant and256 future-schema rows, but changed native-fixture SHM bytes. Ordinary active backup hit its5s probe deadline while the writer committed1498 transactions. Concurrent pinned probes then crashed the Python fixture writer with SIGBUS in walIndexAppend, FS pagein error22; retained crash reports Python-2026-10-19-015751.ips and015814.ips in the local DiagnosticReports directory. This is an unresolved fixture/runtime interaction, not evidence that live backup is safe or a production SQLite defect. Isolate it before choosing a mechanism. Probe uses only owned temporary databases.
- #689 peer seam: BundleLabel and its separate label/location index gain raw unknown-field preservation, but CatalogRecord/Manifest/SourceIdentity/FileEntry remain strict. Manifest JSON is ID input, so adding extensions requires explicit identity compatibility. PR450 remains unmerged, awaiting morning restore drill; do not build atop its branch or duplicate its codec work.
