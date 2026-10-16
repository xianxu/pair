#!/usr/bin/env bash
# Full init.lua → delivery sequencer → Pair-log transaction failure matrix.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-submission-transaction.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
bash "$ROOT/tests/lib/pair-test-binary.sh" "$RT/pair"

for tx_case in focus-agent write-body submit focus-draft commit; do
  run_headless --timeout 30 -- \
    env PAIR_DATA_DIR='' PAIR_TAG='' PAIR_SCOPE_KEY='' PAIR_RETENTION_PROTOCOL='' \
    PAIR_TEST_TX_CASE="$tx_case" PAIR_TEST_BIN="$RT/pair" PAIR_LOG_PATH="$RT/$tx_case.md" \
    nvim --headless -u "$ROOT/nvim/init.lua" \
    -l "$ROOT/nvim/submission_integration_test.lua"
  printf '  ok   submission transaction %s\n' "$tx_case"
done

echo 'submission-transaction-nvim-test: all passed'

lua "$ROOT/nvim/dom_submission_state_test.lua"
lua "$ROOT/nvim/pane_identity_test.lua"
lua "$ROOT/nvim/dom_submission_test.lua"
lua "$ROOT/nvim/dom_transport_test.lua"
bash "$ROOT/tests/dom-submission-nvim-test.sh"
