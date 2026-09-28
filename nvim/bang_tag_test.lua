local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local bang_tag = dofile(here .. 'bang_tag.lua')

local function check(input, want_text, want_desc, name)
  local got = bang_tag.parse(input)
  if want_text == nil then
    assert(got == nil, name .. ': expected no tag, got ' .. vim.inspect(got))
    return
  end
  assert(got ~= nil, name .. ': expected a tag')
  assert(got.agent_text == want_text, name .. ': agent_text = ' .. vim.inspect(got.agent_text))
  assert(got.description == want_desc, name .. ': description = ' .. vim.inspect(got.description))
end

check('! start working on #337', 'start working on #337', 'start working on #337', 'bang with space')
check('!start', 'start', 'start', 'bang without space')
check('!   spaced   out  ', 'spaced   out', 'spaced   out', 'inner spacing kept, ends trimmed')
check('\n  ! wrapped in blank lines\n\n', 'wrapped in blank lines', 'wrapped in blank lines', 'surrounding blank lines ignored')
check('! -starts with a dash', '-starts with a dash', '-starts with a dash', 'leading dash is text')
check('!\t tab after bang', 'tab after bang', 'tab after bang', 'tab after bang')

check('!', '', nil, 'bare bang sends nothing')
check('!   ', '', nil, 'bang with only spaces sends nothing')

check('! first line\nsecond line', nil, nil, 'multi-line draft is not a tag')
check('first line\n! second line', nil, nil, 'bang on a later line is not a tag')
check('start working on #337', nil, nil, 'no bang')
check('say !hello', nil, nil, 'bang mid-line')
check('', nil, nil, 'empty')

print('bang_tag_test: all passed')
