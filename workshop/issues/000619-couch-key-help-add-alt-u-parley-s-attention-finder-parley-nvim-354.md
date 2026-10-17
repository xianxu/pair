---
id: 000619
status: working
deps: []
github_issue:
created: 2026-10-16
updated: 2026-10-16
estimate_hours:
card_mirror: '32acef3b717ceafe0533a1caeba13812bc8a54ee' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T18:18:08-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:7
    worktree: /Users/xianxu/workspace/worktree/pair-slot7/pair
    repository: github.com/xianxu/pair
---

# Couch key help: add Alt+Space, parley's attention finder (parley.nvim#354)

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

- [ ] Add a red rendered-help regression for Alt+Space and Ctrl+Y then Space, uniquely under Parley editor, with no stale Alt+u.
- [ ] Add explicit companion-editor help metadata/context and curated rows; retain Pair source drift checks, without key handlers or peer-repo changes (ARCH-DRY/ARCH-PURE). Verify Couch/Zellij do not intercept either key.
- [ ] Run package tests, recorded full suite, close review and normal landing.

## Log

### 2026-10-16

## Revisions

### 2026-10-16 — operator selects Alt+Space

TL delivery c16d5cbf-dd0c-488d-b9dd-82f71d275f9c changes the attention finder from Alt+u (M-u) to Alt+Space (M-Space), per the operator. This supersedes the original key in the Problem, Spec and Done when; the old Alt+u conflict check does not establish that Alt+Space is free. Verify the revised key and implement/test its catalog entry only after #604 lands, preserving the assigned order.

### 2026-10-16 — both attention-finder bindings

TL delivery 309fc964-4631-45a3-8423-9825a462ac4c extends the documented keys to both Alt+Space and <C-y><Space> (Ctrl+Y followed by Space). This supersedes the single-key Spec and Done when: show both as Parley editor bindings, not Pair-global bindings. The catalog has no Parley source today; a small explicit companion-editor metadata source preserves its named-source rule without importing a peer repo or inventing executable bindings.
