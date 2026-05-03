---
id: 000010
status: open
deps: [000001]
created: 2026-05-02
updated: 2026-05-02
---

# test pair with codex and gemini

## Problem

`pair` was developed and tested entirely against `claude`. The architecture is agent-agnostic in design — the agent pane just runs `${PAIR_AGENT}` — but the keybindings (especially Alt+i for image attach and the copy-on-select reflow) make assumptions about how the TUI agent receives keystrokes and renders chips. Need to confirm these hold for codex and gemini.

## Spec

For each of `codex` and `gemini`:

1. **Launch:** `pair codex` and `pair gemini` work end-to-end (session creates, nvim drafting pane appears, agent runs in top pane).
2. **Send:** Alt+Return delivers nvim buffer to the agent's input and submits.
3. **Image attach (Alt+i):** put image on clipboard, press Alt+i, confirm the agent attaches the image and shows whatever its chip equivalent is. Document the agent's per-message image numbering scheme (does it match claude's `[Image #N]` convention, or differ?).
4. **Copy-on-select:** select text in the agent pane via mouse, confirm the selection reflows + quotes into nvim on mouse-up.
5. **Detach/reattach (Alt+d):** detach mid-session, re-launch `pair codex`, confirm the picker offers the detached session and reattach works.
6. **Quit (Alt+x):** full quit removes the session from the resurrect list.

Document any per-agent quirks discovered. Common likely sources:
- Different submit key (Enter vs. some other binding).
- Different image-attach semantics (chip rendering, numbering scheme).
- Different terminal-clear or escape-handling that breaks Esc-equivalent sequences.

If quirks need code paths, file follow-up issues per quirk.

## Plan

- [ ] Install codex and gemini CLI binaries.
- [ ] `pair codex` smoke test (steps 1–6 above).
- [ ] `pair gemini` smoke test (steps 1–6 above).
- [ ] Document quirks in atlas/architecture.md or in a new `atlas/agent-quirks.md`.
- [ ] File follow-up issues for any per-agent code paths needed.

## Log

### 2026-05-02

Filed as a punt-out from #000001. The v1 setup work is done; per-agent verification is its own scope.
