---
gate: plan-quality
issue: 234
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-12T15:48:14-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Minor
          title: Done-when "ESC then j within a keystroke reaches nvim as two keys" is broader than the plan's stated 35 ms residual
          detail: 'The plan defers ESC,j typed inside the deadline to #227; tighten the issue''s Done-when to "after the deadline" so the close review does not read the residual as unmet.'
          family: acceptance-criteria-match-scope
          round: 1
        - id: PQ-2
          severity: Minor
          title: Task 4 oracle rationale cites Decide dispositions, but the guarantee is structural
          detail: shortcut.go:283-284 returns DispositionPass for unlisted right-terminal chords. The test is still sound because the pump consumes chord bytes via chordRest and handleChord (run.go:120) has no mux, so no write op is possible; ground the comment in that.
          family: unbacked-existing-behavior-claim
          round: 1
        - id: PQ-3
          severity: Minor
          title: Keystroke-path latency cost of the deadline is not stated as a budget (ARCH-CONSTRAINTS)
          detail: A bare ESC now reaches nvim up to 35 ms later. One line naming the basis (couch precedent, under nvim's ttimeoutlen) is enough.
          family: operating-envelope-unstated
          round: 1
      blocked: false
    - "n": 2
      timestamp: "2026-09-12T15:51:09-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: 'Done-when now says "after the deadline" and names the inside-deadline residual as #227''s.'
          round: 2
        - id: PQ-2
          disposition: addressed
          note: Oracle now grounded in chordRest + handleChord having no mux; also name handleTerminalChord (run.go:560), which holds the mux but only calls tab verbs, so the structural claim is complete.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Latency budget paragraph states 35 ms, couch precedent, and nvim ttimeoutlen 50 ms.
          round: 2
      findings:
        - id: PQ-4
          severity: Minor
          title: Task 3 Step 5 converts one real-timer split test; two same-class siblings remain on the real timer
          detail: 'TestPumpStdinTerminalShortcutsDoNotLeakWhenSplit (run_test.go:232) and the two "split across reads" rows in TestPumpStdinHandlesTerminalTabActions (run_test.go:180-181) hold a prefix across reads through pumpStdin exactly like the named test. Convert all three to the fake timer, or delete the first two as superseded by the Task 4 generated test (ARCH-PURPOSE: the class, not the instance).'
          family: enumerable-siblings-swept
          round: 2
        - id: PQ-5
          severity: Minor
          title: Generated case (b) cannot tell a deadline flush from the EOF flush
          detail: With autoFire the fake tick and splitReader's EOF race in the select; both paths produce exactly write:<head>. Add the same timer.resets >= 1 assertion Task 1 Step 3 adds, so the case proves the deadline armed rather than sampling one interleaving (ARCH-ORDER at-review lens).
          family: oracle-distinguishes-mechanism
          round: 2
      blocked: false
content_hash: db6d806b5dd2da80f3f684f9f29f085669ddbbcb012b1c9d40075eecc4c05afe
---

# Gate ledger — pair#234 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-12T15:48:14-07:00 (claude) — passed

### Raised

- **PQ-1** [Minor] `acceptance-criteria-match-scope` Done-when "ESC then j within a keystroke reaches nvim as two keys" is broader than the plan's stated 35 ms residual
  The plan defers ESC,j typed inside the deadline to #227; tighten the issue's Done-when to "after the deadline" so the close review does not read the residual as unmet.
- **PQ-2** [Minor] `unbacked-existing-behavior-claim` Task 4 oracle rationale cites Decide dispositions, but the guarantee is structural
  shortcut.go:283-284 returns DispositionPass for unlisted right-terminal chords. The test is still sound because the pump consumes chord bytes via chordRest and handleChord (run.go:120) has no mux, so no write op is possible; ground the comment in that.
- **PQ-3** [Minor] `operating-envelope-unstated` Keystroke-path latency cost of the deadline is not stated as a budget (ARCH-CONSTRAINTS)
  A bare ESC now reaches nvim up to 35 ms later. One line naming the basis (couch precedent, under nvim's ttimeoutlen) is enough.

## Round 2 — 2026-09-12T15:51:09-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Done-when now says "after the deadline" and names the inside-deadline residual as #227's.
- PQ-2 — addressed — Oracle now grounded in chordRest + handleChord having no mux; also name handleTerminalChord (run.go:560), which holds the mux but only calls tab verbs, so the structural claim is complete.
- PQ-3 — addressed — Latency budget paragraph states 35 ms, couch precedent, and nvim ttimeoutlen 50 ms.

### Raised

- **PQ-4** [Minor] `enumerable-siblings-swept` Task 3 Step 5 converts one real-timer split test; two same-class siblings remain on the real timer
  TestPumpStdinTerminalShortcutsDoNotLeakWhenSplit (run_test.go:232) and the two "split across reads" rows in TestPumpStdinHandlesTerminalTabActions (run_test.go:180-181) hold a prefix across reads through pumpStdin exactly like the named test. Convert all three to the fake timer, or delete the first two as superseded by the Task 4 generated test (ARCH-PURPOSE: the class, not the instance).
- **PQ-5** [Minor] `oracle-distinguishes-mechanism` Generated case (b) cannot tell a deadline flush from the EOF flush
  With autoFire the fake tick and splitReader's EOF race in the select; both paths produce exactly write:<head>. Add the same timer.resets >= 1 assertion Task 1 Step 3 adds, so the case proves the deadline armed rather than sampling one interleaving (ARCH-ORDER at-review lens).

## Open findings

- **PQ-4** [Minor] `enumerable-siblings-swept` Task 3 Step 5 converts one real-timer split test; two same-class siblings remain on the real timer
- **PQ-5** [Minor] `oracle-distinguishes-mechanism` Generated case (b) cannot tell a deadline flush from the EOF flush
