#!/usr/bin/env bash
# Compatibility shim for the Go-owned copy-on-select command (#93 M4).
# zellij's `copy_command "copy-on-select.sh"` (zellij/config.kdl) keeps
# resolving this name; the selection arrives on stdin and is passed through.
# The Go binary still execs $PAIR_HOME/bin/{flash-pane,clipboard-to-pane}.sh,
# preserving the chain tests/copy-on-select-test.sh stubs by path.

set -uo pipefail

SOURCE="${BASH_SOURCE[0]}"
while [ -L "$SOURCE" ]; do
    DIR="$(cd -P "$(dirname "$SOURCE")" && pwd)"
    SOURCE="$(readlink "$SOURCE")"
    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
done
PAIR_HOME="$(cd -P "$(dirname "$SOURCE")/.." && pwd)"
export PAIR_HOME

cmd="$PAIR_HOME/bin/copy-on-select"
if [ ! -x "$cmd" ]; then
    echo "copy-on-select.sh: missing Go binary at $cmd; run make copy-on-select or source ../ariadne/construct/dev-aliases.sh in a dev shell" >&2
    exit 1
fi

exec "$cmd" "$@"
