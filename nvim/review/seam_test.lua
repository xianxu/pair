-- nvim/review/seam_test.lua — run via `nvim -l nvim/review/seam_test.lua`.
-- Pure seam path + review-mode label/default helpers.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'seam.lua')

local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s: got %q want %q\n', msg, tostring(got), tostring(want)))
    fails = fails + 1
  end
end

eq(M.mode_path('/tmp/pair', 'abc'), '/tmp/pair/review-abc.mode', 'mode path tagged')
eq(M.mode_path('/tmp/pair', ''), '/tmp/pair/review-default.mode', 'mode path default tag')
eq(M.mode_path('', 'abc'), nil, 'mode path missing data dir')

eq(M.default_mode(), 'copy-editing', 'default mode id')
eq(M.mode_label(nil), 'Copy Edit', 'nil label defaults')
eq(M.mode_label(''), 'Copy Edit', 'empty label defaults')
eq(M.mode_label('copy-editing'), 'Copy Edit', 'copy-editing display')
eq(M.mode_label('fact-check'), 'Fact-check', 'hyphenated display')
eq(M.mode_label('free-form'), 'Free-form', 'free-form display')

if fails > 0 then os.exit(1) end
print('seam_test ok')
