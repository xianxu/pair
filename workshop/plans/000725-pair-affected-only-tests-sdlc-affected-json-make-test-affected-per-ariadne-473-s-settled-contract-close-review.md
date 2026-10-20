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

---

## Re-review — 2026-10-20T03:15:47-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 725 — pair affected-only tests: .sdlc/affected.json + make test-affected, per ariadne#473's settled contract |
| repo | pair |
| issue file | workshop/issues/000725-pair-affected-only-tests-sdlc-affected-json-make-test-affected-per-ariadne-473-s-settled-contract.md |
| boundary | whole-issue close |
| milestone | — |
| window | ab32acea6c97663232623b2881e7e7c91aae8a17..daf9ab9f5a8770dce86a8b94a6ebe2b33c33f9c0 |
| command | sdlc close --issue 725 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T03:15:47-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The runner and grouped-recipe coverage checks work, and BR-2’s correction has meaningful regression evidence. However, the inventory still excludes Go tests that directly consume non-Go files. Actual-planner probes confirm those tests are omitted from affected selections, leaving a blocking coverage gap.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The audited command roots and transitive dependency checks remain present; TestActualGroupConsumerCoverage passes against the pinned upstream planner and actual Pair graph.
  - id: BR-2
    disposition: addressed
    note: |
      The inventory now enumerates recursive file inputs for grouped recipes. Actual file-consumer coverage passes; disposable mutations removing shell nvim coverage or Lua's shared perf fixture coverage both make that regression fail.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: affected-consumer-coverage
    title: |
      Non-Go changes omit Go tests that execute or read those inputs
    detail: |
      .sdlc/affected.json:33,123 claims inputs without selecting their consuming Go tests. Against upstream d6309691af55cb39765a55623ed01f5ab31212e2 and Pair's actual graph, nvim/workbench_actions.lua selects Full=false, GoPkgs=[], groups=[build lua review shell], omitting workbenchshortcut's TestRenderedLuaGlobalMapsMatchCommittedFile (shortcut_test.go:354). scripts/supervision-labels/labels.py similarly selects only build, omitting couchcmd's real Python integration test (supervision_fact_storage_test.go:50). Additional confirmed inputs are nvim/init.lua, tests/adapt-golden.expected and zellij/config.kdl; all produce empty GoPkgs despite direct Go-test consumers. The runner adds only runtimebundle for lua, so these consumers remain unexecuted. ARCH-PURPOSE: This is the 3rd finding in family affected-consumer-coverage. Earlier rounds fixed instances. Extend the governing rule and inventory to every test consumer, including Go tests' external file reads and subprocess inputs; sweep the class and enforce consumer execution or full fallback through actual-planner and runner regressions. Reconcile the scripts-only selection requirement explicitly rather than leaving its Go integration consumer uncovered.
```

### 1. Strengths

- Selection validation and argv construction are pure; invalid selections cause no child effects (`scripts/test-affected.py:21`).
- Runner tests exercise the real Make entrypoint, failure propagation, environment scrubbing and private-binary cleanup.
- Recipe ownership is checked against the independent full-suite prerequisite list.
- README and atlas document the new command, fallback behavior and upstream dependency.

### 2. Critical findings

- **Go consumers of non-Go inputs remain uncovered**, as detailed above. The inventory at `tests/affected-config-test.py:229` checks grouped recipes only. Extend it to Go-test consumers and verify that the selected execution actually runs them.

### 3. Important findings

None.

### 4. Minor findings

None.

### 5. Test coverage notes

- All nine runner tests passed.
- Explicit upstream qualification passed, including strict parsing, dependency unions, fallbacks and **732 distinct group/file inputs**.
- Removing shell’s `nvim/**` coverage caused file-consumer regressions to fail.
- Removing Lua’s shared perf-fixture coverage also caused a regression failure.
- Directly invoked dispatcher, workbenchshortcut and couchcmd consumer tests passed, confirming these are runnable tests omitted by the affected selections.
- The full suite was not rerun during this review.

### 6. Architectural notes for upcoming work

- **ARCH-DRY — pass:** existing Make recipes and upstream planner are reused.
- **ARCH-PURE — pass:** validation and argv logic remain separate from execution.
- **ARCH-PURPOSE — flag:** coverage enumeration stops at grouped recipes, excluding Go consumers of external files.
- **ARCH-MOCK — pass:** runner integration uses stateful executable recorders and failure injection.
- **ARCH-CONSTRAINTS — pass:** bounded sequential execution, one selected Go invocation and one group wrapper.
- **ARCH-SECURE — pass:** selection input becomes validated argv before execution.
- **ARCH-ORDER — pass:** synchronous execution carries no authoritative state between invocations.
- **ARCH-FUNERAL — pass:** new test scratch directories have scoped cleanup.
- **ARCH-EVOLVE — pass:** declaration parsing retains the explicitly required strict schema contract; no rewrite path is introduced.

### 7. Plan revision recommendations

Add a `## Revisions` entry broadening the coverage rule to **all executed tests**, including Go tests’ non-Go inputs. Name the complete enumeration, explain how those consumers run under the unchanged upstream selection contract, and reconcile any necessary exceptions with scripts-only selectivity.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f1168227-4b31-4ce0-866f-e74cc12b86b8","repository":"github.com/xianxu/pair","issue_id":"000725","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/05e175f181203a49","reviewed_head":"daf9ab9f5a8770dce86a8b94a6ebe2b33c33f9c0","slot":"pair:3","requested_at":"2026-10-20T10:13:08.570182Z"},"phase":"finished","pid":85184,"started_at":"2026-10-20T10:13:08.739537Z","finished_at":"2026-10-20T10:15:47.38472Z","session_id":"01a158cc-f34c-7af2-a50a-14a754a073e9","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.35240125s
head: "daf9ab9f5a8770dce86a8b94a6ebe2b33c33f9c0"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-20T03:39:57-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 725 — pair affected-only tests: .sdlc/affected.json + make test-affected, per ariadne#473's settled contract |
| repo | pair |
| issue file | workshop/issues/000725-pair-affected-only-tests-sdlc-affected-json-make-test-affected-per-ariadne-473-s-settled-contract.md |
| boundary | whole-issue close |
| milestone | — |
| window | ab32acea6c97663232623b2881e7e7c91aae8a17..f7a52217447bfc52f67fb2054503a256a8146b37 |
| command | sdlc close --issue 725 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T03:39:57-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-3’s non-Go fallback correction is verified, including a regression that fails when the correction is reverted. However, the new Go scanner preflight omits repository-wide consumers misclassified as local fixtures. A disposable-checkout mutation demonstrates the resulting coverage gap.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Actual upstream planner/Pair-graph tests pass for audited group command roots and their transitive dependencies.
  - id: BR-2
    disposition: addressed
    note: |
      Actual file-consumer fixtures pass: audited non-Go inputs now conservatively force full coverage, including the previously omitted Lua inputs.
  - id: BR-3
    disposition: addressed
    note: |
      TestActualNonGoInputsForceFull passes against upstream 113e7767719b5b4b4964ba9a8f5e73fecce825b8. Restoring the pre-fix declaration in a disposable checkout makes it fail for labels.py, workbench_actions.lua, init.lua, adapt-golden.expected and config.kdl, among others.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: affected-consumer-coverage
    title: |
      Repository-wide Go scanners are misclassified and omitted from affected runs
    detail: |
      tests/affected-go-readers.json:91,183 classifies deletedvocabulary_test.go and couchcore/retire_test.go as local_or_literal_fixture, although their tests scan production files across the repository. Neither TestDeletedVocabularyIsNotDescribedAsLive nor TestOnlyTeardownProofHasNoProductionCaller appears in scripts/test-affected.py's SOURCE_GUARDS. For the planner-verified slugcmd/slug.go selection, neither owning package is selected; the preflight filters out both tests. In a disposable pinned-head checkout, appending "// TreeSummary is the active session model." to slugcmd/slug.go made the deleted-vocabulary test fail while the complete fixed preflight exited 0. ARCH-PURPOSE: This is the 4th finding in family affected-consumer-coverage. Earlier rounds fixed instances. Do not fix only these two registrations: state and enforce the rule for cross-package readers, re-audit all 372 local/fixture dispositions including helper-mediated reads, and add regression evidence that affected execution catches cross-package violations.
```

1. **Strengths**

   - Selection validation and argv construction remain pure; malformed inputs are rejected before child effects.
   - Stateful recorder tests exercise the real Make entrypoint, failure propagation and private-binary cleanup.
   - Configuration fixtures use the actual upstream parser/planner and Pair dependency graph.
   - README and atlas updates cover the new command, fallbacks and preflights.

2. **Critical findings**

   The finding above blocks shipping. The incorrect classifications are at `tests/affected-go-readers.json:91,183`; the omitted consumers are implemented at `cmd/internal/artifactpath/deletedvocabulary_test.go:65` and `cmd/internal/couchcore/retire_test.go:308`.

   Re-audit reader classifications against their actual input scope, register every cross-package scanner or conservatively select full, and test coverage independently of the registry being checked.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed 11 runner tests, six reader-inventory tests, and explicit upstream parser/planner qualification. Reverting BR-3’s declaration correction produced the expected coverage failures. The scanner mutation demonstrated a false-negative preflight. Disposable checkouts were removed; the reviewed checkout remains clean. The full suite was not rerun.

6. **Architectural notes**

   - **ARCH-DRY — pass:** reuses upstream planning and existing Make recipes.
   - **ARCH-PURE — pass:** pure selection functions and thin execution shell.
   - **ARCH-PURPOSE — flag:** scanner coverage remains incomplete, as demonstrated above.
   - **ARCH-MOCK — pass:** stateful child recorder and native qualification share production boundaries.
   - **ARCH-CONSTRAINTS — pass:** fixed preflight invocation count and bounded child timeouts.
   - **ARCH-SECURE — pass:** validated selection becomes argv without shell interpolation.
   - **ARCH-ORDER — pass:** synchronous execution stops on failure; no background lifetime.
   - **ARCH-FUNERAL — pass:** temporary artifacts have context-managed cleanup.
   - **ARCH-EVOLVE — pass:** declaration is not rewritten; upstream strict parsing protects selection semantics.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining cross-package readers by actual inputs, including helper-mediated reads. Record the complete disposition re-audit and require an independently seeded violation to fail affected execution. Retain the outstanding full-suite and final timing qualification tasks.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"3aedd0d7-d0dd-4664-8fbf-ded88ba08f27","repository":"github.com/xianxu/pair","issue_id":"000725","recipe":"milestone-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/e63c0cf69697fa42","reviewed_head":"f7a52217447bfc52f67fb2054503a256a8146b37","slot":"pair:3","requested_at":"2026-10-20T10:36:32.958475Z"},"phase":"finished","pid":99974,"started_at":"2026-10-20T10:36:33.120526Z","finished_at":"2026-10-20T10:39:57.360563Z","session_id":"01a158e2-6142-78b2-a733-61a8b1769288","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.581007084s
head: "f7a52217447bfc52f67fb2054503a256a8146b37"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T04:57:16-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 725 — pair affected-only tests: .sdlc/affected.json + make test-affected, per ariadne#473's settled contract |
| repo | pair |
| issue file | workshop/issues/000725-pair-affected-only-tests-sdlc-affected-json-make-test-affected-per-ariadne-473-s-settled-contract.md |
| boundary | whole-issue close |
| milestone | — |
| window | ab32acea6c97663232623b2881e7e7c91aae8a17..55d81f74276404be06a6b7ab86e534a4e26bb393 |
| command | sdlc close --issue 725 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T04:57:16-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned implementation matches the governing Spec/Plan revisions. BR-4 is addressed with broader reader coverage, independent registration enforcement, and a native regression that demonstrably fails without the correction. No new blocking findings emerged.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Current upstream-planner fixtures pass command-root and transitive-dependency coverage for grouped recipes; no remaining defect identified.
  - id: BR-2
    disposition: addressed
    note: |
      Current file-consumer fixtures pass against the actual upstream planner, including conservative full fallback for audited non-Go inputs.
  - id: BR-3
    disposition: addressed
    note: |
      Non-Go reader populations force full; Go selections execute the fixed scanner preflight before the exact selected package argv. Planner and runner checks pass.
  - id: BR-4
    disposition: addressed
    note: |
      The corrected inventory and independent source-walker rule pass all eight reader tests. The native regression catches both seeded violations. Removing the two scanner registrations only in a disposable worktree makes that regression fail because affected execution incorrectly exits zero, confirming meaningful regression evidence.
```

1. **Strengths**

   - Selection validation and argv construction remain pure; subprocess execution uses argument arrays and propagates failures (`scripts/test-affected.py:431`).
   - Recipe ownership checks preserve the existing full-suite prerequisite population (`Makefile.local:224`).
   - BR-4’s regression exercises the actual Make entrypoint and verifies downstream groups stop after scanner failure (`tests/affected-go-readers-test.py:176`).
   - README and atlas explain the new command, conservative fallbacks, preflights, and qualification ownership.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None requiring action.

5. **Test coverage notes**

   - Runner tests: **11 passed**.
   - Reader inventory/native tests: **8 passed**.
   - Upstream qualification against commit `113e7767719b5b4b4964ba9a8f5e73fecce825b8`: **passed**, including strict parsing, package/importer selection, group dependencies, and 732 expanded file-consumer checks.
   - BR-4 registration-removal mutation: regression **failed as expected**.
   - Full suite was not independently rerun during this review. The recorded suite revision and reviewed head differ only in the issue log.
   - Disposable worktrees were removed; tracked checkout remains clean.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Reuses upstream planning and existing Make recipes.
   - **ARCH-PURE — pass:** Pure selection logic is directly tested; IO stays in dispatch functions.
   - **ARCH-PURPOSE — pass:** Consumer inventories, full fallbacks, and native regression address coverage beyond the originally reported instances.
   - **ARCH-MOCK — pass:** Stateful child recorders share production execution boundaries; native qualification complements them.
   - **ARCH-CONSTRAINTS — pass:** Invocation-count tests enforce constant preflight overhead and bounded wrapper lifetimes.
   - **ARCH-SECURE — pass:** Invalid selections refuse before dispatch; inputs never enter shell evaluation.
   - **ARCH-ORDER — pass:** Sequential execution stops on failure and launches no detached work.
   - **ARCH-FUNERAL — pass:** Temporary artifacts and worktrees have cleanup paths.
   - **ARCH-EVOLVE — pass:** Static configuration is not rewritten; strict unknown/duplicate refusal matches the documented safety contract.

7. **Plan revision recommendations:** None. Existing revisions describe the delivered behavior and retain post-landing qualification ownership.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1785520e-44fb-4a53-8556-3cc2e733b496","repository":"github.com/xianxu/pair","issue_id":"000725","recipe":"milestone-review","round":5,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/9bc8532ad945c6b9","reviewed_head":"55d81f74276404be06a6b7ab86e534a4e26bb393","slot":"pair:3","requested_at":"2026-10-20T11:53:14.61292Z"},"phase":"finished","pid":53966,"started_at":"2026-10-20T11:53:14.829242Z","finished_at":"2026-10-20T11:57:15.664987Z","session_id":"01a15928-990c-7131-a0ce-027a52391d33","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.229741375s
head: "55d81f74276404be06a6b7ab86e534a4e26bb393"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-20T09:09:35-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 725 — pair affected-only tests: .sdlc/affected.json + make test-affected, per ariadne#473's settled contract |
| repo | pair |
| issue file | workshop/issues/000725-pair-affected-only-tests-sdlc-affected-json-make-test-affected-per-ariadne-473-s-settled-contract.md |
| boundary | whole-issue close |
| milestone | — |
| window | 1b4b0064a6d20d5247d499a50e6ebc5ecb17a90b..3b8bcbebec9a8fa74253b781cdf6872644ff49ae |
| command | sdlc close --issue 725 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-20T09:09:35-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff refreshes 15 audited reader hashes, conservatively reclassifies six readers, and records the completed review. The updated entries match the pinned source and retain scanner or full-fallback coverage. No scope change or new blocking defect was found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The interdiff preserves the reviewed command-root coverage and dependency fixtures; no regression identified.
  - id: BR-2
    disposition: addressed
    note: |
      Shared fixture inputs remain covered by existing full fallbacks; the inventory refresh does not narrow consumer coverage.
  - id: BR-3
    disposition: addressed
    note: |
      Scanner ordering and exact selected argv remain unchanged; all 11 runner tests pass.
  - id: BR-4
    disposition: addressed
    note: |
      All eight reader tests pass, including independent registration enforcement and the native affected-command regression requiring both seeded scanner failures.
```

1. **Strengths**

   - The inventory remains fail-closed: applying the base inventory to pinned-head sources produces exactly 15 changed-reader errors; the updated inventory produces none.
   - Reclassified readers use conservative full-fallback ownership for shared non-Go inputs.
   - The plan and tracker changes preserve Spec, Done when, and Revisions without expanding scope.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Runner tests: **11 passed**.
   - Reader inventory and native regression tests: **8 passed**.
   - The native regression confirms both seeded cross-package violations stop affected execution before selected groups run.
   - Diff whitespace checks passed. Full suite and upstream planner qualification were not independently rerun during this narrow review.

6. **Architectural notes**

   - **ARCH-DRY — pass:** No duplicated execution or planning logic introduced.
   - **ARCH-PURE — pass:** The refresh leaves the pure core and IO boundaries unchanged.
   - **ARCH-PURPOSE — pass:** Updated reader inputs retain scanner or conservative fallback coverage.
   - **ARCH-EVOLVE — pass:** Inventory version, field meanings, and format remain unchanged.

   No new architectural or user-facing surface requires README or atlas changes in this window.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"93c483ae-c156-4a2c-93cd-0fcd89fd05f4","repository":"github.com/xianxu/pair","issue_id":"000725","recipe":"fix-review","round":6,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/4332d5b3299c037c","reviewed_head":"3b8bcbebec9a8fa74253b781cdf6872644ff49ae","slot":"pair:3","requested_at":"2026-10-20T16:07:52.684656Z"},"phase":"finished","pid":42827,"started_at":"2026-10-20T16:07:52.896785Z","finished_at":"2026-10-20T16:09:34.667741Z","session_id":"01a15a11-b935-7ce3-83c1-d3db5ceeef7e","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.858605041s
head: "3b8bcbebec9a8fa74253b781cdf6872644ff49ae"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
