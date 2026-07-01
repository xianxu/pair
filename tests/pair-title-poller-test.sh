#!/usr/bin/env bash
# Regression + behavior test for bin/pair-title.sh.
#
# Part A — poller_alive() single-instance guard. Guards the load-bearing fix:
# the per-tag singleton check must NOT be a bare `kill -0`. After a host
# sleep/reboot the kernel can recycle a dead poller's PID onto an unrelated
# process; a naive liveness check would then read that stranger as "poller still
# alive" and suppress the respawn, freezing the title even across a `pair`
# restart. poller_alive() additionally verifies the PID's command line is
# genuinely our poller for THIS tag.
#
# Part B (#71) — update_frame_titles(): renames each agent pane's zellij frame
# to "<agent> (<count>) [<cwd>]" and skips redundant renames.
#
# Exercised through the script's PAIR_TITLE_TEST_CALL hook so we assert the REAL
# functions without a live cmux/zellij. `ps`/`zellij`/`pair-context` are faked
# on PATH; process liveness uses real PIDs.
#
# Run: bash tests/pair-title-poller-test.sh   (also wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
POLLER="$ROOT/bin/pair-title.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-title-test.XXXXXX")"
trap 'rm -rf "$RT"; [ -n "${LIVE_PID:-}" ] && kill "$LIVE_PID" 2>/dev/null' EXIT

fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

mkdir -p "$RT/bin"

# ── Part A: poller_alive() ──────────────────────────────────────────────────

# Fake `ps` on PATH: ignore args, print $FAKE_PS_CMD. poller_alive() invokes
# `ps -p <pid> -o command=`; this lets each case dictate the reported argv.
cat > "$RT/bin/ps" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "${FAKE_PS_CMD:-}"
EOF
chmod +x "$RT/bin/ps"

run_alive() {
    local tag="$1" pid="$2" ps_cmd="$3"
    PATH="$RT/bin:$PATH" FAKE_PS_CMD="$ps_cmd" \
        PAIR_TITLE_TEST_CALL=poller_alive \
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
if run_alive 211 "$LIVE_PID" "/usr/sbin/cupsd"; then
    fail "recycled PID (unrelated process) must NOT count as the poller"
else
    pass "recycled PID (unrelated process) is rejected"
fi

# 3. Dead PID → not alive.
if run_alive 211 "$DEAD_PID" "/bin/bash $POLLER 211 claude"; then
    fail "dead PID must NOT count as alive"
else
    pass "dead PID is rejected"
fi

# 4. Tag disambiguation: a poller for tag 211 must NOT satisfy a query for 21.
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

# ── Part B: update_frame_titles() (#71) ─────────────────────────────────────

# Fake zellij records rename-pane calls. The poller calls
#   `zellij --session <s> action rename-pane --pane-id <id> <title>`
# so $1=--session $2=<s> $3=action $4=rename-pane.
cat > "$RT/bin/zellij" <<'EOF'
#!/usr/bin/env bash
[ "${4:-}" = "rename-pane" ] && printf '%s\n' "$*" >> "$RENAME_LOG"
exit 0
EOF
# Fake `pair` dispatcher: `pair context ...` prints a fixed count (or empty when
# FAKE_COUNT unset/empty). The poller now invokes `pair context`, not the legacy
# pair-context binary.
cat > "$RT/bin/pair" <<'EOF'
#!/usr/bin/env bash
if [ "$1" = "context" ]; then
  [ -n "${FAKE_COUNT:-}" ] && printf '%s\n' "$FAKE_COUNT"
fi
exit 0
EOF
chmod +x "$RT/bin/zellij" "$RT/bin/pair"

DD="$RT/data"
mkdir -p "$DD"
printf '{"pane_id":"7","cwd":"/Users/x/repo","cwd_display":"~/repo"}\n' > "$DD/pane-T-claude.json"

run_frame() {
    local fn="$1" count="$2"
    : > "$RT/rename.log"
    PATH="$RT/bin:$PATH" PAIR_DATA_DIR="$DD" RENAME_LOG="$RT/rename.log" FAKE_COUNT="$count" \
        PAIR_TITLE_TEST_CALL="$fn" \
        bash "$POLLER" T claude >/dev/null 2>&1
}

# 6. One tick with a count → one rename with "claude (970k) [~/repo]" for pane 7.
run_frame update_frame_titles 970k
if grep -q -- "--pane-id 7 claude (970k) \[~/repo\]" "$RT/rename.log"; then
    pass "frame title shows agent (count) [cwd]"
else
    fail "frame title missing/wrong: $(cat "$RT/rename.log")"
fi

# 7. No count available → falls back to "claude [~/repo]" (no parens).
run_frame update_frame_titles ""
if grep -q -- "--pane-id 7 claude \[~/repo\]" "$RT/rename.log" && ! grep -q "(" "$RT/rename.log"; then
    pass "no-count fallback is agent [cwd]"
else
    fail "no-count fallback wrong: $(cat "$RT/rename.log")"
fi

# 8. Two ticks, same state → unchanged-skip: exactly ONE rename emitted.
run_frame _test_frame_titles_twice 970k
n=$(grep -c "rename-pane" "$RT/rename.log")
if [ "$n" -eq 1 ]; then
    pass "unchanged title is renamed once, not twice (skip guard)"
else
    fail "expected 1 rename across two identical ticks, got $n"
fi

echo
if [ "$fails" -eq 0 ]; then
    echo "PASS: pair-title poller (guard + frame meter)"
    exit 0
else
    echo "FAIL: $fails case(s)"
    exit 1
fi
