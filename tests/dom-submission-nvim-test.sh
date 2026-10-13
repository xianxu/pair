#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-dom-submission.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
for dom_case in notes insert empty comments failure pre_error uncertain no_notes ack changed progress timeout queue; do
  mkdir -p "$RT/$dom_case/queue"
  touch "$RT/$dom_case/draft.md" "$RT/$dom_case/log.md"
  if [[ "$dom_case" == queue ]]; then
    printf 'draft' > "$RT/$dom_case/draft.md"
    printf 'queued' > "$RT/$dom_case/queue/500000.md"
  fi
  run_headless --timeout 15 -- env \
    PAIR_TEST_DOM_CASE="$dom_case" PAIR_DATA_DIR="$RT/$dom_case" PAIR_TAG=test \
    PAIR_DRAFT_PATH="$RT/$dom_case/draft.md" PAIR_LOG_PATH="$RT/$dom_case/log.md" \
    PAIR_QUEUE_DIR="$RT/$dom_case/queue" PAIR_LAYOUT_MODE_PATH="$RT/$dom_case/layout" \
    nvim --headless -u "$ROOT/nvim/init.lua" "$RT/$dom_case/draft.md" \
    -l "$ROOT/nvim/dom_submission_integration_test.lua"
done
run_headless --timeout 20 -- nvim --headless -u NONE -l "$ROOT/tests/dom-submission-ui-test.lua"
printf 'dom-submission-nvim-test: passed\n'
