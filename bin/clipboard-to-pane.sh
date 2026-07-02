#!/usr/bin/env bash
# Compatibility shim for the Go-owned clipboard-to-pane command (#93 M4).
# copy-on-select execs this name (and tests/copy-on-select-test.sh stubs it by
# path), so it stays a tracked .sh that re-execs the Go binary.

set -uo pipefail

SOURCE="${BASH_SOURCE[0]}"
while [ -L "$SOURCE" ]; do
    DIR="$(cd -P "$(dirname "$SOURCE")" && pwd)"
    SOURCE="$(readlink "$SOURCE")"
    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
done
PAIR_HOME="$(cd -P "$(dirname "$SOURCE")/.." && pwd)"
export PAIR_HOME

cmd="$PAIR_HOME/bin/clipboard-to-pane"
if [ ! -x "$cmd" ]; then
    echo "clipboard-to-pane.sh: missing Go binary at $cmd; run make clipboard-to-pane or source ../ariadne/construct/dev-aliases.sh in a dev shell" >&2
    exit 1
fi

exec "$cmd" "$@"
