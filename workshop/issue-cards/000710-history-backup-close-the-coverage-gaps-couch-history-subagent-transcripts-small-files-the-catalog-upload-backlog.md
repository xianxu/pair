---
id: 000710
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
---

# history-backup: close the coverage gaps (Couch history, subagent transcripts, small files, the catalog, upload backlog)

## Problem

A read-only inventory on 2026-10-19 (ops side doc `teams/devinfra/tl-status-xian-xu.side/2026-10-19-08-history-backup-status.md`,
"What's backed up, how changes are found, and the gaps") found that only 6 of the 17 kinds declared in
`cmd/internal/historyarchive/sources.json` run (`ContinuousKinds`, `continuous_discover.go:16`), and these gaps:
1. **Couch's own history** (~0.7 GB): ledgers, session names, thread store, supervision log, session-sync data. Declared
   "continuous" (sources.json:57-85) but not in `ContinuousKinds`; pair's collector deletes them (session class). Thread
   store excluded on purpose so far (continuous_discover.go:13-15: per-slot / self-referential handling needed).
2. **Claude subagent transcripts** (221 MB, `~/.claude/projects/*/*/subagents/*.jsonl`): the claude-transcripts pattern only
   reaches depth 2 (known, history-backup project.md:69).
3. **Small, irreplaceable files:** the operator's session files `ops/teams/*/tl-status-*.md` (gitignored; the ops pattern
   takes only `tl-status-*.side/`), Claude memory `~/.claude/projects/*/memory/`, Claude prompt history
   `~/.claude/history.jsonl`, Codex `config.toml` and `session_index.jsonl` (never `auth.json`).
4. **The archive catalog itself** (`pair/archive/catalog.jsonl`, 2.3 MB): not archived or uploaded; the `archive-catalog` kind
   is declared but not run. If the SSD dies, finding a file means decrypting every bundle on Drive.
5. **Loose ends:** the five codex-database bundles made 10-19 09:41–10:03Z never reached Drive (cause unknown; possibly the
   sandbox blocking rclone in single-kind runs); and backups run only while Couch runs.

The operator approved closing gaps 1–5 (10-19), before retention deletes anything.
