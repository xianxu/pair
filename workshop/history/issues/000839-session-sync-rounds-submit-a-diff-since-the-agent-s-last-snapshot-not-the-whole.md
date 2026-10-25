---
id: 000839
status: punt
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-24
estimate_hours:
collaboration-mode: collaboration
card_mirror: 'a8c14bfed44b04d5efc95376df53c5d7d5a14431' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-24T17:58:10-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:1
    worktree: /Users/xianxu/workspace/worktree/pair-slot1/pair
    repository: github.com/xianxu/pair
---

# Session-sync rounds submit a diff since the agent's last snapshot, not the whole file

## Problem

## Spec

**Submit what changed, not the whole file (operator, 10-23).** The universal agent input interface decouples what
the agent sees (the transcript) from where the operator edits: edits happen anywhere in a document, and submissions
materialise into a timeline. Since the agent normally already has the file in context (and can read it with one local
tool call when not), a session-sync round can send a **diff of what changed since the agent last saw the file**
instead of the whole document. Source: ops `session.side/2026-10-23-03-mode-simplification.md` (the side note).

Design questions for the plan: what "last seen" is anchored to (the last round's snapshot per agent), how a diff
names its base so the agent can detect a gap and fall back to the full file, and size bounds.

## Done when

- A submit of a file the agent has seen sends a diff against the last snapshot it was given, with the base named;
  a missing or stale base falls back to the full file (tests).

## Plan

- [ ]

## Log


- 2026-10-24: abandoned (punt): operator 10-24: documents are small (session file about 200 lines, side docs 30 to 105), so diff-since-snapshot saves little and adds stitching complexity; revisit if documents grow large. Discussion: ops teams/devinfra/projects/unified-editorial-interface/project.side/2026-10/2026-10-24-01-diff-since-snapshot.md
### 2026-10-23
