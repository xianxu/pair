---
gate: plan-quality
issue: 216
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-09T10:54:07-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: Step 1's "nothing is lost by rebinding" is wrong — Alt+Shift+arrows are a live draft feature
          detail: |-
            nvim/init.lua:3552-3555 binds <S-M-Left>/<S-M-Right> in normal+insert to
            nav_boundary(-1)/nav_boundary(1), documented in pair keys via
            keyhelp/catalog.go:58-59 under "Draft — history and queue". The plan asserts
            the only mapping is scrollback.lua's <nop>. Compounding it, init.lua:3532
            installs the global maps BEFORE line 3552, so the later keymap.set wins:
            adding the two GlobalBindings alone leaves the draft pane still running
            nav_boundary, and the only failing test is a classification failure that
            names nothing about the shadowing. The plan must state nav_boundary's fate
            (deleted, or rebound to another chord), remove init.lua:3552-3555, and
            repoint or drop catalog.go:58-59 so TestEveryCatalogEntryStillExists and
            TestEveryGlobalChordIsClassified both stay green on the right spelling.
          family: plan-claims-existing-behavior
          round: 1
        - id: PQ-2
          severity: Important
          title: '"The right pane handles the chord locally" names no function; the only mux seam trips a guard'
          detail: |-
            runDecision (termcmd/run.go:168) has no mux; handleTerminalChord
            (run.go:504) does, but TestEveryHandledTerminalChordIsDocumented
            (run_test.go:1183-1196) requires every chord it handles to be in
            RoleBindings(), while TestRoleBindingsCoverTerminalSwitch
            (shortcut_test.go:470-474) skips globals. Name the seam and say how the two
            documentation guards reconcile without duplicating Help across
            globalBindings and roleBindings (ARCH-DRY).
          family: executor-seam-undeclared
          round: 1
        - id: PQ-3
          severity: Important
          title: New chords appended after ChordAltShiftEnter silently escape both doc guards
          detail: |-
            run_test.go:1188 and shortcut_test.go:473 both bound the chord space at
            `chord <= ChordAltShiftEnter`, the last const in the iota block
            (shortcut.go:51). Appending ChordAltShiftLeft/Right after it drops them from
            both loops with no failure. The plan should say the bound moves, or better,
            introduce an explicit chordMax sentinel so the next chord cannot repeat this.
          family: chord-enum-sentinel
          round: 1
        - id: PQ-4
          severity: Important
          title: Which of two split terminal halves receives the write is unspecified
          detail: |-
            LastTerminalPaneStore / LiveTerminalPaneIDsFromEnv are stores, not a picker.
            After Alt+Shift+d there are two pair term panes with independent tab sets;
            layoutcmd.go:66-85's pickRightTerminal already encodes the tie-break
            (recorded half > zellij focus > pane order) and is unexported. State whether
            the new path reuses it (ARCH-DRY) or picks otherwise — a divergence means
            Alt+k lands in one half while Alt+Shift+arrow switches the other's tabs.
          family: terminal-pane-resolution
          round: 1
        - id: PQ-5
          severity: Important
          title: No named function or adversarial strategy for the chord byte scanner
          detail: |-
            Two rows are added to chordSequences (shortcut.go:310-334), iterated in
            order by FindChord, DecodeChord, DecodeChordPrefix and IsChordPrefix, which
            feed the partial-chunk `held` buffer at run.go:422-489. Name those functions
            and give one strategy line — e.g. a property over the table asserting no
            sequence is a proper prefix of another, seeded with split-arrival chunking.
          family: test-strategy-missing-for-risky-surface
          round: 1
        - id: PQ-6
          severity: Important
          title: The draft path — the pane the issue exists for — has no test row
          detail: |-
            The test list covers pure argv, DecideGlobal, the agent executor and the
            right pane, but nothing for the draft's Lua function or the new
            `pair term --switch-tab prev|next` argument parsing.
            nvim/workbench_route_test.lua is the existing Lua seam (ARCH-PURPOSE: the
            draft is the stated purpose, not a follow-up).
          family: test-strategy-missing-for-risky-surface
          round: 1
        - id: PQ-7
          severity: Minor
          title: No ordering statement for the events the caller cannot block
          detail: |-
            Say what happens when the terminal pane exits between resolution and
            `zellij action write` (stale pane id), and what a rapid double-press does
            (two independent subprocesses, no ordering guarantee). One line: delivery is
            fire-and-forget, failure is silent, nothing rolls back.
          family: arch-order-unstated
          round: 1
        - id: PQ-8
          severity: Minor
          title: The plan never says what it is deliberately NOT building
          detail: |-
            Candidates: not preserving nav_boundary under a new chord, not addressing
            per-half tab sets after a split, not adding an absolute
            `--switch-tab <n>` form, not touching Alt+Left/Right.
          family: non-goals-absent
          round: 1
        - id: PQ-9
          severity: Minor
          title: '"that chord''s canonical encoding" is singular; ChordAltLeft has three'
          detail: |-
            shortcut.go:331-332 give ChordAltLeft three encodings
            (\x1b[1;3D, \x1b[1;9D, \x1b[3D) and ChordEncodings returns a slice. Say
            which one DeliverChordArgs emits, and what the "both encodings" test row means.
          family: canonical-encoding-ambiguous
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-09T11:38:14-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Verified init.lua:3552-3555, the 3532 install-order shadowing, and catalog.go:58-59; deletion is now its own step with the operator decision recorded in Revisions.
          round: 2
        - id: PQ-2
          disposition: addressed
          note: handleTerminalChord (run.go:504) named; both doc guards reconciled by extending shortcut_test.go:474's globals exemption to the termcmd guard, so Help is authored once.
          round: 2
        - id: PQ-3
          disposition: addressed
          note: chordMax sentinel derives both bounds at run_test.go:1188 and shortcut_test.go:473.
          round: 2
        - id: PQ-4
          disposition: addressed
          note: Exports and reuses layoutcmd.pickRightTerminal (layoutcmd.go:66) so Alt+k and the new chord cannot select different split halves.
          round: 2
        - id: PQ-5
          disposition: addressed
          note: Four scanner functions named; property that no sequence is a proper prefix of another, plus split-arrival chunking rows.
          round: 2
        - id: PQ-6
          disposition: addressed
          note: nvim/workbench_route_test.lua routing row plus pair term --switch-tab argument-parsing rows; the file exists.
          round: 2
        - id: PQ-7
          disposition: addressed
          note: Stale pane id and rapid double-press both covered; fire-and-forget, silent failure, no rollback.
          round: 2
        - id: PQ-8
          disposition: addressed
          note: Four non-goals stated, including the deleted nav_boundary and per-half tab sets.
          round: 2
        - id: PQ-9
          disposition: addressed
          note: Emits ChordEncodings(chord)[0] and asserts that exact vector; zellij action write --pane-id confirmed against the installed 0.44.3 binary.
          round: 2
      blocked: false
content_hash: d682f8d8001af66e7cc099127b2060c05bc2830667b55dd6a46d4309051036c5
---

# Gate ledger — pair#216 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-09T10:54:07-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `plan-claims-existing-behavior` Step 1's "nothing is lost by rebinding" is wrong — Alt+Shift+arrows are a live draft feature
  nvim/init.lua:3552-3555 binds <S-M-Left>/<S-M-Right> in normal+insert to
  nav_boundary(-1)/nav_boundary(1), documented in pair keys via
  keyhelp/catalog.go:58-59 under "Draft — history and queue". The plan asserts
  the only mapping is scrollback.lua's <nop>. Compounding it, init.lua:3532
  installs the global maps BEFORE line 3552, so the later keymap.set wins:
  adding the two GlobalBindings alone leaves the draft pane still running
  nav_boundary, and the only failing test is a classification failure that
  names nothing about the shadowing. The plan must state nav_boundary's fate
  (deleted, or rebound to another chord), remove init.lua:3552-3555, and
  repoint or drop catalog.go:58-59 so TestEveryCatalogEntryStillExists and
  TestEveryGlobalChordIsClassified both stay green on the right spelling.
- **PQ-2** [Important] `executor-seam-undeclared` "The right pane handles the chord locally" names no function; the only mux seam trips a guard
  runDecision (termcmd/run.go:168) has no mux; handleTerminalChord
  (run.go:504) does, but TestEveryHandledTerminalChordIsDocumented
  (run_test.go:1183-1196) requires every chord it handles to be in
  RoleBindings(), while TestRoleBindingsCoverTerminalSwitch
  (shortcut_test.go:470-474) skips globals. Name the seam and say how the two
  documentation guards reconcile without duplicating Help across
  globalBindings and roleBindings (ARCH-DRY).
- **PQ-3** [Important] `chord-enum-sentinel` New chords appended after ChordAltShiftEnter silently escape both doc guards
  run_test.go:1188 and shortcut_test.go:473 both bound the chord space at
  `chord <= ChordAltShiftEnter`, the last const in the iota block
  (shortcut.go:51). Appending ChordAltShiftLeft/Right after it drops them from
  both loops with no failure. The plan should say the bound moves, or better,
  introduce an explicit chordMax sentinel so the next chord cannot repeat this.
- **PQ-4** [Important] `terminal-pane-resolution` Which of two split terminal halves receives the write is unspecified
  LastTerminalPaneStore / LiveTerminalPaneIDsFromEnv are stores, not a picker.
  After Alt+Shift+d there are two pair term panes with independent tab sets;
  layoutcmd.go:66-85's pickRightTerminal already encodes the tie-break
  (recorded half > zellij focus > pane order) and is unexported. State whether
  the new path reuses it (ARCH-DRY) or picks otherwise — a divergence means
  Alt+k lands in one half while Alt+Shift+arrow switches the other's tabs.
- **PQ-5** [Important] `test-strategy-missing-for-risky-surface` No named function or adversarial strategy for the chord byte scanner
  Two rows are added to chordSequences (shortcut.go:310-334), iterated in
  order by FindChord, DecodeChord, DecodeChordPrefix and IsChordPrefix, which
  feed the partial-chunk `held` buffer at run.go:422-489. Name those functions
  and give one strategy line — e.g. a property over the table asserting no
  sequence is a proper prefix of another, seeded with split-arrival chunking.
- **PQ-6** [Important] `test-strategy-missing-for-risky-surface` The draft path — the pane the issue exists for — has no test row
  The test list covers pure argv, DecideGlobal, the agent executor and the
  right pane, but nothing for the draft's Lua function or the new
  `pair term --switch-tab prev|next` argument parsing.
  nvim/workbench_route_test.lua is the existing Lua seam (ARCH-PURPOSE: the
  draft is the stated purpose, not a follow-up).
- **PQ-7** [Minor] `arch-order-unstated` No ordering statement for the events the caller cannot block
  Say what happens when the terminal pane exits between resolution and
  `zellij action write` (stale pane id), and what a rapid double-press does
  (two independent subprocesses, no ordering guarantee). One line: delivery is
  fire-and-forget, failure is silent, nothing rolls back.
- **PQ-8** [Minor] `non-goals-absent` The plan never says what it is deliberately NOT building
  Candidates: not preserving nav_boundary under a new chord, not addressing
  per-half tab sets after a split, not adding an absolute
  `--switch-tab <n>` form, not touching Alt+Left/Right.
- **PQ-9** [Minor] `canonical-encoding-ambiguous` "that chord's canonical encoding" is singular; ChordAltLeft has three
  shortcut.go:331-332 give ChordAltLeft three encodings
  (\x1b[1;3D, \x1b[1;9D, \x1b[3D) and ChordEncodings returns a slice. Say
  which one DeliverChordArgs emits, and what the "both encodings" test row means.

## Round 2 — 2026-09-09T11:38:14-07:00 (claude) — passed

### Disposed

- PQ-1 — addressed — Verified init.lua:3552-3555, the 3532 install-order shadowing, and catalog.go:58-59; deletion is now its own step with the operator decision recorded in Revisions.
- PQ-2 — addressed — handleTerminalChord (run.go:504) named; both doc guards reconciled by extending shortcut_test.go:474's globals exemption to the termcmd guard, so Help is authored once.
- PQ-3 — addressed — chordMax sentinel derives both bounds at run_test.go:1188 and shortcut_test.go:473.
- PQ-4 — addressed — Exports and reuses layoutcmd.pickRightTerminal (layoutcmd.go:66) so Alt+k and the new chord cannot select different split halves.
- PQ-5 — addressed — Four scanner functions named; property that no sequence is a proper prefix of another, plus split-arrival chunking rows.
- PQ-6 — addressed — nvim/workbench_route_test.lua routing row plus pair term --switch-tab argument-parsing rows; the file exists.
- PQ-7 — addressed — Stale pane id and rapid double-press both covered; fire-and-forget, silent failure, no rollback.
- PQ-8 — addressed — Four non-goals stated, including the deleted nav_boundary and per-half tab sets.
- PQ-9 — addressed — Emits ChordEncodings(chord)[0] and asserts that exact vector; zellij action write --pane-id confirmed against the installed 0.44.3 binary.

## Open findings

(none — every finding has been disposed)
