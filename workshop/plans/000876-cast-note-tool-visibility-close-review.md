# Boundary Review — pair#876 (whole-issue close)

| field | value |
|-------|-------|
| issue | 876 — Cast Note tool is missing or undiscoverable in the live tool column |
| repo | pair |
| issue file | workshop/issues/000876-cast-note-tool-visibility.md |
| boundary | whole-issue close |
| milestone | — |
| window | 56986a072ef70b7ee0d00c317395088b2bb44915..95f06244a39371e5d3ec8f68183ec22e4b6a2dcd |
| command | sdlc close --issue 876 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T15:35:57-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The owner/viewer authority separation passes focused behavioral and race checks, and README/atlas updates cover the changed semantics. No runtime defect was demonstrated. SHIP is blocked by contradictions in the mandatory Core concepts tables and unavailable native UI verification.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: core-concept-traceability
    title: |
      Core concepts tables contradict the delivered file locations and statuses
    detail: |
      workshop/plans/000876-note-tool-owner-plan.md:25 names a new board/annotation_authority.go, but the authority is implemented in board/annotation.go:21. Line 40 names owner_annotation_live_test.go, but the delivered test is operator_owner_live_test.go; line 38 marks annotate.css modified although it is unchanged. Reconcile all table paths/statuses and add a ## Revisions entry. Critical follows the review's explicit Core concepts consistency rule (ARCH-PURPOSE).
  - id: new
    severity: Important
    kind: unverifiable
    family: native-conformance-evidence
    title: |
      Native owner-tool reachability could not be verified on this host
    detail: |
      WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run '^TestOperatorOwnerLiveAnnotationOff$' -count=1 -timeout=75s -v failed at operator_owner_live_test.go:48 before opening a page: Chrome reported “GPU process isn't usable.” This prevents verification of actual Note/Pointer hit-testing, capture/save and short-viewport reachability (ARCH-MOCK). Supply a successful run on pinned head 95f06244a39371e5d3ec8f68183ec22e4b6a2dcd from a working Chrome environment, with command, environment and output; the retained PASS log does not identify its tested head.
```

1. **Strengths**

   - `board/annotation.go:129` centralizes bidirectional surface checks, preventing owner authority from entering viewer paths and viewer permits from entering owner paths.
   - `recorder/operator.go:164` mints authority after context verification while retaining the request’s originating generation.
   - Signed fallback and unknown-acknowledgment tests exercise restoration, attribution and exactly-once recovery.
   - README and both atlas updates describe owner availability, viewer grants and cleanup fences.

2. **Critical findings**

   The plan-table contradiction above requires documentation reconciliation; it does not require moving otherwise sound implementation code.

3. **Important findings**

   Native conformance remains unverified because Chrome failed before UI startup. Host evidence can answer this finding without a production change.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Focused Go tests passed across Board, recorder, notedraft and Chrome. Recorder/notedraft packages and affected Board/Chrome checks passed with race detection. JavaScript checks passed: 87 tests total. The broader Board race run timed out in unchanged `TestSketchDeliveryOversizePatchAndBootstrapEncodingBound`; that run is inconclusive. No source files were modified.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — pass:** shared authority and surface checking.
   - **ARCH-PURE — pass:** inspected authority checks use in-memory state; external work remains behind existing seams.
   - **ARCH-PURPOSE — flag:** Core concepts traceability contradicts delivery.
   - **ARCH-MOCK — flag:** fake-backed checks pass; native qualification is unverified here.
   - **ARCH-CONSTRAINTS — pass:** counted delivery tests exercise C/2C contexts and unchanged refreshes.
   - **ARCH-SECURE — pass:** authority follows verified CDP provenance; HTTP viewer admission remains separate.
   - **ARCH-ORDER — pass:** originating-generation and final-write checks remain, with controlled ordering tests.
   - **ARCH-FUNERAL — pass:** no new runtime durable artifact family or worker lifecycle.
   - **ARCH-EVOLVE — pass:** no persisted or wire-format change.

7. **Plan revision recommendations**

   Add a `## Revisions` entry recording the actual authority location, native test filename and unchanged CSS status; update the tables consistently. Preserve operator acceptance as an outstanding landing postcondition.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"fc83e266-c1ef-47b8-ab85-90f5d14478d2","repository":"github.com/xianxu/pair","issue_id":"000876","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/7eb9f3c233f45d96","reviewed_head":"95f06244a39371e5d3ec8f68183ec22e4b6a2dcd","slot":"pair:4","requested_at":"2026-10-24T22:32:36.584965Z"},"phase":"finished","pid":57816,"started_at":"2026-10-24T22:32:36.994929Z","finished_at":"2026-10-24T22:35:53.427922Z","session_id":"01a1700b-66cb-70b2-9694-3fd49d34604a","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 11.943771375s
head: "95f06244a39371e5d3ec8f68183ec22e4b6a2dcd"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000413
