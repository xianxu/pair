# dev-rebuild.sh — dev-mode rebuild hook for the pair launcher (#000046).
#
# `pair-dev` and `couch-dev` (#422) export PAIR_DEV=1. couch-dev calls this once
# before execing couch, and its slots inherit PAIR_DEV. `pair-dev` execs `pair`;
# bin/pair sources this and calls dev_rebuild once on the create path, just
# before the zellij layout execs pair-wrap. In dev mode it recompiles the Go
# binaries from source into $PAIR_HOME/bin (which is first on PATH), so the layout's `exec pair-wrap` — a
# PATH lookup that neither .zshenv nor construct/dev-aliases.sh's rebuild
# function can reach — resolves to a fresh build instead of a stale (or absent,
# since bin/ is gitignored) ~/.local/bin copy. See atlas/architecture.md.
#
# Restart-safe: Alt+n / Shift+Alt+N re-exec $0=bin/pair, and PAIR_DEV rides
# through exec in the environment, so the rebuild re-fires on every restart —
# the session keeps whichever mode it was launched in.
#
# Deployed launches leave PAIR_DEV unset → no-op → zero toolchain dependency.
#
# Usage:  PAIR_HOME=<repo> dev_rebuild      (no-op unless PAIR_DEV is set)
#
# ALWAYS returns 0: a build failure must never abort the launcher under
# `set -e` (bin/pair:20) — least of all mid-restart, when the old session is
# already gone. It warns loudly and falls through to the last-good binaries
# (loud, not silent — the opposite of the staleness this guards against).

dev_rebuild() {
    [ -n "${PAIR_DEV:-}" ] || return 0
    if ! command -v make >/dev/null 2>&1; then
        echo "dev-rebuild: 'make' not on PATH — launching with existing binaries." >&2
        return 0
    fi
    echo "dev-rebuild: rebuilding Go binaries (make build in $PAIR_HOME) …" >&2
    make -C "$PAIR_HOME" build >&2 \
        || echo "dev-rebuild: build FAILED — launching with last-good binaries (fix, then relaunch)." >&2
    return 0
}
