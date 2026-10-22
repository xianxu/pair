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

---

## Re-review — 2026-10-21T20:39:28-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 753 — cast-sketch: operator CLI queries and historical image export |
| repo | pair |
| issue file | workshop/issues/000753-cast-sketch-agent-export.md |
| boundary | whole-issue close |
| milestone | — |
| window | d2540fd6ae6eaa06e8a90217b285c22116247558..4c2bd430be755eae08053496e0babe913f6382a9 |
| command | sdlc close --issue 753 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T20:39:28-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The prior behavioral fixes have meaningful regressions, the corrected inventory matches the implementation, and retained native evidence answers BR-6. A new probe found one blocking gap: omitted required source fields still become fabricated zero-valued evidence.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Cancellation now settles the exact publication through PreviewPublishFailed. The checked-in regression passes on head and fails with pre-fix production code for all five unchanged-input notification cases.
  - id: BR-2
    disposition: addressed
    note: |
      Selected facts now consume the output budget in anchor-first order with mandatory-handle reservations. The starvation regression passes on head and fails with the previous implementation; unused measurement fields were retained solely to compile the counterfactual.
  - id: BR-3
    disposition: addressed
    note: |
      Absent anchor handles are omitted by both note projection and the shared extractor. The region-only Build/ContextAt regression passes on head and fails with the pre-fix implementations.
  - id: BR-4
    disposition: addressed
    note: |
      Numeric tuple members now reject null before Go decoding. The null-pressure regression passes on head and fails with the previous codec, which produces pressure zero and passes Validate. Missing-field handling is a separate remaining instance reported below.
  - id: BR-5
    disposition: addressed
    note: |
      Plan lines 62–91 now locate PreviewSelection in preview.go, distinguish pure SelectSketchPreview from INTEGRATION Store.DeliveredSketchVersions, and name the actual export files. These match the definitions in preview.go, preview_selection.go, store.go, sketch_export.go and sketch_image_export.go; Revisions records the correction.
  - id: BR-6
    disposition: withdrawn
    note: |
      Inspected native-85406e681.log and verified SHA256 d7942734429e891879c962c39102fc7781eaf7f3c4d9daa39e82440efae7799e. The requested native checks passed in 77.227s, including pressure/batch rendering, fidelity fixtures, broken-render detection and cancellation cleanup. The eight-image measurement was 1.796754583s. git diff confirms the reviewed head changes only issue prose after that source commit.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: strict-known-field-decoding
    title: |
      Missing required sketch source fields become fabricated zero-valued evidence
    detail: |
      cmd/internal/submission/codec.go:189 validates only fields present in the JSON object; it never checks required-member presence. Three independent probes deleting sketches[0].source.t, record_index, or scroll from the valid checked-in fixture all passed DecodePreserving and Validate: source time 10 became 0, record index 2 became 0, and absent scroll became [0,0]. This violates the strict required-field and original-source evidence contracts (ARCH-SECURE, ARCH-EVOLVE). This is the 2nd finding in family strict-known-field-decoding: do not patch only these three fields. Define required versus optional members across the new evidence schema, enforce presence before decoding, and sweep omission regressions across that inventory while preserving intentional legacy defaults; measured prevalence here is 3/3 probed source members.
```

1. **Strengths**

   - Controlled preview tests exercise cancellation, late success, host retirement and joined replacement through the production manager.
   - Anchor budgeting preserves mandatory handles and prevents unrelated declarations from consuming selected evidence capacity.
   - Shared image rendering, evidence-file enumeration and durable delivery authority avoid parallel implementations.
   - README, atlas and shipped guidance cover the new commands, preview and privacy behavior.

2. **Critical findings**

   - Required-member validation is missing at `cmd/internal/submission/codec.go:189`. Semantic checks at `sketch.go:77` accept fabricated zeros. Add explicit required-member metadata and rejection before unmarshalling.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - All nine affected Go packages passed.
   - Focused CDP, codec and preview race checks passed.
   - All four prior behavioral regressions failed with their fixes removed through temporary overlays.
   - Three new omission probes failed on head. Temporary artifacts were removed; the checkout remains unchanged.
   - Native results were verified from the retained log and source equivalence, rather than rerun here.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared renderer, painter, evidence inventory and delivery authority.
   - **ARCH-PURE — pass:** corrected inventory separates deterministic logic from filesystem/browser effects.
   - **ARCH-PURPOSE — pass:** operator handoff, historical export and generic element facts are implemented.
   - **ARCH-MOCK — pass:** stateful injected renderer and portable store have native conformance evidence.
   - **ARCH-CONSTRAINTS — pass:** explicit budgets and measured eight-image rendering.
   - **ARCH-SECURE — flag:** omitted source members become fabricated evidence.
   - **ARCH-ORDER — pass:** cancellation settles through the transition model; controlled tests cover adverse ordering.
   - **ARCH-FUNERAL — pass:** owned cleanup, temporary-stage removal and bounded existing storage.
   - **ARCH-EVOLVE — flag:** unknown-field preservation works, but required-member strictness remains incomplete.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining the required-member inventory for new evidence objects, intentional legacy omission defaults, and deletion-based regression coverage. Keep the documented operator acceptance and recorded-suite gates explicitly pending.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"4a17e301-afe7-4298-9075-c5cf61a13aed","repository":"github.com/xianxu/pair","issue_id":"000753","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/f8d411e70d5c53d7","reviewed_head":"4c2bd430be755eae08053496e0babe913f6382a9","slot":"pair:9","requested_at":"2026-10-22T03:35:27.204559Z"},"phase":"finished","pid":5571,"started_at":"2026-10-22T03:35:27.439719Z","finished_at":"2026-10-22T03:39:27.259077Z","session_id":"01a161ad-93b8-7e93-a1dc-3a79e861cac9","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.610221791s
head: "4c2bd430be755eae08053496e0babe913f6382a9"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T20:50:41-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 753 — cast-sketch: operator CLI queries and historical image export |
| repo | pair |
| issue file | workshop/issues/000753-cast-sketch-agent-export.md |
| boundary | whole-issue close |
| milestone | — |
| window | d2540fd6ae6eaa06e8a90217b285c22116247558..052330f68d4f81b9baff10d90e3559c6a5abaf55 |
| command | sdlc close --issue 753 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T20:50:41-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

BR-7 is addressed across the required-member inventory. The regressions fail against the pre-fix codec and pass at the pinned head. Review of the issue, plan, implementation, docs, and targeted integration checks found no additional blocking defects. The checkout remains unchanged.

```findings
dispose:
  - id: BR-7
    disposition: addressed
    note: |
      codec.go:85-101 declares required members across sketch evidence; filterManifest checks presence before typed decoding at lines 203-207. Tests exercise 120 required omissions, optional/legacy compatibility, and six Store read paths. Running the new regressions with the pre-fix codec through a temporary Go overlay fails, including all three original source-member probes; pinned-head tests pass.
```

1. **Strengths**

   - Required-field checks preserve explicit zero values while rejecting absent evidence.
   - Preview publication and preparation share exact-version selection, with controlled cancellation and stale-publication tests.
   - One evidence-file inventory covers publication and Store reads; nested unknown-field preservation has durable-rewrite coverage.
   - README, atlas, and the shipped Couch guide document the new operator and CLI surfaces.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed full submission, query, sketchimage, submission-builder, and CDP package tests; targeted webcast/Couch submission and preview tests, including `TestNode`; and race checks for submission, CDP, preview, and Cast sketch integration.

   The pre-fix codec countercheck failed as expected. Native Chrome conformance, the recorded suite, and real operator acceptance were not rerun here; previously disposed findings remain undisturbed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared rendering, selection, and file enumeration.
   - **ARCH-PURE — pass:** Deterministic projection, validation, and transitions remain separate from IO.
   - **ARCH-PURPOSE — pass:** Sketch evidence reaches the existing operator transaction and local export.
   - **ARCH-MOCK — pass:** Stateful renderer and controllable publication/CDP seams exercise production boundaries.
   - **ARCH-CONSTRAINTS — pass:** Explicit batch, image, byte, deadline, and declaration bounds.
   - **ARCH-SECURE — pass:** Strict evidence decoding, confined assets/output, and original-origin privacy.
   - **ARCH-ORDER — pass:** Explicit preview transitions and exact publication/freeze fencing.
   - **ARCH-FUNERAL — pass:** Owned cleanup, bounded existing storage, and explicit operator-owned exports.
   - **ARCH-EVOLVE — pass:** Additive evidence preserves nested extensions while validating known fields strictly.

7. **Plan revision recommendations:** None. BR-7’s revisions match the implementation; remaining acceptance and landing steps remain explicitly unchecked.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c51462ad-6097-4714-be11-f1f600ea7f11","repository":"github.com/xianxu/pair","issue_id":"000753","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/4a867103b057ef59","reviewed_head":"052330f68d4f81b9baff10d90e3559c6a5abaf55","slot":"pair:9","requested_at":"2026-10-22T03:47:51.997749Z"},"phase":"finished","pid":26082,"started_at":"2026-10-22T03:47:52.170972Z","finished_at":"2026-10-22T03:50:40.974503Z","session_id":"01a161b8-f06b-7eb2-9d9e-069e4223328a","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.224280875s
head: "052330f68d4f81b9baff10d90e3559c6a5abaf55"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-21T21:24:48-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 753 — cast-sketch: operator CLI queries and historical image export |
| repo | pair |
| issue file | workshop/issues/000753-cast-sketch-agent-export.md |
| boundary | whole-issue close |
| milestone | — |
| window | 489393f492663d03bd7eaa82952ede6ed3e4a321..37ef740f65a78dc037696aa62189f71c58c8a22e |
| command | sdlc close --issue 753 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-21T21:24:48-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned interdiff fixes the two structural violations without changing production behavior or issue scope. No new defects found. No prior findings remain open; existing dispositions stand. The checkout is unchanged.

```findings
{}
```

1. **Strengths**
   - The expiry test now waits on owned cancellation while retaining late-success rejection and refresh-recovery assertions (`cmd/internal/couchtty/console_submission_preview_test.go:393`).
   - The cleanup comment correctly attaches to `closeJoined`; its budget and implementation remain unchanged (`cmd/internal/webcast/replay_image.go:176`).
   - Existing repository guards cover both corrected classes.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage:** Passed preview tests, the expiry regression under race detection, replay cleanup tests, godoc and catalog guards, affected-reader inventory, and diff whitespace checks. Full-suite and native acceptance were not rerun for this narrow window.

6. **Architecture**
   - **ARCH-DRY — pass:** Reuses existing guards and cancellation seam.
   - **ARCH-PURE — pass:** No new business logic or IO coupling.
   - **ARCH-PURPOSE — pass:** Both structural corrections are complete; no scope expansion.
   - **ARCH-EVOLVE — pass:** No persisted-format changes.

   No new user-facing or architectural surface requires README or atlas updates.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"39f83b2c-5183-46f4-99a0-dd10f0ad369c","repository":"github.com/xianxu/pair","issue_id":"000753","recipe":"fix-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/0bb12c97bd990fee","reviewed_head":"37ef740f65a78dc037696aa62189f71c58c8a22e","slot":"pair:9","requested_at":"2026-10-22T04:23:40.295333Z"},"phase":"finished","pid":27949,"started_at":"2026-10-22T04:23:40.538714Z","finished_at":"2026-10-22T04:24:47.542095Z","session_id":"01a161d9-b952-7601-b9ce-a74a716b4d37","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.358177958s
head: "37ef740f65a78dc037696aa62189f71c58c8a22e"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T10:20:39-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 753 — cast-sketch: operator CLI queries and historical image export |
| repo | pair |
| issue file | workshop/issues/000753-cast-sketch-agent-export.md |
| boundary | whole-issue close |
| milestone | — |
| window | 39be359a3c55bcce5fd485548cd01e9d318d8b5e..d6c24469c2599d1a6d4ac45257afd8be8d9fa09f |
| command | sdlc close --issue 753 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T10:20:39-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation delivers historical queries, shared viewport evidence, durable delivery filtering, and an acknowledged feedback preview. Focused tests passed. One correctness defect blocks shipping: shared-page rendering silently reanchors ink to replacement elements despite the plan’s explicit refusal contract.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Cancellation settles the exact publication through PreviewPublishFailed; controlled unchanged-refresh, terminal cancellation and expired-publication regressions passed.
  - id: BR-2
    disposition: addressed
    note: |
      Selected facts reserve mandatory anchor capacity before ancestors/context; unselected-declaration and mandatory-handle budget regressions passed.
  - id: BR-3
    disposition: addressed
    note: |
      Zero is excluded from declared anchor identities; region-only Build/ContextAt regression passed, including under race detection.
  - id: BR-4
    disposition: addressed
    note: |
      Codec preflight rejects null numeric evidence before typed decoding; null-pressure and known-scalar regressions passed.
  - id: BR-5
    disposition: addressed
    note: |
      The corrected Core concepts inventory separates pure selection from Store IO and identifies the actual preview, export and renderer locations.
  - id: BR-6
    disposition: withdrawn
    note: |
      Retained native log SHA256 matches its evidence record; its source revision differs from pinned head only in evidence/tracker files. The log records passing shared-page, legacy rendering, cancellation and fidelity checks, with the stress-fixture qualification preserved.
  - id: BR-7
    disposition: addressed
    note: |
      Explicit required-member schemas are reachable through Store decoding; deletion, valid-zero, optional-omission and Store tampering regressions passed.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: historical-anchor-identity
    title: |
      Shared historical images silently move ink onto replacement elements
    detail: |
      cmd/internal/webcast/web/sketch-page-image.js:52 uses Renderer.placementOf without requiring the original recorded node. When that node is removed, annotation-placement.js:123-134 searches for a matching replacement. A read-only probe through the production Renderer.placementOf and preparePageInk removed node 7, inserted a different node with matching text, and successfully projected the original points onto the replacement. This contradicts plan line 295's explicit replaced-anchor refusal (ARCH-SECURE, ARCH-PURPOSE). Require connected original recorded-node identity for shared historical rendering and add a regression through the actual placement adapter; preserve the existing live annotation reanchoring policy.
  - id: new
    severity: Minor
    kind: defect
    family: handoff-contract-alignment
    title: |
      Agent-facing instructions retain obsolete per-sketch evidence references
    detail: |
      nvim/dom_submission_state.lua:43-44 describes sketch-N.png historical source images, while couchcmd/skills/couch/SKILL.md:667-670 directs readers to image.plan.paths. New shared items instead use sketch-page.png and sketch.paths. The canonical appended ReadingGuide and opening skill paragraph are correct, limiting the impact, but these conflicting instructions should distinguish shared and legacy variants (ARCH-PURPOSE). Update the emitted-prompt regression alongside the text.
```

1. **Strengths**

   - Exact record ordinals prevent later same-timestamp mutations from entering historical image scenes.
   - One `EvidenceFiles` inventory serves publication and Store validation for shared and legacy bundles.
   - Preview state is opaque, with controlled cancellation and stale-acknowledgment tests.
   - README and atlas updates cover the new commands, shared-image semantics, privacy, and retry behavior.

2. **Critical findings**

   - **Replacement-anchor acceptance:** [sketch-page-image.js:52](/Volumes/TestScratch/review-checkouts/cbf61589e9f832b6/cmd/internal/webcast/web/sketch-page-image.js:52). Add a strict original-node placement mode and test removal/replacement through the production adapter. The current preflight tests supply a callback that returns `null`, so they miss this behavior.

3. **Important findings**

   None.

4. **Minor findings**

   - Reconcile obsolete per-sketch instructions in the Lua handoff and shipped Couch guide, including the emitted-prompt regression.

5. **Test coverage notes**

   Passed:

   - Complete submission, query, sketchimage, board, and CDP package tests.
   - Focused webcast/couchtty sketch, submission, preview, feedback, replay, and Node tests.
   - Focused race checks, plus complete CDP and submission-builder race tests.
   - Neovim submission transaction, state, adapter, and transport checks.
   - `git diff --check`.

   The independent replacement-anchor probe reproduced the blocking defect. Native evidence was inspected and hash-checked; native Chrome tests were not rerun. The checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** shared file inventory, pressure painter, pixel fitting, and replay lifetime.
   - **ARCH-PURE — Pass:** deterministic selection, validation, geometry, and preview transitions remain separate from IO.
   - **ARCH-PURPOSE — Flag:** replacement-anchor acceptance violates the promised evidence contract; handoff instructions also need reconciliation.
   - **ARCH-MOCK — Pass:** injected renderer/publisher seams, stateful fakes, portable Store fixtures, and retained native conformance.
   - **ARCH-CONSTRAINTS — Pass:** explicit batch, image, recording, asset, and cleanup bounds.
   - **ARCH-SECURE — Flag:** matching text does not establish recorded-node identity.
   - **ARCH-ORDER — Pass:** publication identities and atomic freeze guard stale and canceled outcomes.
   - **ARCH-FUNERAL — Pass:** scoped renderer resources and existing bounded Store lifecycle.
   - **ARCH-EVOLVE — Pass:** strict variant validation and nested extension preservation have regression coverage.

7. **Plan revision recommendations**

   Add a `## Revisions` entry clarifying that shared historical placement requires the connected original recorded node; live annotation replacement matching remains a separate policy. Retain the existing refusal requirement and name the production-adapter regression.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"1c7433db-59e7-47b0-b653-b39692a8d3b4","repository":"github.com/xianxu/pair","issue_id":"000753","recipe":"milestone-review","round":5,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/cbf61589e9f832b6","reviewed_head":"d6c24469c2599d1a6d4ac45257afd8be8d9fa09f","slot":"pair:9","requested_at":"2026-10-22T17:16:31.57146Z"},"phase":"finished","pid":20988,"started_at":"2026-10-22T17:16:31.801659Z","finished_at":"2026-10-22T17:20:38.566348Z","session_id":"01a1649d-4ab7-7d42-945d-aad964ee1af9","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.361764167s
head: "d6c24469c2599d1a6d4ac45257afd8be8d9fa09f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T10:33:27-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 753 — cast-sketch: operator CLI queries and historical image export |
| repo | pair |
| issue file | workshop/issues/000753-cast-sketch-agent-export.md |
| boundary | whole-issue close |
| milestone | — |
| window | 39be359a3c55bcce5fd485548cd01e9d318d8b5e..892b1fc78543154c558fdcf561e5a3b6eff8b040 |
| command | sdlc close --issue 753 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T10:33:27-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The pinned implementation satisfies the reviewed shared-viewport contract. Both open findings are addressed with regressions that independently fail without their fixes. No new blocking findings emerged. README and atlas updates cover the new surface; renewed operator acceptance remains explicitly pending in Task9.

```findings
dispose:
  - id: BR-8
    disposition: addressed
    note: |
      sketch-page-image.js:46-57 routes production rendering through originalOnly placement. The actual Renderer adapter regression passes and fails on disconnected replacement when that option is removed in a disposable copy; live replacement placement remains covered.
  - id: BR-9
    disposition: addressed
    note: |
      dom_submission_state.lua:43-47 and couch SKILL.md:667-678 distinguish sketch-page.png/shared paths from legacy sketch-N.png/image.plan.paths. The emitted-prompt regression passes and fails with the prior prompt restored.
```

1. **Strengths**

   - Historical rendering now requires original recorded-anchor identity through the production adapter.
   - Exact source/finish witnesses and atomic preview freezing protect the displayed selection.
   - `EvidenceFiles` centralizes shared and legacy image references across Store consumers.
   - Strict codec tests cover required fields, invalid types, variant compatibility, and nested extension preservation.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed during review:

   - Six focused Go packages: submission, query, sketchimage, board, submission builder, and CDP.
   - Webcast and couchtty submission/preview/feedback tests.
   - Targeted race tests for submission, CDP, and couchtty.
   - All 497 browser tests.
   - Neovim submission transaction and adapter checks.
   - `git diff --check`.

   Both negative regression probes failed as expected; disposable copies were removed. Native conformance was assessed from the retained log, whose SHA-256 matches its evidence record. Its executable source is unchanged at the pinned head; native Chrome was not rerun here.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Shared evidence inventory, pressure painter, replay lifetime, and compatibility policy.
   - **ARCH-PURE — pass:** Selection, geometry, validation, and preview transitions remain separate from IO.
   - **ARCH-PURPOSE — pass:** Operator handoff, shared image, original evidence, and local export are implemented.
   - **ARCH-MOCK — pass:** Injected renderer/publisher seams have controlled fakes and portable Store fixtures.
   - **ARCH-CONSTRAINTS — pass:** Batch, pixels, bytes, input, and deadline limits have explicit refusal paths.
   - **ARCH-SECURE — pass:** Strict evidence decoding, confined publication, privacy validation, and historical identity checks.
   - **ARCH-ORDER — pass:** Tagged preview transitions and exact publication/freeze identities have cancellation and stale-event tests.
   - **ARCH-FUNERAL — pass:** Render resources have joined cleanup; durable evidence uses the existing bounded Store lifecycle.
   - **ARCH-EVOLVE — pass:** Shared and legacy variants remain distinct; nested unknown extensions survive Store rewrites.

   The inspected Core concepts entries align with implemented entities and boundaries.

7. **Plan revision recommendations:** None. Keep Task9’s remaining operator acceptance and recorded-check requirements open until their evidence is complete.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9e6cc63f-f024-4e09-a7f6-1a87fdd9bcee","repository":"github.com/xianxu/pair","issue_id":"000753","recipe":"milestone-review","round":6,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/49de3740666da5f2","reviewed_head":"892b1fc78543154c558fdcf561e5a3b6eff8b040","slot":"pair:9","requested_at":"2026-10-22T17:31:05.989662Z"},"phase":"finished","pid":87455,"started_at":"2026-10-22T17:31:06.139592Z","finished_at":"2026-10-22T17:33:26.633318Z","session_id":"01a164aa-a1d5-7940-b5ab-9c43b707df85","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.265617458s
head: "892b1fc78543154c558fdcf561e5a3b6eff8b040"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-22T11:38:39-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 753 — cast-sketch: operator CLI queries and historical image export |
| repo | pair |
| issue file | workshop/issues/000753-cast-sketch-agent-export.md |
| boundary | whole-issue close |
| milestone | — |
| window | fea66e36b0d25b3e37f2dc70a084226a8ed1697a..4fc20f722c297c8e1e0c1731f0ca6d443ffdf568 |
| command | sdlc close --issue 753 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-22T11:38:39-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation has strong historical-evidence validation, shared rendering seams, and durable delivery tests. One reproduced correctness bug blocks shipment: a transient viewer connection error removes confirmed display suppression while retaining the displayed recording, causing submitted feedback to reappear.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Canceled publications settle through PreviewPublishFailed before refresh; controlled cancellation, late-success and terminal-join regressions pass in couchtty.
  - id: BR-2
    disposition: addressed
    note: |
      Selected-output budgeting reserves mandatory anchor facts; unselected-declaration and all-mandatory-handle regressions pass in query.
  - id: BR-3
    disposition: addressed
    note: |
      Nonpositive absent anchors are excluded; region-only Build and capture-context regressions pass in webcast/submission.
  - id: BR-4
    disposition: addressed
    note: |
      Schema preflight rejects null numeric tuple members; null-pressure, known-scalar and Store read-path regressions pass.
  - id: BR-5
    disposition: addressed
    note: |
      The revised Core concepts inventory names preview_selection.go, sketch_export.go and the filesystem-backed DeliveredSketchVersions integration; those entities and boundaries exist.
  - id: BR-6
    disposition: withdrawn
    note: |
      Retain the prior withdrawal; this review does not reopen the disposed native-environment finding or claim a fresh native run.
  - id: BR-7
    disposition: addressed
    note: |
      Explicit required-member metadata is consumed by filterManifest; required-member deletion and Store missing-source regressions pass.
  - id: BR-8
    disposition: addressed
    note: |
      prepareHistoricalPageInk requests originalOnly placement; actual Renderer adapter tests reject disconnected/replaced originals and pass connected-original controls.
  - id: BR-9
    disposition: addressed
    note: |
      Lua handoff now describes sketch-page.png, sketch.index and paths alongside legacy image.plan.paths; the real Neovim transaction script passes.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: confirmed-display-continuity
    title: |
      Transient viewer disconnects repaint already-submitted feedback
    detail: |
      cmd/internal/webcast/web/viewer.js:138 calls feedbackMask.beginStream on every EventSource error, including automatic reconnect with readyState 0. beginStream clears the confirmed mask and invokes redraw callbacks, while this path retains the old model and page. A direct production-connect probe changed both exact noteVisible and sketchVisible from false to true and increased redraws from 1 to 2 immediately after the error. Submitted ink/pins therefore reappear throughout a connection outage, contradicting the plan's same-stream reconnect-retains-mask contract (ARCH-ORDER, ARCH-PURPOSE). Preserve the confirmed mask across transient reconnects, retaining revision fencing; replace it atomically when the reconnect's authoritative scope/full snapshot arrives. Add a production-connect regression covering the outage interval, repeated errors, stale revisions and recovery.
```

1. **Strengths**

   - Historical page rendering preflights every chosen path through the strict original-anchor adapter before painting.
   - The preserving codec enforces required members and known types while retaining nested extensions through Store rewrites.
   - Preview publication uses explicit pure transitions, exact acknowledgment identities, and controlled cancellation tests.
   - README and atlas changes cover the new CLI, shared-page bundle, preview, privacy, and live-mask surfaces.

2. **Critical findings**

   - **Confirmed display continuity:** [viewer.js:138](/Volumes/TestScratch/review-checkouts/b7209862b3272fbd/cmd/internal/webcast/web/viewer.js:138), calling [feedback-mask.js:30](/Volumes/TestScratch/review-checkouts/b7209862b3272fbd/cmd/internal/webcast/web/feedback-mask.js:30). Preserve confirmed suppression until authoritative replacement evidence arrives.

3. **Important findings**

   None.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed eight Go packages: submission, query, sketchimage, board, cdp, webcast, couchtty, and webcast/submission. Also passed 53 selected browser tests, the Neovim submission transaction script, and range whitespace checks.

   Existing reconnect tests cover stream replacement and eventual mask installation, but omit visibility during a transient automatic reconnect. The independent probe exposes that gap. No source files were changed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared evidence inventory, viewport compatibility, replay lifetime, and pressure painter.
   - **ARCH-PURE — pass:** deterministic selection, validation, and preview transitions have direct tests; IO remains behind integration seams.
   - **ARCH-PURPOSE — flag:** confirmed feedback reappears during reconnect.
   - **ARCH-MOCK — pass:** renderer seams have stateful fakes; Store tests use portable temporary storage.
   - **ARCH-CONSTRAINTS — pass:** explicit image, byte, batch, and deadline bounds have representative tests.
   - **ARCH-SECURE — pass:** strict evidence decoding, recorded-source validation, and confined export publication.
   - **ARCH-ORDER — flag:** transport uncertainty incorrectly discards confirmed display knowledge.
   - **ARCH-FUNERAL — pass:** projections share cast lifetime; renderer cleanup and temporary-artifact ownership are explicit.
   - **ARCH-EVOLVE — pass:** legacy/shared variants remain distinct, with nested-extension preservation and compatibility tests.

7. **Plan revision recommendations**

   Add a `## Revisions` entry defining transient reconnect as retaining the last confirmed mask until an authoritative full replacement arrives. Name the outage-interval regression and preserve stale-revision rejection across reconnect.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"f6f33495-d562-4ed6-97cd-cb1ca7ab6811","repository":"github.com/xianxu/pair","issue_id":"000753","recipe":"milestone-review","round":7,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/b7209862b3272fbd","reviewed_head":"4fc20f722c297c8e1e0c1731f0ca6d443ffdf568","slot":"pair:9","requested_at":"2026-10-22T18:35:33.2005Z"},"phase":"finished","pid":63408,"started_at":"2026-10-22T18:35:33.406808Z","finished_at":"2026-10-22T18:38:38.055582Z","session_id":"01a164e5-a5b3-7843-9582-7a0c4ea59218","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 4.043897666s
head: "4fc20f722c297c8e1e0c1731f0ca6d443ffdf568"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
