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

## Open findings

- **BR-1** [Minor] `budget-owner-untested-branch` Release's drag-cancellation WriteTimeout bound has no regression test
- **BR-2** [Minor] `struct-literal-style` WriteFailure is built positionally in transport.go and keyed in presenter.go
