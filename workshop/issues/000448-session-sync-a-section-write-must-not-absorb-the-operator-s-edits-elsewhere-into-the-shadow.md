---
id: 000448
status: open
deps: []
github_issue:
created: 2026-10-11
updated: 2026-10-11
estimate_hours:
card_mirror: '3e0629762fd0e785bbe587f5d9ce2c501116f5ae' # card fields mirrored from issue-cards; edit via sdlc
---

# session-sync: a section write must not absorb the operator's edits elsewhere into the shadow

## Problem

Since pair#439, a section write succeeds while the operator has edits in other sections (by design). But the write then refreshes the shadow copy from the whole file, so those other edits become part of the shadow and `couch --session-sync diff` no longer shows them. Seen 10-11 12:25 on the ops status file: the operator added `🤖[done]` under Needs you and submitted; the TL's Threads write landed first, and the following `diff` was empty. The TL found the marker only by grepping. A marker the agent never sees goes unanswered.

## Spec

The shadow means "the file as the agent last saw it". A section write updates only that section in the shadow, leaving every other section as it was, so `diff` keeps showing the operator's unread edits elsewhere. A `write --all` with a matching whole-file hash replaces the whole shadow (the agent read everything). Folding a submit doesn't change the shadow; reading the diff is the agent's job, and an `--accept-edits`-style acknowledgement isn't reintroduced. Decide in the design whether `read --all` should also refresh the shadow (probably yes: it is the agent seeing the file).

## Done when

- Operator edits section A; the agent writes section B; `diff` still shows the edit to A (test).
- `write --all` with the right hash leaves `diff` empty (test).

## Plan

- [ ] Section-scoped shadow update; tests for the Done-when rows; atlas couch.md.

## Log

### 2026-10-11
