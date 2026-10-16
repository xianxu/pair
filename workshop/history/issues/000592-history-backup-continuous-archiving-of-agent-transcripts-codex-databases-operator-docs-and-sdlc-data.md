---
id: 000592
status: done
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
card_mirror: '3204bf3e56cbee50554c1963e4724a0d0e8a2dd1' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T08:50:30-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
flow: {kind: full, provenance: inferred}
actual_hours: 1.08
---

# history-backup: continuous archiving of agent transcripts, Codex databases, operator docs and sdlc data

## Problem

Agent transcripts, Codex databases, ops operator-local docs and sdlc runtime data have no copy anywhere, and Claude
Code prunes transcripts after 30 days by default. Ops project history-backup, design §2.1 "continuous" mode.

## Spec

- A daily pass over the inventory's continuous sources: archive each file once it has been quiet (unmodified) for 3
  days, keyed by content hash so nothing is uploaded twice; a file changed after archiving gets a new version.
- SQLite sources (Codex `*.sqlite`) are copied with SQLite's online backup into a private temp file, never as raw
  bytes and never by opening the live database in a way that creates files beside it.
- Never deletes anything locally (local retention for these sources is a later, separate decision).
- Sources: `~/.claude/projects`, `~/.codex/sessions` + `*.sqlite` + `history.jsonl`, `~/.qoder/projects`, ops
  `teams/*/qa`, `teams/*/assets`, `teams/*/tl-status-*.side`, each repo's `.git/sdlc/` (keyed by machine + repo +
  worktree path + relative path + content hash, per the design's side-doc rule).
- Depends on the foundation issue (bundles, catalog, sink).

## Done when

- Tests: quiet-period selection, content-hash dedup, a changed file gets a new version, SQLite copy is coherent and
  leaves the source directory's file set unchanged.
- A live daily pass to the local-directory sink covering every listed source, with catalog entries.

## Plan

Durable plan: workshop/plans/000592-history-backup-continuous-archiving-of-agent-transcripts-codex-databases-operator-docs-and-sdlc-data-plan.md (TL approval 82ac1242: daily Codex DB snapshots with -wal copied first and size/mtime stability, Couch 24h scheduler with lock, standalone command).

## Log

### 2026-10-15

### 2026-10-16 — build and live pass (pair:2; TL 82ac1242, b066acc1)
- 2026-10-16: closed — Round-2: BR-4 Sweep is in the Snapshotter interface; the pass test plants a leftover sqlite-* copy and asserts it is swept (RED with the pass sweep disabled). Minors: one failing source no longer aborts the others (joined errors, pass not marked complete); scheduler retries a failed pass after 6 h, not hourly (test); continuous.log holds the latest pass only; sdlc identity keyed by git common dir alone; changed quiet transcript archived as a new version (pass test). BR-2 disputed with measurement. Recorded suite green on 7c735051; historyarchive, archivecmd, couchcmd scheduler green after fixes. Live passes verified (34/34 records).; review verdict: FIX-THEN-SHIP
- 2026-10-16: flow upgraded quick → full — 967 added lines in code files (limit 100); an earlier round of this close already ran the full review

- Built per the durable plan: `ContinuousConfig`, `SelectQuiet`, `ArchivedIndex` (per-file dedup by identity+path+hash, size+mtime fast path), `Discover` + `ContinuousRoots` (ops per team, sdlc per git common dir), `SQLiteSnapshotter` (wal copied first, size/mtime stability, private `.backup`, `journal_mode=DELETE`, `integrity_check`; never opens the live DB), `ContinuousPass` (lock, last-run marker, never deletes), `pair archive continuous`, Couch `archiveScheduler` (hourly check, runs when configured and the last pass is over 24h old, nice 10, passes the global PAIR_DATA_DIR because a Pair session scopes it per repo). #593's snapshot refusal stays for manual requests; only the pass admits its own collected snapshots.
- Live first pass (config: sink ~/Backups/pair-history, catalog ~/.local/share/pair/archive/catalog.jsonl, brain key 0ECF6AC0…C2F0): exit 0, wall 975 s niced; 34 bundles, 6,978 files, 6.57 GB raw to 0.59 GB compressed (claude 8 bundles/4,344 files, codex transcripts 22/2,484, codex databases 2/5, qoder 2/145). Independent check: all 34 catalog records' bundles match their recorded sha256 and size in the sink, labels present. ops-documents and sdlc-runtime archived 0: every such file is newer than 72 h (measured), so they start as files settle. thread_history_1.sqlite (1.5 GB) skipped by the 1 GiB bound, follow-up pair#616.
- Second pass: 75 s; 10 Claude + 1 Codex transcripts archived, each verified new (just under 72 h at pass 1, over by pass 2; none duplicates); one changed database snapshot version; logs_2 and thread_history "changed during copy; retry next pass" (the stability check working on live databases).
- Backup is on the same SSD: not yet disk-failure protection; Drive is next (design §2.4).
