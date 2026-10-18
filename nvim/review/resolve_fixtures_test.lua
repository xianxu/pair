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

-- Propose fixtures: Couch writes the markers; the pane's grammar must read
-- each one back so that accepting every marker gives `accepted` and rejecting
-- every marker gives `rejected` (one marker per changed line, pair#661).
local markers = dofile(here .. 'markers.lua')
local function settle(text, accept)
  local lines = vim.split(text, '\n', { plain = true })
  local found = markers.parse_markers(lines)
  for i = #found, 1, -1 do
    local m = found[i]
    local agent = ''
    for _, sec in ipairs(m.sections) do if sec.type == 'agent' then agent = sec.text end end
    local line = lines[m.line + 1]
    local keep = accept and agent or (m.strike and m.strike.text or '')
    lines[m.line + 1] = line:sub(1, m.col) .. keep .. line:sub(m.col + #m.raw + 1)
  end
  return table.concat(lines, '\n'), #found
end
local proposed = 0
for _, file in ipairs(vim.fn.glob(dir .. 'propose/*.json', false, true)) do
  local fx = vim.json.decode(table.concat(vim.fn.readfile(file, 'b'), '\n'))
  if fx.markers then
    proposed = proposed + 1
    local acc, n = settle(fx.markers, true)
    local rej = settle(fx.markers, false)
    -- Every 🤖 Couch writes must parse; a bare or broken one never settles.
    local _, robots = fx.markers:gsub('🤖', '')
    if n ~= robots then fail(fx.name, string.format('%d of %d markers parsed', n, robots)) end
    if acc ~= fx.accepted then fail(fx.name, string.format('accepted %q want %q', acc, fx.accepted)) end
    if rej ~= fx.rejected then fail(fx.name, string.format('rejected %q want %q', rej, fx.rejected)) end
  end
end
if proposed == 0 then fail('propose fixtures', 'none found') end
if fails > 0 then os.exit(1) end
print(string.format('ok — %d shared edit-record fixtures, %d propose fixtures', seen, proposed))
