# Boundary Review — pair#494 (whole-issue close)

| field | value |
|-------|-------|
| issue | 494 — broadcast-web: the operator annotates in the cast Chrome itself (pointer strokes and notes, injected overlay) |
| repo | pair |
| issue file | workshop/issues/000494-broadcast-web-the-operator-annotates-in-the-cast-chrome-itself-pointer-strokes-and-notes-injected-overlay.md |
| boundary | whole-issue close |
| milestone | — |
| window | addd45e19d528471163e0d6803e8cb65718b89a7..c406696b642eed4a946fbf4cdd14d9eca70594f1 |
| command | sdlc close --issue 494 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-12T23:27:04-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared annotation UI, Board attribution, and capture-suppression integration are well aligned with the issue. Two behavioral defects block shipping: reconciliation can silently discard a different draft, and Escape misses page-focused input. The plan also misclassifies a DOM-dependent helper as PURE. Some verification remained blocked by the checkout and Chrome environment.

1. **Strengths**

   - Operator writes reuse Board validation and settle from recorder outcomes rather than enqueue success.
   - Prepared drafts retain recorded node identity; tests cover replaced paths, stale contexts, and ordering-time pause/driver refusals.
   - Fidelity and submission captures both use `CaptureScreenshot`, with confirmation checks and exact-context restoration.
   - README and atlas document the new controls, lifecycle, and recovery behavior.

2. **Critical findings**

   - **Different drafts can inherit another operation’s success** — `cmd/internal/webcast/operator.js:31–42` (**ARCH-ORDER**). After draft A times out, cancel it and save draft B. The transport queries A’s status and returns HTTP-style `204` when A was accepted; the dialog consequently closes B although B was never submitted. I reproduced this against the production transport: commands were only `save-note(A)` and `status(A)`, but B returned success. Bind reconciliation to the originating draft/action; acceptance must only complete that operation. Preserve unrelated drafts and add cancel/new-draft and resolve/save sequence regressions.

   - **Escape does not work when focus remains on the page** — `cmd/internal/webcast/web/annotate.js:591–593` (**ARCH-PURPOSE**). The operator passes a ShadowRoot as `root`; page-focused keyboard events never traverse it. Selecting Pen through Couch’s menu deliberately preserves focus, so Escape cannot perform the documented transition to Off. Listen at the document/window boundary and route applicable Escape events through the policy. Test page focus plus menu-selected Pen/Note.

   - **Core-concepts PURE classification contradicts the helper** — `workshop/plans/000494-operator-overlay-plan.md:23` (**ARCH-PURE**). The recorded path helper reads live DOM parents, children, attributes, shadow hosts, and the global document (`recorder/observer.js:53–69`). Classify it as INTEGRATION, or extract a deterministic core over supplied tree data. The requested core-concepts gate treats this contradiction as Critical.

3. **Important findings**

   None additional.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Passed: 123 webcast Node tests; Go Board, recorder, and webcast packages; pinned-range whitespace check.
   - The transport test currently endorses success after changed payload reconciliation, but omits cancellation followed by a different draft.
   - Escape tests exercise policy directly, not keyboard delivery across the shadow boundary.
   - Couch tests could not compile: generated `cmd/internal/runtimebundle/assets/runtime/files` is absent. Artifact checks also failed because that generated tree is missing.
   - Real Chrome acceptance could not run: Chrome exited during startup with “GPU process isn't usable.”
   - No prior findings required disposition. Repository files remained unchanged.

6. **Architectural notes**

   | Marker | Assessment |
   |---|---|
   | ARCH-DRY | Pass — shared gestures, dialog, validation, and capture helper. |
   | ARCH-PURE | Flag — DOM helper classification above. |
   | ARCH-PURPOSE | Flag — documented Escape behavior is not delivered for page focus. |
   | ARCH-MOCK | Pass structurally — stateful seams and conformance test exist; live verification was blocked here. |
   | ARCH-CONSTRAINTS | Pass — bounded request queues, retained outcomes, and stroke projection. |
   | ARCH-SECURE | Pass — trusted context admission, bounded parsing, and reserved operator attribution. |
   | ARCH-ORDER | Flag — transport reconciliation conflates distinct user operations. |
   | ARCH-FUNERAL | Pass — bounded transient state and existing recording retention; workers follow Chrome lifetime. |

7. **Plan revision recommendations**

   Add `## Revisions` entries that:
   
   - Reclassify the recorded DOM path helper as INTEGRATION.
   - Specify that reconciliation settles only its originating draft/action, including cancellation and replacement-draft sequences.
   - Require Escape verification with page focus and annotation selected through Couch.

```findings
findings:
  - id: new
    severity: Critical
    family: operation-result-identity
    title: |
      Reconciliation reports another draft's save as successful
    detail: |
      cmd/internal/webcast/operator.js:31–42: after draft A times out, saving a different draft B queries A and returns 204 when A was accepted, closing B without submitting it. Reproduced against production transport. Bind completion to its originating draft/action and test cancellation followed by a new draft (ARCH-ORDER).
  - id: new
    severity: Critical
    family: keyboard-event-boundary
    title: |
      Escape cannot turn annotation off while focus remains on the page
    detail: |
      cmd/internal/webcast/web/annotate.js:591–593 registers Escape only on the operator ShadowRoot. Page-focused events do not reach it, including after Couch's focus-preserving mode command. Handle Escape at the document/window boundary and test this input path (ARCH-PURPOSE).
  - id: new
    severity: Critical
    family: core-concept-purity-classification
    title: |
      The plan labels a live-DOM helper PURE
    detail: |
      workshop/plans/000494-operator-overlay-plan.md:23 classifies the recorded node path helper as PURE, but recorder/observer.js:53–69 reads live DOM structure and global document identity. Promote it to INTEGRATION and record the correction under Revisions, or extract a deterministic core over supplied data (ARCH-PURE).
```

---

## Re-review — 2026-10-12T23:36:38-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 494 — broadcast-web: the operator annotates in the cast Chrome itself (pointer strokes and notes, injected overlay) |
| repo | pair |
| issue file | workshop/issues/000494-broadcast-web-the-operator-annotates-in-the-cast-chrome-itself-pointer-strokes-and-notes-injected-overlay.md |
| boundary | whole-issue close |
| milestone | — |
| window | addd45e19d528471163e0d6803e8cb65718b89a7..d0f3e0ca02a44ca4d4a0d478ca98215884cefa5b |
| command | sdlc close --issue 494 |
| reviewer | codex |
| isolation | fenced |
| timestamp | 2026-10-12T23:36:38-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All three prior findings are addressed, with mutation-tested evidence for BR-1 and BR-2. Shared UI reuse, recording admission, and capture suppression are well covered. Two remaining problems need correction: unresolved operations survive a document-scope change and block subsequent saves, and routine record updates replace color buttons during interaction.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      operator.js:34–42 binds reconciliation to the originating action/body. Reverting this fix in a disposable checkout failed all three transport regressions; restored HEAD passed.
  - id: BR-2
    disposition: addressed
    note: |
      annotate.js:594 registers operator Escape on window capture. Reverting this fix failed the page-focused Escape regression; restored HEAD passed.
  - id: BR-3
    disposition: addressed
    note: |
      Plan line 48 now classifies the recorded node path helper as INTEGRATION, with the correction recorded at lines 417–419. observer.js:53–69 confirms its live-DOM and document dependencies.
findings:
  - id: new
    severity: Critical
    family: operation-result-identity
    title: |
      A lost save reply blocks future saves after switching away and back
    detail: |
      operator.js:31–43 retains unsettled across projection/document changes at lines 100–104, while recorder/operator.go:156–161 replaces OperatorSession when the recording document changes. Reproduced an accepted save with its reply lost, followed by A→B→A: the recorder returns “result is no longer retained,” and every new save sends status for the old request forever. ARCH-ORDER, ARCH-PURPOSE. This is the 2nd finding in family operation-result-identity. Do NOT fix only this instance: define operation ownership across session/context/document/action, enumerate the lifecycle boundaries, and retire old-scope transport state without resubmitting old writes or accepting late replies into a new scope.
  - id: new
    severity: Important
    family: interactive-dom-identity
    title: |
      Ordinary record updates replace color buttons during interaction
    detail: |
      annotate.js:587 calls sync for every record; transition at line 392 calls renderRail, whose line 403 invokes renderSwatches. collab.js:127 replaces every color button. Confirmed on both viewer and operator mounts that an unchanged-model scroll record detaches the pressed button; updates between pointer-down and pointer-up can therefore cancel color selection. Preserve button identity and update attributes in place, or avoid rebuilding on unchanged state. Add an interleaved-record click regression. ARCH-ORDER, ARCH-CONSTRAINTS.
```

1. **Strengths**
   - Viewer and operator share gesture geometry, dialogs, and annotation policy.
   - Recorder tests exercise ordering-time refusal, stale contexts, stable anchors, and uncertain write outcomes.
   - Capture suppression requires confirmation before and after capture and releases the exact document’s token.
   - README and atlas cover the new controls, transport, capture behavior, and recovery path.

2. **Critical findings**
   - The document-scope transport failure above remains blocking. The disposable recorder test confirmed that an old accepted operation becomes unqueryable after same-context tab return; the production browser adapter repeatedly queried that old ID instead of submitting the new draft.

3. **Important findings**
   - Preserve interactive color-button identity across unrelated record updates.

4. **Minor findings**
   - None raised.

5. **Test coverage notes**
   - Passed: all **206 browser tests**, all webcast Go packages, and targeted Board/recorder race tests.
   - Mutation checks independently confirmed BR-1 and BR-2 regressions fail without their fixes.
   - Headless conformance could not execute: Chrome crashed during startup with “GPU process isn't usable.”
   - Menu tests and artifact checks failed because generated runtime assets are absent from this checkout.
   - Disposable test directories were removed; tracked files remain unchanged.

6. **Architectural notes**
   - **ARCH-DRY — pass:** shared UI, validation, path grammar, and capture helper.
   - **ARCH-PURE — pass:** pure policies and protocol values are separated from DOM/CDP integration; BR-3 classification corrected.
   - **ARCH-PURPOSE — flag:** document changes can permanently prevent further note saves.
   - **ARCH-MOCK — pass:** stateful recorder/CDP seams exercise production paths; live conformance remains unverified here.
   - **ARCH-CONSTRAINTS — flag:** unconditional control rebuilding adds work on every record and disrupts interaction.
   - **ARCH-SECURE — pass:** bounded parsing, trusted context admission, and reserved operator attribution.
   - **ARCH-ORDER — flag:** transport ownership does not follow recorder document scope; record arrivals can invalidate active UI targets.
   - **ARCH-FUNERAL — pass:** new state is bounded or owned by context/cast lifetime; no new persistent artifact family.

7. **Plan revision recommendations**
   - Add a `## Revisions` entry defining operation lifetime across tab switches, document replacement, late replies, and unresolved outcomes. Require sequence tests spanning browser transport and recorder scope changes.
   - Record the invariant that unrelated projection updates preserve interactive DOM nodes, with viewer/operator interleaving coverage.
