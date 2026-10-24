---
id: 000855
status: open
deps: []
github_issue:
created: 2026-10-23
updated: 2026-10-23
estimate_hours:
card_mirror: '877e3444b6c8c6b127ff4fa1a6d1785492d5293f' # card fields mirrored from issue-cards; edit via sdlc
---

# Cast submissions paste a short pointer; the reading rules move into a cast-submission skill

## Problem

## Spec

**Cast submissions paste a dozen paragraphs of reading rules into the agent every time (echo operator, 10-23).** A
terminal or Chrome cast DOM submission pastes the submission path plus long reading rules (crops, sketch provenance
and replay, privacy decisions, recording queries, issue-writing and asset policy) into the agent's composer. The
rules are right, but they shouldn't travel in every submission: they bloat the conversation and bury the actual ask.

Change:
- The pasted message becomes a short pointer: the submission manifest path, the recording path, and one line on what
  to read first (for example "read submission.json, then each referenced crop and sketch image; the cast-submission
  skill has the rules").
- The rules move into a skill (for example `couch-cast-submissions`, covering terminal and Chrome cast submissions),
  shipped with Couch's embedded skill set and loaded when an agent receives a submission. Its content is today's text,
  unchanged in substance (privacy and asset rules included).
- Agents that don't load the skill still have enough in the pointer to find it (its name and `couch --skill`).

## Done when

- A cast submission pastes a message of a few lines with the paths and the skill name (test against the composed text).
- The skill contains all of today's reading rules (a test that each current rule is present), and `couch --skill`
  lists it.

## Plan

- [ ]

## Log

### 2026-10-23
