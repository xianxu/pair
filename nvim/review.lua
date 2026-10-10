-- nvim/review.lua — init for the review pane (issue #66 M3). Loaded via
-- `nvim -u $PAIR_HOME/nvim/review.lua <file>` (from bin/pair-review-open).
-- Self-contained (no rtp), mirroring scrollback.lua/changelog.lua: dofiles the
-- review core + the agent poke + the marker highlighter, starts a docflow review
-- on the file, wires the review-view keymaps (Alt+Return = finish human turn),
-- renders 🤖 markers, and tears everything down on exit.
vim.opt.compatible = false
vim.opt.termguicolors = true
vim.opt.wrap = true
vim.opt.linebreak = true
vim.opt.fixendofline = false
vim.opt.breakindent = true
vim.opt.smoothscroll = true
vim.opt.clipboard = 'unnamedplus'
vim.opt.guicursor = 'n-v-c-sm:block-blinkon250-blinkoff250,i-ci-ve:block-blinkon250-blinkoff250,r-cr-o:hor20'
vim.opt.cmdheight = 0
-- The review pane is an EDITABLE document workbench, not a read-only viewer (the
-- scrollback look these were copied from) — show the gutter: absolute line numbers
-- + a stable sign column for the review diagnostics.
vim.opt.number = true
vim.opt.signcolumn = 'yes'
-- Smartcase search (#101): `/foo` matches case-insensitively, but a query with any
-- uppercase (`/Foo`) stays case-sensitive. Pane-local — this self-contained
-- `nvim -u` init never touches the draft nvim or the user's own config.
vim.opt.ignorecase = true
vim.opt.smartcase = true

local restoration
local REVIEW_DIAG_NS = vim.api.nvim_create_namespace('review_diag')
local ACTIVE_DIAG_NS = vim.api.nvim_create_namespace('review_active_diag')

local function cursor_over_diagnostic(d, row, col)
  if not row or not col then
    local ok, cur = pcall(vim.api.nvim_win_get_cursor, 0)
    if not ok then return true end
    row, col = cur[1] - 1, cur[2]
  end
  local first, last = d.lnum, d.end_lnum or d.lnum
  if row < first or row > last then return false end
  if row == first and d.col and col < d.col then return false end
  if row == last and d.end_col then
    if d.col and first == last and d.end_col == d.col then
      return col == d.col
    end
    if col >= d.end_col then return false end
  end
  return true
end

local function diagnostic_span_len(d)
  if not d then return math.huge end
  local first, last = d.lnum or 0, d.end_lnum or d.lnum or 0
  return ((last - first) * 1000000) + ((d.end_col or d.col or 0) - (d.col or 0))
end

local function active_diagnostic(buf)
  buf = buf or vim.api.nvim_get_current_buf()
  local ok, cur = pcall(vim.api.nvim_win_get_cursor, 0)
  if not ok then return nil end
  local row, col = cur[1] - 1, cur[2]
  local pick
  for _, d in ipairs(vim.diagnostic.get(buf, { namespace = REVIEW_DIAG_NS })) do
    if cursor_over_diagnostic(d, row, col) then
      if not pick or diagnostic_span_len(d) < diagnostic_span_len(pick) then
        pick = d
      end
    end
  end
  return pick
end

local function message_virt_lines(message)
  local lines = {}
  for _, line in ipairs(vim.split(message or '', '\n', { plain = true })) do
    lines[#lines + 1] = { { line, 'DiagnosticVirtualTextInfo' } }
  end
  return lines
end

local function render_active_diagnostic(buf)
  buf = buf or vim.api.nvim_get_current_buf()
  if not vim.api.nvim_buf_is_valid(buf) then return false end
  vim.api.nvim_buf_clear_namespace(buf, ACTIVE_DIAG_NS, 0, -1)
  local d = active_diagnostic(buf)
  if not d or not d.message or d.message == '' then return false end
  local line = math.max(0, math.min(d.end_lnum or d.lnum, vim.api.nvim_buf_line_count(buf) - 1))
  vim.api.nvim_buf_set_extmark(buf, ACTIVE_DIAG_NS, line, 0, {
    virt_lines = message_virt_lines(d.message),
    virt_lines_above = false,
  })
  return true
end

local function open_review_diagnostic_float()
  local d = active_diagnostic(vim.api.nvim_get_current_buf())
  render_active_diagnostic(vim.api.nvim_get_current_buf())
  local buf, win
  if not d or not d.message or d.message == '' then
    buf, win = vim.diagnostic.open_float(nil, { scope = 'cursor', focus = false })
  else
    buf, win = vim.lsp.util.open_floating_preview(vim.split(d.message, '\n', { plain = true }), 'markdown', {
      border = 'rounded', focus = false,
      close_events = { 'CursorMoved', 'CursorMovedI', 'BufHidden', 'InsertCharPre', 'WinLeave' },
    })
  end
  if buf and win then
    vim.keymap.set('n', '<Esc>', function()
      if vim.api.nvim_win_is_valid(win) then vim.api.nvim_win_close(win, true) end
    end, { buffer = buf, silent = true, desc = 'review: close diagnostic float' })
  end
  return buf, win
end

-- Diagnosis display — review/apply.lua sets each record's `explain` as an INFO
-- diagnostic (the "why" behind an edit). Keep diagnostic signs/underlines in
-- Neovim's native layer, but render the expanded virtual-line text ourselves so
-- same-line edits do not produce combined or blank built-in virtual lines.
if not pcall(vim.diagnostic.config, {
  virtual_lines = false,
  virtual_text = false,
  signs = true,
  underline = true,
  severity_sort = true,
}) then
  vim.diagnostic.config({ virtual_text = true, signs = true })
end

-- Diagnostic navigation (M4a issue 1): the user tried `<C-w>d` (→ E388, vim's
-- define-search) and `]d` (jumped without showing the why). Bind floats
-- explicitly; `<C-w>d`/`gl` use the active cursor span, while `]d`/`[d` retain
-- native diagnostic navigation.
vim.keymap.set('n', '<C-w>d', open_review_diagnostic_float, { desc = 'review: diagnostic float' })
vim.keymap.set('n', 'gl', open_review_diagnostic_float, { desc = 'review: diagnostic float' })
local function diag_jump(count)
  if vim.diagnostic.jump then
    vim.diagnostic.jump({ count = count, float = true }) -- nvim 0.11 API
  else
    (count > 0 and vim.diagnostic.goto_next or vim.diagnostic.goto_prev)({ float = true })
  end
end
vim.keymap.set('n', ']d', function() diag_jump(1) end, { desc = 'review: next diagnostic (float)' })
vim.keymap.set('n', '[d', function() diag_jump(-1) end, { desc = 'review: prev diagnostic (float)' })
vim.api.nvim_create_autocmd({ 'CursorMoved', 'CursorMovedI' }, {
  callback = function(args)
    pcall(render_active_diagnostic, args.buf)
  end,
})

-- Markdown appearance — mirrors the user's nvim where possible, while still
-- working as a standalone `nvim -u pair/nvim/review.lua` process. The normal
-- config loads moonfly through lazy.nvim, which this minimal init does not run,
-- so add common lazy theme paths to rtp and prefer the local theme before
-- falling back to bundled colors.
vim.cmd('syntax enable')
vim.cmd('filetype plugin indent on')
vim.g.markdown_fenced_languages = {
  'python', 'py=python', 'javascript', 'js=javascript', 'typescript', 'ts=typescript',
  'lua', 'bash', 'sh=bash', 'zsh=bash', 'json', 'yaml', 'yml=yaml', 'toml',
  'html', 'css', 'c', 'cpp', 'cxx=cpp', 'rust', 'rs=rust', 'go', 'sql',
  'ruby', 'rb=ruby', 'java', 'kotlin', 'kt=kotlin', 'swift', 'dockerfile', 'make', 'diff', 'vim',
}

local function disable_review_markdown_html_syntax()
  if vim.bo.filetype ~= 'markdown' and vim.bo.syntax ~= 'markdown' then return end
  -- Vim's regex markdown embeds HTML. In the review pane, 🤖<old>{new}
  -- markers are first-class review syntax; parsing <old> as an HTML tag can
  -- leak htmlTag/htmlString highlighting into following prose.
  pcall(vim.cmd, 'syntax clear htmlTag htmlEndTag htmlTagN htmlTagName htmlSpecialTagName htmlTagError')
  pcall(vim.cmd, 'syntax clear htmlString htmlValue htmlArg htmlEvent htmlCssDefinition')
  pcall(vim.cmd, 'syntax sync fromstart')
end

vim.api.nvim_create_autocmd({ 'FileType', 'Syntax' }, {
  pattern = 'markdown',
  callback = disable_review_markdown_html_syntax,
})

local function add_local_theme_rtp()
  local lazy = vim.fn.stdpath('data') .. '/lazy'
  for _, dir in ipairs({ 'moonfly', 'catppuccin', 'gruvbox.nvim', 'rose-pine', 'everforest', 'melange-nvim' }) do
    local path = lazy .. '/' .. dir
    if vim.fn.isdirectory(path) == 1 then
      vim.opt.runtimepath:append(path)
    end
  end
end

local function apply_review_colorscheme()
  add_local_theme_rtp()
  local names = {}
  if vim.env.PAIR_REVIEW_COLORSCHEME and vim.env.PAIR_REVIEW_COLORSCHEME ~= '' then
    names[#names + 1] = vim.env.PAIR_REVIEW_COLORSCHEME
  end
  vim.list_extend(names, { 'moonfly', 'catppuccin', 'slate', 'default' })
  for _, name in ipairs(names) do
    if pcall(vim.cmd.colorscheme, name) then
      return name
    end
  end
  return vim.g.colors_name or 'default'
end

apply_review_colorscheme()

-- The pane owns buffer projection. Activation reconciles disk bytes explicitly;
-- background focus/checktime must not reload text from another checkout or scan
-- full Git history on the editor thread.
vim.opt.autoread = false

do
  local here = debug.getinfo(1, 'S').source:sub(2):match('(.*/)') or './'
  local workbench_route = dofile(here .. 'workbench_route.lua')
  workbench_route.install_global_maps(false)
end

local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local review = dofile(here .. 'review/init.lua')
local poke = dofile(here .. 'pair_poke.lua')
local workbench_route = dofile(here .. 'workbench_route.lua')
local markers = dofile(here .. 'review/markers.lua')
local comments = dofile(here .. 'review/comment.lua')
local seam = dofile(here .. 'review/seam.lua')
local poke_bodies = dofile(here .. 'review/poke_bodies.lua')
local resolve = dofile(here .. 'review/resolve.lua')
local mode = dofile(here .. 'review/mode.lua')
local menu = dofile(here .. 'review/menu.lua')
local spinner = dofile(here .. 'review/spinner.lua')
local define = dofile(here .. 'review/define.lua')
local definition_seam = dofile(here .. 'review/definition_seam.lua')
local recovery = dofile(here .. 'review/recovery.lua')
local document_bytes = dofile(here .. 'review/document_bytes.lua')
local identity = dofile(here .. 'review/identity.lua')
local restore_policy = dofile(here .. 'review/restore.lua')

-- 🤖 marker highlight groups. Keep these aligned with parley.nvim's review mode
-- so pair and parley render the marker language consistently across themes.
local function setup_review_marker_hl()
  vim.api.nvim_set_hl(0, 'ParleyReviewUser', { link = 'DiagnosticWarn' })
  vim.api.nvim_set_hl(0, 'ParleyReviewAgent', { link = 'DiagnosticInfo' })
  vim.api.nvim_set_hl(0, 'ParleyReviewQuoted', { reverse = true, bold = true })
  vim.api.nvim_set_hl(0, 'ParleyReviewStrike', { strikethrough = true })
  vim.api.nvim_set_hl(0, 'ParleyReviewBroken', { link = 'DiagnosticError' })
  vim.api.nvim_set_hl(0, 'ParleyCommentUser', { link = 'DiffChange' })
  vim.api.nvim_set_hl(0, 'ParleyCommentAgent', { link = 'DiffAdd' })
end
setup_review_marker_hl()
vim.api.nvim_create_autocmd('ColorScheme', { callback = setup_review_marker_hl })

local function render_markers(buf) comments.render(buf) end

local REVIEW_MARKER = '🤖'

local function split_replacement(text)
  if text == '' then return {} end
  return vim.split(text, '\n', { plain = true })
end

local function marker_end_pos(marker)
  local raw_lines = vim.split(marker.raw, '\n', { plain = true })
  local end_row = marker.line + #raw_lines - 1
  local end_col = (#raw_lines == 1) and (marker.col + #marker.raw) or #raw_lines[#raw_lines]
  return end_row, end_col
end

local function apply_marker_resolution(buf, marker, action)
  local replacement = resolve.resolve(marker, action)
  local end_row, end_col = marker_end_pos(marker)
  vim.api.nvim_buf_set_text(buf, marker.line, marker.col, end_row, end_col,
    split_replacement(replacement))
end

-- Accept/reject the 🤖 marker on the cursor's line (M4b, parley §5 via resolve.lua):
-- turn the `🤖…{…}` chain into plain text. A normal buffer edit — undo-able, and
-- committed by the agent on the next human round (Alt+Return).
local function resolve_at_cursor(buf, action)
  local win = vim.api.nvim_get_current_win()
  local cur = vim.api.nvim_win_get_cursor(win) -- {row 1-based, col 0-based byte}
  local row0 = cur[1] - 1
  local lines = vim.api.nvim_buf_get_lines(buf, 0, -1, false)
  local target
  for _, m in ipairs(markers.parse_markers(lines)) do
    local end_row, end_col = marker_end_pos(m)
    if row0 >= m.line and row0 <= end_row then -- cursor on any line the marker spans (#89 M1)
      if not target then target = m end -- first marker overlapping this line
      -- precise containment across the (possibly multi-line) marker span
      local after_start = row0 > m.line or cur[2] >= m.col
      local before_end = row0 < end_row or cur[2] < end_col
      if after_start and before_end then target = m; break end
    end
  end
  if not target then
    if action == 'accept' and review.clear_decoration_at_line(buf, row0) then
      return
    end
    vim.notify('review: no 🤖 marker on this line', vim.log.levels.INFO); return
  end
  apply_marker_resolution(buf, target, action)
  render_markers(buf)
end

local function paragraph_bounds(lines, row0)
  local first = row0
  while first > 0 and lines[first] ~= '' do first = first - 1 end
  local last = row0
  while last < (#lines - 1) and lines[last + 2] ~= '' do last = last + 1 end
  return first, last
end

local function resolve_paragraph_to_cursor(buf, action)
  local win = vim.api.nvim_get_current_win()
  local cur = vim.api.nvim_win_get_cursor(win)
  local row0, col = cur[1] - 1, cur[2]
  local lines = vim.api.nvim_buf_get_lines(buf, 0, -1, false)
  local first, last = paragraph_bounds(lines, row0)
  local targets = {}
  for _, m in ipairs(markers.parse_markers(lines)) do
    if m.line >= first and m.line <= last and (m.line < row0 or (m.line == row0 and m.col <= col)) then
      targets[#targets + 1] = m
    end
  end
  if #targets == 0 then
    vim.notify('review: no 🤖 markers before cursor in this paragraph', vim.log.levels.INFO)
    return
  end
  for i = #targets, 1, -1 do
    apply_marker_resolution(buf, targets[i], action)
  end
  render_markers(buf)
end

local function insert_review_marker(buf)
  local win = vim.api.nvim_get_current_win()
  local cur = vim.api.nvim_win_get_cursor(win)
  local row0, col = cur[1] - 1, cur[2]
  local marker = REVIEW_MARKER .. '[]'
  vim.api.nvim_buf_set_text(buf, row0, col, row0, col, { marker })
  vim.api.nvim_win_set_cursor(win, { cur[1], col + #(REVIEW_MARKER .. '[') })
  render_markers(buf)
end

-- Replace the byte range [start_pos, end_pos) with 🤖<selected>[] and put the
-- cursor inside the human-comment brackets. start_pos/end_pos are {row1, col0}.
local function quote_selection(buf, start_pos, end_pos)
  local srow, scol = start_pos[1] - 1, start_pos[2]
  local erow, ecol = end_pos[1] - 1, end_pos[2]
  if erow < srow or (erow == srow and ecol < scol) then
    srow, erow, scol, ecol = erow, srow, ecol, scol
  end
  local selected = table.concat(vim.api.nvim_buf_get_text(buf, srow, scol, erow, ecol, {}), '\n')
  local marker = REVIEW_MARKER .. '<' .. markers.esc_quote(selected) .. '>[]'
  vim.api.nvim_buf_set_text(buf, srow, scol, erow, ecol, vim.split(marker, '\n', { plain = true }))
  local marker_lines = vim.split(marker, '\n', { plain = true })
  local cursor_row = srow + #marker_lines
  local cursor_col = (#marker_lines == 1) and (scol + #marker - 1) or (#marker_lines[#marker_lines] - 1)
  vim.api.nvim_win_set_cursor(vim.api.nvim_get_current_win(), { cursor_row, cursor_col })
  render_markers(buf)
end

local function quote_visual_selection(buf)
  local a = vim.fn.getpos("'<")
  local b = vim.fn.getpos("'>")
  -- Visual marks are inclusive and 1-based byte columns.
  quote_selection(buf, { a[2], math.max(a[3] - 1, 0) }, { b[2], b[3] })
  vim.cmd('startinsert')
end

-- Jump to the next/prev 🤖 marker (dir = 1 | -1), wrapping. Lets the human move
-- between pending suggestions to accept/reject them.
local function jump_marker(buf, dir)
  local win = vim.api.nvim_get_current_win()
  local row0 = vim.api.nvim_win_get_cursor(win)[1] - 1
  local marks = markers.parse_markers(vim.api.nvim_buf_get_lines(buf, 0, -1, false))
  if #marks == 0 then vim.notify('review: no 🤖 markers', vim.log.levels.INFO); return end
  local pick
  if dir > 0 then
    for _, m in ipairs(marks) do if m.line > row0 then pick = m; break end end
    pick = pick or marks[1] -- wrap
  else
    for i = #marks, 1, -1 do if marks[i].line < row0 then pick = marks[i]; break end end
    pick = pick or marks[#marks] -- wrap
  end
  vim.api.nvim_win_set_cursor(win, { pick.line + 1, pick.col })
end

-- The review state file: the draft's PairReviewToggle reads it (pid → liveness)
-- to know a review is open (file-select vs. visibility-toggle branch). Path comes
-- from the shared seam module so the writer here and the reader in init.lua agree.
local function state_file()
  return seam.open_state(vim.env.PAIR_REVIEW_OPEN_PATH)
end

local awaiting_since, spinner_tick
local status_timer_interval = 100

local function current_mode()
  return seam.read_mode(vim.env.PAIR_REVIEW_MODE_PATH)
end

local function mode_label()
  return seam.mode_label(current_mode())
end

local function statusline_text()
  local hints = 'Alt+Return review · Alt+c draft · Alt+a/r accept/reject'
  if awaiting_since then
    local frame = spinner.frames[((spinner_tick or 0) % #spinner.frames) + 1]
    return ' ' .. frame .. ' %{v:lua._pair_review_elapsed()} ' .. mode_label() .. ' • ' .. hints .. ' • %<%t%m %= L%l/%L '
  end
  return ' 🪄 ' .. mode_label() .. ' • ' .. hints .. ' • %<%t%m %= L%l/%L '
end

local function refresh_statusline()
  vim.opt.statusline = statusline_text()
  pcall(vim.cmd, 'redrawstatus')
end

function _G._pair_review_elapsed()
  return spinner.elapsed(os.time() - (awaiting_since or os.time()))
end

local function mark_awaiting()
  awaiting_since = awaiting_since or os.time()
  refresh_statusline()
end

local function clear_awaiting()
  awaiting_since = nil
  refresh_statusline()
end

-- The v0 snapshot must join exactly as the reconcile engine's v1 (apply.buf_content),
-- so use the one shared implementation rather than re-deriving it (#89 M3).
local buf_content = review.buf_content

-- Apply-gate state (#89 M3). The human is never locked; a landed round only DEFERS
-- while they're mid-edit on the pane. pending_records holds the single deferred
-- round; the winbar is the "results ready" cue.
local pane_focused = true -- the pane is created focused
local pending_records = nil
local pending_definition = nil
local definition_timer = nil
local write_review_context
local DEF_PENDING = vim.api.nvim_create_namespace('review_definition_pending')

-- Track pane focus for the gate (case 2: not focused → apply immediately). Terminal
-- focus events may not fire on a zellij pane switch — the failure mode is benign
-- (focused stays true → at worst we DEFER when we could have applied, never a wrong
-- apply; spec §8). FocusGained/BufEnter re-assert focus on return; FocusLost drops it.
vim.api.nvim_create_autocmd({ 'FocusGained', 'BufEnter' }, { callback = function() pane_focused = true end })
vim.api.nvim_create_autocmd({ 'FocusLost' }, { callback = function() pane_focused = false end })

local function show_winbar(on)
  vim.wo.winbar = on and '%#WarningMsg# ✨ agent results ready · ⌥⏎ to apply ' or ''
end

-- Injected into review.on_agent_round's gate (init.lua): focus + current mode.
review.pane_state = function(_)
  return { focused = pane_focused, mode = vim.fn.mode() }
end

-- Deferral (gate → defer): secure the human's edits to disk FIRST (durability
-- invariant §8), stash the round, drop the spinner (the agent has replied), raise
-- the winbar. Nothing of the agent's applies until the human acts.
review.on_defer = function(buf, records, context)
  if review.human_round(buf, 'defer', context) ~= true then return false end
  pending_records = records
  clear_awaiting()
  show_winbar(true)
  return true
end

review.after_agent_round = function(buf)
  -- Any round that actually APPLIED supersedes a stale pending slot (§8: a fresh
  -- round replaces the pending one). on_defer never reaches here, so this clears
  -- pending only when a round landed — the direct-apply-while-pending edge (a
  -- second handoff arriving in normal mode) no longer leaves a stale Alt+Return.
  pending_records = nil
  clear_awaiting()
  show_winbar(false)
  review.rehydrate_definitions(buf)
  render_active_diagnostic(buf)
end

local status_timer = vim.loop.new_timer()
if status_timer then
  status_timer:start(status_timer_interval, status_timer_interval, vim.schedule_wrap(function()
    if awaiting_since then
      spinner_tick = (spinner_tick or 0) + 1
    end
    refresh_statusline()
  end))
end

local function finish_human_turn(buf, file, mode_name, instruction)
  if restoration and not restoration:guard(buf) then return false end
  if vim.fn.mode():match('^i') then vim.cmd('stopinsert') end
  -- Pending round waiting (#89 M3): Alt+Return APPLIES it (reconcile against the
  -- standing v0 base) rather than starting a new send. Consumes the pending slot +
  -- winbar; apply_round pokes the agent to commit the round.
  if pending_records then
    local r = pending_records
    local applied,dropped=review.apply_round(buf, r)
    if applied then pending_records=nil; show_winbar(false) end
    return applied,dropped
  end
  review.clear_decorations(buf)
  render_active_diagnostic(buf)
  review.human_round(buf, 'updated') -- saves; the agent commits the human round
  -- Snapshot v0 = the just-saved content (what the agent reviews). AFTER the save,
  -- and here (not mark_awaiting — request_ship also calls that WITHOUT saving), so
  -- the base is the submitted buffer, not an unsaved one (#89 M3, spec §8).
  review.set_base(buf, buf_content(buf))
  -- Poke the agent with the commit-request signal (absolute path: the agent pane's
  -- cwd is pair's, not the doc's repo). The agent commits the human round + reviews.
  -- (Once ariadne#000121's SKILL recognizes review-mode from these signals, this is
  -- the whole trigger — the M3 `/xx-fix` stopgap is retired here.)
  local m = seam.normalize_mode(mode_name or current_mode())
  seam.write_mode(vim.env.PAIR_REVIEW_MODE_PATH, m)
  local context_path = write_review_context(buf)
  refresh_statusline()
  if poke.send(poke_bodies.human_finished(vim.fn.fnamemodify(file, ':p'), m,
      instruction or '', seam.mode_label(m), context_path, restoration and restoration:context(buf))) then
    mark_awaiting()
  end
end

local function request_ship(file)
  if restoration and not restoration:guard(vim.api.nvim_get_current_buf()) then return false end
  if poke.send(poke_bodies.ship_requested(vim.fn.fnamemodify(file, ':p'), restoration and restoration:context())) then
    mark_awaiting()
  end
end

function write_review_context(buf)
  local path = vim.env.PAIR_REVIEW_CONTEXT_PATH
  if not path or path == '' then return nil end
  local tmp = path .. '.tmp'
  local body = define.strip_definition_footnote_footer(buf_content(buf))
  local ok = pcall(vim.fn.writefile, vim.split(body, '\n', { plain = true }), tmp)
  if not ok then
    vim.notify('review: could not write stripped review context', vim.log.levels.ERROR)
    return nil
  end
  local renamed_ok, renamed = pcall(vim.fn.rename, tmp, path)
  if not renamed_ok or renamed ~= 0 then
    pcall(os.remove, tmp)
    vim.notify('review: could not publish stripped review context', vim.log.levels.ERROR)
    return nil
  end
  return path
end

local function stop_definition_poll()
  if definition_timer then
    pcall(definition_timer.stop, definition_timer)
    pcall(definition_timer.close, definition_timer)
    definition_timer = nil
  end
end

local function clear_pending_definition(buf)
  if pending_definition and pending_definition.mark_id then
    pcall(vim.api.nvim_buf_del_extmark, buf, DEF_PENDING, pending_definition.mark_id)
  end
  pending_definition = nil
end

local function pending_definition_range(buf)
  if pending_definition and pending_definition.mark_id then
    local mark = vim.api.nvim_buf_get_extmark_by_id(buf, DEF_PENDING, pending_definition.mark_id, { details = true })
    if mark and #mark >= 3 and mark[3] and mark[3].end_row and mark[3].end_col then
      return mark[1] + 1, mark[2], mark[3].end_row + 1, mark[3].end_col - 1
    end
  end
  return pending_definition.l1, pending_definition.c1, pending_definition.l2, pending_definition.c2
end

local function apply_definition_result(buf)
  local result,generation = definition_seam.read_result(vim.env.PAIR_REVIEW_DEFINITION_RESULT_PATH)
  if not result or not pending_definition then return false end
  if result.request_id ~= pending_definition.request_id then return false end
  if pending_definition.review_context and not restore_policy.validate_context(result.context,pending_definition.review_context) then return false end
  local request=pending_definition
  if review.authorize and not review.authorize(buf,request.review_context) then return false end
  if pending_definition~=request then return false end
  local base = buf_content(buf)
  local lines = vim.api.nvim_buf_get_lines(buf, 0, -1, false)
  local l1, c1, l2, c2 = pending_definition_range(buf)
  local current_term = define.slice_selection(lines, l1, c1, l2, c2):gsub('^%s*(.-)%s*$', '%1')
  if current_term ~= pending_definition.term then
    vim.notify('review: definition selection changed; request ignored', vim.log.levels.WARN)
    clear_pending_definition(buf)
    clear_awaiting()
    stop_definition_poll()
    return false
  end
  local applied = define.apply_definition_footnote(
    lines,
    l1,
    c1,
    l2,
    c2,
    result.term or pending_definition.term,
    result.definition
  )
  review.projected_mutation(buf, base, function()
    vim.api.nvim_buf_call(buf, function()
      vim.cmd('silent! let &undolevels = &undolevels')
      vim.api.nvim_buf_set_lines(buf, 0, -1, false, applied.lines)
    end)
    review.rehydrate_definitions(buf)
    render_active_diagnostic(buf)
  end)
  clear_pending_definition(buf)
  definition_seam.clear_result(vim.env.PAIR_REVIEW_DEFINITION_RESULT_PATH,generation)
  clear_awaiting()
  stop_definition_poll()
  pcall(function() vim.cmd('silent keepalt write') end)
  return true
end

local function start_definition_poll(buf)
  stop_definition_poll()
  definition_timer = vim.loop.new_timer()
  if not definition_timer then return end
  definition_timer:start(500, 500, vim.schedule_wrap(function()
    pcall(apply_definition_result, buf)
  end))
end

local function request_definition(buf, file, start_pos, end_pos, opts)
  if restoration and not restoration:guard(buf) then return nil end
  opts = opts or {}
  local srow, scol = start_pos[1], start_pos[2]
  local erow, ecol = end_pos[1], end_pos[2]
  if erow < srow or (erow == srow and ecol < scol) then
    srow, erow, scol, ecol = erow, srow, ecol, scol
  end
  local lines = vim.api.nvim_buf_get_lines(buf, 0, -1, false)
  local term = define.slice_selection(lines, srow, scol, erow, ecol):gsub('^%s*(.-)%s*$', '%1')
  if term == '' then
    vim.notify('review: select text to define', vim.log.levels.INFO)
    return nil
  end
  local request = {
    request_id = string.format('def-%s', tostring(vim.loop.hrtime())),
    file = vim.fn.fnamemodify(file, ':p'),
    term = term,
    range = { l1 = srow, c1 = scol, l2 = erow, c2 = ecol },
    context = define.strip_definition_footnote_footer(buf_content(buf)),
    review_context = restoration and restoration:context(buf),
  }
  if not definition_seam.write_request(vim.env.PAIR_REVIEW_DEFINITION_REQUEST_PATH, request) then
    vim.notify('review: could not write definition request', vim.log.levels.ERROR)
    return nil
  end
  clear_pending_definition(buf)
  local mark_id = vim.api.nvim_buf_set_extmark(buf, DEF_PENDING, srow - 1, scol, {
    end_row = erow - 1,
    end_col = ecol + 1,
    right_gravity = false,
    end_right_gravity = true,
  })
  pending_definition = {
    request_id = request.request_id,
    review_context = request.review_context,
    term = term,
    l1 = srow,
    c1 = scol,
    l2 = erow,
    c2 = ecol,
    mark_id = mark_id,
  }
  if opts.poke ~= false then
    if poke.send(poke_bodies.definition_requested(request.file, request.request_id, term, request.review_context)) then
      mark_awaiting()
      start_definition_poll(buf)
    end
  end
  return request
end

local function request_visual_definition(buf, file, opts)
  local a = vim.fn.getpos("'<")
  local b = vim.fn.getpos("'>")
  return request_definition(buf, file, { a[2], math.max(a[3] - 1, 0) }, { b[2], math.max(b[3] - 1, 0) }, opts)
end

local function open_mode_menu(buf, file)
  -- With a round pending, Alt+Shift+Return APPLIES it too — a mode/instruction
  -- selector is meaningless against results already produced (#89 M3). The menu is
  -- only offered in the no-pending state.
  if pending_records then
    return finish_human_turn(buf, file)
  end
  local modes = mode.list(here .. 'review/modes')
  return menu.open({
    modes = modes,
    seam = seam,
    mode = current_mode(),
    on_submit = function(choice)
      finish_human_turn(buf, file, choice.mode, choice.instruction)
    end,
  })
end

local function escape_review()
  for _, win in ipairs(vim.api.nvim_tabpage_list_wins(0)) do
    if vim.api.nvim_win_get_config(win).relative ~= '' then
      vim.api.nvim_win_close(win, true)
    end
  end
end

local recovery_dir
local recovery_observer
local render_group=vim.api.nvim_create_augroup('PairReviewActivationRender',{clear=true})
local function stop_review(buf)
  comments.detach(buf)
  if recovery_observer then recovery_observer:cancel(buf) end
  vim.api.nvim_clear_autocmds({group=render_group,buffer=buf})
  review.stop(buf)
end
local function start_review(buf, file, resolved)
  local tag = vim.env.PAIR_TAG
  review.start({ buf = buf, file = file, tag = (tag and tag ~= '') and tag or nil, identity=resolved })

  -- Alt+Return = finish the human turn: save the human round, then poke the agent
  -- to commit it and continue in the default Edit posture.
  for _, mode in ipairs({ 'n', 'i' }) do
    vim.keymap.set(mode, '<M-CR>', function() finish_human_turn(buf, file) end,
      { buffer = buf, silent = true, desc = 'review: finish human turn (normal/insert)' })
  end
  pcall(vim.api.nvim_del_user_command, 'PairReviewShip')
  vim.api.nvim_create_user_command('PairReviewShip', function() request_ship(file) end, {})

  vim.keymap.set('n', '<Esc>', escape_review,
    { buffer = buf, silent = true, desc = 'review: dismiss float (normal mode)' })
  vim.keymap.set({ 'n', 'i' }, '<M-c>', workbench_route.return_to_draft,
    { buffer = buf, silent = true, desc = 'review: return to draft (normal/insert)' })
  vim.keymap.set('n', '<M-n>', function() jump_marker(buf, 1) end,
    { buffer = buf, silent = true, desc = 'review: next 🤖 marker (normal mode)' })
  vim.keymap.set('n', '<M-N>', function() jump_marker(buf, -1) end,
    { buffer = buf, silent = true, desc = 'review: previous 🤖 marker (normal mode; overrides agent restart)' })

  -- Accept/reject the 🤖 suggestion on the cursor line (M4b, §5), insert human
  -- comment markers, and jump between markers. Leader maps stay as a fallback;
  -- Alt+a/r/q mirror parley.nvim's review-mode shortcuts now that the pane toggle
  -- moved to Alt+c.
  vim.keymap.set('n', '<leader>a', function() resolve_at_cursor(buf, 'accept') end,
    { buffer = buf, silent = true, desc = 'review: accept 🤖 suggestion (§5)' })
  vim.keymap.set('n', '<leader>r', function() resolve_at_cursor(buf, 'reject') end,
    { buffer = buf, silent = true, desc = 'review: reject 🤖 suggestion (§5)' })
  vim.keymap.set('n', '<M-a>', function() resolve_at_cursor(buf, 'accept') end,
    { buffer = buf, silent = true, desc = 'review: accept 🤖 suggestion (§5)' })
  vim.keymap.set('n', '<M-r>', function() resolve_at_cursor(buf, 'reject') end,
    { buffer = buf, silent = true, desc = 'review: reject 🤖 suggestion (§5)' })
  vim.keymap.set('n', '<M-A>', function() resolve_paragraph_to_cursor(buf, 'accept') end,
    { buffer = buf, silent = true, desc = 'review: accept paragraph 🤖 suggestions through cursor' })
  vim.keymap.set('n', '<M-R>', function() resolve_paragraph_to_cursor(buf, 'reject') end,
    { buffer = buf, silent = true, desc = 'review: reject paragraph 🤖 suggestions through cursor' })
  vim.keymap.set('n', '<M-q>', function()
    insert_review_marker(buf)
    vim.cmd('startinsert')
  end, { buffer = buf, silent = true, desc = 'review: insert human comment marker' })
  vim.keymap.set('i', '<M-q>', function() insert_review_marker(buf) end,
    { buffer = buf, silent = true, desc = 'review: insert human comment marker' })
  vim.keymap.set('x', '<M-q>', function() quote_visual_selection(buf) end,
    { buffer = buf, silent = true, desc = 'review: quote selection for human comment' })
  for _, define_key in ipairs({ '<M-D>', '\027[68;4u' }) do
    vim.keymap.set('x', define_key, function() request_visual_definition(buf, file) end,
      { buffer = buf, silent = true, desc = 'review: define selection' })
  end
  vim.keymap.set('n', ']m', function() jump_marker(buf, 1) end,
    { buffer = buf, silent = true, desc = 'review: next 🤖 marker' })
  vim.keymap.set('n', '[m', function() jump_marker(buf, -1) end,
    { buffer = buf, silent = true, desc = 'review: prev 🤖 marker' })

  comments.attach(buf)
  render_markers(buf)
  render_active_diagnostic(buf)
  -- Rendering belongs to the current activation, including rollback/restart.
  vim.api.nvim_clear_autocmds({group=render_group,buffer=buf})
  vim.api.nvim_create_autocmd({ 'TextChanged', 'InsertLeave', 'FileChangedShellPost' }, {
    group=render_group, buffer = buf, callback = function()
      render_markers(buf)
      render_active_diagnostic(buf)
    end,
  })

  local pidfile = vim.env.PAIR_NVIM_PID_FILE
  if pidfile and pidfile ~= '' then vim.fn.writefile({tostring(vim.fn.getpid())},pidfile) end
  if restoration then restoration:publish() end
  if restoration and restoration:context(buf) then
    local snapshot,err=recovery.inspect(recovery_dir(),restoration:context(buf))
    if snapshot then vim.notify('review: unsaved recovery available; use :PairReviewRecover or :PairReviewDiscardRecovery')
    elseif err then vim.notify(err,vim.log.levels.WARN) end
  end
  refresh_statusline()
end

function recovery_dir()
  return vim.fn.fnamemodify(state_file() or (vim.fn.stdpath('data') .. '/pair/review.open'),':h') .. '/review-recovery'
end
local function preserve_buffer(buf,ctx)
  local path,err=recovery.save(recovery_dir(),ctx,buf,restoration.token)
  if not path then error(err) end
  vim.notify('review: unsaved text preserved at '..path..'; return to '..ctx.branch..' and use :PairReviewRecover')
end
local function preserve_mismatch()
  if not restoration then return true end
  for buf,ctx in pairs(restoration.bindings) do
    if vim.api.nvim_buf_is_valid(buf) and vim.bo[buf].modified and not restoration:guard(buf,true) then
      preserve_buffer(buf,ctx)
    end
  end
  return true
end
restoration=dofile(here..'review/restore_controller.lua').new({
  open_path=state_file(), session=vim.env.PAIR_SESSION_ID or '',
  start=start_review, stop=stop_review,
  pending=function()
    if pending_records then return 'deferred round; finish the original review first' end
    if pending_definition then return 'definition request pending; finish the original review first' end
    if awaiting_since then return 'agent request pending; finish the original review first' end
  end,
})
recovery_observer=dofile(here..'review/recovery_observer.lua').new({
  context=function(buf)
    local ctx=restoration:context(buf)
    if not ctx or not vim.api.nvim_buf_is_valid(buf) then return nil end
    local path=vim.api.nvim_buf_get_name(buf)
    if (vim.uv.fs_realpath(path) or path)~=ctx.repo..'/'..ctx.file then return nil end
    return ctx
  end,
  modified=function(buf)return vim.api.nvim_buf_is_valid(buf) and vim.bo[buf].modified end,
  revision=function(buf)return {tick=vim.api.nvim_buf_get_changedtick(buf),eol=vim.bo[buf].endofline,format=vim.bo[buf].fileformat,encoding=vim.bo[buf].fileencoding,bomb=vim.bo[buf].bomb} end,
  selection=function(buf)
    local selected=restoration.state.identity
    if buf==restoration.buf and selected then return {file=selected.file,head=selected.head} end
  end,
  preserve=preserve_buffer,
  refresh=function(buf,_,observed)
    if restoration.buf~=buf or vim.api.nvim_get_current_buf()~=buf or vim.bo[buf].modified then return end
    -- No filesystem read here: these exact bytes were captured before the
    -- resolver's final branch/HEAD check and travel with that identity.
    local decoded,decode_error=document_bytes.decode(observed.snapshot)
    if not decoded then vim.notify(decode_error,vim.log.levels.WARN); return end
    local changed=not vim.deep_equal(decoded.lines,vim.api.nvim_buf_get_lines(buf,0,-1,false))
    for _,option in ipairs({'endofline','fileformat','fileencoding','bomb','fixendofline'}) do
      changed=changed or vim.bo[buf][option]~=decoded[option]
    end
    if not changed then return end
    local base=buf_content(buf)
    review.projected_mutation(buf,base,function()
      vim.api.nvim_buf_call(buf,function()
        vim.cmd('silent! let &undolevels = &undolevels')
        vim.api.nvim_buf_set_lines(buf,0,-1,false,decoded.lines)
      end)
      for _,option in ipairs({'endofline','fileformat','fileencoding','bomb','fixendofline'}) do vim.bo[buf][option]=decoded[option] end
      review.reconstruct_on_open(buf,vim.api.nvim_buf_get_name(buf),observed)
    end,true)
    vim.bo[buf].modified=false
    render_markers(buf)
    render_active_diagnostic(buf)
  end,
})
review.authorize=function(buf,context)
  if context and not restore_policy.validate_context(context,restoration:context(buf),false) then return false end
  local allowed=restoration:guard(buf)
  return allowed and (not context or restore_policy.validate_context(context,restoration:context(buf),false))
end
review.context=function(buf) return restoration:context(buf) end
review.admit=function(payload,buf) return restoration:admit(payload,buf) end
review.did_land=function(body,_,buf)
  local bytes,err=document_bytes.encode(vim.api.nvim_buf_get_lines(buf,0,-1,false),vim.bo[buf])
  assert(bytes,err)
  restoration:did_land(body,bytes)
end
vim.api.nvim_create_autocmd('BufWritePre',{callback=function(ev)
  if restoration:context(ev.buf) and not restoration:guard(ev.buf) then error('review: branch mismatch; write refused') end
end})
vim.api.nvim_create_autocmd('BufWritePost',{callback=function(ev)
  local ctx=restoration:context(ev.buf)
  if ctx and restoration:guard(ev.buf,true) then
    local _,err=recovery.saved(recovery_dir(),ctx,ev.buf,restoration.token)
    if err then vim.notify(err,vim.log.levels.WARN) end
  end
end})
vim.api.nvim_create_autocmd({'FocusGained','BufEnter'},{callback=function(ev)
  recovery_observer:request(ev.buf,true)
end})
vim.api.nvim_create_autocmd({'TextChanged','TextChangedI'},{callback=function(ev)
  recovery_observer:request(ev.buf)
end})
vim.api.nvim_create_autocmd('QuitPre',{callback=preserve_mismatch})
vim.api.nvim_create_autocmd('VimLeave',{callback=function()
  recovery_observer:close()
  local ok,err=pcall(preserve_mismatch); if not ok then vim.notify(err,vim.log.levels.ERROR) end
  local buf=restoration.buf
  if buf and vim.api.nvim_buf_is_valid(buf) then
    if vim.bo[buf].modified and restoration:guard(buf,true) then pcall(review.human_round,buf,'exit') end
    pcall(stop_review,buf)
  end
  restoration:close()
  if status_timer then status_timer:stop(); status_timer:close() end
  stop_definition_poll()
end})
vim.api.nvim_create_user_command('PairReviewRecover',function()
  local buf=restoration.buf
  if not restoration:guard(buf) then return end
  if vim.bo[buf].modified then vim.notify('review: save current edits before recovering older text',vim.log.levels.WARN); return end
  local snapshot,err=recovery.restore(recovery_dir(),restoration:context(),buf)
  if err then vim.notify(err,vim.log.levels.ERROR)
  elseif not snapshot then vim.notify('review: no recovery snapshot for this document') end
end,{})
vim.api.nvim_create_user_command('PairReviewDiscardRecovery',function()
  local _,err=recovery.discard(recovery_dir(),restoration:context())
  if err then vim.notify(err,vim.log.levels.ERROR) end
end,{})

-- Exposed for the headless test.
_G.PairReviewPane = { restore = function(request)
    if type(request)=='string' then request=vim.json.decode(request) end
    return restoration:request(request)
  end, controller=restoration, start_review = start_review, render_markers = render_markers,
  state_file = state_file, finish_human_turn = finish_human_turn,
  request_ship = request_ship,
  status_timer_interval = status_timer_interval,
  open_mode_menu = function(file)
    return open_mode_menu(vim.api.nvim_get_current_buf(), file)
  end,
  request_definition = request_definition,
  request_visual_definition = request_visual_definition,
  apply_definition_result = apply_definition_result,
  write_review_context = write_review_context,
  rehydrate_definitions = function(buf) return review.rehydrate_definitions(buf or vim.api.nvim_get_current_buf()) end,
  resolve_at_cursor = resolve_at_cursor, insert_review_marker = insert_review_marker,
  resolve_paragraph_to_cursor = resolve_paragraph_to_cursor,
  quote_selection = quote_selection, current_mode = current_mode, mode_label = mode_label,
  active_diagnostic = active_diagnostic, render_active_diagnostic = render_active_diagnostic,
  open_diagnostic_float = open_review_diagnostic_float,
  -- #89 M3 apply-gate hooks, exposed for the window test.
  pane_state = function() return review.pane_state(vim.api.nvim_get_current_buf()) end,
  set_focused = function(v) pane_focused = v end,
  on_defer = function(buf, recs) return review.on_defer(buf or vim.api.nvim_get_current_buf(), recs) end,
  has_pending = function() return pending_records ~= nil end,
  winbar = function() return vim.wo.winbar end }

-- Start once the file is loaded (the buffer doesn't exist yet at init time).
vim.api.nvim_create_autocmd('VimEnter', {
  once = true,
  callback = function()
    local buf = vim.api.nvim_get_current_buf()
    local file = vim.api.nvim_buf_get_name(buf)
    if file ~= nil and file ~= '' then
      local observed
      if vim.env.PAIR_REVIEW_IDENTITY then
        local ok,want=pcall(vim.json.decode,vim.env.PAIR_REVIEW_IDENTITY)
        if ok and type(want)=='table' then observed=identity.resolve(want.repo,want,true) end
        if not observed or observed.status~='resolved' or not restore_policy.same(want,observed) or want.head~=observed.head then
          vim.notify('review: checkout changed before opening',vim.log.levels.ERROR)
          return
        end
      else
        observed=identity.resolve(vim.fn.fnamemodify(file,':h'),nil,true)
      end
      restoration:init(buf,observed,vim.env.PAIR_REVIEW_IDENTITY~=nil)
      start_review(buf,file,observed and observed.status=='resolved' and observed or nil)
    end
  end,
})

vim.opt.laststatus = 2
-- Compact review bar. %m → "[+]" when the buffer is unsaved (the save happens at
-- Alt+Return / after the agent applies records, never per-keystroke — so the cue
-- matters). %= right-aligns the line position.
refresh_statusline()
