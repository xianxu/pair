#!/usr/bin/env bash
# Regression test for bin/pair's cmux_rename_workspace ownership claim.
#
# Guards the fix for the "🔴 ♋-♋" stuck-title bug: a launch/attach/restart must
# claim the cmux workspace it is provably running in, EVEN IF the owner file
# names a different tag whose `pair-<owner>` zellij session is still alive
# somewhere else. The old code deferred to any live different-tag owner, which
# froze the title forever when that owner had moved to a different cmux
# workspace and left a stale owner file behind ("presence beats a stale flag").
#
# Drives the REAL bin/pair through its PAIR_TEST_CALL seam with process-level
# fakes (HOME/XDG_DATA_HOME pinned, fake `cmux` + `zellij` on PATH), exactly
# like the session_blocks_reuse / resolve_config_file cases in
# tests/pair-continue-test.sh.
#
# Run: bash tests/cmux-ownership-test.sh   (also wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PAIR="$ROOT/bin/pair"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-cmuxown-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT

fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

WS="testws-1"
mkdir -p "$RT/bin" "$RT/xdg/pair" "$RT/.cache/pair"

# Fake `cmux`: record every invocation's args so we can assert the rename fired
# (and with what title). `command -v cmux` must also succeed for the rename path.
cat > "$RT/bin/cmux" <<EOF
#!/usr/bin/env bash
printf '%s\n' "\$*" >> "$RT/cmux.log"
EOF
chmod +x "$RT/bin/cmux"

# Fake `zellij`: report that pair-<owner> sessions are alive — the old code
# would have deferred on this. (cmux_rename_workspace no longer consults it; the
# fake exists to prove the claim happens despite a live different-tag owner.)
cat > "$RT/bin/zellij" <<'EOF'
#!/usr/bin/env bash
case "$1 $2" in
  "list-sessions --short") printf 'pair-other\npair-211\n' ;;
  *) exit 0 ;;
esac
EOF
chmod +x "$RT/bin/zellij"

run_rename() {  # <tag> <title-arg>
    local tag="$1" title="$2"
    env HOME="$RT" XDG_DATA_HOME="$RT/xdg" PATH="$RT/bin:$PATH" \
        CMUX_WORKSPACE_ID="$WS" PAIR_TAG="$tag" \
        PAIR_TEST_CALL=cmux_rename_workspace PAIR_TEST_ARGS="$title" \
        "$PAIR" >/dev/null 2>&1
}

OWNER_FILE="$RT/xdg/pair/cmux-owner-$WS"

# 1. Stale different-tag owner ("other") whose session is alive elsewhere:
#    a launch by tag 211 must STILL claim — owner flips to 211 and cmux is
#    asked to rename. This is the core regression.
printf 'other\n' > "$OWNER_FILE"
: > "$RT/cmux.log"
run_rename 211 "pair-211"
if [ "$(cat "$OWNER_FILE")" = "211" ] && grep -q 'rename-workspace' "$RT/cmux.log"; then
    pass "claims a workspace held by a stale different-tag owner"
else
    fail "should claim over a stale different-tag owner (owner=$(cat "$OWNER_FILE") log=$(cat "$RT/cmux.log"))"
fi

# 2. The rename applies the personal display substitution (pair → ♋).
if grep -q 'rename-workspace ♋-211' "$RT/cmux.log"; then
    pass "applies the pair→♋ title substitution"
else
    fail "expected 'rename-workspace ♋-211', got: $(cat "$RT/cmux.log")"
fi

# 3. No owner file yet → claims cleanly.
rm -f "$OWNER_FILE"; : > "$RT/cmux.log"
run_rename 211 "pair-211"
if [ "$(cat "$OWNER_FILE")" = "211" ] && grep -q 'rename-workspace' "$RT/cmux.log"; then
    pass "claims an unowned workspace"
else
    fail "should claim an unowned workspace"
fi

# 4. Outside cmux (no CMUX_WORKSPACE_ID) → silent no-op, no rename, no owner file.
#    `env -u CMUX_WORKSPACE_ID` scrubs the var this very session leaks in (see
#    the make-test session-env-leak gotcha); cases 1-3 set it explicitly so are
#    unaffected.
: > "$RT/cmux.log"; rm -f "$OWNER_FILE"
env -u CMUX_WORKSPACE_ID HOME="$RT" XDG_DATA_HOME="$RT/xdg" PATH="$RT/bin:$PATH" \
    PAIR_TAG=211 PAIR_TEST_CALL=cmux_rename_workspace PAIR_TEST_ARGS="pair-211" \
    "$PAIR" >/dev/null 2>&1
if [ ! -s "$RT/cmux.log" ] && [ ! -f "$OWNER_FILE" ]; then
    pass "no-op outside cmux"
else
    fail "should be a no-op outside cmux"
fi

echo
if [ "$fails" -eq 0 ]; then
    echo "PASS: cmux-ownership claim"
    exit 0
else
    echo "FAIL: $fails case(s)"
    exit 1
fi
