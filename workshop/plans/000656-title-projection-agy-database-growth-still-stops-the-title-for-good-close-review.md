# Boundary Review — pair#656 (whole-issue close)

| field | value |
|-------|-------|
| issue | 656 — Title projection: agy database growth still stops the title for good |
| repo | pair |
| issue file | workshop/issues/000656-title-projection-agy-database-growth-still-stops-the-title-for-good.md |
| boundary | whole-issue close |
| milestone | — |
| window | 479b91376e5055f8106fe95b34bc64c71c6d2bc6..5afa91e728b992b16d112cc49d2fb8aba39c6efc |
| command | sdlc close --issue 656 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-18T05:57:48-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

Normal Agy database growth now recovers through the shared projection, and the existing tests pass. However, growth can hide concurrent transcript rewrites or artifact replacements, violating the explicit requirement that mutations remain terminal.

Done-when: current — it covers recovery, mutation precedence, authority revalidation, cancellation, and counted work; the approval wording change introduces no uncovered scope.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: mutation-precedence-before-readmission
    title: |
      Database growth bypasses terminal mutation checks during reads
    detail: |
      cmd/internal/sessioninventory/title_projection.go:399 compares the transcript only against p.transcript.Size. After an append, a same-size rewrite during the boundary or suffix read escapes this comparison, while errTitleReadmit skips the paired fingerprint checks at lines 446 and 461. Both interleavings stop without database growth but return nil with growth, as confirmed by temporary overlay regressions. The other instances in this family are transcript and database replacement during the new retained-boundary read at line 403: neither is re-statted before line 411 returns transient recovery, and both overlay regressions fail. Preserve the paired fingerprint checks before transient classification, fence the added boundary read with artifact metadata checks, and add regressions for all four cases. ARCH-PURPOSE: growth must not hide the mutation class the Spec explicitly requires to remain terminal.
```

1. **Strengths**
   - Recovery reuses the existing reset and full validator without adding consumer-specific recovery.
   - Actual SQLite tests verify Couch source reuse, activity recovery, steady-neighbor behavior, and unchanged-poll costs.
   - Atlas updates describe readmission accurately; no new command, flag, or configuration requires a README update.

2. **Critical findings**
   - The finding above blocks shipping. Complete mutation checks before returning `errTitleReadmit`.

3. **Important findings**
   - None separately.

4. **Minor findings**
   - None.

5. **Test coverage**
   - Passed all `sessioninventory` and `titlepoller` tests and the actual-SQLite Couch recovery matrix.
   - Confirmed the existing growth regression fails against base production code.
   - Four temporary overlay race regressions fail at head; both rewrite controls without database growth pass. Temporary files were removed; tracked files remain unchanged.
   - `git diff --check` passed.

6. **Architecture**
   - **ARCH-DRY: pass** — shared predicates, reset, and validation are reused.
   - **ARCH-PURE: pass** — added artifact IO stays behind the injected Runtime seam.
   - **ARCH-PURPOSE: flag** — ordinary recovery works, but concurrent mutation precedence is incomplete.

7. **Plan revision recommendation**
   - Add a `## Revisions` entry enumerating boundary-read and suffix-read rewrites plus transcript/database replacement during the recovery boundary read. Require terminal results for all four before marking the mutation-precedence work complete.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c51cfd62-3f66-4aa1-b876-95659cbbf09a","repository":"github.com/xianxu/pair","issue_id":"000656","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/5e1e913eea8bea39","reviewed_head":"5afa91e728b992b16d112cc49d2fb8aba39c6efc","slot":"pair:6","requested_at":"2026-10-18T12:55:45.872106Z"},"phase":"finished","pid":21603,"started_at":"2026-10-18T12:55:45.969015Z","finished_at":"2026-10-18T12:57:38.600546Z","session_id":"01a14f15-1d17-72f3-b357-a0ce9aa44cce","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.565209917s
head: "5afa91e728b992b16d112cc49d2fb8aba39c6efc"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
