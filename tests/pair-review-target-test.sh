#!/usr/bin/env bash
# tests/pair-review-target-test.sh — review-target session stamping.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-review-target-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
fails=0
pass() { printf '  ok   %s\n' "$1"; }
fail() { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

doc="$RT/doc.md"
printf 'doc\n' > "$doc"

PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=codex \
  PAIR_SESSION_ID=envsid "$ROOT/bin/pair" review target "$doc" ready >/dev/null
got="$(jq -r '.session' "$RT/review-target-test.json")"
[ "$got" = envsid ] && pass "uses PAIR_SESSION_ID when set" || fail "env session stamp ($got)"

printf '{"agent":"codex","args":[],"session_id":"cfgsid"}\n' > "$RT/config-test-codex.json"
PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=codex PAIR_SESSION_ID="" \
  "$ROOT/bin/pair" review target "$doc" ready >/dev/null
got="$(jq -r '.session' "$RT/review-target-test.json")"
[ "$got" = cfgsid ] && pass "falls back to config session_id" || fail "config session stamp ($got)"

[ "$fails" -eq 0 ] || { printf 'pair-review-target-test FAILED (%d)\n' "$fails"; exit 1; }
printf 'pair-review-target-test ok\n'
