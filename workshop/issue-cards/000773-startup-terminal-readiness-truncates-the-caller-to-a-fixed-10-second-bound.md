---
id: 000773
status: open
created: 2026-10-21
updated: 2026-10-21
estimate_hours:
github_issue:
severity: high
---

# startup: terminal readiness truncates the caller to a fixed 10-second bound

## Problem

Production terminal readiness imposes a fixed10s deadline even when its caller has a longer remaining deadline. A cold native agent launch under load can be refused before the caller budget expires. Discovered by #744 final native matrix, independently from #772's attach InputEnded failure. This is a startup-budget design defect; do not classify it away as a flake or increase a numeric timeout.
