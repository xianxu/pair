#!/usr/bin/env bash
# pair#436: a test goal drops the live session's PAIR_*/COUCH_*/ZELLIJ* identity
# but keeps every opt-in knob the docs tell a caller to pass to make or go test.
set -euo pipefail
cd "$(dirname "$0")/.."

knobs=$(grep -rhoE '\b(PAIR|COUCH)_[A-Z0-9_]+=[^ ]* +(\\[[:space:]]*)?(make|go test)' README.md atlas Makefile.local doctor 2>/dev/null \
  | grep -oE '^(PAIR|COUCH)_[A-Z0-9_]+' | sort -u)
[ -n "$knobs" ] || { echo "FAIL: found no documented knobs (grep drifted?)"; exit 1; }
session="PAIR_TAG PAIR_DATA_DIR PAIR_SCOPE_KEY PAIR_LOG_PATH COUCH_PAIR_DATA_DIR COUCH_THREAD_TAG ZELLIJ ZELLIJ_SESSION_NAME"

args=()
for k in $knobs; do args+=("$k=knob-$k"); done
for s in $session; do args+=("$s=session-$s"); done
out=$(env "${args[@]}" make -s --no-print-directory test-make-env-probe)

fail=0
for k in $knobs; do
  grep -qx "$k=knob-$k" <<<"$out" || { echo "FAIL: knob $k did not reach the test recipe"; fail=1; }
done
for s in $session; do
  grep -q "^$s=" <<<"$out" && { echo "FAIL: session variable $s leaked into the test recipe"; fail=1; }
done
[ "$fail" -eq 0 ] && echo "PASS make-test-env ($(wc -w <<<"$knobs" | tr -d ' ') documented knobs kept, session identity dropped)"
exit "$fail"
