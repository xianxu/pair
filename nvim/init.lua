-- pair/nvim/init.lua
-- Minimal nvim config for the pair input pane. Loaded via `nvim -u`,
-- so this is fully isolated from the user's normal nvim setup.

vim.g.mapleader = ' '

-- Drafting-friendly editor settings
vim.opt.number = false
vim.opt.relativenumber = false
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
  pcall(vim.cmd, 'redrawstatus')
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

  -- send-from-+N consumes that queue file. The buffer (possibly edited) is
  -- what ships, and the queue slot vanishes.
  if type(nav.pos) == 'table' and nav.pos.kind == 'queue' then
    local key = queue_key_for_n(nav.pos.n)
    if key then queue_remove(key) end
  end

  local was_at_star = (nav.pos == '*')

  append_log(body)
  send_to_agent(body)

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
local function is_dirty_history_slot()
  return type(nav.pos) == 'table'
     and nav.pos.kind == 'history'
     and buffer_text() ~= nav.baseline
end

-- Statusline format:
--   " Alt: <- history H < pos[*] > Q queued -> "
-- The flanking arrows hint that Alt+← walks toward history and Alt+→ walks
-- toward the queue. The trailing "*" on `pos` appears only when on -N with
-- an unsent fork.
function _G.PairStatusline()
  local h = #read_history()
  local q = queue_count()
  local label = pos_label(nav.pos)
  if is_dirty_history_slot() then label = label .. '*' end
  return string.format(' Alt: <- history %d < %s > %d queued -> ', h, label, q)
end

-- Persist any pending edit on a mutable slot to its underlying file. No-op
-- for -N (immutable; user must explicitly pick Send/Queue/Discard via the
-- leave-dirty-history prompt) and for any state where there's nothing to do.
local function autosave_current_slot()
  if nav.pos == '*' then
    pcall(vim.cmd, 'silent! write')
  elseif type(nav.pos) == 'table' and nav.pos.kind == 'queue' then
    local key = queue_key_for_n(nav.pos.n)
    if key then queue_write(key, buffer_text()) end
  end
end

-- Send the current buffer to the agent and return to *, preserving *'s
-- persistent draft. Used only by the dirty-`-N` prompt's Send branch
-- (send_and_clear has its own variant that handles the from-* case).
local function ship_buffer_and_reset(body)
  append_log(body)
  send_to_agent(body)
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

-- Alt+BS — delete the current +N queue item without sending it. "Stay near":
-- after delete, items at +(N+1)..+M shift down by one, so the same +N slot
-- now displays what used to be next. Lets the user tap-tap to clean out a
-- run. If queue empties, fall back to *.
local function delete_current_queue_item()
  if type(nav.pos) ~= 'table' or nav.pos.kind ~= 'queue' then return end
  local key = queue_key_for_n(nav.pos.n)
  if not key then return end
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
end
pair_apply_statusline_hl()

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

vim.keymap.set({ 'n', 'i' }, '<M-q>', queue_current,
  { silent = true, desc = 'pair: queue current draft for later (back of queue)' })

vim.keymap.set({ 'n', 'i' }, '<M-BS>', delete_current_queue_item,
  { silent = true, desc = 'pair: delete the current +N queue item' })

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
  callback = function() autosave_current_slot() end,
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

-- Fire on both events: TextChangedI when popup is hidden, TextChangedP when
-- popup is visible — refreshing the menu as the user types more characters.
vim.api.nvim_create_autocmd({ 'TextChangedI', 'TextChangedP' }, {
  group = pair_aug,
  callback = path_complete,
})
