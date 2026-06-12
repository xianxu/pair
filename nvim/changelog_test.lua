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

if fails > 0 then
  io.stderr:write(string.format('changelog_test: %d failure(s)\n', fails))
  os.exit(1)
end
print('ok changelog_test')
