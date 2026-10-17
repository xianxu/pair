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

parley.nvim#354 (the attention finder, part of ops project universal-agent-input-interface) binds **Alt+Space** (M-Space) in
parley to open the list of documents waiting on or answered by an agent. Couch's key help
(`cmd/internal/keyhelp/catalog.go`) doesn't list it, so the operator can't discover it from Couch.

## Spec

- Add one entry for Alt+Space, "parley: attention finder (documents waiting on / answered by an agent)", in the section
  where parley's editor keys are listed, matching the existing entries' style.
- Confirm Alt+Space is not bound by Couch or pair's zellij config (verify for the revised key during implementation) and keep it that way: if the
  catalog has a uniqueness test, it covers the new entry.

## Done when

- The key help shows Alt+Space with that description; the catalog's existing tests pass (plus the uniqueness check).

## Plan

- [ ]

## Log

### 2026-10-16

## Revisions

### 2026-10-16 — operator selects Alt+Space

TL delivery c16d5cbf-dd0c-488d-b9dd-82f71d275f9c changes the attention finder from Alt+u (M-u) to Alt+Space (M-Space), per the operator. This supersedes the original key in the Problem, Spec and Done when; the old Alt+u conflict check does not establish that Alt+Space is free. Verify the revised key and implement/test its catalog entry only after #604 lands, preserving the assigned order.
