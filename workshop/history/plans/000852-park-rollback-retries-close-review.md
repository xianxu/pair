# Boundary Review — pair#852 (whole-issue close)

| field | value |
|-------|-------|
| issue | 852 — Park rollback retries fail on main |
| repo | pair |
| issue file | workshop/issues/000852-park-rollback-retries.md |
| boundary | whole-issue close |
| milestone | — |
| window | e5015984cf765d1dc34686e37c9821ec106351fc..44cde4961c6e9a3856a578254eac887cecdafaf6 |
| command | sdlc close --issue 852 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T18:27:04-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The fixture correction matches the revised issue contract. It removes incidental wall-clock expiration from drive-count tests while preserving cancellation, missing-proof failure, and budget-exhaustion controls. Production behavior is unchanged. No blocking findings.

Done-when: current — the existing clauses still cover root-cause evidence, `/15` and `/30` outcomes, and race/affected validation.

```findings
{}
```

1. **Strengths**
   - Existing deadline seams control fixture timing without changing production policy (`cmd/internal/couchcore/leave_park_test.go:249`).
   - Tiny-budget and withheld-proof controls exercise both successful completion and bounded failure (`:338`, `:355`).
   - Original exact drive, trigger, ordering, and concurrency assertions remain intact (`:288`).

2. **Critical findings:** None.

3. **Important findings:** None. No new architectural or user-facing surface requires atlas or README changes.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Focused LeavePark, retry-invariant, and joined-future suite passed: 125.399s.
   - Same suite with `-race` passed: 54.262s.
   - Reader inventory and diff whitespace checks passed.
   - Inspected retained finalization traces supporting the documented cause. No mutation-based red check was performed during this read-only review.

6. **Architectural notes**
   - **ARCH-DRY: pass** — reuses existing fixture and injection seams.
   - **ARCH-PURE: pass** — IO remains in the integration fixture; retry policy remains pure.
   - **ARCH-PURPOSE: pass** — preserves the intended fresh/recovery contract and exercises its failure controls.
   - **ARCH-EVOLVE: pass** — no persisted-format changes.

7. **Plan revision recommendations:** None. The existing revision accurately describes the fixture-only correction and outstanding downstream gates.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"aa23928c-8a74-4b04-b156-97bbb303a07b","repository":"github.com/xianxu/pair","issue_id":"000852","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/76015ad3a264c0b3","reviewed_head":"44cde4961c6e9a3856a578254eac887cecdafaf6","slot":"pair:9","requested_at":"2026-10-24T01:22:52.72265Z"},"phase":"finished","pid":65338,"started_at":"2026-10-24T01:22:52.894598Z","finished_at":"2026-10-24T01:27:02.932134Z","session_id":"01a16b80-eafc-7cc3-ba3c-3dc9e417c8ce","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.867270583s
head: "44cde4961c6e9a3856a578254eac887cecdafaf6"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
