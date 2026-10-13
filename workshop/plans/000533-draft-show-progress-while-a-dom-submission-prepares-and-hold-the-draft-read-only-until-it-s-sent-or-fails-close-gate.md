---
gate: boundary-review
issue: 533
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-13T14:47:41-07:00"
      dispatched: "2026-10-13T14:41:38-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Critical
          title: Read-only DOM preparation permits partial navigation that overwrites queued text
          detail: 'nvim/init.lua:1666 applies nomodifiable without guarding go_to, which changes nav.pos at line 2527 before its buffer write fails. Reproduced: hold preparation with draft="draft" and queue="queued", press Alt+Right, return preparation failure, then press Alt+Left; autosave_current_slot at line 2439 overwrites the queue with "draft". ARCH-ORDER and ARCH-PURPOSE: guard owned-draft actions before state or file mutations, sweep navigation/queue/delete/reset/send-clear siblings, and add production regression tests preserving slot identity and stored contents through failure and timeout.'
          family: operation-ownership-must-guard-mutators
          round: 1
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 2
      timestamp: "2026-10-13T14:58:42-07:00"
      dispatched: "2026-10-13T14:54:05-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: not-addressed
          note: 'The navigation correction passes production regressions; disabling its guard makes the Alt+Right regression fail. The required mutator sweep remains incomplete: nvim/init.lua:343 still writes from the scheduled FocusLost handler while DOM ownership sets nomodifiable. An embedded-UI reproduction enters insert mode at EOL during held preparation, fires FocusLost, and reaches “Buffer is not modifiable” followed by a blocking “Press ENTER” prompt. Cover focus padding/removal and other asynchronous mutators under the same ownership rule, with production event-sequence regressions (ARCH-ORDER, ARCH-PURPOSE).'
          round: 2
      recipe: milestone-review
      isolation: fenced
      blocked: true
    - "n": 3
      timestamp: "2026-10-13T15:10:03-07:00"
      dispatched: "2026-10-13T15:04:08-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: Entry guards protect navigation and draft mutations; focus padding now respects ownership and retains deferred cleanup. Production draft/queue failure/timeout cases and embedded focus tests pass. Removing navigation guards makes dom_submission_integration_test.lua:72 fail; removing the focus guard at init.lua:338 reproduces nomodifiable, Press ENTER, and RPC timeout in dom-submission-ui-test.lua:129.
          round: 3
      recipe: milestone-review
      isolation: fenced
      reviewed: 16f6b7fd9ee457f42a7fb1a56aeb158a97e4cd9f
      blocked: false
---

# Gate ledger — pair#533 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-13T14:47:41-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Critical] `operation-ownership-must-guard-mutators` Read-only DOM preparation permits partial navigation that overwrites queued text
  nvim/init.lua:1666 applies nomodifiable without guarding go_to, which changes nav.pos at line 2527 before its buffer write fails. Reproduced: hold preparation with draft="draft" and queue="queued", press Alt+Right, return preparation failure, then press Alt+Left; autosave_current_slot at line 2439 overwrites the queue with "draft". ARCH-ORDER and ARCH-PURPOSE: guard owned-draft actions before state or file mutations, sweep navigation/queue/delete/reset/send-clear siblings, and add production regression tests preserving slot identity and stored contents through failure and timeout.

## Round 2 — 2026-10-13T14:58:42-07:00 (codex) — BLOCKED

### Disposed

- BR-1 — not-addressed — The navigation correction passes production regressions; disabling its guard makes the Alt+Right regression fail. The required mutator sweep remains incomplete: nvim/init.lua:343 still writes from the scheduled FocusLost handler while DOM ownership sets nomodifiable. An embedded-UI reproduction enters insert mode at EOL during held preparation, fires FocusLost, and reaches “Buffer is not modifiable” followed by a blocking “Press ENTER” prompt. Cover focus padding/removal and other asynchronous mutators under the same ownership rule, with production event-sequence regressions (ARCH-ORDER, ARCH-PURPOSE).

## Round 3 — 2026-10-13T15:10:03-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — Entry guards protect navigation and draft mutations; focus padding now respects ownership and retains deferred cleanup. Production draft/queue failure/timeout cases and embedded focus tests pass. Removing navigation guards makes dom_submission_integration_test.lua:72 fail; removing the focus guard at init.lua:338 reproduces nomodifiable, Press ENTER, and RPC timeout in dom-submission-ui-test.lua:129.

## Open findings

(none — every finding has been disposed)
