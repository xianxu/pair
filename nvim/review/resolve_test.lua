-- nvim/review/resolve_test.lua — run via `nvim -l nvim/review/resolve_test.lua`
-- (or `make test-lua`). Pure; no buffer/IO. One case per review-convention §5 row.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local R = dofile(here .. 'resolve.lua')

local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s:\n  got  %s\n  want %s\n', msg, vim.inspect(got), vim.inspect(want)))
    fails = fails + 1
  end
end

local function user(t) return { type = 'user', text = t } end
local function agent(t) return { type = 'agent', text = t } end

-- 🤖[H]            accept → ''   reject → ''
local hH = { sections = { user('H') } }
eq(R.resolve(hH, 'accept'), '', '🤖[H] accept')
eq(R.resolve(hH, 'reject'), '', '🤖[H] reject (remove markup)')

-- 🤖<X>[H]         accept → X    reject → X
local qXH = { quoted = { text = 'X' }, sections = { user('H') } }
eq(R.resolve(qXH, 'accept'), 'X', '🤖<X>[H] accept')
eq(R.resolve(qXH, 'reject'), 'X', '🤖<X>[H] reject (keep quoted)')

-- 🤖<old>{new}     accept → new  reject → old
local qXaN = { quoted = { text = 'old' }, sections = { agent('new') } }
eq(R.resolve(qXaN, 'accept'), 'new', '🤖<old>{new} accept (apply agent replacement)')
eq(R.resolve(qXaN, 'reject'), 'old', '🤖<old>{new} reject (keep quoted)')

-- 🤖{R}            accept → R    reject → ''
local aR = { sections = { agent('R') } }
eq(R.resolve(aR, 'accept'), 'R', '🤖{R} accept')
eq(R.resolve(aR, 'reject'), '', '🤖{R} reject (discard)')

-- 🤖[H]{R}         accept → ''   reject → ''
local hHaR = { sections = { user('H'), agent('R') } }
eq(R.resolve(hHaR, 'accept'), '', '🤖[H]{R} accept')
eq(R.resolve(hHaR, 'reject'), '', '🤖[H]{R} reject (remove markup)')

-- 🤖{R}[H]         accept → ''   reject → ''
local aRhH = { sections = { agent('R'), user('H') } }
eq(R.resolve(aRhH, 'accept'), '', '🤖{R}[H] accept')
eq(R.resolve(aRhH, 'reject'), '', '🤖{R}[H] reject (remove markup)')

-- 🤖~D~            accept → '' (delete)   reject → D
local sD = { strike = { text = 'D' }, sections = {} }
eq(R.resolve(sD, 'accept'), '', '🤖~D~ accept (delete)')
eq(R.resolve(sD, 'reject'), 'D', '🤖~D~ reject (keep D)')

-- 🤖~D~{N}         accept → N    reject → D
local sDaN = { strike = { text = 'D' }, sections = { agent('N') } }
eq(R.resolve(sDaN, 'accept'), 'N', '🤖~D~{N} accept (apply N)')
eq(R.resolve(sDaN, 'reject'), 'D', '🤖~D~{N} reject (keep D)')

-- 🤖~D~[N]         accept → N    reject → D
local sDuN = { strike = { text = 'D' }, sections = { user('N') } }
eq(R.resolve(sDuN, 'accept'), 'N', '🤖~D~[N] accept (apply N)')
eq(R.resolve(sDuN, 'reject'), 'D', '🤖~D~[N] reject (keep D)')

-- longer chain 🤖{R}[H]{R2}    accept → ''   reject → ''
local chain = { sections = { agent('R'), user('H'), agent('R2') } }
eq(R.resolve(chain, 'accept'), '', 'longer chain accept (discard)')
eq(R.resolve(chain, 'reject'), '', 'longer chain reject (remove markup)')


-- Canonical fixtures authored independently of the encoder (#426).
local parser = dofile(here .. 'markers.lua')
for slashes = 0, 4 do
  local wire = string.rep('\\', slashes) .. '<br>'
  local want = string.rep('\\', math.floor(slashes / 2)) .. (slashes % 2 == 0 and '\n' or '<br>')
  for _, prefix in ipairs({ '🤖', '🤖<anchor>', '🤖~old~' }) do
    local parsed = parser.parse_markers({ prefix .. '{' .. wire .. '}' })[1]
    eq(R.resolve(parsed, 'accept'), want, 'canonical resolution parity ' .. slashes)
  end
end
local anchor = parser.parse_markers({ '🤖<literal <br>>{a<br>b}' })[1]
eq(R.resolve(anchor, 'reject'), 'literal <br>', 'anchor stays literal')
local legacy = parser.parse_markers({ '🤖{a<br>', 'b}' })[1]
eq(R.resolve(legacy, 'accept'), 'a<br>\nb', 'legacy multiline stays literal')

if fails > 0 then os.exit(1) end
print('resolve_test ok')
