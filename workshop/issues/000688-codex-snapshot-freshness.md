---
id: 000688
status: open
deps: []
github_issue:
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
collaboration-mode: collaboration
card_mirror: '70193796ba33bd914e51092c2c97c2a730696714' # card fields mirrored from issue-cards; edit via sdlc
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
