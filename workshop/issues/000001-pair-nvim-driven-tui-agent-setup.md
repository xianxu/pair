---
id: 000001
status: open
deps: []
created: 2026-05-02
updated: 2026-05-02
---

# pair — nvim-driven TUI coding agent setup

## Problem

The input box in every TUI coding agent (Claude Code, Codex, Gemini CLI) is cramped, lacks editing power (no real undo, no search/replace, no snippets, no syntax highlighting), and conflates the input affordance with the output affordance in the same scrollable region. Composing non-trivial prompts there is friction; reviewing output while drafting the next message is friction. The TUI vendors will not fix this — it's not their layer to fix.

## Spec

Full design captured as a pensive in the brain repo: `~/workspace/brain/docs/vision/2026-05-02-01-pensive-nvim-as-input-field-for-tui-coding-agents.md`.

Pattern, briefly: split a zellij session into two panes. Top pane (~65%) runs the TUI agent — owns the *output* affordance. Bottom pane (~35%) runs nvim on a persistent draft file — owns the *input* affordance. Keystrokes drive bidirectional flow via `zellij action write-chars`. Universal across agents — Claude first, Codex/Gemini follow with a flag flip.

**Tool name:** `pair`. Agent passed positionally: `pair claude`, `pair codex`, `pair gemini`. Default = `claude`.

**Project location:** this repo (`~/workspace/pair/`).

**Packaging approach:** modular local layout for v1 — files under `bin/`, `nvim/`, `zellij/`, launcher script that points zellij at the bundled config dir and layout, nothing in the user's `~/.config/{nvim,zellij}` is touched. Brew/install packaging deferred to v2.

**Key bindings (Alt+ family for global, leader for nvim-local):**

| Key | Scope | Action |
|---|---|---|
| **Alt+u** | zellij (any pane) | Toggle nvim pane fullscreen (works whether focus is in claude or nvim) |
| **Alt+n** | zellij (any pane) | Reverse-paste: clipboard → nvim cursor, reflowed via `par` and prefixed with `> ` (quote block), trailing blank line, focus stays in nvim |
| **Alt+Return** | nvim (normal/insert) | Send buffer to agent + Enter, log to history file, clear buffer, save, drop into insert mode |
| **`<leader>cs`** | nvim | Send only the section between nearest `---` markers |
| **`<leader>cp`** | nvim | Paste-and-reflow at cursor (raw paste, no quoting) |

**Draft file:** persistent at `~/scratch/pair-draft-<agent>.md` (per-agent, so claude/codex/gemini drafts don't collide). Cleared on send. History appended to `~/scratch/pair-log-<agent>.md` with timestamp before clear — preserves grep-able prompt history.

**Why both clear-and-save and log:** clearing without saving leaves disk in pre-send state (next launch shows old prompt — confusing). Saving an empty file makes the buffer state and disk state agree. Logging before clearing means we don't lose the prompt — it's just moved from the active draft to the historical log. Aligns with the "data into central location, shell-ed agent runs free" thesis.

**Image paste:** Ctrl+V in the agent pane reads OS clipboard directly. Document the macOS recipe (`osascript ... «class PNGf»`) for putting images on the clipboard. Linux: `wl-copy --type image/png < file` or X11 equivalent. Out of scope for v1 to wrap this — users handle it themselves.

**Out of scope for v1:**
- Brew formula / install scripts (manual symlink for now).
- Hammerspoon alternative for Alt+u/Alt+n (zellij-internal binds work; only needed if those flicker or fight other tools).
- Nvim plugin manager integration — v1 uses `nvim -u <bundled-init.lua>` which fully isolates from user's main config.
- Fancy nvim setup inside the input pane (LSP, snippets, telescope). v1 ships a deliberately minimal init.lua. Power users get richer integration in v2 by installing a real `pair.nvim` plugin into their normal config.

## Plan

- [ ] Create directory structure: `bin/`, `nvim/`, `zellij/layouts/`
- [ ] `zellij/layouts/main.kdl` — horizontal split, top pane runs `$PAIR_AGENT`, bottom pane runs `nvim -u $PAIR_HOME/nvim/init.lua` on the per-agent draft file
- [ ] `zellij/config.kdl` — Alt+u (`MoveFocus Down; TogglePaneFullscreen`), Alt+n (Run `clipboard-to-pane.sh`)
- [ ] `bin/clipboard-to-pane.sh` — `pbpaste | par -w99999 | sed 's/^/> /'` then write to focused-down pane with trailing blank
- [ ] `nvim/init.lua` — `send_and_clear` (Alt+Return), `send_section` (`<leader>cs`), `paste_and_reflow` (`<leader>cp`), log-before-clear helper
- [ ] `bin/pair` launcher — positional agent arg, sets `PAIR_HOME` and `PAIR_AGENT`, execs zellij with `--config-dir`/`--layout`/`--session pair-<agent>`
- [ ] Verify Alt+u works from both panes
- [ ] Verify Alt+n produces correctly-quoted, correctly-reflowed paste in nvim
- [ ] Verify Alt+Return sends, logs, clears, leaves cursor in insert mode in empty buffer
- [ ] Test `pair claude` end-to-end
- [ ] Test `pair codex` — confirm submit semantics, paste behavior, image-paste support if applicable
- [ ] Test `pair gemini` — same checks
- [ ] Document any per-agent quirks discovered
- [ ] Write README with install steps (manual symlink), keybind summary, image-paste recipes per OS
- [ ] Iterate on annoyances surfaced during a week of real use before considering v2 packaging

## Log

### 2026-05-02

Created. Spec consolidated from conversation thread (in brain repo) that produced the pensive. This is the first issue in the pair repo.
