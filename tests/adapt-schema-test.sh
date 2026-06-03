#!/usr/bin/env bash
# Cross-emitter contract test for the adaptation flight recorder (#000045 M2).
#
# The whole feature rests on three emitters — Go (cmd/internal/adapt), shell
# (bin/lib/adapt-log.sh), and Lua (nvim/adapt.lua) — writing the SAME schema so
# doctor/doctor.sh reads every component uniformly. Frozen unit tests per
# emitter can't catch the three drifting apart; this does:
#
#   1. Golden: all three emit byte-identical lines (ts normalized) for the same
#      inputs — same field order, same escaping (incl. unescaped '>').
#   2. Atomicity: many concurrent appenders never tear a line (the O_APPEND
#      multi-process assumption the design relies on).
#
# Run: bash tests/adapt-schema-test.sh   (also wired into `make test`)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-adapt-schema.XXXXXX")"
trap 'rm -rf "$RT"' EXIT

fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

export PAIR_TAG=golden PAIR_DATA_DIR="$RT" PAIR_AGENT=codex
DETAIL='press > to continue? (y/n)'
norm() { jq -c '.ts="TS"'; } # normalize the only field that legitimately varies

# Single source of truth for the canonical line. The Go emitter is validated
# against this same fixture by TestGoldenMatchesFixture (the Go internal package
# can't be imported from tests/), so all three emitters are pinned to one shape.
expected="$(cat tests/adapt-golden.expected)"

# ── shell ──────────────────────────────────────────────────────────────────
rm -f "$RT/adapt-golden.jsonl"
( . bin/lib/adapt-log.sh; adapt_log golden codex 2 overlay-detect near-miss "$DETAIL" )
sh_line="$(norm < "$RT/adapt-golden.jsonl")"

# ── Lua ──────────────────────────────────────────────────────────────────────
# Emits into the same file (PAIR_TAG=golden); it appends, so read the last line.
nvim -l - <<EOF
local adapt = dofile('nvim/adapt.lua')
adapt.log(2, 'overlay-detect', 'near-miss', [[$DETAIL]], 'golden')
EOF
lua_line="$(tail -1 "$RT/adapt-golden.jsonl" | norm)"

# 1) Golden: shell and Lua match the canonical fixture (Go matches it via the
#    Go unit test) → all three emitters share one schema + field order.
if [ "$sh_line" = "$expected" ] && [ "$lua_line" = "$expected" ]; then
    pass "shell == Lua == canonical fixture (== Go, see TestGoldenMatchesFixture)"
else
    fail "emitter schemas diverged:"
    printf '    want: %s\n    sh  : %s\n    lua : %s\n' "$expected" "$sh_line" "$lua_line"
fi

# ── 2) Atomicity under concurrent multi-process appends ──────────────────────
conc="$RT/adapt-conc.jsonl"
: > "$conc"
PAIR_TAG=conc
N=60
for i in $(seq 1 "$N"); do
    ( . bin/lib/adapt-log.sh
      PAIR_TAG=conc adapt_log golden codex 1 return-remap fired "writer-$i" ) &
done
wait
got="$(wc -l < "$conc" | tr -d ' ')"
# Every line must be independently valid JSON (no torn/interleaved writes) and
# the count must match the number of writers.
valid="$(jq -c . < "$conc" 2>/dev/null | wc -l | tr -d ' ')"
if [ "$got" = "$N" ] && [ "$valid" = "$N" ]; then
    pass "$N concurrent appenders → $N intact JSON lines (no torn writes)"
else
    fail "concurrency: wrote=$got valid=$valid want=$N"
fi

if [ "$fails" -ne 0 ]; then
    printf '\n%d failure(s)\n' "$fails"
    exit 1
fi
printf '\nall adapt-schema tests passed\n'
