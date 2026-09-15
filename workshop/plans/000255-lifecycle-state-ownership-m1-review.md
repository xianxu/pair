# Boundary Review — 000255-lifecycle-state-ownership#255 (milestone M1)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | 000255-lifecycle-state-ownership |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | b11ab67ff1d290386cf12177f2dacdccbc8551c0..8343fc2debe04511a8a63a24c9cee0e94fb1308d |
| command | sdlc milestone-close --issue 255 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-15T10:17:29-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

M1 correctly isolates qualification from production and reproducibly rejects unchanged backend adoption. Focused normal/race tests pass. However, the harness does not enforce its promised whole/split equivalence, omits rendering attributes, and emits less evidence than the plan requires. These gaps need correction before closing M1.

## 1. Strengths

- `Report.Validate` rejects missing, duplicate, unexpected, and invalid results; unmet obligations cannot qualify.
- Candidate tests exercise blocked replies, cancellation, isolation, and joined teardown through controlled IO.
- The probe reproduces **41 pass / 15 fail / 14 not-covered**, with exit 1.
- The wrapper audit matches `wrap.go`: query tracking and terminal observation consume raw bytes, while parent output receives transformed bytes. Production migration remains explicitly deferred.

## 2. Critical findings

**Whole/split equivalence is never checked — ARCH-PURPOSE.**  
`cmd/internal/terminalqualify/runner.go:43` compares each delivery variant only against the fixture’s sparse expectations. It never retains the whole-input observation or compares subsequent observations against it.

For example, the ASCII fixture can still pass if split delivery corrupts `cell:7,3`, changes its color, or emits an unexpected reply: none is in that fixture’s expected subset. This contradicts the completed Task 2 promise that any changed cell/cursor/style/link is detected.

**Fix:** retain the whole-input observation and compare complete observations across variants, alongside the independent literal oracle. Add regressions changing only an otherwise-unasserted cell, style, link, or reply, plus tests verifying the actual partitions.

## 3. Important findings

- **Rendering attributes are unobservable — ARCH-PURPOSE.**  
  `cmd/internal/terminalqualify/candidate.go:204` copies foreground/background colors but drops `Style.Attrs`, `Underline`, and `UnderlineColor`. The matrix neither exercises these attributes nor lists them as uncovered. Bold, underline, and inverse are distinct rendering semantics in the [xterm protocol](https://invisible-island.net/xterm/ctlseqs/ctlseqs.html). Extend snapshots and literal set/reset/preservation fixtures; test that dropping these fields fails.

- **JSON lacks promised expected/observed evidence — ARCH-PURPOSE.**  
  `cmd/internal/terminalqualify/report.go:21` stores status and detail only. Passing results contain a delivery-count sentence; failures contain only the first mismatch. Plan line 135 explicitly promises expected/observed results. Add bounded structured evidence identifying what was checked and observed, with JSON regression tests.

- **README update appears missing for the qualification probe.**  
  `cmd/probes/terminalqualify/main.go:41` introduces a runnable command with meaningful exit statuses. Atlas documents it, but README is unchanged in the pinned range. Add invocation, output, exit-status meanings, and the qualification-report link.

## 4. Minor findings

None.

## 5. Test coverage notes

- Passed: focused unit tests, focused race tests, and pinned-range `git diff --check`.
- Reproduced the documented negative probe result.
- Full `go test ./...` was interrupted after it stopped producing progress; full-suite success was **not independently established**.
- Existing tests miss the observation-completeness and report-evidence gaps above.
- No tracked files were edited. The full-suite attempt created an untracked `.nvimlog`.

## 6. Architectural notes

| Marker | Assessment |
|---|---|
| ARCH-DRY | **Pass:** shared runner/comparator; no competing production parser introduced. |
| ARCH-PURE | **Pass:** comparison/report logic is IO-free; emulator access is isolated. |
| ARCH-PURPOSE | **Flag:** qualification promises exceed the observations and evidence enforced. |
| ARCH-MOCK | **Pass for M1:** controlled transport doubles exercise the candidate seam; live/composed obligations remain explicitly uncovered. |
| ARCH-CONSTRAINTS | **Pass for fixed fixtures:** input, cells, replies, history, and execution have explicit limits; measurements are not presented as production budgets. |
| ARCH-SECURE | **Pass for changed scope:** synthetic inputs, validated report structure, no ambient credential or production-state dependency. |
| ARCH-ORDER | **Pass for diagnostic lifecycle:** serialized execution, private snapshots, cancellation, and joined teardown have focused tests. |
| ARCH-FUNERAL | **Pass:** disposable candidates have cleanup; no recurring production artifact family is introduced. |

The M1 entities exist at their documented paths. The broader Core concepts table describes proposed M2–M3 entities, rather than delivered M1 components; their absence is consistent with the explicitly bounded milestone.

## 7. Plan revision recommendations

Add `## Revisions` entries that:

- Correct the premature Task 2 completion claim and name complete observation equivalence plus its regression strategy.
- Enumerate rendering attributes and their executable qualification coverage.
- Specify the bounded expected/observed report schema and refresh the measured report after corrections.

```findings
findings:
  - id: new
    severity: Critical
    family: qualification-observation-equivalence
    title: |
      Split qualification never compares complete observations
    detail: |
      cmd/internal/terminalqualify/runner.go:43 checks only sparse literal expectations, so split-only corruption outside those keys passes. ARCH-PURPOSE: retain the whole-input observation, compare complete observations across variants, and add regressions for otherwise-unasserted cells, styles, links, and replies.
  - id: new
    severity: Important
    family: required-capability-coverage
    title: |
      Non-color rendering attributes are neither observed nor qualified
    detail: |
      cmd/internal/terminalqualify/candidate.go:204 omits Style.Attrs, Underline, and UnderlineColor; fixtures and Coverage contain no corresponding obligations. ARCH-PURPOSE: capture these fields and add literal set/reset/preservation cases with regression evidence.
  - id: new
    severity: Important
    family: qualification-evidence-contract
    title: |
      JSON results omit the promised expected and observed evidence
    detail: |
      cmd/internal/terminalqualify/report.go:21 emits status/detail without the expected/observed results promised at plan line 135. ARCH-PURPOSE: add bounded structured evidence for executable cases and verify it through JSON tests.
  - id: new
    severity: Important
    family: readme-surface-documentation
    title: |
      README update appears missing for the terminal qualification probe
    detail: |
      cmd/probes/terminalqualify/main.go:41 adds a runnable diagnostic with distinct exit statuses, but README.md is unchanged in the pinned range. Document invocation, JSON output, exit meanings, and the qualification report.
```
