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

-- Completion popup behavior. menuone: show popup even with one match.
-- noinsert,noselect: never auto-insert or auto-highlight a match — Enter
-- in the draft is overwhelmingly "newline", so accidental confirmation
-- would be disruptive. The user explicitly cycles with Tab/Shift-Tab.
vim.opt.completeopt = { 'menu', 'menuone', 'noinsert', 'noselect' }

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
  -- XDG data dir, honoring $PAIR_DATA_DIR if exported by bin/pair, else
  -- computing the same way (XDG_DATA_HOME with spec fallback).
  local data_dir = os.getenv('PAIR_DATA_DIR')
                or (os.getenv('XDG_DATA_HOME') or vim.fn.expand('~/.local/share'))
                   .. '/pair'
  local log_path = data_dir .. '/log-' .. pair_tag() .. '.md'
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
-- PairPasteQuote: triggered from bin/clipboard-to-pane.sh after a copy_command
-- selection. The shell hands off the *raw* clipboard body via
-- $PAIR_DATA_DIR/quote-<tag>; we decide the formatting here based on where
-- the cursor is.
--
--   * col == 0  → quote-mode. The user is at the start of a line, treating
--                 the selection as a fresh block. Reflow with par, prefix
--                 every line with `> `, scroll the first inserted line to
--                 the top of the window (zt), flash the block, land on a
--                 single empty line below in insert mode.
--
--   * col >  0  → inline-mode. The user is mid-text and stitching the
--                 selection in at the cursor. Reflow with par so hard-wrapped
--                 lines collapse into one continuous run (paragraph breaks
--                 preserved); no prefix, no scroll. Flash the inserted span,
--                 leave the cursor in insert mode immediately after it.
--
-- A single namespace `pair_flash` carries the IncSearch highlight in both
-- modes; cleared after 500ms via vim.defer_fn.
-- ---------------------------------------------------------------------------

local pair_flash_ns = vim.api.nvim_create_namespace('pair_flash')

local function quote_path()
  local data_dir = os.getenv('PAIR_DATA_DIR')
                or (os.getenv('XDG_DATA_HOME') or vim.fn.expand('~/.local/share'))
                   .. '/pair'
  return data_dir .. '/quote-' .. pair_tag()
end

local function clear_flash_after(buf, ms)
  vim.defer_fn(function()
    if vim.api.nvim_buf_is_valid(buf) then
      vim.api.nvim_buf_clear_namespace(buf, pair_flash_ns, 0, -1)
    end
  end, ms)
end

local function reflow_par(body)
  -- par 1000 (large width) acts as a paragraph rejoin/reflow; safe to skip
  -- if par is missing or errors out.
  if vim.fn.executable('par') == 0 then return body end
  local out = vim.fn.system({ 'par', '1000' }, body)
  if vim.v.shell_error ~= 0 then return body end
  return out
end

local function paste_as_quote(body, row)
  body = body:gsub('\n+$', '')
  local reflowed = reflow_par(body):gsub('\n+$', '')
  local quoted_lines = {}
  for line in (reflowed .. '\n'):gmatch('([^\n]*)\n') do
    quoted_lines[#quoted_lines + 1] = '> ' .. line
  end
  if #quoted_lines == 0 then return end

  local buf = vim.api.nvim_get_current_buf()
  local cur_line = vim.api.nvim_buf_get_lines(buf, row - 1, row, false)[1] or ''
  -- If the cursor's line is empty we replace it (so we don't end up with
  -- a leading blank above the quote); otherwise we insert above and let
  -- the existing line slide down.
  local insert_start, insert_end
  if cur_line == '' then
    insert_start, insert_end = row - 1, row
  else
    insert_start, insert_end = row - 1, row - 1
  end

  local payload = {}
  for _, l in ipairs(quoted_lines) do payload[#payload + 1] = l end
  payload[#payload + 1] = ''  -- the empty line the cursor will land on

  vim.api.nvim_buf_set_lines(buf, insert_start, insert_end, false, payload)

  local block_start = insert_start                  -- 0-indexed, inclusive
  local block_end   = insert_start + #quoted_lines  -- 0-indexed, exclusive
  local cursor_row  = block_end + 1                 -- 1-indexed empty line

  vim.api.nvim_win_set_cursor(0, { block_start + 1, 0 })
  vim.cmd('normal! zt')
  vim.api.nvim_win_set_cursor(0, { cursor_row, 0 })

  vim.api.nvim_buf_clear_namespace(buf, pair_flash_ns, 0, -1)
  for i = block_start, block_end - 1 do
    vim.api.nvim_buf_add_highlight(buf, pair_flash_ns, 'IncSearch', i, 0, -1)
  end
  clear_flash_after(buf, 500)

  vim.cmd('startinsert')
end

local function paste_inline(body, row, col)
  body = body:gsub('\n+$', '')
  if body == '' then return end
  body = reflow_par(body):gsub('\n+$', '')
  local lines = vim.split(body, '\n', { plain = true })

  local buf = vim.api.nvim_get_current_buf()
  vim.api.nvim_buf_set_text(buf, row - 1, col, row - 1, col, lines)

  local end_row, end_col
  if #lines == 1 then
    end_row = row - 1
    end_col = col + #lines[1]
  else
    end_row = row - 1 + #lines - 1
    end_col = #lines[#lines]
  end

  vim.api.nvim_buf_clear_namespace(buf, pair_flash_ns, 0, -1)
  vim.api.nvim_buf_set_extmark(buf, pair_flash_ns, row - 1, col, {
    end_row = end_row,
    end_col = end_col,
    hl_group = 'IncSearch',
  })
  clear_flash_after(buf, 500)

  -- Place cursor at the end of the inserted text, then enter insert mode.
  -- nvim normal-mode cursors clamp to (line length - 1), but startinsert
  -- promotes us to insert mode where end-of-line positioning works
  -- correctly — type-next-character lands at end_col as intended.
  vim.api.nvim_win_set_cursor(0, { end_row + 1, end_col })
  vim.cmd('startinsert')
end

function _G.PairPasteQuote()
  local f = io.open(quote_path(), 'r')
  if not f then return end
  local body = f:read('*a')
  f:close()
  if not body or body == '' then return end

  local row, col = unpack(vim.api.nvim_win_get_cursor(0))
  -- Defensive: nvim returns 1-indexed row but in some uninitialized states
  -- (e.g. just-opened headless instances) it can return 0. Clamp.
  if row < 1 then row = 1 end
  if col == 0 then
    paste_as_quote(body, row)
  else
    paste_inline(body, row, col)
  end
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
-- as-you-type path completion (plugin-free, fzf-style on the basename)
--
-- TextChangedI/P fires per keystroke. We pull the path-shaped token at the
-- cursor (anything ending in `/foo` or starting with `~`/`./`), split it on
-- the last `/` into <dir>/<filter>, list <dir> via getcompletion, then fuzzy-
-- filter by <filter> via matchfuzzy. Results go straight to vim.fn.complete()
-- — bypassing <C-x><C-f> avoids feedkeys reentrancy.
--
-- Reload via `:luafile $PAIR_HOME/nvim/init.lua` (works because all autocmds
-- live in the `pair` augroup with clear=true).
-- ---------------------------------------------------------------------------

-- Lua pattern for a path-ish token: word chars, slash, dot, dash, underscore,
-- tilde. Matches the longest such run anchored at end-of-prefix.
local PATH_TOKEN_RE = '([%w%./%-_~]+)$'

local function path_complete()
  local line = vim.api.nvim_get_current_line()
  local col = vim.fn.col('.') - 1  -- 0-indexed cursor byte position
  if col == 0 then return end
  local before = line:sub(1, col)
  local token_start, _, token = before:find(PATH_TOKEN_RE)
  if not token then return end
  -- Only trigger on path-shaped tokens. Plain words stay quiet.
  if not (token:find('/') or token:match('^~')) then return end

  local dir, filter = token:match('^(.*/)([^/]*)$')
  if not dir then dir, filter = '', token end

  local ok, matches = pcall(vim.fn.getcompletion, dir, 'file')
  if not ok or type(matches) ~= 'table' or #matches == 0 then return end
  if filter ~= '' then
    local ok2, fuzzy = pcall(vim.fn.matchfuzzy, matches, filter)
    if ok2 then matches = fuzzy end
  end
  if #matches == 0 then return end

  -- complete() col is 1-indexed start of the span being replaced.
  vim.fn.complete(token_start, matches)
end

local function pum_visible()
  return vim.fn.pumvisible() == 1
end

local function pum_has_selection()
  -- complete_info().selected is -1 when nothing highlighted.
  return vim.fn.complete_info({ 'selected' }).selected ~= -1
end

-- ---------------------------------------------------------------------------
-- keymaps
-- ---------------------------------------------------------------------------

vim.keymap.set({ 'n', 'i' }, '<M-CR>', send_and_clear,
  { silent = true, desc = 'pair: send buffer + clear' })

vim.keymap.set({ 'n', 'i' }, '<M-i>', attach_image,
  { silent = true, desc = 'pair: attach clipboard image (Ctrl+V to agent + ref)' })

vim.keymap.set('i', '<Tab>', function()
  return pum_visible() and '<C-n>' or '<Tab>'
end, { expr = true, desc = 'pair: cycle completion or insert tab' })

vim.keymap.set('i', '<S-Tab>', function()
  return pum_visible() and '<C-p>' or '<S-Tab>'
end, { expr = true, desc = 'pair: reverse-cycle completion or shift-tab' })

vim.keymap.set('i', '<CR>', function()
  return (pum_visible() and pum_has_selection()) and '<C-y>' or '<CR>'
end, { expr = true, desc = 'pair: accept completion if selected else newline' })

-- ---------------------------------------------------------------------------
-- autocmds — all under the `pair` augroup so :luafile reloads cleanly.
-- ---------------------------------------------------------------------------

local pair_aug = vim.api.nvim_create_augroup('pair', { clear = true })

-- autosave on transitions so disk and buffer agree
vim.api.nvim_create_autocmd({ 'BufLeave', 'FocusLost', 'InsertLeave' }, {
  group = pair_aug,
  pattern = '*',
  callback = function() pcall(vim.cmd, 'silent! write') end,
})

-- start at end of buffer in insert mode — drafting is the default activity,
-- so don't make the user press `i` after every fresh launch.
vim.api.nvim_create_autocmd('VimEnter', {
  group = pair_aug,
  callback = function()
    vim.cmd('normal! G')
    vim.cmd('startinsert!')
  end,
})

-- Fire on both events: TextChangedI when popup is hidden, TextChangedP when
-- popup is visible — refreshing the menu as the user types more characters.
vim.api.nvim_create_autocmd({ 'TextChangedI', 'TextChangedP' }, {
  group = pair_aug,
  callback = path_complete,
})
