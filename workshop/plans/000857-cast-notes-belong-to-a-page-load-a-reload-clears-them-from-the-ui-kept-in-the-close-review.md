# Boundary Review — pair#857 (whole-issue close)

| field | value |
|-------|-------|
| issue | 857 — Cast notes belong to a page load; a reload clears them from the UI (kept in the recording) |
| repo | pair |
| issue file | workshop/issues/000857-cast-notes-belong-to-a-page-load-a-reload-clears-them-from-the-ui-kept-in-the.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7cb0b672ebb6aca081ae63f19c27b0aafc45270f..f5ffa47e6942019bc54a639630cfa1c8fe37a75b |
| command | sdlc close --issue 857 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T04:39:55-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: medium
```

The implementation passed the core Go suites, focused race tests, and all 615 browser tests. No runtime correctness defect was established. Two boundary requirements remain: reconcile the plan’s Core concepts table with the pinned diff, and obtain independent native qualification after Chrome failed during startup in this review environment.

```findings
findings:
  - id: new
    severity: Critical
    family: core-concept-traceability
    title: |
      Core concepts table contradicts the delivered file ownership
    detail: |
      workshop/plans/000857-page-load-feedback-plan.md:48–51 marks recording/catchup.go, web/feedback-entries.js and operator_live_test.go as modified, but none changes in the pinned range; chrome_live_test.go does not exist. Native qualification is instead added in page_load_live_test.go. Under the requested Core concepts gate, reconcile these rows and add a Revisions entry explaining which existing consumers required no modification (ARCH-PURPOSE).
  - id: new
    severity: Important
    kind: unverifiable
    family: native-conformance-evidence
    title: |
      Native page-load lifecycle could not be independently qualified
    detail: |
      WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run '^TestPageLoadLiveFeedbackLifecycle$' -count=1 -v -timeout=150s failed at page_load_live_test.go:53 before opening a page: Chrome reported GPU process exit_code=6 and “GPU process isn't usable.” Reload, tab-return and retained-history acceptance therefore remain unverified here. Supply a non-skipped successful run bound to the pinned head, or rerun review with working native Chrome; the committed implementor log alone does not resolve this inspection failure (ARCH-MOCK).
  - id: new
    severity: Minor
    family: documentation-behavior-parity
    title: |
      Historical liveNotes documentation still describes the old predicate
    detail: |
      atlas/webcast-format.md:229 says liveNotes filters through State.DocSeg. web/annotate.js:138 now uses page-load membership for recording Models. Update the description to explain creation-segment load membership and same-load tab restoration (ARCH-DRY).
```

1. **Strengths**

   - Exact target/loader/root identity avoids URL inference and JavaScript integer rounding.
   - Segment tests exercise strict owned fields, recursive extension preservation, and older preserving-schema rewrites.
   - Recorder tests control capture and sink-write ordering, including supersession, detach, and close drain.
   - UI and submission filtering preserve unfiltered historical queries and previously admitted delivery recovery.
   - README and atlas updates cover the new user-visible behavior and compatibility limits.

2. **Critical findings**

   - Reconcile the Core concepts ownership table as detailed above. This is a plan correction, not a request to modify already-sufficient consumers.

3. **Important findings**

   - Native qualification remains blocked by Chrome startup failure. No production fix is inferred from that environmental failure.

4. **Minor findings**

   - Correct the stale `liveNotes` description at `atlas/webcast-format.md:229`.

5. **Test coverage notes**

   Passed independently:

   - Recording, recorder, Board, query, sketchimage, submission, webcast, notedraft, and artifactpath Go packages.
   - Focused recorder/Board page-load race tests.
   - All 615 browser tests, zero skips.

   Native lifecycle failed before scenario execution. The checkout remains unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared membership and provenance helpers; minor documentation inconsistency noted.
   - **ARCH-PURE — pass:** deterministic identity/transition logic with thin IO ownership.
   - **ARCH-PURPOSE — flag:** Core concepts traceability requires reconciliation.
   - **ARCH-MOCK — flag:** stateful fake coverage passes; native conformance remains unverified.
   - **ARCH-CONSTRAINTS — pass:** counted projections and serialized capture preserve declared bounds.
   - **ARCH-SECURE — pass:** strict identity parsing and conservative unknown-scope admission.
   - **ARCH-ORDER — pass:** reservations separate historical writes from current UI confirmation; adversarial ordering tests pass.
   - **ARCH-FUNERAL — pass:** metadata follows recording retention; owner cleanup releases runtime references.
   - **ARCH-EVOLVE — pass:** additive metadata and nested unknown-field preservation have regression coverage.

7. **Plan revision recommendations**

   Add a `## Revisions` entry identifying `page_load_live_test.go` as the new native fixture and correcting unchanged consumer paths/statuses. Record successful native rerun evidence when available.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"6564a6e9-672e-4626-85cb-75665ec4166c","repository":"github.com/xianxu/pair","issue_id":"000857","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/e356e9a0092e6986","reviewed_head":"f5ffa47e6942019bc54a639630cfa1c8fe37a75b","slot":"pair:4","requested_at":"2026-10-24T11:32:46.963438Z"},"phase":"finished","pid":67228,"started_at":"2026-10-24T11:32:47.111139Z","finished_at":"2026-10-24T11:35:43.42524Z","session_id":"01a16daf-4d1f-7cd1-9c9f-b7b89ff3dfa0","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.751922584s
head: "f5ffa47e6942019bc54a639630cfa1c8fe37a75b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T04:47:27-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 857 — Cast notes belong to a page load; a reload clears them from the UI (kept in the recording) |
| repo | pair |
| issue file | workshop/issues/000857-cast-notes-belong-to-a-page-load-a-reload-clears-them-from-the-ui-kept-in-the.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7cb0b672ebb6aca081ae63f19c27b0aafc45270f..eadf97328b414bc627c434b9a561f3b181f21408 |
| command | sdlc close --issue 857 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T04:47:27-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation matches the page-load design in the inspected paths, and component, browser and focused race tests pass. BR-1 and BR-3 are addressed. BR-2 remains blocking: native Chrome again crashed before opening a page, preventing independent lifecycle qualification.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Plan integration rows now distinguish reused consumers from changed files and identify page_load_live_test.go as the native fixture. The Revisions passage at workshop/plans/000857-page-load-feedback-plan.md:250 explains unchanged ownership. recording/catchup.go retains complete Seg records; passing replay/compaction tests qualify that reuse (ARCH-PURPOSE).
  - id: BR-2
    disposition: not-addressed
    note: |
      At pinned head eadf97328b414bc627c434b9a561f3b181f21408, WEBCAST_LIVE_CHROME=1 go test ./cmd/internal/webcast -run '^TestPageLoadLiveFeedbackLifecycle$' -count=1 -v -timeout=150s failed at page_load_live_test.go:53 before page creation: GPU process exit_code=6, “GPU process isn't usable.” Inspected the committed successful rerun log and confirmed subsequent changes are documentation/evidence only, but the implementor log alone does not resolve the required independent qualification. BR-2 remains kind: unverifiable; provide independently verifiable source-bound host evidence or rerun review with working native Chrome (ARCH-MOCK).
  - id: BR-3
    disposition: addressed
    note: |
      atlas/webcast-format.md:227–231 now describes creation-segment load membership, same-load restoration and the standalone adapter fallback, matching web/annotate.js:138 and web/page-load.js. This is a prose-only correction supported by the code and passing browser tests (ARCH-DRY).
```

1. **Strengths**

   - Exact load identity is separate from activation segments, preserving same-load tab return.
   - Recorder tests control snapshot and sink ordering, including supersession during reserved writes.
   - Recursive unknown-field preservation and strict identity validation have substantive round-trip tests.
   - README and atlas document current-load selection, pending scope and retained history.

2. **Critical findings:** None remaining.

3. **Important findings:** BR-2 remains open at `cmd/internal/webcast/page_load_live_test.go:53`. Restore native qualification and rerun the lifecycle scenario.

4. **Minor findings:** None remaining.

5. **Test coverage notes**

   Passed recording, recorder, Board, query, sketchimage and submission package suites; all 615 browser tests; focused recorder/Board race tests; and targeted module assembly tests. Native lifecycle qualification failed during Chrome startup. No repository files were changed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared membership and provenance helpers.
   - **ARCH-PURE — pass:** identity, codecs and transitions are directly testable without IO.
   - **ARCH-PURPOSE — pass:** UI and selection filter current-load feedback; queries retain history.
   - **ARCH-MOCK — flag:** fake coverage passes; native conformance remains unverified under BR-2.
   - **ARCH-CONSTRAINTS — pass:** bounded requests, counted work and existing deadlines retained.
   - **ARCH-SECURE — pass:** strict owned identity validation; no URL-derived authority.
   - **ARCH-ORDER — pass:** explicit transitions and controlled adverse-order tests.
   - **ARCH-FUNERAL — pass:** owner cleanup joins work; metadata follows recording retention.
   - **ARCH-EVOLVE — pass:** additive metadata and recursive extension preservation are tested.

7. **Plan revision recommendations:** None additional. The ownership reconciliation is accurate; native acceptance remains pending verification.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"984e204a-0da9-451c-b6c7-bcbe4239e7f5","repository":"github.com/xianxu/pair","issue_id":"000857","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/5280461bc6ef126c","reviewed_head":"eadf97328b414bc627c434b9a561f3b181f21408","slot":"pair:4","requested_at":"2026-10-24T11:43:48.906502Z"},"phase":"finished","pid":72588,"started_at":"2026-10-24T11:43:49.040883Z","finished_at":"2026-10-24T11:46:20.428631Z","session_id":"01a16db9-66a3-79b1-a4cf-61b824a6deb7","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.091305208s
head: "eadf97328b414bc627c434b9a561f3b181f21408"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T04:56:53-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 857 — Cast notes belong to a page load; a reload clears them from the UI (kept in the recording) |
| repo | pair |
| issue file | workshop/issues/000857-cast-notes-belong-to-a-page-load-a-reload-clears-them-from-the-ui-kept-in-the.md |
| boundary | whole-issue close |
| milestone | — |
| window | 7cb0b672ebb6aca081ae63f19c27b0aafc45270f..6100fb6a8fea05d1b5822677e999f17e1badd7d8 |
| command | sdlc close --issue 857 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T04:56:53-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The implementation matches the documented page-load contract: live feedback and submission selection use exact load identity, while recording queries retain history. No new blocking defects found. Local native qualification again failed before page creation, but the independently retained, source-bound host evidence resolves BR-2.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Verified the reconciled Core concepts and Integration points tables against the pinned diff and filesystem: unchanged catch-up and generic adapters are identified as reused, and submission provenance and page_load_live_test.go are included.
  - id: BR-3
    disposition: addressed
    note: |
      atlas/webcast-format.md:227 now describes creation-segment load membership and the standalone adapter fallback, matching liveNotes in cmd/internal/webcast/web/annotate.js:136 and the shared page-load helpers.
  - id: BR-2
    disposition: withdrawn
    note: |
      Inspected independent-host metadata, passing non-skipped lifecycle output, and cleanup evidence. The log SHA256 matches fd63b1446d35513c95bfc31edc608061d6e5edf5101ccbdc96d7e0fac8b13c27. Git confirms only tracker/evidence changes between qualified source eadf97328b414bc627c434b9a561f3b181f21408 and pinned HEAD. My exact native command reproduced the pre-page Chrome GPU exit_code=6 failure; it does not contradict the independent lifecycle measurement.
```

1. **Strengths**

   - Exact target/loader/root identity stays lossless across Go, JSON, and JavaScript; unknown identity never matches.
   - Recorder reservation and confirmation distinguish preserved historical writes from current UI authority.
   - Board selection, note projections, and sketch delivery consistently filter by current load.
   - Codec tests cover nested extension preservation, strict identity validation, and older preserving-reader rewrites.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Independently passed:

   - Recording, recorder, Board, query, sketchimage, and submission packages.
   - Full webcast package.
   - All 615 Node tests, zero skipped.
   - Focused recorder/Board ordering tests under the race detector.

   Inspected deterministic capture/write barriers, stale completion rejection, detach, close drain, same-load restoration, and catch-up preservation. Native host evidence covers reload, pause/resume, tab return, app mutation, same-URL new target, UI selection, and retained history. No runtime correction was claimed for the prior findings, so fix-removal testing was unnecessary.

6. **Architectural notes for upcoming work**

   - **ARCH-DRY — Pass:** Shared identity, membership, and source-facts helpers serve consumers.
   - **ARCH-PURE — Pass:** Identity/codecs and recorder transitions are directly testable; IO remains in owners.
   - **ARCH-PURPOSE — Pass:** Live consumers and press selection are scoped; historical queries remain complete.
   - **ARCH-MOCK — Pass:** Stateful fake CDP exercises production seams; native evidence qualifies real behavior.
   - **ARCH-CONSTRAINTS — Pass:** Serialized snapshots, counted workload checks, and existing queue/deadline bounds remain.
   - **ARCH-SECURE — Pass:** Persisted identity is validated strictly; no added credential surface found.
   - **ARCH-ORDER — Pass:** Typed transitions, private reservations, generation fences, and controlled interleavings enforce ownership.
   - **ARCH-FUNERAL — Pass:** Metadata follows recording retention; runtime cells and UI caches have cleanup paths.
   - **ARCH-EVOLVE — Pass:** Additive metadata preserves nested unknown fields; older-reader limitations are documented.

   README and atlas updates cover the changed user behavior and architectural surface.

7. **Plan revision recommendations:** None. Remaining unchecked stage/merge work is explicitly pending, not claimed delivered.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"29782dc2-e470-4498-b0ec-2818180cf046","repository":"github.com/xianxu/pair","issue_id":"000857","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/43211b260c17c4a5","reviewed_head":"6100fb6a8fea05d1b5822677e999f17e1badd7d8","slot":"pair:4","requested_at":"2026-10-24T11:54:26.866769Z"},"phase":"finished","pid":52876,"started_at":"2026-10-24T11:54:27.021521Z","finished_at":"2026-10-24T11:56:36.360994Z","session_id":"01a16dc3-22ce-72e0-8007-92015f0b1c5f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.415830792s
head: "6100fb6a8fea05d1b5822677e999f17e1badd7d8"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-24T06:07:48-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 857 — Cast notes belong to a page load; a reload clears them from the UI (kept in the recording) |
| repo | pair |
| issue file | workshop/issues/000857-cast-notes-belong-to-a-page-load-a-reload-clears-them-from-the-ui-kept-in-the.md |
| boundary | whole-issue close |
| milestone | — |
| window | c79121698d445dea557528ce8e34149abf376c27..6615515753ca49f840209747317995d81913d24b |
| command | sdlc close --issue 857 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-24T06:07:48-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The interdiff repairs the synthetic Couch recording fixture without weakening production admission or changing requests and behavioral assertions. Documentation matches the existing unknown-load policy. The Revisions additions record qualification work, with no scope change. No new defects found.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      The reconciled ownership tables remain intact; this interdiff introduces no new production entities or ownership changes.
  - id: BR-2
    disposition: withdrawn
    note: |
      Retained independent-host metadata, non-skipped lifecycle output, and cleanup evidence remain intact. The log hash matches fd63b1446d35513c95bfc31edc608061d6e5edf5101ccbdc96d7e0fac8b13c27; this interdiff changes no native lifecycle or production code.
  - id: BR-3
    disposition: addressed
    note: |
      atlas/webcast-format.md:231–235 retains the corrected creation-segment load predicate. The added legacy refusal passage at lines 96–98 matches Board.pageLocked and the server refusal text.
```

1. **Strengths**
   - Fixture identity matches the builder’s target and supplies the authority normally provided by `commitTopSnapshot` (`console_webnotes_test.go:69`).
   - HTTP requests and menu/copy/resolve assertions remain unchanged.
   - Failure diagnostics now include the response body and check read errors (`console_webnotes_test.go:110`).

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   
   Independently passed:
   - Three affected Couch tests under `-race`.
   - Complete Couch package: 65.496s.
   - Existing unknown-load fresh-feedback refusal test.
   - `make test-affected-inventory` and `git diff --check`.

   Inspected retained RED evidence. No production behavior changed in this window.

6. **Architectural notes**
   - **ARCH-DRY — Pass:** Shared fixture repairs all three affected tests.
   - **ARCH-PURE — Pass:** Changes remain in integration-test setup and diagnostics.
   - **ARCH-PURPOSE — Pass:** Fix preserves downstream behavior assertions and trusted admission.
   - **ARCH-EVOLVE — Pass:** No persisted format or rewrite changes.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5ebe5930-229c-44a4-87bf-cc0f25873ef7","repository":"github.com/xianxu/pair","issue_id":"000857","recipe":"fix-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/9b2dd91d4ef60070","reviewed_head":"6615515753ca49f840209747317995d81913d24b","slot":"pair:4","requested_at":"2026-10-24T13:05:20.694219Z"},"phase":"finished","pid":69951,"started_at":"2026-10-24T13:05:20.866992Z","finished_at":"2026-10-24T13:07:37.732211Z","session_id":"01a16e04-0ce6-76b0-88cb-603b322d9e49","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 20.574810125s
head: "6615515753ca49f840209747317995d81913d24b"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000413
