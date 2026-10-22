---
id: 000795
status: open
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
github_issue:
---

# Settle remote submit evidence before exact-ID forget

## Problem

During pair#759 M3 integration, exact-ID forget settlement could not be proved from retained receipts: broker receipts expire, ledger trimming removes old uncertain rows, and audit retention removes old unmatched intents. An empty bounded scan is not proof an old potentially committed submit settled. TL split this mechanism after its third refinement (delivery d789d04d); M3 conservatively blocks forget for documents with remote or marked-local submits.
