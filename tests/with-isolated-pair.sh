#!/bin/sh
# Run a smoke command without inherited Pair/Couch artifacts or user stores.
set -eu
repo=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
. "$repo/tests/lib/pair-test-go-env.sh"
pair_test_resolve_go_env || exit $?
fixture=$(mktemp -d "${TMPDIR:-/tmp}/pair-smoke.XXXXXX")
fixture=$(CDPATH= cd -- "$fixture" && pwd -P)
trap 'rm -rf "$fixture"' EXIT HUP INT TERM
mkdir -p "$fixture/home" "$fixture/data/pair/couch" "$fixture/tmp"
# Reuse tool paths through the same canonical empty-environment boundary.
# Positional assignments retain explicit empty values and literal whitespace.
set -- -- "$@"
for key in GOTMPDIR GOMODCACHE GOCACHE; do
    case "$key" in
        GOTMPDIR) [ "${GOTMPDIR+x}" != x ] || set -- "GOTMPDIR=$GOTMPDIR" "$@" ;;
        GOMODCACHE) [ "${GOMODCACHE+x}" != x ] || set -- "GOMODCACHE=$GOMODCACHE" "$@" ;;
        GOCACHE) [ "${GOCACHE+x}" != x ] || set -- "GOCACHE=$GOCACHE" "$@" ;;
    esac
done
/bin/sh "$repo/tests/lib/pair-test-env.sh" "$fixture/home" \
 "TERM=${TERM:-xterm-256color}" "TMPDIR=$fixture/tmp" \
 "COUCH_ISOLATED_ROOT=$fixture" "XDG_DATA_HOME=$fixture/data" \
 "PAIR_DATA_DIR=$fixture/data/pair" "COUCH_STORE_DIR=$fixture/data/pair/couch" "$@"
