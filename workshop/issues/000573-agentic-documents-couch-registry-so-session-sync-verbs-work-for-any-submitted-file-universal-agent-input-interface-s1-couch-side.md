---
id: 000573
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'd62d5d18f5cfda8ff0a71ef13574314b46ddb00e' # card fields mirrored from issue-cards; edit via sdlc
---

# Agentic documents: Couch registry so session-sync verbs work for any submitted file (universal-agent-input-interface S1, Couch side)

## Problem

Project universal-agent-input-interface (ops `teams/devinfra/projects/universal-agent-input-interface/design.md`,
slice S1): Couch's session-sync verbs (`read`, `diff`, `write`, `status`, the owed-event ledger, the CAS write and the
operator-lock refusal) work only for files with `type: session-sync` frontmatter. The operator wants every markdown
file submitted from Couch's right-pane nvim to become an **agentic document** with the same guarantees, without
writing anything into the file.

## Spec

- A Couch-side **agentic registry** keyed by canonical path: `couch --session-sync register FILE --owner SLOT` (called
  by parley on the first submit), `couch --session-sync forget FILE`, and listing. Agentic status lasts until forgotten.
- All `--session-sync` verbs accept a registered file without `type: session-sync`; the owner comes from the registry
  (or an explicit `couch-owner:`, which wins).
- The owed-event ledger works for registered files; the submit delivery names the file.
- **Forgotten lock:** if the agent needs to write and the operator has held the lock for more than 30 minutes, the
  write refusal says so (lock age), so the agent can ping the operator (or raise a Needs you item for the TL's file).
- A path move or rename updates the registry entry.

## Done when

- Tests: register, then read/diff/write/status on a file with no frontmatter; owed events; lock-age in the refusal
  after 30 minutes (fake clock); forget; rename. Existing session-sync files are unaffected.
- `couch --skill` documents the registry and the 30-minute rule.

## Plan

- [ ]

## Log

### 2026-10-15
