#!/usr/bin/env bash
# Invoked by zellij as copy_command — fires whenever a selection is finalized
# (mouse up after drag, or any keyboard-driven copy). Stdin = the selected
# text. We do three things:
#
#   1. Put the selection on the OS clipboard (zellij's default clipboard
#      behavior is replaced when copy_command is set, so we have to do this
#      ourselves to keep cross-app paste working).
#   2. If the selection happened in a pane other than the nvim "draft" pane,
#      flash that pane's bg via `zellij action set-pane-color` for a brief
#      visual cue at the source end of the copy. Best-effort: many TUIs paint
#      their own bg on every redraw, in which case the flash may be subtle or
#      invisible. The nvim-side flash on the inserted text is the primary cue.
#   3. Hand the selection off to clipboard-to-pane.sh, which writes it to
#      $PAIR_DATA_DIR/quote-<tag> and triggers nvim's PairPasteQuote() to
#      decide formatting (quote-mode vs inline) based on cursor position.
#
# When the selection happened *in* nvim, we skip steps 2 + 3 — otherwise it'd
# loop back and insert your own selection beneath itself.

LOG="${XDG_CACHE_HOME:-$HOME/.cache}/pair/clipboard-debug.log"
mkdir -p "$(dirname "$LOG")"

# Log immediately so we can tell if zellij is even invoking us.
{
    echo "=== copy-on-select INVOKED $(date) ==="
    echo "PAIR_HOME=${PAIR_HOME:-unset}"
    echo "ZELLIJ_SESSION_NAME=${ZELLIJ_SESSION_NAME:-unset}"
    echo "args: $*"
} >> "$LOG"

set -uo pipefail

sel=$(cat)
echo "sel bytes: ${#sel}" >> "$LOG"
if [ -z "$sel" ]; then
    echo "empty sel, exiting" >> "$LOG"
    exit 0
fi

# 1. Mirror to the OS clipboard so other apps see it.
if command -v pbcopy >/dev/null 2>&1; then
    printf '%s' "$sel" | pbcopy
elif command -v wl-copy >/dev/null 2>&1; then
    printf '%s' "$sel" | wl-copy
elif command -v xclip >/dev/null 2>&1; then
    printf '%s' "$sel" | xclip -selection clipboard -i
fi

# 2. Inspect the focused pane (where the selection happened). One jq pass
# extracts both the pane id and the in-nvim signal. Filter out plugin and
# floating panes — when a floating plugin (e.g. About Zellij) is open,
# zellij reports BOTH the plugin and the underlying terminal as
# is_focused=true; without the filter we'd pick the plugin and (a)
# misclassify in_nvim and (b) point set-pane-color at the wrong terminal
# id. flash-pane.sh applies the same filter when called with no args;
# passing $focused_id explicitly avoids a second jq round-trip.
#
# The in-nvim signal is the pane's terminal_command (the fixed launch
# string — "exec nvim … draft-<tag>.md" for the draft, "pair-wrap <agent>"
# for the agent), NEVER the title: the agent (e.g. claude) overwrites its
# pane title with "claude [<cwd>]", so a repo whose path contains "nvim"
# (e.g. parley.nvim) would match the nvim regex and misclassify the agent
# pane as the draft — copy-on-select would then think the selection was
# made in nvim and skip the paste entirely. terminal_command never embeds
# the cwd, so it stays clean. This mirrors clipboard-to-pane.sh, which
# resolves the draft pane the same way (terminal_command | test("nvim")).
in_nvim=false
focused_id=""
if command -v jq >/dev/null 2>&1; then
    focused=$(zellij action list-panes --json --command 2>/dev/null \
              | jq -r '[.. | objects
                        | select(.is_focused == true
                                 and (.is_plugin   // false) == false
                                 and (.is_floating // false) == false)][0]
                       | "\(.id // .pane_id // "")\t\(.terminal_command // "")"' 2>/dev/null)
    focused_id=${focused%%$'\t'*}
    cmd=${focused#*$'\t'}
    if printf '%s' "$cmd" | grep -qiE 'nvim|draft'; then
        in_nvim=true
    fi
fi

{
    echo "=== copy-on-select $(date) ==="
    echo "sel bytes: ${#sel}"
    echo "in_nvim: $in_nvim"
    echo "focused_id: ${focused_id:-(none)}"
} >> "$LOG"

if [ "$in_nvim" = true ]; then
    exit 0
fi
# Note: the "only insert when nvim is in insert mode" gate is now implemented
# entirely on the nvim side — clipboard-to-pane.sh sends Ctrl-_ which is
# mapped to PairPasteQuote *only in insert mode*. No shell-side mode probe
# is needed.

# 3. Flash the source pane's background so the user gets a visual cue at
# the selection site (in addition to the nvim-side flash on the inserted
# text). Delegated to bin/flash-pane.sh so the flash idiom (color, duration,
# bg-reset) lives in one place — see that script for the visibility caveat
# (best-effort: the TUI may repaint the bg).
if [ -n "$focused_id" ] && [ -x "$PAIR_HOME/bin/flash-pane.sh" ]; then
    "$PAIR_HOME/bin/flash-pane.sh" "$focused_id"
fi

# 4. Hand off to clipboard-to-pane.sh for the actual insert into nvim.
# It reads the OS clipboard (which we populated in step 1).
exec "$PAIR_HOME/bin/clipboard-to-pane.sh"
