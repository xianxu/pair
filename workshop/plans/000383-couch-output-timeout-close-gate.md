---
gate: boundary-review
issue: 383
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-02T08:58:03-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Minor
          title: Release's drag-cancellation WriteTimeout bound has no regression test
          detail: presenter.go:140 wraps cancelDrag in WithTimeout(ctx, WriteTimeout), but no test runs Release with a pending drag whose child input stalls. Removing the wrap would let Release(context.Background()) from Console or termcmd hang forever, and every test would still pass. The only other budget application in this window, the reset write via write(), is covered by the stall tests.
          family: budget-owner-untested-branch
          round: 1
        - id: BR-2
          severity: Minor
          title: WriteFailure is built positionally in transport.go and keyed in presenter.go
          detail: 'transport.go:135 uses WriteFailure{"child input", accepted, len(p), err} and presenter.go:212 uses keyed fields. Op is a free-form string with no zero-value guard, so keyed literals at both sites would keep a future field reorder or omission from silently printing "terminal:  write accepted".'
          family: struct-literal-style
          round: 1
      recipe: small-diff-review
      blocked: false
    - "n": 2
      timestamp: "2026-10-02T09:01:53-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: The redundant WithTimeout wrap was removed, and TestPresenterReleaseBoundsDragCancellationWhenChildStalls guards the real owner (InputWriter); it passes when run outside the sandbox.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Both WriteFailure literals in the tree (transport.go:135, presenter.go:211) now use field names.
          round: 2
      recipe: small-diff-review
      blocked: false
---

# Gate ledger — pair#383 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-02T08:58:03-07:00 (claude) — passed

### Raised

- **BR-1** [Minor] `budget-owner-untested-branch` Release's drag-cancellation WriteTimeout bound has no regression test
  presenter.go:140 wraps cancelDrag in WithTimeout(ctx, WriteTimeout), but no test runs Release with a pending drag whose child input stalls. Removing the wrap would let Release(context.Background()) from Console or termcmd hang forever, and every test would still pass. The only other budget application in this window, the reset write via write(), is covered by the stall tests.
- **BR-2** [Minor] `struct-literal-style` WriteFailure is built positionally in transport.go and keyed in presenter.go
  transport.go:135 uses WriteFailure{"child input", accepted, len(p), err} and presenter.go:212 uses keyed fields. Op is a free-form string with no zero-value guard, so keyed literals at both sites would keep a future field reorder or omission from silently printing "terminal:  write accepted".

## Round 2 — 2026-10-02T09:01:53-07:00 (claude) — passed

### Disposed

- BR-1 — addressed — The redundant WithTimeout wrap was removed, and TestPresenterReleaseBoundsDragCancellationWhenChildStalls guards the real owner (InputWriter); it passes when run outside the sandbox.
- BR-2 — addressed — Both WriteFailure literals in the tree (transport.go:135, presenter.go:211) now use field names.

## Open findings

(none — every finding has been disposed)
