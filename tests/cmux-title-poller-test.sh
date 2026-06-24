#!/usr/bin/env bash
# Regression test for bin/pair-cmux-title.sh's poller_alive() single-instance
# guard.
#
# Guards the load-bearing fix: the per-tag singleton check must NOT be a bare
# `kill -0`. After a host sleep/reboot the kernel can recycle a dead poller's
# PID onto an unrelated process; a naive liveness check would then read that
# stranger as "poller still alive" and suppress the respawn, freezing the cmux
# workspace title even across a `pair` restart. poller_alive() additionally
# verifies the PID's command line is genuinely our poller for THIS tag.
#
# Exercised through the script's PAIR_CMUX_TITLE_TEST_CALL hook so we assert the
# REAL function without a live cmux/zellij. `ps` is faked on PATH so we control
# the reported command line; process liveness uses real PIDs (a backgrounded
# sleep for "alive", a reaped child for "dead").
#
# Run: bash tests/cmux-title-poller-test.sh   (also wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
POLLER="$ROOT/bin/pair-cmux-title.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-cmuxtitle-test.XXXXXX")"
trap 'rm -rf "$RT"; [ -n "${LIVE_PID:-}" ] && kill "$LIVE_PID" 2>/dev/null' EXIT

fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

# Fake `ps` on PATH: ignore args, print $FAKE_PS_CMD. poller_alive() invokes
# `ps -p <pid> -o command=`; this lets each case dictate the reported argv.
mkdir -p "$RT/bin"
cat > "$RT/bin/ps" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "${FAKE_PS_CMD:-}"
EOF
chmod +x "$RT/bin/ps"

# Run poller_alive <pid> with TAG=<tag>, a controlled fake-ps output, against
# the real script. Echoes nothing; returns the function's exit status.
run_alive() {
    local tag="$1" pid="$2" ps_cmd="$3"
    PATH="$RT/bin:$PATH" FAKE_PS_CMD="$ps_cmd" \
        PAIR_CMUX_TITLE_TEST_CALL=poller_alive \
        bash "$POLLER" "$tag" claude "$pid"
}

# A real live PID we can poke with kill -0.
sleep 600 & LIVE_PID=$!

# A real dead PID: spawn, reap, so kill -0 fails with ESRCH.
sh -c 'exit 0' & DEAD_PID=$!
wait "$DEAD_PID" 2>/dev/null

# 1. Live PID whose argv IS our poller for this tag → alive.
if run_alive 211 "$LIVE_PID" "/bin/bash $POLLER 211 claude"; then
    pass "matches a live poller for the tag"
else
    fail "should match a live poller for the tag"
fi

# 2. Live PID but argv is an UNRELATED process (recycled PID) → not alive.
#    This is the core regression: a bare kill -0 would wrongly return true.
if run_alive 211 "$LIVE_PID" "/usr/sbin/cupsd"; then
    fail "recycled PID (unrelated process) must NOT count as the poller"
else
    pass "recycled PID (unrelated process) is rejected"
fi

# 3. Dead PID → not alive (regardless of what ps would say).
if run_alive 211 "$DEAD_PID" "/bin/bash $POLLER 211 claude"; then
    fail "dead PID must NOT count as alive"
else
    pass "dead PID is rejected"
fi

# 4. Tag disambiguation: a poller for tag 211 must NOT satisfy a query for
#    tag 21 (the trailing space prevents the prefix collision).
if run_alive 21 "$LIVE_PID" "/bin/bash $POLLER 211 claude"; then
    fail "tag 21 must not match a poller for tag 211"
else
    pass "tag prefix collision (21 vs 211) is rejected"
fi

# 5. Empty PID (no pidfile yet) → not alive.
if run_alive 211 "" "/bin/bash $POLLER 211 claude"; then
    fail "empty PID must NOT count as alive"
else
    pass "empty PID is rejected"
fi

echo
if [ "$fails" -eq 0 ]; then
    echo "PASS: cmux-title poller guard"
    exit 0
else
    echo "FAIL: $fails case(s)"
    exit 1
fi
