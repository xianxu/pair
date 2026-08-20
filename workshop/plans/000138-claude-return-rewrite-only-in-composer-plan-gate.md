---
gate: plan-quality
issue: 138
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-08-19T19:37:16-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Fixture replay hardcodes LF as the composer Return; Claude emits backslash-CR
          detail: |-
            harness_tty_fixture_test.go:284-286 sets wantReturn = "\n" for any composer
            fixture, but Claude's keymap.plainCR is {'\\','\r'} (harness_tty.go:26), so
            claude/<version>/composer.raw fails at the unsplit baseline. Derive wantReturn
            from the profile keymap the way assertHarnessTTYLiveDecision already does
            (harness_tty_live_test.go:646-650). The plan lists this file as modified but no
            step names the change, and Task 3 Step 4 claims all tests pass. atlas/architecture.md:529
            restates the same LF assumption in prose and must change with it.
          round: 1
        - id: PQ-2
          severity: Important
          title: Task 2 depends on fixtures Task 3 creates, but the capture needs Task 3's flip
          detail: |-
            Task 2 Step 1 puts the literal composer and declining fixtures in the differential
            and Step 4 expects GREEN, but capturing them requires the positive gate and a
            registered recognizer: newHarnessTTYLiveClassifier fatals when recognize is nil
            (harness_tty_live_test.go:216-218) and configureHarnessTTY only builds a terminal
            model for composerGatePositive (wrap.go:1473-1476). Muse's differential reads its
            literal with t.Fatalf (composer_recognizers_test.go:255-258), so Task 2 Step 5 would
            commit a red test. Reorder: generated rows, flip, capture, then add literal rows.
          round: 1
        - id: PQ-3
          severity: Important
          title: Permission-prompt step conflates unreachable-screen with recognizer-accepts-it
          detail: |-
            Task 3 Step 3 handles only "cannot reach the prompt". It does not say what must
            happen if the prompt IS reached and claudeComposerActive returns true because
            Claude keeps the composer box painted with the cursor inside. That outcome is a
            recognizer defect this issue must fix (ARCH-PURPOSE), not a gap entry, or the
            issue ships a gate that leaves wrap.go:661-669's single OSC 777 match as the sole
            defense. Also, a Claude with no overlay.raw needs entries in BOTH
            ttyFixtureNegativeGaps and ttyFixtureDiscriminationGaps
            (harness_tty_fixture_test.go:125-137); the plan names only the latter.
          round: 1
        - id: PQ-4
          severity: Important
          title: Blast radius of the flip on pair's default agent is unstated
          detail: |-
            The flip turns on the x/vt terminal model for every Claude session (the default
            agent, atlas/architecture.md:267) and inverts the failure mode: today a wrong guess
            is inert in a menu, after the flip a false negative in a real composer emits bare CR,
            which Claude reads as submit and sends a half-written message. That is exactly what
            the plan's own theme-dependence risk would cause. State the consequence, and add one
            live dogfood step in a real pair session before close; the fixture replay runs at a
            fixed 120x38 over frozen bytes and cannot observe it.
          round: 1
        - id: PQ-5
          severity: Minor
          title: Claude recognizer missing from the adversarial-snapshot guard; compress the case list
          detail: |-
            TestComposerRecognizersRejectAdversarialSnapshotsWithoutBlocking
            (composer_recognizers_test.go:329-336) enumerates codex/agy/muse and must gain claude.
            That mechanical guard is worth more than Task 2 Step 1's prose enumeration of
            fourteen cases, which should compress to a strategy line per risky function.
          round: 1
        - id: PQ-6
          severity: Minor
          title: Evidence covers only the default prompt glyph, not Claude's mode-switched composers
          detail: |-
            All captures show the default U+276F composer. Claude's bash-mode and memory-mode
            composers may paint a different glyph at column 0, which would decline and turn
            Return into a submit. State the expected behavior or record it as a known
            false-negative alongside the theme risk.
          round: 1
        - id: PQ-7
          severity: Minor
          title: atlas/architecture.md's Muse recognizer sentence is stale in the paragraph being edited
          detail: |-
            Line 527 still says museComposerActive requires the prompt "within one row of the
            cursor", which pair#139's enclosing-rule, multi-line implementation superseded
            (composer_recognizers.go:104-131). Task 4 Step 1 edits this paragraph anyway.
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-08-20T07:25:13-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: 'Task 3 derives wantReturn from profile.keymap.plainCR; residual: architecture.md''s Conformance "must remap to LF" line is not named in Task 6.'
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Explicit ordering-constraint section; Task 2 is generated-snapshots-only and literal rows land with the Task 4 capture.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: 'Three-outcome branch makes recognizer-accepts-prompt a blocking defect; residual: outcome 3 names only ttyFixtureDiscriminationGaps, not ttyFixtureNegativeGaps.'
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Blast-radius table plus Task 5 live dogfood with an adapt-<tag>.jsonl telemetry read.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: claude joins the adversarial guard; the case list is now four strategy groups, not fourteen prose bullets.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: Mode table captured; recognizer is glyph- and colour-agnostic and bash-mode.raw is pinned as a second positive fixture.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: Task 6 Step 1 fixes the stale "within one row of the cursor" sentence in the same paragraph.
          round: 2
      blocked: false
    - "n": 3
      timestamp: "2026-08-20T07:31:26-07:00"
      agent: claude
      blocked: false
content_hash: b21a26bf7015cc72d621d12562ea8fbfff62450120dd0e7c155430f2d9a79704
---

# Gate ledger — pair#138 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-08-19T19:37:16-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] Fixture replay hardcodes LF as the composer Return; Claude emits backslash-CR
  harness_tty_fixture_test.go:284-286 sets wantReturn = "\n" for any composer
  fixture, but Claude's keymap.plainCR is {'\\','\r'} (harness_tty.go:26), so
  claude/<version>/composer.raw fails at the unsplit baseline. Derive wantReturn
  from the profile keymap the way assertHarnessTTYLiveDecision already does
  (harness_tty_live_test.go:646-650). The plan lists this file as modified but no
  step names the change, and Task 3 Step 4 claims all tests pass. atlas/architecture.md:529
  restates the same LF assumption in prose and must change with it.
- **PQ-2** [Important] Task 2 depends on fixtures Task 3 creates, but the capture needs Task 3's flip
  Task 2 Step 1 puts the literal composer and declining fixtures in the differential
  and Step 4 expects GREEN, but capturing them requires the positive gate and a
  registered recognizer: newHarnessTTYLiveClassifier fatals when recognize is nil
  (harness_tty_live_test.go:216-218) and configureHarnessTTY only builds a terminal
  model for composerGatePositive (wrap.go:1473-1476). Muse's differential reads its
  literal with t.Fatalf (composer_recognizers_test.go:255-258), so Task 2 Step 5 would
  commit a red test. Reorder: generated rows, flip, capture, then add literal rows.
- **PQ-3** [Important] Permission-prompt step conflates unreachable-screen with recognizer-accepts-it
  Task 3 Step 3 handles only "cannot reach the prompt". It does not say what must
  happen if the prompt IS reached and claudeComposerActive returns true because
  Claude keeps the composer box painted with the cursor inside. That outcome is a
  recognizer defect this issue must fix (ARCH-PURPOSE), not a gap entry, or the
  issue ships a gate that leaves wrap.go:661-669's single OSC 777 match as the sole
  defense. Also, a Claude with no overlay.raw needs entries in BOTH
  ttyFixtureNegativeGaps and ttyFixtureDiscriminationGaps
  (harness_tty_fixture_test.go:125-137); the plan names only the latter.
- **PQ-4** [Important] Blast radius of the flip on pair's default agent is unstated
  The flip turns on the x/vt terminal model for every Claude session (the default
  agent, atlas/architecture.md:267) and inverts the failure mode: today a wrong guess
  is inert in a menu, after the flip a false negative in a real composer emits bare CR,
  which Claude reads as submit and sends a half-written message. That is exactly what
  the plan's own theme-dependence risk would cause. State the consequence, and add one
  live dogfood step in a real pair session before close; the fixture replay runs at a
  fixed 120x38 over frozen bytes and cannot observe it.
- **PQ-5** [Minor] Claude recognizer missing from the adversarial-snapshot guard; compress the case list
  TestComposerRecognizersRejectAdversarialSnapshotsWithoutBlocking
  (composer_recognizers_test.go:329-336) enumerates codex/agy/muse and must gain claude.
  That mechanical guard is worth more than Task 2 Step 1's prose enumeration of
  fourteen cases, which should compress to a strategy line per risky function.
- **PQ-6** [Minor] Evidence covers only the default prompt glyph, not Claude's mode-switched composers
  All captures show the default U+276F composer. Claude's bash-mode and memory-mode
  composers may paint a different glyph at column 0, which would decline and turn
  Return into a submit. State the expected behavior or record it as a known
  false-negative alongside the theme risk.
- **PQ-7** [Minor] atlas/architecture.md's Muse recognizer sentence is stale in the paragraph being edited
  Line 527 still says museComposerActive requires the prompt "within one row of the
  cursor", which pair#139's enclosing-rule, multi-line implementation superseded
  (composer_recognizers.go:104-131). Task 4 Step 1 edits this paragraph anyway.

## Round 2 — 2026-08-20T07:25:13-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Task 3 derives wantReturn from profile.keymap.plainCR; residual: architecture.md's Conformance "must remap to LF" line is not named in Task 6.
- PQ-2 — addressed — Explicit ordering-constraint section; Task 2 is generated-snapshots-only and literal rows land with the Task 4 capture.
- PQ-3 — addressed — Three-outcome branch makes recognizer-accepts-prompt a blocking defect; residual: outcome 3 names only ttyFixtureDiscriminationGaps, not ttyFixtureNegativeGaps.
- PQ-4 — addressed — Blast-radius table plus Task 5 live dogfood with an adapt-<tag>.jsonl telemetry read.
- PQ-5 — addressed — claude joins the adversarial guard; the case list is now four strategy groups, not fourteen prose bullets.
- PQ-6 — addressed — Mode table captured; recognizer is glyph- and colour-agnostic and bash-mode.raw is pinned as a second positive fixture.
- PQ-7 — addressed — Task 6 Step 1 fixes the stale "within one row of the cursor" sentence in the same paragraph.

## Round 3 — 2026-08-20T07:31:26-07:00 (claude) — passed

## Open findings

(none — every finding has been disposed)
