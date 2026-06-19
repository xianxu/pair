#!/usr/bin/env bash
# tests/review-handoff-test.sh — review.handoff: an atomic write appears to a
# timer-poll watcher, which decodes the records, fires the callback, and unlinks
# the file. XDG_DATA_HOME is redirected so the handoff lands in a temp dir.
#
# Run: bash tests/review-handoff-test.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-handoff-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
RESULT="$RT/result.txt"

cat > "$RT/driver.lua" <<'LUA'
local ROOT = os.getenv('PAIR_ROOT')
local handoff = dofile(ROOT .. '/nvim/review/handoff.lua')
local OUT = io.open(os.getenv('RESULT'), 'w')
local fails = 0
local function ok(cond, m)
  if cond then OUT:write('ok ' .. m .. '\n') else OUT:write('FAIL ' .. m .. '\n'); fails = fails + 1 end
end

local tag = 'test'
local got
local stop = handoff.watch(tag, function(recs) got = recs end, { interval = 20 })
handoff.write(tag, { { old = 'a', occurrence = 1, new = 'b', new_occurrence = 1, explain = 'x' } })
vim.wait(2000, function() return got ~= nil end, 20)
stop()

ok(got ~= nil, 'watch fired on the handoff')
ok(got and got[1] and got[1].new == 'b', 'received the decoded records')
ok(got and got[1] and got[1].new_occurrence == 1, 'records carry new_occurrence')
ok(vim.uv.fs_stat(handoff.path(tag)) == nil, 'handoff unlinked after consume')

OUT:write(fails == 0 and 'handoff_test ok\n' or ('FAILED ' .. fails .. '\n'))
OUT:close()
LUA

PAIR_ROOT="$ROOT" RESULT="$RESULT" XDG_DATA_HOME="$RT/xdg" \
  run_headless --timeout 30 -- nvim --headless -u NONE -c "luafile $RT/driver.lua" -c 'qa!'

echo "--- results ---"; cat "$RESULT"
if grep -q FAIL "$RESULT" || ! grep -q 'handoff_test ok' "$RESULT"; then
  echo "review-handoff-test FAILED"; exit 1
fi
echo "review-handoff-test ok"
