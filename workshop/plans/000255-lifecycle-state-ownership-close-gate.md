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

## Open findings

- **BR-1** [Critical] `qualification-observation-equivalence` Split qualification never compares complete observations
- **BR-2** [Important] `required-capability-coverage` Non-color rendering attributes are neither observed nor qualified
- **BR-3** [Important] `qualification-evidence-contract` JSON results omit the promised expected and observed evidence
- **BR-4** [Important] `readme-surface-documentation` README update appears missing for the terminal qualification probe
