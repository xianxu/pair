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

# Clear the draft so the next session starts on a blank buffer. Truncate
# rather than rm so that the file's persistent-undo entry stays addressable
# (the undo history under ~/.local/share/pair/undo/ is keyed by file path).
# Best-effort: silently skip if env vars or file are missing.
data_dir="${PAIR_DATA_DIR:-${XDG_DATA_HOME:-$HOME/.local/share}/pair}"
tag="${PAIR_TAG:-${PAIR_AGENT:-claude}}"
draft="$data_dir/draft-$tag.md"
[ -f "$draft" ] && : > "$draft"

exec zellij kill-session "$session"
