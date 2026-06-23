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

eq(M.default_mode(), 'edit', 'default mode id')
eq(M.normalize_mode(nil), 'edit', 'nil normalizes to default')
eq(M.normalize_mode(''), 'edit', 'empty normalizes to default')
eq(M.normalize_mode('developmental'), 'generate', 'legacy developmental normalizes to generate')
eq(M.normalize_mode('line-editing'), 'edit', 'legacy line-editing normalizes to edit')
eq(M.normalize_mode('copy-editing'), 'edit', 'legacy copy-editing normalizes to edit')
eq(M.normalize_mode('proofreading'), 'proofread', 'legacy proofreading normalizes to proofread')
eq(M.normalize_mode('free-form'), 'edit', 'legacy free-form normalizes to edit')
eq(M.normalize_mode('fact-check'), 'edit', 'legacy fact-check normalizes to edit')
eq(M.mode_label(nil), 'Edit', 'nil label defaults')
eq(M.mode_label(''), 'Edit', 'empty label defaults')
eq(M.mode_label('generate'), 'Generate', 'generate display')
eq(M.mode_label('edit'), 'Edit', 'edit display')
eq(M.mode_label('proofread'), 'Proofread', 'proofread display')
eq(M.mode_label('copy-editing'), 'Edit', 'legacy copy-editing display')

if fails > 0 then os.exit(1) end
print('seam_test ok')
