---
id: 000599
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
---

# Couch: plain peer deliveries become owed on every agentic document of the owner (duplicate session-sync pings)

## Problem

Since the 10-15 evening Couch restart (pair#573, agentic documents), every peer delivery to ops:0 becomes an owed
event on **every** registered document ops:0 owns, not only the document it concerns. Example: worker reports
b9c9305a and 4e51644c (about pair#597 and #586) were owed on both the TL session file and the QA file
`ops/teams/devinfra/qa/xian-xu/2026-10-15-39-why-reboot-refuses.md`, and Couch pinged "owes a response" for each file
separately. The TL has to fold the same IDs once per file; with several agentic documents open, pings multiply.
