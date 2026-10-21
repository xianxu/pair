# Boundary Review — pair#764 (whole-issue close)

| field | value |
|-------|-------|
| issue | 764 — Controlled Zellij fixture closes its PTY twice and hides cleanup errors |
| repo | pair |
| issue file | workshop/issues/000764-controlled-zellij-double-close.md |
| boundary | whole-issue close |
| milestone | — |
| window | 960eed3584667d9b9d58cdd3ba4fe7f85dda8068..7455aedf23348fe760356aeeff7e9add60588e8a |
| command | sdlc close --issue 764 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T03:30:20-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The lifecycle fix satisfies the issue’s behavioral requirements: detach and final cleanup share handle retirement, genuine errors remain visible, and callers report cleanup failures. Focused tests pass, including under the race detector. One inexpensive atlas update remains before closing the boundary.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: atlas-surface-documentation
    title: |
      Atlas update appears missing for the controlled-zellij-close lint rule
    detail: |
      internal/repolint/local_catalog_test.go:32 introduces the controlled-zellij-close enforcement rule, but this range contains no atlas changes. atlas/go-lint-catalog.md:139-144 explicitly requires an atlas mapping for Pair-specific ownership policies. The complete new-surface family in this window is this one rule, implemented by discardedControlledZellijCloses and controlledCloseErrorHandled. Add its owner path, rejected cleanup patterns, accepted reporting/propagation patterns, and documented syntax-analysis limits to the catalog guide.
```

1. **Strengths**

   - [live_zellij.go:121](cmd/internal/pairlifecycletest/live_zellij.go:121) centralizes handle retirement and preserves close, signal, and wait errors.
   - [live_zellij_close_test.go:253](cmd/internal/pairlifecycletest/live_zellij_close_test.go:253) exercises a real child and PTY, verifies reaping, and checks exact server evidence before root removal.
   - Caller changes use nonfatal test errors, preserving subsequent cleanup proof. All five startup refusal paths join cleanup errors.

2. **Critical findings**

   None.

3. **Important findings**

   The atlas mapping above is missing. No README update is required: this range introduces no command, flag, keybinding, or configuration key.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed:

   - Both `pairlifecycletest` and `repolint` packages, normally and with `-race`.
   - Compilation of affected `couchcmd`, `couchcore`, `launcher`, and `layoutcmd` packages.
   - `make test-affected-inventory` and pinned-range `git diff --check`.

   Coverage includes double-close prevention, first-close error preservation, shutdown/observation failures, startup error joining, and 28 lint cases. Native live scenarios were not independently run.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Detach and close reuse one handle-consumption helper; lint reuses the composed scanner.
   - **ARCH-PURE: pass.** AST policy remains pure; process and PTY operations stay within the fixture’s IO boundary.
   - **ARCH-PURPOSE: pass.** The fix and caller sweep cover the stated cleanup purpose.
   - **ARCH-EVOLVE: pass.** No persisted schema or interpretation changes; inventory edits update existing hashes.

7. **Plan revision recommendations**

   No substantive revision needed. Add the missing atlas mapping before recording completion.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f1ba071b-f79e-4578-bff6-507602a176a0","repository":"github.com/xianxu/pair","issue_id":"000764","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/964a2817e3b46a10","reviewed_head":"7455aedf23348fe760356aeeff7e9add60588e8a","slot":"pair:4","requested_at":"2026-10-21T10:28:24.121565Z"},"phase":"finished","pid":14993,"started_at":"2026-10-21T10:28:24.311546Z","finished_at":"2026-10-21T10:30:18.220608Z","session_id":"01a15e01-4809-72b1-ac47-ac2643ceb46d","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.823049917s
head: "7455aedf23348fe760356aeeff7e9add60588e8a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
