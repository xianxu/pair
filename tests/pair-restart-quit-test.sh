#!/usr/bin/env bash
# Drives `pair restart` / `pair quit` (the Go ports of bin/pair-{restart,quit}.sh,
# #94 M1) against a stubbed kill-session, asserting the marker files land where
# the launcher's restart loop reads them (~/.cache/pair). PAIR_KILL_CMD swaps the
# terminal `zellij kill-session` exec for `true` so the writes are observable.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PAIR="$ROOT/bin/pair" # built by the test-pair-restart-quit prereq
TMP="$(mktemp -d "${TMPDIR:-/tmp}/pair-restart-quit.XXXXXX")"
trap 'rm -rf "$TMP"' EXIT

export HOME="$TMP"
export PAIR_KILL_CMD="true" # ExecKillSession runs `true <session>` instead of zellij
export ZELLIJ_SESSION_NAME="pair-smoke"
MARK="$TMP/.cache/pair"
# NB: deliberately do NOT pre-create $MARK — the first `pair restart` must create
# it via WriteAtomic/Touch's MkdirAll, so this smoke is load-bearing for that path.

# (a) restart writes the restart marker (tag/new_session/rename_to) + touches quit.
"$PAIR" restart --new-session --rename-to renamed
grep -qx 'tag=smoke'         "$MARK/restart-pair-smoke"
grep -qx 'new_session=1'     "$MARK/restart-pair-smoke"
grep -qx 'rename_to=renamed'  "$MARK/restart-pair-smoke"
test -f "$MARK/quit-pair-smoke"

# (b) quit touches quit AND writes NO restart marker. Clear both first so the
#     post-quit assertions pin quit's behavior, not restart's leftovers.
rm -f "$MARK/quit-pair-smoke" "$MARK/restart-pair-smoke"
"$PAIR" quit
test -f "$MARK/quit-pair-smoke"
test ! -f "$MARK/restart-pair-smoke" # quit must not write a restart marker

# (c) missing session → exit 1, no marker written.
unset ZELLIJ_SESSION_NAME
rm -f "$MARK/quit-pair-smoke"
if "$PAIR" quit 2>/dev/null; then
    echo "FAIL: pair quit with no ZELLIJ_SESSION_NAME should exit non-zero" >&2
    exit 1
fi
test ! -f "$MARK/quit-pair-smoke"

echo "PASS pair-restart-quit"
