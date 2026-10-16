# History-backup continuous archiving Implementation Plan

> **For agentic workers:** Consult AGENTS.md Section 3 (Subagent Strategy) to determine the appropriate execution approach: use superpowers-subagent-driven-development (if subagents are suitable per AGENTS.md) or superpowers-executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A daily, quiet, never-deleting pass that archives every continuous history source once (content-hash dedup), with coherent private snapshots of Codex's live SQLite databases, run by Couch on a 24h schedule and runnable by hand.

**Architecture:** `pair archive continuous` discovers files from the inventory's continuous rules, selects the quiet ones (3 days; Codex databases daily via a snapshot), drops anything the catalog already holds by (source identity, path, content hash), and hands the rest to #593's existing `Archiver` in request-sized batches. A Couch scheduler invokes the command once a day; the command holds its own lock, so manual and scheduled runs never overlap.

**Tech Stack:** Go (`cmd/internal/historyarchive`, `cmd/internal/archivecmd`, `cmd/internal/couchcmd`), the system `sqlite3` CLI (already used by `sessioninventory`), #593's codec (zstd + GPG) and `LocalSink`.

**Approved decisions (TL 82ac1242):** Codex DBs are snapshotted daily regardless of quiet, deduped by content hash; the `-wal` is copied BEFORE the main `.sqlite`, both sizes and mtimes are recorded before and after, and a copy whose source changed (checkpoint mid-copy) or that fails `PRAGMA integrity_check` is skipped and retried on the next pass. The daily pass runs inside Couch on a 24h scheduler with a lock; the command stays standalone. Codex `history.jsonl` is added.

---

## Core concepts

### Pure entities

| Name | Lives in | Status |
|------|----------|--------|
| `ContinuousConfig` | `cmd/internal/historyarchive/continuous_config.go` | new |
| `Candidate` | `cmd/internal/historyarchive/continuous_select.go` | new |
| `SelectQuiet` | `cmd/internal/historyarchive/continuous_select.go` | new |
| `ArchivedIndex` | `cmd/internal/historyarchive/continuous_index.go` | new |
| `SnapshotObservation` / `SnapshotStable` | `cmd/internal/historyarchive/sqlite_snapshot.go` | new |
| `SourceRule` patterns (codex `history.jsonl`, ops `tl-status-*.side`) | `cmd/internal/historyarchive/sources.json` | modified |

- **ContinuousConfig** — operator-written `$PAIR_DATA_DIR/archive/continuous.json`: `recipients`, `sink` (absolute dir), `catalog` (absolute file), `ops_root` (ops checkout), `workspace` (dir whose child git repos supply `sdlc-runtime`). Strict decode; absent file means "not configured" and the pass does nothing.
  - **Relationships:** 1 per Pair data dir; read by the command, never by Couch.
  - **DRY rationale:** one place for the destination and recipients that #593 requests carried per call.
  - **Future extensions:** per-kind sink overrides once #593's routing grows beyond `local`.
- **Candidate** — one discovered file: kind, `SourceIdentity`, relative path, size, mtime (hash filled lazily).
  - **Relationships:** N per source root; 1:1 with a `FileEntry` once archived.
- **SelectQuiet(candidates, now, quiet)** — keeps regular files whose mtime is at or before `now-quiet`. Pure; tested with fixed times.
- **ArchivedIndex** — set of `(identity, path, sha256)` and the fast path `(identity, path, size, mtime) -> sha256`, built from catalog records' manifests. `Seen(c)` answers without hashing when size and mtime match an archived entry; otherwise the caller hashes and asks `SeenHash(c, sha)`.
  - **DRY rationale:** per-file dedup over the existing catalog; no second database.
  - **Future extensions:** restore (`find`, piece 5) can reuse the same index.
- **SnapshotObservation / SnapshotStable** — size and mtime of the main DB and its `-wal` before and after the copy; stable only when unchanged and both present-or-absent alike.

### Integration points

| Name | Lives in | Status | Wraps |
|------|----------|--------|-------|
| `Discover` | `cmd/internal/historyarchive/continuous_discover.go` | new | `filepath.WalkDir` / `fs.Glob` over source roots |
| `SQLiteSnapshotter` | `cmd/internal/historyarchive/sqlite_snapshot.go` | new | file copy + `sqlite3` CLI (`.backup`, `PRAGMA integrity_check`) |
| `ContinuousPass` | `cmd/internal/historyarchive/continuous.go` | new | `Archiver.Create`, `OpenCatalog`, lock file, last-run marker |
| `pair archive continuous` | `cmd/internal/archivecmd/run.go` | modified | CLI |
| `ArchiveScheduler` | `cmd/internal/couchcmd/archive_schedule.go` | new | timer + exec of the pair binary |

- **Discover** — expands the inventory's patterns under each resolved root (bounded: `MaxFiles` per request, skips symlinks and non-regular files). Roots: existing tokens via `ResolveRoot`; `ops-worktree` = each `teams/*` dir under `ops_root`; `git-common` = each git repo directly under `workspace` (its `git rev-parse --git-common-dir`). Identity = `SourceIdentity{Host, Root, Repository, Worktree}` per the side-doc rule.
  - **Injected into:** `ContinuousPass` (a fake filesystem root in tests is a temp dir; no mocks).
- **SQLiteSnapshotter** — for each `*.sqlite`/`*.sqlite3`: observe; copy `-wal` (if present) then the main file into a private temp dir; observe again; if unstable skip; run `sqlite3 <private> ".backup '<staging>/<name>'"` and `PRAGMA integrity_check` on the result; publish into the stable staging dir `$PAIR_DATA_DIR/archive/snapshots/codex-database/` (the snapshot kind's `Source.Root`, so identity stays stable across passes). Never opens the live file.
  - **Injected into:** `ContinuousPass` via a `Snapshotter` interface; tests use a real `sqlite3` on temp DBs (live conformance) and a fake that returns "unstable".
- **ContinuousPass** — holds `$PAIR_DATA_DIR/archive/continuous.lock` (nonblocking flock; held means exit 0 "already running"), loads config (absent = no-op), runs every continuous kind (or one `--kind`), builds the index once per catalog, archives new files in batches of at most `MaxFiles` through `Archiver.Create`, and writes `continuous.last` on completion. Never deletes; writes only under the archive store.
- **ArchiveScheduler** — a Couch goroutine: hourly check; if `continuous.last` is older than 24h (or absent) and config exists, execs `pair archive continuous` at low priority with output to a log under the archive store; never blocks the console; one notice on failure.
  - **Future extensions:** move to the shared supervision cadence if one appears.

---

## Tasks

### Task 1: Inventory and config

**Files:** Modify `cmd/internal/historyarchive/sources.json` (codex-transcripts adds `history.jsonl`; ops-documents patterns become `qa/*`, `assets/*`, `tl-status-*.side`); create `continuous_config.go` + test.

- [ ] Test: `Defaults()` codex-transcripts includes `history.jsonl`; ops-documents has no bare `*`.
- [ ] Test: `LoadContinuousConfig` absent file returns `(nil, nil)`; strict decode refuses unknown fields and relative paths.
- [ ] Implement; run `go test ./cmd/internal/historyarchive -run 'Inventory|ContinuousConfig'`; commit.

### Task 2: Quiet selection and archived index (pure)

**Files:** create `continuous_select.go`, `continuous_index.go` + tests.

- [ ] Test `SelectQuiet`: files at now-3d-1s kept, at now-3d+1s dropped, non-regular dropped.
- [ ] Test `ArchivedIndex`: built from two catalog records; `Seen` true for same path+size+mtime; a changed file (new size or mtime) needs hashing and a new sha is not seen (new version); same sha under another identity is not seen.
- [ ] Implement; commit.

### Task 3: Discovery

**Files:** create `continuous_discover.go` + test (temp-dir trees).

- [ ] Test: claude `projects/*/*.jsonl` found; symlinks and dirs skipped; ops `teams/a/qa/x.md` and `teams/a/tl-status-1.side` found with Root = `teams/a`; sdlc-runtime enumerates two git repos under a workspace dir and keys each by its common dir.
- [ ] Implement; commit.

### Task 4: SQLite snapshot

**Files:** create `sqlite_snapshot.go` + test.

- [ ] Test (real sqlite3, temp WAL DB with rows): snapshot passes `integrity_check`, row count matches, and the source directory's file set (names) is identical before and after.
- [ ] Test: observations differing before/after (fake stat) reports unstable and produces no snapshot.
- [ ] Test: the `-wal` copy happens before the main file (ordered copier fake records order).
- [ ] Implement; commit.

### Task 5: Continuous pass

**Files:** create `continuous.go` + test (fake codec from `archivetest`, real `LocalSink`, temp catalog).

- [ ] Test: first pass archives quiet files; second pass archives nothing; a changed quiet file is archived again as a new version; a fresh file waits; nothing under any source root is created, changed or deleted; lock held -> "already running", exit 0; config absent -> no-op.
- [ ] Implement; commit.

### Task 6: CLI and Couch scheduler

**Files:** modify `cmd/internal/archivecmd/run.go` (+ test); create `cmd/internal/couchcmd/archive_schedule.go` (+ test with fake clock and fake exec); wire start in Couch's run.

- [ ] Test CLI: `pair archive continuous [--kind K]` usage, exit codes.
- [ ] Test scheduler: runs when last is >24h or absent and config exists; not when recent or unconfigured; never two at once.
- [ ] Implement; commit.

### Task 7: Atlas, live pass, close

- [ ] Update `atlas/history-backup.md` (continuous mode, config, snapshot rule, schedule).
- [ ] Live: write a config pointing the sink and catalog to a local directory with the operator's recipient; run `pair archive continuous`; confirm catalog entries for every listed source and a coherent Codex snapshot; run again and confirm nothing new.
- [ ] Recorded suite, close review, land.
