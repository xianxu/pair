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

-- #57 M2: changelog annotate wiring + reload-guard smoke. Drives the data path
-- (attach → marker-as-text → emit) headlessly; the floating Alt+q prompt UI is
-- the documented headless limit.
do
  local annotate = dofile(here .. 'annotate.lua')
  local MARKER = '\240\159\164\150'  -- 🤖
  local buf = vim.api.nvim_create_buf(false, true)
  vim.api.nvim_buf_set_lines(buf, 0, -1, false, { '## 2026-06-12', '', '- M1 done for #53' })
  vim.api.nvim_set_current_buf(buf)
  M.setup(buf)
  local pend = (os.getenv('TMPDIR') or '/tmp') .. '/pair-cl-annotate-test.md'
  os.remove(pend)
  annotate.attach({
    bufnr = buf, pending_path = pend,
    footer = false, source_label = 'change log', quit_noun = 'change log',
  })
  -- footer=false: no overall-comment affordance line appended.
  check(vim.api.nvim_buf_line_count(buf) == 3, 'footer=false adds no affordance line')
  -- Simulate Alt+q dropping a bare marker on line 3 (as buffer text), toggling
  -- the read-only lock exactly as annotate's rewrite_line does.
  vim.bo[buf].modifiable = true
  vim.bo[buf].readonly = false
  vim.api.nvim_buf_set_lines(buf, 2, 3, false, { '- M1 done for #53 ' .. MARKER .. '[why M1 first?]' })
  vim.bo[buf].modifiable = false
  vim.bo[buf].readonly = true
  check(annotate.has_new_markers(buf) == true, 'has_new_markers true after add')
  -- emit ships a source-tagged block to the sidecar the draft picks up.
  annotate.emit(buf)
  local got = table.concat(vim.fn.readfile(pend), '\n')
  check(got:match('> %[change log%] .-why M1 first%?') ~= nil,
    'sidecar block tagged with change-log source')
  -- Reload guard (plan rev #3): with a marker present, the guard predicate skips
  -- the distiller reload, so the marker text survives. Mirror safe_reload's gate.
  local before = vim.api.nvim_buf_get_lines(buf, 0, -1, false)
  if not annotate.has_new_markers(buf) then M.reload(buf, pend) end
  local after = vim.api.nvim_buf_get_lines(buf, 0, -1, false)
  check(vim.deep_equal(before, after),
    'reload guard skips reload while a marker is present (marker survives)')
  vim.b[buf].pair_annotate = false  -- stop the exit-time VimLeavePre re-emit
  os.remove(pend)
end

-- #57 follow-up bug: the viewer autocmd fires on BufWinEnter for ANY buffer in a
-- window — incl. annotate's floating Alt+q prompt (an unnamed scratch buffer).
-- on_buf_enter must SKIP unnamed buffers so the prompt stays typeable; only the
-- named change-log buffer gets locked read-only. (Regression: without the guard,
-- M.setup locked the prompt modifiable=false and the user couldn't type.)
do
  -- Unnamed scratch buffer (stands in for the annotate prompt) — must be skipped.
  local scratch = vim.api.nvim_create_buf(false, true)
  check(M.on_buf_enter(scratch) == false, 'on_buf_enter skips an unnamed (prompt) buffer')
  check(vim.bo[scratch].modifiable == true, 'skipped prompt buffer stays modifiable (user can type)')
  check(vim.b[scratch].pair_annotate ~= true, 'skipped prompt buffer is not annotate-attached')

  -- Named buffer (the real change-log file) — must be set up read-only.
  local named = vim.api.nvim_create_buf(true, false)
  vim.api.nvim_buf_set_name(named, (os.getenv('TMPDIR') or '/tmp') .. '/pair-cl-named-test.md')
  vim.api.nvim_buf_set_lines(named, 0, -1, false, { '## 2026-06-12', '', '- entry' })
  vim.api.nvim_set_current_buf(named)
  check(M.on_buf_enter(named) == true, 'on_buf_enter sets up the named change-log buffer')
  check(vim.bo[named].modifiable == false, 'named change-log buffer locked read-only')
  vim.b[named].pair_annotate = false  -- stop the exit-time VimLeavePre re-emit
end

if fails > 0 then
  io.stderr:write(string.format('changelog_test: %d failure(s)\n', fails))
  os.exit(1)
end
print('ok changelog_test')
