#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-session-watch-test.XXXXXX")"
trap 'rm -rf "$RT"; [ -z "${live_pid:-}" ] || kill "$live_pid" 2>/dev/null || true' EXIT

mkdir -p "$RT/bin" "$RT/data" "$RT/home/.codex/sessions/2026/06/25"

sid="019eff64-6ceb-7e72-9d41-a735a97029ac"
session_file="$RT/home/.codex/sessions/2026/06/25/rollout-2026-06-25T08-27-12-$sid.jsonl"
printf '{"type":"session_meta","payload":{"id":"%s","parent_thread_id":null,"source":"cli"}}\n' "$sid" > "$session_file"

cat > "$RT/bin/lsof" <<SH
#!/usr/bin/env bash
if [ "\$1" = "-p" ] && [ "\$2" = "__LIVE_PID__" ]; then
  printf 'p%s\nn%s\n' "__LIVE_PID__" "$session_file"
fi
SH
chmod +x "$RT/bin/lsof"

sleep 10 &
live_pid=$!
sed "s/__LIVE_PID__/$live_pid/g" "$RT/bin/lsof" > "$RT/bin/lsof.tmp"
mv "$RT/bin/lsof.tmp" "$RT/bin/lsof"
chmod +x "$RT/bin/lsof"

bound="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
sleep 1
printf '%s\n' "$live_pid" > "$RT/data/agent-pid-test"
sleep 0.2

PATH="$RT/bin:$PATH" \
HOME="$RT/home" \
PAIR_DATA_DIR="$RT/data" \
PAIR_TAG=test \
PAIR_SESSION_WATCH_PID_WAIT_SECONDS=3 \
"$ROOT/bin/pair" session-watch codex test "$ROOT" --pid-not-before "$bound" -- resume old-session 'say "hi"' --no-alt-screen

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

echo 999999 > "$RT/data/agent-pid-stale"
touch -t 200001010000 "$RT/data/agent-pid-stale"
stale_bound="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"
PATH="$RT/bin:$PATH" \
HOME="$RT/home" \
PAIR_DATA_DIR="$RT/data" \
PAIR_TAG=stale \
PAIR_SESSION_WATCH_PID_WAIT_SECONDS=3 \
"$ROOT/bin/pair" session-watch codex stale "$ROOT" --pid-not-before "$stale_bound" -- --no-alt-screen &
watch_pid=$!

sleep 0.2
printf '%s\n' "$live_pid" > "$RT/data/agent-pid-stale"
wait "$watch_pid"

stale_got="$(jq -r '.session_id // empty' "$RT/data/config-stale-codex.json")"
[ "$stale_got" = "$sid" ] || {
  echo "stale replacement session_id mismatch: got '$stale_got', want '$sid'" >&2
  exit 1
}

echo "pair session-watch launcher-bound pidfile tests PASS"
