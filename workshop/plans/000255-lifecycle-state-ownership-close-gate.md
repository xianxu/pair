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

## Open findings

- **BR-5** [Important] `qualification-observation-equivalence` Promised partition-coverage regression tests do not verify delivered bytes
