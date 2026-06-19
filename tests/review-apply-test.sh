#!/usr/bin/env bash
# tests/review-apply-test.sh — review.apply applies records as ONE undo-able
# block, decorates from the actual ranges, enriches new_occurrence, and is
# E790-safe. Runs under real `nvim --headless` (NOT `nvim -l`, whose Ex-batch
# undo semantics are unreliable). Driver does its own asserts → result file.
#
# Run: bash tests/review-apply-test.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-apply-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
RESULT="$RT/result.txt"

cat > "$RT/driver.lua" <<'LUA'
local ROOT = os.getenv('PAIR_ROOT')
local apply = dofile(ROOT .. '/nvim/review/apply.lua')
local reconstruct = dofile(ROOT .. '/nvim/review/reconstruct.lua')
local OUT = io.open(os.getenv('RESULT'), 'w')
local fails = 0
local function ok(cond, m)
  if cond then OUT:write('ok ' .. m .. '\n') else OUT:write('FAIL ' .. m .. '\n'); fails = fails + 1 end
end
local function newbuf(lines)
  local b = vim.api.nvim_create_buf(true, false)
  vim.api.nvim_buf_set_lines(b, 0, -1, false, lines)
  vim.api.nvim_set_current_buf(b)
  return b
end
local function content(b) return table.concat(vim.api.nvim_buf_get_lines(b, 0, -1, false), '\n') end
local function undo(b) vim.api.nvim_buf_call(b, function() vim.cmd('silent undo') end) end

-- (a) basic apply + enrichment
local b = newbuf({ 'teh cat', 'sat' })
local enr = apply.apply(b, { { old = 'teh', occurrence = 1, new = 'the', explain = 'typo' } })
ok(content(b) == 'the cat\nsat', 'apply replaced text')
ok(enr[1] and enr[1].new_occurrence == 1, 'enriched new_occurrence')

-- (b) single undo reverts the WHOLE round (two records → one undo block)
local b2 = newbuf({ 'aaa bbb ccc' })
apply.apply(b2, {
  { old = 'aaa', occurrence = 1, new = 'AAA', explain = '1' },
  { old = 'ccc', occurrence = 1, new = 'CCC', explain = '2' },
})
ok(content(b2) == 'AAA bbb CCC', 'two edits applied')
undo(b2)
ok(content(b2) == 'aaa bbb ccc', 'single undo reverts BOTH edits')

-- (c) extmark + diagnostic present
local b3 = newbuf({ 'x here' })
apply.apply(b3, { { old = 'x', occurrence = 1, new = 'yy', explain = 'why' } })
ok(#vim.api.nvim_buf_get_extmarks(b3, apply.HL, 0, -1, {}) >= 1, 'extmark placed')
local dg = vim.diagnostic.get(b3)
ok(#dg >= 1 and dg[1].message == 'why', 'diagnostic carries explain')

-- (d) drift: earlier edit changes length; both land right (bottom-to-top)
local b4 = newbuf({ 'one two three' })
local e4 = apply.apply(b4, {
  { old = 'one', occurrence = 1, new = '1', explain = 'a' },
  { old = 'three', occurrence = 1, new = 'THREE!!', explain = 'b' },
})
ok(content(b4) == '1 two THREE!!', 'drift: both edits correct')
ok(#reconstruct.decorate(e4, content(b4), 'new').highlights == 2, 'enriched records reconstruct both decorations')

-- (e) edges: empty no-op, single record no E790, single-undo reverts
local b5 = newbuf({ 'z' })
ok(pcall(function() apply.apply(b5, {}) end) and content(b5) == 'z', 'empty records: no-op, no error')
local b6 = newbuf({ 'z z' })
ok(pcall(function() apply.apply(b6, { { old = 'z', occurrence = 2, new = 'Z', explain = 'e' } }) end), 'single record: no E790')
ok(content(b6) == 'z Z', 'single record applied at occurrence 2')
undo(b6)
ok(content(b6) == 'z z', 'single-record single-undo reverts')

OUT:write(fails == 0 and 'apply_test ok\n' or ('FAILED ' .. fails .. '\n'))
OUT:close()
LUA

PAIR_ROOT="$ROOT" RESULT="$RESULT" \
  run_headless --timeout 30 -- nvim --headless -u NONE -c "luafile $RT/driver.lua" -c 'qa!'

echo "--- results ---"; cat "$RESULT"
if grep -q FAIL "$RESULT" || ! grep -q 'apply_test ok' "$RESULT"; then
  echo "review-apply-test FAILED"; exit 1
fi
echo "review-apply-test ok"
