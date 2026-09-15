---
gate: boundary-review
issue: 255
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-15T10:17:29-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Split qualification never compares complete observations
          detail: 'cmd/internal/terminalqualify/runner.go:43 checks only sparse literal expectations, so split-only corruption outside those keys passes. ARCH-PURPOSE: retain the whole-input observation, compare complete observations across variants, and add regressions for otherwise-unasserted cells, styles, links, and replies.'
          family: qualification-observation-equivalence
          round: 1
        - id: BR-2
          severity: Important
          title: Non-color rendering attributes are neither observed nor qualified
          detail: 'cmd/internal/terminalqualify/candidate.go:204 omits Style.Attrs, Underline, and UnderlineColor; fixtures and Coverage contain no corresponding obligations. ARCH-PURPOSE: capture these fields and add literal set/reset/preservation cases with regression evidence.'
          family: required-capability-coverage
          round: 1
        - id: BR-3
          severity: Important
          title: JSON results omit the promised expected and observed evidence
          detail: 'cmd/internal/terminalqualify/report.go:21 emits status/detail without the expected/observed results promised at plan line 135. ARCH-PURPOSE: add bounded structured evidence for executable cases and verify it through JSON tests.'
          family: qualification-evidence-contract
          round: 1
        - id: BR-4
          severity: Important
          title: README update appears missing for the terminal qualification probe
          detail: cmd/probes/terminalqualify/main.go:41 adds a runnable diagnostic with distinct exit statuses, but README.md is unchanged in the pinned range. Document invocation, JSON output, exit meanings, and the qualification report.
          family: readme-surface-documentation
          round: 1
      boundary: M1
      blocked: true
    - "n": 2
      timestamp: "2026-09-15T10:26:56-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: runner.go:53 compares complete observations in both directions. Disabling this comparison makes TestRunCaseDetectsUnassertedSplitStateChanges fail for cells, attributes, links, replies, and added keys.
          round: 2
        - id: BR-2
          disposition: addressed
          note: candidate.go:207 captures attributes, underline style, and underline color; screen_cases.go adds 12 literal set/reset/preservation cases. Removing attribute capture makes TestCandidateCapturesCompleteStyle fail.
          round: 2
        - id: BR-3
          disposition: addressed
          note: report.go:29 and runner.go:94 provide bounded expected/observed evidence. Removing expected JSON serialization makes TestRunJSONPreservesStructuredEvidence fail; the real probe emits evidence for every executable case.
          round: 2
        - id: BR-4
          disposition: addressed
          note: The pinned README.md:773 addition documents invocation, JSON evidence, exit meanings, and the qualification report. These match cmd/probes/terminalqualify/main.go:16 and the reproduced negative qualification.
          round: 2
      findings:
        - id: BR-5
          severity: Important
          title: Promised partition-coverage regression tests do not verify delivered bytes
          detail: 'cmd/internal/terminalqualify/runner_test.go:11 checks chunk counts and call counts, but never their contents; the completed plan checkbox at workshop/plans/000255-terminal-abstraction-plan.md:161 promises detection of missed splits. Replacing cases.go:20 with []string{input, ""} leaves both complete harness packages green. This is the 2nd finding in family qualification-observation-equivalence. Earlier corrections covered observation comparison; enforce the complete rule across partition generation and comparison: preserve the input bytes, enumerate every byte boundary, include byte-at-a-time delivery, and detect changed observations. Add independent partition expectations covering empty, single-byte, multibyte, and control-sequence inputs, and require this mutation to fail. ARCH-PURPOSE.'
          family: qualification-observation-equivalence
          round: 2
      boundary: M1
      blocked: true
    - "n": 3
      timestamp: "2026-09-15T10:30:52-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: 'Retained: runner.go compares complete whole/split observations in both directions; TestRunCaseDetectsUnassertedSplitStateChanges passes.'
          round: 3
        - id: BR-2
          disposition: addressed
          note: 'Retained: candidate.go captures attributes, underline style and color; snapshot regression tests and twelve literal style fixtures pass.'
          round: 3
        - id: BR-3
          disposition: addressed
          note: 'Retained: report.go and runner.go preserve bounded structured evidence; runner and CLI JSON regression tests pass.'
          round: 3
        - id: BR-4
          disposition: addressed
          note: README.md:773 documents invocation, evidence and exit meanings, matching cmd/probes/terminalqualify/main.go and the reproduced probe result.
          round: 3
        - id: BR-5
          disposition: addressed
          note: cases_test.go:11, :58 and :67 verify literal partitions, byte preservation, every boundary and production executor delivery. A temporary Go overlay replacing the split pair with []string{input, ""} makes all three tests fail; unmodified tests pass.
          round: 3
      boundary: M1
      blocked: false
    - "n": 4
      timestamp: "2026-09-15T12:55:07-07:00"
      agent: codex
      findings:
        - id: BR-6
          severity: Critical
          title: A gesture beginning on chrome leaks motion and release into the child
          detail: 'cmd/internal/terminal/presenter.go:367-393 drops an outside press without recording ownership or suppressing its remainder. With 1002/1006 enabled, press at (2,4) on the reserved row, move to (2,2), then release: the child receives "\x1b[<32;3;3M\x1b[<0;3;3m" despite receiving no press. The scratch TestReviewChromeGesture reproduces this. ARCH-ORDER: model child-owned and parent-owned gestures explicitly in View, and admit button motion/release only under the corresponding ownership. Cover chrome-to-child, panel-to-child, orphan events, and tracking changes during a gesture.'
          family: gesture-origin-ownership
          round: 4
        - id: BR-7
          severity: Critical
          title: Parameter overflow executes a truncated CSI command
          detail: 'third_party/vt/emulator.go:103 bounds parameter storage, but third_party/vt/csi.go:11-15 dispatches the truncated parameters without overflow rejection. Feeding CSI ?1002; followed by 32 copies of 1006; and then 1004h enables tracking 1002 and SGR while dropping the final requested mode. TestReviewOverflowParameters reproduces this, contradicting the plan''s reject-overflow-effects contract. ARCH-SECURE / ARCH-CONSTRAINTS: retain overflow evidence and reject the entire command through its terminator. Sweep CSI/DCS parameter consumers and test boundary counts, split input, no partial effects, and subsequent recovery.'
          family: overflow-command-atomicity
          round: 4
        - id: BR-8
          severity: Critical
          title: Endpoint cursor metadata remains stale after backend reset
          detail: 'cmd/internal/terminal/endpoint.go:113-115 maintains cursor metadata through callbacks, and capture at line 205 publishes that shadow state. third_party/vt/screen.go:35-40 resets the actual cursor without a style callback. Feeding "\x1b[6 q\x1bc" therefore publishes Shape:3, Blink:false after reset instead of the backend''s default blinking block. TestReviewCursorResetPublication reproduces this. ARCH-DRY / ARCH-ORDER: derive the full cursor publication from authoritative backend state, or enforce complete notifications for every mutation. Sweep reset, saved-cursor restore, and screen switching rather than repairing only RIS.'
          family: authoritative-snapshot-coherence
          round: 4
      boundary: M2
      blocked: true
    - "n": 5
      timestamp: "2026-09-15T13:11:26-07:00"
      agent: codex
      dispose:
        - id: BR-6
          disposition: addressed
          note: Chrome, panel, orphan-event and negotiation-change regressions pass. Restoring the previous input handler makes the committed chrome/panel/orphan regressions fail with leaked motion and release.
          round: 5
        - id: BR-7
          disposition: addressed
          note: CSI/DCS count, numeric-overflow, split-input and recovery tests pass. Removing overflow evidence collection makes atomicity and boundary-count regressions fail.
          round: 5
        - id: BR-8
          disposition: addressed
          note: Endpoint capture and qualification observations read authoritative backend cursor state. Restoring callback-maintained cursor metadata makes reset, restore and alternate-buffer regressions fail.
          round: 5
      findings:
        - id: BR-9
          severity: Critical
          title: Failed resize resumes a gesture after delivering its cancellation release
          detail: 'cmd/internal/terminal/presenter.go:505-509 cancels the drag before resizing, but returns directly when the resize callback fails; cancelDrag at lines 202-214 never transitions ownership. With 1002/1006 enabled, press (1,1), fail the resize callback, then move/release at (2,2): the child receives press, synthetic release, motion, and another release. TestReviewFailedResizeCancelsGesture reproduces this on the pinned head. ARCH-ORDER: cancellation must revoke child ownership independently of subsequent geometry success. This is the 2nd finding in family gesture-origin-ownership. Do NOT fix only this instance: enforce that rule across all six cancellation callers—release, failure, selection, panel, negotiation reconciliation and resize—including interrupted delivery and retry.'
          family: gesture-origin-ownership
          round: 5
      boundary: M2
      blocked: true
    - "n": 6
      timestamp: "2026-09-15T13:23:31-07:00"
      agent: codex
      dispose:
        - id: BR-9
          disposition: addressed
          note: presenter.go:205 commits CancelMouse before delivery and tracks its pending acknowledgment. Tests at presenter_test.go:608,638,661,727,760 cover failed resize, interrupted delivery, all six callers, child-write failure, and cancellation during selection. Removing ownership revocation in a temporary overlay makes the resize regression and all six caller cases fail.
          round: 6
        - id: BR-6
          disposition: addressed
          note: 'Existing disposition retained: explicit gesture ownership and negotiation epochs remain enforced; chrome, panel, orphan-event, and mode-change regressions pass.'
          round: 6
        - id: BR-7
          disposition: addressed
          note: 'Existing disposition retained: CSI/DCS dispatch rejects retained overflow evidence; parameter boundary, numeric overflow, split-input, and recovery regressions pass.'
          round: 6
        - id: BR-8
          disposition: addressed
          note: 'Existing disposition retained: Endpoint captures authoritative backend cursor state; reset, restore, and buffer-switch regressions pass.'
          round: 6
      findings:
        - id: BR-10
          severity: Important
          title: Native discovery runs retain temporary artifacts without cleanup or a bound
          detail: 'tests/terminal-oracle/discovery/zellij_oracle.py:5 creates a new /tmp/pw* directory for every invocation, while its finally block at lines 34–46 only stops processes and closes handles. All six discovery probes share this driver, and README.md:24–26 explicitly retains the directories without defining removal or a retention bound. ARCH-FUNERAL: make the driver remove its directory after teardown, including failure paths; any retained diagnostic mode needs an explicit bounded lifecycle. Cover successful and failed runs with cleanup regression tests.'
          family: artifact-lifetime-ownership
          round: 6
      boundary: M2
      blocked: true
    - "n": 7
      timestamp: "2026-09-15T13:31:32-07:00"
      agent: codex
      dispose:
        - id: BR-10
          disposition: addressed
          note: zellij_oracle.py:34 scopes setup, execution and teardown inside TemporaryDirectory. All five discovery tests pass; replacing cleanup with a no-op in memory makes all four cleanup tests fail on leaked directories.
          round: 7
        - id: BR-6
          disposition: addressed
          note: Explicit parent/child gesture ownership remains enforced; presenter_test.go:507 exercises chrome, panel and orphan gestures. Focused normal/race suites pass.
          round: 7
        - id: BR-7
          disposition: addressed
          note: parameterGuard preserves overflow evidence and rejects dispatch; pair_parameter_test.go:13 covers atomic rejection. Fork normal/race suites pass.
          round: 7
        - id: BR-8
          disposition: addressed
          note: Endpoint captures the backend's authoritative cursor; endpoint_test.go:294 covers reset, restore and buffer transitions. Focused normal/race suites pass.
          round: 7
        - id: BR-9
          disposition: addressed
          note: Cancellation revokes ownership before delivery and tracks pending release. presenter_test.go:608, :638 and :661 cover failed resize, interrupted delivery and cancellation callers.
          round: 7
      findings:
        - id: BR-11
          severity: Critical
          title: Successful release leaves autowrap disabled after an interrupted paint
          detail: 'ARCH-ORDER: render.go:34 emits CSI ?7l, but presenter.go:127 omits CSI ?7h from release cleanup. A production-presenter test injecting failure immediately after ?7l, followed by successful Release, reproduces ABCDEFGI on one eight-column row instead of ABCDEFGH followed by I in the independent xterm oracle. Enumerate all parent state changed during painting and restore the required post-release state after any accepted prefix; add interrupted-paint cleanup regressions, including hyperlink state.'
          family: parent-terminal-restoration
          round: 7
      boundary: M2
      blocked: true
    - "n": 8
      timestamp: "2026-09-15T13:42:44-07:00"
      agent: codex
      dispose:
        - id: BR-6
          disposition: addressed
          note: Parent/orphan gesture and negotiation-epoch regressions remain present and pass in the focused normal/race suites.
          round: 8
        - id: BR-7
          disposition: addressed
          note: Atomic parameter overflow guards and boundary regressions remain present; fork normal/race suites pass.
          round: 8
        - id: BR-8
          disposition: addressed
          note: Endpoint and qualification snapshots consume the authoritative copied cursor; reset/restore/buffer regressions pass.
          round: 8
        - id: BR-9
          disposition: addressed
          note: Cancellation commits independently of resize and retains pending delivery; failed-resize and interrupted-cancellation regressions pass.
          round: 8
        - id: BR-10
          disposition: addressed
          note: Native discovery uses invocation-scoped temporary storage; all five cleanup and bounded-capture tests pass.
          round: 8
        - id: BR-11
          disposition: addressed
          note: presenter.go:244 restores the parent baseline. The independent interrupted-presentation test passes; scratch mutations removing autowrap restoration or hyperlink closure both make it fail.
          round: 8
      findings:
        - id: BR-12
          severity: Critical
          title: Zero-width Unicode output permanently fails the presenter
          detail: 'third_party/vt/utf8.go:49-51 stores an initial zero-width grapheme as a nonempty Width:0 cell, which cmd/internal/terminal/frame.go:97-100 rejects. Production Feed → Present → Flush reproduces Failed state for U+0301, U+200D, U+FE0F, and a combining mark following SGR or cursor movement. ARCH-PURPOSE: define coherent zero-width rendering across backend, frame validation, and serialization; cover the entire class with split-input and production-presentation regressions rather than weakening frame validation.'
          family: unicode-cell-coherence
          round: 8
      boundary: M2
      blocked: true
    - "n": 9
      timestamp: "2026-09-15T13:57:53-07:00"
      agent: codex
      dispose:
        - id: BR-12
          disposition: addressed
          note: Backend orphan handling, strict grapheme validation, and StyledRows now agree. Production presentation/input and split-input regressions pass at HEAD and fail with the three pre-fix implementation files restored through a temporary Go overlay.
          round: 9
        - id: BR-6
          disposition: addressed
          note: Prior disposition retained; parent/orphan gesture and negotiation-epoch regressions pass.
          round: 9
        - id: BR-7
          disposition: addressed
          note: Prior disposition retained; atomic parameter-overflow handling and fork regressions pass.
          round: 9
        - id: BR-8
          disposition: addressed
          note: Prior disposition retained; snapshots consume authoritative cursor state and cursor qualification cases pass.
          round: 9
        - id: BR-9
          disposition: addressed
          note: Prior disposition retained; failed-resize and interrupted-cancellation regressions pass.
          round: 9
        - id: BR-10
          disposition: addressed
          note: Prior disposition retained; scoped temporary-directory cleanup and five native-driver tests pass.
          round: 9
        - id: BR-11
          disposition: addressed
          note: Prior disposition retained; independent interrupted-presentation restoration tests pass.
          round: 9
      boundary: M2
      blocked: false
    - "n": 10
      timestamp: "2026-09-15T14:54:01-07:00"
      agent: codex
      findings:
        - id: BR-13
          severity: Critical
          title: History serialization drops backgrounds on erased interior cells
          detail: 'cmd/internal/terminal/history_render.go:213 emits only CUF for empty-content cells; background restoration covers only the trailing extent. A production-presenter scratch regression with red EL2 followed by default-style text fails independent xterm comparison: BG=-1 instead of BG=1. Preserve erased-cell attributes across viewport/history append and rebuild paths, with independent regressions. ARCH-PURPOSE.'
          family: terminal-cell-attribute-fidelity
          round: 10
        - id: BR-14
          severity: Important
          title: Couch removes exited children without disposing their publication workers
          detail: cmd/internal/couchtty/console.go:934 retires the origin without Child.Close; cmd/internal/ptychild/publication.go:93 waits indefinitely without cancellation. A scratch regression confirms the endpoint remains usable after removal and a subsequent console command. This is the 2nd finding in family artifact-lifetime-ownership. State the drain/deselect/retire/dispose rule and sweep natural exit, park/relaunch, failed attachment, and teardown rather than fixing only this instance. ARCH-FUNERAL / ARCH-ORDER.
          family: artifact-lifetime-ownership
          round: 10
        - id: BR-15
          severity: Important
          title: Runtime generation directly invokes tic without an injectable compiler seam
          detail: cmd/internal/runtimebundlegen/generate.go:84 introduces a direct external compiler dependency; generator tests use the installed binary rather than a fake behind the production seam. Inject compilation, model output files and failures in portable test storage, and add a dedicated real-compiler conformance comparison. ARCH-MOCK.
          family: external-dependency-seam
          round: 10
      boundary: M3
      blocked: true
    - "n": 11
      timestamp: "2026-09-15T15:11:38-07:00"
      agent: codex
      dispose:
        - id: BR-13
          disposition: addressed
          note: Independent erased-background oracle tests pass at head; restoring the previous serializer makes indexed, RGB, and alternate viewport regressions fail.
          round: 11
        - id: BR-14
          disposition: addressed
          note: Production ownership and disposal paths plus exit, teardown, refusal, and rollback tests were inspected; restoring the previous Console makes the exited-origin disposal regression fail.
          round: 11
        - id: BR-15
          disposition: addressed
          note: Injected compiler and real tic/infocmp conformance tests pass; bypassing the injected compiler makes the PATH-isolated regression fail.
          round: 11
      findings:
        - id: BR-16
          severity: Critical
          title: Presenter resize allocates before validating geometry
          detail: cmd/internal/terminal/presenter.go:582 allocates using host.Cols before validation. A scratch production-API regression with selected chrome and Cols=-1 crashes the presenter goroutine. Validate before allocating and cover invalid and over-limit geometry through both resize entrypoints. ARCH-SECURE / ARCH-CONSTRAINTS.
          family: validate-before-allocation
          round: 11
        - id: BR-17
          severity: Important
          title: Teardown regression mistakes ordinary paint output for shutdown completion
          detail: 'cmd/internal/couchtty/vtscreen_test.go:165 accepts ResetRegion already emitted by history_render.go:301; simulated shell writes race teardown. Reproduced in the normal suite and 2 of 20 isolated repetitions. This is the 3rd finding in this family: require lifecycle completion acknowledgments and sweep teardown assertions across both consumers. ARCH-ORDER / ARCH-PURPOSE.'
          family: qualification-observation-equivalence
          round: 11
        - id: BR-18
          severity: Minor
          title: Harness guide still recommends deleted wrapper filtering
          detail: atlas/how-to-bring-up-a-new-harness-cli.md:108 describes stdoutChunk and Codex synchronized-output stripping, both removed by this range. Update the guide to the implemented wrapper contract.
          family: documentation-surface-accuracy
          round: 11
      boundary: M3
      blocked: true
---

# Gate ledger — 000255-lifecycle-state-ownership#255 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-15T10:17:29-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `qualification-observation-equivalence` Split qualification never compares complete observations
  cmd/internal/terminalqualify/runner.go:43 checks only sparse literal expectations, so split-only corruption outside those keys passes. ARCH-PURPOSE: retain the whole-input observation, compare complete observations across variants, and add regressions for otherwise-unasserted cells, styles, links, and replies.
- **BR-2** [Important] `required-capability-coverage` Non-color rendering attributes are neither observed nor qualified
  cmd/internal/terminalqualify/candidate.go:204 omits Style.Attrs, Underline, and UnderlineColor; fixtures and Coverage contain no corresponding obligations. ARCH-PURPOSE: capture these fields and add literal set/reset/preservation cases with regression evidence.
- **BR-3** [Important] `qualification-evidence-contract` JSON results omit the promised expected and observed evidence
  cmd/internal/terminalqualify/report.go:21 emits status/detail without the expected/observed results promised at plan line 135. ARCH-PURPOSE: add bounded structured evidence for executable cases and verify it through JSON tests.
- **BR-4** [Important] `readme-surface-documentation` README update appears missing for the terminal qualification probe
  cmd/probes/terminalqualify/main.go:41 adds a runnable diagnostic with distinct exit statuses, but README.md is unchanged in the pinned range. Document invocation, JSON output, exit meanings, and the qualification report.

## Round 2 — 2026-09-15T10:26:56-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — addressed — runner.go:53 compares complete observations in both directions. Disabling this comparison makes TestRunCaseDetectsUnassertedSplitStateChanges fail for cells, attributes, links, replies, and added keys.
- BR-2 — addressed — candidate.go:207 captures attributes, underline style, and underline color; screen_cases.go adds 12 literal set/reset/preservation cases. Removing attribute capture makes TestCandidateCapturesCompleteStyle fail.
- BR-3 — addressed — report.go:29 and runner.go:94 provide bounded expected/observed evidence. Removing expected JSON serialization makes TestRunJSONPreservesStructuredEvidence fail; the real probe emits evidence for every executable case.
- BR-4 — addressed — The pinned README.md:773 addition documents invocation, JSON evidence, exit meanings, and the qualification report. These match cmd/probes/terminalqualify/main.go:16 and the reproduced negative qualification.

### Raised

- **BR-5** [Important] `qualification-observation-equivalence` Promised partition-coverage regression tests do not verify delivered bytes
  cmd/internal/terminalqualify/runner_test.go:11 checks chunk counts and call counts, but never their contents; the completed plan checkbox at workshop/plans/000255-terminal-abstraction-plan.md:161 promises detection of missed splits. Replacing cases.go:20 with []string{input, ""} leaves both complete harness packages green. This is the 2nd finding in family qualification-observation-equivalence. Earlier corrections covered observation comparison; enforce the complete rule across partition generation and comparison: preserve the input bytes, enumerate every byte boundary, include byte-at-a-time delivery, and detect changed observations. Add independent partition expectations covering empty, single-byte, multibyte, and control-sequence inputs, and require this mutation to fail. ARCH-PURPOSE.

## Round 3 — 2026-09-15T10:30:52-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Retained: runner.go compares complete whole/split observations in both directions; TestRunCaseDetectsUnassertedSplitStateChanges passes.
- BR-2 — addressed — Retained: candidate.go captures attributes, underline style and color; snapshot regression tests and twelve literal style fixtures pass.
- BR-3 — addressed — Retained: report.go and runner.go preserve bounded structured evidence; runner and CLI JSON regression tests pass.
- BR-4 — addressed — README.md:773 documents invocation, evidence and exit meanings, matching cmd/probes/terminalqualify/main.go and the reproduced probe result.
- BR-5 — addressed — cases_test.go:11, :58 and :67 verify literal partitions, byte preservation, every boundary and production executor delivery. A temporary Go overlay replacing the split pair with []string{input, ""} makes all three tests fail; unmodified tests pass.

## Round 4 — 2026-09-15T12:55:07-07:00 (codex) — BLOCKED

### Raised

- **BR-6** [Critical] `gesture-origin-ownership` A gesture beginning on chrome leaks motion and release into the child
  cmd/internal/terminal/presenter.go:367-393 drops an outside press without recording ownership or suppressing its remainder. With 1002/1006 enabled, press at (2,4) on the reserved row, move to (2,2), then release: the child receives "\x1b[<32;3;3M\x1b[<0;3;3m" despite receiving no press. The scratch TestReviewChromeGesture reproduces this. ARCH-ORDER: model child-owned and parent-owned gestures explicitly in View, and admit button motion/release only under the corresponding ownership. Cover chrome-to-child, panel-to-child, orphan events, and tracking changes during a gesture.
- **BR-7** [Critical] `overflow-command-atomicity` Parameter overflow executes a truncated CSI command
  third_party/vt/emulator.go:103 bounds parameter storage, but third_party/vt/csi.go:11-15 dispatches the truncated parameters without overflow rejection. Feeding CSI ?1002; followed by 32 copies of 1006; and then 1004h enables tracking 1002 and SGR while dropping the final requested mode. TestReviewOverflowParameters reproduces this, contradicting the plan's reject-overflow-effects contract. ARCH-SECURE / ARCH-CONSTRAINTS: retain overflow evidence and reject the entire command through its terminator. Sweep CSI/DCS parameter consumers and test boundary counts, split input, no partial effects, and subsequent recovery.
- **BR-8** [Critical] `authoritative-snapshot-coherence` Endpoint cursor metadata remains stale after backend reset
  cmd/internal/terminal/endpoint.go:113-115 maintains cursor metadata through callbacks, and capture at line 205 publishes that shadow state. third_party/vt/screen.go:35-40 resets the actual cursor without a style callback. Feeding "\x1b[6 q\x1bc" therefore publishes Shape:3, Blink:false after reset instead of the backend's default blinking block. TestReviewCursorResetPublication reproduces this. ARCH-DRY / ARCH-ORDER: derive the full cursor publication from authoritative backend state, or enforce complete notifications for every mutation. Sweep reset, saved-cursor restore, and screen switching rather than repairing only RIS.

## Round 5 — 2026-09-15T13:11:26-07:00 (codex) — BLOCKED

### Disposed

- BR-6 — addressed — Chrome, panel, orphan-event and negotiation-change regressions pass. Restoring the previous input handler makes the committed chrome/panel/orphan regressions fail with leaked motion and release.
- BR-7 — addressed — CSI/DCS count, numeric-overflow, split-input and recovery tests pass. Removing overflow evidence collection makes atomicity and boundary-count regressions fail.
- BR-8 — addressed — Endpoint capture and qualification observations read authoritative backend cursor state. Restoring callback-maintained cursor metadata makes reset, restore and alternate-buffer regressions fail.

### Raised

- **BR-9** [Critical] `gesture-origin-ownership` Failed resize resumes a gesture after delivering its cancellation release
  cmd/internal/terminal/presenter.go:505-509 cancels the drag before resizing, but returns directly when the resize callback fails; cancelDrag at lines 202-214 never transitions ownership. With 1002/1006 enabled, press (1,1), fail the resize callback, then move/release at (2,2): the child receives press, synthetic release, motion, and another release. TestReviewFailedResizeCancelsGesture reproduces this on the pinned head. ARCH-ORDER: cancellation must revoke child ownership independently of subsequent geometry success. This is the 2nd finding in family gesture-origin-ownership. Do NOT fix only this instance: enforce that rule across all six cancellation callers—release, failure, selection, panel, negotiation reconciliation and resize—including interrupted delivery and retry.

## Round 6 — 2026-09-15T13:23:31-07:00 (codex) — BLOCKED

### Disposed

- BR-9 — addressed — presenter.go:205 commits CancelMouse before delivery and tracks its pending acknowledgment. Tests at presenter_test.go:608,638,661,727,760 cover failed resize, interrupted delivery, all six callers, child-write failure, and cancellation during selection. Removing ownership revocation in a temporary overlay makes the resize regression and all six caller cases fail.
- BR-6 — addressed — Existing disposition retained: explicit gesture ownership and negotiation epochs remain enforced; chrome, panel, orphan-event, and mode-change regressions pass.
- BR-7 — addressed — Existing disposition retained: CSI/DCS dispatch rejects retained overflow evidence; parameter boundary, numeric overflow, split-input, and recovery regressions pass.
- BR-8 — addressed — Existing disposition retained: Endpoint captures authoritative backend cursor state; reset, restore, and buffer-switch regressions pass.

### Raised

- **BR-10** [Important] `artifact-lifetime-ownership` Native discovery runs retain temporary artifacts without cleanup or a bound
  tests/terminal-oracle/discovery/zellij_oracle.py:5 creates a new /tmp/pw* directory for every invocation, while its finally block at lines 34–46 only stops processes and closes handles. All six discovery probes share this driver, and README.md:24–26 explicitly retains the directories without defining removal or a retention bound. ARCH-FUNERAL: make the driver remove its directory after teardown, including failure paths; any retained diagnostic mode needs an explicit bounded lifecycle. Cover successful and failed runs with cleanup regression tests.

## Round 7 — 2026-09-15T13:31:32-07:00 (codex) — BLOCKED

### Disposed

- BR-10 — addressed — zellij_oracle.py:34 scopes setup, execution and teardown inside TemporaryDirectory. All five discovery tests pass; replacing cleanup with a no-op in memory makes all four cleanup tests fail on leaked directories.
- BR-6 — addressed — Explicit parent/child gesture ownership remains enforced; presenter_test.go:507 exercises chrome, panel and orphan gestures. Focused normal/race suites pass.
- BR-7 — addressed — parameterGuard preserves overflow evidence and rejects dispatch; pair_parameter_test.go:13 covers atomic rejection. Fork normal/race suites pass.
- BR-8 — addressed — Endpoint captures the backend's authoritative cursor; endpoint_test.go:294 covers reset, restore and buffer transitions. Focused normal/race suites pass.
- BR-9 — addressed — Cancellation revokes ownership before delivery and tracks pending release. presenter_test.go:608, :638 and :661 cover failed resize, interrupted delivery and cancellation callers.

### Raised

- **BR-11** [Critical] `parent-terminal-restoration` Successful release leaves autowrap disabled after an interrupted paint
  ARCH-ORDER: render.go:34 emits CSI ?7l, but presenter.go:127 omits CSI ?7h from release cleanup. A production-presenter test injecting failure immediately after ?7l, followed by successful Release, reproduces ABCDEFGI on one eight-column row instead of ABCDEFGH followed by I in the independent xterm oracle. Enumerate all parent state changed during painting and restore the required post-release state after any accepted prefix; add interrupted-paint cleanup regressions, including hyperlink state.

## Round 8 — 2026-09-15T13:42:44-07:00 (codex) — BLOCKED

### Disposed

- BR-6 — addressed — Parent/orphan gesture and negotiation-epoch regressions remain present and pass in the focused normal/race suites.
- BR-7 — addressed — Atomic parameter overflow guards and boundary regressions remain present; fork normal/race suites pass.
- BR-8 — addressed — Endpoint and qualification snapshots consume the authoritative copied cursor; reset/restore/buffer regressions pass.
- BR-9 — addressed — Cancellation commits independently of resize and retains pending delivery; failed-resize and interrupted-cancellation regressions pass.
- BR-10 — addressed — Native discovery uses invocation-scoped temporary storage; all five cleanup and bounded-capture tests pass.
- BR-11 — addressed — presenter.go:244 restores the parent baseline. The independent interrupted-presentation test passes; scratch mutations removing autowrap restoration or hyperlink closure both make it fail.

### Raised

- **BR-12** [Critical] `unicode-cell-coherence` Zero-width Unicode output permanently fails the presenter
  third_party/vt/utf8.go:49-51 stores an initial zero-width grapheme as a nonempty Width:0 cell, which cmd/internal/terminal/frame.go:97-100 rejects. Production Feed → Present → Flush reproduces Failed state for U+0301, U+200D, U+FE0F, and a combining mark following SGR or cursor movement. ARCH-PURPOSE: define coherent zero-width rendering across backend, frame validation, and serialization; cover the entire class with split-input and production-presentation regressions rather than weakening frame validation.

## Round 9 — 2026-09-15T13:57:53-07:00 (codex) — passed

### Disposed

- BR-12 — addressed — Backend orphan handling, strict grapheme validation, and StyledRows now agree. Production presentation/input and split-input regressions pass at HEAD and fail with the three pre-fix implementation files restored through a temporary Go overlay.
- BR-6 — addressed — Prior disposition retained; parent/orphan gesture and negotiation-epoch regressions pass.
- BR-7 — addressed — Prior disposition retained; atomic parameter-overflow handling and fork regressions pass.
- BR-8 — addressed — Prior disposition retained; snapshots consume authoritative cursor state and cursor qualification cases pass.
- BR-9 — addressed — Prior disposition retained; failed-resize and interrupted-cancellation regressions pass.
- BR-10 — addressed — Prior disposition retained; scoped temporary-directory cleanup and five native-driver tests pass.
- BR-11 — addressed — Prior disposition retained; independent interrupted-presentation restoration tests pass.

## Round 10 — 2026-09-15T14:54:01-07:00 (codex) — BLOCKED

### Raised

- **BR-13** [Critical] `terminal-cell-attribute-fidelity` History serialization drops backgrounds on erased interior cells
  cmd/internal/terminal/history_render.go:213 emits only CUF for empty-content cells; background restoration covers only the trailing extent. A production-presenter scratch regression with red EL2 followed by default-style text fails independent xterm comparison: BG=-1 instead of BG=1. Preserve erased-cell attributes across viewport/history append and rebuild paths, with independent regressions. ARCH-PURPOSE.
- **BR-14** [Important] `artifact-lifetime-ownership` Couch removes exited children without disposing their publication workers
  cmd/internal/couchtty/console.go:934 retires the origin without Child.Close; cmd/internal/ptychild/publication.go:93 waits indefinitely without cancellation. A scratch regression confirms the endpoint remains usable after removal and a subsequent console command. This is the 2nd finding in family artifact-lifetime-ownership. State the drain/deselect/retire/dispose rule and sweep natural exit, park/relaunch, failed attachment, and teardown rather than fixing only this instance. ARCH-FUNERAL / ARCH-ORDER.
- **BR-15** [Important] `external-dependency-seam` Runtime generation directly invokes tic without an injectable compiler seam
  cmd/internal/runtimebundlegen/generate.go:84 introduces a direct external compiler dependency; generator tests use the installed binary rather than a fake behind the production seam. Inject compilation, model output files and failures in portable test storage, and add a dedicated real-compiler conformance comparison. ARCH-MOCK.

## Round 11 — 2026-09-15T15:11:38-07:00 (codex) — BLOCKED

### Disposed

- BR-13 — addressed — Independent erased-background oracle tests pass at head; restoring the previous serializer makes indexed, RGB, and alternate viewport regressions fail.
- BR-14 — addressed — Production ownership and disposal paths plus exit, teardown, refusal, and rollback tests were inspected; restoring the previous Console makes the exited-origin disposal regression fail.
- BR-15 — addressed — Injected compiler and real tic/infocmp conformance tests pass; bypassing the injected compiler makes the PATH-isolated regression fail.

### Raised

- **BR-16** [Critical] `validate-before-allocation` Presenter resize allocates before validating geometry
  cmd/internal/terminal/presenter.go:582 allocates using host.Cols before validation. A scratch production-API regression with selected chrome and Cols=-1 crashes the presenter goroutine. Validate before allocating and cover invalid and over-limit geometry through both resize entrypoints. ARCH-SECURE / ARCH-CONSTRAINTS.
- **BR-17** [Important] `qualification-observation-equivalence` Teardown regression mistakes ordinary paint output for shutdown completion
  cmd/internal/couchtty/vtscreen_test.go:165 accepts ResetRegion already emitted by history_render.go:301; simulated shell writes race teardown. Reproduced in the normal suite and 2 of 20 isolated repetitions. This is the 3rd finding in this family: require lifecycle completion acknowledgments and sweep teardown assertions across both consumers. ARCH-ORDER / ARCH-PURPOSE.
- **BR-18** [Minor] `documentation-surface-accuracy` Harness guide still recommends deleted wrapper filtering
  atlas/how-to-bring-up-a-new-harness-cli.md:108 describes stdoutChunk and Codex synchronized-output stripping, both removed by this range. Update the guide to the implemented wrapper contract.

## Open findings

- **BR-16** [Critical] `validate-before-allocation` Presenter resize allocates before validating geometry
- **BR-17** [Important] `qualification-observation-equivalence` Teardown regression mistakes ordinary paint output for shutdown completion
- **BR-18** [Minor] `documentation-surface-accuracy` Harness guide still recommends deleted wrapper filtering
