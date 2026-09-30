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

-- `!!` (#358): set the description after the fact, never send.
local function check_describe(input, want, name)
  local got = bang_tag.parse(input)
  assert(got ~= nil and got.agent_text == nil, name .. ': expected a describe tag, got ' .. vim.inspect(got))
  assert((got.describe_previous == true) == (want == nil), name .. ': describe_previous = ' .. vim.inspect(got))
  assert(got.description == want, name .. ': description = ' .. vim.inspect(got.description))
end

check_describe('!!', nil, 'bare !! describes the previous prompt')
check_describe('  !!  \n', nil, 'bare !! with surrounding whitespace')
check_describe('!! set after the fact', 'set after the fact', '!! sentence')
check_describe('!!set after the fact', 'set after the fact', '!!sentence without space')
check_describe('!!   spaced\t out  ', 'spaced out', '!! sentence collapses whitespace')
check_describe('!!! still text', '! still text', 'third bang is sentence text')
check('!! first\nsecond', nil, nil, 'multi-line !! is not a tag')

local function check_line(input, want, name)
  local got = bang_tag.one_line(input)
  assert(got == want, name .. ': one_line = ' .. vim.inspect(got))
end

check_line('\n\n  fix the   parser\tbug\nthen add tests\n', 'fix the parser bug', 'first non-blank line, collapsed')
check_line('  \n \t\n', nil, 'blank text has no line')
check_line(string.rep('a', 120), string.rep('a', 120), 'exactly the cap is kept')
check_line(string.rep('a', 121), string.rep('a', 119) .. '…', 'over the cap is cut with an ellipsis')
check_line(string.rep('é', 121), string.rep('é', 119) .. '…', 'cap counts characters, not bytes')

local function check_previous(entry, want, name)
  local got = bang_tag.previous_description(entry)
  assert(got == want, name .. ': previous_description = ' .. vim.inspect(got))
end

check_previous('refactor the\nsubmission path', 'refactor the', 'multi-line prompt keeps its first line')
check_previous('! start working on #358', 'start working on #358', 'bang prompt drops its !')
check_previous('!', nil, 'bare ! prompt has nothing to describe')
check_previous('', nil, 'empty entry has nothing to describe')
check_previous('! first\nsecond', '! first', 'multi-line ! reached the agent verbatim')

print('bang_tag_test: all passed')
