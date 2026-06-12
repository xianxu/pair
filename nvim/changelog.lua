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

-- start_refresh runs the render+distill as a background job, animating a winbar
-- spinner and reloading the buffer on completion. No-op unless the orchestrator
-- set PAIR_CHANGELOG_REFRESH=1.
function M.start_refresh(bufnr)
  if os.getenv('PAIR_CHANGELOG_REFRESH') ~= '1' then return end
  local render  = os.getenv('PAIR_CHANGELOG_RENDER')
  local distill = os.getenv('PAIR_CHANGELOG_DISTILL')
  local raw     = os.getenv('PAIR_CHANGELOG_RAW')
  local events  = os.getenv('PAIR_CHANGELOG_EVENTS')
  local cleaned = os.getenv('PAIR_CHANGELOG_CLEANED')
  local log     = os.getenv('PAIR_CHANGELOG_LOG')
  local anchor  = os.getenv('PAIR_CHANGELOG_ANCHOR')
  local agent   = os.getenv('PAIR_CHANGELOG_AGENT') or 'claude'
  local today   = os.getenv('PAIR_CHANGELOG_TODAY') or ''
  if not (render and distill and raw and events and cleaned and log and anchor) then return end

  local esc = vim.fn.shellescape
  local cmd = esc(render) .. ' --plain --max-lines 0 ' .. esc(raw) .. ' ' .. esc(events) .. ' ' .. esc(cleaned)
    .. ' && ' .. esc(distill) .. ' --cleaned ' .. esc(cleaned) .. ' --log ' .. esc(log)
    .. ' --anchor ' .. esc(anchor) .. ' --agent ' .. esc(agent) .. ' --today ' .. esc(today)

  local first_run = vim.api.nvim_buf_line_count(bufnr) <= 1
    and (vim.api.nvim_buf_get_lines(bufnr, 0, 1, false)[1] or '') == ''
  local msg = first_run and 'Computing change log…' or 'Refreshing change log…'
  local frames = { '⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏' }
  local i = 0
  -- The spinner renders as a virtual line at the bottom — where the new entry
  -- will be inserted — not in the winbar. virt_lines don't touch buffer content,
  -- so the buffer stays read-only / unmodified.
  local ns = vim.api.nvim_create_namespace('pair_changelog_spinner')
  local function paint()
    i = (i % #frames) + 1
    local last = math.max(0, vim.api.nvim_buf_line_count(bufnr) - 1)
    pcall(vim.api.nvim_buf_set_extmark, bufnr, ns, last, 0, {
      id = 1,
      virt_lines = { { { '', 'Comment' } }, { { '  ' .. frames[i] .. '  ' .. msg, 'Comment' } } },
    })
  end
  paint()
  local timer = vim.fn.timer_start(90, paint, { ['repeat'] = -1 })

  local function finish()
    pcall(vim.fn.timer_stop, timer)
    pcall(vim.api.nvim_buf_del_extmark, bufnr, ns, 1)
    M.reload(bufnr, log)
  end

  local job = vim.fn.jobstart({ 'sh', '-c', cmd }, {
    on_stderr = function(_, data)
      for _, line in ipairs(data or {}) do
        local n = line:match('distilling (%d+) lines')
        if n then msg = 'Refreshing change log (' .. n .. ' new lines)…' end
        if line:match('up to date') then msg = 'Up to date' end
      end
    end,
    on_exit = function() finish() end,
  })
  if job <= 0 then finish() end -- job failed to start: clear the spinner
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
