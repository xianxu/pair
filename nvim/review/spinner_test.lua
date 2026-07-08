-- nvim/review/spinner_test.lua — run via `nvim -l nvim/review/spinner_test.lua`
-- (or `make test-lua`). Pure; no IO.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'spinner.lua')

local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s: got %q want %q\n', msg, tostring(got), tostring(want)))
    fails = fails + 1
  end
end

eq(M.elapsed(0), '0s', 'elapsed 0')
eq(M.elapsed(45), '45s', 'elapsed seconds')
eq(M.elapsed(120), '2m 0s', 'elapsed minutes include seconds')
eq(M.elapsed(310), '5m 10s', 'elapsed minutes keep seconds')
eq(M.elapsed(3600), '1h 0m', 'elapsed hours include minutes')
eq(M.elapsed(3723), '1h 2m', 'elapsed hours drop seconds')
eq(M.elapsed(-5), '0s', 'elapsed clamps negative')

eq(M.cell(nil, 100, 0), '', 'idle → empty cell')
eq(M.cell(100, 145, 2), '⣻ 45s ', 'awaiting → frame + elapsed (tick 2 = 3rd frame)')
-- frame advances with tick; budget ≤6 visible cols for spinner+elapsed
local c = M.cell(100, 102, 0)
if #c == 0 or not c:find('2s', 1, true) then
  io.stderr:write('FAIL cell tick0: ' .. c .. '\n'); fails = fails + 1
end

if fails > 0 then os.exit(1) end
print('spinner_test ok')
