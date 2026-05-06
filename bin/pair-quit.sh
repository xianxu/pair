#!/usr/bin/env bash
# Triggered by Alt+x via zellij keybind. Writes a marker file so bin/pair
# (the parent process) knows the user asked for a *full* quit, then kills
# the zellij session — which terminates all panes including this script.
# (Ctrl+q is unbound in pair's config, so Alt+x is the only quit path; this
# marker exists for parity with whatever future quit semantics we add.)
#
# bin/pair sees the marker on resume and runs `zellij delete-session --force`
# to clear the session entry from the resurrect list.

set -uo pipefail

MARKER_DIR="$HOME/.cache/pair"
mkdir -p "$MARKER_DIR"

session="${ZELLIJ_SESSION_NAME:-}"
if [ -z "$session" ]; then
    echo "pair-quit: ZELLIJ_SESSION_NAME unset; cannot quit cleanly." >&2
    exit 1
fi

touch "$MARKER_DIR/quit-$session"

# Note: the draft (`*` slot), queue (`+N`), and history (`-N`) all survive
# Alt+x. Use Shift+Alt+Backspace (forget_all) for the destructive "start
# anew" path — Alt+x is just "kill the session and its processes".

exec zellij kill-session "$session"
