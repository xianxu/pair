---
gate: boundary-review
issue: 404
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-10-07T11:57:01-07:00"
      agent: codex
      findings:
        - id: BR-1
          severity: Important
          title: README omits the new live-capture configuration
          detail: cmd/internal/couchcmd/capture.go:40 introduces COUCH_CAPTURE_DIR and COUCH_CAPTURE_MAX_MIB, but README.md is unchanged in the pinned range. Document activation, limits and stopped-state behavior there, linking to the atlas runbook.
          family: user-surface-readme-coverage
          round: 1
        - id: BR-2
          severity: Important
          title: Per-session caps leave accumulated capture storage unbounded
          detail: 'cmd/internal/terminalcapture/recorder.go:106 creates a fresh session directory on every enabled launch, and atlas/couch-live-capture.md:90 explicitly requires operator deletion. ARCH-FUNERAL requires more than manual cleanup for per-session growth: enforce an aggregate admission budget or automatic retention with incident-evidence protection, and test repeated launches.'
          family: durable-artifact-aggregate-retention
          round: 1
        - id: BR-3
          severity: Important
          title: Recorder lifecycle transitions remain embedded in the IO shell
          detail: cmd/internal/terminalcapture/recorder.go:177 changes phases while closing channels, and :299 independently changes phase from the writer loop. Under ARCH-ORDER and ARCH-PURE, route authoritative state changes through an encapsulated pure state/event transition component; exercise failure, repeated close, timeout and late worker completion through that production model.
          family: authoritative-pure-lifecycle-transitions
          round: 1
      recipe: milestone-review
      blocked: true
    - "n": 2
      timestamp: "2026-10-07T12:12:27-07:00"
      agent: codex
      dispose:
        - id: BR-1
          disposition: addressed
          note: README.md:1211–1232 documents activation, configuration limits, stopped-state badges, aggregate admission and the atlas runbook. These match captureSettings, captureBadge and storage admission.
          round: 2
        - id: BR-2
          disposition: addressed
          note: Recorder.Open calls budgeted storage admission. Reservations survive completed sessions; byte/count limits and concurrent admission preserve existing evidence. Tests cover repeated launches, exhaustion, legacy/corrupt metadata and public Open wiring.
          round: 2
        - id: BR-3
          disposition: addressed
          note: lifecycle.go owns authoritative phase/error transitions; Recorder executes its effects. Production-model sequence tests cover repeated close, failure, timeout, completion-before-timeout and late completion, supplemented by blocked-writer integration tests.
          round: 2
      recipe: milestone-review
      blocked: false
---

# Gate ledger — pair#404 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-10-07T11:57:01-07:00 (codex) — BLOCKED

### Raised

- **BR-1** [Important] `user-surface-readme-coverage` README omits the new live-capture configuration
  cmd/internal/couchcmd/capture.go:40 introduces COUCH_CAPTURE_DIR and COUCH_CAPTURE_MAX_MIB, but README.md is unchanged in the pinned range. Document activation, limits and stopped-state behavior there, linking to the atlas runbook.
- **BR-2** [Important] `durable-artifact-aggregate-retention` Per-session caps leave accumulated capture storage unbounded
  cmd/internal/terminalcapture/recorder.go:106 creates a fresh session directory on every enabled launch, and atlas/couch-live-capture.md:90 explicitly requires operator deletion. ARCH-FUNERAL requires more than manual cleanup for per-session growth: enforce an aggregate admission budget or automatic retention with incident-evidence protection, and test repeated launches.
- **BR-3** [Important] `authoritative-pure-lifecycle-transitions` Recorder lifecycle transitions remain embedded in the IO shell
  cmd/internal/terminalcapture/recorder.go:177 changes phases while closing channels, and :299 independently changes phase from the writer loop. Under ARCH-ORDER and ARCH-PURE, route authoritative state changes through an encapsulated pure state/event transition component; exercise failure, repeated close, timeout and late worker completion through that production model.

## Round 2 — 2026-10-07T12:12:27-07:00 (codex) — passed

### Disposed

- BR-1 — addressed — README.md:1211–1232 documents activation, configuration limits, stopped-state badges, aggregate admission and the atlas runbook. These match captureSettings, captureBadge and storage admission.
- BR-2 — addressed — Recorder.Open calls budgeted storage admission. Reservations survive completed sessions; byte/count limits and concurrent admission preserve existing evidence. Tests cover repeated launches, exhaustion, legacy/corrupt metadata and public Open wiring.
- BR-3 — addressed — lifecycle.go owns authoritative phase/error transitions; Recorder executes its effects. Production-model sequence tests cover repeated close, failure, timeout, completion-before-timeout and late completion, supplemented by blocked-writer integration tests.

## Open findings

(none — every finding has been disposed)
