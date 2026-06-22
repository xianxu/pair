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
-- The review pane is an EDITABLE document workbench, not a read-only viewer (the
-- scrollback look these were copied from) — show the gutter: absolute line numbers
-- + a stable sign column for the review diagnostics.
vim.opt.number = true
vim.opt.signcolumn = 'yes'

-- Diagnosis display — review/apply.lua sets each record's `explain` as an INFO
-- diagnostic (the "why" behind an edit). Render it parley-style: a sign in the
-- gutter on every edit, and the (wrapped) why auto-expanded as a virtual line below
-- the edit ONLY when the cursor is in its region. The review pane has no LSP, so a
-- global config is safe (review diagnostics are the only source). pcall-guarded:
-- `virtual_lines` is nvim 0.11+; degrade to virtual_text on anything older.
-- `format` returns just the message (already hard-wrapped in apply.lua) so the
-- inline virtual_lines carry no source/severity prefix — trimming the leading
-- "header" columns (M4a issue 2.2). pcall-guarded; degrade to virtual_text on
-- older nvim or if virtual_lines.format is unsupported.
if not pcall(vim.diagnostic.config, {
  virtual_lines = { current_line = true, format = function(d) return d.message end },
  virtual_text = false,
  signs = true,
  underline = true,
  severity_sort = true,
}) then
  vim.diagnostic.config({ virtual_text = true, signs = true })
end

-- Diagnostic navigation (M4a issue 1): the user tried `<C-w>d` (→ E388, vim's
-- define-search) and `]d` (jumped without showing the why). Bind the float
-- explicitly, and make `]d`/`[d` jumps pop the float so the "why" is visible.
vim.keymap.set('n', '<C-w>d', vim.diagnostic.open_float, { desc = 'review: diagnostic float' })
vim.keymap.set('n', 'gl', vim.diagnostic.open_float, { desc = 'review: diagnostic float' })
local function diag_jump(count)
  if vim.diagnostic.jump then
    vim.diagnostic.jump({ count = count, float = true }) -- nvim 0.11 API
  else
    (count > 0 and vim.diagnostic.goto_next or vim.diagnostic.goto_prev)({ float = true })
  end
end
vim.keymap.set('n', ']d', function() diag_jump(1) end, { desc = 'review: next diagnostic (float)' })
vim.keymap.set('n', '[d', function() diag_jump(-1) end, { desc = 'review: prev diagnostic (float)' })

-- Markdown appearance — mirrors the draft's setup (nvim/init.lua ~L33-79). nvim's
-- bundled `default` colorscheme is near-monochrome, so the review doc's syntax
-- reads as muted; `slate` + fenced-language highlighting gives readable headings,
-- emphasis, code spans, and per-language code blocks. fenced_languages must be set
-- before the .md loads, so it sits at top-of-init. (Two md-nvims share this now;
-- factor into a module if a third appears.)
vim.cmd('syntax enable')
vim.cmd('filetype plugin indent on')
vim.g.markdown_fenced_languages = {
  'python', 'py=python', 'javascript', 'js=javascript', 'typescript', 'ts=typescript',
  'lua', 'bash', 'sh=bash', 'zsh=bash', 'json', 'yaml', 'yml=yaml', 'toml',
  'html', 'css', 'c', 'cpp', 'cxx=cpp', 'rust', 'rs=rust', 'go', 'sql',
  'ruby', 'rb=ruby', 'java', 'kotlin', 'kt=kotlin', 'swift', 'dockerfile', 'make', 'diff', 'vim',
}
vim.cmd('colorscheme slate')
vim.api.nvim_set_hl(0, 'Comment', { fg = '#999999', ctermfg = 247 }) -- lift slate's faded Comment

-- The agent edits the doc on disk (M3: via xx-fix's file write; M4: the pane
-- applies records in-buffer, so this won't fire). autoread + a checktime when the
-- user returns to the pane reloads those external edits instead of the cryptic W12
-- prompt — provided the buffer is unmodified (finish a human turn with Alt+Return,
-- which saves, before the agent edits). A genuine both-changed conflict still
-- prompts: the human's unsaved edits are theirs to resolve.
vim.opt.autoread = true
vim.api.nvim_create_autocmd({ 'FocusGained', 'BufEnter' }, {
  callback = function()
    if vim.fn.mode() == 'n' then pcall(vim.cmd, 'silent! checktime') end
  end,
})

-- Stub the draft-pane globals so a stray zellij Alt+Up/Down/x/n/d pressed while
-- the review pane is focused degrades silently (same rationale as scrollback.lua).
for _, name in ipairs({
  'PairLayoutBigger', 'PairLayoutSmaller', 'PairConfirmQuit',
  'PairConfirmDetach', 'PairConfirmRestart', 'PairConfirmRestartNewSession',
}) do
  _G[name] = function() end
end

-- Alt+c pressed while THIS review pane is focused: the zellij bind's relative
-- MoveFocus Down may not escape a floating pane, so the `:lua PairReviewToggle()`
-- it types lands here instead of in the draft. The review pane only ever needs to
-- hide itself (it's necessarily visible to receive the keystrokes); focus falls
-- back to the tiled layer. The draft's PairReviewToggle owns open/show; this one
-- is the hide half (robust whether or not MoveFocus escapes).
function _G.PairReviewToggle()
  vim.fn.system({ 'zellij', 'action', 'hide-floating-panes' })
end

local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local review = dofile(here .. 'review/init.lua')
local poke = dofile(here .. 'pair_poke.lua')
local markers = dofile(here .. 'review/markers.lua')
local seam = dofile(here .. 'review/seam.lua')
local poke_bodies = dofile(here .. 'review/poke_bodies.lua')
local resolve = dofile(here .. 'review/resolve.lua')

-- 🤖 marker highlight groups (parley's names; linked, overridable by a colorscheme).
vim.api.nvim_set_hl(0, 'ParleyReviewQuoted', { link = 'Comment', default = true })
vim.api.nvim_set_hl(0, 'ParleyReviewStrike', { strikethrough = true, default = true })
vim.api.nvim_set_hl(0, 'ParleyReviewUser', { link = 'Question', default = true })
vim.api.nvim_set_hl(0, 'ParleyReviewAgent', { link = 'Identifier', default = true })

local MARK_NS = vim.api.nvim_create_namespace('review_markers')
local function render_markers(buf)
  if not vim.api.nvim_buf_is_valid(buf) then return end
  local lines = vim.api.nvim_buf_get_lines(buf, 0, -1, false)
  vim.api.nvim_buf_clear_namespace(buf, MARK_NS, 0, -1)
  for _, s in ipairs(markers.highlight_spans(lines)) do
    pcall(vim.api.nvim_buf_set_extmark, buf, MARK_NS, s.row, s.col_start, {
      end_col = s.col_end, hl_group = s.hl_group,
    })
  end
end

local REVIEW_MARKER = '🤖'

local function split_replacement(text)
  if text == '' then return {} end
  return vim.split(text, '\n', { plain = true })
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
    if m.line == row0 then
      local raw_lines = vim.split(m.raw, '\n', { plain = true })
      local end_col = (#raw_lines == 1) and (m.col + #m.raw) or #raw_lines[#raw_lines]
      if not target then target = m end -- same-line fallback for legacy cursor placement
      if cur[2] >= m.col and cur[2] < end_col then target = m; break end
    end
  end
  if not target then
    vim.notify('review: no 🤖 marker on this line', vim.log.levels.INFO); return
  end
  local replacement = resolve.resolve(target, action)
  -- The marker's raw span (🤖 → end of last section) may cross lines (bounded).
  local raw_lines = vim.split(target.raw, '\n', { plain = true })
  local end_row = target.line + #raw_lines - 1
  local end_col = (#raw_lines == 1) and (target.col + #target.raw) or #raw_lines[#raw_lines]
  vim.api.nvim_buf_set_text(buf, target.line, target.col, end_row, end_col,
    split_replacement(replacement))
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
  return seam.open_state(vim.env.PAIR_DATA_DIR, vim.env.PAIR_TAG)
end

local function finish_human_turn(buf, file)
  if vim.fn.mode():match('^i') then vim.cmd('stopinsert') end
  review.human_round(buf, 'updated') -- saves; the agent commits the human round
  -- Poke the agent with the commit-request signal (absolute path: the agent pane's
  -- cwd is pair's, not the doc's repo). The agent commits the human round + reviews.
  -- (Once ariadne#000121's SKILL recognizes review-mode from these signals, this is
  -- the whole trigger — the M3 `/xx-fix` stopgap is retired here.)
  poke.send(poke_bodies.human_finished(vim.fn.fnamemodify(file, ':p')))
end

local function request_ship(file)
  poke.send(poke_bodies.ship_requested(vim.fn.fnamemodify(file, ':p')))
end

local function start_review(buf, file)
  local tag = vim.env.PAIR_TAG
  review.start({ buf = buf, file = file, tag = (tag and tag ~= '') and tag or nil })

  -- Alt+Return = finish the human turn: save the human round, then poke the agent
  -- to commit it and continue in the default Copy Edit posture.
  for _, mode in ipairs({ 'n', 'i' }) do
    vim.keymap.set(mode, '<M-CR>', function() finish_human_turn(buf, file) end,
      { buffer = buf, silent = true })
  end
  pcall(vim.api.nvim_del_user_command, 'PairReviewShip')
  vim.api.nvim_create_user_command('PairReviewShip', function() request_ship(file) end, {})

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
  vim.keymap.set('n', '<M-q>', function()
    insert_review_marker(buf)
    vim.cmd('startinsert')
  end, { buffer = buf, silent = true, desc = 'review: insert human comment marker' })
  vim.keymap.set('i', '<M-q>', function() insert_review_marker(buf) end,
    { buffer = buf, silent = true, desc = 'review: insert human comment marker' })
  vim.keymap.set('x', '<M-q>', function() quote_visual_selection(buf) end,
    { buffer = buf, silent = true, desc = 'review: quote selection for human comment' })
  vim.keymap.set('n', ']m', function() jump_marker(buf, 1) end,
    { buffer = buf, silent = true, desc = 'review: next 🤖 marker' })
  vim.keymap.set('n', '[m', function() jump_marker(buf, -1) end,
    { buffer = buf, silent = true, desc = 'review: prev 🤖 marker' })

  render_markers(buf)
  -- Re-render on local edits AND after an external reload (autoread/checktime
  -- pulling in the agent's on-disk edits) so markers track the new content.
  vim.api.nvim_create_autocmd({ 'TextChanged', 'InsertLeave', 'FileChangedShellPost' }, {
    buffer = buf, callback = function() render_markers(buf) end,
  })

  -- pid file (reaped by bin/pair's cleanup) + the open-state file.
  local pidfile = vim.env.PAIR_NVIM_PID_FILE
  if pidfile and pidfile ~= '' then
    pcall(vim.fn.writefile, { tostring(vim.fn.getpid()) }, pidfile)
  end
  -- state file: line 1 = the pane nvim's pid (liveness); line 2 = the absolute
  -- doc path (the draft reads it to render the review-mode indicator on its
  -- line 1). The draft's PairReviewToggle decides show vs. hide from live
  -- `are-floating-panes-visible` (reliable now that no transient floating
  -- toggle-pane confounds it), so visibility is NOT tracked here.
  local sf = state_file()
  if sf then pcall(vim.fn.writefile, { tostring(vim.fn.getpid()), file }, sf) end

  -- Announce the workbench to the agent (M4a smoke fix): opening the pane is the
  -- review-START signal a chat "please review" otherwise lacks — without it the
  -- agent can't tell it's in the workbench and falls back to summarize-and-ask.
  -- Context only (no review triggered yet), so no branch/commit until the operator
  -- actually asks. Deferred so the agent pane is resolvable + settled first.
  vim.defer_fn(function()
    pcall(poke.send, poke_bodies.review_opened(vim.fn.fnamemodify(file, ':p')))
  end, 200)

  -- Lifecycle (the M1-carried "VimLeave timer cleanup"): tear down the handoff
  -- poll timer + projection autocmd + state file when the pane nvim exits.
  vim.api.nvim_create_autocmd('VimLeave', {
    callback = function()
      pcall(review.stop, buf)
      if sf then pcall(os.remove, sf) end
    end,
  })
end

-- Exposed for the headless test.
_G.PairReviewPane = { start_review = start_review, render_markers = render_markers,
  state_file = state_file, finish_human_turn = finish_human_turn,
  request_ship = request_ship,
  resolve_at_cursor = resolve_at_cursor, insert_review_marker = insert_review_marker,
  quote_selection = quote_selection }

-- Start once the file is loaded (the buffer doesn't exist yet at init time).
vim.api.nvim_create_autocmd('VimEnter', {
  once = true,
  callback = function()
    local buf = vim.api.nvim_get_current_buf()
    local file = vim.api.nvim_buf_get_name(buf)
    if file ~= nil and file ~= '' then start_review(buf, file) end
  end,
})

vim.opt.laststatus = 2
-- Compact review bar. %m → "[+]" when the buffer is unsaved (the save happens at
-- Alt+Return / after the agent applies records, never per-keystroke — so the cue
-- matters). %= right-aligns the line position.
vim.opt.statusline = ' Review • Alt+⏎ Send • Alt+c → agent • %f%m %= L%l/%L '
