#!/bin/sh
# Run a smoke command without inherited Pair/Couch artifacts or user stores.
set -eu
fixture=$(mktemp -d "${TMPDIR:-/tmp}/pair-smoke.XXXXXX")
fixture=$(CDPATH= cd -- "$fixture" && pwd -P)
trap 'rm -rf "$fixture"' EXIT HUP INT TERM
mkdir -p "$fixture/home" "$fixture/data/pair/couch"
# Reuse build caches, not user Pair state. Keep the environment allowlist small.
go_cache=$(go env GOCACHE)
go_modules=$(go env GOMODCACHE)
env -i PATH="$PATH" TERM="${TERM:-xterm-256color}" TMPDIR="${TMPDIR:-/tmp}" \
 HOME="$fixture/home" XDG_DATA_HOME="$fixture/data" \
 PAIR_DATA_DIR="$fixture/data/pair" COUCH_STORE_DIR="$fixture/data/pair/couch" \
 GOCACHE="$go_cache" GOMODCACHE="$go_modules" "$@"
