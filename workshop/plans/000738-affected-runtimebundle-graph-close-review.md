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
