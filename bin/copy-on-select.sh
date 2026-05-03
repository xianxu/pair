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
# extracts both the pane id and a signature for the in-nvim check.
in_nvim=false
focused_id=""
if command -v jq >/dev/null 2>&1; then
    focused=$(zellij action list-panes --json --command 2>/dev/null \
              | jq -r '[.. | objects | select(.is_focused == true)][0]
                       | "\(.id // .pane_id // "")\t\(.title // "") \(.terminal_command // "")"' 2>/dev/null)
    focused_id=${focused%%$'\t'*}
    sig=${focused#*$'\t'}
    if printf '%s' "$sig" | grep -qiE 'nvim|draft'; then
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

# Gate the auto-insert on nvim's current mode: only insert quotes if nvim is
# in insert mode. In normal mode the user is typically browsing/navigating
# (history, queue) and an unsolicited buffer mutation is disruptive. The
# selection is still on the OS clipboard from step 1, so they can paste
# manually if they want it. nvim writes its mode to this file via a
# ModeChanged autocmd; missing file = first-run / no recent mode change ⇒
# default to "insert" so existing copy-on-select behavior isn't lost.
mode_file="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}/mode-${PAIR_TAG:-${PAIR_AGENT:-claude}}"
if [ -f "$mode_file" ]; then
    nvim_mode=$(cat "$mode_file" 2>/dev/null)
    if [ "${nvim_mode:0:1}" != "i" ]; then
        echo "nvim mode '$nvim_mode' is not insert; skipping auto-insert" >> "$LOG"
        exit 0
    fi
fi

# 3. Flash the source pane's background so the user gets a visual cue at the
# selection site (in addition to the nvim-side flash on the inserted text).
# set-pane-color sets the pane's *default* bg, so this only shows through in
# cells the agent isn't actively painting — best-effort. Override the color
# via $PAIR_FLASH_BG, the duration via $PAIR_FLASH_MS.
flash_bg="${PAIR_FLASH_BG:-#5a4a00}"
flash_ms="${PAIR_FLASH_MS:-100}"
if [ -n "$focused_id" ]; then
    zellij action set-pane-color --pane-id "$focused_id" --bg "$flash_bg" 2>>"$LOG"
    # Background a delayed reset so the flash auto-clears. Survives the exec
    # below because backgrounded children outlive parent process replacement.
    (
        # Convert ms → seconds for sleep(1) (POSIX sleep accepts decimals).
        secs=$(awk "BEGIN{printf \"%.3f\", $flash_ms/1000}")
        sleep "$secs"
        zellij action set-pane-color --pane-id "$focused_id" --reset 2>>"$LOG"
    ) &
    disown 2>/dev/null || true
    echo "flash: bg=$flash_bg ms=$flash_ms id=$focused_id" >> "$LOG"
fi

# 4. Hand off to clipboard-to-pane.sh for the actual insert into nvim.
# It reads the OS clipboard (which we populated in step 1).
exec "$PAIR_HOME/bin/clipboard-to-pane.sh"
