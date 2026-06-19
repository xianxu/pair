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
vim.opt.number = false
vim.opt.signcolumn = 'no'

-- Stub the draft-pane globals so a stray zellij Alt+Up/Down/x/n/d pressed while
-- the review pane is focused degrades silently (same rationale as scrollback.lua).
for _, name in ipairs({
  'PairLayoutBigger', 'PairLayoutSmaller', 'PairConfirmQuit',
  'PairConfirmDetach', 'PairConfirmRestart', 'PairConfirmRestartNewSession',
}) do
  _G[name] = function() end
end

local here = debug.getinfo(1, 'S').source:match('@?(.*/)') or './'
local review = dofile(here .. 'review/init.lua')
local poke = dofile(here .. 'pair_poke.lua')
local markers = dofile(here .. 'review/markers.lua')

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

-- The review state file: pair-review-toggle reads it to know a review is open.
local function state_file()
  local dir = vim.env.PAIR_DATA_DIR
  if not dir or dir == '' then return nil end
  return dir .. '/review-' .. (vim.env.PAIR_TAG or 'default') .. '.open'
end

local function finish_human_turn(buf, file)
  if vim.fn.mode():match('^i') then vim.cmd('stopinsert') end
  review.human_round(buf, 'updated')
  poke.send('updated, please review ' .. vim.fn.fnamemodify(file, ':t'))
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
  vim.api.nvim_create_autocmd({ 'TextChanged', 'InsertLeave' }, {
    buffer = buf, callback = function() render_markers(buf) end,
  })

  -- pid file (reaped by bin/pair's cleanup) + the open-state file.
  local pidfile = vim.env.PAIR_NVIM_PID_FILE
  if pidfile and pidfile ~= '' then
    pcall(vim.fn.writefile, { tostring(vim.fn.getpid()) }, pidfile)
  end
  -- state file: line 1 = pid (liveness), line 2 = visibility (pair-review-toggle
  -- flips it). Tracking visibility here avoids `are-floating-panes-visible` being
  -- confounded by the toggle's own transient floating pane.
  local sf = state_file()
  if sf then pcall(vim.fn.writefile, { tostring(vim.fn.getpid()), 'visible' }, sf) end

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
vim.opt.statusline = ' pair review · Alt+Return finish human turn · Alt+r → agent · %f %= L%l/%L '
