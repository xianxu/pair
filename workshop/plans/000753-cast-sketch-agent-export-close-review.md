# Boundary Review — pair#753 (whole-issue close)

| field | value |
|-------|-------|
| issue | 753 — cast-sketch: operator CLI queries and historical image export |
| repo | pair |
| issue file | workshop/issues/000753-cast-sketch-agent-export.md |
| boundary | whole-issue close |
| milestone | — |
| window | d2540fd6ae6eaa06e8a90217b285c22116247558..813350c3bbf43e64f2e37d1381a99035840ce371 |
| command | sdlc close --issue 753 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T20:13:17-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The main query, export, submission, and privacy paths are implemented, and the checked-in component tests pass. Four additional regression probes fail on the pinned head: preview recovery stalls, anchor facts lose priority, region-only notes become unsendable, and null pressure becomes fabricated evidence. The plan inventory also needs correction. Native Chrome verification could not complete successfully.

1. **Strengths**

   - Exact source and history record indices prevent later same-timestamp events from replacing selected evidence.
   - `EvidenceFiles` centralizes note and sketch image references across store publication and reads.
   - Nested unknown-field preservation, exact-version delivery suppression, and CDP cancellation have substantive tests.
   - README, atlas, and the shipped guide document the new surfaces and privacy restrictions.

2. **Critical findings**

   The machine-readable block below contains the precise locations, reproductions, and fix sketches.

3. **Important findings**

   Native rendering, preview, and fidelity conformance remain unverified because Chrome failed during execution; see the `unverifiable` finding below.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed all tests in query, sketchimage, submission, CDP, board, webcast/submission, webcast, couchtty, and couchcmd.
   - Passed focused race tests for CDP writes and preview transitions/integration.
   - Temporary Go test overlays reproduced all four behavioral defects. They were removed; the checkout remains unchanged.
   - Existing preview tests omit an unchanged-identity notification during an active publication. Element-fact tests omit byte-budget starvation by unrelated nodes; note tests omit region-only anchors; codec tests omit null tuple members.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared evidence inventory, pressure painter, renderer, and delivery authority.
   - **ARCH-PURE — flag:** the plan labels filesystem-backed delivery derivation PURE.
   - **ARCH-PURPOSE — flag:** preview recovery and anchor-priority defects undermine the promised handoff.
   - **ARCH-MOCK — flag:** useful stateful seams exist, but native conformance remains unverified.
   - **ARCH-CONSTRAINTS — pass:** explicit batch, byte, pixel, and deadline bounds are present.
   - **ARCH-SECURE — flag:** null pressure is silently converted into numeric evidence.
   - **ARCH-ORDER — flag:** cancellation can leave Publishing authoritative after its operation has ended.
   - **ARCH-FUNERAL — pass:** temporary export cleanup, bounded store admission, and owned browser cleanup are implemented.
   - **ARCH-EVOLVE — flag:** unknown-field preservation works, but known-field type strictness is incomplete.

7. **Plan revision recommendations**

   Add `## Revisions` entries recording the corrected cancellation transition, anchor-priority budgeting, region-only note compatibility, and strict numeric decoding. Reconcile the Core concepts table with actual symbols, paths, and PURE/INTEGRATION boundaries. Keep native/operator acceptance explicitly pending until supported by passing evidence.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: canceled-operation-state-settlement
    title: |
      Canceled preview publications can remain permanently Publishing
    detail: |
      cmd/internal/couchtty/console_submission_preview.go:154 cancels and joins a publication after a refresh notification, then continues without settling its state. If host and candidates are unchanged, NextPreview ignores both repeated inputs and PreviewRefresh cannot leave Publishing. Production SubmissionPreviewUpdates preloads precisely such a notification at chrome_submission_preview.go:125; receiver-progress notifications can also reproduce it. A held publisher plus a buffered unchanged-host notification deterministically failed the review probe: no replacement publication occurred and preparation remained refused. ARCH-ORDER: settle the canceled publication through the transition model before refreshing, and cover initial, duplicate, and catchup notifications.
  - id: new
    severity: Critical
    kind: defect
    family: mandatory-evidence-budget-priority
    title: |
      Unrelated DOM attributes consume the budget before mandatory anchor facts
    detail: |
      cmd/internal/webcast/query/element_facts.go:98 spends the shared 64 KiB attribute budget during whole-DOM traversal, before facts() prioritizes anchors. A probe with 70 preceding unrelated elements carrying bounded data attributes and a later anchor with a valid 1024-byte ID lost that ID, although the selected anchor facts easily fit the output budget. ARCH-PURPOSE: reserve or allocate declared-fact bytes in anchor-first, ancestor-next, nearest-context order; test starvation by unselected nodes.
  - id: new
    severity: Critical
    kind: defect
    family: optional-anchor-compatibility
    title: |
      Valid region-only notes now fail submission preparation
    detail: |
      cmd/internal/webcast/query/querier.go:225 always inserts n.At.Node into the anchor list, including zero. The resulting synthetic node-zero fact is rejected by cmd/internal/submission/sketch.go:138. recording/record.go:548 explicitly permits a note with a region and no node; a review probe successfully encoded those notes but Build failed with "invalid declared element fact". Preserve the existing note contract by omitting absent anchor handles while retaining region intersection facts, and add a region-only submission regression.
  - id: new
    severity: Critical
    kind: defect
    family: strict-known-field-decoding
    title: |
      Null pressure passes decoding and validation as fabricated zero pressure
    detail: |
      cmd/internal/submission/codec.go:103 checks tuple length but not numeric member types, then json.Unmarshal at line 89 converts null into a float's zero value. Replacing [20,20,0.2] with [20,20,null] in a valid sketch manifest passed both DecodePreserving and Validate in the review probe, yielding pressure 0. ARCH-SECURE/ARCH-EVOLVE: reject null and other invalid types for required numeric evidence, sweep the known scalar and tuple schemas, and add strict-type regressions.
  - id: new
    severity: Critical
    kind: defect
    family: core-concept-traceability
    title: |
      The Core concepts inventory contradicts implemented entities and boundaries
    detail: |
      workshop/plans/000753-cast-sketch-agent-export-plan.md:70-83 lists PreviewSelection in sketch.go, a PURE DeliveredSketchSet in sketch.go, query/render_sketch.go, and webcast/submission/sketch.go. Those definitions/files do not exist at those locations: PreviewSelection lives in preview.go, delivery derivation is the filesystem-backed Store.DeliveredSketchVersions in store.go, and export is implemented through sketch_export.go and sketch_image_export.go. ARCH-PURE: revise the inventory to name actual entities and paths, separate pure selection from integration delivery derivation, and record the correction under Revisions.
  - id: new
    severity: Important
    kind: unverifiable
    family: native-conformance-evidence
    title: |
      Native historical rendering and fidelity preservation remain unverified
    detail: |
      Attempted WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run 'TestLiveSketchImage|TestOperatorSketchLiveDrawing|TestLiveFidelity' -count=1 -v. Fidelity fixtures, broken-render detection, operator drawing, and historical pressure/batch rendering failed with Chrome/CDP EOF; operator Chrome reported "GPU process isn't usable", and historical rendering also reported "operation not permitted". Cancellation cleanup passed. ARCH-MOCK: provide passing results for these checks on the final source-equivalent head in a supported native environment, including pressure pixels, eight-image timing, fidelity preservation, and joined cleanup.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"20c83b16-0476-44c9-92fa-909102ca7784","repository":"github.com/xianxu/pair","issue_id":"000753","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/bfa179fdcc579e45","reviewed_head":"813350c3bbf43e64f2e37d1381a99035840ce371","slot":"pair:9","requested_at":"2026-10-22T03:07:49.277355Z"},"phase":"finished","pid":29862,"started_at":"2026-10-22T03:07:49.44463Z","finished_at":"2026-10-22T03:13:16.906191Z","session_id":"01a16194-4716-7602-8b3a-edf738cbaead","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.771238s
head: "813350c3bbf43e64f2e37d1381a99035840ce371"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000409
