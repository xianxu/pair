-- nvim/review/wrap_test.lua — run via `nvim -l nvim/review/wrap_test.lua`
-- (or `make test-lua`). Pure; no buffer/IO. Models record_test.lua.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'wrap.lua')

local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s:\n  got  %q\n  want %q\n', msg, tostring(got), tostring(want)))
    fails = fails + 1
  end
end

eq(M.wrap('a b c', 80), 'a b c', 'short text unchanged')
eq(M.wrap('aaa bbb ccc', 7), 'aaa bbb\nccc', 'greedy wrap at width 7')
eq(M.wrap('one two three', 3), 'one\ntwo\nthree', 'each word its own line at width 3')
eq(M.wrap('superlongword x', 4), 'superlongword\nx', 'word longer than width is not split')
eq(M.wrap('para one\n\npara two', 80), 'para one\n\npara two', 'preserves blank-line paragraph breaks')
-- idempotent at the same width (re-wrapping wrapped text reflows to the same)
eq(M.wrap(M.wrap('aaa bbb ccc', 7), 7), M.wrap('aaa bbb ccc', 7), 'idempotent at the same width')

if fails > 0 then os.exit(1) end
print('wrap_test ok')
