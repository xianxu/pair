# Architecture

## What pair is

A launcher that starts a zellij session with a fixed two-pane split. The top pane runs a TUI coding agent; the bottom pane runs Neovim on a persistent draft file. Keystrokes — and mouse-up after a selection — drive bidirectional flow between the panes via `zellij action write-chars` and `zellij action focus-pane-id`.

The whole thing is deliberately small — a handful of shell scripts, one nvim init, and two zellij KDL files. Required deps: `zellij`, `nvim`, `fzf`, `jq`, `par`, plus the agent itself.

## Pieces

```
bin/pair                     # entry point (launcher)
bin/clipboard-to-pane.sh     # reflow + > -prefix + write into nvim
bin/copy-on-select.sh        # invoked by zellij copy_command on mouse-up
bin/pair-quit.sh             # invoked by Alt+x — marks + kills session
nvim/init.lua                # bundled nvim config (loaded via -u)
zellij/config.kdl            # mouse, copy_command, keybinds
zellij/layouts/main.kdl      # the split + agent/draft commands
```

### `bin/pair` — launcher

Resolves `$PAIR_HOME` from its own real path (portable bash, no `readlink -f`), prepends `$PAIR_HOME/bin` to `$PATH` (idempotent across re-launches) so all helper scripts resolve by bare name in zellij configs and keybinds, sets `$PAIR_AGENT` from the positional argument (default `claude`) and `$PAIR_TAG` from `<agent>` or `<agent>-<variant>`, ensures `~/scratch/` exists, and dispatches:

**Decision tree.** Finds *all* detached pair-* sessions on the machine (any agent, any naming). Then:

- 0 detached → run create flow directly (validate agent, prompt for name, create).
- ≥1 detached → fzf picker over the detached sessions plus a `+ new <agent> session` sentinel. Pick a session → attach. Pick the sentinel → fall through to create.

The agent argument doesn't filter the picker — reattach is agent-agnostic (the existing session already runs whatever it runs). The agent argument only matters for the create path: it labels the sentinel, drives the auto-suggested default name, and is the binary that gets exec'd in the new session.

There is **no silent auto-attach**. Every reattach goes through the picker so the user explicitly sees what they're connecting to.

Detection of attached-vs-detached uses `zellij --session NAME action list-clients`, which prints a header plus one row per connected client. Zero rows = detached.

**Naming prompt.** When the create flow runs, the launcher prompts the user with the auto-suggested name as the default (`Session name [pair-claude-2]:`). Pressing Enter accepts; typing a custom name (`pair-bugfix`, or just `bugfix`) overrides it.

**Agent validation deferred.** `command -v "$AGENT"` runs only inside the create branch, not at startup, so attaching to a custom-named session whose tag isn't a real binary still works.

**Title.** The launcher emits an OSC 0 escape sequence right before invoking zellij, so the terminal title shows the session name on both create and attach paths (zellij itself only sets it on create).

**Cleanup on quit.** zellij is run as a child (not `exec`) so the launcher resumes when zellij exits. On resume it checks for `~/.cache/pair/quit-<session>` (the marker that `pair-quit.sh` writes when Alt+x fires) and, if present, runs `zellij delete-session --force <session>` to clear the resurrect entry. No marker → leave the session as zellij left it (running if detached, EXITED-resurrectable if Ctrl+q).

### `zellij/layouts/main.kdl` — pane split

Horizontal split. Top pane runs `$PAIR_AGENT` (auto-fills remaining height). Bottom pane is a fixed 10 rows running `nvim -u $PAIR_HOME/nvim/init.lua` on the per-tag draft file.

Both panes wrap their command in `sh -c "..."` so the shell expands `$PAIR_AGENT`, `$PAIR_TAG`, and `$PAIR_HOME` at exec time — zellij itself does not interpolate env vars in `command`/`args` fields.

The bottom pane has `focus=true` (drafting pane gets focus on launch) and a `name=` set to the help string (`Alt: ⏎=send  u=max  i=img  d=detach  x=quit`) so zellij renders that as the pane's frame title.

### `zellij/config.kdl` — mouse, copy, keybinds

Top-level config:

- `mouse_click_through true` — first click on an unfocused pane goes through to the pane (so click-and-drag selects in one motion) instead of being consumed by zellij just to change focus.
- `copy_command "copy-on-select.sh"` — on every selection finalize (mouse-up after drag), zellij pipes the selected text to this script. `copy_command` replaces zellij's default OS-clipboard write, so the script does that part too. Resolved by PATH (which `bin/pair` populated).

Keybinds added on top of zellij defaults (`clear-defaults=false`):

- `unbind "Alt i"` — release Alt+i (zellij's default binds it to MoveTab; we want nvim to see it for image attach).
- `Alt+d` — `Detach` — detach from the session.
- `Alt+u` — `MoveFocus Down; ToggleFocusFullscreen` — toggle nvim pane fullscreen, regardless of which pane has focus.
- `Alt+x` — `Run "pair-quit.sh"` — full quit (writes marker, kills session).

Alt+n (clipboard → nvim quote) used to be a manual keybind here too, but became redundant once `copy_command` started auto-firing on mouse-up. Removed.

### `bin/clipboard-to-pane.sh` — reflow + write into nvim

Read OS clipboard (`pbpaste` / `wl-paste` / `xclip`). Reflow paragraphs with `par 1000` (par's width arg caps at 9999; 1000 is plenty for any soft-wrapping target). Prefix every line with `> `.

Find the nvim pane via `zellij action list-panes --json`, looking for the pane whose `terminal_command` contains `nvim`. Focus it via `zellij action focus-pane-id <id>` — this is critical because the script runs inside a transient `Run` pane (when invoked directly) or as a child of the zellij server (when invoked via `copy_command`), and we cannot rely on positional `move-focus` to land on nvim.

Once focus is on nvim: send `Ctrl-\ Ctrl-N` (force normal mode), then `i` (enter insert), then the quoted body, then a trailing blank line. The Ctrl-\ Ctrl-N sequence is critical — the obvious alternative `Esc` + `i` is interpreted by terminals as the encoding for `Alt+i`, which would fire nvim's `attach_image` keymap and insert a stray `[Image #N]` chip. Ctrl-\ Ctrl-N has no Alt-encoding ambiguity.

Diagnostic log lives at `~/scratch/pair-clipboard-debug.log` (overwritten each invocation).

### `bin/copy-on-select.sh` — zellij copy_command wrapper

Receives selected text on stdin from zellij. Mirrors the text to the OS clipboard (zellij's default clipboard write is bypassed when `copy_command` is set, so this is mandatory). Then checks if the focused pane (where the selection happened) is the nvim draft pane; if so, exits without further action (selecting in nvim shouldn't loop back). Otherwise execs `clipboard-to-pane.sh` to do the reflow + insert.

Pane detection: parse `list-panes --json --command`, find the focused pane, check if its `title` or `terminal_command` matches `nvim|draft`.

### `bin/pair-quit.sh` — Alt+x handler

Touches the marker file `~/.cache/pair/quit-$ZELLIJ_SESSION_NAME`, then `exec zellij kill-session $ZELLIJ_SESSION_NAME`. The kill terminates the session including the script itself; on the launcher side, `bin/pair` resumes, sees the marker, and runs `delete-session --force` to clean up the resurrect entry.

### `nvim/init.lua` — drafting buffer config

Loaded via `nvim -u`, fully isolated from the user's main nvim config. Provides:

- Drafting-friendly defaults: no line numbers, wrap, linebreak, breakindent, spell, persistent undo under `~/.local/share/pair/undo/`, `laststatus=0` and `cmdheight=0` to maximize editing space.
- `<M-CR>` (Alt+Return, normal+insert) — `send_and_clear`: append buffer to log, send to agent pane via `zellij action focus-pane-id` + `write-chars` + Enter, clear buffer, save, drop into insert mode.
- `<M-i>` (Alt+i, normal+insert) — `attach_image`: increment per-session counter, send Ctrl+V to the agent pane (claude reads OS clipboard, attaches image), insert `[Image #N]` at cursor. If cursor is on an existing `[Image #N]`, sync the counter to N instead.
- `<leader>cs` — `send_section`: send only the section between nearest `---` markers.
- `<leader>cp` — `paste_and_reflow`: paste clipboard at cursor through `par 1000`.
- Autosave on `BufLeave`, `FocusLost`, `InsertLeave` so disk and buffer agree.

## Quit semantics

Two ways to end a session, with different aftermath:

- **Ctrl+q** (zellij default) — kills the session, leaves it in the resurrect list. `pair pick` will surface it under `+ new` since it counts as a detached/EXITED candidate.
- **Alt+x** (pair-specific) — kills the session AND removes the resurrect entry. After Alt+x, the session is fully gone.

Both work; pick based on whether you want the option to come back to that exact session shell later.

## Data layout

Drafts and prompt history live in `~/scratch/`, keyed by tag (the agent name, or `<agent>-<variant>`, or a custom name from the prompt):

- `pair-draft-<tag>.md` — the active draft file. Cleared by `send_and_clear`, persists across launches.
- `pair-log-<tag>.md` — append-only log of every send, with timestamp. Searchable via `rg`.

Per-tag files mean `pair claude`, `pair codex`, `pair claude work`, and `pair claude bugfix` (custom name) all have independent draft state.

Internal: `~/.cache/pair/quit-<session>` — marker file used to communicate "user asked for full quit" between `pair-quit.sh` and the launcher. Touched on Alt+x, removed by the launcher after delete-session.

## Path resolution

`bin/pair` prepends `$PAIR_HOME/bin` to `$PATH` before exec'ing zellij. zellij and all its child processes (panes, copy_command, Run actions) inherit the PATH and can resolve `clipboard-to-pane.sh`, `copy-on-select.sh`, `pair-quit.sh` by bare name. This lets the zellij KDL configs reference scripts without `sh -c` env-var quoting hacks.

## Design intent

- **Asymmetric panes by design.** Most chat UIs cram input and output into the same constrained box. The split makes the asymmetry explicit — agent owns *output*, nvim owns *input* — and lets each side specialize.
- **Selection is the gesture.** Click-and-drag in the agent pane, mouse up — the quote is in nvim, ready for your reaction. No keystroke between.
- **Self-contained.** Uses `--config-dir` and `nvim -u` to fully isolate from the user's normal configs. No invasive install.
- **Agent-agnostic.** Same plumbing works for any TUI agent that accepts typed input. Switching is one keystroke.
- **Prompt history is just a markdown file.** Aligns with the "data into central location, shell-ed agent runs free" pattern: every send appends to a grep-able log.

## Future work

Tracked in workshop issues. v2 candidates include a real nvim plugin (for users who want LSP/snippets/telescope inside the input pane), a homebrew formula, and per-agent shims if Codex/Gemini have submit semantics that differ from Claude Code.
