#!/usr/bin/env bash
# Conformance: every key nvim/doctor.lua expects, doctor/perf.sh must emit.
#
# WHY A SEPARATE TEST. The key set is declared once in doctor.lua
# (HEADLINE_KEYS/PROBE_KEYS), but perf.sh is shell and cannot import it, so its
# `kv` calls and n/a helpers are a hand-maintained restatement. A restatement
# with no test is a deferred consumer, not a finished one: renaming a key in
# perf.sh drops the row from the prompt silently, which reads as "this tool has
# no such section" — exactly the defect the single-sourcing was meant to end.
#
# Note the failure this replaces. doctor_test.lua's degraded-capture test builds
# its input FROM HEADLINE_KEYS, so it agrees with itself by construction and
# cannot see a producer rename. This test reads the keys from doctor.lua and
# checks them against a REAL perf.sh run, so the producer is what is tested.
#
# Run: bash tests/perf-key-conformance-test.sh   (wired into `make test`)
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
fails=0
bad() { echo "  FAIL $*"; fails=$((fails + 1)); }

KEYDUMP="${TMPDIR:-/tmp}/perf-keydump.$$.lua"
cat > "$KEYDUMP" <<'LUA'
local M = dofile('nvim/doctor.lua')
local seen = {}
for _, list in ipairs({ M.HEADLINE_KEYS, M.PROBE_KEYS }) do
  for _, k in ipairs(list) do
    -- io.stdout, not print: `print` under `nvim -l` goes to STDERR, so a
    -- command substitution captures nothing and the test silently reads zero
    -- keys -- passing by checking an empty list.
    if not seen[k] then seen[k] = true; io.stdout:write(k, '\n') end
  end
end
LUA
keys=$(cd "$ROOT" && nvim -l "$KEYDUMP")
if [ -z "$keys" ]; then
  echo "perf-key-conformance-test: could not read the key declaration from doctor.lua"
  exit 1
fi

# A real run. Collectors may fail here (a sandbox denies ps, iostat, top) — that
# is fine and is itself part of the contract: a failed collector must render
# under its SUCCESS key, so the key is present either way. That is what makes
# this test meaningful in a restricted environment rather than skipped.
# Written to a file, not held in a variable: `printf "$out" | grep -q` makes
# grep exit on its first match and SIGPIPE the printf, which under make's shell
# turns every check into a spurious failure -- and made a mutation check read as
# "caught" for entirely the wrong reason.
CAPOUT="${TMPDIR:-/tmp}/perf-conformance.$$.txt"
trap 'rm -f "$KEYDUMP" "$CAPOUT"' EXIT
(cd "$ROOT" && PAIR_HOME="$ROOT" sh doctor/perf.sh 2>/dev/null) > "$CAPOUT"

echo "perf-key-conformance-test:"
while IFS= read -r k; do
  [ -n "$k" ] || continue
  if grep -q "^$k=" "$CAPOUT"; then
    printf '  ok   perf.sh emits %s\n' "$k"
  else
    bad "doctor.lua expects '$k' but perf.sh emits no such key (renamed producer?)"
  fi
done <<< "$keys"

if [ "$fails" -ne 0 ]; then
  echo "perf-key-conformance-test: $fails failure(s)"
  exit 1
fi
echo "perf-key-conformance-test: all passed"
