-- nvim/review/resolve_fixtures_test.lua — run via `nvim -l nvim/review/resolve_fixtures_test.lua`.
-- The review pane's resolver must agree with Couch's Go resolver on every
-- shared fixture in testdata/edit-records (pair#661). Exits non-zero on failure.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'reconstruct.lua')
local dir = here .. '../../testdata/edit-records/'
local fails, seen = 0, 0
local function fail(name, msg)
  io.stderr:write(string.format('FAIL %s: %s\n', name, msg))
  fails = fails + 1
end

for _, file in ipairs(vim.fn.glob(dir .. '*.json', false, true)) do
  seen = seen + 1
  local fx = vim.json.decode(table.concat(vim.fn.readfile(file, 'b'), '\n'))
  local records = fx.records
  if type(records) == 'table' and records.records then records = records.records end
  local text, bad = M.resolve_text(fx.base, records)
  if fx.result ~= nil then
    if text ~= fx.result then fail(fx.name, string.format('text %q want %q', tostring(text), fx.result)) end
  else
    local got = {}
    for _, b in ipairs(bad) do got[#got + 1] = b.index .. ':' .. b.reason end
    local want = {}
    for _, b in ipairs(fx.invalid) do want[#want + 1] = b.index .. ':' .. b.reason end
    if table.concat(got, ',') ~= table.concat(want, ',') then
      fail(fx.name, string.format('bad %s want %s', table.concat(got, ','), table.concat(want, ',')))
    end
  end
end
if seen == 0 then fail('fixtures', 'none found under ' .. dir) end
if fails > 0 then os.exit(1) end
print(string.format('ok — %d shared edit-record fixtures', seen))
