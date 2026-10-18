#!/bin/sh
# Source before replacing HOME or crossing the empty-environment boundary.
# Only Go's tool caches/temp path survive; explicit settings remain literal.
pair_test_resolve_go_env() {
    command -v go >/dev/null 2>&1 || return 0
    if [ "${GOCACHE+x}" != x ]; then
        GOCACHE=$(go env GOCACHE) || return $?
        export GOCACHE
    fi
    if [ "${GOMODCACHE+x}" != x ]; then
        GOMODCACHE=$(go env GOMODCACHE) || return $?
        export GOMODCACHE
    fi
    if [ "${GOTMPDIR+x}" != x ]; then
        GOTMPDIR=$(go env GOTMPDIR) || return $?
        export GOTMPDIR
    fi
}
