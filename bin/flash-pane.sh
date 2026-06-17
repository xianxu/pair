#!/usr/bin/env bash
# Flash a zellij pane's background as a brief visual cue. Used at
# focus-changing moments where the user might miss the change otherwise:
# the focus-loss return timer (nvim auto-yanks focus back after 10s) is
# the canonical example, in the same idiom as copy-on-select.sh's
# selection-site flash.
#
# Usage: flash-pane.sh [<pane-id>]
#
# With no arg, flashes the currently focused pane. The fg phase (set bg)
# runs synchronously so callers can chain a focus-change immediately
# after; the cleanup (reset bg after $PAIR_FLASH_MS) is backgrounded and
# disowned so it survives the caller exiting.
#
# Override the color via $PAIR_FLASH_BG, the duration via $PAIR_FLASH_MS.

set -uo pipefail

LOG="${XDG_CACHE_HOME:-$HOME/.cache}/pair/clipboard-debug.log"
mkdir -p "$(dirname "$LOG")"

pane_id="${1:-}"
if [ -z "$pane_id" ] && command -v jq >/dev/null 2>&1; then
    # Filter out plugin and floating panes: zellij reports BOTH a focused
    # floating plugin (e.g. "About Zellij") AND the underlying terminal as
    # is_focused=true, and `[..|objects|select(...)]` would pick the plugin
    # first. set-pane-color then resolves bare "<n>" as "terminal_<n>" — so
    # picking plugin id=1 silently flashes the wrong terminal (nvim, which
    # also has id=1 in the terminal id-space). Explicitly require a
    # non-plugin, non-floating pane so we land on the actual focused
    # terminal.
    pane_id=$(zellij action list-panes --json --command 2>/dev/null \
              | jq -r '[.. | objects
                        | select(.is_focused == true
                                 and (.is_plugin   // false) == false
                                 and (.is_floating // false) == false)][0]
                       | .id // .pane_id // ""' 2>/dev/null)
fi
if [ -z "$pane_id" ]; then
    echo "flash-pane: no pane id $(date)" >> "$LOG"
    exit 0
fi

# Default to dracula's "green" (the active-frame color in the user's zellij
# theme) so the flash visually echoes the pane border. Overridable via
# $PAIR_FLASH_BG; if you swap themes, change the default here to match.
flash_bg="${PAIR_FLASH_BG:-#50fa7b}"
flash_ms="${PAIR_FLASH_MS:-100}"

zellij action set-pane-color --pane-id "$pane_id" --bg "$flash_bg" 2>>"$LOG"
(
    secs=$(awk "BEGIN{printf \"%.3f\", $flash_ms/1000}")
    sleep "$secs"
    zellij action set-pane-color --pane-id "$pane_id" --reset 2>>"$LOG"
) &
disown 2>/dev/null || true
echo "flash-pane: bg=$flash_bg ms=$flash_ms id=$pane_id $(date)" >> "$LOG"
