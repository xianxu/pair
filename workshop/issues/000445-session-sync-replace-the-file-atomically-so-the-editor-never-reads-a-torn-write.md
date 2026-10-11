---
id: 000445
status: codecomplete
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '46af15c46c904d9b53adf07191ff5a73d8d23041' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T11:15:07-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
flow: {kind: quick, provenance: inferred, spec: "784079ba", done: "c26513d8"}
actual_hours: 0.30
---

# session-sync: replace the file atomically, so the editor never reads a torn write

## Problem

`couch --session-sync write` rewrites the session-sync file in place (`os.WriteFile`, `cmd/internal/sessionsync/sync.go:353`: truncate, then write). pair#444 found a second Couch process reading it mid-write and seeing a torn file; it fixed that reader by gating it behind the flock. Readers outside the flock are still exposed, and the most important one is the operator's editor: parley reloads a clean buffer when the file changes on disk (parley.nvim#315 makes that silent). A reload that lands between the truncate and the write loads a partial file; if the operator then edits and saves, the missing tail is lost.

## Spec

Write the session-sync file the way the shadow already is: a temp file in the same directory, fsync, then rename over the original (`writeAtomic`). Keep the file mode. nvim notices the change by mtime, and a renamed file is fine for it (check that parley's reload and the `.clobbered` path still behave).

## Done when

- A reader polling the file while writers run in a loop never sees a partial file (test).
- parley's reload of a clean buffer still works after an agent write, as checked live in the ops status file.

## Plan

- [x] Switch the file write to an atomic replace (`replaceShared`: `durablefile.WriteAtomicMode` with the file's own mode,
  through a symlink to its target so the link survives); add the torn-read test and a mode/symlink test.
- [x] Check parley's reload and `.clobbered` path against a copy of the ops status file (headless nvim, real config).

After landing: `make build` on pair:0 (whose couch the ops TL runs), then watch the real ops status file across live
TL writes. This is recorded in the report to ops:0, since it can only run after the merge.

## Log

### 2026-10-11
- 2026-10-11: closed — post-close delta is a comment-only change to replaceShared (ledger.go) naming the accepted rename losses (hard links, foreign owner, xattrs, ACLs) per close-review Minor; go build ./cmd/... ok, gofmt clean (fixed-then-ship); review verdict: SHIP
- 2026-10-11: closed — make -k test exit 0 incl. go test ./... (scrubbed env, sandbox off); new replace_test.go: torn-read test fails 3/3 with os.WriteFile restored (reader saw 0 bytes), mode and symlink assertions each fail under their own mutation; live headless nvim+parley (operator config) on a copy of ops tl-status: 8 couch writes -> 8 silent reloads, buffer==disk, mode kept, no prompt/.clobbered; operator-lock race -> operator copy restored, agent copy in .clobbered; review verdict: SHIP

- `os.WriteFile` truncates and then writes, so a reader can see 0 bytes. The comment "parley watches this file's inode"
  is out of date: parley polls `fs_stat(path).mtime` every second (`session_sync.lua` `disk_mtime`/`start_watch`), and
  nvim's checktime compares the mtime and size, so a rename is visible to both.
- Rename over a symlink would replace the link with a regular file, so `replaceShared` resolves it first. It also
  keeps the mode, since `durablefile.publish` chmods to 0600. `WriteAtomicMode` is the mode-taking variant; `WriteAtomic`
  stays 0600 for the agent-private files.
- Torn-read test (`replace_test.go`): 40 writes of a ~240 KB section while a goroutine polls the file. Mutation: with
  `os.WriteFile` back, it fails 3/3 ("a reader saw a partial file (0 bytes)"). The mode and symlink assertions each
  fail under their own mutation.
- Live check (headless nvim with the operator's config and parley, a copy of `ops/teams/devinfra/tl-status-xian-xu.md`,
  this branch's couch writing `## Fleet` 8 times, 1.7 s apart): 8 silent reloads, the buffer ends equal to the disk,
  the inode changed on every write, mode 0644 kept, no prompt, no `.clobbered`. The `.clobbered` path (operator lock
  taken, then an agent rename lands, owner set to a dead `nobody:9` so no real message is sent): the operator's copy
  was written back, the agent's copy was kept in `.clobbered`, and the notice was the usual one.
- `make -k test` (default TMPDIR, five-var scrub, sandbox off) exits 0, including its `go test ./... -count=1` recipe.
- Close review (SHIP, round 1): two advisory Minors. Hard links, a foreign owner, xattrs and ACLs are lost on rename; this is accepted and stated in the `replaceShared` comment. The real-file live check runs after the merge and is reported to ops:0.
