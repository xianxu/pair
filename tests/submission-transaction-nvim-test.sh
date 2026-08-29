#!/usr/bin/env bash
# Full init.lua → delivery sequencer → Pair-log transaction failure matrix.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-submission-transaction.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
go build -p 20 -o "$RT/pair" ./cmd/pair-go

for tx_case in focus-agent write-body submit newline focus-draft focus-draft-compose commit; do
  run_headless --timeout 30 -- \
    env PAIR_TEST_TX_CASE="$tx_case" PAIR_TEST_BIN="$RT/pair" PAIR_LOG_PATH="$RT/$tx_case.md" \
    nvim --headless -u "$ROOT/nvim/init.lua" \
    -c "luafile $ROOT/nvim/submission_integration_test.lua"
  printf '  ok   submission transaction %s\n' "$tx_case"
done

echo 'submission-transaction-nvim-test: all passed'
