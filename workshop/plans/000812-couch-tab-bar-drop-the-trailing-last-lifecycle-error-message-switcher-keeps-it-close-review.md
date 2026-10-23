# Boundary Review — pair#812 (whole-issue close)

| field | value |
|-------|-------|
| issue | 812 — Couch tab bar: drop the trailing last lifecycle/error message (switcher keeps it) |
| repo | pair |
| issue file | workshop/issues/000812-couch-tab-bar-drop-the-trailing-last-lifecycle-error-message-switcher-keeps-it.md |
| boundary | whole-issue close |
| milestone | — |
| window | c0dbb32decc052500172ded01ff36e3fa8e49cc9..3c335cfdb96d94ceea14b5a8aff42c2ae5801925 |
| command | sdlc close --issue 812 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T05:37:50-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The shared renderer removal, switcher projection, and attention navigation are well implemented. Targeted tests, race checks, and reader inventory pass. Two routing gaps block shipping: local lifecycle warnings still bypass attention, and successful Chrome fidelity reports now create attention obligations.

1. **Strengths**

   - Removing `StatusModel.Notice` eliminates the trailing-message path for both layouts; generated tests compare complete geometry and hit spans.
   - `menuSnapshotLocked` supplies both snapshot and publication paths, preserving operation and clipboard precedence.
   - Warning projection bounds and sanitizes attention text while preserving exact source identity and full feed text.
   - README and atlas describe the changed surfaces and navigation.

2. **Critical findings**

   - **Local lifecycle warnings remain feed-only.** [console_lifecycle_failure.go:48](/Volumes/TestScratch/review-checkouts/253c3ca3b6440386/cmd/internal/couchtty/console_lifecycle_failure.go:48) calls `Notify` for terminal diagnostics, including “lifecycle completed with unproven cleanup.” The new dispatcher warning callback does not cover queue-owned attempts: `DispatchOperation` skips that callback when the context already carries an attempt. The plan explicitly names `finishLifecycleAttempt` as an attention producer. Route this owner through `Warn`, preserving its source, and add a local queue completion regression. **ARCH-PURPOSE**.

   - **Successful recorded casts create false obligations.** [console_webcast.go:58](/Volumes/TestScratch/review-checkouts/253c3ca3b6440386/cmd/internal/couchtty/console_webcast.go:58) promotes every `Logf` message to a warning. However, [cast.go:524](/Volumes/TestScratch/review-checkouts/253c3ca3b6440386/cmd/internal/webcast/cast.go:524) logs successful fidelity summaries through the same callback; production wires this check for recorded casts. A normal successful recording therefore produces “Attention required,” contrary to the explicit noise guard. Separate actionable diagnostics from routine reports at their producers, without matching English text. Add success and failure attention regressions. **ARCH-PURPOSE**.

3. **Important findings**

   None beyond the regression coverage required with those fixes.

4. **Minor findings**

   None.

5. **Test coverage notes**

   Passed independently:

   - Focused warning, projection, navigation, and composition tests.
   - Plan-selected couchtty and couchcmd test groups.
   - Focused couchtty race checks.
   - Operation-attempt and end-of-recording tests.
   - `make test-affected-inventory` and `git diff --check`.

   Existing tests establish that successful fidelity summaries reach `Logf`, but the normal-stop attention tests do not exercise a recorded cast with frames. Cleanup attention coverage exercises the remote queue, leaving the local completion owner uncovered. No complete-suite or live-console acceptance is claimed here.

6. **Architectural notes**

   - **ARCH-DRY — pass:** one feed history owner and shared menu projection.
   - **ARCH-PURE — pass:** renderers and warning projection remain deterministic; integration stays in console methods.
   - **ARCH-PURPOSE — flag:** actionable producer coverage is incomplete, while routine success is promoted.
   - **ARCH-MOCK — pass:** changed integration paths reuse injected stateful fixtures; no new external dependency.
   - **ARCH-CONSTRAINTS — pass:** bounded attention text and existing list capacity; no new per-frame IO.
   - **ARCH-SECURE — pass:** attention text is sanitized; cast-code tests preserve secret-safe wording.
   - **ARCH-ORDER — pass:** publication follows the established lock order; existing transition owners retain cast-generation admission.
   - **ARCH-FUNERAL — pass:** notifications reuse dismissal, capacity, expiry, and console lifetime.
   - **ARCH-EVOLVE — pass:** no persisted-format change.

7. **Plan revision recommendations**

   Add a `## Revisions` entry documenting the completed producer sweep: distinguish local queue, remote queue, and direct-dispatch lifecycle owners, and classify Chrome reporting sources by actionable versus routine outcomes. Retain the existing requirements; qualify the checked producer-routing claim until both regressions pass.

```findings
findings:
  - id: new
    severity: Critical
    kind: defect
    family: actionable-producer-coverage
    title: |
      Local queue lifecycle diagnostics still bypass attention
    detail: |
      cmd/internal/couchtty/console_lifecycle_failure.go:48 publishes DiagnosticWarning through feed-only Notify, including unproven cleanup and diagnostic persistence failures. DispatchOperation skips its new warning callback for the attempt already carried by local queue requests, so it cannot compensate. ARCH-PURPOSE: route this terminal owner through Warn with source identity and add a local queue regression proving attention publication, as required by the plan's explicit finishLifecycleAttempt audit anchor.
  - id: new
    severity: Critical
    kind: defect
    family: attention-requires-actionable-source
    title: |
      Successful Chrome fidelity reports become attention obligations
    detail: |
      cmd/internal/couchtty/console_webcast.go:58-60 routes every CastConfig.Logf message through setWarning. cmd/internal/webcast/cast.go:519-524 emits successful fidelity summaries through that callback, and production webcastSettings enables the check for recorded casts. ARCH-PURPOSE: classify outcomes at their producers so routine success remains feed-only; add recorded-cast success and actionable-failure attention regressions without English-text classification.
```

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d681bbc3-7ae8-4bed-b01d-ce214d481b3c","repository":"github.com/xianxu/pair","issue_id":"000812","recipe":"milestone-review","round":1,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/253c3ca3b6440386","reviewed_head":"3c335cfdb96d94ceea14b5a8aff42c2ae5801925","slot":"pair:1","requested_at":"2026-10-23T12:34:36.109541Z"},"phase":"finished","pid":83066,"started_at":"2026-10-23T12:34:36.201473Z","finished_at":"2026-10-23T12:37:49.310047Z","session_id":"01a168c1-893b-7130-80f3-369223c706b3","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.009584833s
head: "3c335cfdb96d94ceea14b5a8aff42c2ae5801925"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
