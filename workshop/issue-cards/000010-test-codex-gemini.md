---
id: '000010'
status: wontfix
created: 2026-05-02
updated: 2026-06-17
---

# test pair with codex and gemini

## Problem

`pair` was developed and tested entirely against `claude`. The architecture is agent-agnostic in design — the agent pane just runs `${PAIR_AGENT}` — but the keybindings (especially Alt+i for image attach and the copy-on-select reflow) make assumptions about how the TUI agent receives keystrokes and renders chips. Need to confirm these hold for codex and gemini.
