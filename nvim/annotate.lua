-- nvim/annotate.lua — the shared 🤖-marker annotation subsystem (#57).
--
-- Lifted out of nvim/scrollback.lua so both read-only pair viewers — the
-- scrollback viewer (Alt+/) and the change-log viewer (Alt+l) — share ONE
-- marker implementation instead of duplicating ~400 lines (ARCH-DRY).
--
-- Shape (ARCH-PURE):
--   * Pure core — find_markers_in_line / escape-unescape / strip_markers /
--     marker_key / collect_markers_by_line / format_extraction /
--     new_marker_count / width helpers. No buffer reads, no IO, no api mutation
--     (a couple call read-only vim.fn.strdisplaywidth, deterministic width
--     math). Exposed on M and unit-tested directly in annotate_test.lua.
--   * Thin IO/UI seam — M.attach{bufnr,pending_path,footer,source_label,quit_noun}
--     wires the Alt+q keymaps, the floating prompt, the read-only rewrite dance,
--     the VimLeavePre sidecar emit, and the quit-confirm gate, parameterized per
--     buffer.
--
-- Each viewer keeps what genuinely differs: scrollback owns SGR rendering +
-- Alt+b + the footer affordance config (footer=true); changelog owns markdown
-- colorize + the async distill refresh + the reload-vs-marker guard
-- (has_new_markers / on_reloaded). Loaded by both via a dir-relative `dofile`
-- (same pattern as adapt.lua) because each viewer launches with `nvim -u
-- <viewer>.lua` and may not have nvim/ on its runtimepath.

local M = {}
local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local marker_codec = dofile(here .. 'marker_codec.lua')

-- 🤖 = U+1F916, four bytes in UTF-8: F0 9F A4 96. Lua patterns aren't
-- UTF-8-aware so we use the literal byte sequence with `find(..., 1, true)`.
local MARKER_BOT = '\240\159\164\150'

-- ---------------------------------------------------------------------------
-- Pure core — marker parse / escape / extract. No IO, no buffer state.

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
local esc_x = marker_codec.esc_x
local esc_y = marker_codec.esc_y
local unescape = marker_codec.unescape

-- Find first occurrence of `char` in `line` starting at `start_pos`
-- that is NOT escaped (i.e., not preceded by an odd number of `\`s).
-- Returns nil if none. Used to locate the real `>` and `]` that
-- close a marker, ignoring escaped ones inside X / Y.
local find_unescaped = marker_codec.find_unescaped

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
--
-- opts.source_label (optional): when set, each quote line is prefixed
-- `> [<label>] <quote>` instead of `> <quote>`, so the agent can tell a
-- change-log question from a raw-scrollback one. Crucially this keeps
-- exactly ONE `> ` per marker, so the draft-pickup count
-- (init.lua's `content:gmatch('\n> ')`) stays a faithful marker count;
-- a standalone `> [change log]` header line would inflate it by one.
local function format_extraction(buf_lines, baseline_by_line, opts)
  opts = opts or {}
  local qprefix = opts.source_label and ('> [' .. opts.source_label .. '] ') or '> '
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
        -- Pre-existing in the transcript itself (the captured pane
        -- already had 🤖[…] tokens, e.g. from a previous session's
        -- draft picked up by the agent). Skip — only emit markers the
        -- user typed during *this* viewing.
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
      table.insert(pieces, qprefix .. quote .. '\n' .. m.Y)
      ::continue::
    end
    ::next_line::
  end
  return table.concat(pieces, '\n\n')
end

-- Count user-added (beyond-baseline), non-empty markers across buf_lines.
-- Same subtraction format_extraction does, but counts rather than formats —
-- so the confirm gate's "N marker(s) will be sent" stays independent of any
-- source_label prefix (counting `> ` in the formatted block would be fragile).
local function new_marker_count(buf_lines, baseline_by_line)
  baseline_by_line = baseline_by_line or {}
  local n = 0
  for i, line in ipairs(buf_lines) do
    local markers = find_markers_in_line(line)
    local skip = {}
    for k, v in pairs(baseline_by_line[i] or {}) do skip[k] = v end
    for _, m in ipairs(markers) do
      if not m.Y:match('^%s*$') then
        local k = marker_key(m)
        if (skip[k] or 0) > 0 then
          skip[k] = skip[k] - 1
        else
          n = n + 1
        end
      end
    end
  end
  return n
end

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

-- Expose the pure core for direct unit testing (ARCH-PURE boundary).
M.find_markers_in_line    = find_markers_in_line
M.strip_markers           = strip_markers
M.marker_key              = marker_key
M.collect_markers_by_line = collect_markers_by_line
M.format_extraction       = format_extraction
M.new_marker_count        = new_marker_count
M.esc_x                   = esc_x
M.esc_y                   = esc_y
M.unescape                = unescape
M.find_unescaped          = find_unescaped
M.truncate_to_width       = truncate_to_width
M.wrap_to_width           = wrap_to_width
M.MARKER_BOT              = MARKER_BOT

-- ---------------------------------------------------------------------------
-- Per-buffer state. Each viewer is its own `nvim` process, so this table
-- holds exactly one entry in practice — keyed by bufnr for robustness and to
-- match the shape of the old scrollback `*_by_buf` tables.
--   state[bufnr] = {
--     pending_path, footer (bool), source_label (string|nil), quit_noun,
--     baseline (collect_markers_by_line snapshot at attach),
--     footer_row (1-based|nil), footer_text (string|nil),
--   }
local state = {}

local FOOTER_HINT   = 'For overall comment, Alt+q on this line.'
local FOOTER_PREFIX = 'Overall comment: '

-- Toggle the read-only lock around a programmatic buffer edit.
local function set_modifiable(bufnr, on)
  vim.bo[bufnr].modifiable = on
  vim.bo[bufnr].readonly   = not on
end

-- ---------------------------------------------------------------------------
-- Marker highlighting.

-- 🤖[] marker syntax highlighting. Separate namespace from any viewer
-- rendering (scrollback's SGR extmarks, changelog's syntax match) so we can
-- re-render markers (after each Alt+q insertion) without rebuilding the
-- viewer's own decoration. `default = true` lets a colorscheme override.
vim.api.nvim_set_hl(0, 'PairRobotIcon',      { default = true, link = 'Special'    })
vim.api.nvim_set_hl(0, 'PairRobotBracket',   { default = true, link = 'Delimiter'  })
vim.api.nvim_set_hl(0, 'PairRobotSelection', { default = true, link = 'String'     })
vim.api.nvim_set_hl(0, 'PairRobotComment',   { default = true, link = 'Identifier', bold = true })

-- Re-render 🤖[] markers across the buffer (or a single line range).
-- Cheap enough to run on every Alt+q insertion since markers are
-- typically a handful per buffer; if that ever stops being true,
-- pass `lo, hi` to scope to the modified line.
local function highlight_markers(bufnr, lo, hi)
  local ns = vim.api.nvim_create_namespace('pair_annotate_markers')
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
M.highlight_markers = highlight_markers

-- ---------------------------------------------------------------------------
-- Floating prompt — the Alt+q comment editor.

-- Floating-window single-line prompt with a markdown-style quote header.
--   `quote`   : the context line to display as `> quote` above the input
--   `default` : initial text in the input field
--   `on_done(result)` : called with the user's input on Return or Esc
--                       (both accept). Empty-string result is the
--                       "clear and accept" path — callers map it to
--                       delete-marker (edit case) or no-op (new-marker
--                       case). `nil` is reserved for the BufLeave
--                       focus-loss path below, which is the only true
--                       cancel — Esc as cancel was a frequent footgun
--                       after long comments, so the only way to drop
--                       text is to clear it first (Option+BS / ⌥⌫).
-- Reason this isn't vim.ui.input / vim.fn.input: cmdline-based prompts
-- on macOS terminals mishandle Option+Delete by letting the ESC half
-- cancel the cmdline before nvim can fuse it with the trailing byte
-- into <M-BS>/<M-Del>, and the trailing byte then leaks into normal
-- mode where it can edit the underlying viewer buffer. Owning the
-- buffer + window + keymaps cleanly side-steps that whole class of bug.
local function open_marker_prompt(quote, default, on_done)
  default = default or ''
  local quote_text = (quote == '' or quote == nil) and '(no context)' or quote
  -- Window width: ~80% of the editor width. Bounded below at 40 so
  -- it stays usable on narrow terminals, and above at `columns - 4`
  -- so the rounded border has visual breathing room from the edge.
  local width = math.max(40, math.min(math.floor(vim.o.columns * 0.8), vim.o.columns - 4))
  local max_inner = width
  -- Window height cap: up to 10 rows total. Reserve 2 for the blank
  -- separator + input line; the remaining 8 are available for wrapped
  -- quote context. Long quotes wrap; if they exceed the cap the last
  -- visible quote line gets an ellipsis. Input itself stays single-line.
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
    title      = ' 🤖[] — Return/Esc: accept · Alt+Return: accept & ship to draft ',
    title_pos  = 'left',
  })

  -- Wrap long input at whitespace instead of mid-word.
  vim.wo[win].wrap = true
  vim.wo[win].linebreak = true

  -- Dim every quote line so the eye lands on the editable text below.
  local ns = vim.api.nvim_create_namespace('PairAnnotatePrompt')
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

  -- Esc accepts (same as Return), rather than cancelling. A long comment
  -- typed and then fat-fingered into Esc was being silently discarded —
  -- much higher pain than the inverse mistake. To explicitly cancel/delete,
  -- clear the input first (Option+BS / Option+Del → `<C-U>`) then Return:
  -- empty result is the delete-marker / no-op path for either marker kind.
  local opts = { buffer = buf, silent = true, nowait = true }
  vim.keymap.set({ 'i', 'n' }, '<CR>',  accept, opts)
  vim.keymap.set({ 'i', 'n' }, '<Esc>', accept, opts)
  vim.keymap.set('n', 'q', accept, opts)
  -- Alt+Return: accept this comment AND immediately quit the viewer so
  -- the pending markers ship to the draft pane via the VimLeavePre →
  -- sidecar pickup path. Mirrors the draft pane's <M-CR> = send-to-agent
  -- muscle memory ("I'm done, push it"). Skips the viewer-level exit
  -- confirm — the user explicitly chose the ship gesture. Bound here (not
  -- just at the viewer level) because without an explicit binding terminals
  -- split Alt+CR into ESC+CR past ttimeoutlen, then the BufLeave-on-prompt
  -- focus-loss cancel kicks in and the typed comment is silently dropped.
  vim.keymap.set({ 'i', 'n' }, '<M-CR>', function()
    accept()
    vim.cmd('qa')
  end, opts)
  -- Option+Delete / Option+Backspace in insert mode → delete to line
  -- start, matching macOS Cocoa text-field convention. Buffer-local so
  -- it can't leak elsewhere. Two spellings because terminal emitters
  -- disagree on which keycode Option+Delete maps to.
  vim.keymap.set('i', '<M-BS>',  '<C-U>', opts)
  vim.keymap.set('i', '<M-Del>', '<C-U>', opts)

  -- Pin the cursor on the input line: clicks or arrow keys that would
  -- wander into the quote/blank get bounced back.
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
M.open_marker_prompt = open_marker_prompt

-- ---------------------------------------------------------------------------
-- Marker add / edit — the read-only unlock→insert→relock dance.

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

-- Helper: write a single line back into the viewer buffer, toggling
-- the read-only lock around the edit. Triggers marker re-highlighting.
local function rewrite_line(bufnr, row, new_line)
  set_modifiable(bufnr, true)
  vim.api.nvim_buf_set_lines(bufnr, row - 1, row, false, { new_line })
  set_modifiable(bufnr, false)
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

-- ---------------------------------------------------------------------------
-- Footer overall-comment affordance (#000021) — scrollback only (footer=true).
-- A single trailing buffer row gives the user a natural place to drop a summary
-- comment after annotating with Alt+q (nvim's virt_lines aren't cursor-
-- navigable, so Alt+q couldn't fire on a virtual line). Gated entirely by
-- state[bufnr].footer_row being non-nil; when footer=false every check below is
-- a no-op (number == nil → false).

-- Rewrite the affordance line to reflect the stored footer comment:
-- empty/nil → hint text, non-empty → "Overall comment: <text>".
local function update_footer_line(bufnr)
  local st = state[bufnr]
  local row = st and st.footer_row
  if not row then return end
  local text = st.footer_text
  local new_line = (text and text ~= '') and (FOOTER_PREFIX .. text) or FOOTER_HINT
  set_modifiable(bufnr, true)
  vim.api.nvim_buf_set_lines(bufnr, row - 1, row, false, { new_line })
  set_modifiable(bufnr, false)
end

-- Open the marker prompt for the overall comment. No quote context — it's a
-- standalone footer, not tied to a line. Empty submit clears the comment;
-- cancel (nil) leaves state untouched.
local function add_footer_comment(bufnr)
  local st = state[bufnr]
  local current = (st and st.footer_text) or ''
  open_marker_prompt('(overall comment for this ' .. (st and st.quit_noun or 'viewer') .. ')', current, function(new_text)
    if new_text == nil then return end
    st.footer_text = (new_text ~= '') and new_text or nil
    update_footer_line(bufnr)
  end)
end

local function add_marker_normal(bufnr)
  local st = state[bufnr]
  local row = vim.api.nvim_win_get_cursor(0)[1]
  -- Footer affordance row: route to the overall-comment flow rather
  -- than treating the hint text as transcript to annotate.
  if st and row == st.footer_row then
    add_footer_comment(bufnr)
    return
  end
  -- Context-sensitive: cursor on an existing 🤖[…] or 🤖<…>[…] →
  -- offer to edit it in place; otherwise drop a new bare marker at
  -- end-of-line.
  local hit_row, hit_line, hit_marker = marker_under_cursor(bufnr)
  if hit_marker then
    edit_marker(bufnr, hit_row, hit_line, hit_marker)
    return
  end
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
  local st = state[bufnr]
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
  if st and sr == st.footer_row then
    vim.notify('🤖 marker: use Alt+q on this line for the overall comment',
               vim.log.levels.INFO)
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

-- ---------------------------------------------------------------------------
-- Pending-sidecar emit + the draft-pickup path.

-- Default sidecar path the draft nvim picks up on FocusGained. Both viewers
-- feed this to attach() as pending_path (the same file init.lua reads), so
-- the resolver lives once here rather than copied per viewer (ARCH-DRY).
function M.default_pending_path()
  local data_dir = vim.env.PAIR_DATA_DIR
    or ((vim.env.XDG_DATA_HOME or (vim.env.HOME .. '/.local/share')) .. '/pair')
  local tag = vim.env.PAIR_TAG or vim.env.PAIR_AGENT or 'claude'
  return data_dir .. '/scrollback-pending-' .. tag .. '.md'
end

-- Extract the buffer's user-added markers (+ footer comment when footer=true)
-- and write the markdown block to the buffer's pending_path via tmp+rename.
-- One public entry point: the VimLeavePre autocmd and the headless tests both
-- call this.
function M.emit(bufnr)
  local st = state[bufnr]
  if not st then return end
  local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
  -- Drop the footer affordance row from the marker scan; the stored
  -- overall comment (if any) is appended to the block below.
  if st.footer_row then table.remove(lines, st.footer_row) end
  local block = format_extraction(lines, st.baseline, { source_label = st.source_label })
  local footer = st.footer_text
  if footer and footer ~= '' then
    block = (block == '') and footer or (block .. '\n\n' .. footer)
  end
  if block == '' then return end  -- silent no-op when nothing to ship
  local path = st.pending_path
  if not path then return end
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

-- ---------------------------------------------------------------------------
-- Reload guard. Markers are buffer TEXT; a viewer reload that replaces all
-- lines would wipe markers added after attach. has_pending_annotations lets a
-- viewer skip a reload when annotations are present; on_reloaded realigns the
-- baseline after a reload that DID run, so the baseline tracks the reloaded
-- content. For footer-enabled viewers, on_reloaded also recreates the trailing
-- footer affordance row that a full-buffer reload removes.

function M.has_new_markers(bufnr)
  local st = state[bufnr]
  if not st then return false end
  local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
  if st.footer_row and st.footer_row <= #lines then table.remove(lines, st.footer_row) end
  return new_marker_count(lines, st.baseline) > 0
end

function M.has_pending_annotations(bufnr)
  local st = state[bufnr]
  if not st then return false end
  return M.has_new_markers(bufnr) or (st.footer_text and st.footer_text ~= '') or false
end

function M.on_reloaded(bufnr)
  local st = state[bufnr]
  if not st then return end
  st.footer_row = nil
  st.baseline = collect_markers_by_line(vim.api.nvim_buf_get_lines(bufnr, 0, -1, false))
  if st.footer then
    local row0 = vim.api.nvim_buf_line_count(bufnr)
    local footer_line = (st.footer_text and st.footer_text ~= '')
      and (FOOTER_PREFIX .. st.footer_text) or FOOTER_HINT
    set_modifiable(bufnr, true)
    vim.api.nvim_buf_set_lines(bufnr, row0, row0, false, { footer_line })
    set_modifiable(bufnr, false)
    st.footer_row = row0 + 1
  end
  highlight_markers(bufnr)
end

-- ---------------------------------------------------------------------------
-- Quit-confirm gate — bound by each viewer to its chosen quit key(s).
-- Confirm only when there's something to ship (user-added markers beyond the
-- baseline, or a footer comment). Passive reads quit instantly.

function M.confirm_quit(bufnr)
  local st = state[bufnr]
  local lines = vim.api.nvim_buf_get_lines(bufnr, 0, -1, false)
  if st and st.footer_row then table.remove(lines, st.footer_row) end
  local n = st and new_marker_count(lines, st.baseline) or 0
  local has_footer = st and st.footer and st.footer_text and st.footer_text ~= ''
  if n == 0 and not has_footer then
    vim.cmd('qa')
    return
  end
  local parts = {}
  if n > 0 then
    table.insert(parts, string.format('%d 🤖[] marker%s', n, n == 1 and '' or 's'))
  end
  if has_footer then table.insert(parts, 'overall comment') end
  local noun = (st and st.quit_noun) or 'viewer'
  local prompt = 'Exit ' .. noun .. '? ' .. table.concat(parts, ' + ') .. ' will be sent.'
  if vim.fn.confirm(prompt, '&Yes\n&No', 1, 'Question') == 1 then
    vim.cmd('qa')
  end
end

-- ---------------------------------------------------------------------------
-- attach — the one entry point a viewer calls after it has loaded and
-- read-only-set-up its buffer.
--   opts.bufnr        : the viewer buffer
--   opts.pending_path : sidecar to emit to (default: M.default_pending_path())
--   opts.footer       : bool — append the overall-comment affordance line
--   opts.source_label : string|nil — tag emitted quotes (`> [label] quote`)
--   opts.quit_noun    : string — the noun in "Exit <noun>?" (default 'viewer')
-- Snapshots the load-time baseline (so only user-added markers ship),
-- highlights existing markers, appends the footer line iff footer=true, sets
-- the Alt+q keymaps, and tags the buffer for VimLeavePre. Does NOT bind quit
-- keys — each viewer binds its own (scrollback omits `q`; changelog keeps it)
-- to M.confirm_quit.
function M.attach(opts)
  local b = opts.bufnr
  local lines = vim.api.nvim_buf_get_lines(b, 0, -1, false)
  state[b] = {
    pending_path = opts.pending_path or M.default_pending_path(),
    footer       = opts.footer or false,
    source_label = opts.source_label,
    quit_noun    = opts.quit_noun or 'viewer',
    baseline     = collect_markers_by_line(lines),  -- snapshot BEFORE footer line
    footer_text  = nil,
  }
  highlight_markers(b)
  if state[b].footer then
    -- Append the affordance line at end-of-buffer. Must run while modifiable.
    local row0 = vim.api.nvim_buf_line_count(b)
    set_modifiable(b, true)
    vim.api.nvim_buf_set_lines(b, row0, row0, false, { FOOTER_HINT })
    set_modifiable(b, false)
    state[b].footer_row = row0 + 1  -- 1-based
  end
  vim.keymap.set('n', '<M-q>', function() add_marker_normal(b) end,
                 { buffer = b, silent = true })
  vim.keymap.set('x', '<M-q>', function() add_marker_visual(b) end,
                 { buffer = b, silent = true })
  -- Buffer-local sentinel for the VimLeavePre lookup — robust against any
  -- other nofile buffers (terminals, plugin scratches) that might exist.
  vim.b[b].pair_annotate = true
end

-- Emit every attached buffer's pending markers on quit. Registered once at
-- module load; fires for any quit path (including `qa` / `qa!`). Emits for
-- EVERY attached buffer (no early return) — in the real single-buffer-per-
-- process viewer that's exactly one buffer; if a process ever attached two,
-- each would correctly write its own pending_path.
vim.api.nvim_create_autocmd('VimLeavePre', {
  callback = function()
    for _, b in ipairs(vim.api.nvim_list_bufs()) do
      if vim.api.nvim_buf_is_loaded(b) and vim.b[b].pair_annotate then
        M.emit(b)
      end
    end
  end,
})

return M
