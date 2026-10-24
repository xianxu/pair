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

---

## Re-review — 2026-10-23T05:55:18-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 812 — Couch tab bar: drop the trailing last lifecycle/error message (switcher keeps it) |
| repo | pair |
| issue file | workshop/issues/000812-couch-tab-bar-drop-the-trailing-last-lifecycle-error-message-switcher-keeps-it.md |
| boundary | whole-issue close |
| milestone | — |
| window | c0dbb32decc052500172ded01ff36e3fa8e49cc9..2e1dccf9dd632fefffcbdf854c45b636b4f7525a |
| command | sdlc close --issue 812 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T05:55:18-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The rendering change and both prior fixes are verified, including regression failures when those fixes are reversed. Shipping remains blocked by another actionable background warning that bypasses attention and by failing regression tests across the affected packages.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      finishLifecycleAttempt now calls Warn with the origin thread. Reversing that change made TestLifecycleLocalQueuedCompletionWarningOwnsExactOrigin fail for cleanup and diagnostic persistence; the pinned-head regression passes under race.
  - id: BR-2
    disposition: addressed
    note: |
      Typed Warnf separates actionable failures from informational Logf. Restoring generic-log warning routing made the actual recorded-cast healthy-fidelity regression fail with attention=1; healthy/error regressions pass under race on the pinned head.
findings:
  - id: new
    severity: Critical
    kind: defect
    family: actionable-producer-coverage
    title: |
      The background low-priority-agent warning still bypasses attention
    detail: |
      cmd/internal/couchcmd/message_service.go:582 wires the renicer to feed-only Notify. renicer.go:180 emits a once-per-slot instruction to relaunch an agent whose priority cannot be restored automatically; removing tab notices leaves this actionable asynchronous warning without attention. ARCH-PURPOSE: This is the 2nd finding in family `actionable-producer-coverage`. Earlier rounds fixed instances. State the rule covering all asynchronous sources requiring operator intervention, enumerate every remaining Notify/setNotice callback against it, and sweep that enumeration in this round. Route this source through Warn with its binding identity and add a production-wiring regression plus ordinary-operation negative controls.
  - id: new
    severity: Important
    kind: defect
    family: regression-oracles-follow-contract
    title: |
      Affected regression tests still enforce removed behavior or use an invalid success fixture
    detail: |
      cmd/internal/couchtty/terminal_migration_test.go:23,89,144,170, console_test.go:267, and cmd/internal/couchcmd/attention_ephemeral_test.go:90 still require notice text in terminal chrome and fail on the pinned head. Preserve their parser-isolation, publication-order, resize, focus, and persistence invariants using supported visible surfaces. Separately, notice_attention_test.go:124 calls a fixture with a background borrowed attachment “ordinary successful startup”; four of five diagnostic runs failed with the legitimate BORROWED ATTACH warning. Supply confirmed ownership and wait for background completions before asserting absence. ARCH-ORDER: retain controlled ordering coverage while migrating these oracles.
```

1. **Strengths**

   - Removing `StatusModel.Notice` and its shared rendering path covers both tab layouts.
   - `console_menu.go:54` derives switcher text from the existing feed, preserving operation and clipboard precedence.
   - Lifecycle diagnostics retain terminal ownership and exact origin; Chrome severity is classified without English-text matching.
   - README and atlas explain the changed presentation and navigation.

2. **Critical findings**

   - `message_service.go:582`: the renicer’s relaunch warning needs attention publication. Apply the source-class sweep described above.

3. **Important findings**

   - Migrate the stale rendering assertions while preserving their original invariants.
   - Repair the successful-startup fixture and synchronize its negative assertion with background completion.

4. **Minor findings**

   - None.

5. **Test coverage notes**

   - Focused rendering, switcher, lifecycle, fidelity, ordering, and lifetime regressions passed under race.
   - Targeted recorder and drive classification tests passed.
   - The broader five-package selection failed in `couchtty` and `couchcmd`; `couchcore`, `webcast`, and `recorder` passed.
   - Reversal tests reproduced both prior defects. The disposable copy was removed; the reviewed checkout remains unchanged.
   - No complete-suite or live-acceptance pass is claimed.

6. **Architectural notes**

   - **ARCH-DRY — pass:** shared feed ownership and warning publication.
   - **ARCH-PURE — pass:** pure rendering/projection; thin publication shell.
   - **ARCH-PURPOSE — flag:** actionable producer coverage remains incomplete.
   - **ARCH-MOCK — pass:** existing stateful external seams are reused.
   - **ARCH-CONSTRAINTS — pass:** bounded text and attention capacity; no added synchronous external work.
   - **ARCH-SECURE — pass:** attention text is sanitized and bounded; source identity is explicit.
   - **ARCH-ORDER — flag:** affected publication/resize sequence tests need valid replacement oracles.
   - **ARCH-FUNERAL — pass:** existing capacity and expiry govern new warnings.
   - **ARCH-EVOLVE — pass:** no persisted-format change; `CleanupWarning` is excluded from serialization.

7. **Plan revision recommendations**

   Add `## Revisions` entries recording the complete remaining producer enumeration, including the renicer, and the regression-oracle migration across affected packages. Keep qualification unchecked until those tests pass; live acceptance remains pending under the documented close-before-stage sequencing.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"5c1563ac-423e-457c-afa5-4d89e9d5ab4e","repository":"github.com/xianxu/pair","issue_id":"000812","recipe":"milestone-review","round":2,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/06e98f190d7d2bb0","reviewed_head":"2e1dccf9dd632fefffcbdf854c45b636b4f7525a","slot":"pair:1","requested_at":"2026-10-23T12:48:58.477558Z"},"phase":"finished","pid":73169,"started_at":"2026-10-23T12:48:58.57536Z","finished_at":"2026-10-23T12:55:17.272569Z","session_id":"01a168ce-b1e1-7723-8dee-3f2535a0c8f8","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 1.890501584s
head: "2e1dccf9dd632fefffcbdf854c45b636b4f7525a"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-23T06:30:41-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 812 — Couch tab bar: drop the trailing last lifecycle/error message (switcher keeps it) |
| repo | pair |
| issue file | workshop/issues/000812-couch-tab-bar-drop-the-trailing-last-lifecycle-error-message-switcher-keeps-it.md |
| boundary | whole-issue close |
| milestone | — |
| window | c0dbb32decc052500172ded01ff36e3fa8e49cc9..cafbf526b5ca0ecfbd4b75518de7ac3758585ebf |
| command | sdlc close --issue 812 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T06:30:41-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The presentation change and both prior corrections are supported by code and regression evidence. One new test-ordering defect blocks qualification: an asynchronous warning test can open an empty attention list before publication, then wait indefinitely for a closed panel to render. The complete `couchcmd` run stalled there; focused passes do not resolve that ordering gap.

```findings
dispose:
  - id: BR-3
    disposition: addressed
    note: |
      Production uses newConsoleRenicer with exact binding identity. Reverting Warn to Notify in a disposable checkout fails TestRenicerConsoleWarningIsOncePerExactBinding; normal-priority controls pass. The plan now enumerates remaining producer classes.
  - id: BR-4
    disposition: addressed
    note: |
      Named terminal oracles now assert supported badges, retained feed and switcher surfaces while preserving ordering/focus invariants. Startup supplies retained ownership evidence and joins restart-drained. It passes 20 repetitions; restoring the previous fixture fails all five reversal runs.
findings:
  - id: new
    severity: Important
    kind: defect
    family: regression-oracles-follow-contract
    title: |
      Async warning test opens attention before publication is guaranteed
    detail: |
      cmd/internal/couchcmd/lifecycle_diagnostic_attention_test.go:19 calls openWarningAttention immediately after queuing the diagnostic. The helper at notice_attention_test.go:45 toggles first, but console_attention.go:144 leaves an empty list closed; later warning publication does not open it. The complete package run stalled here for minutes, confirmed by its captured stack, and was interrupted at 470.7 seconds. This is the 2nd finding in family regression-oracles-follow-contract. ARCH-ORDER: fix the class by requiring publication acknowledgment before opening attention across every asynchronous helper caller, and force both publication/input orderings in regression coverage.
```

1. **Strengths**

   - Removing `StatusModel.Notice` eliminates the trailing message path from both layouts.
   - `console_menu.go:54` derives switcher text from retained Feed history while preserving operation/copy precedence.
   - `renicer.go:199` preserves the producer’s exact binding through production warning wiring.
   - README and atlas document the changed presentation, warning classification and navigation.

2. **Critical findings**

   None.

3. **Important findings**

   The asynchronous attention-test race above. Add a publication barrier to the shared testing contract and audit its callers; avoid fixing only this test.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Complete `couchtty`: PASS, 54.4 seconds.
   - Focused tests across five affected packages: PASS.
   - Focused `couchtty`/`couchcmd` race checks: PASS.
   - Startup success fixture: PASS ×20; both reversal checks failed as expected.
   - Inventory and diff whitespace checks: PASS.
   - Complete `couchcmd`: interrupted after the confirmed stall; no package pass claimed.
   - Disposable reversal checkout removed; reviewed checkout unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Feed remains the history owner; shared snapshot projection serves both menu paths.
   - **ARCH-PURE — pass:** rendering and warning projection remain pure; publication stays in console glue.
   - **ARCH-PURPOSE — pass:** removal, switcher retention and explicit producer classification fulfill the requested behavior.
   - **ARCH-MOCK — pass:** changed behavior uses existing injected process/browser/terminal seams.
   - **ARCH-CONSTRAINTS — pass:** attention projection retains existing capacity and expiry bounds.
   - **ARCH-SECURE — pass:** projected text is bounded and sanitized; lockout tests check secret safety.
   - **ARCH-ORDER — flag:** asynchronous test input lacks a publication barrier, as detailed above.
   - **ARCH-FUNERAL — pass:** no new durable artifact family; attention retains existing dismissal/expiry.
   - **ARCH-EVOLVE — pass:** no persisted-format change; cleanup warning is explicitly nonserialized.

7. **Plan revision recommendations**

   Add a `## Revisions` entry requiring publication acknowledgment before attention-panel assertions, an audit of all asynchronous helper callers, and controlled coverage of both event orderings. Preserve the separately pending recorded-check and live-acceptance requirements.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"a09cbb80-6972-4c9c-b4ed-07931fc6c3c5","repository":"github.com/xianxu/pair","issue_id":"000812","recipe":"milestone-review","round":3,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/53e4eba121cf521e","reviewed_head":"cafbf526b5ca0ecfbd4b75518de7ac3758585ebf","slot":"pair:1","requested_at":"2026-10-23T13:20:21.224581Z"},"phase":"finished","pid":68622,"started_at":"2026-10-23T13:20:21.333787Z","finished_at":"2026-10-23T13:30:39.668318Z","session_id":"01a168eb-6c8a-7a01-b817-90049483eede","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.219503916s
head: "cafbf526b5ca0ecfbd4b75518de7ac3758585ebf"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-23T06:41:59-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 812 — Couch tab bar: drop the trailing last lifecycle/error message (switcher keeps it) |
| repo | pair |
| issue file | workshop/issues/000812-couch-tab-bar-drop-the-trailing-last-lifecycle-error-message-switcher-keeps-it.md |
| boundary | whole-issue close |
| milestone | — |
| window | c0dbb32decc052500172ded01ff36e3fa8e49cc9..58b2e43e6af14e82aa9413c7a960c46d6b44a0ed |
| command | sdlc close --issue 812 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T06:41:59-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The implementation delivers the tab-bar removal, retained switcher message, and explicit warning routing. BR-5’s correction is verified, including a regression that fails when its publication wait is removed. One additional asynchronous test still asserts attention before publication is guaranteed; this needs correction before the boundary.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Local queued completion uses Warn with the retained origin; TestLifecycleLocalQueuedCompletionWarningOwnsExactOrigin passes in the complete couchtty run.
  - id: BR-2
    disposition: addressed
    note: |
      Informational Logf and actionable Warnf remain separate; recorded fidelity success/failure and drive classification regressions pass.
  - id: BR-3
    disposition: addressed
    note: |
      newConsoleRenicer preserves the exact binding and publishes through Warn; once-per-binding and normal-priority controls pass.
  - id: BR-4
    disposition: addressed
    note: |
      Migrated rendering oracles and corrected startup fixtures pass in complete couchtty and couchcmd package runs.
  - id: BR-5
    disposition: addressed
    note: |
      The shared helper waits for current VT-rendered publication, and the lifecycle test joins Warn completion. Controlled ordering tests pass; removing the helper wait makes helper-before-publication fail immediately.
findings:
  - id: new
    severity: Important
    kind: defect
    family: regression-oracles-follow-contract
    title: |
      Unexpected-exit regression asserts attention before warning publication
    detail: |
      cmd/internal/couchtty/console_warning_test.go:139 waits only for pane removal before immediately asserting attention. onExit releases c.mu at console.go:1258 and calls Warn at :1266, so the assertion can observe zero warnings during a legal interleaving. A temporary Go overlay that channel-gates this existing gap deterministically fails with “unexpected exit produced 0 warnings.” ARCH-ORDER, ARCH-PURPOSE: this is the 3rd finding in family regression-oracles-follow-contract. Do NOT fix only this instance: state and enumerate the rule that asynchronous warning assertions must acknowledge the actual published postcondition, sweep the affected tests, and retain controlled coverage of both orderings.
```

1. **Strengths**

   - Removing `StatusModel.Notice` eliminates the trailing-message path in the shared renderer.
   - `menuSnapshotLocked` derives the latest message from existing feed history; operation notices and copy feedback retain precedence.
   - Warning projection sanitizes and bounds text while preserving explicit source identity.
   - README and atlas document the changed presentation and attention navigation.

2. **Critical findings**

   None.

3. **Important findings**

   The ordering defect above at `cmd/internal/couchtty/console_warning_test.go:139`. Wait for actual warning publication before asserting count and source. Keep the controlled removal-before-publication case as regression evidence.

4. **Minor findings**

   None.

5. **Test coverage notes**

   - Complete package tests passed: couchtty **42.096s**, couchcmd **72.989s**, recorder **4.692s**, webcast **12.164s**.
   - Focused publication, navigation, rendering and classification race tests passed.
   - `make test-affected-inventory` and pinned-range `git diff --check` passed.
   - BR-5 reversal failed as expected. The newly identified ordering gap also failed deterministically under a temporary channel gate.
   - Repository files remain unchanged. Recorded-suite and coordinated live acceptance remain separate pending qualification steps.

6. **Architectural notes**

   - **ARCH-DRY — pass:** feed and attention retain their existing ownership.
   - **ARCH-PURE — pass:** rendering and warning projection remain pure; console glue applies effects.
   - **ARCH-PURPOSE — flag:** the regression-oracle correction still misses an enumerable sibling.
   - **ARCH-MOCK — pass:** affected integrations exercise existing stateful fakes.
   - **ARCH-CONSTRAINTS — pass:** bounded text, attention capacity and existing rendering limits remain enforced.
   - **ARCH-SECURE — pass:** attention text is sanitized; cast-warning coverage checks secret-safe wording.
   - **ARCH-ORDER — flag:** the exit regression mistakes an intermediate state for completed publication.
   - **ARCH-FUNERAL — pass:** new values remain console-scoped; notification expiry remains existing policy.
   - **ARCH-EVOLVE — pass:** no persisted format changes; `CleanupWarning` is explicitly excluded from serialization.

7. **Plan revision recommendations**

   Add a `## Revisions` entry extending the publication-acknowledgment rule from `openWarningAttention` callers to **all asynchronous warning assertions**, with the audited test enumeration and controlled-order evidence.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"9a8d6407-9d16-4b5c-9abe-0125e35ab12e","repository":"github.com/xianxu/pair","issue_id":"000812","recipe":"milestone-review","round":4,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/25f24d1b136fb5ba","reviewed_head":"58b2e43e6af14e82aa9413c7a960c46d6b44a0ed","slot":"pair:1","requested_at":"2026-10-23T13:38:34.963442Z"},"phase":"finished","pid":48139,"started_at":"2026-10-23T13:38:35.125699Z","finished_at":"2026-10-23T13:41:57.157573Z","session_id":"01a168fc-1d8a-7e02-a46e-719c40033ab0","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.725257208s
head: "58b2e43e6af14e82aa9413c7a960c46d6b44a0ed"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-23T06:54:07-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 812 — Couch tab bar: drop the trailing last lifecycle/error message (switcher keeps it) |
| repo | pair |
| issue file | workshop/issues/000812-couch-tab-bar-drop-the-trailing-last-lifecycle-error-message-switcher-keeps-it.md |
| boundary | whole-issue close |
| milestone | — |
| window | c0dbb32decc052500172ded01ff36e3fa8e49cc9..d29420112ffe74bae7e2b7bb443c93a7a5dcbf8f |
| command | sdlc close --issue 812 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T06:54:07-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change satisfies the documented presentation and warning-routing contracts. BR-6 is addressed: assertions acknowledge actual publication, and the affected tests have explicit completion joins. No new blocking findings. Recorded qualification and coordinated live acceptance remain prerequisites before landing, as the revised plan states.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Queue-owned diagnostics use Warn with the retained origin; real local-queue regressions cover cleanup, persistence failure, success and duplicate completion.
  - id: BR-2
    disposition: addressed
    note: |
      Typed Warnf remains separate from informational Logf; recorded-cast regressions distinguish healthy fidelity from failed checks.
  - id: BR-3
    disposition: addressed
    note: |
      newConsoleRenicer retains exact binding identity; regressions verify one relaunch warning, exact selection and quiet normal priority.
  - id: BR-4
    disposition: addressed
    note: |
      Affected rendering and lifecycle assertions use retained feed, switcher or attention contracts; targeted checks pass.
  - id: BR-5
    disposition: addressed
    note: |
      openWarningAttention acknowledges current terminal cells before input; controlled tests cover early invocation, prior publication and historical badges.
  - id: BR-6
    disposition: addressed
    note: |
      console_warning_test.go:136 waits for exact warning publication and joins SetForget completion. With the same temporary 200 ms publication-gap overlay, the previous oracle fails with zero warnings and the repaired oracle passes ten runs. The plan enumerates affected assertions and completion joins.
```

1. **Strengths**

   - Removing `StatusModel.Notice` eliminates the trailing-message rendering path in both layouts.
   - `console_menu.go:54` derives switcher text from the existing feed, preserving one history owner.
   - `console.go:2356` publishes feed and attention under consistent locking, with repaint after unlocking.
   - README and atlas describe the changed presentation, warning classification and navigation.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed targeted warning suites across couchtty, couchcmd, webcast and recorder; repeated race checks; renderer, switcher-copy and switch-cleanup regressions; reader inventory; and diff checks. Attention fuzzing passed 8,641 executions. The BR-6 reversal comparison failed before correction and passed afterward. Repository files remain unchanged.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Existing feed and attention owners are reused.
   - **ARCH-PURE — pass:** Projection/rendering logic remains separate from console IO.
   - **ARCH-PURPOSE — pass:** Both layout removal and actionable-source visibility are covered.
   - **ARCH-MOCK — pass:** Integration tests use existing stateful injected fixtures.
   - **ARCH-CONSTRAINTS — pass:** Text, history and attention remain bounded; no new per-frame IO.
   - **ARCH-SECURE — pass:** Attention text is sanitized; exact source identity and secret-safe wording are tested.
   - **ARCH-ORDER — pass:** Publication is atomic, and affected assertions acknowledge completed effects.
   - **ARCH-FUNERAL — pass:** Existing caps, expiry and console lifetime govern added state.
   - **ARCH-EVOLVE — pass:** No persisted format changes; cleanup-warning metadata is explicitly nonserialized.

7. **Plan revision recommendations:** None. The existing revisions document the source rule, assertion sweep and qualification sequencing.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"bc944c98-8e6f-474b-8226-8b8cb985c903","repository":"github.com/xianxu/pair","issue_id":"000812","recipe":"milestone-review","round":5,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/b679bf607ac19b3c","reviewed_head":"d29420112ffe74bae7e2b7bb443c93a7a5dcbf8f","slot":"pair:1","requested_at":"2026-10-23T13:50:30.349768Z"},"phase":"finished","pid":7393,"started_at":"2026-10-23T13:50:30.497707Z","finished_at":"2026-10-23T13:54:06.403538Z","session_id":"01a16907-07e0-73a2-a9cf-39219e37f0ca","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.15502825s
head: "d29420112ffe74bae7e2b7bb443c93a7a5dcbf8f"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000411

---

## Re-review — 2026-10-23T09:35:21-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 812 — Couch tab bar: drop the trailing last lifecycle/error message (switcher keeps it) |
| repo | pair |
| issue file | workshop/issues/000812-couch-tab-bar-drop-the-trailing-last-lifecycle-error-message-switcher-keeps-it.md |
| boundary | whole-issue close |
| milestone | — |
| window | 6ba2287858a9eb9536160c78f24c9e72eb8e7043..5eb99587cf9010715c99837953d7b81cd4d2eae4 |
| command | sdlc close --issue 812 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T09:35:21-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The narrow interdiff fixes the artifact-classification failure by reusing the existing continuation notice category. Regression reversal confirms the test passes with the fix and fails without it. No new defects, scope changes, or documentation gaps were found. Prior dispositions remain intact; recorded qualification and live acceptance remain separate landing prerequisites.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Retained prior disposition; this interdiff does not alter queue diagnostic routing.
  - id: BR-2
    disposition: addressed
    note: |
      Retained prior disposition; typed Chrome warning classification is unchanged.
  - id: BR-3
    disposition: addressed
    note: |
      Retained prior disposition; background-agent warning routing is unchanged.
  - id: BR-4
    disposition: addressed
    note: |
      Retained prior disposition; the corrected regression oracles and fixtures are unchanged.
  - id: BR-5
    disposition: addressed
    note: |
      Retained prior disposition; warning publication synchronization is unchanged.
  - id: BR-6
    disposition: addressed
    note: |
      Retained prior disposition; unexpected-exit publication acknowledgment and completion joins are unchanged.
```

1. **Strengths**

   - `console_continuation.go:147` reuses the established category without weakening artifact inventory enforcement.
   - Scan-error episode suppression remains independent of category; attention identity derives from its own sequence.
   - Spec, Done-when criteria, and implementation plan remain unchanged.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   - Artifact-classification regression passed on head.
   - In a prepared temporary copy, reversing only the changed literal caused that regression to fail with exactly the continuation vocabulary violations. Temporary copies were removed.
   - Continuation warning episode and controlled publication/selection tests passed ten race-enabled runs.
   - `make test-affected-inventory` and range diff checks passed. Reviewed checkout remained clean.

6. **Architectural notes**

   - **ARCH-DRY — pass:** Reuses the existing continuation category and warning publisher.
   - **ARCH-PURE — pass:** Adds no logic or IO coupling.
   - **ARCH-PURPOSE — pass:** Corrects classification while preserving actionable warning publication and episode suppression.
   - **ARCH-EVOLVE — pass:** No persisted format changes.
   - No new architectural or user-facing surface requires atlas or README changes.

7. **Plan revision recommendations:** None.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c7112497-37f8-42b8-a2a2-02c2cbc210dc","repository":"github.com/xianxu/pair","issue_id":"000812","recipe":"fix-review","round":6,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/e144432255e577cd","reviewed_head":"5eb99587cf9010715c99837953d7b81cd4d2eae4","slot":"pair:1","requested_at":"2026-10-23T16:33:10.12555Z"},"phase":"finished","pid":64909,"started_at":"2026-10-23T16:33:10.382938Z","finished_at":"2026-10-23T16:35:15.140167Z","session_id":"01a1699b-f514-7c62-98ed-f187b2ba0582","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.530801666s
head: "5eb99587cf9010715c99837953d7b81cd4d2eae4"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-23T10:06:42-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 812 — Couch tab bar: drop the trailing last lifecycle/error message (switcher keeps it) |
| repo | pair |
| issue file | workshop/issues/000812-couch-tab-bar-drop-the-trailing-last-lifecycle-error-message-switcher-keeps-it.md |
| boundary | whole-issue close |
| milestone | — |
| window | c0dbb32decc052500172ded01ff36e3fa8e49cc9..6582490dac8db921ce44f864b9875cd32eb0195c |
| command | sdlc close --issue 812 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T10:06:42-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned change satisfies the revised Spec and Plan: both tab-bar layouts omit general messages, the switcher retains display/copy behavior, and actionable warnings use existing attention ownership. The restart-noise correction preserves configuration and cleanup alerts while keeping routine diagnostics feed-only. No new findings. This review does not establish the separately tracked recorded-suite or live-acceptance prerequisites for landing.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Local queue diagnostics preserve exact origin and publish once. Reversing Warn to Notify makes both cleanup and diagnostic-persistence regressions fail.
  - id: BR-2
    disposition: addressed
    note: |
      Typed warning callbacks separate actionable failures from informational reports. Reversing the informational callback makes the real healthy-fidelity regression fail.
  - id: BR-3
    disposition: addressed
    note: |
      Renicer warnings retain exact binding and once ownership. Reversing Warn to Notify makes the integration regression fail awaiting publication.
  - id: BR-4
    disposition: addressed
    note: |
      Corrected tab-row oracles and startup fixtures match the contract. Targeted code-menu, guessing, exact-handle resume, and hub-startup regressions pass.
  - id: BR-5
    disposition: addressed
    note: |
      The helper acknowledges current terminal publication before input. Removing that wait fails both helper-first and historical-badge regressions.
  - id: BR-6
    disposition: addressed
    note: |
      Exit assertions observe exact warning publication; absence controls join completion. Controlled-order and affected regression tests pass repeated race checks.
```

1. **Strengths**

   - Removing `StatusModel.Notice` makes trailing general messages unavailable to both rendering layouts.
   - [menuSnapshotLocked](/Volumes/TestScratch/review-checkouts/07c927a292e92b08/cmd/internal/couchtty/console_menu.go:54) derives switcher text from existing history, preserving operation/copy precedence without another history owner.
   - [publishWarningLocked](/Volumes/TestScratch/review-checkouts/07c927a292e92b08/cmd/internal/couchtty/console.go:2356) commits feed and attention under consistent locking, then repaints after unlocking.
   - README and atlas document the changed presentation, navigation, warning classification, and routine-restart exceptions.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**

   Passed targeted tests across couchtty, couchcmd, couchcore, webcast, and recorder; additional renderer/copy/startup regressions; three repeated race runs of ordering and restart-noise tests; affected-reader inventory; and range whitespace checks.

   Seven temporary-overlay reversal checks failed as expected, including all three latest noise corrections. Attention fuzzing passed 1,764 executions. Repository files remain unchanged. No full-suite or live-acceptance result is claimed.

6. **Architectural notes**

   - **ARCH-DRY — Pass:** Existing Feed and AttentionList remain authoritative.
   - **ARCH-PURE — Pass:** Projection/rendering helpers stay pure; console integration owns publication and repaint.
   - **ARCH-PURPOSE — Pass:** Producer enumeration covers actionable siblings and explicit routine-feedback exclusions.
   - **ARCH-MOCK — Pass:** Changed integrations exercise existing stateful fakes; no new external dependency.
   - **ARCH-CONSTRAINTS — Pass:** Text bounds, attention capacity/expiry, and bounded projections remain enforced.
   - **ARCH-SECURE — Pass:** Attention text is sanitized and bounded; source identity is explicit; lockout tests check secret-safe wording.
   - **ARCH-ORDER — Pass:** Publication is atomic under the existing owner; controlled ordering tests and completion joins cover changed behavior.
   - **ARCH-FUNERAL — Pass:** Existing notification eviction/expiry owns warning lifetime; no new durable artifacts.
   - **ARCH-EVOLVE — Pass:** No persisted-format change; `CleanupWarning` explicitly excludes serialization.

7. **Plan revision recommendations:** None. Existing revisions cover the final classification changes and qualification sequencing.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"c8381809-2bdf-4b70-9af5-c1fb1f9bc525","repository":"github.com/xianxu/pair","issue_id":"000812","recipe":"milestone-review","round":7,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/07c927a292e92b08","reviewed_head":"6582490dac8db921ce44f864b9875cd32eb0195c","slot":"pair:1","requested_at":"2026-10-23T17:02:51.727052Z"},"phase":"finished","pid":88108,"started_at":"2026-10-23T17:02:51.906446Z","finished_at":"2026-10-23T17:06:41.12786Z","session_id":"01a169b7-23b6-7fe1-808b-4f29b250d6e5","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 2.605610958s
head: "6582490dac8db921ce44f864b9875cd32eb0195c"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412

---

## Re-review — 2026-10-23T17:01:34-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 812 — Couch tab bar: drop the trailing last lifecycle/error message (switcher keeps it) |
| repo | pair |
| issue file | workshop/issues/000812-couch-tab-bar-drop-the-trailing-last-lifecycle-error-message-switcher-keeps-it.md |
| boundary | whole-issue close |
| milestone | — |
| window | 4ce9260a4ed5fef2b3b4b12da9bd8139e51eae07..432189a799971b1663e8a7ab8e16f989ac0eebcc |
| command | sdlc close --issue 812 |
| reviewer | codex (openai; author openai) |
| isolation | fenced |
| timestamp | 2026-10-23T17:01:34-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned interdiff preserves both sides’ intent: main’s bell navigation, capability limits, and folded-slot rendering coexist with removed tab-bar notices and retained switcher display/copy. No new defects or scope expansion found. BR-1–BR-6 remain disposed; this window provides no evidence requiring reopening them.

```findings
{}
```

1. **Strengths**
   - Capability tests retain independent expected text, Unicode width, and control-sanitization checks.
   - Restart acceptance checks inspect current terminal cells and verify the complete copied diagnostic.
   - README correctly combines switcher feed behavior with capability diagnostic copy precedence.

2. **Critical findings:** None.

3. **Important findings:** None.

4. **Minor findings:** None.

5. **Test coverage notes**
   - Focused couchtty and couchcmd tests passed.
   - Repeated race checks passed three times, including changed tests, Ctrl-Return navigation, and folded-state rendering.
   - Affected-reader inventory and range whitespace checks passed; the refreshed certificate matches the file digest.
   - No full-suite or live-acceptance result is claimed by this review.

6. **Architectural notes**
   - **ARCH-DRY — Pass:** Reuses existing feed and clipboard assertion helpers.
   - **ARCH-PURE — Pass:** Rendering remains pure; console fixtures exercise integration behavior.
   - **ARCH-PURPOSE — Pass:** Conflict resolutions preserve notice removal and retained diagnostic access.
   - **ARCH-EVOLVE — Pass:** No persisted-format change; certificate refresh retains the existing schema.

7. **Plan revision recommendations:** None. The added conflict-repair entry records the fix without changing scope.

<!-- sdlc-review-usage [{"version":1,"intent":{"disposable_root":true,"id":"d36b6f9d-0657-4992-b138-41403e927d38","repository":"github.com/xianxu/pair","issue_id":"000812","recipe":"fix-review","round":8,"attempt":1,"agent":"codex","root":"/Volumes/TestScratch/review-checkouts/22583e556e19799f","reviewed_head":"432189a799971b1663e8a7ab8e16f989ac0eebcc","slot":"pair:1","requested_at":"2026-10-23T23:59:56.329959Z"},"phase":"finished","pid":16018,"started_at":"2026-10-23T23:59:56.5838Z","finished_at":"2026-10-24T00:01:33.256969Z","session_id":"01a16b34-fd39-7f10-b493-73c1ea9d920f","outcome":"returned"}] -->

## Review preparation and protocol diagnostics

Quoted values are diagnostic data, not review verdicts.
Preparation attempt 1; duration 3.211854584s
head: "432189a799971b1663e8a7ab8e16f989ac0eebcc"
command: "make" "runtimebundle-generate"
error: ""
output: "go run ./cmd/internal/runtimebundle/generatecmd --repo . --out cmd/internal/runtimebundle/assets/runtime\n"

truncated: false
retained-bytes: 0000000412
