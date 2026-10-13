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
