#!/usr/bin/env bash
# pair-title.sh — compatibility shim over the Go title poller (#93 M1). The
# logic moved to cmd/internal/titlepoller; this name survives because bin/pair's
# ensure_title_poller spawns it by path. Re-execs the Go binary so the running
# process is "<…>/pair-title <tag> <agent>" — the shape the poller's
# single-instance argv guard recognizes.
set -uo pipefail
SOURCE="${BASH_SOURCE[0]}"
while [ -L "$SOURCE" ]; do
    DIR="$(cd -P "$(dirname "$SOURCE")" && pwd)"
    SOURCE="$(readlink "$SOURCE")"
    [[ $SOURCE != /* ]] && SOURCE="$DIR/$SOURCE"
done
PAIR_HOME="$(cd -P "$(dirname "$SOURCE")/.." && pwd)"
export PAIR_HOME

cmd="$PAIR_HOME/bin/pair-title"
if [ ! -x "$cmd" ]; then
    echo "pair-title.sh: missing Go poller at $cmd; run make pair-title or source ../ariadne/construct/dev-aliases.sh in a dev shell" >&2
    exit 1
fi

exec "$cmd" "$@"
