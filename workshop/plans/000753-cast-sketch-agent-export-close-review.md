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
