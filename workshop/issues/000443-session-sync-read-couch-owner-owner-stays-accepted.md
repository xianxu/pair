---
id: 000443
status: open
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: 'dfd26ab7c7fc229c5e24aa3d2a6e43fee1220b33' # card fields mirrored from issue-cards; edit via sdlc
---

# session-sync: read couch-owner (owner stays accepted)

## Problem

parley.nvim#316 renames the frontmatter key that names a document's Couch slot from `owner:` to `couch-owner:` (the operator, 10-11): a bare `owner` collides with sdlc's issue owner and doesn't say it's a Couch address. Couch's session-sync reads `owner:` (`cmd/internal/sessionsync/file.go:75`, `ParseOwner`) to decide the owner slot.

## Spec

`ParseOwner` reads `couch-owner:` first and falls back to `owner:` (accepted as a legacy alias until no session-sync file uses it). The frontmatter reader accepts hyphenated keys and must not choke on a `process-instruction: [copy-edit] …` line (not valid YAML; read lines raw, as today).

## Done when

- A file with `couch-owner: ops:0` reports owner ops:0 in `couch --session-sync status` (test); `owner:` alone still does (test); with both, `couch-owner` wins (test).
- A `process-instruction: [copy-edit] text` line in the frontmatter doesn't disturb parsing (test).

## Plan

- [ ] `ParseOwner`: `couch-owner` with the `owner` fallback; tests for the Done-when rows; atlas couch.md.

## Log

### 2026-10-11
