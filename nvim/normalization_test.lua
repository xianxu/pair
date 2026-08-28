local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'normalization.lua')
local raw = table.concat(vim.fn.readfile(here .. '../cmd/internal/sessioninventory/testdata/normalization/v1.json'), '\n')
local golden = vim.json.decode(raw)

local fails = 0
for _, case in ipairs(golden.cases) do
  local got = M.normalize_pair_text(case.input)
  if got ~= case.want then
    io.stderr:write(string.format('FAIL %s: got %q want %q\n', case.name, got, case.want))
    fails = fails + 1
  end
end
if fails > 0 then os.exit(1) end
print('all normalization.lua tests passed')
