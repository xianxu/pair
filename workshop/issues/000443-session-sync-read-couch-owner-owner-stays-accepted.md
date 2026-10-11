---
id: 000443
status: working
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '5dc4a666cc6318dffb656b105fb85a66c83b1f1a' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-11T10:12:09-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: MacBook Pro
    workspace: pair:5
    worktree: /Users/xianxu/workspace/worktree/pair-slot5/pair
    repository: github.com/xianxu/pair
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

- [x] `ParseOwner`: `couch-owner` with the `owner` fallback; tests for the Done-when rows; atlas couch.md.

## Log

### 2026-10-11
- 2026-10-11: closed — go test ./cmd/internal/sessionsync green: TestParseOwnerPrefersCouchOwnerOverTheLegacyAlias (couch-owner; owner alone; both either order -> couch-owner; process-instruction [copy-edit] line skipped) + TestStatusReportsCouchOwnerOrTheLegacyOwner (Status().Owner, the couch --session-sync status path); mutation disabling the couch-owner lookup fails both. make -k test rc=0 (unsandboxed). go test ./...: 5 launcher/checkpoint fails reproduce on main; TestContinuationWriterPublishesExactCheckpointAcrossWorktrees is continuation-writer code, untouched here. Round 2: BR-1 swept (SKILL.md, atlas, refusal texts); couchcmd tests green.; review verdict: SHIP
- `ParseOwner` tries `couch-owner` then `owner`; the line-wise frontmatter reader already handled hyphenated keys and non-YAML lines, so only a doc comment pinned that. Refusal texts now name `couch-owner:`. Tests: `TestParseOwnerPrefersCouchOwnerOverTheLegacyAlias` (unit, incl. owner-first ordering + process-instruction), `TestStatusReportsCouchOwnerOrTheLegacyOwner` (the `status` path). Mutation (couch-owner lookup disabled) fails both.
- Close review round 1 (FIX-THEN-SHIP): BR-1 couch SKILL.md still taught `owner:` — swept every live restatement of the key (SKILL.md, atlas Owed bullet), and the two refusal texts now name the value with both keys instead of asserting `couch-owner:`.
