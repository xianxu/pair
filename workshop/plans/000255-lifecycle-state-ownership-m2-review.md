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
