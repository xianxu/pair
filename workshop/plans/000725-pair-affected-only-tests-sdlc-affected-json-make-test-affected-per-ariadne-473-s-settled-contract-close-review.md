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

---

## Re-review — 2026-10-20T02:59:29-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 725 — pair affected-only tests: .sdlc/affected.json + make test-affected, per ariadne#473's settled contract |
| repo | pair |
| issue file | workshop/issues/000725-pair-affected-only-tests-sdlc-affected-json-make-test-affected-per-ariadne-473-s-settled-contract.md |
| boundary | whole-issue close |
| milestone | — |
| window | ab32acea6c97663232623b2881e7e7c91aae8a17..96360dc5d701388f8644d533e2034d465d7a618e |
| command | sdlc close --issue 725 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T02:59:29-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The runner’s validation, exact Go argv, failure propagation, and private-binary lifecycle are covered and pass. BR-1’s Go-root correction is verified by a failing regression when reverted. A separate gap remains: cross-language file consumers are missing from group selection, allowing affected runs to skip existing regression suites.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The added consumer roots pass the actual upstream planner/Pair graph regression. Removing the lua retentioncmd/gccmd/pairlog and review reviewapply/pairlog roots in a disposable pinned-head copy makes TestActualGroupConsumerCoverage fail with omitted consumer groups.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: affected-consumer-coverage
    title: |
      Cross-language source changes omit consuming shell suites
    detail: |
      .sdlc/affected.json:137 omits nvim inputs consumed by shell recipes. Against upstream d6309691af55cb39765a55623ed01f5ab31212e2 and Pair's actual graph, both nvim/zellij_trace.lua and nvim/init.lua select Full=false and groups=[lua review], skipping test-zellij-trace and test-term-pane-shortcuts. The former loads zellij_trace.lua at tests/zellij-trace-test.sh:25; the latter asserts init.lua content at tests/term-pane-shortcuts-test.sh:320. ARCH-PURPOSE: This is the 2nd finding in family affected-consumer-coverage. Earlier rounds fixed instances. Do NOT fix only these instances: state and enforce the rule that every grouped recipe's executed or text-read input selects its consuming group or full. Enumerate recursive cross-language inputs across all groups, extend the inventory beyond Go roots, and add actual-planner regressions for that enumeration.
```

1. **Strengths**

   - Pure selection validation rejects empty, duplicate, foreign, and wildcard inputs before child effects (`scripts/test-affected.py:21`).
   - Runner tests exercise the real Make entrypoint, preserved output, failure stopping, environment scrub, and private-binary cleanup.
   - Actual upstream fixtures verify strict parsing, transitive/test importers, embedded Lua, and conservative unknown-path fallbacks.
   - README and atlas both document the new command and operational ownership.

2. **Critical findings**

   - The finding above blocks shipping. Extend `.sdlc/affected.json:137` and the consumer inventory using a complete file-dependency audit. The existing regression at `tests/affected-config-test.py:204` checks Go dependencies only.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - All nine runner tests passed.
   - Explicit upstream qualification passed; automatic discovery reported its documented dependency skip.
   - BR-1 mutation failed as expected in a disposable copy.
   - Added review probes failed for both omitted shell consumers.
   - Existing `tests/zellij-trace-test.sh` passed.
   - Full suite and timing comparison were not rerun; the tracker still marks that qualification pending. Reviewed checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** reuses existing Make recipes and upstream planning.
   - **ARCH-PURE — Pass:** declared pure functions perform no IO; execution remains in the shell layer.
   - **ARCH-PURPOSE — Flag:** consumer coverage remains incomplete across language boundaries.
   - **ARCH-MOCK — Pass:** recorder tests exercise the production entrypoint; actual Make/Go qualification supplements them.
   - **ARCH-CONSTRAINTS — Pass:** sequential work, bounded suite timeout, and invocation-count checks match the declared envelope.
   - **ARCH-SECURE — Pass:** validated tokens become argv; selection errors remain visible.
   - **ARCH-ORDER — Pass:** synchronous execution stops on failure; no new durable event state.
   - **ARCH-FUNERAL — Pass:** temporary fixtures and private binaries have scoped cleanup.
   - **ARCH-EVOLVE — Pass:** static declaration has no rewrite path; strict refusal protects selection semantics.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining consumer coverage to include recursively executed files, fixtures, and source-text reads—not just Go command dependencies—and name the inventory and planner regression enforcing that rule across all five groups.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"e8002f6b-2595-48fb-b895-c15f016a65f0","repository":"github.com/xianxu/pair","issue_id":"000725","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/f208cdb307316362","reviewed_head":"96360dc5d701388f8644d533e2034d465d7a618e","slot":"pair:3","requested_at":"2026-10-20T09:56:23.489877Z"},"phase":"finished","pid":94092,"started_at":"2026-10-20T09:56:23.692563Z","finished_at":"2026-10-20T09:59:28.711231Z","session_id":"01a158bd-9dbb-7323-96cb-7e2ac8cd8369","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.952102542s
head: "96360dc5d701388f8644d533e2034d465d7a618e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
