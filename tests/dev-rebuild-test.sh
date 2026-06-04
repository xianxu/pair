#!/usr/bin/env bash
# Regression test for bin/lib/dev-rebuild.sh — the PAIR_DEV-gated rebuild hook
# (#000046).
#
# Guards the load-bearing invariant flagged in the plan-quality review:
# DEPLOYED mode (PAIR_DEV unset) must invoke NO toolchain, so a future refactor
# that fires the build unconditionally can't silently add a Go/make dependency
# to every deployed launch. Also asserts dev mode DOES build, and that a build
# failure is errexit-safe (dev_rebuild returns 0 and the launcher continues —
# critical because bin/pair runs under `set -e` and a failed restart-time build
# must not strand the user with a dead session).
#
# Run: bash tests/dev-rebuild-test.sh   (also wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
LIB="$ROOT/bin/lib/dev-rebuild.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-devrebuild-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT

fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

# A fake `make` on PATH that records it was invoked (with its args) to a
# sentinel file, and exits with $FAKE_MAKE_EXIT. Lets us assert whether
# dev_rebuild reached the toolchain WITHOUT ever running the real build.
mkdir -p "$RT/bin"
cat > "$RT/bin/make" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" > "$RT/make-called"
exit \${FAKE_MAKE_EXIT:-0}
EOF
chmod +x "$RT/bin/make"
export PATH="$RT/bin:$PATH"
export PAIR_HOME="$ROOT"

# 1) Deployed mode: PAIR_DEV unset → dev_rebuild must NOT touch the toolchain.
rm -f "$RT/make-called"
( unset PAIR_DEV; . "$LIB"; dev_rebuild ) >/dev/null 2>&1
if [ ! -f "$RT/make-called" ]; then
    pass "deployed (PAIR_DEV unset) invokes no toolchain"
else
    fail "deployed mode called make: $(cat "$RT/make-called")"
fi

# 2) Dev mode: PAIR_DEV=1 → dev_rebuild runs `make … build`.
rm -f "$RT/make-called"
( export PAIR_DEV=1; . "$LIB"; dev_rebuild ) >/dev/null 2>&1
if [ -f "$RT/make-called" ] && grep -q 'build' "$RT/make-called"; then
    pass "dev (PAIR_DEV=1) runs make build"
else
    fail "dev mode did not invoke 'make build' (sentinel: $(cat "$RT/make-called" 2>/dev/null))"
fi

# 3) Errexit-safe: a failing build under `set -e` must not abort — dev_rebuild
#    returns 0 and the caller continues to launch with last-good binaries.
rm -f "$RT/make-called"
out="$( ( set -e; export PAIR_DEV=1 FAKE_MAKE_EXIT=1; . "$LIB"; dev_rebuild; echo SURVIVED ) 2>/dev/null )"
if [ "$out" = "SURVIVED" ]; then
    pass "build failure is errexit-safe (launcher continues)"
else
    fail "build failure aborted under set -e (got: '${out:-<empty>}')"
fi

if [ "$fails" -ne 0 ]; then
    printf '\n%d failure(s)\n' "$fails"
    exit 1
fi
printf '\nall dev-rebuild tests passed\n'
