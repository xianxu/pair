#!/usr/bin/env bash
# Compatibility shim for the Go-owned pair-session-watch command.

set -uo pipefail

SOURCE="${BASH_SOURCE[0]}"
while [ -L "$SOURCE" ]; do
    DIR="$(cd -P "$(dirname "$SOURCE")" && pwd)"
    SOURCE="$(readlink "$SOURCE")"
    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
done
PAIR_HOME="$(cd -P "$(dirname "$SOURCE")/.." && pwd)"
export PAIR_HOME

cmd="$PAIR_HOME/bin/pair-session-watch"
if [ ! -x "$cmd" ]; then
    echo "pair-session-watch.sh: missing Go watcher at $cmd; run make pair-session-watch or source ../ariadne/construct/dev-aliases.sh in a dev shell" >&2
    exit 1
fi

exec "$cmd" "$@"
