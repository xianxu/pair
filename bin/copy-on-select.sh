#!/usr/bin/env bash
# Invoked by zellij as copy_command — fires whenever a selection is finalized
# (mouse up after drag, or any keyboard-driven copy). Stdin = the selected
# text. We do two things:
#
#   1. Put the selection on the OS clipboard (zellij's default clipboard
#      behavior is replaced when copy_command is set, so we have to do this
#      ourselves to keep cross-app paste working).
#   2. If the selection happened in a pane other than the nvim "draft" pane,
#      reflow, > -prefix, focus nvim, insert at cursor (via clipboard-to-
#      pane.sh). This collapses select-in-claude → quote-into-draft into a
#      single motion — no keystroke between selection and insert.
#
# When the selection happened *in* nvim, we skip step 2 — otherwise it'd
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

# 2. Detect if the focused pane (where the selection happened) is the nvim
# draft pane. If so, skip the auto-insert.
in_nvim=false
if command -v jq >/dev/null 2>&1; then
    sig=$(zellij action list-panes --json --command 2>/dev/null \
          | jq -r '[.. | objects | select(.is_focused == true)][0]
                   | "\(.title // "") \(.terminal_command // "")"' 2>/dev/null)
    if printf '%s' "$sig" | grep -qiE 'nvim|draft'; then
        in_nvim=true
    fi
fi

{
    echo "=== copy-on-select $(date) ==="
    echo "sel bytes: ${#sel}"
    echo "in_nvim: $in_nvim"
} >> "$LOG"

if [ "$in_nvim" = true ]; then
    exit 0
fi

# Selection was in the agent pane (or some other pane) — quote it into nvim.
# clipboard-to-pane.sh reads the OS clipboard (which we just populated above).
exec "$PAIR_HOME/bin/clipboard-to-pane.sh"
