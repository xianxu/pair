# Boundary Review — pair#524 (whole-issue close)

| field | value |
|-------|-------|
| issue | 524 — broadcast-web: a note crop can catch a fade-in animation (dark crop); settle or finish animations in the box before capture |
| repo | pair |
| issue file | workshop/issues/000524-broadcast-web-a-note-crop-can-catch-a-fade-in-animation-dark-crop-settle-or-finish-animations-in-the-box-before-capture.md |
| boundary | whole-issue close |
| milestone | — |
| window | addd45e19d528471163e0d6803e8cb65718b89a7..bff299dae247d10846bc6bba22846d09f69fd7d0 |
| command | sdlc close --issue 524 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-12T23:35:53-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The viewport-preserving capture path, metadata propagation, and compatibility handling are well covered. One correctness gap blocks shipment: animations affecting overflowing descendants can be skipped and falsely reported as settled. The affected Go packages pass, but independent browser verification was blocked by Chrome crashing during startup.

1. **Strengths**

   - Screenshot capture disables `captureBeyondViewport`; the browser regression checks pixels and animation identity.
   - Settlement and clipping metadata propagate through recorder, Cast, validation, and storage. Older manifests retain unknown settlement.
   - Polling has explicit deadlines, cancellation handling, and viewport/document checks. README and atlas describe the changed behavior.

2. **Critical findings**

   - [capture_ready.js:58](/Users/xianxu/Library/Caches/sdlc/review-checkouts/6fb8d5e15fb8be81/cmd/internal/webcast/recorder/capture_ready.js:58) — **ARCH-PURPOSE:** The probe excludes an animation solely because its target’s bounding box misses the crop. A zero-size parent with `overflow: visible` can still fade a visible child inside the crop. The probe returns `pending:false`, causing Go to publish `settled` while that fade remains active. Account for affected descendants and overflowing pseudo-elements within the inspection budget; report uncertainty when influence cannot be determined.

3. **Important findings:** None additional.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Passed: submission, recorder, submission builder, and webcast packages.
   - An independent Node fixture exercising the production probe fails for all three overflowing-parent cases: finite, paused, and infinite animations.
   - Existing fixtures cover enclosing ancestors, not overflowing descendants.
   - Both live capture tests failed before opening a page because Chrome’s GPU process crashed; disabling GPU rendering did not resolve this. Browser pixel and mutation checks remain unverified in this review.
   - Artifact inventory tests also failed because generated runtime assets are absent and related exception entries appear stale. These failures are not established as regressions from this diff.
   - Checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared settlement validation and crop intersection.
   - **ARCH-PURE — pass:** Geometry/schema logic remains separate from browser observation.
   - **ARCH-PURPOSE — flag:** Animation influence detection misses overflowing descendants.
   - **ARCH-MOCK — pass structurally:** Existing injected CDP seam, controlled fixtures, and live conformance tests; live execution was blocked.
   - **ARCH-CONSTRAINTS — pass:** Explicit polling, traversal, animation, pixel, and overall limits.
   - **ARCH-SECURE — pass:** Geometry and metadata validation precede evidence acceptance; tests use temporary profiles.
   - **ARCH-ORDER — pass:** Sequential capture preserves document barriers and rejects viewport drift.
   - **ARCH-FUNERAL — pass:** No new durable artifact family; metadata uses the existing bounded store.

7. **Plan revision recommendations**

   Add a `## Revisions` entry enumerating animation influence cases—direct targets, enclosing ancestors, overflowing descendants, and pseudo-elements—and the bounded handling and regression coverage for each.

```findings
findings:
  - id: new
    severity: Critical
    family: animation-influence-coverage
    title: |
      Target-only geometry can falsely label an animated crop settled
    detail: |
      cmd/internal/webcast/recorder/capture_ready.js:58 skips animations whose target box misses the crop, even when an overflowing descendant inside the crop inherits the target's animated opacity. A production-probe fixture returns pending:false for finite, paused, and infinite animations on such a parent. ARCH-PURPOSE: account for descendant and pseudo-element influence within the inspection budget, conservatively report uncertainty where necessary, and add failing regression coverage including browser pixels.
```

---

## Re-review — 2026-10-12T23:54:29-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 524 — broadcast-web: a note crop can catch a fade-in animation (dark crop); settle or finish animations in the box before capture |
| repo | pair |
| issue file | workshop/issues/000524-broadcast-web-a-note-crop-can-catch-a-fade-in-animation-dark-crop-settle-or-finish-animations-in-the-box-before-capture.md |
| boundary | whole-issue close |
| milestone | — |
| window | addd45e19d528471163e0d6803e8cb65718b89a7..8c4f794d6b8df7e7fdaa70420e593f4597052217 |
| command | sdlc close --issue 524 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-12T23:54:29-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The capture path preserves viewport geometry, propagates validated settlement evidence, and passes the affected package tests. BR-1’s correction has meaningful regression coverage, but its exclusion rule still mistakes an element’s box for its painted footprint in remaining cases. That correctness gap blocks shipment. Browser pixel verification was unavailable because Chrome crashed before opening a page.

```findings
dispose:
  - id: BR-1
    disposition: not-addressed
    note: |
      ARCH-PURPOSE: capture_ready.js:63-71 still excludes off-box leaves whose box shadows or filters paint into the crop. A production-probe fixture returned pending:false with no reason for finite, paused, and infinite animations on such a leaf. Existing child/shadow/pseudo fixtures pass and fail against the pre-fix probe, confirming a real but incomplete correction. Keep family animation-influence-coverage open; establish and test a conservative painted-influence rule rather than adding another isolated exception.
```

1. **Strengths**

   - Screenshot capture uses `captureBeyondViewport:false`, with document and viewport checks around capture.
   - Settlement vocabulary, unavailable-crop invariants, legacy JSON compatibility, and manifest propagation have focused tests.
   - Open-shadow enumeration and polling have explicit bounds; browser animations are observed without mutation.
   - README and atlas describe the changed viewport policy and evidence semantics.

2. **Critical findings**

   - **BR-1 remains open — `cmd/internal/webcast/recorder/capture_ready.js:63–71` (ARCH-PURPOSE).** An empty element can paint outside its border box through `box-shadow`; animating its opacity changes those pixels. The probe checks children and pseudo-elements but then skips that element before examining animation state. For a leaf at viewport coordinates `[-100,20,40,30]` with `box-shadow:110px 0 white`, the shadow covers the fixture’s crop, yet finite, paused, and infinite cases all return `pending:false`.
   
     Fix the family’s rule: exclude an animation only when its painted influence is safely outside the crop; otherwise conservatively wait or report uncertainty within the existing budgets. Enumerate remaining paint-expansion mechanisms, including shadows and filters, and add deterministic plus browser-pixel regressions. This is a continuation of BR-1, not a new finding.

3. **Important findings**

   None separately raised.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - All four requested affected Go packages passed.
   - Production JavaScript fixtures passed.
   - Running the current regression fixture against the pre-correction probe failed on overflowing-child influence, confirming that the correction is exercised.
   - The additional leaf-shadow fixture reproduced false settlement in three animation states.
   - Live capture regression failed during Chrome startup with a GPU-process crash; no browser pixel assertions ran. Full-suite claims were not independently rerun.
   - Repository files remained unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** one observation helper and one screenshot path.
   - **ARCH-PURE — pass:** geometry/schema logic remains separate from browser IO; declared core entities exist.
   - **ARCH-PURPOSE — flag:** BR-1’s painted-influence class remains incompletely handled.
   - **ARCH-MOCK — pass:** production and controllable test flows share the Caller seam; live conformance tests exist.
   - **ARCH-CONSTRAINTS — pass:** bounded polling, animation enumeration, traversal, and capture sizes.
   - **ARCH-SECURE — pass:** observed geometry and metadata are validated before use.
   - **ARCH-ORDER — pass:** sequential capture, cancellation, identity barriers, and injected viewport changes cover the added sequencing.
   - **ARCH-FUNERAL — pass:** no new durable artifact family or independent background worker.

7. **Plan revision recommendations**

   Add a `## Revisions` entry replacing the claim that plain off-box leaves are safely irrelevant with a conservative painted-influence contract. Document the enumerated mechanisms and regression coverage, and update the matching atlas statement.

---

## Re-review — 2026-10-13T07:53:28-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 524 — broadcast-web: a note crop can catch a fade-in animation (dark crop); settle or finish animations in the box before capture |
| repo | pair |
| issue file | workshop/issues/000524-broadcast-web-a-note-crop-can-catch-a-fade-in-animation-dark-crop-settle-or-finish-animations-in-the-box-before-capture.md |
| boundary | whole-issue close |
| milestone | — |
| window | 805031105e60cc0eae0a37adee4b937b39c21f49..f5338dbc5ad7996f2ce6cb30d861d662acddb7b2 |
| command | sdlc close --issue 524 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T07:53:28-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The implementation matches the revised, approved document-wide observation policy and addresses BR-1 without another paint-geometry exception. No blocking code findings remain. Affected package tests and mutation checks passed; confidence is limited because live Chrome crashed during startup, preventing independent pixel verification in this review.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      capture_ready.js:44-63 removes target-geometry exclusions across supported animation roots. The pinned Node regression passes; substituting the prior probe fails, including the leaf-shadow assertion at capture_ready_test.js:96. Live fixtures cover overflowing children, pseudo-elements, box-shadow and drop-shadow pixels, but Chrome startup failed before those assertions could execute here.
```

1. **Strengths**

   - Conservative observation resolves the entire known influence class rather than accumulating CSS exceptions (`cmd/internal/webcast/recorder/capture_ready.js:44`).
   - Bounded polling distinguishes timeout evidence from cancellation and preserves explicit uncertainty (`cmd/internal/webcast/recorder/capture_ready.go:87`).
   - Capture preserves viewport geometry and uses the existing document-scoped screenshot helper (`cmd/internal/webcast/recorder/submission.go:181`).
   - Metadata validation, propagation, old-bundle compatibility, README and atlas updates accompany the behavior change.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Passed all four affected packages: submission, recorder, webcast/submission and webcast.
   - Confirmed the production JavaScript regression fails with the prior probe and passes at pinned HEAD. Temporary mutation fixtures were removed.
   - Both `TestLiveSubmissionCaptureReady` and `TestLiveSubmissionCrop` failed before opening a page: Chrome reported “GPU process isn't usable.” Their pixel/timeline assertions were inspected but not independently executed successfully.
   - Full repository suite was not rerun; implementor logs were not treated as independent verification.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared crop validation, intersection and screenshot boundaries remain centralized.
   - **ARCH-PURE — pass:** Geometry and schema rules remain pure; browser observation belongs to integration code.
   - **ARCH-PURPOSE — pass:** Document-wide observation eliminates the known geometry-exclusion family; viewport-preserving capture addresses the restart mechanism.
   - **ARCH-MOCK — pass:** Production and stateful fake use the same CDP seam; live conformance fixtures exist. Execution limitation noted above.
   - **ARCH-CONSTRAINTS — pass:** Polling, animation enumeration and traversal have explicit bounds and uncertainty outcomes.
   - **ARCH-SECURE — pass:** Browser replies and evidence receive validation; tests use temporary browser profiles and storage.
   - **ARCH-ORDER — pass:** Sequential capture retains identity barriers, cancellation and viewport checks; no persistent animation ownership is introduced.
   - **ARCH-FUNERAL — pass:** No new durable artifact family; added metadata uses existing bounded submission storage.

7. **Plan revision recommendations:** None required. The latest approval revision explicitly supersedes geometry-based relevance and records integration with the screenshot-suppression helper.

---

## Re-review — 2026-10-13T11:28:20-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 524 — broadcast-web: a note crop can catch a fade-in animation (dark crop); settle or finish animations in the box before capture |
| repo | pair |
| issue file | workshop/issues/000524-broadcast-web-a-note-crop-can-catch-a-fade-in-animation-dark-crop-settle-or-finish-animations-in-the-box-before-capture.md |
| boundary | whole-issue close |
| milestone | — |
| window | 844d29e698e481712f6a58f37f04e4d717a405f3..a5edec14048bde1abd649e89ab789cd99c1323f9 |
| command | sdlc close --issue 524 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T11:28:20-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The pinned implementation matches the revised capture and fresh-send contracts. BR-1 is verified with a failing pre-fix regression, and no blocking code findings remain. Confidence is limited by this environment’s Chrome startup failure, which prevented independent live pixel/timeline verification.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      capture_ready.js observes animations without target-geometry exclusions. The BR-1 overflowing-child fixture passes at pinned HEAD and fails against the pre-fix probe with “child running overflow cannot be settled.”
```

1. **Strengths**

   - Capture preserves the viewport, validates actual clipping, and propagates settlement evidence through the durable manifest (`cmd/internal/webcast/recorder/submission.go:115`, `cmd/internal/webcast/cast_submission.go:121`).
   - Supersession preserves delivery truth, refuses stale begins, and permits genuine late acknowledgments. Publication-failure tests exercise restart and retry ordering (`cmd/internal/submission/store.go:451`).
   - Independent acknowledgment receipts and epoch fencing protect newer drafts; adapter tests cover stale preparation callbacks and edits during delivery (`nvim/dom_submission_state.lua:55`, `nvim/dom_submission.lua:47`).
   - README and atlas changes document viewport limitations, uncertainty, fresh-send behavior, and recovery.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Passed: submission, recorder, webcast/submission, and webcast Go packages; focused couchtty submission tests after generating runtime assets in a disposable copy.
   - Passed: Lua state/adapter tests, sender tests under Neovim, and the production DOM-chord headless suite.
   - Verified BR-1 RED/GREEN against the pre-fix probe. Disposable files were removed.
   - Both live Chrome tests failed before opening a page: Chrome reported an unusable GPU process. Pixel/timeline assertions therefore did not execute.
   - The broader transaction shell suite could not build in the untouched checkout because generated runtime assets were absent. No full-suite pass is claimed.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** shared manifest writer, delivery transitions, and screenshot helper remain authoritative.
   - **ARCH-PURE — Pass:** geometry and disposition logic are pure; browser observation and persistence remain integration boundaries.
   - **ARCH-PURPOSE — Pass:** document-wide observation addresses the animation-influence class, rather than another geometry exception.
   - **ARCH-MOCK — Pass:** injected CDP, store-failure, and scheduler seams exercise production paths; live conformance tests exist, with execution limited as noted above.
   - **ARCH-CONSTRAINTS — Pass:** polling, traversal, animation, receipt, and storage bounds are explicit.
   - **ARCH-SECURE — Pass:** capture metadata validation and immutable target guards remain enforced.
   - **ARCH-ORDER — Pass:** explicit transitions preserve uncertain delivery, fence stale callbacks, and support late acknowledgment recovery.
   - **ARCH-FUNERAL — Pass:** receipts retire boundedly; durable evidence retains existing storage caps and cleanup ownership.

7. **Plan revision recommendations:** No substantive revision required. The later revisions explicitly supersede the earlier geometry exclusions and discard proposal.

---

## Re-review — 2026-10-13T11:36:13-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 524 — broadcast-web: a note crop can catch a fade-in animation (dark crop); settle or finish animations in the box before capture |
| repo | pair |
| issue file | workshop/issues/000524-broadcast-web-a-note-crop-can-catch-a-fade-in-animation-dark-crop-settle-or-finish-animations-in-the-box-before-capture.md |
| boundary | whole-issue close |
| milestone | — |
| window | 844d29e698e481712f6a58f37f04e4d717a405f3..a5edec14048bde1abd649e89ab789cd99c1323f9 |
| command | sdlc close --issue 524 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T11:36:13-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The pinned changes match the amended capture and fresh-send requirements. BR-1 is addressed with executable regression evidence; no new blocking defects were found. Confidence is limited because both live-browser tests failed during Chrome startup, before exercising capture.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      capture_ready.js:44–63 removes target-geometry exclusions across supported animation roots. The pinned Node regression passes and fails when the preceding geometry-based probe is restored in a disposable copy.
```

1. **Strengths**
   - Document-wide animation observation handles uncertainty explicitly without modifying page styles or timelines.
   - Capture preserves viewport geometry and validates settlement/clipping metadata through durable storage.
   - Supersession preserves delivery truth and orders durable markers before successor publication (`cmd/internal/submission/store.go:451`).
   - Independent acknowledgment receipts fence stale and duplicate callbacks (`nvim/dom_submission_state.lua:54`).

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**
   - Passed: submission, recorder, webcast/submission and webcast Go packages; focused couchtty submission tests; Lua model/adapter/sender tests; complete headless submission transaction suite.
   - Mutation checks: restoring the previous animation probe fails its regression; restoring `captureBeyondViewport=true` fails the recorder capture regression.
   - Runtime assets were generated only in a disposable copy, subsequently removed.
   - Live readiness and crop tests could not execute their assertions: Chrome reported an unusable GPU process, including with `--disable-gpu`. Full repository suite was not rerun.

6. **Architecture**
   - **ARCH-DRY — pass:** shared manifest writer and resolution scheduling.
   - **ARCH-PURE — pass:** geometry, validation and transitions remain separable from browser/storage adapters.
   - **ARCH-PURPOSE — pass:** addresses capture-induced restart and the approved document-wide influence rule.
   - **ARCH-MOCK — pass:** injected stateful seams and browser conformance tests exist; live execution limitation noted above.
   - **ARCH-CONSTRAINTS — pass:** bounded polling, traversal, receipts and server concurrency.
   - **ARCH-SECURE — pass:** capture values and persisted evidence are validated; target guards remain.
   - **ARCH-ORDER — pass:** explicit transitions, publication failure tests and controlled stale-callback coverage.
   - **ARCH-FUNERAL — pass:** receipts retire; staging is cleaned; retained evidence uses existing storage bounds.

7. **Plan revisions:** None required. Authoritative revisions explain the expanded scope, and README/atlas updates cover the resulting behavior.
