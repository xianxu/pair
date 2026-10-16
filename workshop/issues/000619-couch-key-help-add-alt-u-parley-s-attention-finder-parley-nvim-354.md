---
id: 000619
status: open
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: 'a8dd917bb9859e9181a884382493515731559773' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch key help: add Alt+u, parley's attention finder (parley.nvim#354)

## Problem

parley.nvim#354 (the attention finder, part of ops project universal-agent-input-interface) binds **Alt+u** (M-u) in
parley to open the list of documents waiting on or answered by an agent. Couch's key help
(`cmd/internal/keyhelp/catalog.go`) doesn't list it, so the operator can't discover it from Couch.

## Spec

- Add one entry for Alt+u, "parley: attention finder (documents waiting on / answered by an agent)", in the section
  where parley's editor keys are listed, matching the existing entries' style.
- Confirm Alt+u is not bound by Couch or pair's zellij config (checked 10-16: it isn't) and keep it that way: if the
  catalog has a uniqueness test, it covers the new entry.

## Done when

- The key help shows Alt+u with that description; the catalog's existing tests pass (plus the uniqueness check).

## Plan

- [ ]

## Log

### 2026-10-16
