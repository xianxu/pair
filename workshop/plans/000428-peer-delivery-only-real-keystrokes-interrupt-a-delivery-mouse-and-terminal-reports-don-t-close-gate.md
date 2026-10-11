---
gate: boundary-review
issue: 428
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-10T18:15:09-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: 'TestPeerDraftPasteWaitsForDelivery is flaky (about 1 in 20): it asserts !holding while the trailing focus-out chunk is still queued'
          detail: peer_input_test.go:250-256 reads d.holding right after the second CR shows up; the draft's trailing focus-out can still be waiting on its replay gap. Wait on !holding with waitFor instead (or on the focus-out suffix). Reproduced with go test -count=20.
          family: test-observes-single-interleaving
          round: 1
        - id: BR-2
          severity: Important
          title: atlas/couch.md says typed keystrokes are never held, but holdLocked queues every chunk after a held paste
          detail: 'peer_delivery.go holdLocked returns true for all input while holding. Reword: a keystroke never starts a hold, and later input queues behind a held paste to keep order.'
          family: docs-contradict-behavior
          round: 1
        - id: BR-3
          severity: Minor
          title: dispatchPeer does not check d.holding before pasting the next message
          detail: A held draft paste is not admitted until release, so it never touches lastInput. Today only the settle window and the loop order keep the next delivery from pasting ahead of the released draft. An explicit !d.holding paste gate would make this structural (ARCH-ORDER).
          family: hold-state-not-gating-dispatch
          round: 1
        - id: BR-4
          severity: Minor
          title: composerInView runs zellij dump-screen with no fake or conformance check of its scrolled-viewport semantics
          detail: The viewProbe seam exists and the pure part is tested, but the zellij 0.45 behaviour the design depends on is only recorded in the Log (ARCH-MOCK).
          family: external-binary-without-fake
          round: 1
        - id: BR-5
          severity: Minor
          title: Each viewport reading renders the full scrollback under the model lock and scans every shift
          detail: Capped at one reading per 500ms while a delivery waits; no measurement recorded for a long scrollback (ARCH-CONSTRAINTS).
          family: unmeasured-hot-path-cost
          round: 1
      recipe: milestone-review
      blocked: true
---

# Gate ledger — pair#428 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-10T18:15:09-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `test-observes-single-interleaving` TestPeerDraftPasteWaitsForDelivery is flaky (about 1 in 20): it asserts !holding while the trailing focus-out chunk is still queued
  peer_input_test.go:250-256 reads d.holding right after the second CR shows up; the draft's trailing focus-out can still be waiting on its replay gap. Wait on !holding with waitFor instead (or on the focus-out suffix). Reproduced with go test -count=20.
- **BR-2** [Important] `docs-contradict-behavior` atlas/couch.md says typed keystrokes are never held, but holdLocked queues every chunk after a held paste
  peer_delivery.go holdLocked returns true for all input while holding. Reword: a keystroke never starts a hold, and later input queues behind a held paste to keep order.
- **BR-3** [Minor] `hold-state-not-gating-dispatch` dispatchPeer does not check d.holding before pasting the next message
  A held draft paste is not admitted until release, so it never touches lastInput. Today only the settle window and the loop order keep the next delivery from pasting ahead of the released draft. An explicit !d.holding paste gate would make this structural (ARCH-ORDER).
- **BR-4** [Minor] `external-binary-without-fake` composerInView runs zellij dump-screen with no fake or conformance check of its scrolled-viewport semantics
  The viewProbe seam exists and the pure part is tested, but the zellij 0.45 behaviour the design depends on is only recorded in the Log (ARCH-MOCK).
- **BR-5** [Minor] `unmeasured-hot-path-cost` Each viewport reading renders the full scrollback under the model lock and scans every shift
  Capped at one reading per 500ms while a delivery waits; no measurement recorded for a long scrollback (ARCH-CONSTRAINTS).

## Open findings

- **BR-1** [Important] `test-observes-single-interleaving` TestPeerDraftPasteWaitsForDelivery is flaky (about 1 in 20): it asserts !holding while the trailing focus-out chunk is still queued
- **BR-2** [Important] `docs-contradict-behavior` atlas/couch.md says typed keystrokes are never held, but holdLocked queues every chunk after a held paste
- **BR-3** [Minor] `hold-state-not-gating-dispatch` dispatchPeer does not check d.holding before pasting the next message
- **BR-4** [Minor] `external-binary-without-fake` composerInView runs zellij dump-screen with no fake or conformance check of its scrolled-viewport semantics
- **BR-5** [Minor] `unmeasured-hot-path-cost` Each viewport reading renders the full scrollback under the model lock and scans every shift
