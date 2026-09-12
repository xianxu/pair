---
id: 000235
status: open
deps: []
github_issue:
created: 2026-09-12
updated: 2026-09-12
estimate_hours:
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

## Spec

A check wired into `make test`, beside `tests/plan-superseded-facts-test.sh`,
reads each active plan's Core concepts tables: the rows under a
`| Name | Lives in | Status |` header, with name and path taken from the
backticked cells. For each row it asserts:
- the name is declared in the file the row names, as a Go identifier's
  declaration in that file;
- a `modified` row names a file changed in the plan's issue window, and a
  `new` row names a file created in it.

Rows whose name is not a Go identifier are skipped and counted in the output,
so the check's reach is visible. Archived plans in `workshop/history` are out
of scope.

## Done when

- A Core concepts row naming an entity its file does not declare fails
  `make test`, citing the plan and line.
- A `modified` row whose file is unchanged in the plan's window fails, and so
  does a `new` row whose file existed before the window.
- Every active plan passes, or its rows are corrected in the same change.

## Plan

- [ ] Parse the Core concepts tables: rows under the header, name and path
      from the backticked cells.
- [ ] The declaration check: the name is declared in the named file.
- [ ] The window check: `modified` and `new` rows against the plan's issue
      window.
- [ ] Wire it into `make test`, and correct any active plan it fails.

## Log

### 2026-09-12

- Filed from #206's issue-close review, the third finding in the plan-drift
  family. The #206 plan's Revisions entry for the issue close describes the
  three rows it found.
