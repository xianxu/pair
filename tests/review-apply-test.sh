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

-- (f) self-overlapping `new` adjacent to identical bytes: LIVE decoration must
-- NOT be dropped (milestone review I1). base 'aX' → 'aaa'.
local b7 = newbuf({ 'aX' })
apply.apply(b7, { { old = 'X', occurrence = 1, new = 'aa', explain = 'overlap' } })
ok(content(b7) == 'aaa', 'self-overlapping edit applied')
ok(#vim.api.nvim_buf_get_extmarks(b7, apply.HL, 0, -1, {}) >= 1, 'self-overlap: live decoration not dropped')
ok(#vim.diagnostic.get(b7) >= 1, 'self-overlap: diagnostic present')

-- (g) apply with a DIFFERENT buffer focused: the single-undo-block must still
-- hold (milestone review I2 — undojoin must target `buf`, not the current one).
local target = newbuf({ 'aaa bbb ccc' })
local other = vim.api.nvim_create_buf(true, false)
vim.api.nvim_set_current_buf(other)
apply.apply(target, {
  { old = 'aaa', occurrence = 1, new = 'AAA', explain = '1' },
  { old = 'ccc', occurrence = 1, new = 'CCC', explain = '2' },
})
ok(content(target) == 'AAA bbb CCC', 'applied to non-current buffer')
vim.api.nvim_buf_call(target, function() vim.cmd('silent undo') end)
ok(content(target) == 'aaa bbb ccc', 'single undo reverts whole round when buf not current')

-- (h) partial anchor failure: anchored subset applies, the rest reported dropped
local b8 = newbuf({ 'hello world' })
local en8, dr8 = apply.apply(b8, {
  { old = 'hello', occurrence = 1, new = 'HI', explain = 'ok' },
  { old = 'missing', occurrence = 1, new = 'X', explain = 'no' },
  { old = '', occurrence = 1, new = 'Y', explain = 'empty' },
})
ok(#en8 == 1 and content(b8) == 'HI world', 'partial: anchored subset applied')
ok(#dr8 == 2, 'partial: unanchorable records reported as dropped (not silent)')

-- (i) total failure: buffer untouched, all dropped with a reason
local b9 = newbuf({ 'abc' })
local en9, dr9 = apply.apply(b9, { { old = 'zzz', occurrence = 1, new = 'Z', explain = 'no' } })
ok(#en9 == 0 and content(b9) == 'abc', 'total failure: buffer untouched')
ok(#dr9 == 1 and dr9[1].reason == 'not found', 'total failure: dropped with reason')

-- (j) overlap: the intersecting record is dropped (no silent corruption)
local b10 = newbuf({ 'abcdef' })
local en10, dr10 = apply.apply(b10, {
  { old = 'abcd', occurrence = 1, new = '1', explain = 'first' },
  { old = 'cdef', occurrence = 1, new = '2', explain = 'overlaps' },
})
ok(#en10 == 1 and content(b10) == '1ef', 'overlap: only the non-overlapping record applied')
ok(#dr10 == 1 and dr10[1].reason == 'overlap', 'overlap: intersecting record dropped with reason')

-- (k) live decoration lines EQUAL the resume(reconstruct) lines — the
-- new_occurrence invariant that keeps live-render and resume-render identical.
local bk = newbuf({ 'one two three four' })
local ek = apply.apply(bk, {
  { old = 'two', occurrence = 1, new = 'TWO', explain = 'a' },
  { old = 'four', occurrence = 1, new = 'IV', explain = 'b' },
})
local live, res = {}, {}
for _, m in ipairs(vim.api.nvim_buf_get_extmarks(bk, apply.HL, 0, -1, {})) do live[m[2]] = true end
for _, h in ipairs(reconstruct.decorate(ek, content(bk), 'new').highlights) do res[h.line] = true end
local same = true
for l in pairs(live) do if not res[l] then same = false end end
for l in pairs(res) do if not live[l] then same = false end end
ok(same, 'live decoration lines == reconstruct (resume) lines')

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

local brm = newbuf({ 'prefix 🤖<bad>{GOOD} suffix' })
apply.render(brm, { { new = '🤖<bad>{GOOD}', new_occurrence = 1, explain = 'proposal' } }, content(brm))
ok(#vim.api.nvim_buf_get_extmarks(brm, apply.HL, 0, -1, {}) == 0, 'render marker proposal: no redundant highlight')
local rmd = vim.diagnostic.get(brm, { namespace = apply.DIAG })
ok(#rmd == 1 and rmd[1].message == 'proposal', 'render marker proposal keeps diagnostic')

-- (m) snapshot/apply_snapshot round-trip, including exact columns and a
-- MULTI-LINE decoration — the span must survive projection.
local bs = newbuf({ 'p1', 'old', 'q', 'tail' })
apply.apply(bs, {
  { old = 'old', occurrence = 1, new = 'NEW1\nNEW2', explain = 'multi' },
  { old = 'tail', occurrence = 1, new = 'T', explain = 'single' },
})
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

-- (n) Direct-rendered changes highlight only the inserted `new` bytes, not the
-- whole line/paragraph.
local bn = newbuf({ 'prefix bad suffix' })
apply.apply(bn, { { old = 'bad', occurrence = 1, new = 'GOOD', explain = 'grammar' } })
local em = vim.api.nvim_buf_get_extmarks(bn, apply.HL, 0, -1, { details = true })[1]
ok(em ~= nil, 'direct edit: highlight extmark placed')
ok(em ~= nil and em[2] == 0 and em[3] == #'prefix ' and em[4].end_row == 0 and em[4].end_col == #'prefix GOOD',
  'direct edit highlight spans only inserted new text')

-- (o) Marker-rendered proposals carry their own visible delta, so the renderer
-- keeps diagnostics but does not add a redundant change highlight.
local bm = newbuf({ 'prefix bad suffix' })
apply.apply(bm, { { old = 'bad', occurrence = 1, new = '🤖<bad>{GOOD}', explain = 'proposal' } })
ok(content(bm) == 'prefix 🤖<bad>{GOOD} suffix', 'marker proposal inserted')
ok(#vim.api.nvim_buf_get_extmarks(bm, apply.HL, 0, -1, {}) == 0, 'marker proposal: no redundant highlight')
local md = vim.diagnostic.get(bm, { namespace = apply.DIAG })
ok(#md == 1 and md[1].message == 'proposal', 'marker proposal keeps diagnostic')

-- (p) Empty direct deletions have no new span to highlight. Agents should use
-- 🤖~deleted~ when a deletion needs visible review, but diagnostics still carry
-- the reason if a direct deletion lands.
local bd = newbuf({ 'remove me' })
apply.apply(bd, { { old = 'remove ', occurrence = 1, new = '', explain = 'delete' } })
ok(content(bd) == 'me', 'empty direct deletion applied')
ok(#vim.api.nvim_buf_get_extmarks(bd, apply.HL, 0, -1, {}) == 0, 'empty direct deletion: no fake highlight')
local dd = vim.diagnostic.get(bd, { namespace = apply.DIAG })
ok(#dd == 1 and dd[1].message == 'delete', 'empty direct deletion keeps diagnostic')

-- (q) targeted clear: accepting a styled region without a marker clears only
-- that region's review highlight + matching diagnostic.
local bc = newbuf({ 'alpha', 'beta', 'gamma' })
apply.apply(bc, {
  { old = 'alpha', occurrence = 1, new = 'ALPHA', explain = 'first' },
  { old = 'gamma', occurrence = 1, new = 'GAMMA', explain = 'third' },
})
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
  run_headless --timeout 30 -- nvim --headless -u NONE -c "luafile $RT/driver.lua" -c 'qa!'

echo "--- results ---"; cat "$RESULT"
if grep -q FAIL "$RESULT" || ! grep -q 'apply_test ok' "$RESULT"; then
  echo "review-apply-test FAILED"; exit 1
fi
echo "review-apply-test ok"
