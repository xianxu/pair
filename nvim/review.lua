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
if not pcall(vim.diagnostic.config, {
  virtual_lines = { current_line = true },
  virtual_text = false,
  signs = true,
  underline = true,
  severity_sort = true,
}) then
  vim.diagnostic.config({ virtual_text = true, signs = true })
end

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

-- Alt+r pressed while THIS review pane is focused: the zellij bind's relative
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
  poke.send(poke_bodies.human_committed(vim.fn.fnamemodify(file, ':p')))
end

local function start_review(buf, file)
  local tag = vim.env.PAIR_TAG
  review.start({ buf = buf, file = file, tag = (tag and tag ~= '') and tag or nil })

  -- Alt+Return = finish the human turn (the fix-skill gesture): commit the human
  -- round, then poke the agent "updated, please review <file>".
  for _, mode in ipairs({ 'n', 'i' }) do
    vim.keymap.set(mode, '<M-CR>', function() finish_human_turn(buf, file) end,
      { buffer = buf, silent = true })
  end

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
  state_file = state_file, finish_human_turn = finish_human_turn }

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
vim.opt.statusline = ' Review • Alt+⏎ Send • Alt+r → agent • %f%m %= L%l/%L '
