#!/usr/bin/env bash
# Pull whatever is on the OS clipboard, reflow paragraph wraps, prefix every
# line with "> " (markdown quote), and inject into the nvim draft pane.
#
# Triggered by Alt+n via zellij keybind. zellij's `Run` action spawns a
# transient pane that grabs focus, so we cannot rely on positional `move-focus`
# to find nvim. Instead, we look up the nvim pane by its layout name ("draft")
# via `zellij action list-panes --json` and target it explicitly with
# `zellij action focus-pane-id`.
#
# Diagnostic log: ~/scratch/pair-clipboard-debug.log (overwritten each invocation).

set -uo pipefail

LOG="$HOME/scratch/pair-clipboard-debug.log"
mkdir -p "$(dirname "$LOG")"
{
    echo "=== $(date) ==="
    echo "ZELLIJ_SESSION_NAME=${ZELLIJ_SESSION_NAME:-unset}"
    echo "ZELLIJ_PANE_ID=${ZELLIJ_PANE_ID:-unset}"
    echo "PAIR_HOME=${PAIR_HOME:-unset}"
} > "$LOG"

# --- clipboard read --------------------------------------------------------
if command -v pbpaste >/dev/null 2>&1; then
    clip="$(pbpaste)"
elif command -v wl-paste >/dev/null 2>&1; then
    clip="$(wl-paste --no-newline)"
elif command -v xclip >/dev/null 2>&1; then
    clip="$(xclip -selection clipboard -o)"
else
    echo "ERROR: no clipboard tool found" >> "$LOG"
    exit 1
fi
echo "clipboard bytes: ${#clip}" >> "$LOG"
[ -z "$clip" ] && { echo "empty clipboard, exiting" >> "$LOG"; exit 0; }

# --- reflow + quote --------------------------------------------------------
# par may exit non-zero on weird input; fall back to raw rather than die.
if command -v par >/dev/null 2>&1; then
    if reflowed=$(printf '%s' "$clip" | par 1000 2>>"$LOG"); then
        echo "par: ok" >> "$LOG"
    else
        echo "par: failed, using raw clipboard" >> "$LOG"
        reflowed="$clip"
    fi
else
    echo "par: not installed, using raw clipboard" >> "$LOG"
    reflowed="$clip"
fi

quoted=$(printf '%s' "$reflowed" | sed 's/^/> /')

# --- find nvim's pane id by layout name "draft" ---------------------------
# `list-panes --json` returns an array of pane objects with id and name.
# We named the nvim pane in zellij/layouts/main.kdl with `name="draft"`.
nvim_id=""
panes_json=""
if command -v jq >/dev/null 2>&1; then
    panes_json=$(zellij action list-panes --json 2>>"$LOG" || true)
    echo "--- list-panes --json ---" >> "$LOG"
    printf '%s\n' "$panes_json" >> "$LOG"
    echo "--- end list-panes ---" >> "$LOG"
    # Walk all panes and find the one running nvim. Match on terminal_command
    # rather than title — the title is the layout's `name` attribute, which
    # we use for displaying help text in the pane frame, not as an identifier.
    nvim_id=$(printf '%s' "$panes_json" \
              | jq -r '[.. | objects
                        | select(.terminal_command != null
                                 and (.terminal_command | test("nvim")))][0]
                       | (.id // .pane_id // empty)' 2>>"$LOG")
fi
echo "resolved nvim pane id: '${nvim_id:-(none)}'" >> "$LOG"

# --- target nvim and write ------------------------------------------------
if [ -n "$nvim_id" ]; then
    # Pane IDs from list-panes can be bare integers or `terminal_<N>`. Try
    # both forms because zellij's parser accepts either.
    zellij action focus-pane-id "$nvim_id" 2>>"$LOG" \
        || zellij action focus-pane-id "terminal_$nvim_id" 2>>"$LOG" \
        || echo "WARN: focus-pane-id failed for '$nvim_id'" >> "$LOG"
else
    echo "WARN: could not resolve 'draft' pane; falling back to move-focus down" >> "$LOG"
    zellij action move-focus down 2>>"$LOG"
fi

# Ensure nvim is in insert mode before typing the body — otherwise the first
# character ('>') would be interpreted as a normal-mode command.
#
# We DON'T use Esc + 'i' here, because in terminal-land `Esc` followed by `i`
# is the exact encoding for `Alt+i` — and our nvim keymap binds <M-i> to
# attach_image, which inserts a stray `[Image #N]` at the cursor. Instead,
# use nvim's "force normal mode" sequence Ctrl-\ Ctrl-N, then `i`. Ctrl-\
# is 0x1c (28), Ctrl-N is 0x0e (14). No Esc, no Alt-encoding ambiguity.
zellij action write 28 2>>"$LOG"          # Ctrl-\
zellij action write 14 2>>"$LOG"          # Ctrl-N → normal mode (any context)
zellij action write-chars 'i' 2>>"$LOG"   # enter insert mode at cursor
zellij action write-chars "$quoted"$'\n\n' 2>>"$LOG"
echo "wrote $((${#quoted} + 2)) bytes" >> "$LOG"
