local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local ok, T = pcall(dofile, here .. 'comment_thread.lua')
assert(ok, 'thread model must exist')
local P = dofile(here .. 'markers.lua')
local C = dofile(here .. 'comment_codec.lua')
local function eq(a, b, msg) assert(vim.deep_equal(a, b), msg .. ': ' .. vim.inspect({ got = a, want = b })) end
local function parse(raw) return assert(P.parse_markers({ raw })[1]) end
for n = 0, 4 do
  local wire = string.rep('\\', n) .. '<br>'
  local decoded = string.rep('\\', math.floor(n / 2)) .. (n % 2 == 0 and '\n' or '<br>')
  local lines, meta = T.to_lines(parse('🤖{' .. wire .. '}'))
  local expected = { '🤖: ' .. decoded, '💬: ' }
  if n % 2 == 0 then expected = { '🤖: ' .. string.rep('\\', math.floor(n / 2)), '', '💬: ' } end
  eq(lines, expected, 'independent float wire ' .. n)
  local raw = assert(T.from_lines(lines, meta))
  eq(C.turn_text(parse(raw), parse(raw).sections[1]), decoded, 'float fixture saving ' .. n)
end
eq(type(T.roles), 'function', 'thread role projection exists')
if T.roles then eq(T.roles({ '🤖: x', 'continued', '\\💬: literal', '💬: reply', '' }), { 'agent', 'agent', 'agent', 'user', 'user' }, 'roles follow unescaped turn headers') end
local raw = '🤖<literal <br>>{a<br>b}'
local lines, meta = T.to_lines(parse(raw))
eq(lines, { '🤖: a', 'b', '💬: ' }, 'chat prefixes and reply slot')
eq(T.from_lines(lines, meta), raw, 'empty appended reply omitted')
lines[4], lines[5] = '💬: revised', 'last'
lines[3] = '💬: prior edit'
local saved = assert(T.from_lines(lines, meta))
eq(saved, '🤖<literal <br>>{a<br>b}[prior edit][revised<br>last]', 'all turns editable; raw anchor retained')
local empty_lines, empty_meta = T.to_lines(parse('🤖{}[]'))
eq(T.from_lines(empty_lines, empty_meta), '🤖{}[]', 'preexisting empty human retained')
local _, err = T.from_lines({ 'no role prefix' }, meta)
assert(err, 'invalid structure refused')
local legacy = P.parse_markers({ '🤖{a', 'b}' })[1]
assert(not T.to_lines(legacy), 'legacy multiline cannot open')
assert(not T.to_lines(parse('🤖[okay]{broken')), 'incomplete prefix cannot open')
-- Role-looking continuation lines and every intentional trailing blank survive.
local texts = { '', '\n', 'x\n\n', '💬: x\n🤖: y', 'a\n\\💬: b\n\\\\🤖: c\n', '[{}] <br> \\', '字\n🤖: literal' }
for _, value in ipairs(texts) do
  local original = '🤖<' .. P.esc_quote('a > b <br>') .. '>[' .. C.encode_turn(value) .. ']'
  local rendered, metadata = assert(T.to_lines(parse(original)))
  local rebuilt = assert(T.from_lines(rendered, metadata))
  eq(rebuilt, original, 'lossless thread text ' .. vim.inspect(value))
end
-- Seeded content/role properties with full serialization consumption.
local seed, alphabet = 426, { 'x', '\n', '\\', '[', ']', '{', '}', '<br>', '💬: ', '🤖: ', '字', '`' }
for trial = 1, 250 do
  local turns, original = {}, '🤖<anchor>'
  for count = 1, 1 + trial % 7 do
    local pieces = {}
    for _ = 1, trial % 19 do
      seed = seed * 48271 % 2147483647
      pieces[#pieces + 1] = alphabet[seed % #alphabet + 1]
    end
    local content = table.concat(pieces)
    local role = count % 2 == 0 and 'user' or 'agent'
    turns[#turns + 1] = { type = role, text = content }
    original = original .. (role == 'user' and '[' or '{') .. C.encode_turn(content) .. (role == 'user' and ']' or '}')
  end
  local rendered, metadata = T.to_lines(parse(original))
  local rebuilt = assert(T.from_lines(rendered, metadata))
  local parsed = parse(rebuilt)
  eq(parsed.raw, rebuilt, 'exact parser consumption')
  eq(#parsed.sections, #turns, 'role count preservation')
  for i, turn in ipairs(turns) do
    eq(parsed.sections[i].type, turn.type, 'role preservation')
    eq(C.turn_text(parsed, parsed.sections[i]), turn.text, 'content preservation')
  end
end
-- Transaction state: no source overwrite and exactly one close effect.
local session = T.new_session('initial')
eq(session:snapshot().state, 'editing', 'open state')
local snapshot = session:snapshot()
snapshot.expected, snapshot.state = 'tampered', 'closed'
eq(session:snapshot(), { state = 'editing', expected = 'initial' }, 'snapshot cannot mutate authoritative lifecycle')
eq(session:transition({ kind = 'save', observed = 'changed', proposed = 'new' }).kind, 'refuse', 'changed source refused')
eq(session:snapshot().state, 'conflicted', 'conflicted state')
eq(session:snapshot().expected, 'initial', 'refusal never changes expected')
eq(session:transition({ kind = 'save', observed = 'initial', proposed = 'new' }), { kind = 'replace', raw = 'new' }, 'retry after undo')
eq(session:snapshot().expected, 'initial', 'effect not yet confirmation')
eq(session:transition({ kind = 'saved', raw = 'new' }).kind, 'none', 'confirmed write')
eq(session:snapshot().expected, 'new', 'successful save updates expectation')
eq(session:snapshot().state, 'editing', 'successful save clears conflict')
eq(session:transition({ kind = 'save', observed = 'new', error = 'invalid thread' }).kind, 'refuse', 'invalid draft refused')
eq(session:transition({ kind = 'save', proposed = 'newer' }).kind, 'refuse', 'missing source refused')
eq(session:transition({ kind = 'close', dirty = true }), { kind = 'close', rescue = true }, 'forced close rescues dirty draft')
eq(session:transition({ kind = 'close', dirty = true }).kind, 'none', 'duplicate close harmless')
eq(session:transition({ kind = 'save', observed = 'new', proposed = 'x' }).kind, 'none', 'closed session never writes')
local discarded = T.new_session('x')
eq(discarded:transition({ kind = 'close', dirty = true, discard = true }), { kind = 'close', rescue = false }, 'explicit discard')
local failed = T.new_session('x')
failed:transition({ kind = 'save', observed = 'x', proposed = 'y' })
eq(failed:transition({ kind = 'failed', error = 'write failed' }).kind, 'refuse', 'failed write visible')
eq(failed:snapshot().expected, 'x', 'failed write never changes expectation')
eq(failed:transition({ kind = 'saved', raw = 'y' }).kind, 'refuse', 'late confirmation refused')
-- Generated sequences against independent no-overwrite / no-loss invariants.
for trial = 1, 100 do
  local s, source, dirty, closed, closes = T.new_session('base'), 'base', false, false, 0
  for step = 1, 60 do
    seed = seed * 48271 % 2147483647
    local choice = seed % 7
    local event
    if choice == 0 then source = 'external' .. step
    elseif choice == 1 then source = s:snapshot().expected
    elseif choice == 2 then dirty = true
    elseif choice <= 4 then event = { kind = 'save', observed = source, proposed = 'draft' .. step }
    else event = { kind = 'close', dirty = dirty, discard = choice == 6 } end
    if event then
      local before = s:snapshot().expected
      local effect = s:transition(event)
      if closed then eq(effect.kind, 'none', 'closed resources never reactivate') end
      if effect.kind == 'replace' then
        eq(source, before, 'replace only matching known source')
        source = effect.raw
        s:transition({ kind = 'saved', raw = source })
        dirty = false
      elseif effect.kind == 'close' then
        closes = closes + 1
        eq(effect.rescue, dirty and not event.discard, 'close never loses undiscarded draft')
        closed = true
      else eq(s:snapshot().expected, before, 'non-write preserves known source') end
      assert(closes <= 1, 'only one resource teardown')
    end
  end
end
print('comment_thread_test ok')
