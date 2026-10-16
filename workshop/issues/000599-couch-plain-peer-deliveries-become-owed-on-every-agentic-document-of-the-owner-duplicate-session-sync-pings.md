---
id: 000599
status: working
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-16
estimate_hours:
card_mirror: 'e8e7b10d923c0446fa8e47a36e0ab15f70bfa634' # card fields mirrored from issue-cards; edit via sdlc
started: 2026-10-16T08:22:54-07:00
claimant:
    operator: Xian Xu
    machine: 4716879978a7b90f6b583da1716fd0e9
    machine_name: Xian’s MacBook Pro
    workspace: pair:2
    worktree: /Users/xianxu/workspace/worktree/pair-slot2/pair
    repository: github.com/xianxu/pair
---

# Couch: plain peer deliveries become owed on every agentic document of the owner (duplicate session-sync pings)

## Problem

Since the 10-15 evening Couch restart (pair#573, agentic documents), every peer delivery to ops:0 becomes an owed
event on **every** registered document ops:0 owns, not only the document it concerns. Example: worker reports
b9c9305a and 4e51644c (about pair#597 and #586) were owed on both the TL session file and the QA file
`ops/teams/devinfra/qa/xian-xu/2026-10-15-39-why-reboot-refuses.md`, and Couch pinged "owes a response" for each file
separately. The TL has to fold the same IDs once per file; with several agentic documents open, pings multiply.

## Spec

- An owed event belongs to the document it was submitted for: a `submitted: PATH` delivery owes on PATH only.
- A plain peer delivery (a worker report with no document) owes on the owner's **session file** only (the one
  registered as the owner's session-sync file), never on other agentic documents.
- Folding an ID on one document doesn't require folding it on others.

## Done when

- Tests: a document submit owes only on its PATH; a plain delivery owes only on the session file; two agentic documents
  of one owner don't duplicate owed events or pings.

## Plan

- [x] `sessionsync.OwedForDocument` takes whether the document owes plain deliveries; `Sync.owed` passes `IsSessionSync(content)`, so only the owner's `type: session-sync` file owes them. Submits stay owed by the document they name.
- [x] Tests: unit (non-session document owes only its submits) and Sync-level (session file plus registered plain document of one owner: delivery owed once, submit only on its document); fold-size test moved to submits; atlas Owed rule.

## Log

### 2026-10-15

### 2026-10-16 — fix (pair:2)

Cause: `OwedForDocument` owed every plain delivery on every session-sync document of the owner by design ("each folds it for itself"); #573 made plain registered markdown such documents, so each delivery was owed and pinged once per document. RED test reproduced it (the QA document owed the worker report). Fix keys plain deliveries to the owner's `type: session-sync` file; status and Parley pings follow from Sync.owed.
