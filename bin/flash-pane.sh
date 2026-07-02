#!/usr/bin/env bash
# Compatibility shim for the Go-owned flash-pane command (#93 M4).
# copy-on-select execs this name (and tests/copy-on-select-test.sh stubs it by
# path), so it stays a tracked .sh that re-execs the Go binary.
#
# Usage: flash-pane.sh [<pane-id>]  (no arg → flash the focused pane)

set -uo pipefail

SOURCE="${BASH_SOURCE[0]}"
while [ -L "$SOURCE" ]; do
    DIR="$(cd -P "$(dirname "$SOURCE")" && pwd)"
    SOURCE="$(readlink "$SOURCE")"
    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
done
PAIR_HOME="$(cd -P "$(dirname "$SOURCE")/.." && pwd)"
export PAIR_HOME

cmd="$PAIR_HOME/bin/flash-pane"
if [ ! -x "$cmd" ]; then
    echo "flash-pane.sh: missing Go binary at $cmd; run make flash-pane or source ../ariadne/construct/dev-aliases.sh in a dev shell" >&2
    exit 1
fi

exec "$cmd" "$@"
