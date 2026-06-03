-- adapt_test.lua — headless test for nvim/adapt.lua's emitter.
-- Run: nvim -l nvim/adapt_test.lua   (wired into `make test-lua`)
--
-- Guards the M2-review finding: detail truncation must be rune-safe so the Lua
-- emitter never writes invalid UTF-8 (which would break the JSON line and
-- diverge from the byte-identical Go/shell contract).

local here = debug.getinfo(1, 'S').source:sub(2):match('(.*/)') or './'
local adapt = dofile(here .. 'adapt.lua')

local tmp = os.getenv('TMPDIR') or '/tmp'
vim.env.PAIR_TAG = 'adapttest'
vim.env.PAIR_DATA_DIR = tmp
vim.env.PAIR_AGENT = 'codex'
local path = tmp .. '/adapt-adapttest.jsonl'
os.remove(path)

-- 100 × 'あ' (U+3042, 3 bytes) = 300 bytes; must cap to <=200 bytes WITHOUT
-- splitting a rune. Old byte-sub cut at 200 (= 66*3 + 2) → mid-rune → invalid.
local long = string.rep('あ', 100)
adapt.log(2, 'overlay-detect', 'near-miss', long)

local f = assert(io.open(path, 'r'), 'adapt.log wrote no file')
local line = f:read('*l')
f:close()
os.remove(path)

-- Must parse: a split rune would make the line invalid UTF-8 / JSON.
local ok, obj = pcall(vim.json.decode, line)
assert(ok, 'emitted line is not valid JSON (split rune?): ' .. tostring(line))
assert(obj.detail, 'detail field missing')
assert(#obj.detail <= 200, 'detail not capped to 200 bytes: ' .. #obj.detail)
assert(#obj.detail % 3 == 0, 'truncation split a 3-byte rune: ' .. #obj.detail .. ' bytes')
-- Field-order/shape sanity (the schema the doctor relies on).
for _, k in ipairs({ 'ts', 'comp', 'agent', 'aspect', 'signal', 'outcome', 'detail' }) do
  assert(obj[k] ~= nil, 'missing field: ' .. k)
end

print('nvim/adapt.lua: emitter tests passed')
