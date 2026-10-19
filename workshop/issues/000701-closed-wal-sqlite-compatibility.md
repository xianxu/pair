---
id: 000701
status: working
deps: []
github_issue:
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
collaboration-mode: collaboration
card_mirror: '9803967e2ac6b7895981494f3b55f412d5b48df5' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-19T12:08:04-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
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

- [ ] Qualify the closed-WAL refusal and private-copy route with disposable native fixtures and adversarial ordering controls.
- [ ] Implement the bounded, descriptor-owned cold capture and share existing validation, staging and cleanup.
- [ ] Verify cold and active routes, source preservation, catalog publication, cancellation and counted costs; update the archive map.
- [ ] Run the recorded branch suite and close review, then land through SDLC and report to the hub.

Detailed design: [closed WAL compatibility plan](../plans/000701-closed-wal-compatibility-plan.md). Await hub approval before change-code.

## Log

- Hub 38daa8cd-726f-4984-aff5-c511b42bb896 explicitly keeps this out of #688's size-budget fix; file unclaimed. memories_1 is small and rarely changes, so not urgent.
- Owned private probe: Python creates a WAL database then closes it, removing sidecars. System /usr/bin/sqlite3 -readonly returns unable to open database file; /opt/homebrew/opt/sqlite/bin/sqlite3 succeeds. No machine changes or source repairs were attempted. A subsequent native readonly schema query succeeded once normal sidecars were present; who created those sidecars was not established.

### 2026-10-19 — qualification and design

- Claimed in pair:7; start-plan completed, including its slot-owned weave refresh. Read the existing snapshot, subprocess, descriptor safety and continuous-publication seams. A read-only subagent audited those seams and challenged the consistency argument; no production files changed.
- Disposable probe under pair701-probe.hxsJxp: system SQLite 3.54.0 refuses a canonical closed-WAL path with exit 1 at schema_version and leaves its sole source file unchanged; installed Homebrew SQLite 3.53.4 succeeds and creates normal WAL/SHM sidecars. On an independent private copy, system SQLite's existing DELETE/integrity validation succeeds and returns rowid 73, unknown text and exact BLOB 0001FF. Original source still contains only its byte-identical database. The first probe used /var's symlink spelling and both CLIs refused -nofollow; canonical paths separate that different refusal from the actual closed-WAL failure. Tests must canonicalize fixture roots.
- SQLite documents ordinary read-only WAL access as depending on readable sidecars, permission to create them, or an immutable connection; raw copies of active transactions can be inconsistent. Sources: [read-only WAL](https://www.sqlite.org/wal.html#read_only_databases), [copying during a transaction](https://www.sqlite.org/howtocorrupt.html#_backup_or_restore_while_a_transaction_is_active). Immutable is not proposed on a native source.

## Revisions

### 2026-10-19 12:14 PDT — proposed compatibility route, pending approval

Choose a private plain copy only for a WAL-header source with all three coordination names (-wal, -shm, -journal) absent. Admission and two bounded full reads compare held file and parent identities, size, mtime and ctime; copy and reread digests must match. Parent mutation detects sidecars that appeared and disappeared between observations. Unknown evidence refuses the candidate without retry, preserving the existing staged file and verified catalog. This is an observed stable-input contract for ordinary SQLite on reliable local filesystems, not a claim of exclusive ownership. A writer beginning after the final capture fence does not invalidate the earlier snapshot. Active sidecars continue through the current pinned SQLite backup without any copy-then-compare fallback. No new dependency or format change. See the durable plan for assumptions, failure projection, costs and tests. #710 is queued by the hub after this issue; its coverage work is outside #701.
