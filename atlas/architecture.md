# Architecture

## What pair is

A launcher that starts a zellij session with a fixed two-pane split. The top pane runs a TUI coding agent; the bottom pane runs Neovim on a persistent draft file. Keystrokes drive bidirectional flow between the panes via `zellij action write-chars`.

The whole thing is deliberately small — five files, no dependencies beyond `zellij`, `nvim`, and the agent itself. `par` is recommended for paragraph reflow but optional.

## Pieces

```
bin/pair                     # entry point (launcher)
bin/clipboard-to-pane.sh     # helper for Alt+n
nvim/init.lua                # bundled nvim config (loaded via -u)
zellij/config.kdl            # zellij keybinds (Alt+u, Alt+n)
zellij/layouts/main.kdl      # the 65/35 split + agent/draft commands
```

### `bin/pair` — launcher

Resolves `$PAIR_HOME` from its own real path (portable bash, no `readlink -f`), sets `$PAIR_AGENT` from the positional argument (default `claude`) and `$PAIR_TAG` from the agent + optional variant, checks that the agent and zellij are installed, ensures `~/scratch/` exists, and dispatches:

**Decision tree.** For `pair <agent> [variant]`, the launcher finds *all* detached pair-* sessions on the machine (any agent, any naming). Then:

- 0 detached → run create flow directly (validate agent, prompt for name, create).
- ≥1 detached → fzf picker over the detached sessions plus a `+ new <agent> session` sentinel. Pick a session → attach. Pick the sentinel → fall through to create.

The agent argument doesn't filter the picker — only attached sessions you might want to reattach to are useful, and reattach is agent-agnostic (the existing session already runs whatever it runs). The agent argument only matters for the create path: it labels the sentinel, drives the auto-suggested default name, and is the binary that gets exec'd in the new session.

There is **no silent auto-attach**. Every reattach goes through the picker so the user explicitly sees what they're connecting to. (Long-lived sessions make silent attach surprising; explicit confirmation is the right default.)

Detection of attached-vs-detached is via `zellij --session NAME action list-clients`, which prints a header plus one row per connected client. Zero rows = detached.

**Naming prompt.** When the create flow runs, the launcher prompts the user with the auto-suggested name as the default (e.g. `Session name [pair-claude-2]:`). Pressing Enter accepts; typing a custom name like `pair-bugfix` (or just `bugfix`) overrides it. Custom-named sessions show up in the picker as long as they share the agent prefix; `pair-blogging` (no agent prefix) only matches if you re-invoke as `pair blogging`.

**Agent validation deferred.** `command -v "$AGENT"` runs only inside the create branch, not at startup, so attaching to a custom-named session whose tag isn't a real command (like `pair-blogging` via `pair blogging`) still works.

**Title.** The launcher emits an OSC 0 escape sequence right before `exec zellij`, so the terminal title shows the session name on both create and attach paths (zellij itself only sets it on create).

### `zellij/layouts/main.kdl` — pane split

Horizontal split, top 65% / bottom 35%. Both panes wrap their command in `sh -c "..."` so the shell expands `$PAIR_AGENT` and `$PAIR_HOME` at exec time — zellij itself does not interpolate env vars in `command`/`args` fields.

The bottom pane gets `focus=true` so the drafting pane has focus on launch.

### `zellij/config.kdl` — keybinds

Two binds added on top of zellij's defaults (`clear-defaults=false`):

- `Alt+u` — `MoveFocus Down; ToggleFocusFullscreen` — toggles nvim pane fullscreen regardless of which pane has focus. `MoveFocus Down` is a no-op when already at the bottom.
- `Alt+n` — runs `bin/clipboard-to-pane.sh` via `Run`. Opens a transient pane that closes on script exit (brief flicker; acceptable for v1).

### `nvim/init.lua` — drafting buffer config

Loaded via `nvim -u`, fully isolated from the user's main nvim config. Provides:

- Drafting-friendly defaults: no line numbers, wrap, linebreak, breakindent, spell, persistent undo under `~/.local/share/pair/undo/`.
- `<M-CR>` (Alt+Return, normal+insert) — `send_and_clear`: append buffer to log, send to agent pane via `zellij action write-chars` + Enter, clear buffer, save, drop into insert mode.
- `<leader>cs` — `send_section`: send only the section between nearest `---` markers.
- `<leader>cp` — `paste_and_reflow`: paste clipboard at cursor through `par` for paragraph reflow.
- Autosave on `BufLeave`, `FocusLost`, `InsertLeave` so disk and buffer agree.

### `bin/clipboard-to-pane.sh` — reverse-paste helper

OS-aware clipboard read (pbpaste / wl-paste / xclip), reflows with `par` if available, prefixes every line with `> ` to make it a markdown quote, and writes into the pane below current focus (the draft pane in pair's layout). Adds a trailing blank line for the user's reaction. Focus stays in nvim.

## Data layout

Drafts and prompt history live in `~/scratch/`:

- `pair-draft-<agent>.md` — the active draft file. Cleared by `send_and_clear`, persists across launches.
- `pair-log-<agent>.md` — append-only log of every send, with timestamp. Searchable via `rg`.

Per-agent files mean `pair claude`, `pair codex`, and `pair gemini` don't fight over the same draft.

## Design intent

- **Asymmetric panes by design.** Most chat UIs cram input and output into the same constrained box. The split makes the asymmetry explicit — agent owns *output*, nvim owns *input* — and lets each side specialize.
- **Self-contained.** v1 uses `--config-dir` and `nvim -u` to fully isolate from the user's normal configs. No invasive install.
- **Agent-agnostic.** Same plumbing works for any TUI agent that accepts typed input. Switching is one keystroke.
- **Prompt history is just a markdown file.** Aligns with the "data into central location, shell-ed agent runs free" pattern: every send appends to a grep-able log.

## Future work

Tracked in workshop issues. v2 candidates include a real nvim plugin (for users who want LSP/snippets/telescope inside the input pane), a homebrew formula, and per-agent shims if Codex/Gemini have submit semantics that differ from Claude Code.
