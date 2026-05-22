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

-- Stub out the pair-launcher cmdline targets so a stray zellij Alt+Up
-- / Alt+Down / Alt+x / Alt+n / Alt+d / Shift+Alt+N pressed while the
-- scrollback viewer is the focused pane degrades silently rather than
-- erroring on an undefined global.
--
-- zellij's bindings for those chords issue `MoveFocus Down` and then
-- `WriteChars ":lua PairLayoutBigger()"` (plus a CR). The MoveFocus
-- assumes the destination is the draft pane (where init.lua defines
-- the Pair* globals), but from inside the Alt+/ floating viewer the
-- focus shift doesn't escape the floating layer — the cmdline call
-- lands here instead. Defining the names as no-ops at scrollback-
-- nvim scope side-steps the error without touching zellij's
-- assumptions. Performing the actual layout / quit / restart action
-- from within scrollback would be the wrong default anyway; the user
-- almost certainly meant to act on the draft.
for _, name in ipairs({
  'PairLayoutBigger', 'PairLayoutSmaller',
  'PairConfirmQuit',  'PairConfirmDetach',
  'PairConfirmRestart', 'PairConfirmRestartNewSession',
}) do
  _G[name] = function() end
end

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

-- Per-agent user-prompt marker. Used by Alt+b to skip backward through
-- the user's turns in the scrollback. vim's regex engine (which
-- vim.fn.search uses) is UTF-8 aware, so the literal character in the
-- pattern works. Each agent paints its prompt-input line with a
-- distinct leading glyph:
--   claude — ❯  (U+276F, HEAVY RIGHT-POINTING ANGLE QUOTATION MARK)
--   codex  — ›  (U+203A, SINGLE RIGHT-POINTING ANGLE QUOTATION MARK)
--   gemini — leading space + `>` (no glyph; matches the indented `> `
--            input row gemini renders below its border)
-- Lookup falls back to claude's pattern so unknown agents still get a
-- useful default.
local PROMPT_PATTERN_BY_AGENT = {
  claude = [[^❯]],
  codex  = [[^›]],
  gemini = [[^ >]],
}

local function prompt_pattern()
  local agent = vim.env.PAIR_AGENT or ''
  return PROMPT_PATTERN_BY_AGENT[agent] or PROMPT_PATTERN_BY_AGENT.claude
end

-- Alt+b / Alt+Shift+B: search backward / forward for the next line
-- starting with the per-agent prompt marker. Lands on the prompt line
-- and pulls it to the top of the window (zt) so the response below
-- stays in view. No-wrap: hitting either end of scrollback with no
-- further matches reports "no {previous,next} prompt" rather than
-- silently jumping to the other end.
local function jump_to_prompt(direction)
  local pat = prompt_pattern()
  -- 'b' = backward, omitted = forward. 'W' = no wrap.
  local flags = (direction == 'prev') and 'bW' or 'W'
  if vim.fn.search(pat, flags) == 0 then
    vim.notify('no ' .. direction .. ' prompt', vim.log.levels.INFO)
  else
    vim.cmd('normal! zt')
  end
end

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

-- Walk one line, return every marker as { kind, X?, Y, range, parts }.
--   range = { byte_lo, byte_hi } — 1-based, inclusive, covers the whole marker.
--   parts = byte ranges of each colored component (for the highlighter):
--     bare:   { robot, lb, y, rb }
--     scoped: { robot, lt, x, gt, lb, y, rb }
--   each part is { lo, hi } 1-based inclusive; an empty X yields lo > hi
--   (caller skips zero-width parts).
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
            parts = {
              robot = { s, s + #MARKER_BOT - 1 },
              lt    = { after, after },
              x     = { after + 1, close_q - 1 },
              gt    = { close_q, close_q },
              lb    = { close_q + 1, close_q + 1 },
              y     = { close_q + 2, close_b - 1 },
              rb    = { close_b, close_b },
            },
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
          parts = {
            robot = { s, s + #MARKER_BOT - 1 },
            lb    = { after, after },
            y     = { after + 1, close_b - 1 },
            rb    = { close_b, close_b },
          },
        })
        consumed = close_b + 1
      end
    end
    i = consumed or (s + #MARKER_BOT)
  end
  return out
end

-- 🤖[] marker syntax highlighting. Separate namespace from the SGR ANSI
-- decoration so we can re-render markers (after each Alt+q insertion)
-- without rebuilding the much larger SGR span set. `default = true`
-- lets a colorscheme override; the fallback links keep things readable
-- against most schemes without picking absolute colors.
vim.api.nvim_set_hl(0, 'PairRobotIcon',      { default = true, link = 'Special'    })
vim.api.nvim_set_hl(0, 'PairRobotBracket',   { default = true, link = 'Delimiter'  })
vim.api.nvim_set_hl(0, 'PairRobotSelection', { default = true, link = 'String'     })
vim.api.nvim_set_hl(0, 'PairRobotComment',   { default = true, link = 'Identifier', bold = true })

-- Re-render 🤖[] markers across the buffer (or a single line range).
-- Cheap enough to run on every Alt+q insertion since markers are
-- typically a handful per buffer; if that ever stops being true,
-- pass `lo, hi` to scope to the modified line.
local function highlight_markers(bufnr, lo, hi)
  local ns = vim.api.nvim_create_namespace('pair_scrollback_markers')
  lo = lo or 0
  hi = hi or -1
  vim.api.nvim_buf_clear_namespace(bufnr, ns, lo, hi)
  local lines = vim.api.nvim_buf_get_lines(bufnr, lo, hi, false)
  local function emit(row, part, hl)
    -- Skip empty parts (e.g. scoped marker with empty X) — col_start ==
    -- col_end is a no-op extmark but we save the API call.
    if part[2] < part[1] then return end
    vim.api.nvim_buf_set_extmark(bufnr, ns, row, part[1] - 1, {
      end_col = part[2],
      hl_group = hl,
      priority = 200,  -- above SGR (default 0/100ish) so colors win
    })
  end
  for offset, line in ipairs(lines) do
    local row = lo + offset - 1
    for _, m in ipairs(find_markers_in_line(line)) do
      local p = m.parts
      emit(row, p.robot, 'PairRobotIcon')
      emit(row, p.lb,    'PairRobotBracket')
      emit(row, p.y,     'PairRobotComment')
      emit(row, p.rb,    'PairRobotBracket')
      if m.kind == 'scoped' then
        emit(row, p.lt, 'PairRobotBracket')
        emit(row, p.x,  'PairRobotSelection')
        emit(row, p.gt, 'PairRobotBracket')
      end
    end
  end
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
-- Stable identity for a marker, used to subtract the load-time baseline
-- from the set of markers present at emit time. NUL separator keeps
-- distinct (X, Y) pairs from colliding when one contains the other's
-- text.
local function marker_key(m)
  if m.kind == 'scoped' then
    return 's\0' .. m.X .. '\0' .. m.Y
  end
  return 'b\0' .. m.Y
end

-- Snapshot per-line marker counts. Counts (not sets) so a line with two
-- identical 🤖[foo] markers at load time doesn't silently absorb a new
-- 🤖[foo] added on the same line via Alt+q.
local function collect_markers_by_line(buf_lines)
  local by_line = {}
  for i, line in ipairs(buf_lines) do
    local markers = find_markers_in_line(line)
    if #markers > 0 then
      local set = {}
      for _, m in ipairs(markers) do
        local k = marker_key(m)
        set[k] = (set[k] or 0) + 1
      end
      by_line[i] = set
    end
  end
  return by_line
end

-- Walk `buf_lines`, return a markdown block of "> quote\nY" entries for
-- every 🤖[]-marker except those already present at load time. The
-- baseline (per-line marker-key counts from collect_markers_by_line) is
-- decremented as we walk, so a load-time marker absorbs exactly one
-- current-state marker with the same key on the same line.
local function format_extraction(buf_lines, baseline_by_line)
  baseline_by_line = baseline_by_line or {}
  local pieces = {}
  for i, line in ipairs(buf_lines) do
    local markers = find_markers_in_line(line)
    if #markers == 0 then goto next_line end
    local stripped = strip_markers(line, markers)
    -- Per-iteration mutable copy of the baseline counts for this line.
    local skip = {}
    for k, v in pairs(baseline_by_line[i] or {}) do skip[k] = v end
    for _, m in ipairs(markers) do
      -- Empty `[]` body = unfinished marker (Alt+q dropped the syntax
      -- and the user moved on without typing a comment). Drop it
      -- silently rather than ship a quote-only block to the draft.
      if m.Y:match('^%s*$') then goto continue end
      local k = marker_key(m)
      if (skip[k] or 0) > 0 then
        -- Pre-existing in the transcript itself (the captured agent
        -- pane already had 🤖[…] tokens, e.g. from a previous session's
        -- draft picked up by the agent). Skip — only emit markers the
        -- user typed during *this* scrollback viewing.
        skip[k] = skip[k] - 1
        goto continue
      end
      local quote = (m.kind == 'scoped') and m.X or stripped
      if quote == '' then
        -- Edge: bare marker on a line that's *only* the marker. Fall
        -- back to a placeholder so the pickup side knows there was a
        -- standalone note.
        quote = '(no context)'
      end
      table.insert(pieces, '> ' .. quote .. '\n' .. m.Y)
      ::continue::
    end
    ::next_line::
  end
  return table.concat(pieces, '\n\n')
end

-- Per-buffer snapshot of markers present at buffer load. Subtracted
-- from format_extraction so only markers the user added during this
-- viewing of the scrollback get shipped to the draft; any 🤖[…] tokens
-- baked into the captured transcript (from a prior session's draft
-- pickup) stay put.
local initial_markers_by_buf = {}

-- Truncate `s` to fit within `max_cols` of display width, appending an
-- ellipsis if cut. Used for the quote-context line in the prompt.
local function truncate_to_width(s, max_cols)
  if vim.fn.strdisplaywidth(s) <= max_cols then return s end
  -- vim.fn.strcharpart is char-based, not display-width-based, but is a
  -- close enough proxy for the kind of mixed-ASCII content we get from
  -- the agent pane. Underestimates for wide CJK, which just means the
  -- ellipsis lands a hair earlier — fine.
  return vim.fn.strcharpart(s, 0, max_cols - 1) .. '…'
end

-- Greedy word-wrap of `s` to `max_cols` display width per line. Returns
-- a list of lines (no trailing newlines). Single tokens that exceed
-- max_cols are character-sliced at the boundary — pathological case,
-- shouldn't show up in practice but better than overflowing the prompt
-- window. Whitespace runs collapse to single spaces by construction
-- (gmatch('%S+') skips them); for a context-preview line that's an
-- acceptable lossy display.
local function wrap_to_width(s, max_cols)
  if max_cols < 1 then max_cols = 1 end
  if vim.fn.strdisplaywidth(s) <= max_cols then return { s } end
  local lines = {}
  local current = ''
  for word in s:gmatch('%S+') do
    local candidate = (current == '') and word or (current .. ' ' .. word)
    if vim.fn.strdisplaywidth(candidate) <= max_cols then
      current = candidate
    else
      if current ~= '' then table.insert(lines, current) end
      current = word
      -- Force-break an over-long single token (rare).
      while vim.fn.strdisplaywidth(current) > max_cols do
        table.insert(lines, vim.fn.strcharpart(current, 0, max_cols))
        current = vim.fn.strcharpart(current, max_cols)
      end
    end
  end
  if current ~= '' then table.insert(lines, current) end
  return lines
end

-- Expose for tests; headless harness pokes these directly. Lives down
-- here (vs. immediately after format_extraction) so the table can
-- include the layout helpers — Lua's local scoping would otherwise
-- have them resolve to globals at the table-construction site.
_G.PairScrollbackTest = {
  find_markers_in_line     = find_markers_in_line,
  strip_markers            = strip_markers,
  format_extraction        = format_extraction,
  collect_markers_by_line  = collect_markers_by_line,
  marker_key               = marker_key,
  esc_x                    = esc_x,
  esc_y                    = esc_y,
  unescape                 = unescape,
  truncate_to_width        = truncate_to_width,
  wrap_to_width            = wrap_to_width,
}

-- Floating-window single-line prompt with a markdown-style quote header.
--   `quote`   : the context line to display as `> quote` above the input
--   `default` : initial text in the input field
--   `on_done(result)` : called with the user's input on Return, or nil
--                       on Esc/cancel. Empty-string result means "the
--                       user accepted with an empty input", distinct
--                       from cancel — callers map this to delete-marker.
-- Reason this isn't vim.ui.input / vim.fn.input: cmdline-based prompts
-- on macOS terminals mishandle Option+Delete by letting the ESC half
-- cancel the cmdline before nvim can fuse it with the trailing byte
-- into <M-BS>/<M-Del>, and the trailing byte then leaks into normal
-- mode where it can edit the underlying scrollback buffer. Owning the
-- buffer + window + keymaps cleanly side-steps that whole class of bug.
local function open_marker_prompt(quote, default, on_done)
  default = default or ''
  local quote_text = (quote == '' or quote == nil) and '(no context)' or quote
  -- Window width: ~80% of the editor width. Bounded below at 40 so
  -- it stays usable on narrow terminals, and above at `columns - 4`
  -- so the rounded border has visual breathing room from the edge.
  -- The width is fixed (not content-hugging) — earlier UX feedback
  -- was that the prior 50%-ish hug felt cramped for context preview.
  local width = math.max(40, math.min(math.floor(vim.o.columns * 0.8), vim.o.columns - 4))
  -- Quote lines wrap to fit the window: the buffer's text spans
  -- `width` cells (style='minimal' removes signcolumn/numbers, so
  -- the inner area is the full width); 2 of those cells go to the
  -- `> ` markdown-blockquote prefix.
  local max_inner = width
  -- Window height cap: up to 10 rows total. Reserve 2 for the blank
  -- separator + input line; the remaining 8 are available for wrapped
  -- quote context. Long quotes wrap; if they exceed the cap the last
  -- visible quote line gets an ellipsis. Input itself stays single-
  -- line (multi-line entry is an explicit non-feature).
  local MAX_WINDOW_ROWS = 10
  local max_quote_rows = MAX_WINDOW_ROWS - 2
  local quote_rows = wrap_to_width(quote_text, max_inner - 2)  -- 2 for "> "
  if #quote_rows > max_quote_rows then
    -- Replace the last shown row with a truncated version that hints
    -- there's more, instead of silently dropping context.
    quote_rows = { table.unpack(quote_rows, 1, max_quote_rows) }
    quote_rows[#quote_rows] = truncate_to_width(quote_rows[#quote_rows], max_inner - 2)
    if not quote_rows[#quote_rows]:match('…$') then
      quote_rows[#quote_rows] = quote_rows[#quote_rows] .. ' …'
    end
  end
  local lines = {}
  for _, r in ipairs(quote_rows) do
    table.insert(lines, '> ' .. r)
  end
  table.insert(lines, '')        -- separator
  table.insert(lines, default)   -- editable input
  local input_row = #lines       -- 1-based row of the editable line

  local buf = vim.api.nvim_create_buf(false, true)
  vim.bo[buf].buftype = 'nofile'
  vim.bo[buf].bufhidden = 'wipe'
  vim.bo[buf].swapfile = false
  vim.api.nvim_buf_set_lines(buf, 0, -1, false, lines)

  local height = #lines

  local win = vim.api.nvim_open_win(buf, true, {
    relative   = 'editor',
    row        = math.floor((vim.o.lines - height) / 2),
    col        = math.floor((vim.o.columns - width) / 2),
    width      = width,
    height     = height,
    style      = 'minimal',
    border     = 'rounded',
    title      = ' 🤖[] — Return to accept, Esc to cancel ',
    title_pos  = 'left',
  })

  -- Dim every quote line so the eye lands on the editable text below.
  -- All rows from 0 through input_row-2 (i.e. including the blank
  -- separator at input_row-2) get the muted highlight; the input line
  -- itself uses default Normal.
  local ns = vim.api.nvim_create_namespace('PairScrollbackPrompt')
  for i = 0, input_row - 2 do
    pcall(vim.api.nvim_buf_add_highlight, buf, ns, 'Comment', i, 0, -1)
  end

  -- Place cursor at end of input line and enter insert mode.
  pcall(vim.api.nvim_win_set_cursor, win, { input_row, 0 })
  vim.cmd('startinsert!')

  local finished = false
  local function finish(result)
    if finished then return end
    finished = true
    pcall(vim.cmd, 'stopinsert')
    if vim.api.nvim_win_is_valid(win) then
      pcall(vim.api.nvim_win_close, win, true)
    end
    on_done(result)
  end

  local function accept()
    local text = vim.api.nvim_buf_get_lines(buf, input_row - 1, input_row, false)[1] or ''
    finish(text)
  end
  local function cancel() finish(nil) end

  local opts = { buffer = buf, silent = true, nowait = true }
  vim.keymap.set({ 'i', 'n' }, '<CR>',  accept, opts)
  vim.keymap.set({ 'i', 'n' }, '<Esc>', cancel, opts)
  vim.keymap.set('n', 'q', cancel, opts)
  -- Option+Delete / Option+Backspace in insert mode → delete to line
  -- start, matching macOS Cocoa text-field convention. Buffer-local so
  -- it can't leak elsewhere. Two spellings because terminal emitters
  -- disagree on which keycode Option+Delete maps to.
  vim.keymap.set('i', '<M-BS>',  '<C-U>', opts)
  vim.keymap.set('i', '<M-Del>', '<C-U>', opts)

  -- Pin the cursor on the input line: clicks or arrow keys that would
  -- wander into the quote/blank get bounced back. Also pin the input
  -- line itself as the only editable surface — multi-line input is an
  -- explicit non-feature here.
  local function bounce()
    if not vim.api.nvim_win_is_valid(win) then return end
    local cur = vim.api.nvim_win_get_cursor(win)
    if cur[1] ~= input_row then
      vim.api.nvim_win_set_cursor(win, { input_row, cur[2] })
    end
  end
  vim.api.nvim_create_autocmd({ 'CursorMoved', 'CursorMovedI' }, {
    buffer = buf,
    callback = bounce,
  })
  -- Treat focus loss (user clicked another pane) as cancel so the
  -- prompt never gets orphaned.
  vim.api.nvim_create_autocmd('BufLeave', {
    buffer = buf,
    once = true,
    callback = function() finish(nil) end,
  })
end

-- Locate the marker (if any) whose byte range contains the cursor's
-- current 1-based byte column on the current line. Returns
-- (row, line, marker) on hit, nil otherwise.
local function marker_under_cursor(bufnr)
  local row, col0 = unpack(vim.api.nvim_win_get_cursor(0))  -- col0 is 0-indexed byte
  local col = col0 + 1
  local line = vim.api.nvim_buf_get_lines(bufnr, row - 1, row, false)[1] or ''
  for _, m in ipairs(find_markers_in_line(line)) do
    if col >= m.range[1] and col <= m.range[2] then
      return row, line, m
    end
  end
  return nil
end

-- Helper: write a single line back into the scrollback buffer, toggling
-- the read-only lock around the edit. Triggers marker re-highlighting.
local function rewrite_line(bufnr, row, new_line)
  vim.bo[bufnr].modifiable = true
  vim.bo[bufnr].readonly   = false
  vim.api.nvim_buf_set_lines(bufnr, row - 1, row, false, { new_line })
  vim.bo[bufnr].modifiable = false
  vim.bo[bufnr].readonly   = true
  highlight_markers(bufnr, row - 1, row)
end

-- Rewrite (or remove) an existing marker.
--   nil result → cancel, buffer untouched
--   ""         → delete the marker (scoped: restore X; bare: pure
--                removal with adjacent-space collapse)
--   == m.Y     → no-op
--   else       → replace Y in place, preserving kind + X
local function edit_marker(bufnr, row, line, m)
  local quote = (m.kind == 'scoped') and m.X
    or strip_markers(line, find_markers_in_line(line))
  open_marker_prompt(quote, m.Y, function(new_y)
    if new_y == nil then return end
    if new_y == m.Y then return end
    local new_line
    if new_y == '' then
      local before = line:sub(1, m.range[1] - 1)
      local after  = line:sub(m.range[2] + 1)
      if m.kind == 'scoped' then
        -- Scoped markers wrap a user-selected span (add_marker_visual
        -- replaced `sel` with `🤖<sel>[comment]`). Deleting the marker
        -- restores that span; the prose it was attached to stays.
        new_line = before .. m.X .. after
      else
        -- Bare marker: pure removal. Collapse one adjacent space so the
        -- surrounding prose doesn't end up with a double gap, and trim
        -- trailing whitespace for the end-of-line case.
        if before:match(' $') and after:match('^ ') then
          after = after:sub(2)
        end
        new_line = (before .. after):gsub('%s+$', '')
      end
    else
      local marker_text = (m.kind == 'scoped')
        and (MARKER_BOT .. '<' .. esc_x(m.X) .. '>[' .. esc_y(new_y) .. ']')
        or  (MARKER_BOT .. '['                       .. esc_y(new_y) .. ']')
      new_line = line:sub(1, m.range[1] - 1) .. marker_text .. line:sub(m.range[2] + 1)
    end
    rewrite_line(bufnr, row, new_line)
  end)
end

local function add_marker_normal(bufnr)
  -- Context-sensitive: cursor on an existing 🤖[…] or 🤖<…>[…] →
  -- offer to edit it in place; otherwise drop a new bare marker at
  -- end-of-line.
  local hit_row, hit_line, hit_marker = marker_under_cursor(bufnr)
  if hit_marker then
    edit_marker(bufnr, hit_row, hit_line, hit_marker)
    return
  end
  local row = vim.api.nvim_win_get_cursor(0)[1]
  local line = vim.api.nvim_buf_get_lines(bufnr, row - 1, row, false)[1] or ''
  local quote = strip_markers(line, find_markers_in_line(line))
  open_marker_prompt(quote, '', function(comment)
    if comment == nil or comment == '' then return end
    local marker = MARKER_BOT .. '[' .. esc_y(comment) .. ']'
    local sep = (line == '' or line:match('%s$')) and '' or ' '
    rewrite_line(bufnr, row, line .. sep .. marker)
  end)
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
  vim.cmd('normal! \27')  -- exit visual so the upcoming prompt has focus
  local line = vim.api.nvim_buf_get_lines(bufnr, sr - 1, sr, false)[1] or ''
  ec = math.min(ec, #line)  -- clamp for line-wise V (col is huge there)
  local before = line:sub(1, sc - 1)
  local sel    = line:sub(sc, ec)
  local after  = line:sub(ec + 1)
  if sel == '' then return end
  open_marker_prompt(sel, '', function(comment)
    if comment == nil or comment == '' then return end
    local marker = MARKER_BOT .. '<' .. esc_x(sel) .. '>[' .. esc_y(comment) .. ']'
    rewrite_line(bufnr, sr, before .. marker .. after)
  end)
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
  local block = format_extraction(lines, initial_markers_by_buf[bufnr])
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
    highlight_markers(bufnr)
    vim.bo[bufnr].modifiable = false
    vim.bo[bufnr].readonly = true
    vim.bo[bufnr].buftype = 'nofile'  -- prevent accidental :w
    vim.bo[bufnr].swapfile = false
    -- Buffer-local sentinel for VimLeavePre lookup. Picking via
    -- buftype='nofile' alone is fragile — any plugin spawning a scratch
    -- nofile buffer would shadow ours. Tag the one we own.
    vim.b[bufnr].pair_scrollback = true
    -- Snapshot 🤖[…] markers that came in with the transcript itself
    -- (an agent that echoed back a prior draft's markers, or a re-run
    -- where the previous session's sidecar landed in the input log).
    -- format_extraction subtracts these so only markers the user adds
    -- during this viewing get shipped to the draft.
    initial_markers_by_buf[bufnr] = collect_markers_by_line(
      vim.api.nvim_buf_get_lines(bufnr, 0, -1, false))
    -- ESC is the only quit binding. `q` was tempting (less-style, no
    -- Esc-prefix timeout) but a fat-fingered `q` instead of `Alt+q`
    -- (the marker-comment binding) was a frequent footgun — one
    -- mistype and the viewer slammed shut, dropping any pending
    -- markers along with the session. Built-in `ZZ` / `ZQ` are
    -- shadowed with no-ops for the same reason.
    --
    -- Confirm on ESC *only when* there's something to ship to the
    -- draft pane (i.e. user-added markers beyond the load-time
    -- baseline). A stray ESC during marker entry used to slam the
    -- viewer shut and felt like data loss; the prompt is the guard.
    -- For passive reads, there's nothing at stake — quit instantly,
    -- no friction.
    vim.keymap.set('n', '<Esc>', function()
      local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
      local block = format_extraction(lines, initial_markers_by_buf[bufnr])
      if block == '' then
        vim.cmd('qa')
        return
      end
      local n = 0
      for _ in block:gmatch('\n> ') do n = n + 1 end
      n = n + 1  -- first marker has no preceding "\n> "
      local prompt = string.format(
        'Exit scrollback? %d pending 🤖[] marker%s will be sent.',
        n, n == 1 and '' or 's')
      local choice = vim.fn.confirm(prompt, '&Yes\n&No', 1, 'Question')
      if choice == 1 then vim.cmd('qa') end
    end, { buffer = bufnr, silent = true })
    vim.keymap.set('n', 'ZZ', '<nop>', { buffer = bufnr, silent = true })
    vim.keymap.set('n', 'ZQ', '<nop>', { buffer = bufnr, silent = true })
    vim.keymap.set('n', '<M-q>', function() add_marker_normal(bufnr) end,
                   { buffer = bufnr, silent = true })
    vim.keymap.set('x', '<M-q>', function() add_marker_visual(bufnr) end,
                   { buffer = bufnr, silent = true })
    vim.keymap.set('n', '<M-b>', function() jump_to_prompt('prev') end,
                   { buffer = bufnr, silent = true })
    vim.keymap.set('n', '<M-B>', function() jump_to_prompt('next') end,
                   { buffer = bufnr, silent = true })
    -- Disable the Alt-arrow / Shift-Alt-arrow combos. Two distinct
    -- failure modes had to be neutralised:
    --   • <M-Up> / <M-Down>: nvim's defaults bind these to "move line
    --     up / down", which errors loudly on the scrollback's read-
    --     only buffer.
    --   • <M-Left> / <M-Right> (+ Shift variants): terminals emit
    --     them as `ESC <Arrow>` and the two bytes can split apart
    --     past ttimeoutlen — the ESC half then fires our exit binding
    --     and the viewer silently closes.
    -- Mapping all of them to <nop> at buffer scope takes the default
    -- handler out of the picture for the first class; bumping
    -- ttimeoutlen below maximises the chance nvim fuses ESC + Arrow
    -- into a single <M-Arrow> chord before our <Esc> handler sees a
    -- bare ESC.
    for _, key in ipairs({
      '<M-Up>', '<M-Down>', '<M-Left>', '<M-Right>',
      '<M-S-Up>', '<M-S-Down>', '<M-S-Left>', '<M-S-Right>',
    }) do
      vim.keymap.set({ 'n', 'x' }, key, '<nop>', { buffer = bufnr, silent = true })
    end
    vim.opt.ttimeoutlen = 100

    -- Keep the buffer's last line pinned at-or-above the bottom of the
    -- window, so j / Ctrl-E / Ctrl-D / G past EOF can't re-introduce
    -- phantom `~` rows that confuse the reader into thinking there's
    -- more content. The clamp runs on every cursor move or scroll;
    -- nvim has no built-in "stop scrolling at EOF" option, so this
    -- autocmd-driven retopline is the canonical pattern.
    local function clamp_no_phantom_rows()
      if not vim.api.nvim_win_is_valid(0) then return end
      local last = vim.api.nvim_buf_line_count(0)
      local winh = vim.api.nvim_win_get_height(0)
      local max_top = math.max(1, last - winh + 1)
      if vim.fn.line('w0') > max_top then
        local saved = vim.api.nvim_win_get_cursor(0)
        pcall(vim.api.nvim_win_set_cursor, 0, { max_top, 0 })
        vim.cmd('normal! zt')
        pcall(vim.api.nvim_win_set_cursor, 0, saved)
      end
    end
    vim.api.nvim_create_autocmd({ 'CursorMoved', 'WinScrolled' }, {
      buffer = bufnr,
      callback = clamp_no_phantom_rows,
    })

    -- Align the viewer's top with the agent pane's current viewport.
    -- The renderer writes a sibling `<ansi>.viewport` containing the
    -- 1-indexed line where the visible buffer (em.Height() rows at the
    -- bottom of the emulator) starts in the rendered output — i.e.
    -- the first line the user was just looking at in the agent pane.
    -- Position cursor there and `zt` to top so opening Alt+/ feels
    -- like a seamless lift of the current screen into a scrollable
    -- viewer.
    --
    -- Fallback: if the sidecar is missing or out of range, drop to
    -- the original "G + zb" bottom-anchored behaviour so a stale
    -- pair install with the new viewer but the old renderer still
    -- opens to something useful.
    --
    -- The positioning runs on `vim.schedule` rather than inline because
    -- a synchronous `zt` during BufReadPost can be undone by a window-
    -- size update that fires later in the same tick (the floating pane
    -- only finalises its layout after the file is loaded), leaving the
    -- viewer scrolled to wherever the layout-change put it instead of
    -- our requested top. Deferring one tick puts our positioning *after*
    -- those layout passes complete.
    local ansi_path = vim.api.nvim_buf_get_name(bufnr)
    local vp_path = ansi_path:match('%.ansi$')
      and (ansi_path:gsub('%.ansi$', '.viewport'))
      or nil
    vim.schedule(function()
      if not vim.api.nvim_buf_is_valid(bufnr) then return end
      local last = vim.api.nvim_buf_line_count(bufnr)
      local top
      if vp_path then
        local f = io.open(vp_path, 'r')
        if f then
          top = tonumber(f:read('*l'))
          f:close()
        end
      end
      if top and top >= 1 and top <= last then
        -- Clamp topline so we never show phantom `~` rows past EOF:
        -- if there's not enough content past `top` to fill the window
        -- (common when the scrollback viewer is taller than the agent
        -- pane), shift topline up to backfill with pre-viewport
        -- content. The cursor still lands at the agent's-viewport-top
        -- line — it's just no longer pinned to *row 1* of the window.
        local winh = vim.api.nvim_win_get_height(0)
        local max_top = math.max(1, last - winh + 1)
        local clamped_top = math.min(top, max_top)
        pcall(vim.api.nvim_win_set_cursor, 0, { clamped_top, 0 })
        vim.cmd('normal! zt')
        if clamped_top < top then
          pcall(vim.api.nvim_win_set_cursor, 0, { top, 0 })
        end
      elseif last > 0 then
        pcall(vim.api.nvim_win_set_cursor, 0, { last, 0 })
        vim.cmd('normal! zb')
      end
    end)
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
-- Route yank/delete to the system clipboard. The viewer launches with
-- `nvim -u nvim/scrollback.lua`, which bypasses the user's regular
-- init — so any clipboard config there doesn't apply here. Without
-- this, `y` lands in the unnamed register only and cmd+V outside the
-- terminal pulls whatever was on the system clipboard before. Neovim
-- autodetects pbcopy/pbpaste on macOS so no provider config needed.
vim.opt.clipboard = 'unnamedplus'
vim.opt.laststatus = 2
vim.opt.statusline = ' pair scrollback · Esc quit (confirm) · Alt+q 🤖[] (add/edit) · Alt+b/B prompts · :N jump %= L%l/%L '
