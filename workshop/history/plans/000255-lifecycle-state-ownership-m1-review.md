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

---

## Re-review — 2026-09-15T10:26:56-07:00 (REWORK)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | 000255-lifecycle-state-ownership |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | b11ab67ff1d290386cf12177f2dacdccbc8551c0..021afbe2cf803bbfaa80dfb6dced63b898785f68 |
| command | sdlc milestone-close --issue 255 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-15T10:26:56-07:00 |
| verdict | REWORK |

## Review

```verdict
verdict: REWORK
confidence: high
```

All four prior findings are addressed, including regression tests that fail when the executable fixes are removed. The probe reproduces **53 pass, 15 fail, 14 not-covered**, correctly rejecting unchanged backend adoption. One Important testing gap remains: the plan claims missed byte partitions are detected, but the complete harness suite passes when every two-part split is replaced with unsplit input.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      runner.go:53 compares complete observations in both directions. Disabling this comparison makes TestRunCaseDetectsUnassertedSplitStateChanges fail for cells, attributes, links, replies, and added keys.
  - id: BR-2
    disposition: addressed
    note: |
      candidate.go:207 captures attributes, underline style, and underline color; screen_cases.go adds 12 literal set/reset/preservation cases. Removing attribute capture makes TestCandidateCapturesCompleteStyle fail.
  - id: BR-3
    disposition: addressed
    note: |
      report.go:29 and runner.go:94 provide bounded expected/observed evidence. Removing expected JSON serialization makes TestRunJSONPreservesStructuredEvidence fail; the real probe emits evidence for every executable case.
  - id: BR-4
    disposition: addressed
    note: |
      The pinned README.md:773 addition documents invocation, JSON evidence, exit meanings, and the qualification report. These match cmd/probes/terminalqualify/main.go:16 and the reproduced negative qualification.
findings:
  - id: new
    severity: Important
    family: qualification-observation-equivalence
    title: |
      Promised partition-coverage regression tests do not verify delivered bytes
    detail: |
      cmd/internal/terminalqualify/runner_test.go:11 checks chunk counts and call counts, but never their contents; the completed plan checkbox at workshop/plans/000255-terminal-abstraction-plan.md:161 promises detection of missed splits. Replacing cases.go:20 with []string{input, ""} leaves both complete harness packages green. This is the 2nd finding in family qualification-observation-equivalence. Earlier corrections covered observation comparison; enforce the complete rule across partition generation and comparison: preserve the input bytes, enumerate every byte boundary, include byte-at-a-time delivery, and detect changed observations. Add independent partition expectations covering empty, single-byte, multibyte, and control-sequence inputs, and require this mutation to fail. ARCH-PURPOSE.
```

## 1. Strengths

- Full-observation comparison is separate from bounded presentation evidence.
- Candidate tests exercise origin isolation, blocked replies, cancellation, and joined cleanup through the actual transport seam.
- README, atlas, and qualification report consistently distinguish diagnostic success from backend suitability.
- The M1 implementation remains isolated from production consumers.

## 2. Critical findings

None.

## 3. Important findings

**Partition coverage lacks the promised regression protection** — `cmd/internal/terminalqualify/cases.go:20`, `runner_test.go:11`.

Add tests that inspect the actual partitions, independently asserting byte preservation and complete boundary coverage. Current partition generation appears correct; the demonstrated gap is in its regression protection and the completed Plan claim.

## 4. Minor findings

None.

## 5. Test coverage notes

- Passed focused race tests for `terminalqualify`, its probe, and `artifactpath`.
- Reproduced the documented probe results and exit status.
- Temporary overlays confirmed BR-1, BR-2, and BR-3 regressions fail without their fixes.
- The partition mutation unexpectedly passed both complete harness packages.
- Source/docs diff checks passed; the full range reports four Markdown trailing-space occurrences in the archived review.
- Full repository and live conformance suites were not rerun. Repository files remained unchanged.

## 6. Architectural notes

| Marker | Assessment |
|---|---|
| ARCH-DRY | **Pass:** shared runner/comparator; executable fixtures supply requirements. |
| ARCH-PURE | **Pass:** observation/report logic uses direct tests; candidate IO remains separate. |
| ARCH-PURPOSE | **Flag:** promised partition regression protection is incomplete. Negative qualification otherwise fulfills M1’s approved scope. |
| ARCH-MOCK | **Pass for M1:** controlled transport doubles share the candidate seam; production/live obligations remain explicitly not-covered. |
| ARCH-CONSTRAINTS | **Pass for M1:** fixture, geometry, reply, evidence, and execution bounds are explicit. |
| ARCH-SECURE | **Pass:** synthetic inputs, visible infrastructure failures, and no production credentials or sessions touched. |
| ARCH-ORDER | **Pass for M1:** serialized execution and joined teardown have controlled lifecycle tests. |
| ARCH-FUNERAL | **Pass:** disposable candidates close their workers; ordinary launches gain no durable artifacts. |

The M1 concept-table entities exist at their stated locations. The broader production entities remain explicitly future work.

## 7. Plan revision recommendations

Add a `## Revisions` entry acknowledging the partition-test gap and recording the byte-preservation/boundary-coverage invariant plus mutation evidence once corrected. Keep M2–M4 behind the documented backend re-plan checkpoint.

---

## Re-review — 2026-09-15T10:30:52-07:00 (SHIP)

| field | value |
|-------|-------|
| issue | 255 — Establish a faithful terminal abstraction for Couch and Pair |
| repo | 000255-lifecycle-state-ownership |
| issue file | workshop/issues/000255-lifecycle-state-ownership.md |
| boundary | milestone M1 |
| milestone | M1 |
| window | b11ab67ff1d290386cf12177f2dacdccbc8551c0..fbcd3d79db19061ec947cc2246391f06cd7f69ef |
| command | sdlc milestone-close --issue 255 --milestone M1 |
| reviewer | codex |
| timestamp | 2026-09-15T10:30:52-07:00 |
| verdict | SHIP |

## Review

```verdict
verdict: SHIP
confidence: high
```

The pinned range satisfies the approved M1 qualification scope. BR-5 is addressed with regression tests that fail under the exact reported mutation. The probe independently reproduces **53 pass, 15 fail, 14 not-covered; qualified=false**. This verdict approves M1’s negative qualification result; backend adoption and issue closure remain subject to the documented later gates.

```findings
dispose:
  - id: BR-1
    disposition: addressed
    note: |
      Retained: runner.go compares complete whole/split observations in both directions; TestRunCaseDetectsUnassertedSplitStateChanges passes.
  - id: BR-2
    disposition: addressed
    note: |
      Retained: candidate.go captures attributes, underline style and color; snapshot regression tests and twelve literal style fixtures pass.
  - id: BR-3
    disposition: addressed
    note: |
      Retained: report.go and runner.go preserve bounded structured evidence; runner and CLI JSON regression tests pass.
  - id: BR-4
    disposition: addressed
    note: |
      README.md:773 documents invocation, evidence and exit meanings, matching cmd/probes/terminalqualify/main.go and the reproduced probe result.
  - id: BR-5
    disposition: addressed
    note: |
      cases_test.go:11, :58 and :67 verify literal partitions, byte preservation, every boundary and production executor delivery. A temporary Go overlay replacing the split pair with []string{input, ""} makes all three tests fail; unmodified tests pass.
```

## 1. Strengths

- Partition coverage now checks actual delivered bytes, including empty, single-byte, Unicode, control-sequence and all-byte inputs.
- Qualification separates literal correctness, split equivalence and bounded presentation evidence.
- Candidate lifecycle tests exercise blocked replies, read failure, cancellation, isolation and joined teardown.
- README and atlas document the new probe; explicit uncovered obligations prevent premature backend approval.

## 2. Critical findings

None.

## 3. Important findings

None.

## 4. Minor findings

None.

## 5. Test coverage notes

Verified:

- Required pinned stat/name-status inspections and targeted patches.
- Normal and race tests for both qualification packages.
- Artifact inventory tests.
- BR-5 mutation failure through a temporary overlay.
- Probe exit 1 and all documented failure results.
- Pinned-range whitespace check.

The full repository suite and live terminal conformance were not rerun during this review. Repository files were unchanged.

## 6. Architectural notes for upcoming work

| Marker | Assessment |
|---|---|
| ARCH-DRY | **Pass:** shared runner, comparator and partition generator; no production parser duplication. |
| ARCH-PURE | **Pass:** observation/report logic runs without IO; emulator and probe are classified as integration. |
| ARCH-PURPOSE | **Pass:** partition generation and delivery now enforce the complete BR-5 rule; negative qualification remains explicit. |
| ARCH-MOCK | **Pass for M1:** controlled transport doubles share the candidate seam; live/composed conformance remains an explicit unmet obligation. |
| ARCH-CONSTRAINTS | **Pass:** fixture, geometry, reply and evidence bounds are present; diagnostic measurements make no production performance claim. |
| ARCH-SECURE | **Pass:** synthetic inputs and bounded evidence introduce no operator-session or credential access. |
| ARCH-ORDER | **Pass for M1:** serialized execution, private snapshots and cancellation tests cover the diagnostic lifecycle. Production ownership remains deferred explicitly. |
| ARCH-FUNERAL | **Pass:** disposable candidates close transport and join workers; ordinary launches gain no durable artifact family. |

The M1 entities exist at their stated locations. The broader proposed core-concept rows belong to later milestones.

## 7. Plan revision recommendations

None required for M1. Preserve the negative adoption decision and the M2–M4 re-plan checkpoint.
