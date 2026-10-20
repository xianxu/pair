---
id: 000710
status: open
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '3f70ddd41d3793ed3cbb0f8d7e35cf08f47f7ac6' # card fields mirrored from issue-cards; edit via sdlc
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

## Spec

- **Gap 1:** run the declared Couch kinds (ledgers, session names, supervision log, session-sync). For the thread store, solve
  the noted per-slot / self-referential problem (e.g. a consistent snapshot like the SQLite path, or a copy taken under the
  store's own lock) rather than leaving it out.
- **Gap 2:** extend the claude-transcripts pattern to nested subagent transcripts (any depth under `projects/`), keeping the
  72-hour quiet rule.
- **Gap 3:** add the small files as kinds (or extend existing ones): ops `teams/*/tl-status-*.md`, Claude memory, Claude
  `history.jsonl`, Codex `config.toml` and `session_index.jsonl`. Explicitly exclude credentials (`auth.json`, rclone
  config, keys) with a test.
- **Gap 4:** archive the catalog itself, encrypted like any bundle, after each pass, and upload it; restore can bootstrap
  from the newest catalog bundle on Drive without decrypting every bundle. Document the bootstrap in the atlas.
- **Gap 5:** find why the five single-kind bundles didn't upload and fix it (a pass must upload everything it catalogued, or
  report the backlog loudly; the stale-backup alert from #700 should cover "catalogued but not on Drive"). Backups-only-
  while-Couch-runs: report it (the alert covers a gap of > 24 h); a launchd schedule is out of scope unless trivial.
- ARCH-EVOLVE: catalog and manifest changes are additive; older readers tolerate new kinds.

## Done when

- One real pass archives and uploads each new kind (counts and sizes logged); the catalog bundle is on Drive.
- A restore drill (the #689 script) restores one subagent transcript, one Couch ledger and the session file from Drive.
- A fixture proves credentials are never selected; the upload-backlog case is reported.

## Plan

- [ ]

## Log

### 2026-10-19
