---
id: 000680
status: open
created: 2026-10-18
updated: 2026-10-18
estimate_hours:
github_issue:
---

# pair doctor resume: daily read-only check that every slot's resume target resolves on real data

## Problem

pair#676: every Claude resume refused once ~/.claude/projects crossed 10,000 entries (a discovery cap reached resume through #556). Small test fixtures could not show it; only a run against the real tree did (pair:5 proved the fix with a read-only conformance run, ~0.9 s per slot). Nothing checks resume against real data routinely, so the operator found it by a failed relaunch.
