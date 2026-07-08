local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local define = dofile(here .. 'define.lua')

local failures = 0

local function inspect(v)
  if vim and vim.inspect then return vim.inspect(v) end
  if type(v) ~= 'table' then return tostring(v) end
  local parts = {}
  for k, val in pairs(v) do
    parts[#parts + 1] = tostring(k) .. '=' .. inspect(val)
  end
  table.sort(parts)
  return '{' .. table.concat(parts, ',') .. '}'
end

local function same(a, b)
  if type(a) ~= type(b) then return false end
  if type(a) ~= 'table' then return a == b end
  for k, v in pairs(a) do
    if not same(v, b[k]) then return false end
  end
  for k in pairs(b) do
    if a[k] == nil then return false end
  end
  return true
end

local function eq(got, want, msg)
  if same(got, want) then return end
  failures = failures + 1
  io.stderr:write('FAIL ', msg, '\n  got:  ', inspect(got), '\n  want: ', inspect(want), '\n')
end

eq(define.footnote_id('Amazon Standard Identification Number'),
  'amazon-standard-identification-number',
  'footnote_id slugifies prose')
eq(define.footnote_id(''), 'definition', 'empty footnote id falls back')

eq(define.slice_selection({ 'before ASIN after' }, 1, 7, 1, 10), 'ASIN',
  'slice_selection extracts single-line inclusive visual span')
eq(define.slice_selection({ 'aa bb', 'cc dd' }, 1, 3, 2, 1), 'bb\ncc',
  'slice_selection extracts multi-line inclusive visual span')

local applied = define.apply_definition_footnote(
  { 'here is ASIN in context' },
  1, 8, 1, 11,
  'ASIN',
  'Amazon Standard Identification Number.'
)
eq(applied.lines, {
  'here is ASIN[^asin] in context',
  '',
  '---',
  '',
  '[^asin]: Amazon Standard Identification Number.',
}, 'apply_definition_footnote inserts inline ref and managed footer')
eq(applied.diagnostic_span, {
  line = 0,
  col = 8,
  end_line = 0,
  end_col = 19,
}, 'apply_definition_footnote returns exact selected-ref span')

local redefined = define.apply_definition_footnote(applied.lines, 1, 8, 1, 11, 'ASIN', 'Updated.')
eq(redefined.lines, {
  'here is ASIN[^asin] in context',
  '',
  '---',
  '',
  '[^asin]: Updated.',
}, 'redefining updates the footer without duplicating the inline reference')

local ordinary = table.concat({
  'main text',
  '',
  '---',
  '',
  'not a footnote',
}, '\n')
eq(define.strip_definition_footnote_footer(ordinary), ordinary,
  'strip_definition_footnote_footer preserves ordinary trailing divider prose')
eq(define.strip_definition_footnote_footer(table.concat(redefined.lines, '\n')),
  'here is ASIN[^asin] in context',
  'strip_definition_footnote_footer removes only final managed footnote footer')

eq(define.footnote_diagnostics(redefined.lines), {
  {
    id = 'asin',
    term = 'ASIN',
    definition = 'Updated.',
    line = 0,
    col = 8,
    end_line = 0,
    end_col = 19,
  },
}, 'footnote_diagnostics derives exact span and stored definition')

if failures > 0 then
  error(string.format('define_test failed: %d failure(s)', failures))
end

print('define_test ok')
