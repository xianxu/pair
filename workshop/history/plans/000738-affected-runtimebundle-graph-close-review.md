# Boundary Review — pair#738 (whole-issue close)

| field | value |
|-------|-------|
| issue | 738 — Affected package graph falls back to full suite before runtime bundle preparation |
| repo | pair |
| issue file | workshop/issues/000738-affected-runtimebundle-graph.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3b5e8efb9c94035076c5edf5683db97422e6de76..35628e88869ae45ff3e02d3388493557dd07bd6a |
| command | sdlc close --issue 738 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T02:45:25-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The declaration delivers the runtime preparation prerequisite, and qualification passes against a compatible provider. The clean-checkout regression exercises missing embeds, successful preparation, exact selection, and broken-graph refusal. README needs updating before close; operator local-gates verification also remains explicitly outstanding.

Done-when: current — its clauses still cover clean graph discovery, reproduction without borrowed assets, and conservative failure handling.

```findings
findings:
  - id: new
    severity: Important
    kind: defect
    family: user-facing-contract-documentation
    title: |
      README update is missing for graph_prepare and the new provider requirement
    detail: |
      .sdlc/affected.json:7 introduces graph_prepare, but README.md has no change in this range. Enumerated gaps: the affected-check instructions at README.md:2011 omit preparation, its two-minute bound and fallback behavior; README.md:2051 still says qualification requires ariadne#473 although tests/affected-config-test.py:59 now requires #517. Document the new configuration and update the qualification prerequisite, distinguishing existing affected support from preparation-capable support.
```

1. **Strengths**

   - Reuses the existing generator through bounded configuration rather than duplicating generation logic.
   - The regression uses a tracked-only archive and confirms generated assets are initially absent ([tests/affected-config-test.py:173](/Volumes/TestScratch/review-checkouts/4daffde9a15eb5c1/tests/affected-config-test.py:173)).
   - Asserts exact package/group selection and rejects a deliberately broken import.
   - Atlas documents ordering, isolation, timeout, and fallback.

2. **Critical findings:** None.

3. **Important findings:** The README gap above. Update the existing affected-check documentation with the new declaration and dependency requirement.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Default qualification visibly skips with the older installed provider.
   - Explicit qualification against clean provider checkout `fb6c4e165c9f3dd37e43ad248e8fa57d343f04fd` passed all tests in 8.8 seconds.
   - Removing `graph_prepare` from a temporary configuration made the new regression fail with `missing bounded runtime preparation: <nil>`.
   - Inspected provider orchestration and fallback tests: graph errors select the suite and retain their cause. These upstream tests were inspected, not executed.
   - `git diff --check` passed. No tracked changes were made.

6. **Architectural notes**

   - **ARCH-DRY: pass** — existing generator and actual provider are reused.
   - **ARCH-PURE: pass** — added IO is confined to integration qualification; no production business logic is introduced.
   - **ARCH-PURPOSE: pass** — preparation is declared at the graph owner’s seam, before discovery.
   - **ARCH-EVOLVE: pass** — configuration addition is additive; this diff introduces no decoder or rewrite path.

7. **Plan revision recommendations:** None. The plan accurately leaves operator verification unchecked.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5412ce6b-6151-49f1-b692-61d04fecabc7","repository":"github.com/xianxu/pair","issue_id":"000738","recipe":"small-diff-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/4daffde9a15eb5c1","reviewed_head":"35628e88869ae45ff3e02d3388493557dd07bd6a","slot":"pair:7","requested_at":"2026-10-21T09:43:37.61508Z"},"phase":"finished","pid":60525,"started_at":"2026-10-21T09:43:37.753548Z","finished_at":"2026-10-21T09:45:18.0566Z","session_id":"01a15dd8-4953-7f63-903f-7deee93d9f07","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.232391625s
head: "35628e88869ae45ff3e02d3388493557dd07bd6a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T02:51:32-07:00 (FIX-THEN-SHIP)

| field | value |
|-------|-------|
| issue | 738 — Affected package graph falls back to full suite before runtime bundle preparation |
| repo | pair |
| issue file | workshop/issues/000738-affected-runtimebundle-graph.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3b5e8efb9c94035076c5edf5683db97422e6de76..ee61d9db3024123a67b207a2c4a7f707e44cbd77 |
| command | sdlc close --issue 738 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T02:51:32-07:00 |
| verdict | FIX-THEN-SHIP |

## Review

```verdict
verdict: FIX-THEN-SHIP
confidence: high
```

The bounded preparation declaration works against the actual provider, and the clean-checkout qualification passes. BR-1’s prerequisite and preparation omissions are repaired, but its fallback documentation now incorrectly includes cancellation. Correct that distinction before closing. Operator verification remains explicitly unchecked.

Done-when: current — the clauses cover clean discovery, reproduction without borrowed assets, and conservative handling of genuine graph failures.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      README.md:2014–2018 now documents preparation and its bound, and README.md:2056–2060 correctly distinguishes #473 from #517. However, README.md:2018 says cancellation retains full-suite fallback. The provider's runAffectedCheck instead returns “graph interrupted” when its parent context is cancelled, before selection or suite execution (cmd/sdlc/localcheck_affected.go:50–51). TestAffectedGraphPreparationCancellationAndPipeDrain confirms cancellation returns an error without a successful affected or suite status; that test passed during this review. Enumerated documentation instances needing this distinction are README.md:2017–2018's explicit cancellation claim and atlas/index.md:36–37's blanket preparation-failure claim. Document ordinary preparation errors/timeouts and broken graphs as suite fallbacks, while cancellation aborts without recording a pass. Apply that distinction consistently to the issue's Revisions narrative.
```

1. **Strengths**

   - `.sdlc/affected.json:7` reuses the existing generator with an explicit two-minute limit.
   - `tests/affected-config-test.py:173–180` establishes absent generated assets and reproduces the missing-embed failure before preparation.
   - The regression asserts exact selection after generation and rejects a deliberately broken import afterward.

2. **Critical findings:** None.

3. **Important findings:** BR-1 remains open for the cancellation documentation mismatch described above. No new finding is raised for the same family.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Default `make test-affected-config` visibly skipped because its available provider predates #517.
   - Explicit qualification against provider `fa9761e6b45a3bce3dc32872fd42e50b2797f837` passed, including 732 audited group/file inputs.
   - Provider fallback and cancellation regressions passed.
   - `git diff --check` passed. No repository files were modified.

6. **Architectural notes**

   - **ARCH-DRY: pass** — existing generator and provider implementation are reused.
   - **ARCH-PURE: pass** — added IO remains in integration qualification.
   - **ARCH-PURPOSE: pass** — preparation reaches the graph owner’s seam and enables clean graph discovery.
   - **ARCH-EVOLVE: pass** — additive configuration; no decoder or rewrite path changes in this window.

7. **Plan revision recommendations**

   Add a `## Revisions` entry clarifying that ordinary preparation failure, preparation timeout, and broken graphs select the suite; parent cancellation aborts without recording successful verification. Preserve the unchecked operator-verification item until its evidence exists.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f2d0de05-4a96-4254-a1d3-d475c36c1f22","repository":"github.com/xianxu/pair","issue_id":"000738","recipe":"small-diff-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/1b1daa9387674f98","reviewed_head":"ee61d9db3024123a67b207a2c4a7f707e44cbd77","slot":"pair:7","requested_at":"2026-10-21T09:49:34.323055Z"},"phase":"finished","pid":47914,"started_at":"2026-10-21T09:49:34.440888Z","finished_at":"2026-10-21T09:51:31.650467Z","session_id":"01a15ddd-baa7-7d90-be73-7cd90d93d8d9","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.839706458s
head: "ee61d9db3024123a67b207a2c4a7f707e44cbd77"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T02:55:39-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 738 — Affected package graph falls back to full suite before runtime bundle preparation |
| repo | pair |
| issue file | workshop/issues/000738-affected-runtimebundle-graph.md |
| boundary | whole-issue close |
| milestone | — |
| window | 3b5e8efb9c94035076c5edf5683db97422e6de76..71623e8e30520581a091fdd4536230ed43190e09 |
| command | sdlc close --issue 738 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T02:55:39-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The declaration establishes runtime generation before graph discovery, and qualification passes against the actual preparation-capable provider. BR-1 is addressed: README, atlas, and the issue’s revision now correctly distinguish fallback from parent cancellation. No new findings. Operator verification remains an explicit prerequisite before landing.

Done-when: current — covers clean discovery, reproduction without borrowed assets, and conservative graph-failure handling.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      README.md:2014–2019 documents preparation, its two-minute bound, fallback and cancellation; README.md:2056–2063 distinguishes #473 from #517. atlas/index.md:35–38 and the issue's Revisions:144–151 agree. Provider localcheck_affected.go:49–56 confirms cancellation aborts while graph errors select full; its fallback and cancellation regressions passed.
```

1. **Strengths**
   - Reuses the existing generator through bounded configuration.
   - `tests/affected-config-test.py:173–178` verifies absent assets and reproduces the original embed failure.
   - The regression asserts exact selection after preparation and rejects a broken import afterward.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Explicit qualification against provider `fa9761e6` passed, including 732 audited group/file inputs.
   - The new regression failed with the base declaration in a temporary module.
   - Provider fallback and cancellation tests passed.
   - Default provider discovery visibly skipped; explicitly selecting the incompatible provider correctly failed.
   - Reader inventory and `git diff --check` passed. No tracked files changed.

6. **Architectural notes**
   - **ARCH-DRY: pass** — existing generator and provider reused.
   - **ARCH-PURE: pass** — added IO stays in integration qualification.
   - **ARCH-PURPOSE: pass** — preparation reaches the graph owner before discovery.
   - **ARCH-EVOLVE: pass** — additive configuration; no decoder or rewrite changes.

7. **Plan revision recommendations:** None. Preserve the unchecked operator-verification item until its evidence exists.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"90a00c7b-862d-44be-81fe-7e447147abb3","repository":"github.com/xianxu/pair","issue_id":"000738","recipe":"small-diff-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/5077219bab1126df","reviewed_head":"71623e8e30520581a091fdd4536230ed43190e09","slot":"pair:7","requested_at":"2026-10-21T09:53:37.548218Z"},"phase":"finished","pid":8132,"started_at":"2026-10-21T09:53:37.740615Z","finished_at":"2026-10-21T09:55:39.352051Z","session_id":"01a15de1-713f-7a32-a8a6-f3ffcb616ebc","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.691918833s
head: "71623e8e30520581a091fdd4536230ed43190e09"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
