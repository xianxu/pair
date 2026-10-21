#!/bin/sh
# Run a smoke command without inherited Pair/Couch artifacts or user stores.
set -eu
repo=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
. "$repo/tests/lib/pair-test-go-env.sh"
pair_test_resolve_go_env || exit $?
# Isolated message sockets include the physical root, incarnation PID and hash.
# Prefer RAM scratch, then a short inherited temp parent, then the short system
# temp parent. Never move sockets outside the owned isolation root.
fixture=
#lint:allow tmp-literal — AF_UNIX roots must stay short after physical canonicalization
for candidate in /Volumes/TestScratch "${TMPDIR:-/tmp}" /tmp; do
    [ -d "$candidate" ] || continue
    parent=$(CDPATH= cd -- "$candidate" && pwd -P) || continue
    [ "${#parent}" -le 25 ] || continue
    fixture=$(mktemp -d "$parent/pc.XXXXXX") || continue
    break
done
[ -n "$fixture" ] || { echo "cannot allocate a private short root for Unix sockets" >&2; exit 1; }
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
