#!/usr/bin/env bash
# Self-test for the headless-nvim timeout watchdog (tests/lib/run-headless.sh,
# issue #60). The watchdog exists to turn a stuck `nvim --headless` into a
# bounded, loud failure instead of an infinite hang of the whole `make test`
# suite. Once the qall→qall! fix lands, NOTHING in the real suite hangs, so a
# green `make test` never exercises the watchdog's timeout path — its safety
# contract would ship unproven. This test pins that contract directly with
# deliberate-hang fixtures.
#
# Run: bash tests/run-headless-test.sh   (also wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"

fails=0
ok()   { printf '  ok   %s\n' "$1"; }
bad()  { printf '  FAIL %s\n' "$1"; fails=$((fails + 1)); }

echo "run-headless-test:"

# 1. Happy path: a clean-quitting nvim returns 0 (and the watchdog is quiet).
if out="$(run_headless --timeout 10 -- nvim --headless -c 'qall!' 2>&1)"; then
  [ -z "$out" ] && ok "clean exit returns 0, no noise" \
                || bad "clean exit returned 0 but was noisy: $out"
else
  bad "clean-quitting nvim should return 0, got rc=$?"
fi

# 2. THE contract — the real #60 failure mode: a driver that dirties the buffer
#    then bare-`qall`s hits E37, refuses to quit, and would hang forever. The
#    watchdog must kill it and return 124 within ~the timeout, loudly.
start=$SECONDS
out="$(run_headless --timeout 3 -- \
         nvim --headless -c 'call setline(1, "x")' -c 'qall' 2>&1)"
rc=$?
elapsed=$((SECONDS - start))
if [ "$rc" -eq 124 ]; then ok "modified-buffer bare-qall hang → rc=124"
else bad "modified-buffer hang should return 124, got rc=$rc"; fi
if [ "$elapsed" -le 8 ]; then ok "hang bounded (~${elapsed}s, timeout 3s)"
else bad "hang not bounded: took ${elapsed}s for a 3s timeout"; fi
case "$out" in
  *TIMEOUT*\#60*) ok "fails loud (TIMEOUT diagnostic names #60)" ;;
  *)              bad "missing loud TIMEOUT/#60 diagnostic; got: $out" ;;
esac

# 3. Mechanism backstop: a non-nvim never-terminating child is also bounded.
start=$SECONDS
run_headless --timeout 2 -- sh -c 'sleep 30' >/dev/null 2>&1
rc=$?
elapsed=$((SECONDS - start))
if [ "$rc" -eq 124 ] && [ "$elapsed" -le 6 ]; then
  ok "generic non-terminating child bounded → rc=124 (~${elapsed}s)"
else
  bad "sleep-30 under 2s timeout should be rc=124 ≤6s, got rc=$rc ${elapsed}s"
fi

if [ "$fails" -ne 0 ]; then
  echo "run-headless-test: $fails failure(s)"
  exit 1
fi
echo "run-headless-test: all passed"
