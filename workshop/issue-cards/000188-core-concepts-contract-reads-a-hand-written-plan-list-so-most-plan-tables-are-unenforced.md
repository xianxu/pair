---
id: 000188
status: open
created: 2026-09-04
updated: 2026-09-04
estimate_hours:
github_issue:
---

# Core-concepts contract reads a hand-written plan list, so most plan tables are unenforced

## Problem

`TestCoreConceptsContract` turns a plan's **Core concepts** table into an
executable contract: rows must name real symbols at real paths, PURE sources may
not import IO seams, deleted symbols must be absent. It is a good guard. Its
INPUT is `conceptPlans`, a hand-written list of two plan filenames.

So a plan absent from that list has its entire table unenforced — and the
failure is invisible, because the guard passes. `pair#182`'s plan was absent and
shipped two rows naming `paneState` and `RenderHoldingPane` in
`cmd/internal/couchtty/holding.go`: no such symbols, no such file. Found by a
boundary reviewer, not by the test written to find exactly this.

This is the tenth instance in one issue's review history of a single shape: **a
guard whose input is a hand-maintained list is a guard the next addition skips.**
The others were fixed at the class (`OperationConfirms`, `AllInterceptorHits`,
`Operation.RowAction`, the README guard scoped to the couch section). This one
was measured and deferred rather than fixed, for the reason in `## Log`.
