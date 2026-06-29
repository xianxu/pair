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

local function tmpdir()
  local base = vim.fn.tempname()
  vim.fn.mkdir(base, 'p')
  return base
end

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

  -- 4. Refresh helper: re-renders the backing .ansi file, reloads this buffer,
  -- strips ANSI escapes back to text, and preserves read-only viewer state.
  do
    assert(type(M.refresh_buffer) == 'function', 'refresh_buffer helper must exist')
    local dir = tmpdir()
    local ansi = dir .. '/scrollback-test-codex.ansi'
    local raw = dir .. '/scrollback-test-codex.raw'
    local events = dir .. '/scrollback-test-codex.events.jsonl'
    vim.fn.writefile({ 'old line' }, ansi)
    vim.fn.writefile({ 'raw' }, raw)
    vim.fn.writefile({ '{"type":"resize","width":80,"height":24,"offset":0}' }, events)

    local buf = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(buf, ansi)
    ansi = vim.api.nvim_buf_get_name(buf)
    raw = ansi:gsub('%.ansi$', '.raw')
    events = ansi:gsub('%.ansi$', '.events.jsonl')
    vim.api.nvim_buf_set_lines(buf, 0, -1, false, { 'old line' })
    vim.bo[buf].modifiable = false
    vim.bo[buf].readonly = true

    local called = false
    local ok = M.refresh_buffer(buf, {
      renderer = function(paths)
        called = true
        eq(paths.ansi, ansi, 'refresh passes ansi path')
        eq(paths.raw, raw, 'refresh derives raw path')
        eq(paths.events, events, 'refresh derives events path')
        vim.fn.writefile({ '\27[31mnew red\27[0m', 'new tail' }, ansi)
        return true
      end,
    })
    eq(ok, true, 'refresh_buffer returns true')
    eq(called, true, 'refresh renderer called')
    eq(table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n'), 'new red\nnew tail',
       'refresh reloads and strips ANSI')
    eq(vim.bo[buf].modifiable, false, 'refresh leaves buffer unmodifiable')
    eq(vim.bo[buf].readonly, true, 'refresh leaves buffer readonly')
  end

  -- 5. Refresh failure: keep the old visible buffer and locked viewer state.
  do
    local dir = tmpdir()
    local ansi = dir .. '/scrollback-test-codex.ansi'
    vim.fn.writefile({ 'new on disk' }, ansi)

    local buf = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(buf, ansi)
    vim.api.nvim_buf_set_lines(buf, 0, -1, false, { 'old visible', 'still here' })
    vim.bo[buf].modifiable = false
    vim.bo[buf].readonly = true

    local ok = M.refresh_buffer(buf, {
      renderer = function()
        return false, 'boom'
      end,
    })
    eq(ok, false, 'refresh_buffer returns false on renderer failure')
    eq(table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n'),
       'old visible\nstill here',
       'renderer failure keeps visible buffer intact')
    eq(vim.bo[buf].modifiable, false, 'renderer failure leaves buffer unmodifiable')
    eq(vim.bo[buf].readonly, true, 'renderer failure leaves buffer readonly')
  end

  -- 6. G behavior: refresh first, then land at the refreshed end.
  do
    assert(type(M.refresh_then_end) == 'function', 'refresh_then_end helper must exist')
    local dir = tmpdir()
    local ansi = dir .. '/scrollback-test-codex.ansi'
    local raw = dir .. '/scrollback-test-codex.raw'
    local events = dir .. '/scrollback-test-codex.events.jsonl'
    vim.fn.writefile({ 'one', 'two' }, ansi)
    vim.fn.writefile({ 'raw' }, raw)
    vim.fn.writefile({ '{"type":"resize","width":80,"height":24,"offset":0}' }, events)

    local buf = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(buf, ansi)
    ansi = vim.api.nvim_buf_get_name(buf)
    vim.api.nvim_buf_set_lines(buf, 0, -1, false, { 'one', 'two' })
    local win = vim.api.nvim_open_win(buf, true, {
      relative = 'editor',
      width = 80,
      height = 10,
      row = 0,
      col = 0,
      style = 'minimal'
    })
    vim.api.nvim_win_set_cursor(win, { 1, 0 })
    local ok = M.refresh_then_end(buf, {
      renderer = function()
        vim.fn.writefile({ 'one', 'two', 'three', 'four' }, ansi)
        return true
      end,
    })
    eq(ok, true, 'refresh_then_end returns true')
    eq(vim.api.nvim_win_get_cursor(win)[1], 4, 'refresh_then_end lands on refreshed last line')
    vim.api.nvim_win_close(win, true)
  end

  -- 7. Refresh with a pending marker must not replace the annotate-attached
  -- buffer, because markers are buffer text until VimLeavePre emits them.
  do
    local annotate = M.annotate
    local MARKER = '\240\159\164\150'  -- 🤖
    local dir = tmpdir()
    local ansi = dir .. '/scrollback-test-codex.ansi'
    vim.fn.writefile({ 'fresh one', 'fresh two' }, ansi)

    local buf = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(buf, ansi)
    vim.api.nvim_buf_set_lines(buf, 0, -1, false, { 'old line' })
    vim.bo[buf].modifiable = false
    vim.bo[buf].readonly = true
    annotate.attach({ bufnr = buf, pending_path = dir .. '/pending.md', footer = true, quit_noun = 'scrollback' })

    vim.bo[buf].modifiable = true
    vim.bo[buf].readonly = false
    vim.api.nvim_buf_set_lines(buf, 0, 1, false, { 'old line ' .. MARKER .. '[keep me]' })
    vim.bo[buf].modifiable = false
    vim.bo[buf].readonly = true
    eq(annotate.has_new_markers(buf), true, 'precondition: pending marker exists')

    local called = false
    local ok = M.refresh_buffer(buf, {
      renderer = function()
        called = true
        vim.fn.writefile({ 'fresh one', 'fresh two' }, ansi)
        return true
      end,
    })
    eq(ok, true, 'marker-protected refresh reports success')
    eq(called, true, 'marker-protected refresh still renders backing ansi')
    eq(table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n'),
       'old line ' .. MARKER .. '[keep me]\nFor overall comment, Alt+q on this line.',
       'marker-protected refresh keeps annotated buffer intact')
    eq(annotate.has_new_markers(buf), true, 'pending marker still pending after skipped reload')
    vim.b[buf].pair_annotate = false
  end

  -- 8. Clean annotate-attached refresh reloads content, rebaselines marker
  -- state, and recreates the scrollback footer affordance at the new end.
  do
    local annotate = M.annotate
    local dir = tmpdir()
    local ansi = dir .. '/scrollback-test-codex.ansi'
    vim.fn.writefile({ 'old line' }, ansi)

    local buf = vim.api.nvim_create_buf(false, true)
    vim.api.nvim_buf_set_name(buf, ansi)
    vim.api.nvim_buf_set_lines(buf, 0, -1, false, { 'old line' })
    vim.bo[buf].modifiable = false
    vim.bo[buf].readonly = true
    annotate.attach({ bufnr = buf, pending_path = dir .. '/pending.md', footer = true, quit_noun = 'scrollback' })

    local ok = M.refresh_buffer(buf, {
      renderer = function()
        vim.fn.writefile({ 'fresh one', 'fresh two' }, ansi)
        return true
      end,
    })
    eq(ok, true, 'clean annotate-attached refresh returns true')
    eq(table.concat(vim.api.nvim_buf_get_lines(buf, 0, -1, false), '\n'),
       'fresh one\nfresh two\nFor overall comment, Alt+q on this line.',
       'clean refresh reloads content and restores footer')
    eq(annotate.has_new_markers(buf), false, 'clean refresh rebaselines annotate state')
    eq(vim.bo[buf].modifiable, false, 'clean annotate refresh leaves buffer unmodifiable')
    eq(vim.bo[buf].readonly, true, 'clean annotate refresh leaves buffer readonly')
    vim.b[buf].pair_annotate = false
  end

else
  io.stderr:write('WARN: vim.api unavailable — prompt pattern tests skipped\n')
end

-- Wiring smoke (#57): annotate.attach with scrollback's config (footer=true, no
-- source_label) must emit the LEGACY un-prefixed `> quote` format — proof the
-- extraction is behavior-preserving. Drives the data path (attach → marker-as-
-- text → emit); the interactive floating prompt is the documented headless limit.
if vim and vim.api then
  local annotate = dofile(here .. 'annotate.lua')
  local MARKER = '\240\159\164\150'  -- 🤖
  local buf = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_buf_set_lines(buf, 0, -1, false, { 'line one', 'line two' })
  vim.api.nvim_set_current_buf(buf)
  local pend = (os.getenv('TMPDIR') or '/tmp') .. '/pair-sb-annotate-test.md'
  os.remove(pend)
  annotate.attach({ bufnr = buf, pending_path = pend, footer = true, quit_noun = 'scrollback' })
  eq(vim.api.nvim_buf_line_count(buf), 3, 'scrollback attach appends footer affordance line')
  -- Simulate Alt+q dropping a bare marker on line 1 (as buffer text). Toggle
  -- BOTH modifiable + readonly, exactly as annotate's rewrite_line does.
  vim.bo[buf].modifiable = true
  vim.bo[buf].readonly = false
  vim.api.nvim_buf_set_lines(buf, 0, 1, false, { 'line one ' .. MARKER .. '[why?]' })
  vim.bo[buf].modifiable = false
  vim.bo[buf].readonly = true
  eq(annotate.has_new_markers(buf), true, 'scrollback has_new_markers true after add')
  annotate.emit(buf)
  local got = table.concat(vim.fn.readfile(pend), '\n')
  eq(got:match('> line one\nwhy%?') ~= nil, true, 'scrollback emit = legacy un-prefixed format')
  eq(got:match('%[change log%]') == nil, true, 'scrollback emit has NO source label')
  vim.b[buf].pair_annotate = false  -- stop the exit-time VimLeavePre re-emit
  os.remove(pend)
end

if fails > 0 then
  io.stderr:write(fails .. ' failure(s)\n')
  os.exit(1)
end
print('nvim/scrollback.lua: prompt pattern tests passed')
