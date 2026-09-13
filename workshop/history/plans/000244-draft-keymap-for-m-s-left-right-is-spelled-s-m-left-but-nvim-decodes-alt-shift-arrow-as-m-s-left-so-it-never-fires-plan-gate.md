---
gate: plan-quality
issue: 244
id_prefix: PQ
rounds:
    - "n": 1
      timestamp: "2026-09-13T11:33:50-07:00"
      agent: claude
      findings:
        - id: PQ-1
          severity: Critical
          title: 'Root cause is refuted: nvim canonicalizes <S-M-Left> to <M-S-Left>, so the planned spelling change is a no-op'
          detail: 'Verified on nvim v0.11.7 headless: a map set as <S-M-Left> lists its lhs as <M-S-Left>, fires on fed <M-S-Left>, and setting both spellings leaves one keymap (the issue''s g:a=0 g:b=1 is an overwrite artifact). The plan needs a diagnosis step capturing the bytes the draft nvim actually receives live; test first the hypothesis that the terminal emits the meta-style \x1b[1;10D, which pair term accepts at shortcut.go:408 but nvim''s TUI decodes without the meta bit, while letter globals are immune because zellij synthesizes their bytes.'
          family: unbacked-claim-existing-behavior
          round: 1
        - id: PQ-2
          severity: Important
          title: The planned probe (bare nvim + \x1b[1;4D) passes on main today and cannot fail for this bug or its regression
          detail: Once live bytes are known, drive the real generated table via install_global_maps with every byte form chordSequences accepts for ChordAltShiftLeft/Right (the ;4 and ;10 forms, plus split arrival), asserting each fires (ARCH-PURPOSE, ARCH-MOCK). State that the pty probe runs outside the agent sandbox, where pty spawn is denied.
          family: probe-tests-wrong-seam
          round: 1
        - id: PQ-3
          severity: Minor
          title: workbench_route_test.lua:33-34 and atlas/architecture.md:1057 restate the spelling and are not in the plan; no non-goals stated
          detail: 'If the spelling changes, the Lua test fails loudly and the atlas drifts silently. Add the #227 passthrough of typed Alt+Left as an explicit non-goal.'
          family: non-deriving-restatement-unswept
          round: 1
      blocked: true
---

# Gate ledger — pair#244 (plan-quality)

Findings this gate raised, the stable ids the binary assigned them, and how
later rounds disposed of them. Generated — edit the gate, not this file.

## Round 1 — 2026-09-13T11:33:50-07:00 (claude) — BLOCKED

### Raised

- **PQ-1** [Critical] `unbacked-claim-existing-behavior` Root cause is refuted: nvim canonicalizes <S-M-Left> to <M-S-Left>, so the planned spelling change is a no-op
  Verified on nvim v0.11.7 headless: a map set as <S-M-Left> lists its lhs as <M-S-Left>, fires on fed <M-S-Left>, and setting both spellings leaves one keymap (the issue's g:a=0 g:b=1 is an overwrite artifact). The plan needs a diagnosis step capturing the bytes the draft nvim actually receives live; test first the hypothesis that the terminal emits the meta-style \x1b[1;10D, which pair term accepts at shortcut.go:408 but nvim's TUI decodes without the meta bit, while letter globals are immune because zellij synthesizes their bytes.
- **PQ-2** [Important] `probe-tests-wrong-seam` The planned probe (bare nvim + \x1b[1;4D) passes on main today and cannot fail for this bug or its regression
  Once live bytes are known, drive the real generated table via install_global_maps with every byte form chordSequences accepts for ChordAltShiftLeft/Right (the ;4 and ;10 forms, plus split arrival), asserting each fires (ARCH-PURPOSE, ARCH-MOCK). State that the pty probe runs outside the agent sandbox, where pty spawn is denied.
- **PQ-3** [Minor] `non-deriving-restatement-unswept` workbench_route_test.lua:33-34 and atlas/architecture.md:1057 restate the spelling and are not in the plan; no non-goals stated
  If the spelling changes, the Lua test fails loudly and the atlas drifts silently. Add the #227 passthrough of typed Alt+Left as an explicit non-goal.

## Open findings

- **PQ-1** [Critical] `unbacked-claim-existing-behavior` Root cause is refuted: nvim canonicalizes <S-M-Left> to <M-S-Left>, so the planned spelling change is a no-op
- **PQ-2** [Important] `probe-tests-wrong-seam` The planned probe (bare nvim + \x1b[1;4D) passes on main today and cannot fail for this bug or its regression
- **PQ-3** [Minor] `non-deriving-restatement-unswept` workbench_route_test.lua:33-34 and atlas/architecture.md:1057 restate the spelling and are not in the plan; no non-goals stated
