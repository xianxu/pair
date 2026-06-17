#!/usr/bin/env bash
# Pull whatever is on the OS clipboard and inject it into the nvim draft
# pane. Formatting (par reflow, `> ` quote prefix) is decided in nvim, not
# here, so this script just hands off the raw selection.
#
# Invoked from copy-on-select.sh (which is wired to zellij's copy_command,
# firing on every selection finalize). zellij's child processes don't run
# in a stable layout position, so we cannot rely on positional `move-focus`
# to find nvim. Instead, we look up the nvim pane by its terminal_command
# via `zellij action list-panes --json` and target it explicitly with
# `zellij action focus-pane-id`.
#
# Once nvim is focused, we hand the actual insert off to a Lua function
# (PairPasteQuote in nvim/init.lua) by writing the raw clipboard body to a
# temp file at $PAIR_DATA_DIR/quote-<tag> and then triggering
# `:lua PairPasteQuote()<CR>` in the pane. The Lua side dispatches on
# cursor column: col==0 → quote-mode (par reflow + `> ` prefix + scroll
# first line to top + flash + cursor on empty line below); col>0 →
# inline-mode (insert verbatim at cursor + flash, no scroll).
#
# Diagnostic log: ${XDG_CACHE_HOME:-~/.cache}/pair/clipboard-debug.log
# (overwritten each invocation).

set -uo pipefail

LOG="${XDG_CACHE_HOME:-$HOME/.cache}/pair/clipboard-debug.log"
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

# --- stage the raw selection for nvim to read ------------------------------
# We hand off the raw clipboard body — par reflow and `> ` prefixing happen
# in nvim now, conditional on cursor position (col==0 → quote-mode with
# reflow + prefix; otherwise → inline paste verbatim). Keeping formatting
# decisions on the nvim side keeps this script source-agnostic.
data_dir="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
mkdir -p "$data_dir"
tag="${PAIR_TAG:-${PAIR_AGENT:-claude}}"
quote_file="$data_dir/quote-$tag"
printf '%s' "$clip" > "$quote_file"
echo "staged selection at: $quote_file (${#clip} bytes)" >> "$LOG"

# --- target nvim and trigger PairPasteQuote -------------------------------
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

# Trigger PairPasteQuote via a single Ctrl-_ (ASCII 31). On the nvim side,
# `<C-_>` is mapped to PairPasteQuote *only in insert mode* — that mapping
# IS the gate: if the user is in normal mode (e.g. browsing prompt history
# with Alt+←/→), Ctrl-_ hits nvim's default (a near-no-op revins toggle)
# and the buffer isn't touched. So we don't force-normal-mode here; doing
# so would destroy the very mode signal that drives the gate.
zellij action write 31 2>>"$LOG"                              # Ctrl-_
echo "triggered PairPasteQuote (Ctrl-_)" >> "$LOG"
