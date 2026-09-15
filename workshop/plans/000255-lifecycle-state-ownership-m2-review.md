# Boundary Review — 000255-lifecycle-state-ownership#255 (milestone M2)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | 000255-lifecycle-state-ownership |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 29101ebf157ba9663609f5e75278449f34eea722..d44ff360e725c52f7510c52ac9c1cb5342e50182 |
| command | sdlc milestone-close --issue 255 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-09-15T12:55:07-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M2 establishes useful shared boundaries, bounded transports, independent rendering checks, and honest separation from pending M3/M4 adoption. However, three reproduced correctness defects block this boundary: gesture ownership leaks across chrome, oversized CSI commands execute partially, and cursor publications diverge from backend state. No repository files were changed.

## 1. Strengths

- Input admission waits for complete presentation; partial writes close admission. Tests exercise blocked and failed writes through the shared transport.
- Renderer output passes the independent xterm-headless oracle, including wide-cell changes, styles, chrome, and cursor positioning.
- Qualification preserves required obligations: reproduced **81 pass, 0 fail, 6 not-covered**, with incomplete qualification correctly returning exit 1.
- README and atlas cover the new library, profile, fork maintenance, and oracle command. The fork’s provenance and inherited-test changes are documented and inspectable.

## 2. Critical findings

```findings
findings:
  - id: new
    severity: Critical
    family: gesture-origin-ownership
    title: |
      A gesture beginning on chrome leaks motion and release into the child
    detail: |
      cmd/internal/terminal/presenter.go:367-393 drops an outside press without recording ownership or suppressing its remainder. With 1002/1006 enabled, press at (2,4) on the reserved row, move to (2,2), then release: the child receives "\x1b[<32;3;3M\x1b[<0;3;3m" despite receiving no press. The scratch TestReviewChromeGesture reproduces this. ARCH-ORDER: model child-owned and parent-owned gestures explicitly in View, and admit button motion/release only under the corresponding ownership. Cover chrome-to-child, panel-to-child, orphan events, and tracking changes during a gesture.
  - id: new
    severity: Critical
    family: overflow-command-atomicity
    title: |
      Parameter overflow executes a truncated CSI command
    detail: |
      third_party/vt/emulator.go:103 bounds parameter storage, but third_party/vt/csi.go:11-15 dispatches the truncated parameters without overflow rejection. Feeding CSI ?1002; followed by 32 copies of 1006; and then 1004h enables tracking 1002 and SGR while dropping the final requested mode. TestReviewOverflowParameters reproduces this, contradicting the plan's reject-overflow-effects contract. ARCH-SECURE / ARCH-CONSTRAINTS: retain overflow evidence and reject the entire command through its terminator. Sweep CSI/DCS parameter consumers and test boundary counts, split input, no partial effects, and subsequent recovery.
  - id: new
    severity: Critical
    family: authoritative-snapshot-coherence
    title: |
      Endpoint cursor metadata remains stale after backend reset
    detail: |
      cmd/internal/terminal/endpoint.go:113-115 maintains cursor metadata through callbacks, and capture at line 205 publishes that shadow state. third_party/vt/screen.go:35-40 resets the actual cursor without a style callback. Feeding "\x1b[6 q\x1bc" therefore publishes Shape:3, Blink:false after reset instead of the backend's default blinking block. TestReviewCursorResetPublication reproduces this. ARCH-DRY / ARCH-ORDER: derive the full cursor publication from authoritative backend state, or enforce complete notifications for every mutation. Sweep reset, saved-cursor restore, and screen switching rather than repairing only RIS.
```

## 3. Important findings

None additional.

## 4. Minor findings

None.

## 5. Test coverage notes

**Passed:**

- Focused terminal, ttyio, qualification, and probe tests.
- Focused terminal/ttyio/qualification race tests.
- Fork normal and race suites.
- Independent renderer oracle.
- Pinned-range `git diff --check`.

**Three additional regression assertions failed**, exposing the findings above. Reproduce using the temporary overlay:

```sh
go test -overlay /tmp/pair255-review-3uybpf_d/overlay.json \
  ./cmd/internal/terminal -run '^TestReview' -count=1 -v
```

Existing tests check parameter retention without checking rejection, chrome presses without their subsequent gesture, and cursor setters without reset publication. Commit regressions covering those sequences and verify they fail without the fixes.

The full root suite and live consumer workflows were not rerun in this review. There are no prior findings requiring disposition.

## 6. Architectural notes for upcoming work

| Marker | Result | Assessment |
|---|---|---|
| ARCH-DRY | **Flag** | Callback-maintained cursor metadata duplicates backend authority and diverges. |
| ARCH-PURE | **Pass** | Frame composition, rendering, and View transitions have direct literal tests; IO sits in integration components. |
| ARCH-PURPOSE | **Pass for M2** | Shared implementation exists; consumer migration remains explicitly required in M3. |
| ARCH-MOCK | **Pass for M2** | Stateful transport fakes share the production seam; independent interpretation supplements them. Native conformance remains M4 work. |
| ARCH-CONSTRAINTS | **Flag** | Parameter storage is bounded, but overflow behavior violates the declared envelope. |
| ARCH-SECURE | **Flag** | Oversized external commands execute a truncated subset of their effects. |
| ARCH-ORDER | **Flag** | Gesture ownership is incomplete; published cursor state can contradict authoritative state. |
| ARCH-FUNERAL | **Pass** | Workers have joined teardown, retained structures have bounds, and origins have retirement. |

Core-concept implementations are present at the mapped locations. The PURE composition/View core is tested directly; external interpreter and terminfo checks are integration verification.

## 7. Plan revision recommendations

Add `## Revisions` entries specifying:

- **Gesture ownership:** enumerate parent-owned, child-owned, and orphan gesture transitions and their sequence tests.
- **Overflow atomicity:** define command-wide rejection and recovery across parameter-bearing protocols.
- **Publication authority:** require complete cursor snapshots across every backend mutation, including reset, restore, and buffer switching.

Keep the affected M2 tasks open until these regressions pass and the boundary is rerun.

---

## Re-review — 2026-09-15T13:11:26-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | 000255-lifecycle-state-ownership |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 29101ebf157ba9663609f5e75278449f34eea722..6015b51b2d5760e1abef11d1082c1cfcf338f06a |
| command | sdlc milestone-close --issue 255 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-09-15T13:11:26-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

The three prior findings are addressed, with regression tests that fail when the fixes are removed in temporary overlays. The focused normal/race suites and independent renderer oracle pass. One additional cancellation bug blocks M2: a failed resize leaves a gesture admitted after its synthetic release has already reached the child.

```findings
dispose:
  - id: BR-6
    disposition: addressed
    note: |
      Chrome, panel, orphan-event and negotiation-change regressions pass. Restoring the previous input handler makes the committed chrome/panel/orphan regressions fail with leaked motion and release.
  - id: BR-7
    disposition: addressed
    note: |
      CSI/DCS count, numeric-overflow, split-input and recovery tests pass. Removing overflow evidence collection makes atomicity and boundary-count regressions fail.
  - id: BR-8
    disposition: addressed
    note: |
      Endpoint capture and qualification observations read authoritative backend cursor state. Restoring callback-maintained cursor metadata makes reset, restore and alternate-buffer regressions fail.
findings:
  - id: new
    severity: Critical
    family: gesture-origin-ownership
    title: |
      Failed resize resumes a gesture after delivering its cancellation release
    detail: |
      cmd/internal/terminal/presenter.go:505-509 cancels the drag before resizing, but returns directly when the resize callback fails; cancelDrag at lines 202-214 never transitions ownership. With 1002/1006 enabled, press (1,1), fail the resize callback, then move/release at (2,2): the child receives press, synthetic release, motion, and another release. TestReviewFailedResizeCancelsGesture reproduces this on the pinned head. ARCH-ORDER: cancellation must revoke child ownership independently of subsequent geometry success. This is the 2nd finding in family gesture-origin-ownership. Do NOT fix only this instance: enforce that rule across all six cancellation callers—release, failure, selection, panel, negotiation reconciliation and resize—including interrupted delivery and retry.
```

## 1. Strengths

- Cursor publication now uses one authoritative backend value, covering reset, saved-cursor restoration and buffer switching.
- Overflow rejection happens before CSI/DCS dispatch; tests cover exact limits, subparameters, numeric overflow and recovery.
- Stateful transport doubles exercise accepted prefixes, blocked writes, cancellation and joined teardown.
- README and atlas updates describe the new surfaces. Qualification preserves six explicit M3/M4 gaps rather than claiming complete adoption.

## 2. Critical findings

**Failed-resize gesture cancellation**, at [presenter.go:505](/Users/xianxu/workspace/worktree/pair/000255-lifecycle-state-ownership/cmd/internal/terminal/presenter.go:505).

The reproduced child wire is:

```text
Expected: \x1b[<0;2;2M\x1b[<0;2;2m
Actual:   \x1b[<0;2;2M\x1b[<0;2;2m\x1b[<32;3;3M\x1b[<0;3;3m
```

Preserve the old geometry after resize failure, but preserve the cancellation too. Once the synthetic release is admitted, suppress the physical gesture’s remainder. Model interrupted delivery explicitly so retries cannot duplicate releases.

The existing failed-resize test at `presenter_test.go:421` starts without an active gesture and therefore misses this case.

## 3. Important findings

None.

## 4. Minor findings

None.

## 5. Test coverage notes

Verified:

- `terminal`, `terminalqualify`, and `ttyio`: normal and race suites pass.
- Local VT fork: normal and race suites pass.
- Required independent renderer oracle passes.
- Qualification: **84 pass, 0 fail, 6 not-covered**, with expected exit status 1.
- Pinned-range `git diff --check` passes.
- All three prior fixes have failing mutation evidence.

The new reproduction is in [review_test.go](/var/folders/07/b9wcwwld4_v2w9r3hk525bm80000gn/T/pair255-review-6irszsf7/review_test.go). Repository files were unchanged. The full root suite was not rerun.

## 6. Architectural notes for upcoming work

| Marker | Result | Review assessment |
|---|---|---|
| ARCH-DRY | Pass | Cursor consumers share backend authority; terminfo is checked against the capability source. |
| ARCH-PURE | Pass | Frame/view/render logic is directly testable; transport and backend integrations are separate. |
| ARCH-PURPOSE | Flag | Cancellation enforcement remains incomplete across failure paths; sweep the whole family. |
| ARCH-MOCK | Pass | Stateful write doubles share the production seam; PTY and independent interpreter tests add conformance evidence. |
| ARCH-CONSTRAINTS | Pass for M2 | Parser, history and queue bounds are explicit and tested; complete workload acceptance remains M4. |
| ARCH-SECURE | Pass | Overflow commands are rejected atomically; renderer/effect boundaries validate control-bearing data. |
| ARCH-ORDER | Flag | Successful cancellation is lost when the subsequent resize fails. |
| ARCH-FUNERAL | Pass | Workers have joined teardown; origins have retirement and retained buffers have bounds. |

The M2 core-concept locations and implementation roles are present. M3 consumer/history work remains explicitly pending.

## 7. Plan revision recommendations

Add a `## Revisions` entry stating:

> Gesture cancellation commits independently of selection or resize success. Enumerate all six cancellation callers and test subsequent failure, interrupted delivery, retry, physical release and a fresh press. Failed resize preserves geometry but cannot restore a canceled gesture.

---

## Re-review — 2026-09-15T13:23:31-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | 000255-lifecycle-state-ownership |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 29101ebf157ba9663609f5e75278449f34eea722..4bce610f2a47af95faa3b91214db5cf3217e7f1e |
| command | sdlc milestone-close --issue 255 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-09-15T13:23:31-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-9 is addressed: cancellation revokes gesture ownership independently of resize success, and interrupted delivery is tracked without duplicate releases. The committed regressions fail when revocation is removed. M2’s focused tests, race checks, backend suite, and independent renderer checks pass. One Important artifact-lifecycle gap remains in the newly added discovery driver.

```findings
dispose:
  - id: BR-9
    disposition: addressed
    note: |
      presenter.go:205 commits CancelMouse before delivery and tracks its pending acknowledgment. Tests at presenter_test.go:608,638,661,727,760 cover failed resize, interrupted delivery, all six callers, child-write failure, and cancellation during selection. Removing ownership revocation in a temporary overlay makes the resize regression and all six caller cases fail.
  - id: BR-6
    disposition: addressed
    note: |
      Existing disposition retained: explicit gesture ownership and negotiation epochs remain enforced; chrome, panel, orphan-event, and mode-change regressions pass.
  - id: BR-7
    disposition: addressed
    note: |
      Existing disposition retained: CSI/DCS dispatch rejects retained overflow evidence; parameter boundary, numeric overflow, split-input, and recovery regressions pass.
  - id: BR-8
    disposition: addressed
    note: |
      Existing disposition retained: Endpoint captures authoritative backend cursor state; reset, restore, and buffer-switch regressions pass.
findings:
  - id: new
    severity: Important
    family: artifact-lifetime-ownership
    title: |
      Native discovery runs retain temporary artifacts without cleanup or a bound
    detail: |
      tests/terminal-oracle/discovery/zellij_oracle.py:5 creates a new /tmp/pw* directory for every invocation, while its finally block at lines 34–46 only stops processes and closes handles. All six discovery probes share this driver, and README.md:24–26 explicitly retains the directories without defining removal or a retention bound. ARCH-FUNERAL: make the driver remove its directory after teardown, including failure paths; any retained diagnostic mode needs an explicit bounded lifecycle. Cover successful and failed runs with cleanup regression tests.
```

## 1. Strengths

- **Cancellation is now a shared rule:** ownership revocation precedes delivery, and retries wait for the existing release.
- **Presentation controls admission:** blocked and partial writes are exercised through stateful transport doubles.
- **Independent rendering verification passes:** xterm-headless checks actual wire output, including wide-cell replacements and styles.
- **Qualification remains honest:** reproduced **84 pass, 0 fail, 6 not-covered**; consumer and live obligations remain pending.
- README and atlas document the shared library, profile, fork, and verification command.

## 2. Critical findings

None.

## 3. Important findings

**Discovery artifact cleanup:** [zellij_oracle.py:5](/Users/xianxu/workspace/worktree/pair/000255-lifecycle-state-ownership/tests/terminal-oracle/discovery/zellij_oracle.py:5).

Scope temporary storage to the driver’s lifetime and remove it after process teardown. Preserve diagnostic evidence only under a documented retention policy. This is a small shared-driver correction covering all six probes.

## 4. Minor findings

None.

## 5. Test coverage notes

Passed:

- Terminal, ttyio, qualification, and probe tests.
- Terminal/ttyio/qualification race suites.
- Local VT fork normal and race suites.
- Required independent renderer oracle.
- Pinned-range `git diff --check`.

The BR-9 mutation produced actual assertion failures, including revived gestures and duplicate releases. Repository files were unchanged.

The full root suite, native discovery scripts, and sustained live workflows were not rerun.

## 6. Architectural notes

| Marker | Result | Assessment |
|---|---|---|
| ARCH-DRY | Pass | Shared cancellation helper; authoritative cursor snapshots; terminfo checked against its capability source. |
| ARCH-PURE | Pass | Frame, composition, rendering, and View logic have direct tests; IO integrations are separate. |
| ARCH-PURPOSE | Pass for M2 | BR-9 covers the six-caller family; both consumer migrations remain explicit M3 obligations. |
| ARCH-MOCK | Pass for M2 | Stateful transports share production seams; disposable PTY and independent interpreter tests supplement them. |
| ARCH-CONSTRAINTS | Pass for M2 | Parser, geometry, history, and queue bounds are exercised; sustained performance acceptance remains M4. |
| ARCH-SECURE | Pass | Overflow rejection and frame/effect validation protect protocol boundaries. |
| ARCH-ORDER | Pass | View mutations use the transition owner; cancellation and pending delivery remain distinct. |
| ARCH-FUNERAL | **Flag** | New discovery directories lack removal or bounded retention. |

The M2 core-concept implementations exist at the mapped locations. No additional plan/code classification contradiction was identified.

## 7. Plan revision recommendations

Add a `## Revisions` entry defining discovery-artifact ownership: the driver creates temporary storage, teardown removes it, and retained failure evidence—if supported—has an explicit bound. The existing cancellation revision accurately describes the BR-9 correction.

---

## Re-review — 2026-09-15T13:31:32-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | 000255-lifecycle-state-ownership |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 29101ebf157ba9663609f5e75278449f34eea722..bc0d1bcbf41deaea5e134df8caa325ebbec11dde |
| command | sdlc milestone-close --issue 255 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-09-15T13:31:32-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-10 is addressed with verified cleanup regressions. The shared M2 implementation passes focused normal/race tests and the independent renderer oracle. One newly reproduced correctness bug blocks close: an interrupted paint can leave the parent terminal’s autowrap disabled even after `Release` succeeds. Repository files were unchanged.

```findings
dispose:
  - id: BR-10
    disposition: addressed
    note: |
      zellij_oracle.py:34 scopes setup, execution and teardown inside TemporaryDirectory. All five discovery tests pass; replacing cleanup with a no-op in memory makes all four cleanup tests fail on leaked directories.
  - id: BR-6
    disposition: addressed
    note: |
      Explicit parent/child gesture ownership remains enforced; presenter_test.go:507 exercises chrome, panel and orphan gestures. Focused normal/race suites pass.
  - id: BR-7
    disposition: addressed
    note: |
      parameterGuard preserves overflow evidence and rejects dispatch; pair_parameter_test.go:13 covers atomic rejection. Fork normal/race suites pass.
  - id: BR-8
    disposition: addressed
    note: |
      Endpoint captures the backend's authoritative cursor; endpoint_test.go:294 covers reset, restore and buffer transitions. Focused normal/race suites pass.
  - id: BR-9
    disposition: addressed
    note: |
      Cancellation revokes ownership before delivery and tracks pending release. presenter_test.go:608, :638 and :661 cover failed resize, interrupted delivery and cancellation callers.
findings:
  - id: new
    severity: Critical
    family: parent-terminal-restoration
    title: |
      Successful release leaves autowrap disabled after an interrupted paint
    detail: |
      ARCH-ORDER: render.go:34 emits CSI ?7l, but presenter.go:127 omits CSI ?7h from release cleanup. A production-presenter test injecting failure immediately after ?7l, followed by successful Release, reproduces ABCDEFGI on one eight-column row instead of ABCDEFGH followed by I in the independent xterm oracle. Enumerate all parent state changed during painting and restore the required post-release state after any accepted prefix; add interrupted-paint cleanup regressions, including hyperlink state.
```

## 1. Strengths

- BR-10’s directory ownership covers setup and teardown failures; mutation testing establishes causal regression coverage.
- Pure View transitions enforce identity, generation and geometry admission separately from IO.
- Stateful transport doubles preserve accepted bytes and support controlled partial writes and cancellation.
- README and atlas document the new library, profile, fork and independent oracle, while preserving the M3/M4 acceptance boundaries.

## 2. Critical findings

**Restore parent state after interrupted painting.**
[Presenter cleanup](/Users/xianxu/workspace/worktree/pair/000255-lifecycle-state-ownership/cmd/internal/terminal/presenter.go:127) does not undo autowrap suppression from [Render](/Users/xianxu/workspace/worktree/pair/000255-lifecycle-state-ownership/cmd/internal/terminal/render.go:34).

The temporary overlay regression exercised actual `Select`, partial-write failure and `Release`, then interpreted their accepted bytes with xterm-headless. Subsequent ordinary text overwrote the last column. Add autowrap restoration and audit the complete renderer state inventory, including OSC 8 hyperlinks. Test interruption at relevant byte boundaries.

## 3. Important findings

None additional.

## 4. Minor findings

None.

## 5. Test coverage notes

- Passed: terminal, ttyio and terminalqualify normal/race suites; fork normal/race suites; both independent renderer tests; changed wrapper ZWJ regression.
- Passed: five discovery tests. Cleanup mutation produced four expected assertion failures.
- Qualification: **84 pass, 0 fail, 6 not-covered**, with intentional exit 1.
- Pinned-range `git diff --check` passed.
- Broader wrapper tests encountered sandbox-denied storage protection in existing argument-error tests; that suite was not fully validated.
- Existing release tests miss the reproduced mid-render state leak.

## 6. Architectural notes

| Principle | Result |
|---|---|
| ARCH-DRY | Pass: shared terminal ownership and capability-table-derived terminfo. |
| ARCH-PURE | Pass: frame/render/View logic has direct tests; backend and transport remain integrations. |
| ARCH-PURPOSE | Pass for M2: consumer migration and native-history obligations remain explicit M3 work. |
| ARCH-MOCK | Pass for M2: injected stateful transport, real-PTY checks and independent interpretation; sustained native conformance remains M4. |
| ARCH-CONSTRAINTS | Pass: geometry, parser, queue and history bounds are explicit; performance acceptance remains provisional. |
| ARCH-SECURE | Pass: typed effects and validated drawing data prevent raw child-control passthrough. |
| ARCH-ORDER | **Flag:** successful release does not restore parent state after partial painting. |
| ARCH-FUNERAL | Pass: BR-10 cleanup is causally verified; transport workers have joined teardown. |

The M2 core-concept table matches the implementation boundaries; no additional classification contradiction was identified.

## 7. Plan revision recommendations

Add a `## Revisions` entry defining **parent state restoration after interrupted writes**: enumerate renderer/setup state, specify the required release state, and require independent-oracle tests across partial-write boundaries. Preserve conditional keyboard-stack restoration rather than blindly popping unowned state.

---

## Re-review — 2026-09-15T13:42:44-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | 000255-lifecycle-state-ownership |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 29101ebf157ba9663609f5e75278449f34eea722..785cd2a3abae476877914c0403e9f6a607ce0f8b |
| command | sdlc milestone-close --issue 255 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-09-15T13:42:44-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

BR-11 is addressed with verified failing-without-fix tests. The pinned M2 range provides the shared ownership boundaries, bounded transports, and documentation, but a new Unicode defect blocks the boundary: valid zero-width characters can permanently fail the presenter. Repository files were unchanged.

```findings
dispose:
  - id: BR-6
    disposition: addressed
    note: |
      Parent/orphan gesture and negotiation-epoch regressions remain present and pass in the focused normal/race suites.
  - id: BR-7
    disposition: addressed
    note: |
      Atomic parameter overflow guards and boundary regressions remain present; fork normal/race suites pass.
  - id: BR-8
    disposition: addressed
    note: |
      Endpoint and qualification snapshots consume the authoritative copied cursor; reset/restore/buffer regressions pass.
  - id: BR-9
    disposition: addressed
    note: |
      Cancellation commits independently of resize and retains pending delivery; failed-resize and interrupted-cancellation regressions pass.
  - id: BR-10
    disposition: addressed
    note: |
      Native discovery uses invocation-scoped temporary storage; all five cleanup and bounded-capture tests pass.
  - id: BR-11
    disposition: addressed
    note: |
      presenter.go:244 restores the parent baseline. The independent interrupted-presentation test passes; scratch mutations removing autowrap restoration or hyperlink closure both make it fail.
findings:
  - id: new
    severity: Critical
    family: unicode-cell-coherence
    title: |
      Zero-width Unicode output permanently fails the presenter
    detail: |
      third_party/vt/utf8.go:49-51 stores an initial zero-width grapheme as a nonempty Width:0 cell, which cmd/internal/terminal/frame.go:97-100 rejects. Production Feed → Present → Flush reproduces Failed state for U+0301, U+200D, U+FE0F, and a combining mark following SGR or cursor movement. ARCH-PURPOSE: define coherent zero-width rendering across backend, frame validation, and serialization; cover the entire class with split-input and production-presentation regressions rather than weakening frame validation.
```

## 1. Strengths

- BR-11 has causal regression evidence: removing `?7h` reproduces `ABCDEFGI`; removing hyperlink closure exposes the leaked link.
- Presenter admission passes through `Transition`, with forced-order tests covering partial writes, cancellation, and release.
- Terminfo and query capabilities share one table, with compiled-contract verification.
- README and atlas document the new library, fork, profile, and M3 migration boundary.

## 2. Critical findings

**Zero-width Unicode can disable the connection.** At [utf8.go:49](third_party/vt/utf8.go#L49), a grapheme without an extendable predecessor can become a nonempty zero-width cell. [Frame.Validate](cmd/internal/terminal/frame.go#L97) rejects it; refresh reaches `p.fail` through [presenter.go:327](cmd/internal/terminal/presenter.go#L327), leaving `Failed` state and rejecting subsequent input.

Reproduction: select an 8×2 endpoint, feed any input below, then call `Present` and `Flush`:

- `"\u0301"`
- `"\u200d"`
- `"\ufe0f"`
- `"A\x1b[31m\u0301"`
- `"A\x1b[2G\u0301"`

All five scratch regressions failed with `content in continuation` and presenter state `Failed`. Independent xterm interpretation accepts these inputs.

**Fix:** define valid standalone/after-control zero-width behavior, preserve frame invariants, and test continued presentation and input afterward.

## 3. Important findings

None.

## 4. Minor findings

None.

## 5. Test coverage notes

Passed:

- Terminal, ttyio, and qualification normal/race suites.
- Fork normal/race suites.
- Required independent renderer oracle.
- Wrapcmd and ptychild suites.
- Five discovery cleanup tests and resource probe.
- Pinned-range whitespace check.

Qualification reports **84 pass, 0 fail, 6 not-covered**, retaining M3/M4 obligations. The full root suite was interrupted before completion; no full-suite pass is claimed.

Existing grapheme tests cover extensions of printable bases but miss the zero-width cases above.

## 6. Architectural notes

| Marker | Assessment |
|---|---|
| ARCH-DRY | Pass: shared capability source and authoritative cursor reads. |
| ARCH-PURE | Pass: M2 frame/view/render logic is separated from transport and backend IO. |
| ARCH-PURPOSE | **Flag:** supported Unicode can terminate presentation. |
| ARCH-MOCK | Pass for M2: stateful transport seam, independent interpreter, native checks. |
| ARCH-CONSTRAINTS | Pass: explicit bounds, backpressure, deadlines, and resource measurements. |
| ARCH-SECURE | Pass: controls are validated; raw child drawing cannot bypass composition. |
| ARCH-ORDER | Pass: admission and cancellation use explicit transitions and controlled-order tests. |
| ARCH-FUNERAL | Pass: joined workers, bounded retention, origin retirement, and temporary-directory cleanup. |

The M2 concept table matches the implementation. Both consumer migrations and sustained live acceptance remain explicitly assigned to M3/M4.

## 7. Plan revision recommendations

Add a `## Revisions` entry defining the zero-width cell invariant and enumerating standalone marks, joiners, variation selectors, and marks after controls. Require every-split backend tests plus production presenter regressions demonstrating valid frames and continued admission.

---

## Re-review — 2026-09-15T13:57:53-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | 000255-lifecycle-state-ownership |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | milestone M2 |
| milestone | M2 |
| window | 29101ebf157ba9663609f5e75278449f34eea722..214d43e87969cf68d8cb615288318e55fc6af8f5 |
| command | sdlc milestone-close --issue 255 --milestone M2 |
| reviewer | codex |
| timestamp | 2026-09-15T13:57:53-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: medium
```

The pinned range satisfies the M2 shared-library boundary. BR-12 is addressed with meaningful regression evidence, and no new blocking findings emerged. Consumer migration and live acceptance remain explicitly pending for M3/M4. Confidence is qualified by one timing failure in the full suite; that unchanged test passed ten isolated reruns.

```findings
dispose:
  - id: BR-12
    disposition: addressed
    note: |
      Backend orphan handling, strict grapheme validation, and StyledRows now agree. Production presentation/input and split-input regressions pass at HEAD and fail with the three pre-fix implementation files restored through a temporary Go overlay.
  - id: BR-6
    disposition: addressed
    note: |
      Prior disposition retained; parent/orphan gesture and negotiation-epoch regressions pass.
  - id: BR-7
    disposition: addressed
    note: |
      Prior disposition retained; atomic parameter-overflow handling and fork regressions pass.
  - id: BR-8
    disposition: addressed
    note: |
      Prior disposition retained; snapshots consume authoritative cursor state and cursor qualification cases pass.
  - id: BR-9
    disposition: addressed
    note: |
      Prior disposition retained; failed-resize and interrupted-cancellation regressions pass.
  - id: BR-10
    disposition: addressed
    note: |
      Prior disposition retained; scoped temporary-directory cleanup and five native-driver tests pass.
  - id: BR-11
    disposition: addressed
    note: |
      Prior disposition retained; independent interrupted-presentation restoration tests pass.
```

### 1. Strengths

- BR-12 covers the class: orphan Unicode scalars, control boundaries, byte splits, pending wrap, and continued presentation/input (`zero_width_test.go:14`, `pair_zero_width_test.go:33`).
- Presenter admission depends on completed output; controlled partial-write and cancellation tests exercise the production seam (`presenter_test.go:29`).
- Terminfo derives from the capability table, with compiled-contract verification (`profile_query_test.go:51`).
- README and atlas explain the new surface and distinguish M2 completion from consumer/live acceptance.

### 2. Critical findings

None.

### 3. Important findings

None.

### 4. Minor findings

None.

### 5. Test coverage notes

Passed:

- Focused terminal, transport, qualification, and wrapper tests.
- Focused race tests; fork normal and race suites.
- Independent xterm renderer tests.
- Native-driver cleanup tests.
- BR-12 negative-control overlay: regressions fail without the fix.
- Qualification: **84 pass, six not-covered**, matching M3/M4 obligations.
- Representative benchmarks and pinned-range whitespace check.

Full `go test ./...` failed only at `TestParkCoordinatorConstructorDoesNotQueryPairSession`, which exceeded its 100ms deadline (`cmd/internal/couchcore/park_test.go:744`). The package is unchanged in this range; ten isolated repetitions passed. The full run therefore cannot be reported as green.

### 6. Architectural notes for upcoming work

| Principle | Result |
|---|---|
| ARCH-DRY | Pass: shared protocol ownership and mechanically checked capability source. |
| ARCH-PURE | Pass: direct pure-core tests; backend, decoder, and transport classified as integration. |
| ARCH-PURPOSE | Pass for M2: consumer migration remains explicit M3 work. |
| ARCH-MOCK | Pass for M2: stateful transport doubles and independent renderer interpretation. |
| ARCH-CONSTRAINTS | Pass: bounded admission/retention and representative measurements; sustained targets remain M4. |
| ARCH-SECURE | Pass: validated frames, constrained control serialization, origin-bound effects. |
| ARCH-ORDER | Pass: private presenter state transitions, controlled ordering, partial-write and cancellation coverage. |
| ARCH-FUNERAL | Pass: joined transport teardown, origin retirement, bounded history, scoped discovery artifacts. |

The M2 concepts table matches the implementation. Preserve the documented M3 shadow sweep across both consumers and the M4 native/live acceptance requirements.

### 7. Plan revision recommendations

None.
