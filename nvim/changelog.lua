-- nvim/changelog.lua — read-only viewer for the pair Alt+l change log (#53).
--
-- Loaded as `nvim -u nvim/changelog.lua <changelog-<tag>-<agent>.md>` by
-- bin/pair-changelog-open. The distilled counterpart to scrollback.lua, but
-- much simpler: the buffer is plain markdown (no SGR reconstruction, no marker
-- system), so this is a read-only buffer plus a few token-colorizing syntax
-- rules for quick glancing.
--
-- M.setup is exported so nvim/changelog_test.lua can drive it headlessly
-- (`nvim -l`) without launching the interactive UI.

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
    end,
  })

  -- Esc / q quit the whole viewer (and its floating pane).
  for _, key in ipairs({ '<Esc>', 'q' }) do
    vim.keymap.set('n', key, '<cmd>qa!<cr>', { silent = true })
  end
end

return M
