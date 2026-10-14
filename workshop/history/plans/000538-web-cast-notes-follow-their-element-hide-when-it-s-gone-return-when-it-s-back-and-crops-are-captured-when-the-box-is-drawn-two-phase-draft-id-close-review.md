# Boundary Review — pair#538 (whole-issue close)

| field | value |
|-------|-------|
| issue | 538 — web cast: notes follow their element (hide when it's gone, return when it's back) and crops are captured when the box is drawn (two-phase draft id) |
| repo | pair |
| issue file | workshop/issues/000538-web-cast-notes-follow-their-element-hide-when-it-s-gone-return-when-it-s-back-and-crops-are-captured-when-the-box-is-drawn-two-phase-draft-id.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6788a2dffb4d3383d32fae0b4dbb77f0b5bfe7bb..e5a01d9414a754ec7ed4ef94730bb722df635823 |
| command | sdlc close --issue 538 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T18:09:58-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The capture ownership and submission paths have substantial regression coverage, and the targeted Go and JavaScript suites pass. Shipping is blocked by an operator placement defect and contradictions in the required core-concept inventory. Real-Chrome verification was attempted but failed during browser startup because its GPU process crashed.

1. **Strengths**

   - Capture work runs outside the recorder actor, avoiding capture/barrier re-entry deadlocks.
   - Save and publication recovery distinguish unknown outcomes from confirmed absence and preserve evidence through interrupted transfers.
   - Submission selects historical open notes, uses stored captures, and applies privacy per captured origin. README and atlas updates cover the new surface.

2. **Critical findings**

   - **Operator placement lacks connected-node identity — ARCH-PURPOSE.** At `cmd/internal/webcast/operator.js:88`, `createPlacement` receives no `original` resolver. Consequently, every operator note uses replacement matching, even when its original element remains connected. I reproduced two cases using the production placement function: an unchanged common-text “Save” anchor and an unchanged anchor on a page exceeding 2,000 nodes. Viewer placement returned the box; operator placement returned `null`. Supply trusted, document-scoped original-node identity to the operator adapter, retain conflict checks, and add adapter regressions for both cases.

   - **Core-concept inventory contradicts the implementation — ARCH-PURE.** At `workshop/plans/000538-anchored-notes-draw-capture-plan.md:23`, `AnchorEvidence` and `MatchDecision` do not exist at the listed paths; the implementation uses `AnchorCandidate`, `ScoreAnchor`, and `ChooseAnchor`. The `CaptureRecord` row points to `model.go`, while `captureRecord` lives in `service.go:66` and includes admission/cancellation handles. Rows 40 and 44 also mark `board/wire.go` and three capture-engine files modified, although their pinned diffs are empty. Correct the inventory and add a Revisions entry distinguishing pure ownership state from integration records and reused engine files. Critical under the requested core-concept consistency gate; this correction itself is documentary.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed: eight targeted Go packages, including recorder, webcast, notedraft, board, recording, and submission.
   - Passed: all **234 webcast JavaScript tests** and pinned-range `git diff --check`.
   - Independently reproduced the operator/viewer placement discrepancy without modifying files.
   - `TestLiveDrawCaptureProtocolsAndStoredSubmission` failed before opening a page: Chrome reported “GPU process isn't usable.” Browser conformance therefore remains unverified in this review.
   - Full-suite/build results recorded by the implementor were not independently rerun. No prior findings required disposition.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared lifecycle service, browser controller, and scoring fixtures.
   - **ARCH-PURE — flag:** core-concept inventory misidentifies the implemented pure/integration boundary.
   - **ARCH-PURPOSE — flag:** connected-anchor behavior is incomplete on the operator surface.
   - **ARCH-MOCK — pass:** injected capture/writer seams and controlled recovery fixtures; live verification has the limitation above.
   - **ARCH-CONSTRAINTS — pass:** explicit queue, operation, storage, and candidate limits.
   - **ARCH-SECURE — pass:** authenticated ownership checks, bounded reads, and private capture storage.
   - **ARCH-ORDER — pass:** explicit ownership transitions and controlled uncertain-save/publication tests.
   - **ARCH-FUNERAL — pass:** draft expiry, resolution cleanup, reference retirement, and transfer recovery are implemented.

7. **Plan revision recommendations**

   Add a `## Revisions` entry correcting entity names, locations, classifications, and unchanged/reused files. Record the operator original-node identity mechanism and its common-text/large-document regression coverage when fixed.

```findings
findings:
  - id: new
    severity: Critical
    family: connected-anchor-authority
    title: |
      Operator placement never supplies original-node identity
    detail: |
      cmd/internal/webcast/operator.js:88 omits createPlacement's original resolver, forcing unchanged connected anchors through replacement scoring. Production-function probes show common-text anchors and anchors on pages above the 2,000-node scan limit disappear only on the operator surface. Supply trusted document-scoped original-node identity and test both cases through the adapter (ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: core-concept-inventory-consistency
    title: |
      The core-concept table does not describe the pinned implementation
    detail: |
      workshop/plans/000538-anchored-notes-draw-capture-plan.md:23-44 lists nonexistent AnchorEvidence/MatchDecision entities, places CaptureRecord in model.go instead of service.go, and marks unchanged board/wire.go and capture-engine files modified. Correct names, paths, pure/integration classifications and statuses in a Revisions entry; the required core-concept gate treats these contradictions as Critical (ARCH-PURE).
```

---

## Re-review — 2026-10-13T18:22:36-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 538 — web cast: notes follow their element (hide when it's gone, return when it's back) and crops are captured when the box is drawn (two-phase draft id) |
| repo | pair |
| issue file | workshop/issues/000538-web-cast-notes-follow-their-element-hide-when-it-s-gone-return-when-it-s-back-and-crops-are-captured-when-the-box-is-drawn-two-phase-draft-id.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6788a2dffb4d3383d32fae0b4dbb77f0b5bfe7bb..4ad125006c605beedf3bae51ca96bf65c09b7be4 |
| command | sdlc close --issue 538 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-13T18:22:36-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

Both prior findings are addressed, and no new blocking defects were identified. The implementation connects operator placement to trusted original-node identity, preserves capture ownership through uncertain outcomes, and documents the new surfaces. Confidence is limited by Chrome startup failures that prevented fresh live-browser verification.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      operator.js:93 supplies the original resolver through recorder/operator_original.go:13 and chrome_operator_original.go:66. Removing that resolver in a disposable copy made five adapter regressions fail, including common-text originals and pages exceeding the scan cap; the pinned implementation passes.
  - id: BR-2
    disposition: addressed
    note: |
      workshop/plans/000538-anchored-notes-draw-capture-plan.md:209 explicitly supersedes the stale tables. The replacement inventory matches the inspected definitions, classifies captureRecord in service.go as integration state, and correctly identifies board/wire.go and the capture-engine files as reused unchanged.
```

1. **Strengths**

   - Original-node authority is document/context scoped; stale identities and rebinding are rejected.
   - Shared placement logic keeps viewer and operator confidence rules aligned.
   - Capture ownership distinguishes uncertain writes/publications from confirmed absence, with recovery tests.
   - README and atlas updates explain draw-time capture, hidden notes, schema v3, privacy, and legacy fallback.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage**

   - Passed all **242 webcast Node tests**.
   - Passed eight affected Go packages: notedraft, notes, board, recording, recorder, webcast, webcast/submission, and submission.
   - Passed focused original-node bridge/recorder race tests and range whitespace checks.
   - BR-1 mutation check produced **five expected failures**; disposable files were removed.
   - Both requested live Chrome cases failed before opening a page with GPU-process errors; one also reported sandbox initialization failure. No browser assertions ran.
   - Full-suite starvation remains explicitly documented as “not run,” with the plan’s approved verification disposition.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared placement and capture ownership.
   - **ARCH-PURE — pass:** value-based scoring/transitions separated from integration adapters.
   - **ARCH-PURPOSE — pass:** both annotation surfaces and cross-document submission are wired.
   - **ARCH-MOCK — pass:** controlled capture/writer/CDP seams and browser conformance tests exist; live execution limitation noted above.
   - **ARCH-CONSTRAINTS — pass:** bounded candidate scans, capture queue, storage, and bridge retries.
   - **ARCH-SECURE — pass:** authenticated ownership, scoped debugger identities, and bounded persisted-input validation.
   - **ARCH-ORDER — pass:** explicit ownership transitions and uncertainty-preserving reconciliation.
   - **ARCH-FUNERAL — pass:** expiry, resolution cleanup, publication transfer, and debugger-handle release have defined paths.

7. **Plan revision recommendations:** None; the authoritative revision matches the inspected implementation.
