#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PAIR_TEST_GOMODCACHE="$(go env GOMODCACHE)"
PAIR_TEST_GOCACHE="$(go env GOCACHE)"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-session-watch-test.XXXXXX")"
RT="$(cd "$RT" && pwd -P)"
trap 'rm -rf "$RT"; [ -z "${agent_pid:-}" ] || kill "$agent_pid" 2>/dev/null || true' EXIT

mkdir -p "$RT/data" "$RT/home/.codex/sessions/2026/08/28"
sid="019eff64-6ceb-7e72-9d41-a735a97029ac"
text="please inspect the durable watcher boundary now"
session_file="$RT/home/.codex/sessions/2026/08/28/rollout-test-$sid.jsonl"
printf '%s\n' \
  "{\"timestamp\":\"2026-08-28T01:00:00Z\",\"type\":\"session_meta\",\"payload\":{\"id\":\"$sid\",\"parent_thread_id\":null,\"source\":\"cli\"}}" \
  "{\"type\":\"response_item\",\"payload\":{\"type\":\"message\",\"role\":\"user\",\"content\":[{\"type\":\"input_text\",\"text\":\"$text\"}]}}" \
  '{"type":"response_item","payload":{"type":"function_call"}}' > "$session_file"

printf '{"v":1,"kind":"launch","scope_key":"scope","tag":"test","agent":"codex","pair_log_offset":0,"native_watermarks":[]}\n' > "$RT/data/ledger-test.jsonl"
printf '## 2026-08-28 01:00:01\n\n%s\n\n---\n\n' "$text" > "$RT/data/log-test.md"

bash -c 'exec 9<"$1"; sleep 30' _ "$session_file" &
agent_pid=$!
printf '%s\n' "$agent_pid" > "$RT/data/agent-pid-test"

HOME="$RT/home" GOMODCACHE="$PAIR_TEST_GOMODCACHE" GOCACHE="$PAIR_TEST_GOCACHE" PAIR_DATA_DIR="$RT/data" PAIR_TAG=test \
  go run ./cmd/pair-go session-watch codex test "$ROOT" \
  --scope-key scope --launch-ordinal 1 \
  --pid-not-before 2000-01-01T00:00:00Z -- --no-alt-screen

got="$(jq -r '.session_id // empty' "$RT/data/config-test-codex.json")"
[ "$got" = "$sid" ] || {
  echo "session_id mismatch: got '$got', want '$sid'" >&2
  exit 1
}

binding="$(tail -n 1 "$RT/data/ledger-test.jsonl")"
printf '%s' "$binding" | jq -e --arg sid "$sid" '.kind == "binding" and .launch_ordinal == 1 and .root_native_id == $sid' >/dev/null

echo "pair session-watch causal-round tests PASS"
