#!/usr/bin/env bash
# tests/review-reconcile-test.sh — reconcile.reconcile_round on a live buffer
# (issue #89 M2). Clean records apply span-granularly; records whose span the human
# changed become 🤖<…>[reconcile — …] markers placed on the human's changed hunk
# (via real vim.diff). Runs under real `nvim --headless` (needs vim.diff + buffer
# API). Driver does its own asserts → result file.
#
# Run: bash tests/review-reconcile-test.sh
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
. "$ROOT/tests/lib/run-headless.sh"
RT="$(mktemp -d "${TMPDIR:-/tmp}/pair-reconcile-test.XXXXXX")"
trap 'rm -rf "$RT"' EXIT
RESULT="$RT/result.txt"

cat > "$RT/driver.lua" <<'LUA'
local ROOT = os.getenv('PAIR_ROOT')
local reconcile = dofile(ROOT .. '/nvim/review/reconcile.lua')
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
local function has(s, sub) return s:find(sub, 1, true) ~= nil end

-- (a) fast path via reconcile_round: v1 == v0, all clean → delegates to apply.apply
local b0 = newbuf({ 'hello world' })
local _, _, nc0 = reconcile.reconcile_round(b0, { { old = 'hello', occurrence = 1, new = 'HI', explain = 'g' } }, 'hello world')
ok(content(b0) == 'HI world', 'fast path (v1==v0) applies cleanly')
ok(nc0 == 0, 'fast path: no conflicts')

-- (b) clean-only under concurrent edit: human edited ELSEWHERE (added ' EXTRA');
--     the agent's target still anchors → applies, no conflict marker.
local v0b = 'alpha beta gamma'
local b1 = newbuf({ 'alpha beta gamma EXTRA' })
local _, _, nc1 = reconcile.reconcile_round(b1, { { old = 'beta', occurrence = 1, new = 'BETA', explain = 'x' } }, v0b)
ok(content(b1) == 'alpha BETA gamma EXTRA', 'clean-only: non-overlapping edit applies span-granularly')
ok(nc1 == 0, 'clean-only: no conflicts')
ok(not has(content(b1), '🤖'), 'clean-only: no reconcile marker')

-- (c) conflict: human changed the agent's exact target span → 🤖<…>[reconcile] marker
--     wrapping the human's current hunk text, carrying the agent's intent.
local v0c = 'alpha beta gamma'
local b2 = newbuf({ 'alpha CHANGED gamma' })
local _, _, nc2 = reconcile.reconcile_round(b2, { { old = 'beta', occurrence = 1, new = 'BETA', explain = 'reword' } }, v0c)
ok(nc2 == 1, 'conflict: one conflict reported')
ok(has(content(b2), '🤖<alpha CHANGED gamma>'), 'conflict: marker wraps the human hunk')
ok(has(content(b2), 'reconcile'), 'conflict: marker is a reconcile request')
ok(has(content(b2), 'beta') and has(content(b2), 'BETA'), 'conflict: carries the agent intent old→new')

-- (d) MIXED clean + conflict in ONE apply.apply call: clean record keeps its
--     DiffChange highlight, the synthetic conflict record self-highlights as a
--     marker (no redundant HL) and carries a 'reconcile' diagnosis.
local v0d = 'first line\nsecond line'
local b3 = newbuf({ 'first line', 'SECOND EDIT' })   -- human changed line 2
local en3, _, nc3 = reconcile.reconcile_round(b3, {
  { old = 'first', occurrence = 1, new = 'FIRST', explain = 'cap' },   -- clean (line 1 untouched)
  { old = 'second', occurrence = 1, new = 'SEC', explain = 'reword' }, -- conflict (line 2 changed)
}, v0d)
ok(has(content(b3), 'FIRST line'), 'mixed: clean record applied')
ok(has(content(b3), '🤖<SECOND EDIT>'), 'mixed: conflict marker wraps the changed line')
ok(has(content(b3), 'SEC'), 'mixed: conflict carries the agent intent')
ok(nc3 == 1, 'mixed: one conflict')
ok(#vim.api.nvim_buf_get_extmarks(b3, apply.HL, 0, -1, {}) == 1, 'mixed: only the clean edit gets a change highlight')
local msgs = {}
for _, d in ipairs(vim.diagnostic.get(b3, { namespace = apply.DIAG })) do msgs[d.message] = true end
ok(msgs['cap'] and msgs['reconcile'], 'mixed: clean explain + reconcile diagnosis both present')

-- (e) whole reconcile is ONE undo block (clean + conflict revert together)
vim.api.nvim_buf_call(b3, function() vim.cmd('silent undo') end)
ok(content(b3) == 'first line\nSECOND EDIT', 'single undo reverts the whole reconcile round')

-- (f) clean edit sharing a human-changed LINE with a conflict is FOLDED into the
-- marker, not dropped (M2-review 3.1). v0 line: "the foo and bar here"; human
-- changed bar→baz; agent round {foo→FOO clean, bar→BAR conflict}. The whole line
-- becomes one marker citing BOTH intents; FOO does not vanish, nothing dropped.
local v0f = 'the foo and bar here'
local b4 = newbuf({ 'the foo and baz here' })
local en4, dr4, nc4 = reconcile.reconcile_round(b4, {
  { old = 'foo', occurrence = 1, new = 'FOO', explain = 'clean' },
  { old = 'bar', occurrence = 1, new = 'BAR', explain = 'reword' },
}, v0f)
ok(has(content(b4), '🤖<the foo and baz here>'), 'fold: contested line becomes one marker')
ok(has(content(b4), 'FOO') and has(content(b4), 'BAR'), 'fold: marker cites BOTH the clean and conflict intents')
ok(#dr4 == 0, 'fold: nothing dropped (the clean edit was folded, not overlap-dropped)')
ok(nc4 == 1, 'fold: one synthetic conflict record for the line')

OUT:write(fails == 0 and 'reconcile_test ok\n' or ('FAILED ' .. fails .. '\n'))
OUT:close()
LUA

PAIR_ROOT="$ROOT" RESULT="$RESULT" \
  XDG_STATE_HOME="$RT/state" XDG_DATA_HOME="$RT/xdg" XDG_CACHE_HOME="$RT/cache" \
  run_headless --timeout 30 -- nvim --headless -u NONE -c "luafile $RT/driver.lua" -c 'qa!'

echo "--- results ---"; cat "$RESULT"
if grep -q FAIL "$RESULT" || ! grep -q 'reconcile_test ok' "$RESULT"; then
  echo "review-reconcile-test FAILED"; exit 1
fi
echo "review-reconcile-test ok"
