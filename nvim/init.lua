-- pair/nvim/init.lua
-- Minimal nvim config for the pair input pane. Loaded via `nvim -u`,
-- so this is fully isolated from the user's normal nvim setup.

vim.g.mapleader = ' '

-- Drafting-friendly editor settings
vim.opt.number = false
vim.opt.relativenumber = false
vim.opt.signcolumn = 'no'
vim.opt.laststatus = 0   -- no status line
vim.opt.cmdheight = 0    -- no permanent command line; appears on demand only
vim.opt.showmode = false -- nvim's default `-- INSERT --` line; redundant with cmdheight=0
vim.opt.ruler = false
vim.opt.wrap = true
vim.opt.linebreak = true
vim.opt.breakindent = true
vim.opt.spell = true
vim.opt.spelllang = 'en_us'
vim.opt.swapfile = false
vim.opt.backup = false
vim.opt.writebackup = false
vim.opt.clipboard = 'unnamedplus'
vim.opt.expandtab = true
vim.opt.shiftwidth = 2
vim.opt.tabstop = 2

-- Persistent undo so cleared content is recoverable
local undodir = vim.fn.expand('~/.local/share/pair/undo')
vim.fn.mkdir(undodir, 'p')
vim.opt.undodir = undodir
vim.opt.undofile = true

-- ---------------------------------------------------------------------------
-- helpers
-- ---------------------------------------------------------------------------

local function pair_tag()
  return os.getenv('PAIR_TAG') or os.getenv('PAIR_AGENT') or 'claude'
end

local function append_log(body)
  local log_path = vim.fn.expand('~/scratch/pair-log-' .. pair_tag() .. '.md')
  vim.fn.mkdir(vim.fn.fnamemodify(log_path, ':h'), 'p')
  local f = io.open(log_path, 'a')
  if not f then return end
  f:write(os.date('## %Y-%m-%d %H:%M:%S') .. '\n\n')
  f:write(body)
  f:write('\n\n---\n\n')
  f:close()
end

local function send_to_agent(body)
  -- focus up to agent pane, type body, press Enter, focus back down
  vim.fn.system('zellij action move-focus up')
  vim.fn.system({ 'zellij', 'action', 'write-chars', body })
  vim.fn.system('zellij action write 13')
  vim.fn.system('zellij action move-focus down')
end

-- ---------------------------------------------------------------------------
-- per-session image counter — monotonic across the whole claude session.
-- Claude Code numbers image attachments cumulatively per-session, not
-- per-message, so this counter mirrors that and never resets on send.
-- (If it ever drifts from claude's actual count — e.g. nvim restart mid-
-- session — fix the [Image #N] reference by hand.)
-- ---------------------------------------------------------------------------

local pair_image_count = 0

-- ---------------------------------------------------------------------------
-- attach_image: Alt+i — context-sensitive image binding.
--
--   * If the cursor is inside (or just after) an existing [Image #N] token,
--     sync our internal counter to N. This is the user's manual-correction
--     path: when our count drifts from claude's actual count, the user edits
--     the number in-place and presses Alt+i to realign. No image is sent.
--
--   * Otherwise, increment the counter, send Ctrl+V to the agent pane (claude
--     reads the OS clipboard and attaches the image as a chip), insert
--     [Image #N] at cursor, advance past it. Caller must put image bytes on
--     the clipboard first (osascript / wl-copy / xclip recipes in README).
-- ---------------------------------------------------------------------------

local function image_token_at_cursor()
  local row, col = unpack(vim.api.nvim_win_get_cursor(0))
  local line = vim.api.nvim_buf_get_lines(0, row - 1, row, false)[1]
  if not line then return nil end
  local search_from = 1
  while search_from <= #line do
    local mstart, mend, num = line:find('%[Image #(%d+)%]', search_from)
    if not mstart then return nil end
    -- col is 0-indexed byte position. Treat cursor as "on" the token if it
    -- lies anywhere from the opening `[` through one byte past the `]`
    -- (the latter is where insert mode parks after typing the close bracket).
    if col + 1 >= mstart and col + 1 <= mend + 1 then
      return tonumber(num)
    end
    search_from = mend + 1
  end
  return nil
end

local function attach_image()
  local existing = image_token_at_cursor()
  if existing then
    pair_image_count = existing
    vim.notify('pair: image counter synced to ' .. existing, vim.log.levels.INFO)
    return
  end

  pair_image_count = pair_image_count + 1
  local n = pair_image_count

  -- Ctrl+V = 0x16 = 22. Sent to agent pane.
  vim.fn.system('zellij action move-focus up')
  vim.fn.system('zellij action write 22')
  vim.fn.system('zellij action move-focus down')

  -- Insert [Image #N] at cursor and advance past it.
  local row, col = unpack(vim.api.nvim_win_get_cursor(0))
  local token = '[Image #' .. n .. ']'
  vim.api.nvim_buf_set_text(0, row - 1, col, row - 1, col, { token })
  vim.api.nvim_win_set_cursor(0, { row, col + #token })
end

-- ---------------------------------------------------------------------------
-- send_and_clear: Alt+Return — send entire buffer, log, clear, reset
-- ---------------------------------------------------------------------------

local function send_and_clear()
  local lines = vim.api.nvim_buf_get_lines(0, 0, -1, false)
  local body = table.concat(lines, '\n')
  if body:match('^%s*$') then return end

  append_log(body)
  send_to_agent(body)

  vim.api.nvim_buf_set_lines(0, 0, -1, false, { '' })
  vim.cmd('silent! write')
  vim.cmd('startinsert')
end

-- ---------------------------------------------------------------------------
-- send_section: <leader>cs — send only the section between --- markers
-- ---------------------------------------------------------------------------

local function send_section()
  local cur = vim.api.nvim_win_get_cursor(0)[1]
  local last = vim.api.nvim_buf_line_count(0)

  local function is_marker(i)
    local line = vim.api.nvim_buf_get_lines(0, i - 1, i, false)[1]
    return line == '---'
  end

  local first = 1
  for i = cur - 1, 1, -1 do
    if is_marker(i) then first = i + 1; break end
  end
  local stop = last
  for i = cur, last do
    if is_marker(i) then stop = i - 1; break end
  end

  local lines = vim.api.nvim_buf_get_lines(0, first - 1, stop, false)
  local body = table.concat(lines, '\n'):gsub('^%s+', ''):gsub('%s+$', '')
  if body == '' then return end

  append_log(body)
  send_to_agent(body)
end

-- ---------------------------------------------------------------------------
-- paste_and_reflow: <leader>cp — paste clipboard at cursor with par reflow
-- ---------------------------------------------------------------------------

local function paste_and_reflow()
  local clip = vim.fn.getreg('+')
  if clip == '' then return end
  if vim.fn.executable('par') == 1 then
    clip = vim.fn.system({ 'par', '1000' }, clip)
  end
  -- strip trailing newline from system output
  clip = clip:gsub('\n$', '')
  local lines = vim.split(clip, '\n', { plain = true, trimempty = false })
  local row = vim.api.nvim_win_get_cursor(0)[1]
  vim.api.nvim_buf_set_lines(0, row, row, false, lines)
end

-- ---------------------------------------------------------------------------
-- keymaps
-- ---------------------------------------------------------------------------

vim.keymap.set({ 'n', 'i' }, '<M-CR>', send_and_clear,
  { silent = true, desc = 'pair: send buffer + clear' })

vim.keymap.set({ 'n', 'i' }, '<M-i>', attach_image,
  { silent = true, desc = 'pair: attach clipboard image (Ctrl+V to agent + ref)' })

vim.keymap.set('n', '<leader>cs', send_section,
  { silent = true, desc = 'pair: send current section' })

vim.keymap.set('n', '<leader>cp', paste_and_reflow,
  { silent = true, desc = 'pair: paste-and-reflow' })

-- ---------------------------------------------------------------------------
-- autosave on transitions so disk and buffer agree
-- ---------------------------------------------------------------------------

vim.api.nvim_create_autocmd({ 'BufLeave', 'FocusLost', 'InsertLeave' }, {
  pattern = '*',
  callback = function() pcall(vim.cmd, 'silent! write') end,
})
