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
