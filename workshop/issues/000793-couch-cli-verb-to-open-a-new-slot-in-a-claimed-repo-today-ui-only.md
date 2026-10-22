---
id: 000793
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '92ce5c0dd85b4aa9fa441501e519bcc7f6b009f9' # card fields mirrored from issue-cards; edit via sdlc
---

# Couch: CLI verb to open a new slot in a claimed repo (today UI-only)

## Problem

## Spec

10-21: the operator asked the TL to start tools:1 and lend it to another hub. Couch's CLI can restart, relaunch,
reboot, prepare, switch or move an existing slot, but has no verb to **start a new slot** in a claimed repo; it's
UI-only, so the TL had to ask the operator to click. Agent-first operations: Couch ops should be CLI verbs the TL
runs, with UI as an optional follow-up.

Fix: `couch --open-slot repo:N --agent NAME [--confirm]`, hub-only, for a repo the calling hub claims: creates or
attaches the slot's worktree via the same path as the UI's new-slot action, launches the agent fresh, and returns a
receipt (`--operation-status`). Refuses if the slot is live or the repo isn't claimed by the caller.

## Done when

- A hub opens a new slot in a claimed repo from the CLI, with the same result as the UI action (test with fake launcher).
- Refusals: slot already live, repo claimed by another hub, unknown agent (tests).

## Plan

- [ ]

## Log

### 2026-10-21
