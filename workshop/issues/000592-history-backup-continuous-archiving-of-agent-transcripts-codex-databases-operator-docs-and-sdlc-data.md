---
id: 000592
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
card_mirror: 'f538d3287ef8d775d7a762e142a06244a9297c8a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T08:50:30-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
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
