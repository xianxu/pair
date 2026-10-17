# Boundary Review — pair#645 (whole-issue close)

| field | value |
|-------|-------|
| issue | 645 — Park fixture accepts empty client baseline as attachment readiness |
| repo | pair |
| issue file | workshop/issues/000645-park-client-count-readiness.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3fa2851bcb81207d886ed1329b67227820bbd318..41c25a6c7a9070cf880bb2afe8b508c0f3ab3c2c |
| command | sdlc close --issue 645 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-17T10:41:48-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The implementation correctly reuses the production query boundary and fixes empty-baseline false readiness. Focused tests pass, and the recorded suite matches the executable changes. One acceptance-test gap remains: attachment from a known baseline is covered only by isolated counter results, not through the helper’s readiness decision.

Done-when: current — it covers the narrowed query contract and explicitly deferred native acceptance.

```findings
findings:
  - id: new
    severity: Important
    family: readiness-contract-branch-coverage
    title: |
      Exercise successful attachment from a known baseline
    detail: |
      cmd/internal/couchcore/park_client_readiness_test.go:51-130 tests individual zero/one/two counts and the helper's unknown-baseline refusal, but never exercises its known-baseline readiness path at park_lifecycle_live_test.go:474-476. This is the sole missing branch family in this window: unchanged known counts must not establish readiness, while an added client must. Add a disposable stateful test through the helper readiness decision, covering header-only baseline, unchanged count, then an added client; the test should fail if the comparison accepts equality or never accepts growth. The native tests that otherwise exercise successful attachment were explicitly deferred.
```

1. **Strengths**
   - Both fixture observations share `ClientCountContext`; duplicated parsing is removed.
   - The subprocess regression verifies empty-baseline rejection after exactly one query.
   - CLI failures, empty/whitespace output, and zero/one/two client counts have direct controls.
   - Atlas documentation accurately describes the shared boundary.

2. **Critical findings:** None.

3. **Important findings:** The readiness-path coverage gap above.

4. **Minor findings:** None.

5. **Test coverage notes:** Focused couchcore and launcher tests passed; diff whitespace checks passed. Verified the recorded suite’s SHA256 and that subsequent changes affect only the issue document. Native acceptance remains deferred.

6. **Architectural notes**
   - **ARCH-DRY: pass** — one query/parser boundary serves inventory and fixture callers.
   - **ARCH-PURE: pass** — parsing remains pure; subprocess IO stays at the existing boundary.
   - **ARCH-PURPOSE: pass** — baseline and polling consumers both derive from the canonical implementation.
   - Atlas is updated; no new user-facing command, flag, or configuration requires a README change.

7. **Plan revision recommendations:** No contract revision needed; complete the readiness-path test before treating acceptance coverage as finished.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c32bde35-64a1-4e6e-a955-b4103ef93420","repository":"github.com/xianxu/pair","issue_id":"000645","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Users/xianxu/Library/Caches/sdlc/review-checkouts/0d0f22e229fe0805","reviewed_head":"41c25a6c7a9070cf880bb2afe8b508c0f3ab3c2c","slot":"pair:6","requested_at":"2026-10-17T17:40:45.582745Z"},"phase":"finished","pid":22247,"started_at":"2026-10-17T17:40:45.681826Z","finished_at":"2026-10-17T17:41:45.32022Z","session_id":"01a14af3-acf5-70f3-b3ba-5024b3b4f599","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.376932834s
head: "41c25a6c7a9070cf880bb2afe8b508c0f3ab3c2c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
