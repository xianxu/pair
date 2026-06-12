-- Headless tests for nvim/changelog.lua — run via `nvim -l nvim/changelog_test.lua`
-- (or `make test-lua`). Drives M.setup directly; exits non-zero on failure.
_G.PAIR_CHANGELOG_TEST = true -- skip the interactive UI wiring at load
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local M = dofile(here .. 'changelog.lua')

local fails = 0
local function check(cond, msg)
  if not cond then
    io.stderr:write('FAIL ' .. msg .. '\n')
    fails = fails + 1
  end
end

-- Fixture change log with the glance tokens we colorize.
local buf = vim.api.nvim_create_buf(false, true)
vim.api.nvim_buf_set_lines(buf, 0, -1, false, {
  '## 2026-06-12',
  '',
  '- M1 done for #53 on `cmd/pair-changelog`, branch feature/53-changelog',
})
vim.api.nvim_set_current_buf(buf)
M.setup(buf)

-- read-only
check(vim.bo[buf].modifiable == false, 'buffer should be non-modifiable')
check(vim.bo[buf].readonly == true, 'buffer should be readonly')

-- colorization: resolve the syntax group under known tokens on line 3.
local line3 = vim.api.nvim_buf_get_lines(buf, 2, 3, false)[1]
local function group_at(col)
  return vim.fn.synIDattr(vim.fn.synID(3, col, 1), 'name')
end
local function token_group(tok)
  local c = string.find(line3, tok, 1, true)
  check(c ~= nil, 'token present: ' .. tok)
  return c and group_at(c) or '<absent>'
end

check(token_group('#53') == 'ChangelogTicket', 'ticket #53 → ChangelogTicket')
check(token_group('M1') == 'ChangelogMilestone', 'milestone M1 → ChangelogMilestone')
check(token_group('`') == 'ChangelogCode', 'inline code → ChangelogCode')
check(token_group('feature/53-changelog') == 'ChangelogBranch', 'branch → ChangelogBranch')

-- M.reload: simulate the async background distill finishing — write a new log
-- to disk, reload it into the read-only buffer, assert content + cursor + lock.
do
  local path = (os.getenv('TMPDIR') or '/tmp') .. '/pair-changelog-reload-test.md'
  local new_lines = { '## 2026-06-12', '', '- one', '', '- two newest #99' }
  vim.fn.writefile(new_lines, path)
  local rbuf = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_set_current_buf(rbuf)
  M.setup(rbuf)
  M.reload(rbuf, path)
  local got = vim.api.nvim_buf_get_lines(rbuf, 0, -1, false)
  check(vim.deep_equal(got, new_lines), 'reload replaced buffer with file contents')
  check(vim.bo[rbuf].modifiable == false, 'reload left buffer non-modifiable')
  check(vim.bo[rbuf].readonly == true, 'reload left buffer readonly')
  check(vim.api.nvim_win_get_cursor(0)[1] == #new_lines, 'reload put cursor at newest entry')
  os.remove(path)
end

if fails > 0 then
  io.stderr:write(string.format('changelog_test: %d failure(s)\n', fails))
  os.exit(1)
end
print('ok changelog_test')
