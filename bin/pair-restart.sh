#!/usr/bin/env bash
# Triggered by Alt+n via zellij keybind. Tears down the current pair session
# (like Alt+x) and signals bin/pair to re-launch with the same tag + agent +
# agent args, but a fresh agent conversation.
#
# Mechanism: writes BOTH a `quit-<session>` marker (so cleanup_quit_marker in
# bin/pair runs `zellij delete-session` as usual) AND a `restart-<session>`
# marker carrying the agent name. After kill-session returns, bin/pair sees
# the restart marker, drops $DATA_DIR/config-<tag>-<agent>.json (so the new
# run starts a fresh agent session rather than offering to resume), and execs
# itself with PAIR_FORCE_TAG set to the current tag.
#
# The agent name is captured here, while $DATA_DIR/agent-<tag> still exists —
# cleanup_quit_marker deletes that file before bin/pair gets the chance to
# read it.

set -uo pipefail

MARKER_DIR="$HOME/.cache/pair"
mkdir -p "$MARKER_DIR"

session="${ZELLIJ_SESSION_NAME:-}"
if [ -z "$session" ]; then
    echo "pair-restart: ZELLIJ_SESSION_NAME unset; cannot restart cleanly." >&2
    exit 1
fi

tag="${session#pair-}"
DATA_DIR="${XDG_DATA_HOME:-$HOME/.local/share}/pair"
agent=""
[ -f "$DATA_DIR/agent-$tag" ] && agent=$(cat "$DATA_DIR/agent-$tag" | tr -d '\r\n[:space:]')

{
    printf 'tag=%s\n' "$tag"
    printf 'agent=%s\n' "$agent"
} > "$MARKER_DIR/restart-$session"

touch "$MARKER_DIR/quit-$session"

exec zellij kill-session "$session"
