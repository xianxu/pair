---
gate: boundary-review
issue: 243
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-13T11:08:03-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: RunSwitchTerminalTab's new "new" direction, the only path the draft pane uses, has no test
          detail: layoutcmd_test.go:248-252 still lists prev/next only; plan Task 4 Step 3 promised to extend it. Deleting `case "new"` at layoutcmd.go:198 leaves the suite green while from-draft M-S-t exits 2 silently. Add the row and assert the delivered bytes.
          family: consumer-path-untested
          round: 1
        - id: BR-2
          severity: Important
          title: Plan row "live check" is ticked but the Log holds no live-run record
          detail: Done-when's live bullet and plan Task 5 Step 5 ("Record in Log") are unmet; the issue ends in an empty duplicate `## Log` heading. Record what was pressed and observed (including typed Alt+arrows still reaching nvim) or untick the row.
          family: verification-evidence-recorded
          round: 1
        - id: BR-3
          severity: Minor
          title: ChordAltT and ChordAltShiftT are two identical newTab cases in handleTerminalChord
          detail: run.go:604 and :619; the file already merges Left/ShiftLeft at :608 (ARCH-DRY). Merge, and report the newTab error via mux.reportError instead of discarding it.
          family: duplicate-dispatch-case
          round: 1
        - id: BR-4
          severity: Minor
          title: Issue file has a duplicate empty `## Log` heading; atlas sentence at architecture.md:477 is missing a word
          detail: '"The test `TabChordFor` returns a chord for which…" reads as a test named TabChordFor.'
          family: artifact-hygiene
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-13T11:19:00-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: TestRunSwitchTerminalTabDeliversTheGlobalBytes asserts bytes for prev/next/new; deleting `case "new"` in a scratch copy fails both RunSwitchTerminalTab tests.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: Log now records a real pty-level encoding check, but Done-when's in-workbench live bullet and the typed Alt+arrow control are still unmet while plan row 6 stays ticked and the plan has no Revisions entry.
          round: 2
        - id: BR-3
          disposition: not-addressed
          note: Cases merged; error still discarded and the justifying comment at run.go:605-610 is wrong (enqueue is non-blocking, ChordAltShiftD reports from the same function, a failed Start has no EOF path).
          round: 2
        - id: BR-4
          disposition: addressed
          note: Single Log heading; atlas sentence now names the guard test correctly.
          round: 2
      findings:
        - id: BR-5
          severity: Minor
          title: zellij WriteChars byte strings restate chordSequences by hand with no guard tying them together
          detail: config.kdl:146 `Alt T` -> "\u{1b}[84;4u" duplicates shortcut.go:413; a typo breaks typed M-S-t with the suite green. Class covers every letter global (Alt D/N/x/...). One test parsing WriteChars binds and asserting DecodeChord yields a global covers them all; follow-up acceptable.
          family: hand-maintained-restatement
          round: 2
      blocked: true
---

# Gate ledger — pair#243 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-13T11:08:03-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `consumer-path-untested` RunSwitchTerminalTab's new "new" direction, the only path the draft pane uses, has no test
  layoutcmd_test.go:248-252 still lists prev/next only; plan Task 4 Step 3 promised to extend it. Deleting `case "new"` at layoutcmd.go:198 leaves the suite green while from-draft M-S-t exits 2 silently. Add the row and assert the delivered bytes.
- **BR-2** [Important] `verification-evidence-recorded` Plan row "live check" is ticked but the Log holds no live-run record
  Done-when's live bullet and plan Task 5 Step 5 ("Record in Log") are unmet; the issue ends in an empty duplicate `## Log` heading. Record what was pressed and observed (including typed Alt+arrows still reaching nvim) or untick the row.
- **BR-3** [Minor] `duplicate-dispatch-case` ChordAltT and ChordAltShiftT are two identical newTab cases in handleTerminalChord
  run.go:604 and :619; the file already merges Left/ShiftLeft at :608 (ARCH-DRY). Merge, and report the newTab error via mux.reportError instead of discarding it.
- **BR-4** [Minor] `artifact-hygiene` Issue file has a duplicate empty `## Log` heading; atlas sentence at architecture.md:477 is missing a word
  "The test `TabChordFor` returns a chord for which…" reads as a test named TabChordFor.

## Round 2 — 2026-09-13T11:19:00-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — TestRunSwitchTerminalTabDeliversTheGlobalBytes asserts bytes for prev/next/new; deleting `case "new"` in a scratch copy fails both RunSwitchTerminalTab tests.
- BR-2 — not-addressed — Log now records a real pty-level encoding check, but Done-when's in-workbench live bullet and the typed Alt+arrow control are still unmet while plan row 6 stays ticked and the plan has no Revisions entry.
- BR-3 — not-addressed — Cases merged; error still discarded and the justifying comment at run.go:605-610 is wrong (enqueue is non-blocking, ChordAltShiftD reports from the same function, a failed Start has no EOF path).
- BR-4 — addressed — Single Log heading; atlas sentence now names the guard test correctly.

### Raised

- **BR-5** [Minor] `hand-maintained-restatement` zellij WriteChars byte strings restate chordSequences by hand with no guard tying them together
  config.kdl:146 `Alt T` -> "\u{1b}[84;4u" duplicates shortcut.go:413; a typo breaks typed M-S-t with the suite green. Class covers every letter global (Alt D/N/x/...). One test parsing WriteChars binds and asserting DecodeChord yields a global covers them all; follow-up acceptable.

## Open findings

- **BR-2** [Important] `verification-evidence-recorded` Plan row "live check" is ticked but the Log holds no live-run record
- **BR-3** [Minor] `duplicate-dispatch-case` ChordAltT and ChordAltShiftT are two identical newTab cases in handleTerminalChord
- **BR-5** [Minor] `hand-maintained-restatement` zellij WriteChars byte strings restate chordSequences by hand with no guard tying them together
