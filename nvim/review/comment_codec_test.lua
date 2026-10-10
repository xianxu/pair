local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local ok, C = pcall(dofile, here .. 'comment_codec.lua')
assert(ok, 'turn codec must exist')
local function eq(a, b, message) assert(a == b, message .. ': ' .. vim.inspect({ got = a, want = b })) end
local generic = dofile(here .. '../marker_codec.lua')
assert(type(generic.escape_quote_byte) == 'function', 'shared generic quote byte primitive')
for _, ch in ipairs({ '[', ']', '{', '}', '<', '>', '\\', 'x', '`' }) do
  eq(generic.escape_quote_byte(ch), generic.esc_quote(ch), 'generic byte contract')
end
-- Independent canonical wire contract, never passed through encode_turn.
for n = 0, 4 do
  eq(C.decode_turn(string.rep('\\', n) .. '<br>'),
    string.rep('\\', math.floor(n / 2)) .. (n % 2 == 0 and '\n' or '<br>'), 'wire parity ' .. n)
end
eq(C.encode_turn('`'), '\\`', 'backticks cannot span encoded turn boundaries')
eq(C.encode_turn('a\nb'), 'a<br>b', 'newline wire')
eq(C.encode_turn('<br>'), '\\<br>', 'literal token wire')
eq(C.decode_turn('\\[a\\]\\{b\\}\\<c\\>'), '[a]{b}<c>', 'existing delimiters')
eq(C.decode_turn('tail\\'), 'tail\\', 'trailing slash')
eq(C.decode_turn('\\z'), 'z', 'existing generic escape')
local P = dofile(here .. 'markers.lua')
local seed = 426
local alphabet = { '<br>', '\n', '\\', '[', ']', '{', '}', '<', '>', '字', '🤖', 'x', '`', '~' }
for trial = 1, 700 do
  local parts = {}
  for _ = 1, trial % 41 do
    seed = (seed * 48271) % 2147483647
    parts[#parts + 1] = alphabet[seed % #alphabet + 1]
  end
  local value = table.concat(parts)
  local encoded = C.encode_turn(value)
  eq(encoded:find('\n', 1, true), nil, 'wire is one line')
  eq(C.decode_turn(encoded), value, 'seeded roundtrip ' .. trial)
  -- Bracket parser contract is independently verified for every generated turn.
  local marker = P.parse_markers({ '🤖{' .. encoded .. '}' })[1]
  assert(marker and marker.raw == '🤖{' .. encoded .. '}', 'full parser consumption ' .. trial)
  eq(C.turn_text(marker, marker.sections[1]), value, 'parser payload ' .. trial)
end
local began = vim.uv.hrtime()
eq(C.decode_turn(string.rep('\\', 100000)), string.rep('\\', 50000), 'long slash run')
assert((vim.uv.hrtime() - began) / 1e6 < 100, 'slash runs must decode in linear time')
print('comment_codec_test ok')
