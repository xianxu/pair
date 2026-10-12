# Boundary Review — pair#481 (whole-issue close)

| field | value |
|-------|-------|
| issue | 481 — tests: gate wall-clock perf budgets out of the default suite (terminal tail 250ms, slot-git render 100ms, broadcast marks) |
| repo | pair |
| issue file | workshop/issues/000481-tests-gate-wall-clock-perf-budgets-out-of-the-default-suite-terminal-tail-250ms-slot-git-render-100ms-broadcast-marks.md |
| boundary | whole-issue close |
| milestone | — |
| window | 97478ad961f162c25c10eb2f1018f122c5c77441..b1262c35fe7a275515476a09a23b4af6db78da38 |
| command | sdlc close --issue 481 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T08:36:42-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

Diff and docs are read. Next I'll run the tests for the changed packages and check one stale atlas enumeration.

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

All four wall-clock budgets named in the issue are gone from the default suite, and the one budget the plan keeps (the terminal tail) now runs only under `PAIR_PERF_BUDGETS` through `make test-perf-budgets`. Each test now checks a result instead of a time: the tail's 200 lines, a render or offer finishing while the probe or hub loop is held, and the marks walk's visit count. The new `perfbudget` package is now the only place the switch lives; couchcore's private `perfBudgetsEnforced` was deleted and its two callers use `perfbudget.Check`. `walkStrokes` was pulled out of `Marks.Add`, and the refactor keeps its behaviour. The only thing worth fixing before close: the atlas sentence that lists the budget tests is now out of date.

**Inspection notes:** I ran the stat, name-status and full diff recipes against the pinned range. `go vet` passed. `go test` passed for `perfbudget` and for the broadcast `TestMarks*` and `TestHubOfferNeverBlocks` tests. I could not build the `couchtty`, `wrapcmd` or `couchcore` test binaries in this checkout. The generated `cmd/internal/runtimebundle/assets/runtime/files` tree is missing here, so `embed.go:8` fails, and the same gap causes `artifactpath`'s STALE failures. That is a checkout-environment problem, not something this diff causes: the new manifest entry is not reported as unclassified. So the rewritten couchtty and wrapcmd tests were reviewed by reading them, not by running them.

1. **Strengths**
   - `cmd/internal/perfbudget/perfbudget.go:24`: `Check` always logs and fails only when the budget is enforced. Its test goes through all four cases (unset, `1` over budget, `1` under budget, `yes`) using a `testing.TB` recorder, with no IO beyond `t.Setenv`.
   - `cmd/internal/broadcast/marks.go:172`: extracting `walkStrokes` gives the marks test a real cost to count. The cap truncation now runs per cell instead of per point. The final `batch[len-limit:]` is unchanged, because truncation always keeps at least `limit` of the newest cells.
   - `cmd/internal/broadcast/marks_test.go:270`: the huge-coordinate test now checks the exact cells visited, `{0,0},{2,0}`. That is stronger than the old 50 ms check.
   - `hub_test.go:283` gained `defer close(release)`, so the stalled loop is released even when the test fails. Before this change, a fatal failure left the loop stalled.
   - `Makefile.local:121` runs each budget on its own (one `-run` per package invocation), matching the plan's "in isolation".

2. **Critical findings:** none.

3. **Important findings**
   - `atlas/couch.md:2579`: the sentence still names only `TestAcceptanceObserveBudget` and `TestAcceptanceReportBudgetTenSlots` as the clock-gated budgets, cites only pair#436, and doesn't mention `cmd/internal/perfbudget`. `TestTailOverEndpointSocketIsFast` is now also run by `make test-perf-budgets`, and the switch is a shared package. Fix: change the sentence to say budgets go through `perfbudget.Check` and are listed in the `test-perf-budgets` target, rather than listing test names (the list is what went stale). Family in this window: this is the only doc that lists the budget tests. `Makefile.local:116` was already updated.

4. **Minor findings**
   - `perfbudget.Enforced` is exported but nothing outside the package calls it. Either keep it as intentional API or unexport it.
   - The plan says couchcore's `perfBudgetsEnforced` "routes through" `perfbudget`. In fact it was deleted, and its callers call `Check` directly. The wording is off, but the delivered result is better.
   - If either hang guard (`hub_test.go`, `console_slotgit_test.go`) fires, its goroutine leaks. That only happens on failure, so it's acceptable.

5. **Test coverage notes**
   - The `## Done when` clause has two modes, and both are covered. With the variable unset, nothing fails (the default lane, plus `perfbudget_test` case 1). With `=1`, the check fails (case 2) and the Make target sets it.
   - The marks bound `len(points)*(max(cols,rows)+1)` is correct for Bresenham, which visits at most max(dx,dy)+1 cells per segment.

6. **Architecture**
   - **ARCH-DRY — pass.** There is now one env switch. `sessioninventory`'s `PAIR_LIVE_SESSION_INVENTORY` is a different, live-host gate, so it isn't a duplicate.
   - **ARCH-PURE — pass.** `walkStrokes` is a pure function, tested directly with no clock.
   - **ARCH-PURPOSE — pass.** All four listed sites were swept, and couchcore's existing budgets were moved onto the shared switch too. The `waitUpTo` windows were explicitly left to #479.

7. **Plan revision recommendations:** optionally add a `## Revisions` note that `perfBudgetsEnforced` was removed rather than wrapped.

```findings
findings:
  - id: new
    severity: Important
    family: docs-enumeration-drifts-from-source
    title: |
      atlas/couch.md:2579 enumerates only the couchcore budgets for make test-perf-budgets
    detail: |
      The tail budget (wrapcmd TestTailOverEndpointSocketIsFast) and the shared cmd/internal/perfbudget switch (pair#481) are missing. Point the atlas at perfbudget.Check and the Makefile target instead of naming tests. It is the only stale enumeration in this window.
  - id: new
    severity: Minor
    family: unused-exported-surface
    title: |
      perfbudget.Enforced is exported but has no caller outside the package
    detail: |
      Keep it as deliberate API or unexport it to keep Check the single entry point.
```

---

## Re-review — 2026-10-12T08:40:43-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 481 — tests: gate wall-clock perf budgets out of the default suite (terminal tail 250ms, slot-git render 100ms, broadcast marks) |
| repo | pair |
| issue file | workshop/issues/000481-tests-gate-wall-clock-perf-budgets-out-of-the-default-suite-terminal-tail-250ms-slot-git-render-100ms-broadcast-marks.md |
| boundary | whole-issue close |
| milestone | — |
| window | 97478ad961f162c25c10eb2f1018f122c5c77441..c3496125f19c3f5e9b7d4a67c5a2f0d96c89cdaf |
| command | sdlc close --issue 481 |
| reviewer | claude |
| isolation | fenced |
| timestamp | 2026-10-12T08:40:43-07:00 |
| verdict | SHIP |

## Review

Vetting failed on a missing embedded asset in this checkout. I'm narrowing the test run to the packages it doesn't affect.

```verdict
verdict: SHIP
confidence: medium
```

The change does what the issue asks. Each listed test now checks its result instead of a clock. The slot-git render and hub-offer tests use a hang guard that waits while the probe or loop is held. The marks tests count cells with a new `walkStrokes` helper, which `Add` now calls, so the count measures the same walk `Add` does. The terminal tail keeps its 250 ms budget, but it is only enforced under `PAIR_PERF_BUDGETS=1`, and it was added to `make test-perf-budgets`. couchcore's private `perfBudgetsEnforced` was removed and its tests now call the shared `perfbudget.Check`. BR-1 is fixed: the atlas now points at `perfbudget.Check` and the make target, and includes the tail. BR-2 is still open, but it is Minor and does not block. I ran `go test` for perfbudget, broadcast and couchcore, and they pass. couchtty and wrapcmd would not build in this review checkout because a generated embed asset is missing (`runtimebundle/embed.go: assets/runtime/files`). This diff did not cause that. I read the changes to those two tests instead of running them, which is why confidence is medium.

1. **Strengths**
   - `cmd/internal/broadcast/marks.go:172-193`: the extracted `walkStrokes` is the code `Add` actually runs, so the operation-count test checks real behaviour, not a copy of it. Checking the cap after every batch (`marks_test.go:260-264`) is stronger than the old single check at the end.
   - `cmd/internal/broadcast/hub_test.go:283`: `defer close(release)` releases the stalled loop even when the test fails, so a failure doesn't leave a goroutine stuck.
   - `cmd/internal/perfbudget/perfbudget_test.go`: tests the env switch in both states (`""` and `"1"`), plus under-budget and a non-`1` value. Both modes of the Done-when are exercised.
   - `Makefile.local:120-121`: each budget runs in its own `go test` call, so the budgets are measured in isolation as the plan says.
   - `artifactpath/manifest.go:726-728`: the new package is registered as a non-artifact source.

2. **Critical:** none.

3. **Important:** none.

4. **Minor**
   - BR-2 is still open: `perfbudget.Enforced` has no caller outside its package. `perfbudget.Env` is the same, only used by the package's own test. See the disposition below.
   - `atlas/couch.md:2584`: the bracketed list "(couchcore's observe and ten-slot report budgets, and wrapcmd's terminal tail)" is accurate today. But it repeats the Makefile list that the same sentence calls "the one list". Deleting it leaves the atlas with nothing to drift. Folded into BR-1's note rather than raised as a new finding.

5. **Test coverage**
   - Lower bound in the marks test (`visits >= len(points)`): holds, because Bresenham includes both endpoints and every point in the fixture is on the grid.
   - Upper bound (`len(points)*(max(cols,rows)+1)`): holds, because a line between two on-grid points visits at most `max(dx,dy)+1` cells.
   - Huge-coordinate test: checks the exact walk `[{0,0},{2,0}]`, which pins the rule that an off-grid point breaks the stroke.
   - Other default-lane timing checks remain in broadcast (`cloudflared_test.go:216` 3 s, `session_test.go:307` 5 s). They are generous limits that work as hang guards, they are not on the issue's list, and they don't contradict the atlas rule.

6. **Architecture**
   - **ARCH-DRY: pass.** couchcore's private switch was folded into `perfbudget`, and the stroke walk exists in one place. The hang-guard `select` appears twice, in different packages; two small copies don't call for a helper.
   - **ARCH-PURE: pass.** `walkStrokes` is pure and is tested directly. `perfbudget` only reads the env var.
   - **ARCH-PURPOSE: pass.** Every test on the issue's list was converted, and every `PAIR_PERF_BUDGETS` user goes through `perfbudget.Check`. The Makefile is the only place that lists the budgets, apart from the atlas's bracketed list noted above.

7. **Plan revisions:** none needed. One stale line in the Problem section: it cites "sessioninventory budgets" as precedent, but no such use of `PAIR_PERF_BUDGETS` exists in the tree. It's historical text, not a claim the plan makes.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      atlas/couch.md:2579-2585 now cites perfbudget.Check, cmd/internal/perfbudget and make test-perf-budgets as the list, and includes the wrapcmd tail. Remaining nit: the bracketed list at :2584 repeats the Makefile recipe; dropping it removes the drift risk. Rule for the family: docs point at the single source and do not restate it.
  - id: BR-2
    disposition: not-addressed
    note: |
      perfbudget.go:20 Enforced is still exported with no caller outside the package. Env at :17 is the same, used only by perfbudget_test.go, which is in-package. Rule: export only what has an out-of-package caller. Unexport both (enforced, env) or record that the API is deliberate.
```
