---
id: 000567
status: open
created: 2026-10-15
updated: 2026-10-15
estimate_hours:
github_issue:
---

# Lower attachment and owner-pass process-table scans to one

## Problem

#204 establishes strict selected Couch attachment <=1 full process-table scan (parent0 + helper1), but the other paths still enumerate repeatedly. Existing #218 owner-pass sharing counts only its initial capture; each live fence also enumerates a full table.
