---
id: 000701
status: open
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '6eeecf95897298d56c72cf96aee41f028b142d2b' # card fields mirrored from issue-cards; edit via sdlc
---

# Read closed WAL databases without sidecars on macOS SQLite

## Problem

#688's native acceptance found memories_1.sqlite (45,056 bytes, WAL format) had no -wal/-shm files, and macOS system sqlite3 3.54.0 returned exit 1 at PRAGMA schema_version with unable to open database file. The source remained unmodified by the failed read. A disposable closed-WAL fixture reproduces this independent of BEGIN order and -nofollow; installed Homebrew sqlite3 3.53.4 reads it and creates normal coordination sidecars. Active databases with existing sidecars work.

## Spec

Choose a small compatibility route for closed WAL-mode databases without restoring starvation for active writers. Hub's proposed options: (a) a verified plain copy for a database proved closed by no sidecars and unchanged size/mtime before and after, with identity and source-isolation checks; (b) a pinned known SQLite implementation. The owner must assess the closure proof and consistency contract before choosing. Do not assume immutable mode on a live database, silently open the native DB writable, or add a new runtime dependency without approval. Keep failures classified honestly and preserve earlier verified bundles.

## Done when

- A portable cold WAL fixture reproduces the system-CLI refusal and proves the selected compatibility route without changing native DB/WAL content.
- Races with a writer starting, sidecar creation and source replacement refuse unsafe publication; unknown SQL schema and rowids survive.
- Tests qualify the supported CLI(s), bound work and cleanup, and document the operator's one recovery action.

## Plan

- [ ]

## Log

- Hub 38daa8cd-726f-4984-aff5-c511b42bb896 explicitly keeps this out of #688's size-budget fix; file unclaimed. memories_1 is small and rarely changes, so not urgent.
- Owned private probe: Python creates a WAL database then closes it, removing sidecars. System /usr/bin/sqlite3 -readonly returns unable to open database file; /opt/homebrew/opt/sqlite/bin/sqlite3 succeeds. No machine changes or source repairs were attempted. A subsequent native readonly schema query succeeded once normal sidecars were present; who created those sidecars was not established.
