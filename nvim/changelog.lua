-- nvim/changelog.lua — read-only viewer for the pair Alt+l change log (#53).
--
-- Loaded as `nvim -u nvim/changelog.lua <changelog-<tag>-<agent>.md>` by
-- bin/pair-changelog-open. The distilled counterpart to scrollback.lua, but
-- much simpler: the buffer is plain markdown (no SGR reconstruction, no marker
-- system), so this is a read-only buffer plus a few token-colorizing syntax
-- rules for quick glancing.
--
-- It opens IMMEDIATELY on whatever log already exists, then runs the
-- render+distill as a background job (via PAIR_CHANGELOG_* env from the
-- orchestrator), showing a spinner as a bottom virtual line and reloading the
-- buffer when the job finishes. The distiller skips the model when no new turn
-- completed, so an unchanged session clears the spinner near-instantly.
--
-- M.setup is exported so nvim/changelog_test.lua can drive it headlessly
-- (`nvim -l`) without launching the interactive UI / background job.

local M = {}

-- colorize applies the glance-token highlights to bufnr. Runs inside the
-- buffer's context so `:syntax match` targets the right buffer.
function M.colorize(bufnr)
  vim.api.nvim_buf_call(bufnr, function()
    vim.cmd([[
      syntax clear
      syntax match ChangelogTicket    /#\d\+/
      syntax match ChangelogMilestone /\<M\d\+\>/
      syntax match ChangelogCode      /`[^`]\+`/
      syntax match ChangelogBranch    /\<feature\/\S\+/
    ]])
  end)
  vim.cmd([[
    highlight default link ChangelogTicket    Identifier
    highlight default link ChangelogMilestone Type
    highlight default link ChangelogCode      String
    highlight default link ChangelogBranch    Constant
  ]])
end

-- setup makes bufnr a read-only viewer buffer and colorizes it.
function M.setup(bufnr)
  bufnr = bufnr or vim.api.nvim_get_current_buf()
  vim.bo[bufnr].buftype = 'nofile'
  vim.bo[bufnr].swapfile = false
  M.colorize(bufnr)
  vim.bo[bufnr].modifiable = false
  vim.bo[bufnr].readonly = true
end

-- reload re-reads the log file into the (read-only) buffer and re-colorizes,
-- keeping the cursor at the newest entry.
function M.reload(bufnr, logpath)
  local ok, lines = pcall(vim.fn.readfile, logpath)
  if not ok then return end
  -- Clear readonly too, not just modifiable, so the programmatic write doesn't
  -- trip "W10: Changing a readonly file".
  vim.bo[bufnr].readonly = false
  vim.bo[bufnr].modifiable = true
  vim.api.nvim_buf_set_lines(bufnr, 0, -1, false, lines)
  vim.bo[bufnr].modifiable = false
  vim.bo[bufnr].readonly = true
  M.colorize(bufnr)
  pcall(function()
    vim.api.nvim_win_set_cursor(0, { math.max(1, vim.api.nvim_buf_line_count(bufnr)), 0 })
  end)
end

-- start_refresh turns the viewer into a WATCHER of the DETACHED distiller (#58).
-- The orchestrator (bin/pair-changelog-open) launches render+distill as a nohup'd
-- background process — NOT a child of nvim — and records its PID in
-- $PAIR_CHANGELOG_DLOCK. So closing the viewer does NOT stop the build. This
-- polls: the log file (reloading per batch), the status file (batch progress),
-- and the distiller PID — showing a bottom virtual-line spinner while it runs and
-- a final reload (or error tip) when it exits.
function M.start_refresh(bufnr)
  if vim.b[bufnr] and vim.b[bufnr].pair_cl_watching then return end
  pcall(function() vim.b[bufnr].pair_cl_watching = true end)

  local log = os.getenv('PAIR_CHANGELOG_LOG')
  if not log or log == '' then return end
  local dlock = os.getenv('PAIR_CHANGELOG_DLOCK')
  local status = os.getenv('PAIR_CHANGELOG_STATUS')
  local uv = vim.uv or vim.loop
  local ns = vim.api.nvim_create_namespace('pair_changelog_spinner')

  local function distiller_alive()
    if not dlock then return false end
    local ok, ls = pcall(vim.fn.readfile, dlock)
    if not ok or not ls[1] then return false end
    local pid = tonumber(ls[1])
    if not pid then return false end
    vim.fn.system({ 'kill', '-0', tostring(pid) })
    return vim.v.shell_error == 0
  end

  -- Nothing building (no distiller launched, or it already finished) → just show
  -- the existing log; nvim already loaded it and M.setup ran.
  if not distiller_alive() then return end

  local first_run = vim.api.nvim_buf_line_count(bufnr) <= 1
    and (vim.api.nvim_buf_get_lines(bufnr, 0, 1, false)[1] or '') == ''
  local msg = first_run and 'Computing change log…' or 'Refreshing change log…'
  local err = nil
  local frames = { '⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏' }
  local i, tick = 0, 0
  local last_key = ''
  local running = true
  local timer

  -- The spinner is a virtual line at the bottom (where the next entry lands) — a
  -- virt_lines extmark, so the read-only buffer is never mutated.
  local function tip(text, hl)
    local last = math.max(0, vim.api.nvim_buf_line_count(bufnr) - 1)
    pcall(vim.api.nvim_buf_set_extmark, bufnr, ns, last, 0, {
      id = 1, virt_lines = { { { '', 'Comment' } }, { { text, hl } } },
    })
  end

  local function reload_if_changed()
    local st = uv.fs_stat(log)
    if not st then return end
    local key = st.mtime.sec .. '.' .. st.mtime.nsec .. '.' .. st.size
    if key ~= last_key then
      last_key = key
      M.reload(bufnr, log)
    end
  end

  local function read_status()
    if not status then return end
    local ok, ls = pcall(vim.fn.readfile, status)
    if not ok then return end
    for _, l in ipairs(ls) do
      local b = l:match('distilling batch (%d+/%d+)')
      if b then msg = 'Computing change log (batch ' .. b .. ')…' end
      local n = l:match('distilling (%d+) lines')
      if n and not b then msg = 'Refreshing change log (' .. n .. ' new lines)…' end
      local e = l:match('pair%-changelog: (.+)')
      if e and not e:match('^distilling') and not e:match('^up to date') then err = e end
    end
  end

  local function poll()
    tick = tick + 1
    reload_if_changed()            -- per-batch progressive reload
    if tick % 4 == 0 then read_status() end
    if tick % 8 == 0 then running = distiller_alive() end
    if running then
      i = (i % #frames) + 1
      tip('  ' .. frames[i] .. '  ' .. msg, 'Comment')
    else
      read_status() -- catch a final error line
      reload_if_changed()
      M.reload(bufnr, log)
      if err then
        tip('  ⚠  change log refresh failed: ' .. err, 'ErrorMsg')
      else
        pcall(vim.api.nvim_buf_del_extmark, bufnr, ns, 1)
      end
      if timer then pcall(vim.fn.timer_stop, timer); timer = nil end
    end
  end

  tip('  ' .. frames[1] .. '  ' .. msg, 'Comment')
  timer = vim.fn.timer_start(120, poll, { ['repeat'] = -1 })
end

-- Interactive wiring — skipped under the headless test (which sets the guard).
if not _G.PAIR_CHANGELOG_TEST then
  vim.opt.number = false
  vim.opt.signcolumn = 'no'
  vim.opt.laststatus = 0
  vim.opt.fillchars:append({ eob = ' ' })

  vim.api.nvim_create_autocmd({ 'BufReadPost', 'BufWinEnter' }, {
    callback = function(args)
      M.setup(args.buf)
      vim.cmd('normal! G') -- newest entry at the bottom
      M.start_refresh(args.buf)
    end,
  })

  -- Esc / q quit the whole viewer (and its floating pane).
  for _, key in ipairs({ '<Esc>', 'q' }) do
    vim.keymap.set('n', key, '<cmd>qa!<cr>', { silent = true })
  end
end

return M
