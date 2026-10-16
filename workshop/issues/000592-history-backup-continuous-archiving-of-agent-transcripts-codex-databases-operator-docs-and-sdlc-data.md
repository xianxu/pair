---
id: 000592
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: '6442affcbc9a1c72805182d72465cf0d382a7d40' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-15
