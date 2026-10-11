---
id: 000445
status: open
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '5ca74c3d41ab1c7edfc38738d3f1b360dcfd29d2' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ] Switch the file write to `writeAtomic`, keeping the mode; add the torn-read test; check parley live.

## Log

### 2026-10-11
