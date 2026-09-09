---
gate: boundary-review
issue: 216
id_prefix: BR
rounds:
    - "n": 1
      timestamp: "2026-09-09T12:04:16-07:00"
      agent: claude
      findings:
        - id: BR-1
          severity: Important
          title: README.md:131 still documents the deleted nav_boundary as Shift+Alt+arrows, and no row describes the new tab switching
          detail: |-
            This diff removed nav_boundary (init.lua keymaps + both keyhelp catalog rows), but
            README.md:131 still reads "Shift+Alt+left / Shift+Alt+right | nvim (normal/insert) |
            Jump to the next region boundary...". The README preamble at 104-106 frames that table
            as the hand-maintained narrative restatement of the chord surface, so it is the one
            consumer of globalBindings that does not derive and it was not swept (ARCH-PURPOSE).
            Delete line 131 and add a row beside README.md:122 describing the from-any-pane,
            focus-preserving tab switch. The atlas half of the docs gate was done correctly.
          family: docs-restate-chord-surface
          round: 1
        - id: BR-2
          severity: Important
          title: Only the modifier-4 spelling is registered; the meta-family \x1b[1;10D / \x1b[1;10C analog is missing
          detail: |-
            shortcut.go:370-372 adds \x1b[1;4D / \x1b[1;4C only. ChordAltLeft deliberately carries
            three spellings including the meta form \x1b[1;9D (added in e6eee5a3); the shift analog
            of modifier 9 is 10, so \x1b[1;10D / \x1b[1;10C belong in the same table. On a terminal
            that reports meta-style modifiers the chord still works from the draft (nvim resolves
            <S-M-Left> itself) but is silently dead in the agent pane and the right pane — a partial
            failure of the "from any pane" Done-when that a single-terminal manual test cannot rule
            out. Two rows; TestNoChordSequenceIsAProperPrefixOfAnother already guards shadowing.
          family: chord-encoding-family-coverage
          round: 1
        - id: BR-3
          severity: Minor
          title: DeliverChordArgs ranges a string, yielding runes rather than bytes
          detail: |-
            shortcut.go:694 does `for _, b := range encodings[0]`, so b is a rune and
            strconv.Itoa(int(b)) emits a code point. Correct today only because every chord
            encoding is ASCII. `range []byte(encodings[0])` states the intent and stays correct
            if a non-ASCII byte ever enters the table.
          family: bytes-vs-runes
          round: 1
        - id: BR-4
          severity: Minor
          title: pos_rank is orphaned by the nav_boundary deletion
          detail: |-
            nvim/init.lua:2446 defines pos_rank, whose only caller was the deleted nav_boundary.
            The sole remaining mention in the tree is the comment at init.lua:3402. No lua linter
            runs in make test, so nothing flags it. The plan's deletion step named
            nav_boundary/ordered_landmarks but not this one.
          family: dead-code-after-removal
          round: 1
        - id: BR-5
          severity: Minor
          title: catalog.go comment credits a "context column" that pair keys does not render
          detail: |-
            keyhelp/catalog.go:83-87 justifies grouping the two globals under the terminal-tab
            heading by saying "the per-row context column is what distinguishes them". Running
            `pair keys` shows no context column — the distinction actually comes from the Help
            wording ("from any pane, without moving focus"), which does hold. The group heading
            "Terminal tabs (in the right terminal)" is likewise corrected only by the row text.
          family: comment-claims-unrendered-surface
          round: 1
        - id: BR-6
          severity: Minor
          title: The Action to delivered-chord mapping is restated at three call sites
          detail: |-
            prev == ChordAltLeft / next == ChordAltRight appears in termcmd/run.go:197-200,
            wrapcmd/wrap.go:1674-1679 and layoutcmd.go:130-136 (plus implicitly in the
            handleTerminalChord case grouping). A two-line workbenchshortcut.TabChordFor(action)
            would make it one fact, matching the diff's own "one implementation of tab switching"
            framing (ARCH-DRY).
          family: action-to-chord-mapping-restated
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-09T12:19:10-07:00"
      agent: claude
      dispose:
        - id: BR-1
          disposition: addressed
          note: README.md:131 nav_boundary row deleted; new "any pane / without moving focus" row at README.md:123.
          round: 2
        - id: BR-2
          disposition: not-addressed
          note: Rows added but nothing pins them; deleting all four leaves the suite green (measured).
          round: 2
        - id: BR-3
          disposition: withdrawn
          note: ChordEncodings returns [][]byte (shortcut.go:393), so the loop variable is already a byte.
          round: 2
        - id: BR-4
          disposition: addressed
          note: pos_rank is gone tree-wide; sibling orphans of the same deletion raised separately.
          round: 2
        - id: BR-5
          disposition: addressed
          note: Verified Context.String() has zero callers; the corrected comment is accurate.
          round: 2
        - id: BR-6
          disposition: addressed
          note: TabChordFor owns the mapping; all three executors route through it and two tests redden on it.
          round: 2
      findings:
        - id: BR-7
          severity: Important
          title: The BR-2 meta-family rows are unpinned — deleting all four leaves the full suite green
          detail: |-
            This is the 2nd finding in family `chord-encoding-family-coverage`. Round 1
            fixed instances (\x1b[1;10D, \x1b[1;10C, \x1b[1;9A, \x1b[1;9B added at
            shortcut.go:367-368,378-379). Do NOT re-add or re-check those rows. The rule
            that covers all of them: for every arrow chord encoded as \x1b[1;<m><letter>,
            the meta sibling \x1b[1;<m+6><letter> must also be registered (alt 3 -> meta 9,
            shift+alt 4 -> shift+meta 10). That rule is mechanically checkable over
            chordSequences and is what should be written, not four more table rows.
            Measured: the four new encodings appear nowhere in the tree but the source
            table; removing all four from a scratch checkout of 813a3e17 left
            `go test ./cmd/internal/workbenchshortcut/ ./cmd/internal/layoutcmd/` green and
            `-run 'Chord|PumpStdin|Documented'` on termcmd green. TestDecodeAltArrowChords
            (shortcut_test.go:320-334) still lists only the four original rows, and
            TestDecodeGlobalChord:258-259 only the bit-2 Up/Down forms.
          family: chord-encoding-family-coverage
          round: 2
        - id: BR-8
          severity: Important
          title: Two nav_boundary orphans survive BR-4 — a stale init.lua comment and an atlas helper list
          detail: |-
            This is the 2nd finding in family `dead-code-after-removal`. Round 1 fixed the
            instance BR-4 named (pos_rank — verified gone tree-wide). Do NOT fix these two
            sites individually. The rule: a deletion is complete only when the deleted
            thing's full reference set is swept — bodies, callers, comments that describe
            the behaviour, and atlas/ — via an explicit enumeration that is run and recorded
            in the Log, not by chasing the sites a reviewer names. No compiler and no
            `make test` step sees any of these.
            Measured prevalence for this deletion: nvim/init.lua:2441-2445 is a comment block
            still describing "Shift+Alt+arrows steps between exactly three landmarks … the
            coarse jump to the far end / back to draft", now a false description of a live
            chord (plus a stray double blank line at 2446-2447); atlas/architecture.md:955
            still lists "`nav_boundary` (Shift+Alt jumps)" among the nvim helpers a reader
            should go read. The other three hits (catalog.go:91, init.lua:3396, atlas:715)
            are deliberate historical references and are correct.
          family: dead-code-after-removal
          round: 2
        - id: BR-9
          severity: Important
          title: termcmd/run.go:197-202 has no test — deleting the whole case leaves the suite green
          detail: |-
            handleTerminalChord returns true for ChordAltShiftLeft/Right (run.go:522-531), so
            the stdin pump never reaches runDecision for these chords (run.go:452). The case's
            only reachable caller is `pair term --test-shortcut`, which is what
            tests/term-pane-shortcuts-test.sh drives — the repo's fake-zellij harness that
            covers every other terminal chord, and which was not extended. No Go test
            references ActionTerminalPrevTab/NextTab outside shortcut_test.go:550.
            Measured: removing the entire case from a scratch checkout of 813a3e17 left
            `go test ./cmd/internal/termcmd/ ./cmd/internal/layoutcmd/
            ./cmd/internal/workbenchshortcut/` reporting only the pre-existing
            `ptychild: operation not permitted` sandbox failures. Fix: add the two rows to
            term-pane-shortcuts-test.sh asserting `write --pane-id 4 27 91 49 59 51 68` /
            `… 67`, or delete the case because the pump owns this pane. wrap.go:1648 states
            this exact risk for the agent pane; it just wasn't applied here.
          family: untested-executor-branch
          round: 2
        - id: BR-10
          severity: Important
          title: SwitchRightTerminalTab re-types FocusRightTerminal's four-step pane-resolution preamble
          detail: |-
            layoutcmd.go:103-117 duplicates layoutcmd.go:38-53 step for step — ListPanesJSON,
            LastTerminalPaneID degrading to "", TerminalPaneIDs degrading to nil,
            pickRightTerminal — including a reworded copy of the graceful-degradation
            comment. The diff's stated ARCH-DRY guarantee ("reuses pickRightTerminal … so
            Alt+k and Alt+Shift+arrow cannot land on different halves") is only half
            delivered: the picker is shared, its inputs are re-derived. Adding a fourth
            preference signal, or changing the degradation policy, will land at one site and
            miss the other. Extract resolveRightTerminal(rt) (zellijpane.Pane, bool, error);
            FocusRightTerminal keeps its `move-focus right` fallback on !ok, and
            SwitchRightTerminalTab keeps its inert `return nil`.
          family: resolution-sequence-restated
          round: 2
        - id: BR-11
          severity: Minor
          title: atlas:715 "can never disagree about which split half" holds only for the delivery path
          detail: |-
            This is the 2nd finding in family `docs-restate-chord-surface` and is Minor, so it
            does not block. Same rule as the I-1 sweep: hand-maintained restatements need the
            enumeration that derived surfaces get for free. From inside a split half the pump
            switches THAT half (run.go:522), regardless of the recorded last-terminal id, so
            Alt+k and Alt+Shift+arrow can point at different halves. Scope the sentence to the
            draft/agent delivery path.
          family: docs-restate-chord-surface
          round: 2
        - id: BR-12
          severity: Minor
          title: init.lua:3402-3404 copies the PAIR_HOME/bin/pair resolution idiom a fifth time
          detail: |-
            Also at 755, 767, 892, 950. Pre-existing pattern, not introduced by this issue; a
            pair_bin() helper would end it.
          family: dead-code-after-removal
          round: 2
        - id: BR-13
          severity: Minor
          title: _G.PairTermPrevTab / PairTermNextTab bodies are untested
          detail: |-
            workbench_route_test.lua pins the key -> function-name mapping; nothing pins the
            jobstart argv at init.lua:3407-3408.
          family: untested-executor-branch
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-09T12:43:20-07:00"
      agent: claude
      dispose:
        - id: BR-2
          disposition: addressed
          note: Both meta rows registered; superseded by BR-7, which is also now addressed.
          round: 3
        - id: BR-7
          disposition: addressed
          note: Verified by mutation — deleting the meta rows reddens TestEveryArrowChordRegistersBothModifierFamilies.
          round: 3
        - id: BR-8
          disposition: addressed
          note: Re-ran the enumeration at head; only deliberate historical references remain.
          round: 3
        - id: BR-9
          disposition: addressed
          note: Verified by mutation — removing runDecision's case fails two harness rows; see the new finding on what those rows actually cover.
          round: 3
        - id: BR-10
          disposition: addressed
          note: resolveRightTerminal extracted at layoutcmd.go:60; both callers use it.
          round: 3
        - id: BR-11
          disposition: addressed
          note: atlas:715 now scopes the guarantee to the delivery path and states the in-split-half exception.
          round: 3
        - id: BR-12
          disposition: addressed
          note: pair_bin() ends the idiom in init.lua; one member outside that file survives, raised below.
          round: 3
        - id: BR-13
          disposition: addressed
          note: switch_terminal_tab_command is pure and pinned; its caller remains unpinned, folded into the new Important finding.
          round: 3
      findings:
        - id: BR-14
          severity: Important
          title: The draft pane's chord chain crosses two seams and no test crosses either; renaming the Dispatch case leaves go test ./... identical to control
          detail: |-
            This is the 3rd finding in family `untested-executor-branch`. Earlier rounds fixed
            instances (the agent pane's executeWorkbenchDecision case, then termcmd's runDecision
            case). Do NOT fix this instance by adding one more draft-specific test. The rule that
            covers all three: a chord is delivered by a CHAIN, and every process- or
            language-boundary in that chain needs a test that crosses it, not two tests that stop
            on either side. Mechanically checkable form: Dispatch's buffered switch becomes a
            map[string]handler so a table-driven test over Families() can assert every
            Status:"implemented" entry is routable, and a Go test asserts the subcommand string
            nvim/workbench_route.lua emits is a name Families() declares.
            Measured: renaming `case "layout switch-terminal-tab"` (dispatcher.go:203) to a typo in
            a scratch checkout of af1b51b7 left `go test ./...` with a failure set byte-identical to
            the unmutated control (36 pre-existing pty "operation not permitted" failures, zero new).
            Runtime behaviour of the mutant: exit 2, `pair-go: layout switch-terminal-tab has no
            buffered route wired` on stderr -- loud at a shell, invisible through
            jobstart(detach=true), which has no on_stderr. Also measured: the rows at
            tests/term-pane-shortcuts-test.sh:99-115 do not cover this path despite their comment
            ("Driven from the DRAFT's focus"). handleChord short-circuits at DecideGlobal before
            reading pane focus, so rewriting all three to `write_panes terminal` leaves them
            PASSing identically; what they exercise is termcmd's runDecision case, whose only
            caller tree-wide is the `--test-shortcut` seam.
          family: untested-executor-branch
          round: 3
        - id: BR-15
          severity: Minor
          title: The meta-family guard is one-directional, and its bit-8=Meta premise is Super under the kitty protocol zellij enables
          detail: |-
            This is the 3rd finding in family `chord-encoding-family-coverage`. Earlier rounds fixed
            instances (four table rows), then replaced them with the rule. Do NOT patch the two
            spellings this names. The rule needs two corrections: (a) shortcut_test.go:613 skips
            `modifier >= 9`, so a chord registered ONLY in the meta spelling passes with its bit-2
            sibling missing -- the sibling relation should be enforced in both directions; (b) the
            premise "bit 8 (meta) gives 9" is xterm's convention, but zellij/config.kdl:37 sets
            support_kitty_keyboard_protocol true, and under KKP bit 8 is SUPER. So the enforced
            rule now also binds Cmd+arrow to the layout ladder and Cmd+Shift+arrow to tab
            switching in every byte-decoding pane. \x1b[1;9D/C has been in the table since
            e6eee5a3 with no reported misfire, so the practical risk is low -- but an encoding
            should be measured on the terminal in play before a test enforces it tree-wide, which
            is the same "derived is not measured" caveat the issue Log applied to modifier 4 and
            not to 9/10.
          family: chord-encoding-family-coverage
          round: 3
        - id: BR-16
          severity: Minor
          title: tests/term-pane-shortcuts-test.sh:115 asserts nothing the two rows above it do not already prove, and passes vacuously
          detail: |-
            The two rows above assert `"$(actions)"` equals exactly the single `write --pane-id 4 ...`
            line, which already proves no focus action was emitted. Measured: under the
            delete-the-runDecision-case mutation, `actions` is empty and the grep -c row still
            PASSes. Either drop it or make it assert something the exact-match rows cannot.
          family: untested-executor-branch
          round: 3
        - id: BR-17
          severity: Minor
          title: The pair_bin() consolidation swept init.lua but not the idiom's member in nvim/scrollback.lua:273
          detail: |-
            This is the 4th finding in family `dead-code-after-removal`. The family rule already
            states it: a consolidation is complete only when the full instance set is enumerated
            and the enumeration recorded, not when the sites a reviewer named are fixed. BR-12
            enumerated five sites, all inside init.lua; the grep that would have found the sixth
            is `grep -rn "bin/pair'" nvim/`. scrollback.lua:273 additionally lacks the
            empty-PAIR_HOME fallback the helper has, so it errors on nil concat rather than
            falling back to `pair` on PATH.
          family: dead-code-after-removal
          round: 3
        - id: BR-18
          severity: Minor
          title: Held Alt+Shift+arrow in the draft spawns 3 detached processes per auto-repeat with no debounce
          detail: |-
            nvim/init.lua:3399 fires jobstart(detach = true) per press; each spawned `pair layout
            switch-terminal-tab` runs `zellij action list-panes` then `zellij action write`. The
            Estimate's frequency note reasons about a deliberate single press, which is the right
            call for one press; key auto-repeat is the case it does not cover, and `detach = true`
            means the extent is not lexically bounded (ARCH-CONSTRAINTS / ARCH-ORDER extent).
            Note for future, not a gate issue -- Alt+k from the draft has the same shape today.
          family: unbounded-keystroke-fanout
          round: 3
      blocked: true
---

# Gate ledger — pair#216 (boundary-review)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-09T12:04:16-07:00 (claude) — BLOCKED

### Raised

- **BR-1** [Important] `docs-restate-chord-surface` README.md:131 still documents the deleted nav_boundary as Shift+Alt+arrows, and no row describes the new tab switching
  This diff removed nav_boundary (init.lua keymaps + both keyhelp catalog rows), but
  README.md:131 still reads "Shift+Alt+left / Shift+Alt+right | nvim (normal/insert) |
  Jump to the next region boundary...". The README preamble at 104-106 frames that table
  as the hand-maintained narrative restatement of the chord surface, so it is the one
  consumer of globalBindings that does not derive and it was not swept (ARCH-PURPOSE).
  Delete line 131 and add a row beside README.md:122 describing the from-any-pane,
  focus-preserving tab switch. The atlas half of the docs gate was done correctly.
- **BR-2** [Important] `chord-encoding-family-coverage` Only the modifier-4 spelling is registered; the meta-family \x1b[1;10D / \x1b[1;10C analog is missing
  shortcut.go:370-372 adds \x1b[1;4D / \x1b[1;4C only. ChordAltLeft deliberately carries
  three spellings including the meta form \x1b[1;9D (added in e6eee5a3); the shift analog
  of modifier 9 is 10, so \x1b[1;10D / \x1b[1;10C belong in the same table. On a terminal
  that reports meta-style modifiers the chord still works from the draft (nvim resolves
  <S-M-Left> itself) but is silently dead in the agent pane and the right pane — a partial
  failure of the "from any pane" Done-when that a single-terminal manual test cannot rule
  out. Two rows; TestNoChordSequenceIsAProperPrefixOfAnother already guards shadowing.
- **BR-3** [Minor] `bytes-vs-runes` DeliverChordArgs ranges a string, yielding runes rather than bytes
  shortcut.go:694 does `for _, b := range encodings[0]`, so b is a rune and
  strconv.Itoa(int(b)) emits a code point. Correct today only because every chord
  encoding is ASCII. `range []byte(encodings[0])` states the intent and stays correct
  if a non-ASCII byte ever enters the table.
- **BR-4** [Minor] `dead-code-after-removal` pos_rank is orphaned by the nav_boundary deletion
  nvim/init.lua:2446 defines pos_rank, whose only caller was the deleted nav_boundary.
  The sole remaining mention in the tree is the comment at init.lua:3402. No lua linter
  runs in make test, so nothing flags it. The plan's deletion step named
  nav_boundary/ordered_landmarks but not this one.
- **BR-5** [Minor] `comment-claims-unrendered-surface` catalog.go comment credits a "context column" that pair keys does not render
  keyhelp/catalog.go:83-87 justifies grouping the two globals under the terminal-tab
  heading by saying "the per-row context column is what distinguishes them". Running
  `pair keys` shows no context column — the distinction actually comes from the Help
  wording ("from any pane, without moving focus"), which does hold. The group heading
  "Terminal tabs (in the right terminal)" is likewise corrected only by the row text.
- **BR-6** [Minor] `action-to-chord-mapping-restated` The Action to delivered-chord mapping is restated at three call sites
  prev == ChordAltLeft / next == ChordAltRight appears in termcmd/run.go:197-200,
  wrapcmd/wrap.go:1674-1679 and layoutcmd.go:130-136 (plus implicitly in the
  handleTerminalChord case grouping). A two-line workbenchshortcut.TabChordFor(action)
  would make it one fact, matching the diff's own "one implementation of tab switching"
  framing (ARCH-DRY).

## Round 2 — 2026-09-09T12:19:10-07:00 (claude) — BLOCKED

### Disposed

- BR-1 — addressed — README.md:131 nav_boundary row deleted; new "any pane / without moving focus" row at README.md:123.
- BR-2 — not-addressed — Rows added but nothing pins them; deleting all four leaves the suite green (measured).
- BR-3 — withdrawn — ChordEncodings returns [][]byte (shortcut.go:393), so the loop variable is already a byte.
- BR-4 — addressed — pos_rank is gone tree-wide; sibling orphans of the same deletion raised separately.
- BR-5 — addressed — Verified Context.String() has zero callers; the corrected comment is accurate.
- BR-6 — addressed — TabChordFor owns the mapping; all three executors route through it and two tests redden on it.

### Raised

- **BR-7** [Important] `chord-encoding-family-coverage` The BR-2 meta-family rows are unpinned — deleting all four leaves the full suite green
  This is the 2nd finding in family `chord-encoding-family-coverage`. Round 1
  fixed instances (\x1b[1;10D, \x1b[1;10C, \x1b[1;9A, \x1b[1;9B added at
  shortcut.go:367-368,378-379). Do NOT re-add or re-check those rows. The rule
  that covers all of them: for every arrow chord encoded as \x1b[1;<m><letter>,
  the meta sibling \x1b[1;<m+6><letter> must also be registered (alt 3 -> meta 9,
  shift+alt 4 -> shift+meta 10). That rule is mechanically checkable over
  chordSequences and is what should be written, not four more table rows.
  Measured: the four new encodings appear nowhere in the tree but the source
  table; removing all four from a scratch checkout of 813a3e17 left
  `go test ./cmd/internal/workbenchshortcut/ ./cmd/internal/layoutcmd/` green and
  `-run 'Chord|PumpStdin|Documented'` on termcmd green. TestDecodeAltArrowChords
  (shortcut_test.go:320-334) still lists only the four original rows, and
  TestDecodeGlobalChord:258-259 only the bit-2 Up/Down forms.
- **BR-8** [Important] `dead-code-after-removal` Two nav_boundary orphans survive BR-4 — a stale init.lua comment and an atlas helper list
  This is the 2nd finding in family `dead-code-after-removal`. Round 1 fixed the
  instance BR-4 named (pos_rank — verified gone tree-wide). Do NOT fix these two
  sites individually. The rule: a deletion is complete only when the deleted
  thing's full reference set is swept — bodies, callers, comments that describe
  the behaviour, and atlas/ — via an explicit enumeration that is run and recorded
  in the Log, not by chasing the sites a reviewer names. No compiler and no
  `make test` step sees any of these.
  Measured prevalence for this deletion: nvim/init.lua:2441-2445 is a comment block
  still describing "Shift+Alt+arrows steps between exactly three landmarks … the
  coarse jump to the far end / back to draft", now a false description of a live
  chord (plus a stray double blank line at 2446-2447); atlas/architecture.md:955
  still lists "`nav_boundary` (Shift+Alt jumps)" among the nvim helpers a reader
  should go read. The other three hits (catalog.go:91, init.lua:3396, atlas:715)
  are deliberate historical references and are correct.
- **BR-9** [Important] `untested-executor-branch` termcmd/run.go:197-202 has no test — deleting the whole case leaves the suite green
  handleTerminalChord returns true for ChordAltShiftLeft/Right (run.go:522-531), so
  the stdin pump never reaches runDecision for these chords (run.go:452). The case's
  only reachable caller is `pair term --test-shortcut`, which is what
  tests/term-pane-shortcuts-test.sh drives — the repo's fake-zellij harness that
  covers every other terminal chord, and which was not extended. No Go test
  references ActionTerminalPrevTab/NextTab outside shortcut_test.go:550.
  Measured: removing the entire case from a scratch checkout of 813a3e17 left
  `go test ./cmd/internal/termcmd/ ./cmd/internal/layoutcmd/
  ./cmd/internal/workbenchshortcut/` reporting only the pre-existing
  `ptychild: operation not permitted` sandbox failures. Fix: add the two rows to
  term-pane-shortcuts-test.sh asserting `write --pane-id 4 27 91 49 59 51 68` /
  `… 67`, or delete the case because the pump owns this pane. wrap.go:1648 states
  this exact risk for the agent pane; it just wasn't applied here.
- **BR-10** [Important] `resolution-sequence-restated` SwitchRightTerminalTab re-types FocusRightTerminal's four-step pane-resolution preamble
  layoutcmd.go:103-117 duplicates layoutcmd.go:38-53 step for step — ListPanesJSON,
  LastTerminalPaneID degrading to "", TerminalPaneIDs degrading to nil,
  pickRightTerminal — including a reworded copy of the graceful-degradation
  comment. The diff's stated ARCH-DRY guarantee ("reuses pickRightTerminal … so
  Alt+k and Alt+Shift+arrow cannot land on different halves") is only half
  delivered: the picker is shared, its inputs are re-derived. Adding a fourth
  preference signal, or changing the degradation policy, will land at one site and
  miss the other. Extract resolveRightTerminal(rt) (zellijpane.Pane, bool, error);
  FocusRightTerminal keeps its `move-focus right` fallback on !ok, and
  SwitchRightTerminalTab keeps its inert `return nil`.
- **BR-11** [Minor] `docs-restate-chord-surface` atlas:715 "can never disagree about which split half" holds only for the delivery path
  This is the 2nd finding in family `docs-restate-chord-surface` and is Minor, so it
  does not block. Same rule as the I-1 sweep: hand-maintained restatements need the
  enumeration that derived surfaces get for free. From inside a split half the pump
  switches THAT half (run.go:522), regardless of the recorded last-terminal id, so
  Alt+k and Alt+Shift+arrow can point at different halves. Scope the sentence to the
  draft/agent delivery path.
- **BR-12** [Minor] `dead-code-after-removal` init.lua:3402-3404 copies the PAIR_HOME/bin/pair resolution idiom a fifth time
  Also at 755, 767, 892, 950. Pre-existing pattern, not introduced by this issue; a
  pair_bin() helper would end it.
- **BR-13** [Minor] `untested-executor-branch` _G.PairTermPrevTab / PairTermNextTab bodies are untested
  workbench_route_test.lua pins the key -> function-name mapping; nothing pins the
  jobstart argv at init.lua:3407-3408.

## Round 3 — 2026-09-09T12:43:20-07:00 (claude) — BLOCKED

### Disposed

- BR-2 — addressed — Both meta rows registered; superseded by BR-7, which is also now addressed.
- BR-7 — addressed — Verified by mutation — deleting the meta rows reddens TestEveryArrowChordRegistersBothModifierFamilies.
- BR-8 — addressed — Re-ran the enumeration at head; only deliberate historical references remain.
- BR-9 — addressed — Verified by mutation — removing runDecision's case fails two harness rows; see the new finding on what those rows actually cover.
- BR-10 — addressed — resolveRightTerminal extracted at layoutcmd.go:60; both callers use it.
- BR-11 — addressed — atlas:715 now scopes the guarantee to the delivery path and states the in-split-half exception.
- BR-12 — addressed — pair_bin() ends the idiom in init.lua; one member outside that file survives, raised below.
- BR-13 — addressed — switch_terminal_tab_command is pure and pinned; its caller remains unpinned, folded into the new Important finding.

### Raised

- **BR-14** [Important] `untested-executor-branch` The draft pane's chord chain crosses two seams and no test crosses either; renaming the Dispatch case leaves go test ./... identical to control
  This is the 3rd finding in family `untested-executor-branch`. Earlier rounds fixed
  instances (the agent pane's executeWorkbenchDecision case, then termcmd's runDecision
  case). Do NOT fix this instance by adding one more draft-specific test. The rule that
  covers all three: a chord is delivered by a CHAIN, and every process- or
  language-boundary in that chain needs a test that crosses it, not two tests that stop
  on either side. Mechanically checkable form: Dispatch's buffered switch becomes a
  map[string]handler so a table-driven test over Families() can assert every
  Status:"implemented" entry is routable, and a Go test asserts the subcommand string
  nvim/workbench_route.lua emits is a name Families() declares.
  Measured: renaming `case "layout switch-terminal-tab"` (dispatcher.go:203) to a typo in
  a scratch checkout of af1b51b7 left `go test ./...` with a failure set byte-identical to
  the unmutated control (36 pre-existing pty "operation not permitted" failures, zero new).
  Runtime behaviour of the mutant: exit 2, `pair-go: layout switch-terminal-tab has no
  buffered route wired` on stderr -- loud at a shell, invisible through
  jobstart(detach=true), which has no on_stderr. Also measured: the rows at
  tests/term-pane-shortcuts-test.sh:99-115 do not cover this path despite their comment
  ("Driven from the DRAFT's focus"). handleChord short-circuits at DecideGlobal before
  reading pane focus, so rewriting all three to `write_panes terminal` leaves them
  PASSing identically; what they exercise is termcmd's runDecision case, whose only
  caller tree-wide is the `--test-shortcut` seam.
- **BR-15** [Minor] `chord-encoding-family-coverage` The meta-family guard is one-directional, and its bit-8=Meta premise is Super under the kitty protocol zellij enables
  This is the 3rd finding in family `chord-encoding-family-coverage`. Earlier rounds fixed
  instances (four table rows), then replaced them with the rule. Do NOT patch the two
  spellings this names. The rule needs two corrections: (a) shortcut_test.go:613 skips
  `modifier >= 9`, so a chord registered ONLY in the meta spelling passes with its bit-2
  sibling missing -- the sibling relation should be enforced in both directions; (b) the
  premise "bit 8 (meta) gives 9" is xterm's convention, but zellij/config.kdl:37 sets
  support_kitty_keyboard_protocol true, and under KKP bit 8 is SUPER. So the enforced
  rule now also binds Cmd+arrow to the layout ladder and Cmd+Shift+arrow to tab
  switching in every byte-decoding pane. \x1b[1;9D/C has been in the table since
  e6eee5a3 with no reported misfire, so the practical risk is low -- but an encoding
  should be measured on the terminal in play before a test enforces it tree-wide, which
  is the same "derived is not measured" caveat the issue Log applied to modifier 4 and
  not to 9/10.
- **BR-16** [Minor] `untested-executor-branch` tests/term-pane-shortcuts-test.sh:115 asserts nothing the two rows above it do not already prove, and passes vacuously
  The two rows above assert `"$(actions)"` equals exactly the single `write --pane-id 4 ...`
  line, which already proves no focus action was emitted. Measured: under the
  delete-the-runDecision-case mutation, `actions` is empty and the grep -c row still
  PASSes. Either drop it or make it assert something the exact-match rows cannot.
- **BR-17** [Minor] `dead-code-after-removal` The pair_bin() consolidation swept init.lua but not the idiom's member in nvim/scrollback.lua:273
  This is the 4th finding in family `dead-code-after-removal`. The family rule already
  states it: a consolidation is complete only when the full instance set is enumerated
  and the enumeration recorded, not when the sites a reviewer named are fixed. BR-12
  enumerated five sites, all inside init.lua; the grep that would have found the sixth
  is `grep -rn "bin/pair'" nvim/`. scrollback.lua:273 additionally lacks the
  empty-PAIR_HOME fallback the helper has, so it errors on nil concat rather than
  falling back to `pair` on PATH.
- **BR-18** [Minor] `unbounded-keystroke-fanout` Held Alt+Shift+arrow in the draft spawns 3 detached processes per auto-repeat with no debounce
  nvim/init.lua:3399 fires jobstart(detach = true) per press; each spawned `pair layout
  switch-terminal-tab` runs `zellij action list-panes` then `zellij action write`. The
  Estimate's frequency note reasons about a deliberate single press, which is the right
  call for one press; key auto-repeat is the case it does not cover, and `detach = true`
  means the extent is not lexically bounded (ARCH-CONSTRAINTS / ARCH-ORDER extent).
  Note for future, not a gate issue -- Alt+k from the draft has the same shape today.

## Open findings

- **BR-14** [Important] `untested-executor-branch` The draft pane's chord chain crosses two seams and no test crosses either; renaming the Dispatch case leaves go test ./... identical to control
- **BR-15** [Minor] `chord-encoding-family-coverage` The meta-family guard is one-directional, and its bit-8=Meta premise is Super under the kitty protocol zellij enables
- **BR-16** [Minor] `untested-executor-branch` tests/term-pane-shortcuts-test.sh:115 asserts nothing the two rows above it do not already prove, and passes vacuously
- **BR-17** [Minor] `dead-code-after-removal` The pair_bin() consolidation swept init.lua but not the idiom's member in nvim/scrollback.lua:273
- **BR-18** [Minor] `unbounded-keystroke-fanout` Held Alt+Shift+arrow in the draft spawns 3 detached processes per auto-repeat with no debounce
