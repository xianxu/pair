# Boundary Review — pair#725 (whole-issue close)

| field | value |
|-------|-------|
| issue | 725 — pair affected-only tests: .sdlc/affected.json + make test-affected, per ariadne#473's settled contract |
| repo | pair |
| issue file | workshop/issues/000725-pair-affected-only-tests-sdlc-affected-json-make-test-affected-per-ariadne-473-s-settled-contract.md |
| boundary | whole-issue close |
| milestone | — |
| window | ab32acea6c97663232623b2881e7e7c91aae8a17..b44540e126b02ee7bff877046d988eaaaa2ab700 |
| command | sdlc close --issue 725 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T02:44:33-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The runner validates selections, preserves child failures, and reuses existing Make recipes. Focused tests and the actual upstream parser/planner fixtures pass. Shipping is blocked by incomplete group dependencies: several Go changes silently omit the non-Go integration suites that exercise them.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: affected-consumer-coverage
    title: |
      Group Go roots omit dependencies exercised by their integration suites
    detail: |
      .sdlc/affected.json:49 and :79 omit required consumer roots. Using the actual upstream planner against Pair's graph, reviewapply/reviewapply.go selects no groups; retentioncmd/run.go and gccmd/run.go also select no groups; pairlog/store.go selects only shell. Consequently these changes skip review-applied-round-test.sh, retention-test.sh, and submission-transaction-nvim-test.sh despite those suites exercising the changed commands. ARCH-PURPOSE: enumerate Go dependencies of every grouped non-Go suite, add the missing roots or conservative full fallbacks, and add actual-graph regression assertions for the complete enumeration.
```

1. **Strengths**

   - Pure validation and argv construction are separate from subprocess execution in `scripts/test-affected.py:21`.
   - Entrypoint tests verify exact package arguments, environment scrubbing, failure propagation, and shared private-binary ownership.
   - README and atlas both document the new surface, upstream dependency, and conservative fallbacks.
   - Existing full-suite prerequisites remain present; the ownership audit guards against recipe drift.

2. **Critical findings**

   - **Incomplete group dependencies** — `.sdlc/affected.json:49` and `:79`. Confirmed omissions:
   
     | Changed package | Required group | Actual groups |
     |---|---|---|
     | `reviewapply` | `review` | none |
     | `retentioncmd` | `lua` | none |
     | `gccmd` | `lua` | none |
     | `pairlog` | `lua` | `shell` |

     Concrete consumers include `tests/review-applied-round-test.sh:45`, `tests/retention-test.sh:36`, and `nvim/submission_integration_test.lua:9`. Audit all five groups rather than fixing only these examples.

3. **Important findings**

   None separately raised.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Eight runner tests passed.
   - Default configuration qualification explicitly skipped the unavailable provider.
   - Explicit qualification against upstream candidate `d6309691af55cb39765a55623ed01f5ab31212e2` passed.
   - Additional disposable actual-graph assertions failed for all four omissions above. Existing root tests check existence, not dependency completeness.
   - Full suite was not rerun during this review. Repository files remain unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** existing recipes and upstream planner reused.
   - **ARCH-PURE — pass:** declared pure entities have deterministic, effect-free logic.
   - **ARCH-PURPOSE — flag:** incomplete consumer coverage undermines affected-test evidence.
   - **ARCH-MOCK — pass:** executable doubles exercise the production subprocess boundary.
   - **ARCH-CONSTRAINTS — pass:** bounded dispatch counts and explicit test timeouts.
   - **ARCH-SECURE — pass:** validated inputs become argv; malformed selections fail before child execution.
   - **ARCH-ORDER — pass:** synchronous execution holds no authoritative state between invocations.
   - **ARCH-FUNERAL — pass:** test scratch has scoped cleanup; existing runtime-generation lifecycle is reused.
   - **ARCH-EVOLVE — pass:** static configuration is not rewritten; strict upstream parsing was exercised.

7. **Plan revision recommendations**

   Add a `## Revisions` entry requiring an enumeration of every grouped suite’s Go command dependencies and actual-graph tests proving each dependency selects its consumer group. Update the documented group roots accordingly.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"ad61655d-f17f-437e-98f0-d93d543dff7d","repository":"github.com/xianxu/pair","issue_id":"000725","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/a23ae908ea2adba6","reviewed_head":"b44540e126b02ee7bff877046d988eaaaa2ab700","slot":"pair:3","requested_at":"2026-10-20T09:42:03.415163Z"},"phase":"finished","pid":52522,"started_at":"2026-10-20T09:42:03.506611Z","finished_at":"2026-10-20T09:44:32.672667Z","session_id":"01a158b0-7cda-7182-b9cf-800ed9eb6f98","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.399136792s
head: "b44540e126b02ee7bff877046d988eaaaa2ab700"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
