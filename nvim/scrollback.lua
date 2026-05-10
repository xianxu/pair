-- nvim/scrollback.lua — read-only viewer for pair-scrollback-render output.
--
-- Loaded via `nvim -u $PAIR_HOME/nvim/scrollback.lua <path-to-.ansi>`.
-- The .ansi file has SGR escapes inline (`\x1b[...m`); this init strips
-- them from the buffer and re-applies their effect via extmarks so the
-- visible content matches what the agent rendered, line numbers match
-- zellij's scroll indicator, and `:880` jumps where the user expects.
--
-- Filed as #000017 M4. Plugin-free on purpose: this file is the entire
-- viewer. ~150 lines is small enough to audit in one pass; pulling in
-- baleia.nvim or AnsiEsc.vim would add a vendor dir for marginal gain.

vim.opt.termguicolors = true
vim.opt.compatible = false

-- See init.lua for the full rationale: this writes the embed nvim's pid to
-- $PAIR_NVIM_PID_FILE so bin/pair's cleanup_quit_marker can reap it on
-- Alt+x without resorting to argv pattern matching. Scrollback nvim has
-- the same TUI/embed fork shape as the draft, so the same leak applies if
-- the user Alt+x's while this viewer is floating.
do
  local pidfile = vim.env.PAIR_NVIM_PID_FILE
  if pidfile and pidfile ~= '' then
    vim.api.nvim_create_autocmd('VimEnter', {
      once = true,
      callback = function()
        pcall(vim.fn.writefile, { tostring(vim.fn.getpid()) }, pidfile)
      end,
    })
  end
end

-- Standard 16-color palette. Approximate xterm defaults — close enough
-- to most TUI palettes that the rendered colors feel right. No user
-- configuration; the file is supposed to look like the agent's output.
local PALETTE = {
  ['30'] = '#000000', ['31'] = '#cc0000', ['32'] = '#4e9a06', ['33'] = '#c4a000',
  ['34'] = '#3465a4', ['35'] = '#75507b', ['36'] = '#06989a', ['37'] = '#d3d7cf',
  ['90'] = '#555753', ['91'] = '#ef2929', ['92'] = '#8ae234', ['93'] = '#fce94f',
  ['94'] = '#729fcf', ['95'] = '#ad7fa8', ['96'] = '#34e2e2', ['97'] = '#eeeeec',
  ['40'] = '#000000', ['41'] = '#cc0000', ['42'] = '#4e9a06', ['43'] = '#c4a000',
  ['44'] = '#3465a4', ['45'] = '#75507b', ['46'] = '#06989a', ['47'] = '#d3d7cf',
  ['100'] = '#555753', ['101'] = '#ef2929', ['102'] = '#8ae234', ['103'] = '#fce94f',
  ['104'] = '#729fcf', ['105'] = '#ad7fa8', ['106'] = '#34e2e2', ['107'] = '#eeeeec',
}

-- Apply one SGR `m` escape's parameter list to a state table. Mutates
-- `state` in place; codes consume up to 4 list slots for 38;2;r;g;b
-- truecolor and 38;5;n indexed (which we map approximately to RGB via
-- a small 256-color formula — see resolve_256).
local function resolve_256(n)
  if n < 16 then
    -- Standard 16: pull from the same xterm palette as named codes.
    local key = (n < 8) and tostring(30 + n) or tostring(90 + n - 8)
    return PALETTE[key]
  elseif n < 232 then
    -- 6×6×6 cube
    n = n - 16
    local r = math.floor(n / 36)
    local g = math.floor((n % 36) / 6)
    local b = n % 6
    local function comp(c) if c == 0 then return 0 else return 55 + c * 40 end end
    return string.format('#%02x%02x%02x', comp(r), comp(g), comp(b))
  else
    -- Greyscale ramp 232-255
    local v = 8 + (n - 232) * 10
    return string.format('#%02x%02x%02x', v, v, v)
  end
end

local function apply_sgr(state, params)
  -- Empty params == reset (`\x1b[m`).
  if params == '' then
    state.fg, state.bg = nil, nil
    state.bold, state.italic, state.underline, state.reverse, state.strike = false, false, false, false, false
    return
  end
  -- Match every `;`-delimited field, *including empty ones*. ECMA-48
  -- treats an omitted SGR parameter as 0 (reset), so `\x1b[;1m` is
  -- "reset + bold" — splitting on `[^;]+` would drop the empty leading
  -- field and leave any standing fg/bg/decoration in place. Appending
  -- a trailing `;` and matching `([^;]*);` makes every field — empty
  -- or not — a separate code.
  local codes = {}
  for code in (params .. ';'):gmatch('([^;]*);') do
    table.insert(codes, code)
  end
  local i = 1
  while i <= #codes do
    local n = tonumber(codes[i])
    -- Empty field == 0 (reset).
    if codes[i] == '' then n = 0 end
    if n == nil then i = i + 1 ; goto continue end
    if n == 0 then
      state.fg, state.bg = nil, nil
      state.bold, state.italic, state.underline, state.reverse, state.strike = false, false, false, false, false
    elseif n == 1 then state.bold = true
    elseif n == 22 then state.bold = false
    elseif n == 3 then state.italic = true
    elseif n == 23 then state.italic = false
    elseif n == 4 then state.underline = true
    elseif n == 24 then state.underline = false
    elseif n == 7 then state.reverse = true
    elseif n == 27 then state.reverse = false
    elseif n == 9 then state.strike = true
    elseif n == 29 then state.strike = false
    elseif (n >= 30 and n <= 37) or (n >= 90 and n <= 97) then
      state.fg = PALETTE[tostring(n)]
    elseif n == 39 then state.fg = nil
    elseif (n >= 40 and n <= 47) or (n >= 100 and n <= 107) then
      state.bg = PALETTE[tostring(n)]
    elseif n == 49 then state.bg = nil
    elseif n == 38 or n == 48 then
      local mode = tonumber(codes[i + 1] or '')
      if mode == 5 and codes[i + 2] then
        local idx = tonumber(codes[i + 2])
        if idx then
          if n == 38 then state.fg = resolve_256(idx) else state.bg = resolve_256(idx) end
        end
        i = i + 2
      elseif mode == 2 and codes[i + 4] then
        local r = tonumber(codes[i + 2]) or 0
        local g = tonumber(codes[i + 3]) or 0
        local b = tonumber(codes[i + 4]) or 0
        local hex = string.format('#%02x%02x%02x', r, g, b)
        if n == 38 then state.fg = hex else state.bg = hex end
        i = i + 4
      end
    end
    ::continue::
    i = i + 1
  end
end

-- Cache resolved (state → hl-group-name) so we don't create duplicate
-- highlight groups for every cell. Separate counter because hl_cache
-- is string-keyed (`#` on a dict returns 0, not the entry count).
local hl_cache = {}
local hl_counter = 0

local function hl_for(state)
  -- Build a stable cache key from the state attrs.
  local key = (state.fg or '_') .. '|' .. (state.bg or '_')
    .. '|' .. (state.bold and 'B' or '')
    .. (state.italic and 'I' or '')
    .. (state.underline and 'U' or '')
    .. (state.reverse and 'R' or '')
    .. (state.strike and 'S' or '')
  if hl_cache[key] then return hl_cache[key] end
  hl_counter = hl_counter + 1
  local name = 'PairScrollback_' .. hl_counter
  local def = {}
  if state.fg then def.fg = state.fg end
  if state.bg then def.bg = state.bg end
  if state.bold then def.bold = true end
  if state.italic then def.italic = true end
  if state.underline then def.underline = true end
  if state.reverse then def.reverse = true end
  if state.strike then def.strikethrough = true end
  vim.api.nvim_set_hl(0, name, def)
  hl_cache[key] = name
  return name
end

-- Walk one line: peel off SGR escapes (mutating a running state),
-- accumulate the visible bytes into stripped, and emit extmark spans
-- for each contiguous run of visible bytes under a single state.
local SGR = '\27%[[%d;]*m'

local function process_line(line)
  local stripped_parts = {}
  local spans = {} -- { {col_start, col_end, hl_group} ... } using BYTE indices
  local state = { fg = nil, bg = nil, bold = false, italic = false,
                  underline = false, reverse = false, strike = false }
  local i = 1
  local stripped_len = 0
  while i <= #line do
    local s, e = line:find(SGR, i)
    if s and s > i then
      -- Plain text up to the next escape.
      local seg = line:sub(i, s - 1)
      table.insert(stripped_parts, seg)
      local col_start = stripped_len
      stripped_len = stripped_len + #seg
      table.insert(spans, { col_start, stripped_len, hl_for(state) })
      i = s
    elseif s == i then
      -- Escape at current cursor.
      local params = line:sub(s + 2, e - 1)
      apply_sgr(state, params)
      i = e + 1
    else
      -- Tail with no more escapes.
      local seg = line:sub(i)
      table.insert(stripped_parts, seg)
      local col_start = stripped_len
      stripped_len = stripped_len + #seg
      table.insert(spans, { col_start, stripped_len, hl_for(state) })
      i = #line + 1
    end
  end
  return table.concat(stripped_parts), spans
end

local function decorate_buffer(bufnr)
  local ns = vim.api.nvim_create_namespace('pair_scrollback')
  local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
  local stripped = {}
  local all_spans = {} -- per-line spans
  for idx, raw in ipairs(lines) do
    local s, sp = process_line(raw)
    stripped[idx] = s
    all_spans[idx] = sp
  end
  -- Replace buffer content with stripped text first; extmarks attach
  -- to the new content.
  vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, stripped)
  vim.api.nvim_buf_clear_namespace(bufnr, ns, 0, -1)
  for idx, sp in ipairs(all_spans) do
    for _, span in ipairs(sp) do
      local col_start, col_end, hl = span[1], span[2], span[3]
      if col_end > col_start then
        vim.api.nvim_buf_set_extmark(bufnr, ns, idx - 1, col_start, {
          end_col = col_end,
          hl_group = hl,
        })
      end
    end
  end
end

-- ---------------------------------------------------------------------------
-- Alt+q comment markers (#000018) — let the user drop parley-style
-- 🤖[] markers while reading scrollback. Two forms:
--   normal mode: 🤖[<comment>] appended to the current line
--   visual mode: 🤖<selection>[<comment>] in place of the selection
-- The buffer is read-only by default; we lift modifiable for the insert
-- and re-lock immediately. On VimLeavePre the markers are extracted and
-- written to a sidecar file for the draft pane to pick up.

-- 🤖 = U+1F916, four bytes in UTF-8: F0 9F A4 96. Lua patterns aren't
-- UTF-8-aware so we use the literal byte sequence with `find(..., 1, true)`.
local MARKER_BOT = '\240\159\164\150'

-- Escape/unescape so user-supplied X (selection) and Y (comment) can
-- contain the marker delimiters `>` and `]` without prematurely
-- terminating the surrounding `<...>[...]` brackets. Backslash is the
-- escape char and is itself escaped first so the unescape pass can be
-- a single regex-free walk.
--
-- Without this, a selection like `git log --grep '> '` or a comment
-- containing `]` would produce a marker the parser silently truncates,
-- and the user loses their comment between Alt+q and `:q` (data-loss
-- footgun caught in #18 review).
local function esc_x(s)
  return (s:gsub('\\', '\\\\'):gsub('>', '\\>'):gsub(']', '\\]'))
end

local function esc_y(s)
  return (s:gsub('\\', '\\\\'):gsub(']', '\\]'))
end

local function unescape(s)
  -- Walk byte-by-byte so `\\>` correctly unescapes to `\` + `>` (not
  -- `\>`). The placeholder-via-NUL approach failed because Lua patterns
  -- don't treat a literal NUL byte as a single-char match — it ends up
  -- matching empty positions between every character.
  local out = {}
  local i = 1
  while i <= #s do
    local c = s:sub(i, i)
    if c == '\\' and i < #s then
      table.insert(out, s:sub(i + 1, i + 1))
      i = i + 2
    else
      table.insert(out, c)
      i = i + 1
    end
  end
  return table.concat(out)
end

-- Find first occurrence of `char` in `line` starting at `start_pos`
-- that is NOT escaped (i.e., not preceded by an odd number of `\`s).
-- Returns nil if none. Used to locate the real `>` and `]` that
-- close a marker, ignoring escaped ones inside X / Y.
local function find_unescaped(line, char, start_pos)
  local i = start_pos
  while true do
    local idx = line:find(char, i, true)
    if not idx then return nil end
    local bs = 0
    local j = idx - 1
    while j >= start_pos and line:sub(j, j) == '\\' do
      bs = bs + 1
      j = j - 1
    end
    if bs % 2 == 0 then return idx end
    i = idx + 1
  end
end

-- Walk one line, return every marker as { kind, X?, Y, range = {byte_lo, byte_hi} }.
-- Pure function — exposed so headless tests can exercise it without a buffer.
local function find_markers_in_line(line)
  local out = {}
  local i = 1
  while i <= #line do
    local s = line:find(MARKER_BOT, i, true)
    if not s then break end
    local after = s + #MARKER_BOT
    local consumed = nil
    if line:sub(after, after) == '<' then
      local close_q = find_unescaped(line, '>', after + 1)
      if close_q and line:sub(close_q + 1, close_q + 1) == '[' then
        local close_b = find_unescaped(line, ']', close_q + 2)
        if close_b then
          table.insert(out, {
            kind = 'scoped',
            X = unescape(line:sub(after + 1, close_q - 1)),
            Y = unescape(line:sub(close_q + 2, close_b - 1)),
            range = { s, close_b },
          })
          consumed = close_b + 1
        end
      end
    elseif line:sub(after, after) == '[' then
      local close_b = find_unescaped(line, ']', after + 1)
      if close_b then
        table.insert(out, {
          kind = 'bare',
          Y = unescape(line:sub(after + 1, close_b - 1)),
          range = { s, close_b },
        })
        consumed = close_b + 1
      end
    end
    i = consumed or (s + #MARKER_BOT)
  end
  return out
end

-- Remove every marker from `line` (back-to-front so earlier ranges stay
-- valid), trim leading/trailing whitespace. Used for bare-marker context:
-- the marker carries no quoted text, so we quote the whole line minus
-- any markers it contains.
local function strip_markers(line, markers)
  local sorted = vim.deepcopy(markers)
  table.sort(sorted, function(a, b) return a.range[1] > b.range[1] end)
  for _, m in ipairs(sorted) do
    line = line:sub(1, m.range[1] - 1) .. line:sub(m.range[2] + 1)
  end
  return (line:gsub('^%s+', ''):gsub('%s+$', ''))
end

-- Walk the whole buffer and return the formatted extraction block.
-- Each marker becomes:
--   > <quote>
--   <comment>
-- Markers are separated by a blank line. Returns "" when no markers exist.
local function format_extraction(buf_lines)
  local pieces = {}
  for _, line in ipairs(buf_lines) do
    local markers = find_markers_in_line(line)
    if #markers > 0 then
      local stripped = strip_markers(line, markers)
      for _, m in ipairs(markers) do
        local quote = (m.kind == 'scoped') and m.X or stripped
        if quote == '' then
          -- Edge: bare marker on a line that's *only* the marker. Fall
          -- back to a placeholder so the pickup side knows there was a
          -- standalone note.
          quote = '(no context)'
        end
        table.insert(pieces, '> ' .. quote .. '\n' .. m.Y)
      end
    end
  end
  return table.concat(pieces, '\n\n')
end

-- Expose for tests; headless harness pokes these directly.
_G.PairScrollbackTest = {
  find_markers_in_line = find_markers_in_line,
  strip_markers        = strip_markers,
  format_extraction    = format_extraction,
  esc_x                = esc_x,
  esc_y                = esc_y,
  unescape             = unescape,
}

local function prompt_comment()
  -- vim.fn.input swallows a trailing CR; we get whatever the user typed.
  -- Empty input cancels — caller bails out without modifying the buffer.
  local ok, comment = pcall(vim.fn.input, 'Comment: ')
  if not ok then return nil end
  comment = comment or ''
  if comment == '' then return nil end
  return comment
end

local function add_marker_normal(bufnr)
  local comment = prompt_comment()
  if not comment then return end
  local row = vim.api.nvim_win_get_cursor(0)[1]
  local line = vim.api.nvim_buf_get_lines(bufnr, row - 1, row, false)[1] or ''
  local marker = MARKER_BOT .. '[' .. esc_y(comment) .. ']'
  local sep = (line == '' or line:match('%s$')) and '' or ' '
  vim.bo[bufnr].modifiable = true
  vim.bo[bufnr].readonly   = false
  vim.api.nvim_buf_set_lines(bufnr, row - 1, row, false, { line .. sep .. marker })
  vim.bo[bufnr].modifiable = false
  vim.bo[bufnr].readonly   = true
end

local function add_marker_visual(bufnr)
  -- Live selection positions while still in visual mode (the '< / '>
  -- marks aren't set until visual exits, so we read 'v' and '.' which
  -- represent the start and current cursor of the active selection).
  local sr = vim.fn.line('v')
  local sc = vim.fn.col('v')
  local er = vim.fn.line('.')
  local ec = vim.fn.col('.')
  if sr > er or (sr == er and sc > ec) then
    sr, er = er, sr
    sc, ec = ec, sc
  end
  if sr ~= er then
    vim.notify('🤖 marker: multi-line selection not supported',
               vim.log.levels.WARN)
    vim.cmd('normal! \27')
    return
  end
  vim.cmd('normal! \27')  -- exit visual so the upcoming input prompt works
  local line = vim.api.nvim_buf_get_lines(bufnr, sr - 1, sr, false)[1] or ''
  ec = math.min(ec, #line)  -- clamp for line-wise V (col is huge there)
  local before = line:sub(1, sc - 1)
  local sel    = line:sub(sc, ec)
  local after  = line:sub(ec + 1)
  if sel == '' then return end
  local comment = prompt_comment()
  if not comment then return end
  local marker = MARKER_BOT .. '<' .. esc_x(sel) .. '>[' .. esc_y(comment) .. ']'
  vim.bo[bufnr].modifiable = true
  vim.bo[bufnr].readonly   = false
  vim.api.nvim_buf_set_lines(bufnr, sr - 1, sr, false, { before .. marker .. after })
  vim.bo[bufnr].modifiable = false
  vim.bo[bufnr].readonly   = true
end

-- Sidecar path the draft nvim picks up on FocusGained. Resolved at
-- VimLeavePre so we don't need to thread it through the autocmd args.
local function sidecar_path()
  local data_dir = vim.env.PAIR_DATA_DIR
    or ((vim.env.XDG_DATA_HOME or (vim.env.HOME .. '/.local/share')) .. '/pair')
  local tag = vim.env.PAIR_TAG or vim.env.PAIR_AGENT or 'claude'
  return data_dir .. '/scrollback-pending-' .. tag .. '.md'
end

local function emit_pending(bufnr)
  local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
  local block = format_extraction(lines)
  if block == '' then return end  -- silent no-op when no markers were dropped
  local path = sidecar_path()
  local tmp = path .. '.tmp'
  local f = io.open(tmp, 'w')
  if not f then return end
  -- Preceding blank line ensures the draft's existing content gets a
  -- visible separator when the pickup side appends.
  f:write('\n')
  f:write(block)
  f:write('\n')
  f:close()
  os.rename(tmp, path)
end

-- Wire it up: when the file is loaded, decorate then lock the buffer.
vim.api.nvim_create_autocmd('BufReadPost', {
  pattern = '*',
  callback = function(args)
    local bufnr = args.buf
    decorate_buffer(bufnr)
    vim.bo[bufnr].modifiable = false
    vim.bo[bufnr].readonly = true
    vim.bo[bufnr].buftype = 'nofile'  -- prevent accidental :w
    vim.bo[bufnr].swapfile = false
    -- Buffer-local sentinel for VimLeavePre lookup. Picking via
    -- buftype='nofile' alone is fragile — any plugin spawning a scratch
    -- nofile buffer would shadow ours. Tag the one we own.
    vim.b[bufnr].pair_scrollback = true
    -- Two quit keys: `q` (less-style, fast — no Esc-prefix timeout)
    -- and `<Esc>` (more vim/idiomatic for a read-only viewer; matches
    -- the pair-help less binding). Esc only in normal mode so visual-
    -- mode selection still cancels via Esc as usual.
    vim.keymap.set('n', 'q', '<cmd>qa<CR>', { buffer = bufnr, silent = true })
    vim.keymap.set('n', '<Esc>', '<cmd>qa<CR>', { buffer = bufnr, silent = true })
    vim.keymap.set('n', '<M-q>', function() add_marker_normal(bufnr) end,
                   { buffer = bufnr, silent = true })
    vim.keymap.set('x', '<M-q>', function() add_marker_visual(bufnr) end,
                   { buffer = bufnr, silent = true })
  end,
})

vim.api.nvim_create_autocmd('VimLeavePre', {
  callback = function()
    -- Look up by the buffer-local sentinel set in BufReadPost — robust
    -- against any other nofile buffers that might exist (terminals,
    -- plugin scratches, etc.).
    for _, b in ipairs(vim.api.nvim_list_bufs()) do
      if vim.api.nvim_buf_is_loaded(b) and vim.b[b].pair_scrollback then
        emit_pending(b)
        return
      end
    end
  end,
})

-- Reclaim every available column for the rendered scrollback: the
-- agent pane writes lines that nearly fill its width, and any nvim
-- gutter (line numbers, sign column) would push the tail of those
-- lines into a wrap. `:N` still jumps to line N without a visible
-- gutter — line position lives in the statusline instead.
vim.opt.number = false
vim.opt.relativenumber = false
vim.opt.signcolumn = 'no'
vim.opt.foldcolumn = '0'
vim.opt.cursorline = true
vim.opt.laststatus = 2
vim.opt.statusline = ' pair scrollback · q/Esc quit · Alt+q 🤖[] · :N jump %= L%l/%L '
