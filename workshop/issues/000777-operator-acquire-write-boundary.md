---
id: 000777
status: open
deps: []
github_issue:
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
collaboration-mode: collaboration
card_mirror: '063f6abc240ed7572c91b448fb8c66b97fbaa9d6' # card fields mirrored from issue-cards; edit via sdlc
---

# session-sync: serialize operator acquisition with the writer commit boundary

## Problem

The operator requested closing the residual InsertEnter race after parley.nvim#404. Auditing cmd/internal/sessionsync/sync.go shows f.guard at write entry, preparation and replaceShared later, and only a post-write sidecar inspection that emits a warning. The source explicitly notes that guard and write cannot be one step against Parley, which does not take the Couch flock. An agent writer that passed the guard can commit after the operator creates the lock.

## Spec

Define and implement a single exclusion resource/linearization boundary shared by registered operator acquisition and session-sync document writes, so an acknowledged acquisition excludes earlier in-flight writes and future writes observe held protection before committing. A repeated sidecar check without shared exclusion is insufficient. Coordinate the acquisition/acknowledgment contract with Parley's InsertEnter follow-up. Preserve lock ownership, CAS, journal/recovery behavior and the ordinary registered write path; no two-phase protocol.

## Done when

- A deterministic writer pause between guard and mutation races operator acquisition and proves no writer commit crosses an acknowledged operator lock.
- Registered write/apply paths and migration protection use the intended shared exclusion contract; refusal leaves document and metadata consistent.
- Consumer-facing acquisition/acknowledgment semantics and failure recovery are documented and communicated to parley.nvim:1.

## Plan

- [ ]

## Log

Filed by parley.nvim:1 for its TL after read-only producer audit; unclaimed for producer dispatch. Operator request is Couch 61805ff7-ac98-4fcf-9a32-9a8215d5b878. Existing write path: cmd/internal/sessionsync/sync.go around lines 617, 651 and 671. parley.nvim#404 landed PR #327; its residual-race follow-up is separate.
