---
id: 000188
status: open
deps: []
github_issue:
created: 2026-09-04
updated: 2026-09-04
estimate_hours:
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

## Spec

Discover, don't list: walk `workshop/plans/` and `workshop/history/` for any
`*-plan.md` containing a Core concepts table, and pin every row whose declared
path lives in the package under contract. A new plan then becomes enforced the
moment it declares a table — which is also the moment its rows are cheapest to
get right.

The existing `planned` status skip is what makes this tolerable: a row for work
that has not shipped is invisible until its status flips, at which point the
inventory reports it as unexpected until someone pins it. That is the signal.

**The cost is known, because it was measured before deferring** (running
discovery on `cmd/internal/couchtty` at `pair#182`'s HEAD):

- 14 rows across `pair#121`, `pair#181` and `pair#182` are unpinned — they have
  to be added to `conceptInventory` or removed from their tables.
- `CompletionQuery` / `SplitCompletionPath` and `CompletionAccumulator` are
  declared PURE but `menu_completion.go` imports `os`.
- `CompletionAccumulator` has no direct test beside its source.
- `LatestSchedule` is declared at `menu_async.go` and is absent from it.
- Two rows (`Console completion worker`, `Couch projection`) carry no backticked
  Go symbol at all, so the parser cannot check them.

Each is a real finding about a real plan. None is a defect in the code those
plans describe, which is why this is a documentation-integrity issue rather than
a bug.

Consider also: `conceptPackage` is a single package constant, so the contract
defends `couchtty` only. Whether `couchcore`'s rows deserve the same treatment is
worth deciding here rather than discovering later.

## Done when

- `findConceptPlans` discovers plans by scanning; `conceptPlans` is gone.
- Adding a plan with a Core concepts table makes the contract cover it with no
  test edit — proved by a test that a plan naming a nonexistent symbol fails.
- The 14 unpinned rows are pinned or removed, and the five assertion failures
  above are resolved at their cause (fix the row, or fix the code the row
  describes — not by loosening the assertion).
- `go test ./cmd/...` green.

## Plan

- [ ] Replace `conceptPlans` with discovery; keep the `planned` skip.
- [ ] Work the failures it surfaces, one plan at a time, deciding per row
      whether the row or the code is wrong.
- [ ] Decide whether `couchcore` gets the same contract, and record the answer.

## Log

### 2026-09-04

Found by `pair#182`'s round-10 boundary review (BR-35). The instance was fixed
there — `pair#182`'s plan added to the list, its rows pinned, its two moved rows
marked `planned — pair#186`, and `endsItsOwnChild` moved from `console.go` to
`menu.go` so a row declared PURE actually lives in a file with no IO imports.

The CLASS was deferred here deliberately. I implemented discovery first and
measured what it surfaces: a cleanup spanning three other issues' plan documents,
inside a close that was already ten review rounds deep. Widening that close is
the same mistake the `pair#186` split was made to avoid — and the fix is worth
doing properly rather than under time pressure to land a branch.
