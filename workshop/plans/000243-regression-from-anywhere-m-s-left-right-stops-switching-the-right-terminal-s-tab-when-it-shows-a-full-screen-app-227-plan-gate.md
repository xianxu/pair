---
gate: plan-quality
issue: 243
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-13T10:42:57-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Important
          title: Plan omits the zellij config.kdl bind rows that define which bytes Alt+Shift+t delivers
          detail: Every Alt+letter global is captured by zellij and written as fixed KKP bytes (config.kdl:143 Alt D, :174 Alt d, :245 Alt N; reason at :236-238); arrows needed none. Add unbind/bind "Alt T" rows, rebuild the embedded bundle (drift_test.go:88), and write the full surface enumeration incl. namedChord at run.go:85-115.
          family: chord-surface-inventory
          round: 1
        - id: PQ-2
          severity: Important
          title: Issue Plan says ESC[84;4u, plan file says ESC[116;4u; the ChordAltShiftD mirror is 84
          detail: 68;4u is uppercase D, so mirroring it gives 84;4u, matching the uppercase-plus-mod-4 precedent (68;4u, 78;4u) that the zellij WriteChars rows dictate. Reconcile both artifacts on 84;4u; 116;4u may be an additional form, not the only one.
          family: chord-encoding-provenance
          round: 1
        - id: PQ-3
          severity: Minor
          title: Legacy ESC T on a global chord reopens the ESC-ambiguity window under full-screen apps
          detail: Globals are never passed through, so ESC then T within EscapeAmbiguity (shortcut.go:405-416) fires new-tab in nvim or the agent pane. Other letter globals carry only the KKP form; drop the legacy row.
          family: global-chord-legacy-form
          round: 1
      blocked: true
    - "n": 2
      timestamp: "2026-09-13T10:48:26-07:00"
      agent: claude
      dispose:
        - id: PQ-1
          disposition: addressed
          note: Task 4b adds the Alt T unbind/bind rows mirroring config.kdl:116/133/143 and names TestEmbeddedSourcesMatchTree (keyhelp/drift_test.go:88); Task 1 adds the namedChord case.
          round: 2
        - id: PQ-2
          disposition: not-addressed
          note: 'Prose is reconciled on 84;4u, but Task 4''s pump test still delivers "\x1b[116;4u" and Task 5 Step 1 tells the implementer to add 116;4u as a second chordSequences row on a keymap mismatch. Rule: zellij''s WriteChars row dictates the bytes; every consumer derives from it; never add a byte form to fix a consumer. Fix both lines.'
          round: 2
        - id: PQ-3
          disposition: addressed
          note: Legacy ESC T dropped; Task 1 pins the KKP form as the only sequence, matching the ChordAltShiftN precedent at shortcut.go:383.
          round: 2
      findings:
        - id: PQ-4
          severity: Minor
          title: NvimKey should be the uppercase-letter form proven by the ChordAltShiftN row, not a guessed spelling with a byte-form fallback
          detail: '2nd finding in this family. The row at shortcut.go:156 binds 78;4u to NvimKey "<M-N>" and works live; the plan picks "<S-M-t>" and calls the match unprovable, then falls back to adding a 116;4u byte row. Rule: for a shift+letter global the NvimKey is the uppercase-letter form matching the existing "<M-N>" row, and a consumer-side mismatch is fixed at the consumer''s spelling, never by adding a byte form. Two sites in the plan (Task 3 row, Task 5 fallback sentence).'
          family: chord-encoding-provenance
          round: 2
      blocked: true
    - "n": 3
      timestamp: "2026-09-13T10:50:48-07:00"
      agent: claude
      dispose:
        - id: PQ-2
          disposition: addressed
          note: Issue Plan and plan file both carry \x1b[84;4u only, mirroring the 68;4u / 78;4u precedent; 116;4u is gone.
          round: 3
        - id: PQ-4
          disposition: addressed
          note: Task 3 binds NvimKey <M-T> matching the <M-N> row at shortcut.go:156; Task 5 drops the byte-form fallback and states the fix-at-spelling rule.
          round: 3
      blocked: false
content_hash: 5b4ef8d148a7f8ca97443d21a7587019dc5ca25b472fc28eda0747baa5d3fe76
---

# Gate ledger — pair#243 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-13T10:42:57-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Important] `chord-surface-inventory` Plan omits the zellij config.kdl bind rows that define which bytes Alt+Shift+t delivers
  Every Alt+letter global is captured by zellij and written as fixed KKP bytes (config.kdl:143 Alt D, :174 Alt d, :245 Alt N; reason at :236-238); arrows needed none. Add unbind/bind "Alt T" rows, rebuild the embedded bundle (drift_test.go:88), and write the full surface enumeration incl. namedChord at run.go:85-115.
- **PQ-2** [Important] `chord-encoding-provenance` Issue Plan says ESC[84;4u, plan file says ESC[116;4u; the ChordAltShiftD mirror is 84
  68;4u is uppercase D, so mirroring it gives 84;4u, matching the uppercase-plus-mod-4 precedent (68;4u, 78;4u) that the zellij WriteChars rows dictate. Reconcile both artifacts on 84;4u; 116;4u may be an additional form, not the only one.
- **PQ-3** [Minor] `global-chord-legacy-form` Legacy ESC T on a global chord reopens the ESC-ambiguity window under full-screen apps
  Globals are never passed through, so ESC then T within EscapeAmbiguity (shortcut.go:405-416) fires new-tab in nvim or the agent pane. Other letter globals carry only the KKP form; drop the legacy row.

## Round 2 — 2026-09-13T10:48:26-07:00 (claude) — BLOCKED

### Disposed

- PQ-1 — addressed — Task 4b adds the Alt T unbind/bind rows mirroring config.kdl:116/133/143 and names TestEmbeddedSourcesMatchTree (keyhelp/drift_test.go:88); Task 1 adds the namedChord case.
- PQ-2 — not-addressed — Prose is reconciled on 84;4u, but Task 4's pump test still delivers "\x1b[116;4u" and Task 5 Step 1 tells the implementer to add 116;4u as a second chordSequences row on a keymap mismatch. Rule: zellij's WriteChars row dictates the bytes; every consumer derives from it; never add a byte form to fix a consumer. Fix both lines.
- PQ-3 — addressed — Legacy ESC T dropped; Task 1 pins the KKP form as the only sequence, matching the ChordAltShiftN precedent at shortcut.go:383.

### Raised

- **PQ-4** [Minor] `chord-encoding-provenance` NvimKey should be the uppercase-letter form proven by the ChordAltShiftN row, not a guessed spelling with a byte-form fallback
  2nd finding in this family. The row at shortcut.go:156 binds 78;4u to NvimKey "<M-N>" and works live; the plan picks "<S-M-t>" and calls the match unprovable, then falls back to adding a 116;4u byte row. Rule: for a shift+letter global the NvimKey is the uppercase-letter form matching the existing "<M-N>" row, and a consumer-side mismatch is fixed at the consumer's spelling, never by adding a byte form. Two sites in the plan (Task 3 row, Task 5 fallback sentence).

## Round 3 — 2026-09-13T10:50:48-07:00 (claude) — passed

### Disposed

- PQ-2 — addressed — Issue Plan and plan file both carry \x1b[84;4u only, mirroring the 68;4u / 78;4u precedent; 116;4u is gone.
- PQ-4 — addressed — Task 3 binds NvimKey <M-T> matching the <M-N> row at shortcut.go:156; Task 5 drops the byte-form fallback and states the fix-at-spelling rule.

## Open findings

(none — every finding has been disposed)
