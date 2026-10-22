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

---

## Re-review — 2026-10-21T03:33:38-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 764 — Controlled Zellij fixture closes its PTY twice and hides cleanup errors |
| repo | pair |
| issue file | workshop/issues/000764-controlled-zellij-double-close.md |
| boundary | whole-issue close |
| milestone | — |
| window | 960eed3584667d9b9d58cdd3ba4fe7f85dda8068..78f8424c691c2d0949931bc8d93af47617ba9456 |
| command | sdlc close --issue 764 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T03:33:38-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range satisfies the cleanup contract: detach and final close retire handles through one helper, genuine failures remain observable, and callers report cleanup errors while preserving subsequent cleanup checks. BR-1’s atlas correction is complete. No new findings.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      atlas/go-lint-catalog.md:155-174 now documents the rule owner, rejected cleanup patterns, accepted reporting/propagation, and syntax-analysis limits. These match discardedControlledZellijCloses, controlledCloseErrorHandled, and pairCatalog registration. The pinned diff adds this mapping and updates atlas/index.md; this correction is prose-only.
```

1. **Strengths**

   - `live_zellij.go:121` centralizes handle retirement and retains PTY close, process signal, and wait errors.
   - `live_zellij_close_test.go:253` exercises a real child and PTY, verifies reaping, and checks exact server cleanup before cache removal.
   - All five startup refusal paths join cleanup errors. Changed callers use nonfatal test failures so remaining cleanup proof still runs.
   - The lint regression covers 28 accepted/rejected cases, including unrelated closers and the absent-else edge case.

2. **Critical findings**

   None.

3. **Important findings**

   None. The atlas gate is satisfied. No new command, flag, keybinding, configuration key, or usage step requires a README update.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed independently:

   - `go test ./cmd/internal/pairlifecycletest ./internal/repolint`
   - Both packages with `-race`
   - Compilation of affected `couchcmd`, `couchcore`, `launcher`, and `layoutcmd` packages
   - `make test-affected-inventory`
   - Pinned-range `git diff --check`

   Coverage exercises successful KillClient→Close cleanup, first-close error retention, shutdown and observation failures, and startup cleanup-error joining. Native live scenarios and the full repository suite were not independently run.

6. **Architectural notes**

   - **ARCH-DRY: pass.** Detach and final close share handle-retirement logic.
   - **ARCH-PURE: pass.** AST policy is pure; process and filesystem operations remain in the fixture boundary, with injected process observations.
   - **ARCH-PURPOSE: pass.** The lifecycle fix, caller sweep, and enforcement cover the stated cleanup purpose.
   - **ARCH-EVOLVE: pass.** No persisted schema or field interpretation changes; inventory edits retain classifications and refresh affected hashes.

7. **Plan revision recommendations**

   None. The quick plan matches the delivered scope.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"45cbe13b-7f3c-451c-b992-c839423bb5f6","repository":"github.com/xianxu/pair","issue_id":"000764","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/8a4ff2186840070d","reviewed_head":"78f8424c691c2d0949931bc8d93af47617ba9456","slot":"pair:4","requested_at":"2026-10-21T10:32:11.040132Z"},"phase":"finished","pid":94707,"started_at":"2026-10-21T10:32:11.151759Z","finished_at":"2026-10-21T10:33:37.568307Z","session_id":"01a15e04-bdd3-7ce1-828f-7b7d8b732666","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.105550958s
head: "78f8424c691c2d0949931bc8d93af47617ba9456"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
