-- Headless tests for nvim/scrollback.lua — run via `nvim -l nvim/scrollback_test.lua`
-- (or `make test-lua`). Pure Lua; exits non-zero on failure.
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'

-- Load scrollback.lua which defines _G.PairScrollbackTest
dofile(here .. 'scrollback.lua')

local fails = 0
local function eq(got, want, msg)
  if got ~= want then
    io.stderr:write(string.format('FAIL %s: got %q want %q\n', msg, tostring(got), tostring(want)))
    fails = fails + 1
  end
end

local M = _G.PairScrollbackTest
assert(M, "_G.PairScrollbackTest must be defined")

-- Headless Neovim test: we have the vim API available!
if vim and vim.api then
  local function test_agent_pattern(agent, lines_to_test, expected_matches)
    vim.env.PAIR_AGENT = agent
    local pat = M.prompt_pattern()
    
    local buf = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_lines(buf, 0, -1, false, lines_to_test)
    
    -- Open a temporary window so we can set cursor and run vim.fn.search
    local win = vim.api.nvim_open_win(buf, true, {
      relative = 'editor',
      width = 80,
      height = 20,
      row = 0,
      col = 0,
      style = 'minimal'
    })
    
    -- For each expected match, we search from the top and see which lines match
    local matches = {}
    vim.api.nvim_win_set_cursor(win, {1, 0})
    
    local last_match = 0
    local is_first = true
    while true do
      local flags = 'W'
      if is_first then
        flags = 'cW'
        is_first = false
      end
      local match_line = vim.fn.search(pat, flags)
      if match_line == 0 or (last_match > 0 and match_line <= last_match) then
        break
      end
      matches[match_line] = true
      last_match = match_line
      -- Move cursor to avoid infinite loop
      vim.api.nvim_win_set_cursor(win, {match_line, 0})
    end
    
    vim.api.nvim_win_close(win, true)
    
    -- Assert matches
    for idx, expected in ipairs(expected_matches) do
      local got = matches[idx] or false
      eq(got, expected, string.format('Agent %s: line %d %q', agent, idx, lines_to_test[idx]))
    end
  end

  -- 1. Test Claude pattern
  test_agent_pattern('claude', {
    "❯ hello",             -- 1: match
    "❯ real prompt",       -- 2: match
    "  ❯ indented",        -- 3: no match
    "> blockquote",        -- 4: no match
  }, { true, true, false, false })

  -- 2. Test Codex pattern
  test_agent_pattern('codex', {
    "› hello",             -- 1: match
    "› real prompt",       -- 2: match
    "  › indented",        -- 3: no match
    "> blockquote",        -- 4: no match
  }, { true, true, false, false })

  -- 3. Test Agy pattern (including lookbehind rule)
  test_agent_pattern('agy', {
    "────────────────────────────────────────────────────────────",
    "> hello",             -- 2: match (preceded by horizontal rule)
    "──────────────────────────",
    "> real prompt",       -- 4: match (preceded by horizontal rule)
    "> blockquote",        -- 5: no match (no horizontal rule)
    "  > indented",        -- 6: no match (no horizontal rule, and indented)
    "some text",
    "> quoted in markdown",-- 8: no match (preceded by text, no horizontal rule)
  }, { false, true, false, true, false, false, false, false })

else
  io.stderr:write('WARN: vim.api unavailable — prompt pattern tests skipped\n')
end

if fails > 0 then
  io.stderr:write(fails .. ' failure(s)\n')
  os.exit(1)
end
print('nvim/scrollback.lua: prompt pattern tests passed')
