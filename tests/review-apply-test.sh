#!/usr/bin/env bash
# tests/review-apply-test.sh — review.apply, the decoration layer: render places
# resume decorations from records (matching reconstruct), snapshot/apply_snapshot
# round-trip exact and multi-line ranges, and clear_at_line clears one region.
# The buffer apply is gone (pair#664): an agent round reaches the file through
# `pair review apply`. Runs under real `nvim --headless`; the driver does its own
# asserts → result file.
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

-- Decorate spans the way the pane does after a reload (round_watch.decorations
-- → apply_snapshot): {line,col,end_line,end_col,message}.
local function decorate(b, spans)
  local hl, diags = {}, {}
  for _, sp in ipairs(spans) do
    hl[#hl + 1] = { line = sp[1], col = sp[2], end_line = sp[3], end_col = sp[4] }
    diags[#diags + 1] = { lnum = sp[1], col = sp[2], end_lnum = sp[3], end_col = sp[4], message = sp[5] }
  end
  apply.apply_snapshot(b, { hl = hl, diags = diags })
end

-- (l) apply.render — the RESUME-render path M2 consumes: given records with
-- new_occurrence + content, places decorations matching reconstruct.
local br = newbuf({ 'alpha', 'TWO bar', 'IV baz' })
apply.render(br, {
  { new = 'TWO', new_occurrence = 1, explain = 'a' },
  { new = 'IV', new_occurrence = 1, explain = 'b' },
}, content(br))
ok(#vim.api.nvim_buf_get_extmarks(br, apply.HL, 0, -1, {}) == 2, 'render: 2 extmarks placed')
ok(#vim.diagnostic.get(br) == 2, 'render: 2 diagnostics placed')
local rlines = {}
for _, m in ipairs(vim.api.nvim_buf_get_extmarks(br, apply.HL, 0, -1, {})) do rlines[m[2]] = true end
ok(rlines[1] and rlines[2], 'render: decorations on the right lines (1 and 2)')
local br_same = newbuf({ 'A B C' })
apply.render(br_same, {
  { new = 'A', new_occurrence = 1, explain = 'first' },
  { new = 'C', new_occurrence = 1, explain = 'third' },
}, content(br_same))
local br_same_diags = vim.diagnostic.get(br_same, { namespace = apply.DIAG })
table.sort(br_same_diags, function(a, b) return a.message < b.message end)
ok(#br_same_diags == 2
  and br_same_diags[1].message == 'first'
  and br_same_diags[1].lnum == 0
  and br_same_diags[1].col == 0
  and br_same_diags[1].end_col == 1
  and br_same_diags[2].message == 'third'
  and br_same_diags[2].lnum == 0
  and br_same_diags[2].col == 4
  and br_same_diags[2].end_col == 5,
  'render diagnostics preserve distinct same-line columns')

local brm = newbuf({ 'prefix 🤖<bad>{GOOD} suffix' })
apply.render(brm, { { new = '🤖<bad>{GOOD}', new_occurrence = 1, explain = 'proposal' } }, content(brm))
ok(#vim.api.nvim_buf_get_extmarks(brm, apply.HL, 0, -1, {}) == 0, 'render marker proposal: no redundant highlight')
local rmd = vim.diagnostic.get(brm, { namespace = apply.DIAG })
ok(#rmd == 1 and rmd[1].message == 'proposal', 'render marker proposal keeps diagnostic')

-- (m) snapshot/apply_snapshot round-trip, including exact columns and a
-- MULTI-LINE decoration — the span must survive projection.
local bs = newbuf({ 'p1', 'NEW1', 'NEW2', 'q', 'T' })
decorate(bs, { { 1, 0, 2, #'NEW2', 'multi' }, { 4, 0, 4, 1, 'single' } })
local snap = apply.snapshot(bs)
local ml, exact = false, false
for _, h in ipairs(snap.hl) do
  if h.end_line > h.line and h.col == 0 and h.end_col == #'NEW2' then ml = true end
  if h.line == 4 and h.col == 0 and h.end_col == 1 then exact = true end
end
ok(ml, 'snapshot captures a multi-line exact extmark range')
ok(exact, 'snapshot captures exact single-line columns')
ok(#snap.diags >= 2, 'snapshot captures diagnostics')
-- wipe both layers, then restore from the snapshot
vim.api.nvim_buf_clear_namespace(bs, apply.HL, 0, -1)
vim.diagnostic.reset(apply.DIAG, bs)
apply.apply_snapshot(bs, snap)
local restored_ml, restored_exact = false, false
for _, m in ipairs(vim.api.nvim_buf_get_extmarks(bs, apply.HL, 0, -1, { details = true })) do
  if m[4] and m[4].end_row and m[4].end_row > m[2] and m[3] == 0 and m[4].end_col == #'NEW2' then restored_ml = true end
  if m[2] == 4 and m[3] == 0 and m[4] and m[4].end_col == 1 then restored_exact = true end
end
ok(restored_ml, 'apply_snapshot restores the multi-line exact range')
ok(restored_exact, 'apply_snapshot restores exact single-line columns')
ok(#vim.diagnostic.get(bs, { namespace = apply.DIAG }) >= 2, 'apply_snapshot restores diagnostics')

local bmr = newbuf({ 'abc 🤖<def>{DEF} ghi' })
apply.render(bmr, {
  { new = 'abc 🤖<def>{DEF} ghi', new_occurrence = 1, explain = 'emphasis' },
}, content(bmr))
ok(#vim.api.nvim_buf_get_extmarks(bmr, apply.HL, 0, -1, {}) == 0,
  'render embedded marker proposal: no redundant highlight')
local mrd = vim.diagnostic.get(bmr, { namespace = apply.DIAG })
ok(#mrd == 1 and mrd[1].message == 'emphasis',
  'render embedded marker proposal keeps diagnostic')

-- (q) targeted clear: accepting a styled region without a marker clears only
-- that region's review highlight + matching diagnostic.
local bc = newbuf({ 'ALPHA', 'beta', 'GAMMA' })
decorate(bc, { { 0, 0, 0, 5, 'first' }, { 2, 0, 2, 5, 'third' } })
ok(apply.clear_at_line(bc, 0) == true, 'clear_at_line reports a cleared decoration')
local cmarks = vim.api.nvim_buf_get_extmarks(bc, apply.HL, 0, -1, {})
local cdiags = vim.diagnostic.get(bc, { namespace = apply.DIAG })
ok(#cmarks == 1 and cmarks[1][2] == 2, 'clear_at_line leaves other highlights intact')
ok(#cdiags == 1 and cdiags[1].lnum == 2, 'clear_at_line leaves other diagnostics intact')
ok(apply.clear_at_line(bc, 1) == false, 'clear_at_line reports false outside decorations')

OUT:write(fails == 0 and 'apply_test ok\n' or ('FAILED ' .. fails .. '\n'))
OUT:close()
LUA

PAIR_ROOT="$ROOT" RESULT="$RESULT" \
  XDG_STATE_HOME="$RT/state" XDG_DATA_HOME="$RT/xdg" XDG_CACHE_HOME="$RT/cache" \
  run_headless --timeout 30 -- nvim --headless -u NONE -c "luafile $RT/driver.lua" -c 'qa!'

echo "--- results ---"; cat "$RESULT"
if grep -q FAIL "$RESULT" || ! grep -q 'apply_test ok' "$RESULT"; then
  echo "review-apply-test FAILED"; exit 1
fi
echo "review-apply-test ok"
