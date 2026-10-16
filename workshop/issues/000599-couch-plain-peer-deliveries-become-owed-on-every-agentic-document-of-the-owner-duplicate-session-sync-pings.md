---
id: 000599
status: open
deps: []
github_issue:
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
card_mirror: 'e88ea063b0d0ed65e03063d93c61c2cf3ca0f2c3' # card fields mirrored from issue-cards; edit via sdlc
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

- [ ]

## Log

### 2026-10-15
