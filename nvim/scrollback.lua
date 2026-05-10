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
    vim.keymap.set('n', 'q', '<cmd>qa<CR>', { buffer = bufnr, silent = true })
  end,
})

-- Cosmetic: thin status line, no ruler, line numbers on so :880 has a
-- visible target.
vim.opt.number = true
vim.opt.relativenumber = false
vim.opt.cursorline = true
vim.opt.laststatus = 2
vim.opt.statusline = ' pair scrollback · q to quit · :N to jump to line N '
