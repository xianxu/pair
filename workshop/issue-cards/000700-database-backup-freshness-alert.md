---
id: 000700
status: open
created: 2026-10-19
updated: 2026-10-19
estimate_hours:
github_issue:
---

# Warn when a database backup is older than its freshness threshold

## Problem

#688 found thread_history_1.sqlite had no verified bundle for roughly 55 hours while continuous passes could finish with database skips. A pass-completion marker did not expose individual database freshness.
