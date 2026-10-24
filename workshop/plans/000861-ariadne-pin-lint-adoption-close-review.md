# Boundary Review — pair#861 (whole-issue close)

| field | value |
|-------|-------|
| issue | 861 — Adopt Ariadne shared sidedoc pin and qualify newly active lint rules |
| repo | pair |
| issue file | workshop/issues/000861-ariadne-pin-lint-adoption.md |
| boundary | whole-issue close |
| milestone | — |
| window | 54c4a59cae9001306faf90f10c6f7a5e95756fe4..c7723fbda1471ee2ddb1f13bdc8c30e627dbf88d |
| command | sdlc close --issue 861 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T01:09:16-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The implementation preserves production defaults, consolidates JSON validation without apparent owner-contract drift, and passes the catalog and focused tests. No correctness defect was identified. SHIP is blocked by an environment-dependent verification gap: affected-planner qualification repeatedly exceeded its cold-graph deadline, including an isolated retry.

1. **Strengths**

   - JSON policies remain explicit per consumer; tests cover Unicode aliases, nested duplicates, depth limits, opaque numbers, and visitor errors.
   - Deadline and timer seams preserve standard-library production defaults while tests expire requests after readiness.
   - The full catalog and reader inventory passed. Exactly two narrowly documented scanner exceptions remain.
   - Atlas documentation covers the new internal surfaces; no new product command or configuration requires README additions.

2. **Critical findings**

   None.

3. **Important findings**

   `tests/affected-config-test.py:182–187`: required planner qualification could not be verified. Installed-pin runs and the pinned dependency-source run failed during cold-graph qualification. The isolated installed-pin retry also reported `context deadline exceeded`, taking 135.40 seconds for that test. See the machine-readable finding below for answering evidence.

4. **Minor findings**

   `workshop/plans/000861-ariadne-pin-lint-adoption-plan.md:21`: the Core concepts table omits PURE/INTEGRATION classifications. Listed entities and locations otherwise match the implementation.

5. **Test coverage notes**

   Passed: full repolint catalog, scanner, strictjson, submission, testwait, editorial, artifact classification, reader inventory, focused lifecycle tests across eight packages, changed historyarchive owner tests, and three race repetitions for cue deadlines and drawing expiry.

   The broader historyarchive run exceeded my three-minute limit during unchanged `TestDriveCandidateProjectionScansCatalogOnceAtNAnd2N`. This does not establish a regression. Planner qualification failed independently as described above.

6. **Architectural notes**

   | Marker | Result |
   |---|---|
   | ARCH-DRY | Pass: shared scanner and AST adapter consolidate behavior. |
   | ARCH-PURE | Code separation passes; flag minor plan-classification omission. |
   | ARCH-PURPOSE | Implementation covers the inventoried finding classes; qualification remains unverified. |
   | ARCH-MOCK | Pass: existing runtime/process seams remain shared with tests. |
   | ARCH-CONSTRAINTS | Flag: cold-graph qualification could not complete within its declared budget here. |
   | ARCH-SECURE | Pass: malformed JSON refuses visibly; fixture roots remain private. |
   | ARCH-ORDER | Pass: terminal-cause ordering and readiness-before-expiry receive direct coverage. |
   | ARCH-FUNERAL | Pass: cancellation registrations, timers, roots, and child joins retain cleanup ownership. |
   | ARCH-EVOLVE | Pass: extraction preserves owner decoding and raw-extension retention. |

7. **Plan revision recommendations**

   Add a `## Revisions` entry classifying each Core concepts entity as PURE or INTEGRATION, particularly the clock/context-backed cue helper. No implementation redesign is presently supported by the verification failure.

```findings
findings:
  - id: new
    severity: Important
    kind: unverifiable
    family: boundary-qualification-evidence
    title: |
      Required affected-planner qualification remains unverified
    detail: |
      At tests/affected-config-test.py:182–187, GOPROXY=off GOSUMDB=off python3 tests/affected-config-test.py failed against installed pin 5a17f789619d during cold-graph qualification with context deadline exceeded; an isolated retry failed likewise after 135.40 seconds in TestDeclaredPreparationQualifiesCleanPairGraph. Qualification using the supplied dependency checkout at b65c6b11 also failed. ARCH-CONSTRAINTS: provide a complete passing qualification run for the pinned Pair head on a host where the existing budgets complete, recording the Pair head, provider identity, command, and timings; alternatively resolve the measured budget failure and rerun. This is a verification gap, not an established production defect.
  - id: new
    severity: Minor
    family: core-concept-classification
    title: |
      Core concepts table omits PURE/INTEGRATION kinds
    detail: |
      workshop/plans/000861-ariadne-pin-lint-adoption-plan.md:21 lists names, locations, and status without the requested kind column. ARCH-PURE: add explicit classifications based on actual dependencies and record the clarification under Revisions.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1fcb6f0e-c1c5-4482-a11d-72482ceb86db","repository":"github.com/xianxu/pair","issue_id":"000861","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/f86c4965f6d422a1","reviewed_head":"c7723fbda1471ee2ddb1f13bdc8c30e627dbf88d","slot":"pair:3","requested_at":"2026-10-24T07:59:18.373959Z"},"phase":"finished","pid":63462,"started_at":"2026-10-24T07:59:18.777052Z","finished_at":"2026-10-24T08:08:57.06735Z","session_id":"01a16ceb-dd03-7b82-8200-2f0219fe9fe6","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.303078208s
head: "c7723fbda1471ee2ddb1f13bdc8c30e627dbf88d"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T01:19:00-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 861 — Adopt Ariadne shared sidedoc pin and qualify newly active lint rules |
| repo | pair |
| issue file | workshop/issues/000861-ariadne-pin-lint-adoption.md |
| boundary | whole-issue close |
| milestone | — |
| window | 54c4a59cae9001306faf90f10c6f7a5e95756fe4..bddf4ed8812aeaa9f35ba6863f8d0035411f0400 |
| command | sdlc close --issue 861 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T01:19:00-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The pinned Pair range satisfies the adoption contract. Both prior findings are resolved, and no new blocking defects were found. Relevant qualification, catalog, JSON, inventory, and timing checks pass; the broader historyarchive suite remains unqualified by this review.

```findings
dispose:
  - id: BR-2
    disposition: addressed
    note: |
      The plan's Core concepts table at lines 21–26 now supplies accurate PURE/INTEGRATION classifications, and lines 147–149 record the clarification under Revisions. The before/after diff and the scanner, adapter, deadline and watcher implementations support those classifications.
  - id: BR-1
    disposition: withdrawn
    note: |
      The supplied log matches its recorded SHA256, and no executable source changed after its measured c7723fb head. Independently rerunning GOPROXY=off GOSUMDB=off python3 -B tests/affected-config-test.py on pinned bddf4ed passed: cold graph 48.81s, Go fixture 56.376s, Python total 57.270s, within unchanged budgets using the normal 581 provider.
```

1. **Strengths**

   - The contextual lint adapter preserves upstream context while reusing Pair’s AST-only matchers.
   - `internal/jsonstrict/scan.go:32` consolidates structural validation while keeping owner policies explicit.
   - Deadline and timer seams retain production defaults and let tests expire requests after observed readiness.
   - Atlas documentation covers both new internal surfaces; no new user-facing usage requires a README update.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed the full repolint package, reader inventory, scanner and strictjson tests, submission, sessionwatch, editorial, changed historyarchive owner tests, and focused timing tests across eight packages. Three race repetitions passed for cue deadlines and drawing expiry. Differential scanner fuzzing passed 14,404 executions.

   A broader historyarchive run was terminated after 181 seconds without a result. This review does not claim a full-suite pass. The checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared scanner replaces duplicate validation logic.
   - **ARCH-PURE — pass:** deterministic scanning and AST adaptation remain separate from injected timing seams.
   - **ARCH-PURPOSE — pass:** normal pin, catalog integration, qualification dependencies, and lint-site remediation are delivered.
   - **ARCH-MOCK — pass:** changed timing paths use controlled seams alongside existing stateful fixtures.
   - **ARCH-CONSTRAINTS — pass:** linear scanner work is tested; qualification passes unchanged budgets.
   - **ARCH-SECURE — pass:** structural validation precedes typed adoption; duplicate and sensitive-alias rejection remain covered.
   - **ARCH-ORDER — pass:** readiness precedes expiry, and tests cover terminal-cause ordering and late success.
   - **ARCH-FUNERAL — pass:** notification release, timer stopping, temporary-root cleanup, and child joins retain explicit owners.
   - **ARCH-EVOLVE — pass:** no format change; owner strictness and raw unknown-value preservation remain intact.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"fc48439d-c742-4388-b104-660ec830f580","repository":"github.com/xianxu/pair","issue_id":"000861","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/a35e5ed9767a9097","reviewed_head":"bddf4ed8812aeaa9f35ba6863f8d0035411f0400","slot":"pair:3","requested_at":"2026-10-24T08:14:28.556984Z"},"phase":"finished","pid":88678,"started_at":"2026-10-24T08:14:28.782269Z","finished_at":"2026-10-24T08:18:42.585629Z","session_id":"01a16cf9-bf29-7ab0-bde7-0d26fed57e4a","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.343572625s
head: "bddf4ed8812aeaa9f35ba6863f8d0035411f0400"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
