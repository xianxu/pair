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

## Open findings

- **BR-1** [Important] `docs-restate-chord-surface` README.md:131 still documents the deleted nav_boundary as Shift+Alt+arrows, and no row describes the new tab switching
- **BR-2** [Important] `chord-encoding-family-coverage` Only the modifier-4 spelling is registered; the meta-family \x1b[1;10D / \x1b[1;10C analog is missing
- **BR-3** [Minor] `bytes-vs-runes` DeliverChordArgs ranges a string, yielding runes rather than bytes
- **BR-4** [Minor] `dead-code-after-removal` pos_rank is orphaned by the nav_boundary deletion
- **BR-5** [Minor] `comment-claims-unrendered-surface` catalog.go comment credits a "context column" that pair keys does not render
- **BR-6** [Minor] `action-to-chord-mapping-restated` The Action to delivered-chord mapping is restated at three call sites
