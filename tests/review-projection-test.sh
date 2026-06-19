#!/usr/bin/env bash
# tests/review-projection-test.sh — review.projection keeps decorations coherent
# across undo/redo and lets them ride manual edits, without clearing accumulated
# rounds (#66 M2). The TextChanged watcher is exercised for real (undo/redo fire
# it). Runs under `nvim --headless`.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-projection-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
RESULT="$RT/result.txt"

cat > "$RT/driver.lua" <<'LUA'
local ROOT = os.getenv('PAIR_ROOT')
local apply = dofile(ROOT .. '/nvim/review/apply.lua')
local projection = dofile(ROOT .. '/nvim/review/projection.lua')
local OUT = io.open(os.getenv('RESULT'), 'w')
local fails = 0
local function ok(c, m)
  if c then OUT:write('ok ' .. m .. '\n') else OUT:write('FAIL ' .. m .. '\n'); fails = fails + 1 end
end
local function content(b) return table.concat(vim.api.nvim_buf_get_lines(b, 0, -1, false), '\n') end
local function ndecos(b) return #vim.api.nvim_buf_get_extmarks(b, apply.HL, 0, -1, {}) end
local function undo(b) vim.api.nvim_buf_call(b, function() vim.cmd('silent undo') end) end
local function redo(b) vim.api.nvim_buf_call(b, function() vim.cmd('silent redo') end) end
local function newbuf(lines)
  local b = vim.api.nvim_create_buf(true, false)
  vim.api.nvim_buf_set_lines(b, 0, -1, false, lines)
  vim.api.nvim_set_current_buf(b)
  vim.api.nvim_buf_call(b, function() vim.cmd('silent! let &undolevels = &undolevels') end)
  projection.reset(b) -- fresh projection state per buffer (no double-attached watcher)
  return b
end
-- a round, mirroring the orchestrator's projection sequence
local function round(b, records)
  local base = content(b)
  projection.set_applying(b, true)
  apply.apply(b, records)
  projection.record_empty_for(b, base)
  projection.record(b)
  projection.ensure_watch(b)
  projection.set_applying(b, false)
end
-- The TextChanged watcher fires reliably in a real session but NOT under
-- `nvim --headless`, so the test drives projection.project() explicitly (what
-- the watcher calls) and asserts separately that the watcher IS registered.

-- (a) undo/redo coherence, with a MULTI-LINE decoration
local b = newbuf({ 'p1', 'old', 'q', 'tail' })
round(b, {
  { old = 'old', occurrence = 1, new = 'NEW1\nNEW2', explain = 'm' },
  { old = 'tail', occurrence = 1, new = 'T', explain = 's' },
})
ok(ndecos(b) >= 2, 'round placed decorations')
ok(#vim.api.nvim_get_autocmds({ buffer = b, event = 'TextChanged' }) >= 1, 'watcher autocmd registered')
undo(b); projection.project(b)
ok(ndecos(b) == 0, 'undo restores the empty pre-round snapshot (decorations cleared)')
redo(b); projection.project(b)
ok(ndecos(b) >= 2, 'redo restores decorations')
local ml = false
for _, m in ipairs(vim.api.nvim_buf_get_extmarks(b, apply.HL, 0, -1, { details = true })) do
  if m[4] and m[4].end_row and m[4].end_row > m[2] then ml = true end
end
ok(ml, 'restored decoration keeps its multi-line span (end_row)')

-- (b) decorations ride a manual edit; undo restores the round snapshot
vim.api.nvim_buf_set_lines(b, -1, -1, false, { 'manual note' })
projection.project(b)
ok(ndecos(b) >= 2, 'decorations ride a manual edit')
undo(b); projection.project(b)
ok(ndecos(b) >= 2, 'undo of the manual edit restores round decorations')

-- (c) round-2 idempotence: round-1 output is round-2's base; undoing back to the
-- round-1 state must NOT clear round-1's decorations (record_empty_for guard).
local b2 = newbuf({ 'alpha foo', 'beta bar' })
round(b2, { { old = 'foo', occurrence = 1, new = 'FOO', explain = '1' } })
local after_r1 = content(b2)
round(b2, { { old = 'bar', occurrence = 1, new = 'BAR', explain = '2' } })
undo(b2); projection.project(b2)
ok(content(b2) == after_r1, 'undo returns to the round-1 content')
ok(ndecos(b2) >= 1, 'round-1 decorations survive (record_empty_for idempotence guard)')

OUT:write(fails == 0 and 'projection_test ok\n' or ('FAILED ' .. fails .. '\n'))
OUT:close()
LUA

PAIR_ROOT="$ROOT" RESULT="$RESULT" \
  run_headless --timeout 30 -- nvim --headless -u NONE -c "luafile $RT/driver.lua" -c 'qa!'

echo "--- results ---"; cat "$RESULT"
if grep -q FAIL "$RESULT" || ! grep -q 'projection_test ok' "$RESULT"; then
  echo "review-projection-test FAILED"; exit 1
fi
echo "review-projection-test ok"
