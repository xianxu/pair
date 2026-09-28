---
id: 000235
status: open
created: 2026-09-12
updated: 2026-09-12
estimate_hours:
github_issue:
---

# Check a plan's Core concepts tables against the code

## Problem

#206's issue-close review found the plan's Core concepts tables asserting
things nothing checks:
- a function that never changed was marked `modified`;
- a function was placed in the wrong file;
- a row named a file the plumbing had since left.

It was the third finding in the plan-drift family in one issue. The first two
were fixed as rules: a restated count or list points at its one home, and a
restated declaration points at the code. A table row that names an entity at
a path, with a status, is a claim of a different shape, and no test reads it.
