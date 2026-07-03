#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-session-watch-test.XXXXXX")"
trap 'rm -rf "$RT"; [ -z "${live_pid:-}" ] || kill "$live_pid" 2>/dev/null || true' EXIT

mkdir -p "$RT/bin" "$RT/data" "$RT/home/.codex/sessions/2026/06/25"

sid="019eff64-6ceb-7e72-9d41-a735a97029ac"
session_file="$RT/home/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-$sid.jsonl"
: > "$session_file"

cat > "$RT/bin/lsof" <<SH
#!/usr/bin/env bash
if [ "\$1" = "-p" ] && [ "\$2" = "__LIVE_PID__" ]; then
  printf 'p%s\nn%s\n' "__LIVE_PID__" "$session_file"
fi
SH
chmod +x "$RT/bin/lsof"

echo 999999 > "$RT/data/agent-pid-test"
touch -t 200001010000 "$RT/data/agent-pid-test"

sleep 10 &
live_pid=$!
sed "s/__LIVE_PID__/$live_pid/g" "$RT/bin/lsof" > "$RT/bin/lsof.tmp"
mv "$RT/bin/lsof.tmp" "$RT/bin/lsof"
chmod +x "$RT/bin/lsof"

PATH="$RT/bin:$PATH" \
HOME="$RT/home" \
PAIR_DATA_DIR="$RT/data" \
PAIR_TAG=test \
PAIR_SESSION_WATCH_PID_WAIT_SECONDS=3 \
"$ROOT/bin/pair-session-watch" codex test "$ROOT" resume old-session 'say "hi"' --no-alt-screen &
watch_pid=$!

sleep 0.2
printf '%s\n' "$live_pid" > "$RT/data/agent-pid-test"

wait "$watch_pid"

got="$(jq -r '.session_id // empty' "$RT/data/config-test-codex.json")"
[ "$got" = "$sid" ] || {
  echo "session_id mismatch: got '$got', want '$sid'" >&2
  exit 1
}

args="$(jq -c '.args' "$RT/data/config-test-codex.json")"
[ "$args" = '["say \"hi\"","--no-alt-screen"]' ] || {
  echo "args mismatch: got '$args'" >&2
  exit 1
}

echo "pair-session-watch stale pidfile test PASS"
