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

# #155 removed config and live-rollout discovery as session authorities: the
# priority is PAIR_SESSION_ID -> established inventory binding, and any other
# binding state is explicit absence. This case pins that ABSENCE, which is the
# half nothing else covers -- a saved config and a discoverable rollout are both
# present here precisely so that falling back to either one fails the test.
cfgsid="12345678-1234-1234-1234-123456789abc"
rollout="$RT/home/.codex/sessions/2026/08/19/rollout-2026-08-19T00-00-00-$cfgsid.jsonl"
mkdir -p "$(dirname "$rollout")"
printf '{"type":"session_meta","payload":{"id":"%s","parent_thread_id":null,"source":"cli"}}\n' "$cfgsid" > "$rollout"
printf '{"agent":"codex","args":[],"session_id":"%s"}\n' "$cfgsid" > "$RT/config-test-codex.json"
HOME="$RT/home" PAIR_DATA_DIR="$RT" PAIR_TAG=test PAIR_AGENT=codex PAIR_SESSION_ID="" \
  "$ROOT/bin/pair" review target "$doc" ready >/dev/null
got="$(jq -r '.session' "$RT/review-target-test.json")"
[ -z "$got" ] && pass "no established binding stamps no session" \
  || fail "unbound session stamp ($got) — config/rollout must not be an authority"

[ "$fails" -eq 0 ] || { printf 'pair-review-target-test FAILED (%d)\n' "$fails"; exit 1; }
printf 'pair-review-target-test ok\n'
