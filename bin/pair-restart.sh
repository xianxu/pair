#!/usr/bin/env bash
# Triggered by Alt+n (and Shift+Alt+N) via zellij keybinds. Tears down the
# current pair session (like Alt+x) and signals bin/pair to re-launch:
#
#   default        — same tag + agent + args + agent session. Pure reload
#                    of pair itself; the agent conversation is preserved
#                    (resumed via --resume <id> / resume <id>). (Alt+n)
#   --new-session  — same tag + agent + args, but a fresh agent
#                    conversation: bin/pair drops the saved
#                    config-<tag>-<agent>.json so the next launch starts
#                    a brand-new session. (Shift+Alt+N)
#
# Mechanism: writes BOTH a `quit-<session>` marker (so cleanup_quit_marker in
# bin/pair runs `zellij delete-session` as usual) AND a `restart-<session>`
# marker carrying the agent name + optional new_session flag. After
# kill-session returns, bin/pair sees the restart marker and execs itself
# with PAIR_FORCE_TAG pinning the same tag — the new_session flag controls
# whether the saved config-<tag>-<agent>.json is dropped (fresh) or kept
# (resume the prior agent session).
#
# The agent name is captured here, while $DATA_DIR/agent-<tag> still exists —
# cleanup_quit_marker deletes that file before bin/pair gets the chance to
# read it.

set -uo pipefail

new_session=0
case "${1:-}" in
    --new-session) new_session=1 ;;
    "") : ;;
    *) echo "pair-restart: unknown arg ${1:-}" >&2; exit 2 ;;
esac

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
    printf 'new_session=%s\n' "$new_session"
} > "$MARKER_DIR/restart-$session"

touch "$MARKER_DIR/quit-$session"

exec zellij kill-session "$session"
