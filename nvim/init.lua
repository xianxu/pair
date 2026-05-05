-- pair/nvim/init.lua
-- Minimal nvim config for the pair input pane. Loaded via `nvim -u`,
-- so this is fully isolated from the user's normal nvim setup.

vim.g.mapleader = ' '

-- Enable filetype detection + default syntax. Loaded via `nvim -u`, which
-- doesn't bypass nvim's bundled runtime but doesn't auto-enable these
-- either. The draft file is `.md`, so this picks up markdown highlighting.
-- termguicolors is required for the default colorscheme's gui-defined
-- palette to render in the terminal — without it most syntax groups fall
-- back to a near-monochrome cterm palette.
vim.opt.termguicolors = true
vim.cmd('syntax enable')
vim.cmd('filetype plugin indent on')
-- nvim's bundled `default` colorscheme is intentionally near-monochrome —
-- syntax groups get bold/italic but no fg colors. habamax is bundled and
-- gives readable colors for markdown headings, code spans, links, etc.
vim.cmd('colorscheme slate')

-- Drafting-friendly editor settings
vim.opt.number = true
vim.opt.relativenumber = false
vim.opt.numberwidth = 1
vim.opt.signcolumn = 'no'
-- laststatus=2 + a custom statusline are set later (after nav helpers); they
-- back the position indicator `History H < pos > Q Queued` for the prompt
-- history/queue feature.
vim.opt.cmdheight = 0    -- no permanent command line; appears on demand only

-- Suppress the "written" / file-info messages on :w. Without this, every
-- autosave or send-and-clear write briefly pops the cmdline up under
-- `cmdheight=0` and that interaction blanks the custom statusline until the
-- next redraw. With "W" + "F" the messages never fire, so the statusline
-- stays put.
vim.opt.shortmess:append('WF')
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

-- Disable nvim's right-click context menu. Default `mousemodel=popup_setpos`
-- pops up a "Copy/Paste/..." menu that's confusing inside the pair draft pane;
-- `extend` falls back to the vim-traditional behavior of extending the visual
-- selection on right-click.
vim.opt.mousemodel = 'extend'

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

local function pair_data_dir()
  return os.getenv('PAIR_DATA_DIR')
      or (os.getenv('XDG_DATA_HOME') or vim.fn.expand('~/.local/share'))
         .. '/pair'
end

local function log_path_for_tag()
  return pair_data_dir() .. '/log-' .. pair_tag() .. '.md'
end

local function draft_path_for_tag()
  return pair_data_dir() .. '/draft-' .. pair_tag() .. '.md'
end

local function read_file(path)
  local f = io.open(path, 'r')
  if not f then return '' end
  local content = f:read('*a') or ''
  f:close()
  return content
end

local function write_file(path, content)
  vim.fn.mkdir(vim.fn.fnamemodify(path, ':h'), 'p')
  local f = io.open(path, 'w')
  if not f then return false end
  f:write(content)
  f:close()
  return true
end

local function append_log(body)
  local path = log_path_for_tag()
  vim.fn.mkdir(vim.fn.fnamemodify(path, ':h'), 'p')
  local f = io.open(path, 'a')
  if not f then return end
  f:write(os.date('## %Y-%m-%d %H:%M:%S') .. '\n\n')
  f:write(body)
  f:write('\n\n---\n\n')
  f:close()
end

-- Parse log-<tag>.md into a list of entry bodies, oldest first.
-- Entry shape (per append_log): "## YYYY-MM-DD HH:MM:SS\n\n<body>\n\n---\n\n".
-- Splitting on the entry separator yields parts; the trailing chunk is "" since
-- the file ends with the separator. Each non-empty part starts with the
-- timestamp header which we strip to recover just the body.
local function parse_log(text)
  local entries = {}
  if text == '' then return entries end
  local parts = vim.split(text, '\n\n---\n\n', { plain = true })
  for _, part in ipairs(parts) do
    if part ~= '' then
      local body = part:gsub('^## %S+ %S+\n\n', '', 1)
      table.insert(entries, body)
    end
  end
  return entries
end

local function read_history()
  return parse_log(read_file(log_path_for_tag()))
end

-- Rewrite the body of the n-th-most-recent log entry (n=1 is newest) in
-- place, preserving its timestamp header. Used to persist comment-only
-- edits made while navigating history — comments are stripped before the
-- agent sees them, so changing them is a no-op against the agent's view of
-- history. No-op if n is out of range or the file is missing/malformed.
local function write_history_entry(n, body)
  local path = log_path_for_tag()
  local text = read_file(path)
  if text == '' then return end
  local parts = vim.split(text, '\n\n---\n\n', { plain = true })
  local entries = {}
  for _, p in ipairs(parts) do
    if p ~= '' then table.insert(entries, p) end
  end
  local idx = #entries - n + 1
  if idx < 1 or idx > #entries then return end
  local header = entries[idx]:match('^(## %S+ %S+\n\n)')
  if not header then return end
  entries[idx] = header .. body
  write_file(path, table.concat(entries, '\n\n---\n\n') .. '\n\n---\n\n')
end

local function buffer_text()
  return table.concat(vim.api.nvim_buf_get_lines(0, 0, -1, false), '\n')
end

local function set_buffer_text(s)
  -- Replace whole buffer contents, mark unmodified, park cursor at end so the
  -- user can append immediately. nvim's autosave autocmd is gated on nav.pos,
  -- so this won't clobber draft-<tag>.md when we're showing a history entry.
  --
  -- Strip a single trailing "\n" from `s` before splitting: files we read off
  -- disk have nvim's :w-added trailing newline, but the user-visible buffer
  -- representation (and what buffer_text() returns) doesn't include it. Without
  -- this, set_buffer_text(read_file(draft)) produces a buffer with a spurious
  -- empty trailing line that grows with every write/read cycle.
  s = s or ''
  if s:sub(-1) == '\n' then s = s:sub(1, -2) end
  local lines = vim.split(s, '\n', { plain = true })
  vim.api.nvim_buf_set_lines(0, 0, -1, false, lines)
  vim.bo.modified = false
  local last = math.max(1, #lines)
  pcall(vim.api.nvim_win_set_cursor, 0, { last, #(lines[last] or '') })
end

-- Forward-declared navigation state. Accessed by send_and_clear and the
-- autosave autocmd (both before the nav helpers below). Initialized to the
-- "draft slot, clean buffer" state.
local nav = { pos = '*', baseline = '' }

local function refresh_statusline()
  -- Defer to the next event-loop tick so the redraw fires *after* any side
  -- effects from the calling action have settled — e.g. send_and_clear's
  -- vim.fn.system shell-outs to zellij and the :w that follows. Without
  -- this, the trailing redraw work from those operations can blank the
  -- statusline immediately after our refresh, and it stays blank until
  -- the next user action triggers another redraw.
  vim.schedule(function() pcall(vim.cmd, 'redrawstatus') end)
end

-- Brief inverted flash on the "N queued" statusline segment. Confirms a
-- queue-count change actually happened — Alt+q lands an item, send-from-+N
-- consumes one, Alt+BS deletes one. Without it the only visible change is
-- "buffer snapped/cleared," which is the same shape as a discard.
--
-- Both PairQueueCount and PairQueueZero are swapped so the flash is visible
-- whether the queue ends non-empty (PairQueueCount) or hits zero on this
-- transition (PairQueueZero). Uses IncSearch + 500ms to match paste-flash.
local function flash_queue_count()
  vim.api.nvim_set_hl(0, 'PairQueueCount', { link = 'IncSearch' })
  vim.api.nvim_set_hl(0, 'PairQueueZero',  { link = 'IncSearch' })
  refresh_statusline()
  vim.defer_fn(function()
    vim.api.nvim_set_hl(0, 'PairQueueCount', { link = 'WarningMsg' })
    vim.api.nvim_set_hl(0, 'PairQueueZero',  { link = 'Comment' })
    refresh_statusline()
  end, 500)
end

-- ---------------------------------------------------------------------------
-- queue store (issue #000015 §H2)
--
-- Per-tag directory `$DATA_DIR/queue-<tag>/` with one file per queued prompt.
-- Filenames are 6-digit zero-padded keys; sort order = display order (+1 is
-- the lowest key, +M is the highest). Keys start at 500000 in the middle of
-- the address space so push_front and push_back can grow either way without
-- collision in practical use.
-- ---------------------------------------------------------------------------

local QUEUE_KEY_FMT = '%06d'
local QUEUE_KEY_START = 500000

local function queue_dir()
  return pair_data_dir() .. '/queue-' .. pair_tag()
end

local function queue_path(key)
  return queue_dir() .. '/' .. key .. '.md'
end

local function queue_keys_sorted()
  vim.fn.mkdir(queue_dir(), 'p')
  local files = vim.fn.readdir(queue_dir(), function(name)
    return (name:match('^%d+%.md$') and 1 or 0)
  end)
  table.sort(files)
  local keys = {}
  for _, f in ipairs(files) do
    table.insert(keys, (f:gsub('%.md$', '')))
  end
  return keys
end

local function queue_count()
  return #queue_keys_sorted()
end

local function queue_read(key)
  return read_file(queue_path(key))
end

local function queue_write(key, body)
  return write_file(queue_path(key), body)
end

local function queue_remove(key)
  return os.remove(queue_path(key))
end

local function queue_push_back(body)
  local keys = queue_keys_sorted()
  local next_n = (#keys == 0) and QUEUE_KEY_START or (tonumber(keys[#keys]) + 1)
  local key = string.format(QUEUE_KEY_FMT, next_n)
  queue_write(key, body)
  return key
end

local function queue_push_front(body)
  local keys = queue_keys_sorted()
  local next_n = (#keys == 0) and QUEUE_KEY_START or (tonumber(keys[1]) - 1)
  local key = string.format(QUEUE_KEY_FMT, next_n)
  queue_write(key, body)
  return key
end

-- Display index N (1-based, +1 = front) → filename key.
local function queue_key_for_n(n)
  local keys = queue_keys_sorted()
  return keys[n]
end

local function send_to_agent(body)
  -- focus up to agent pane, type body, press Enter, focus back down
  vim.fn.system('zellij action move-focus up')
  vim.fn.system({ 'zellij', 'action', 'write-chars', body })
  vim.fn.system('zellij action write 13')
  vim.fn.system('zellij action move-focus down')
end

-- Strip whole-line comments (^%s*===) before sending. Comments are stored
-- intact in draft/queue/log so they survive history navigation — only what
-- reaches the agent is cleaned. Trailing blank lines left behind by the
-- strip are also dropped so the agent doesn't see a dangling tail.
local function strip_comments(body)
  local out = {}
  for line in (body .. '\n'):gmatch('([^\n]*)\n') do
    if not line:match('^%s*===') then
      table.insert(out, line)
    end
  end
  while #out > 0 and out[#out]:match('^%s*$') do
    table.remove(out)
  end
  return table.concat(out, '\n')
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
  local body = buffer_text()
  if body:match('^%s*$') then return end
  -- Strip-then-check happens before any side effects: a comment-only buffer
  -- is a no-op send, so it must NOT consume a +N queue item or append to log.
  local stripped = strip_comments(body)
  if stripped:match('^%s*$') then return end

  -- send-from-+N consumes that queue file. The buffer (possibly edited) is
  -- what ships, and the queue slot vanishes.
  local consumed_queue = false
  if type(nav.pos) == 'table' and nav.pos.kind == 'queue' then
    local key = queue_key_for_n(nav.pos.n)
    if key then
      queue_remove(key)
      consumed_queue = true
    end
  end

  local was_at_star = (nav.pos == '*')

  -- Log the unstripped body (the user's authored text, comments and all),
  -- send only the stripped version to the agent.
  append_log(body)
  send_to_agent(stripped)

  -- Return to *. If we sent FROM *, that's the existing "clear the draft"
  -- semantic. If we sent from -N or +N, * is unaffected — the user's
  -- draft was already autosaved when they navigated away from *. Reload
  -- it so they see their unfinished work back in the buffer.
  nav.pos = '*'
  if was_at_star then
    vim.api.nvim_buf_set_lines(0, 0, -1, false, { '' })
    vim.cmd('silent! write')
    nav.baseline = ''
  else
    set_buffer_text(read_file(draft_path_for_tag()))
    nav.baseline = buffer_text()
  end
  refresh_statusline()
  if consumed_queue then flash_queue_count() end
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
-- prompt history & queue navigation (issue #000015)
--
-- Position model (M1: history-only):
--   nav.pos = '*'                          -- the persistent draft slot
--          | { kind='history', n=N }       -- Nth-most-recent log entry; n=1 is newest
--          | { kind='queue',   n=N }       -- (M2+) Nth queue item; n=1 is +1 (front)
--
-- nav.baseline = the buffer content at the moment we last loaded the slot.
-- Used to detect dirtiness for the (M3+) edit-flow rules. In M1 we discard
-- dirty content with a warning.
--
-- See workshop/issues/000015-prompt-history-queue.md for the full spec.
-- ---------------------------------------------------------------------------

local function pos_label(pos)
  if pos == '*' then return '*' end
  if pos.kind == 'history' then return '-' .. pos.n end
  if pos.kind == 'queue'   then return '+' .. pos.n end
  return '?'
end

local function load_baseline_for_current_pos()
  if nav.pos == '*' then
    return read_file(draft_path_for_tag())
  end
  if nav.pos.kind == 'history' then
    local entries = read_history()
    -- entries[#entries] is most recent ⇒ pos.n=1.
    local idx = #entries - nav.pos.n + 1
    return entries[idx] or ''
  end
  if nav.pos.kind == 'queue' then
    local key = queue_key_for_n(nav.pos.n)
    return key and queue_read(key) or ''
  end
  return ''
end

-- Dirty only matters for -N: history is immutable, so an edit there is a
-- pending fork that must explicitly become a send / a queue entry / a discard.
-- * and +N are mutable (their edits autosave to the underlying file), so they
-- have no dirty concept from the user's perspective.
--
-- Comment-only / blank-line-only edits don't count as dirty: they vanish under
-- strip_comments before the agent sees them, so there's no fork to resolve.
-- They are persisted back into the log entry by autosave_current_slot, so
-- annotations on history survive navigation and nvim restart.
local function is_dirty_history_slot()
  return type(nav.pos) == 'table'
     and nav.pos.kind == 'history'
     and strip_comments(buffer_text()) ~= strip_comments(nav.baseline)
end

-- Statusline format:
--   " Alt: <- history H < pos[*][ HINT] > Q queued -> "
-- The flanking arrows hint Alt+← / Alt+→. The trailing "*" on `pos` shows
-- when on -N with an unsent fork. The HINT inside the brackets is contextual:
--   * or -N : " [q=queue]" — Alt+q parks the current buffer as +1.
--   +N      : " [⌫=del]"   — Alt+BS deletes the current queue item.
-- Suppressed on +N for [q=queue] because Alt+q from +N is "move-to-front",
-- a different mental action that doesn't grow the queue.
--
-- Wrapped in a pcall guard so any edge-case error in is_dirty_history_slot
-- / read_history / queue_count can't blank the bar — fall back to a minimal
-- safe string.
function _G.PairStatusline()
  if vim.fn.mode():sub(1, 1) == 'n' then
    return '%#PairLocked# <LOCKED> input not accepted — press i to type %*'
  end
  local ok, result = pcall(function()
    local h = #read_history()
    local q = queue_count()
    local pos = pos_label(nav.pos)
    if is_dirty_history_slot() then pos = pos .. '*' end
    local hint
    if type(nav.pos) == 'table' and nav.pos.kind == 'queue' then
      hint = ' [⌫=del]'
    else
      hint = ' [q=queue]'
    end
    local pos_seg = string.format('%%#PairPosLabel#%s%%*', pos)
    local q_hl = (q > 0) and 'PairQueueCount' or 'PairQueueZero'
    local q_seg = string.format('%%#%s#%d queued%%*', q_hl, q)
    return string.format(' Alt: <- history %d < %s%s > %s -> ', h, pos_seg, hint, q_seg)
  end)
  return ok and result or ' pair '
end

-- Persist any pending edit on a mutable slot to its underlying file. For
-- -N, only comment-only edits are persisted in place (the agent never sees
-- comments, so changing them isn't a fork). A real fork — anything that
-- would change the stripped body — is left unsaved so the next go_to can
-- raise the leave-dirty-history prompt.
local function autosave_current_slot()
  if nav.pos == '*' then
    pcall(vim.cmd, 'silent! write')
  elseif type(nav.pos) == 'table' and nav.pos.kind == 'queue' then
    local key = queue_key_for_n(nav.pos.n)
    if key then queue_write(key, buffer_text()) end
  elseif type(nav.pos) == 'table' and nav.pos.kind == 'history' then
    local body = buffer_text()
    if body ~= nav.baseline
       and strip_comments(body) == strip_comments(nav.baseline) then
      write_history_entry(nav.pos.n, body)
      nav.baseline = body
    end
  end
end

-- Send the current buffer to the agent and return to *, preserving *'s
-- persistent draft. Used only by the dirty-`-N` prompt's Send branch
-- (send_and_clear has its own variant that handles the from-* case).
local function ship_buffer_and_reset(body)
  -- Mirror send_and_clear: log full body, send stripped. Skip both if the
  -- stripped result is empty so a comment-only fork doesn't pollute the log.
  local stripped = strip_comments(body)
  if not stripped:match('^%s*$') then
    append_log(body)
    send_to_agent(stripped)
  end
  nav.pos = '*'
  set_buffer_text(read_file(draft_path_for_tag()))
  nav.baseline = buffer_text()
  refresh_statusline()
end

-- Prompt the user with the four-option "what now?" dialog when leaving a
-- dirty -N slot. Returns true if the caller should proceed with the original
-- navigation (i.e. user picked Discard), false otherwise (Send/Queue performed
-- the action and moved us to *; or Stay cancelled the nav).
--
-- Single-key prompt format:  (S)end, (Q)ueue, (D)iscard, [S]tay:
--   () marks the access key, [] marks the default. Send/Stay both start with
--   "S"; we resolve by binding S → Send (the user-typeable choice) and
--   Enter/ESC/anything-else → Stay (the safe default, including covering
--   accidental key presses).
local function leave_dirty_history()
  local body = buffer_text()
  vim.api.nvim_echo({
    { '(S)end, (Q)ueue, (D)iscard, [S]tay: ', 'Question' },
  }, false, {})
  local ok, c = pcall(vim.fn.getchar)
  -- Clear the prompt line so it doesn't linger under cmdheight=0.
  pcall(vim.api.nvim_echo, { { '' } }, false, {})
  if not ok then return false end

  local key = (type(c) == 'number') and vim.fn.nr2char(c) or tostring(c or '')
  key = key:lower()

  if key == 's' then
    ship_buffer_and_reset(body)
    return false
  elseif key == 'q' then
    queue_push_front(body)
    set_buffer_text(read_file(draft_path_for_tag()))
    nav.pos = '*'
    nav.baseline = buffer_text()
    refresh_statusline()
    flash_queue_count()
    return false
  elseif key == 'd' then
    return true
  else
    return false                       -- Stay (Enter, ESC, anything else)
  end
end

-- Move to a new position: save mutable slots, prompt on dirty -N, then load
-- the destination baseline. nav_left / nav_right just compute the target pos.
local function go_to(new_pos)
  if is_dirty_history_slot() then
    if not leave_dirty_history() then return end
  else
    autosave_current_slot()
  end
  nav.pos = new_pos
  set_buffer_text(load_baseline_for_current_pos())
  -- Re-read the baseline from the buffer so its representation matches
  -- buffer_text() exactly (set_buffer_text strips a trailing newline). This
  -- keeps the dirty check (`buffer_text() ~= nav.baseline`) honest.
  nav.baseline = buffer_text()
  refresh_statusline()
end

local function nav_left()
  if nav.pos == '*' then
    local entries = read_history()
    if #entries == 0 then return end
    go_to({ kind = 'history', n = 1 })
  elseif nav.pos.kind == 'history' then
    local entries = read_history()
    if nav.pos.n >= #entries then return end   -- clamp at oldest
    go_to({ kind = 'history', n = nav.pos.n + 1 })
  elseif nav.pos.kind == 'queue' then
    -- +N → +(N-1) → *
    if nav.pos.n <= 1 then
      go_to('*')
    else
      go_to({ kind = 'queue', n = nav.pos.n - 1 })
    end
  end
end

local function nav_right()
  if nav.pos == '*' then
    -- * → +1 if queue has items.
    if queue_count() == 0 then return end
    go_to({ kind = 'queue', n = 1 })
  elseif nav.pos.kind == 'history' then
    -- -N → -(N-1), with -1 → *.
    if nav.pos.n <= 1 then
      go_to('*')
    else
      go_to({ kind = 'history', n = nav.pos.n - 1 })
    end
  elseif nav.pos.kind == 'queue' then
    -- +N → +(N+1), clamp at queue size.
    local total = queue_count()
    if nav.pos.n >= total then return end
    go_to({ kind = 'queue', n = nav.pos.n + 1 })
  end
end

-- Boundary-jump: Shift+Alt+←/→ skips to the next "edge" landmark in the
-- requested direction. Landmarks left-to-right: -h (oldest history),
-- -1 (newest history), *, +1 (front of queue), +q (back of queue).
-- A region with only one entry contributes only one landmark; an empty
-- region contributes none.
local function pos_rank(p)
  if p == '*' then return 0 end
  if p.kind == 'history' then return -p.n end   -- -1 ranks -1, -h ranks -h
  if p.kind == 'queue'   then return  p.n end   -- +1 ranks 1,  +q ranks q
  return 0
end

local function ordered_landmarks()
  local list = {}
  local h = #read_history()
  local q = queue_count()
  if h >= 1 then
    table.insert(list, { kind = 'history', n = h })             -- -h (leftmost)
    if h > 1 then table.insert(list, { kind = 'history', n = 1 }) end  -- -1
  end
  table.insert(list, '*')
  if q >= 1 then
    table.insert(list, { kind = 'queue', n = 1 })               -- +1
    if q > 1 then table.insert(list, { kind = 'queue', n = q }) end    -- +q (rightmost)
  end
  return list
end

local function nav_boundary(direction)
  local landmarks = ordered_landmarks()
  local cur = pos_rank(nav.pos)
  if direction > 0 then
    for _, lm in ipairs(landmarks) do
      if pos_rank(lm) > cur then go_to(lm); return end
    end
  else
    for i = #landmarks, 1, -1 do
      if pos_rank(landmarks[i]) < cur then go_to(landmarks[i]); return end
    end
  end
end

-- Alt+BS — delete the current +N queue item without sending it. "Stay near":
-- after delete, items at +(N+1)..+M shift down by one, so the same +N slot
-- now displays what used to be next. Lets the user tap-tap to clean out a
-- run. If queue empties, fall back to *.
--
-- Confirmation defaults to Yes — light safeguard against Cmd+BS (delete-to-
-- line-start) being mistapped as Alt+BS. Enter/space/y all proceed; only 'n'
-- or Esc cancels, so tap-tap cleanup stays cheap.
local function delete_current_queue_item()
  if type(nav.pos) ~= 'table' or nav.pos.kind ~= 'queue' then return end
  local key = queue_key_for_n(nav.pos.n)
  if not key then return end
  vim.api.nvim_echo({
    { 'Delete this queue item? [Y]es, (n)o: ', 'WarningMsg' },
  }, false, {})
  local ok, c = pcall(vim.fn.getchar)
  pcall(vim.api.nvim_echo, { { '' } }, false, {})
  local function cancel() refresh_statusline(); return end
  if not ok then return cancel() end
  if c == 27 then return cancel() end -- Esc
  local ch = (type(c) == 'number') and vim.fn.nr2char(c) or tostring(c or '')
  if ch:lower() == 'n' then return cancel() end
  queue_remove(key)
  local total = queue_count()
  if total == 0 then
    nav.pos = '*'
  elseif nav.pos.n > total then
    nav.pos = { kind = 'queue', n = total }
  end
  -- nav.pos.n unchanged when there's still something at this slot — the
  -- shifted-down item takes its place.
  set_buffer_text(load_baseline_for_current_pos())
  nav.baseline = buffer_text()
  refresh_statusline()
  flash_queue_count()
end

-- Shift+Alt+BS — wipe history, draft, and queue. "Start anew" for a session
-- whose state has accumulated cruft. Hard delete (no archive) per design;
-- confirmation defaults to No so a stray tap can't nuke a session.
local function forget_all()
  vim.api.nvim_echo({
    { 'Erase history, draft, and queue? (y)es, [N]o: ', 'WarningMsg' },
  }, false, {})
  local ok, c = pcall(vim.fn.getchar)
  pcall(vim.api.nvim_echo, { { '' } }, false, {})
  if not ok then return end
  local key = (type(c) == 'number') and vim.fn.nr2char(c) or tostring(c or '')
  if key:lower() ~= 'y' then return end

  os.remove(log_path_for_tag())
  os.remove(draft_path_for_tag())
  for _, k in ipairs(queue_keys_sorted()) do
    queue_remove(k)
  end

  nav.pos = '*'
  vim.api.nvim_buf_set_lines(0, 0, -1, false, { '' })
  vim.cmd('silent! write')
  nav.baseline = ''
  refresh_statusline()
  vim.cmd('startinsert')
end

-- Alt+q — push current buffer to the FRONT of the queue (+1), return to *.
-- Uniform rule across source slots:
--   * source : "park this draft for later". Clear * after the push.
--   -N source: fork the (possibly edited) history entry into +1. * untouched.
--   +N source: move-to-front. Remove the source +N file before pushing, so the
--              same item ends up at +1 (with edits applied). * untouched.
local function queue_current()
  local body = buffer_text()
  if body:match('^%s*$') then return end

  if type(nav.pos) == 'table' and nav.pos.kind == 'queue' then
    local key = queue_key_for_n(nav.pos.n)
    if key then queue_remove(key) end
  end

  queue_push_front(body)

  if nav.pos == '*' then
    -- Park-the-draft: * is now empty. Persist via :w; shortmess+=W keeps
    -- the cmdline silent so the statusline doesn't get pushed off.
    vim.api.nvim_buf_set_lines(0, 0, -1, false, { '' })
    vim.cmd('silent! write')
  else
    -- From -N or +N: * is untouched. Snap buffer back to *'s on-disk baseline.
    set_buffer_text(read_file(draft_path_for_tag()))
  end

  nav.pos = '*'
  nav.baseline = buffer_text()
  refresh_statusline()
  flash_queue_count()
  vim.cmd('startinsert')
end

vim.opt.laststatus = 2
vim.opt.statusline = '%!v:lua.PairStatusline()'

-- Soften the statusline appearance. nvim's default StatusLine highlight is
-- inverted/bold (looks like a stark contrasting bar), which feels out of
-- place against the editing buffer. Linking it to `Comment` picks up the
-- colorscheme's dimmed-text style — visible as "secondary info" without
-- being visually loud. ColorScheme autocmd reapplies on theme changes.
local function pair_apply_statusline_hl()
  vim.api.nvim_set_hl(0, 'StatusLine',   { link = 'Comment' })
  vim.api.nvim_set_hl(0, 'StatusLineNC', { link = 'Comment' })
  -- Pop the queued-count above the muted baseline when the queue is non-empty
  -- — it's the only segment that means "you have pending work to send."
  -- PairQueueZero matches the muted baseline normally, but exists as its own
  -- group so flash_queue_count can light it up on the N→0 transition too.
  vim.api.nvim_set_hl(0, 'PairQueueCount', { link = 'WarningMsg' })
  vim.api.nvim_set_hl(0, 'PairQueueZero',  { link = 'Comment' })
  -- Locked-mode banner reads at the same muted level as the rest of the
  -- statusline — the bg tint is the loud signal; the text just labels it.
  vim.api.nvim_set_hl(0, 'PairLocked', { link = 'Comment' })
  -- Pop the position marker (*, -N, +N) above the muted baseline so the
  -- "you are here" cue is unmistakable. Identifier is typically a cool
  -- accent (blue/cyan) — distinct from PairQueueCount's warning hue.
  -- Borrow DiffAdd's color so the position marker reads as a green "you
  -- are here" block. Many themes encode DiffAdd as fg=green + reverse so
  -- the rendered look is dark text on a green bg. `link` + extra attrs
  -- doesn't layer (link wins, attrs drop), so resolve and copy.
  local hit = vim.api.nvim_get_hl(0, { name = 'DiffAdd', link = false })
  vim.api.nvim_set_hl(0, 'PairPosLabel', {
    fg      = hit.fg,
    bg      = hit.bg,
    ctermfg = hit.ctermfg,
    ctermbg = hit.ctermbg,
    reverse = hit.reverse,
    bold    = true,
  })
end
pair_apply_statusline_hl()

-- ---------------------------------------------------------------------------
-- mode-tinted background — make non-insert modes visually obvious so a stray
-- middle-click paste / copy-on-select that lands in normal mode is caught
-- before keys get interpreted as commands instead of text.
-- ---------------------------------------------------------------------------
-- Insert mode keeps the colorscheme look. Locked mode (everything else)
-- swaps to a "dimmed sheet": lifted grey bg + uniform dim grey fg for
-- every syntax group, applied via a highlight namespace so the
-- colorscheme's default ns is left untouched.
local pair_bg_insert = '#1c1c1c' -- close to slate default
local pair_bg_locked = '#2a2a2a' -- lifted neutral grey
local pair_fg_locked = '#888888' -- dim grey fg for all syntax when locked
local pair_locked_ns = vim.api.nvim_create_namespace('pair_locked')

-- Snapshot every currently-defined highlight group and clone it into the
-- locked namespace with fg→dim, bg→locked, and decorations stripped.
-- Rebuilt on ColorScheme so new schemes / late-loaded tree-sitter groups
-- get covered. Cursor groups are skipped so the cursor block stays
-- visible against the dimmed sheet.
local function pair_build_locked_ns()
  for name in pairs(vim.api.nvim_get_hl(0, {})) do
    if name ~= 'Cursor' and name ~= 'lCursor' and name ~= 'TermCursor' then
      vim.api.nvim_set_hl(pair_locked_ns, name, {
        fg = pair_fg_locked,
        bg = pair_bg_locked,
      })
    end
  end
end
pair_build_locked_ns()

local function pair_apply_mode_bg(mode)
  if mode == 'n' then
    pair_build_locked_ns() -- catch any groups defined since last build
    vim.api.nvim_set_hl_ns(pair_locked_ns)
  else
    vim.api.nvim_set_hl_ns(0)
    vim.api.nvim_set_hl(0, 'Normal',      { bg = pair_bg_insert })
    vim.api.nvim_set_hl(0, 'NormalNC',    { bg = pair_bg_insert })
    vim.api.nvim_set_hl(0, 'EndOfBuffer', { bg = pair_bg_insert })
  end
end
-- Coalesce + defer: read mode on the next event-loop tick so transient
-- mode flips (e.g. `:normal! zt` inside a Lua callback that ends with
-- `startinsert`) don't strand us in the locked namespace. Synchronous
-- ModeChanged would see the intermediate 'n', and the trailing startinsert
-- only takes effect after the current tick — so by the time our scheduled
-- callback runs, vim.fn.mode() reports the settled mode.
local pair_mode_bg_pending = false
local function pair_schedule_mode_bg()
  if pair_mode_bg_pending then return end
  pair_mode_bg_pending = true
  vim.schedule(function()
    pair_mode_bg_pending = false
    pair_apply_mode_bg(vim.fn.mode():sub(1, 1))
  end)
end
vim.api.nvim_create_autocmd('ModeChanged', {
  callback = pair_schedule_mode_bg,
})
vim.api.nvim_create_autocmd('ColorScheme', {
  callback = pair_build_locked_ns,
})
pair_apply_mode_bg(vim.fn.mode():sub(1, 1))

-- ---------------------------------------------------------------------------
-- quit-blocker — fat-finger guard for muscle-memory :wq / :q / ZZ etc.
-- ---------------------------------------------------------------------------
-- This nvim instance is the pair draft pane, not a standalone editor. The
-- correct exits are Alt+x (full quit) or Alt+d (detach); a stray :wq would
-- kill the draft pane mid-session and orphan zellij's layout. We rewrite
-- the common quit verbs as a no-op that echoes the right path. Saves still
-- happen via autosave, so swallowing the `:w` part of `:wq` costs nothing.
function _G.PairQuitWarn()
  vim.api.nvim_echo({
    { 'pair: ', 'Question' },
    { 'use Alt+x to quit, or Alt+d to detach', 'WarningMsg' },
  }, false, {})
end

-- Match the WHOLE typed command exactly (cmdline ==# 'q' etc.) so this only
-- fires for bare quits, not e.g. `:qall` typed character-by-character or a
-- substitute pattern that happens to contain 'q'. The `<expr>` form lets us
-- branch on getcmdtype() so command-mode-only triggers fire.
local quit_verbs = {
  'q', 'q!', 'wq', 'wq!', 'quit', 'quit!',
  'qa', 'qa!', 'qall', 'qall!',
  'wqa', 'wqa!', 'wqall', 'wqall!',
  'x', 'x!', 'xa', 'xa!', 'xall', 'xall!',
  'exit', 'exit!',
}
for _, v in ipairs(quit_verbs) do
  vim.cmd(string.format(
    [[cnoreabbrev <expr> %s getcmdtype() == ':' && getcmdline() ==# %q ? 'lua PairQuitWarn()' : %q]],
    v, v, v
  ))
end

-- Normal-mode shortcuts that bypass the cmdline (and thus the abbreviations).
vim.keymap.set('n', 'ZZ', function() PairQuitWarn() end, { silent = true, desc = 'pair: quit blocked' })
vim.keymap.set('n', 'ZQ', function() PairQuitWarn() end, { silent = true, desc = 'pair: quit blocked' })

-- ---------------------------------------------------------------------------
-- keymaps
-- ---------------------------------------------------------------------------

vim.keymap.set({ 'n', 'i' }, '<M-CR>', send_and_clear,
  { silent = true, desc = 'pair: send buffer + clear' })

vim.keymap.set({ 'n', 'i' }, '<M-i>', attach_image,
  { silent = true, desc = 'pair: attach clipboard image (Ctrl+V to agent + ref)' })

vim.keymap.set({ 'n', 'i' }, '<M-Left>', nav_left,
  { silent = true, desc = 'pair: navigate to older history entry' })

vim.keymap.set({ 'n', 'i' }, '<M-Right>', nav_right,
  { silent = true, desc = 'pair: navigate toward draft / queue' })

vim.keymap.set({ 'n', 'i' }, '<S-M-Left>',  function() nav_boundary(-1) end,
  { silent = true, desc = 'pair: jump to previous region boundary' })
vim.keymap.set({ 'n', 'i' }, '<S-M-Right>', function() nav_boundary( 1) end,
  { silent = true, desc = 'pair: jump to next region boundary' })

vim.keymap.set({ 'n', 'i' }, '<M-q>', queue_current,
  { silent = true, desc = 'pair: queue current draft for later (back of queue)' })

vim.keymap.set({ 'n', 'i' }, '<M-BS>', delete_current_queue_item,
  { silent = true, desc = 'pair: delete the current +N queue item' })

vim.keymap.set({ 'n', 'i' }, '<S-M-BS>', forget_all,
  { silent = true, desc = 'pair: erase history + draft + queue (with confirm)' })

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

-- autosave on transitions so disk and buffer agree. Routes to the right
-- file per slot: * → draft via :w; +N → queue file via queue_write. -N is
-- immutable (history can't be mutated), so its edits wait for the explicit
-- Send/Queue/Discard choice in leave_dirty_history.
vim.api.nvim_create_autocmd({ 'BufLeave', 'FocusLost', 'InsertLeave' }, {
  group = pair_aug,
  pattern = '*',
  callback = function()
    autosave_current_slot()
    -- The :silent! write inside autosave can occasionally blank the
    -- statusline under cmdheight=0. Re-fire the (deferred) redraw so
    -- it comes back without the user needing an Alt+← to nudge it.
    refresh_statusline()
  end,
})

-- Statusline depends on mode (the dirty-N* mark) and on focus state.
-- ModeChanged and FocusGained/FocusLost should both trigger a redraw,
-- but nvim doesn't always re-evaluate the statusline on these events
-- under cmdheight=0. Defensive explicit refresh.
vim.api.nvim_create_autocmd({ 'ModeChanged', 'FocusGained', 'FocusLost' }, {
  group = pair_aug,
  callback = function() refresh_statusline() end,
})

-- Re-apply the soft statusline highlight on colorscheme changes (each
-- :colorscheme implicitly runs :hi clear, blowing away our link).
vim.api.nvim_create_autocmd('ColorScheme', {
  group = pair_aug,
  callback = pair_apply_statusline_hl,
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

-- Insert-mode-only keymap that triggers PairPasteQuote. This is what
-- bin/clipboard-to-pane.sh sends (as a single Ctrl-_, ASCII 31) after a
-- mouse selection. Defining the keymap *only* in insert mode is the gate:
-- if nvim is in normal mode (e.g. browsing prompt history), Ctrl-_ hits
-- its default — a no-op-ish revins toggle — and PairPasteQuote simply
-- doesn't fire. No buffer mutation, no policy code.
vim.keymap.set('i', '<C-_>', function() PairPasteQuote() end,
  { silent = true, desc = 'pair: insert mouse-selected quote (insert mode only)' })

-- Fire on both events: TextChangedI when popup is hidden, TextChangedP when
-- popup is visible — refreshing the menu as the user types more characters.
vim.api.nvim_create_autocmd({ 'TextChangedI', 'TextChangedP' }, {
  group = pair_aug,
  callback = path_complete,
})

-- "Ghost cursor" while the nvim pane is unfocused. zellij hides the real
-- terminal cursor on FocusLost, leaving the insertion point invisible.
-- Mark the position with a glyph chosen by mode so the indicator mirrors
-- the focused-state cursor:
--   normal-mode unfocused : ▯ (outline of █, the focused block cursor)
--   insert-mode unfocused : ¦ (broken version of |, the focused bar cursor)
local pair_focus_ns = vim.api.nvim_create_namespace('pair_focus_cursor')

local function pair_apply_focus_cursor_hl()
  -- Tie to `Comment` so the glyph picks up the colorscheme's dimmed-text
  -- color — visible but subdued. Reapplied on ColorScheme since :hi clear
  -- (which colorschemes implicitly run) blows highlights away.
  vim.api.nvim_set_hl(0, 'PairFocusCursor', { link = 'Comment' })
end
pair_apply_focus_cursor_hl()

local function pair_show_focus_cursor()
  local mode  = vim.api.nvim_get_mode().mode:sub(1, 1)
  local glyph = (mode == 'i') and '¦' or '▯'
  local row1, col = unpack(vim.api.nvim_win_get_cursor(0))
  local row = row1 - 1
  vim.api.nvim_buf_clear_namespace(0, pair_focus_ns, 0, -1)
  pcall(vim.api.nvim_buf_set_extmark, 0, pair_focus_ns, row, col, {
    virt_text     = { { glyph, 'PairFocusCursor' } },
    virt_text_pos = 'overlay',
    priority      = 200,
  })
end

local function pair_hide_focus_cursor()
  vim.api.nvim_buf_clear_namespace(0, pair_focus_ns, 0, -1)
end

vim.api.nvim_create_autocmd('FocusLost', {
  group = pair_aug,
  callback = function()
    pair_show_focus_cursor()
    -- A delayed full redraw catches the case where zellij's focus-change
    -- rendering fires after our immediate refresh_statusline (which only
    -- defers one event-loop tick). 80ms is comfortably above one terminal
    -- frame and unobtrusive.
    vim.defer_fn(function() pcall(vim.cmd, 'redraw!') end, 80)
  end,
})
vim.api.nvim_create_autocmd('FocusGained', { group = pair_aug, callback = pair_hide_focus_cursor })
vim.api.nvim_create_autocmd('ColorScheme', { group = pair_aug, callback = pair_apply_focus_cursor_hl })

-- Pane name = "draft" + spaces + cheatsheet, sized to the terminal width
-- so the cheatsheet ends up right-aligned in the zellij frame title. Why:
-- zellij sets the OSC 0 terminal title to "<session>: <pane-name>", which
-- typical terminal/multiplexer tab titles truncate. Padding spaces between
-- "draft" and the cheatsheet means the title truncates *during* the
-- spaces, so the visible tab title stays short ("pair-pair: draft") while
-- the in-frame display shows the full cheatsheet right-aligned.
local PAIR_CHEATSHEET = 'Alt: ⏎=send  u=maximize  i=img  d=detach  x=quit'

-- UTF-8 encoding of U+00A0 NO-BREAK SPACE. Same display width as a regular
-- space but zellij doesn't trim/collapse it the way it does ordinary
-- whitespace in pane names.
local NBSP = string.char(0xC2, 0xA0)

local function pair_update_pane_name()
  local cheat_w = vim.fn.strdisplaywidth(PAIR_CHEATSHEET)
  -- vim.o.columns is the nvim window width = the pane's inner width.
  -- Subtract a small fudge for zellij's frame chrome (corners + the
  -- "─ " / " ─" that bracket the title slot).
  local pad = vim.o.columns - 4 - vim.fn.strdisplaywidth('draft') - cheat_w
  if pad < 2 then pad = 2 end
  local name = 'draft' .. string.rep(NBSP, pad) .. PAIR_CHEATSHEET

  -- Target the nvim pane explicitly via its ID, not "the focused pane".
  -- At startup zellij may show a floating "tip" popup that grabs focus,
  -- and rename-pane would then rename the popup instead of us. zellij
  -- exports ZELLIJ_PANE_ID in every pane's env — that's our ID.
  local pane_id = os.getenv('ZELLIJ_PANE_ID')
  local cmd
  if pane_id and pane_id ~= '' then
    cmd = { 'zellij', 'action', 'rename-pane', '--pane-id', pane_id, name }
  else
    cmd = { 'zellij', 'action', 'rename-pane', name }
  end
  pcall(vim.fn.system, cmd)
end

vim.api.nvim_create_autocmd({ 'VimEnter', 'VimResized' }, {
  group = pair_aug,
  callback = vim.schedule_wrap(pair_update_pane_name),
})
