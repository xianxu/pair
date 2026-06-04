#!/usr/bin/env bash
# Regression test for doctor/doctor.sh — the adaptation flight-recorder reader.
#
# Guards I1 (surfaced in #000045 M1 review): a single malformed/partial JSON
# line in the log must NOT abort the diagnostic. The multi-process O_APPEND
# model only guarantees atomicity below PIPE_BUF, so a crashed writer can leave
# a partial line; doctor.sh must drop it, still report the good lines, and exit
# 0 — exactly when the operator most needs the tool.
#
# Run: bash doctor/doctor_test.sh   (also wired into `make test`)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DOCTOR="$ROOT/doctor/doctor.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-doctor-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT

fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

# Fixture: two good lines, one truncated/garbage line in the middle.
log="$RT/adapt-test.jsonl"
{
    echo '{"ts":"2026-06-03T10:00:00Z","comp":"pair-wrap","agent":"codex","aspect":1,"signal":"return-remap","outcome":"fired","detail":"a"}'
    echo '{"ts":"2026-06-03T10:00:01Z","comp":"pair-wrap","agent":"codex","aspect":2,"sig'   # truncated, invalid JSON
    echo '{"ts":"2026-06-03T10:00:02Z","comp":"pair-wrap","agent":"codex","aspect":2,"signal":"overlay-detect","outcome":"near-miss","detail":"Do you want to apply this patch? (y/n)"}'
} > "$log"

# 1) Survives the bad line and exits 0.
if out="$(bash "$DOCTOR" "$log" 2>&1)"; then
    pass "exits 0 despite a malformed line"
else
    fail "non-zero exit on malformed line"
fi

# 2) Still counts the two GOOD lines (bad line dropped).
if grep -q "return-remap/fired: 1" <<<"$out" && grep -q "overlay-detect/near-miss: 1" <<<"$out"; then
    pass "tallies the good lines, drops the bad one"
else
    fail "expected tallies missing; got:"$'\n'"$out"
fi

# 3) Surfaces the near-miss finding.
if grep -q "\[near-miss\] aspect 2 overlay-detect:" <<<"$out"; then
    pass "surfaces the near-miss drift finding"
else
    fail "near-miss finding not surfaced; got:"$'\n'"$out"
fi

# 3b) Emitter-health section renders (deterministic — the header always prints;
#     the per-binary verdict depends on the box, so only the header is asserted).
#     Probe internals are covered by tests/emitter-health-test.sh.
if grep -q "emitter health" <<<"$out"; then
    pass "emitter-health section renders"
else
    fail "emitter-health section missing; got:"$'\n'"$out"
fi

# 4) NO-DATA path on a missing file still exits 0.
if out="$(bash "$DOCTOR" "$RT/nope.jsonl" 2>&1)" && grep -q "NO-DATA" <<<"$out"; then
    pass "NO-DATA path exits 0 with notice"
else
    fail "NO-DATA path failed; got:"$'\n'"$out"
fi

if [ "$fails" -ne 0 ]; then
    printf '\n%d failure(s)\n' "$fails"
    exit 1
fi
printf '\nall doctor tests passed\n'
