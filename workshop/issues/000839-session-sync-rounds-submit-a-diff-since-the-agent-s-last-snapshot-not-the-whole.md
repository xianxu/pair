---
id: 000839
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
collaboration-mode: collaboration
card_mirror: '4fb10f56eddc71911c7063203e256ad9e3ef37b2' # card fields mirrored from issue-cards; edit via sdlc
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

### 2026-10-23
